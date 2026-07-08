# HelixVPN -- In-Memory Per-Tenant Topology Graph Design (HVPN-P1-070)

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Status:** active -- concrete design for the coordinator's in-memory topology graph
**Authority:** This document is the binding design for the coordinator's in-memory per-tenant
topology graph, the data structure that enables fast per-node `buildMap()` computation.
It synthesizes the existing `map.rs` NetworkMap schema, the IPAM 4via6 design, the
policy-compiler CompiledPolicy output, and external research on concurrent graph data
structures in Go and Tailscale/WireGuard topology design.
**Scope:** `helix-go/internal/topology/` -- Go package implementing the topology graph.

---

## 1. Problem Statement

The coordinator must compute a per-node `NetworkMap` (the `buildMap(node)` function from
P1-071) for every enrolled device. This computation requires answering questions like:

- "Which peers are visible to device D?" (need-to-know, policy-filtered)
- "What are the WireGuard `allowed_ips` for each visible peer?"
- "What 4via6 routes does each connector peer advertise?"
- "Which connectors serve which LAN CIDRs?"
- "Is this device a gateway, a connector, or a client?"

Doing this by querying Postgres directly on every map computation (potentially many times
per second as policy changes or devices enroll/revoke) would bottleneck on database latency.

The **topology graph** is an in-memory data structure that hydrates from Postgres at
coordinator startup, then stays in sync via an event stream (Redis Streams). It provides
O(1) node and edge lookups by ID, O(degree) adjacency traversal, and O(1) immutable
snapshots for concurrent `buildMap()` calls. It is the single source of truth for
"what is the current tenant topology?" at the coordinator layer.

### 1.1 Position in the system

```
Postgres (devices, connector_sites, advertised_prefixes, policies, overlay_pools)
       |
       v (hydrate on boot)
[TopologyGraph]  <----  Redis Streams (device.enrolled, device.revoked,
       |                   connector.attached, connector.detached, policy.compiled)
       | Snapshot()
       v
buildMap(node) -> NetworkMap -> WatchNetworkMap stream -> Rust edge reconcile()
```

### 1.2 Governing constraints

| # | Invariant | Source |
|---|---|---|
| T1 | **Per-tenant.** Each tenant has its own `TopologyGraph` instance -- no cross-tenant leakage. | [02-control-plane §2.1] |
| T2 | **In-memory only.** No durable coordinator tables. The graph is rebuilt from Postgres on restart. | This design |
| T3 | **Event-driven incremental mutation.** Events mutate the graph in-place; no periodic full rebuild. | [02-control-plane §7.4] |
| T4 | **Immutable snapshots.** `buildMap()` reads a consistent point-in-time view that does not change during computation. | This design |
| T5 | **Need-to-know visibility.** A node only sees peers the policy explicitly grants. Visibility is computed from `CompiledPolicy.VisibleTo`, not from raw graph membership. | [policies_spec §1.2 P1] |
| T6 | **Concurrent-safe.** Multiple `buildMap()` calls may execute concurrently against the same graph; mutations may interleave with reads. | This design |
| T7 | **Memory-bounded.** For a 1,000-device tenant, the graph must fit comfortably in <10 MiB. | This design §7 |

---

## 2. Graph Model

### 2.1 Nodes

Every enrolled device, connector, and gateway is a **node** in the graph. Nodes are
identified by their `device_id` (UUID, the primary key from the `devices` table).

```go
// NodeKind classifies a node by its role in the topology.
type NodeKind string

const (
    NodeClient    NodeKind = "client"    // end-user device (laptop, phone)
    NodeConnector NodeKind = "connector" // site connector (serves a LAN)
    NodeGateway   NodeKind = "gateway"   // overlay gateway (terminates tunnels)
)

// Node is a vertex in the topology graph.
type Node struct {
    ID          uuid.UUID      // device_id (PK from devices table)
    TenantID    uuid.UUID      // owning tenant
    Kind        NodeKind       // client | connector | gateway
    Name        string         // human-readable device name
    WgPubkey    [32]byte       // WireGuard Curve25519 public key (raw bytes)
    OverlayIP   netip.Addr     // fabric overlay address (e.g. fdXX:XXXX:XXXX::2)
    SiteID      uint16         // connector site ID (0 for clients/gateway)
    Status      NodeStatus     // active | revoked
    Capabilities uint32        // bitmask: CAN_EXIT_NODE | CAN_ADVERTISE | CAN_RELAY
    Tags        []string       // policy tags (e.g. "tag:prod-web")
    UserID      *uuid.UUID     // owning user (nil for gateway/service devices)
    AdvertisedPrefixes []AdvertisedPrefix // (connectors only) LAN CIDRs advertised

    // 4via6-specific
    TenantPrefix netip.Prefix  // the tenant's ULA /48 (e.g. fd12:3456:789a::/48)

    // Timestamps
    EnrolledAt  time.Time
    RevokedAt   *time.Time     // nil if not revoked
}

// NodeStatus is the lifecycle status of a device.
type NodeStatus string

const (
    NodeActive  NodeStatus = "active"
    NodeRevoked NodeStatus = "revoked"
)

// AdvertisedPrefix is a LAN CIDR advertised by a connector.
type AdvertisedPrefix struct {
    IPv4CIDR   netip.Prefix  // the LAN CIDR (e.g. 192.168.1.0/24)
    Via6Prefix netip.Prefix  // the 4via6 overlay prefix (e.g. fd..:4636:0:7::/96)
}
```

**Node invariants:**
- Every node has exactly one `ID` (UUID PK).
- A client has `Kind == NodeClient`, `SiteID == 0`, `AdvertisedPrefixes` empty.
- A connector has `Kind == NodeConnector`, `SiteID > 0`, `AdvertisedPrefixes` non-empty.
- A gateway has `Kind == NodeGateway`, `SiteID == 0`, `AdvertisedPrefixes` empty.
- A revoked node has `Status == NodeRevoked`, `RevokedAt != nil`, and is excluded from all
  peer visibility computations.
- `WgPubkey` is the raw 32-byte Curve25519 key (not hex-encoded -- encode at the
  serialization boundary into `NetworkMap.peers[].wg_pubkey`).
- `OverlayIP` is the node's own fabric address in the tenant's `fd..::/64` prefix.

### 2.2 Edges

Edges represent **relationships** between nodes. Not all edges imply WireGuard tunnels --
some are routing relationships, others are policy-derived visibility.

```go
// EdgeKind classifies the relationship between two nodes.
type EdgeKind string

const (
    // WireGuard tunnel edges (bidirectional in practice).
    EdgeTunnel    EdgeKind = "tunnel"     // a WireGuard peer relationship

    // Connector-to-LAN routing relationship (directional).
    EdgeAdvertises EdgeKind = "advertises" // connector -> LAN CIDR

    // Gateway relationship (directional).
    EdgeGatewayTo EdgeKind = "gateway_to" // device -> gateway (all devices)
)

// Edge is a directed edge in the topology graph.
type Edge struct {
    From      uuid.UUID   // source node ID
    To        uuid.UUID   // target node ID
    Kind      EdgeKind    // tunnel | advertises | gateway_to
    Transport string      // "wireguard" | "masque" | "auto"
    MTU       uint16      // tunnel MTU (0 for non-tunnel edges)
    LatencyUs int64       // last measured RTT in microseconds (-1 if unknown)
    LossRate  float64     // packet loss rate [0.0, 1.0] (-1 if unknown)
    LastSeen  time.Time   // last time this edge was observed active
    Attrs     EdgeAttrs   // edge-type-specific attributes
}

// EdgeAttrs holds edge-type-specific metadata.
type EdgeAttrs struct {
    // For EdgeTunnel:
    AllowedIPs []netip.Prefix // WG AllowedIPs on this peer relationship
    Endpoint   string         // "host:port" of the peer's WireGuard endpoint

    // For EdgeAdvertises:
    Via6Routes []Via6Route    // 4via6 routes advertised through this edge

    // For EdgeGatewayTo:
    MasqueSNI  string         // MASQUE SNI for the gateway
}
```

**Edge invariants:**
- `EdgeTunnel` edges exist between every pair of nodes that have a WireGuard peer
  relationship (i.e., each knows the other's public key and has it in `allowed_ips`).
  In practice these are bidirectional at the graph level even though WireGuard's
  Cryptokey Routing is directional -- the coordinator models "these two nodes peer"
  as one relationship.
- `EdgeAdvertises` edges go from a connector to every other node whose policy grants
  visibility to that connector's advertised prefixes. The edge carries `Via6Routes`.
- `EdgeGatewayTo` edges go from every non-gateway node to the gateway node.
- A revoked node has zero outgoing or incoming `EdgeTunnel` edges.

### 2.3 Graph-level properties

```
For a tenant with:
  - 1 gateway
  - C connectors
  - D clients

The fully-connected topology (before policy filtering) has:
  - Nodes: 1 + C + D
  - EdgeTunnel edges: up to (1+C+D)*(C+D)/2  (full mesh of non-gateway nodes)
  - EdgeAdvertises edges: up to C * (D + C - 1)  (connections from each connector to visible peers)
  - EdgeGatewayTo edges: C + D  (every non-gateway node points to gateway)

With policy filtering (need-to-know), EdgeTunnel edges are dramatically reduced --
a client sees only the connectors and peers its policy grants.
```

---

## 3. Hydration from Postgres on Boot (P1-070.1)

### 3.1 Bootstrap sequence

```
Coordinator starts
  |
  --> For each active tenant:
        1. Query all non-revoked devices:
           SELECT id, tenant_id, kind, name, wg_pubkey, overlay_ip,
                  site_id, capabilities, tags, user_id, enrolled_at
           FROM devices WHERE tenant_id = $1 AND revoked_at IS NULL

        2. Query all connector advertised prefixes:
           SELECT c.device_id, ap.cidr, cs.site_id
           FROM connector_sites cs
           JOIN advertised_prefixes ap ON ap.connector_id = cs.connector_id
           JOIN devices c ON c.id = cs.device_id
           WHERE cs.tenant_id = $1

        3. Query tenant overlay pool:
           SELECT ula_prefix FROM overlay_pools WHERE tenant_id = $1

        4. Build the initial graph:
           a. Create a Node for each device
           b. Attach AdvertisedPrefixes to connector nodes (compute Via6Prefixes
              from ula_prefix + site_id)
           c. Create EdgeGatewayTo from each non-gateway node to the gateway
           d. Create EdgeTunnel edges based on the policy compiler output:
              - Compile the latest active policy for this tenant
              - For each (src, dst) pair in CompiledPolicy.VisibleTo:
                  if both src and dst are active (non-revoked):
                      create EdgeTunnel{From: src, To: dst}
                      populate AllowedIPs from CompiledPolicy.AllowedIPs[src][dst]
              - For each connector, create EdgeTunnel to every client authorized
                to see it (and vice versa)

        5. Subscribe to Redis Streams for ongoing events
```

### 3.2 No durable coordinator tables

The `TopologyGraph` is **purely in-memory**. It has no corresponding Postgres tables
in the coordinator's schema. On coordinator restart, the graph is rebuilt from scratch
by re-querying the source-of-truth tables (`devices`, `connector_sites`,
`advertised_prefixes`, `overlay_pools`, `policies`) and re-compiling policies.

This simplifies the architecture: there is exactly one durable state (Postgres), and
the in-memory graph is a derived cache. If the cache is lost (crash, restart), it is
rebuilt identically from the same durable state.

### 3.3 Partial hydration failure

If any single query in the bootstrap sequence fails (e.g., Postgres connection drops
mid-hydration), the coordinator MUST:

1. Log the error with the exact query that failed and the tenant ID.
2. Retry the FULL bootstrap for that tenant (not just the failed query -- the
   partial state may be internally inconsistent).
3. If retries are exhausted (bounded, e.g. 3 attempts with exponential backoff),
   surface the tenant as DEGRADED in the coordinator health endpoint and proceed
   with the remaining tenants.
4. A tenant that fails to hydrate MUST NOT serve any `buildMap()` requests -- return
   `ErrTopologyNotReady` to callers.

---

## 4. Event-Driven Incremental Mutation (P1-070.2)

### 4.1 Event stream (Redis Streams)

The topology graph subscribes to a Redis Stream per tenant:

```
Key: helix:vpn:<tenant_id>:events
```

Each event is a JSON object with a `type` field and type-specific payload. Events are
produced by the control-plane services (identity registry, IPAM, policy engine) at the
same time they commit to Postgres.

### 4.2 Event types and graph mutations

#### `device.enrolled`

```json
{
  "type": "device.enrolled",
  "device_id": "uuid",
  "tenant_id": "uuid",
  "kind": "client",
  "name": "alice-laptop",
  "wg_pubkey": "base64-encoded-32-bytes",
  "overlay_ip": "fd12:3456:789a::4",
  "site_id": 0,
  "capabilities": 0,
  "tags": [],
  "user_id": "uuid"
}
```

**Graph mutation:**
1. Create `Node{ID: device_id, ...}` and insert into node index.
2. Create `Edge{From: device_id, To: gateway_id, Kind: EdgeGatewayTo}`.
3. Recompile the tenant's policy (or use the just-compiled `CompiledPolicy` if the
   event carries a `compiled_policy_version` reference).
4. For every other node `P` where `CompiledPolicy.VisibleTo[P][device_id]` is non-empty:
   create `EdgeTunnel{From: P, To: device_id}` with the compiled `AllowedIPs`.
5. For every other node `P` where `CompiledPolicy.VisibleTo[device_id][P]` is non-empty:
   create `EdgeTunnel{From: device_id, To: P}` with the compiled `AllowedIPs`.

#### `device.revoked`

```json
{
  "type": "device.revoked",
  "device_id": "uuid"
}
```

**Graph mutation:**
1. Mark `Node.Status = NodeRevoked`, `Node.RevokedAt = now`.
2. Remove ALL `EdgeTunnel` edges where `From == device_id` OR `To == device_id`.
3. Remove the `EdgeGatewayTo` edge from this device.
4. Leave the node in the graph (for audit/history) but excluded from all
   `PeersVisibleTo()` traversals.

#### `connector.attached`

```json
{
  "type": "connector.attached",
  "device_id": "uuid",
  "tenant_id": "uuid",
  "site_id": 7,
  "advertised_prefixes": [
    {"ipv4_cidr": "192.168.1.0/24", "via6_prefix": "fd..:4636:0:7::/96"}
  ]
}
```

**Graph mutation:**
1. Update `Node.SiteID` and `Node.AdvertisedPrefixes`.
2. For each advertised prefix, create `EdgeAdvertises{From: connector, To: <visible peer>}`
   for every peer authorized to reach this connector's prefixes.
3. Recompile policy to regenerate `EdgeTunnel` edges involving this connector
   (its `AllowedIPs` now include the via6 /96 prefix).

#### `connector.detached`

```json
{
  "type": "connector.detached",
  "device_id": "uuid"
}
```

**Graph mutation:**
1. Remove all `EdgeAdvertises` edges from this connector.
2. Clear `Node.AdvertisedPrefixes`.
3. Recompile policy to remove the via6 /96 prefixes from affected `EdgeTunnel`
   `AllowedIPs`.

#### `policy.compiled`

```json
{
  "type": "policy.compiled",
  "tenant_id": "uuid",
  "policy_version": 42
}
```

**Graph mutation:**
1. Load the newly compiled `CompiledPolicy` (version 42).
2. Rebuild ALL `EdgeTunnel` edges for this tenant from the new `CompiledPolicy.VisibleTo`
   and `CompiledPolicy.AllowedIPs` maps.
3. Rebuild ALL `EdgeAdvertises` edges (visibility may have changed).
4. Bump `TopologyGraph.Generation`.

This is the heaviest mutation -- it touches every tunnel edge in the tenant. For a
1,000-device tenant with sparse ACLs, this is bounded at O(V * degree) where degree is
the average number of peers per node (typically small, << V).

### 4.3 Event ordering and idempotency

Events are processed in Redis Stream order within a tenant. Every mutation function is
**idempotent**:

- `device.enrolled` for an already-enrolled device: update the node in-place (no-op if
  identical; update fields if e.g. tags changed).
- `device.revoked` for an already-revoked device: no-op.
- `connector.attached` for an already-attached connector: update `AdvertisedPrefixes`
  in-place.
- `connector.detached` for an already-detached connector: no-op (edges already absent).
- `policy.compiled` for an already-applied version: no-op (version check).

---

## 5. Data Structures (Go)

### 5.1 TopologyGraph

```go
// internal/topology/graph.go

package topology

import (
    "sync"
    "sync/atomic"

    "github.com/google/uuid"
)

// TopologyGraph is the in-memory per-tenant topology graph.
//
// It is the single source of truth for "what is the current tenant topology?"
// at the coordinator layer. All buildMap() calls read from a Snapshot() of
// this graph. Mutations come from Redis Stream events via Apply().
//
// Concurrency: safe for concurrent use. Reads (Snapshot, PeersVisibleTo on
// a snapshot) are lock-free after acquiring the snapshot. Writes (Apply) take
// an exclusive lock. The snapshot pattern means long-running buildMap() calls
// never block event processing.
type TopologyGraph struct {
    mu         sync.RWMutex

    // Core data
    nodes      map[uuid.UUID]*Node      // node index by device ID
    adjOut     map[uuid.UUID]map[uuid.UUID]*Edge  // adjacency: adjOut[from][to] = edge
    adjIn      map[uuid.UUID]map[uuid.UUID]*Edge  // reverse index: adjIn[to][from] = edge

    // Metadata
    tenantID   uuid.UUID
    gatewayID  uuid.UUID                 // cached gateway node ID (O(1) lookup)
    generation int64                     // monotonically increasing; bumped on every mutation
    policyVer  int64                     // last applied policy version

    // Tenant overlay prefix (for 4via6 route computation)
    ulaPrefix  netip.Prefix
}

// NewTopologyGraph creates an empty topology graph for a tenant.
func NewTopologyGraph(tenantID uuid.UUID) *TopologyGraph {
    return &TopologyGraph{
        tenantID:   tenantID,
        nodes:      make(map[uuid.UUID]*Node),
        adjOut:     make(map[uuid.UUID]map[uuid.UUID]*Edge),
        adjIn:      make(map[uuid.UUID]map[uuid.UUID]*Edge),
        generation: 0,
    }
}
```

### 5.2 Snapshot -- immutable read-view

```go
// GraphSnapshot is an immutable point-in-time view of the topology graph.
//
// It is obtained by calling TopologyGraph.Snapshot(). The snapshot remains
// valid even as the underlying graph is mutated -- all data in a snapshot
// is a shallow copy of the graph state at snapshot time.
//
// Snapshots are read-only and concurrency-safe without any locks.
type GraphSnapshot struct {
    TenantID   uuid.UUID
    GatewayID  uuid.UUID
    Generation int64
    PolicyVer  int64
    UlaPrefix  netip.Prefix

    nodes      map[uuid.UUID]*Node              // shallow copy of node pointers
    adjOut     map[uuid.UUID]map[uuid.UUID]*Edge // shallow copy of adjacency
    adjIn      map[uuid.UUID]map[uuid.UUID]*Edge
}

// Snapshot returns an immutable point-in-time view of the graph.
//
// The returned snapshot is safe for concurrent read-only use without any
// locks. Long-running buildMap() calls hold a snapshot for their duration
// without blocking event processing (Apply).
func (g *TopologyGraph) Snapshot() *GraphSnapshot {
    g.mu.RLock()
    defer g.mu.RUnlock()

    // Shallow copy: the Node and Edge pointers themselves are immutable
    // after creation (mutations replace them, never modify in-place).
    nodes := make(map[uuid.UUID]*Node, len(g.nodes))
    for id, n := range g.nodes {
        nodes[id] = n
    }

    adjOut := make(map[uuid.UUID]map[uuid.UUID]*Edge, len(g.adjOut))
    for from, edges := range g.adjOut {
        m := make(map[uuid.UUID]*Edge, len(edges))
        for to, e := range edges {
            m[to] = e
        }
        adjOut[from] = m
    }

    adjIn := make(map[uuid.UUID]map[uuid.UUID]*Edge, len(g.adjIn))
    for to, edges := range g.adjIn {
        m := make(map[uuid.UUID]*Edge, len(edges))
        for from, e := range edges {
            m[from] = e
        }
        adjIn[to] = m
    }

    return &GraphSnapshot{
        TenantID:   g.tenantID,
        GatewayID:  g.gatewayID,
        Generation: g.generation,
        PolicyVer:  g.policyVer,
        UlaPrefix:  g.ulaPrefix,
        nodes:      nodes,
        adjOut:     adjOut,
        adjIn:      adjIn,
    }
}

// Node returns the node with the given ID, or nil if not present.
func (s *GraphSnapshot) Node(id uuid.UUID) *Node {
    return s.nodes[id]
}

// EdgesFrom returns all edges originating at the given node.
func (s *GraphSnapshot) EdgesFrom(from uuid.UUID) []*Edge {
    edgeMap := s.adjOut[from]
    if edgeMap == nil {
        return nil
    }
    edges := make([]*Edge, 0, len(edgeMap))
    for _, e := range edgeMap {
        edges = append(edges, e)
    }
    return edges
}

// Edge returns the edge from src to dst, or nil if none exists.
func (s *GraphSnapshot) Edge(from, to uuid.UUID) *Edge {
    if m := s.adjOut[from]; m != nil {
        return m[to]
    }
    return nil
}

// IsStale returns true if this snapshot is older than the current graph generation.
func (s *GraphSnapshot) IsStale(currentGen int64) bool {
    return s.Generation < currentGen
}
```

### 5.3 Apply -- event-driven mutation

```go
// TopologyEvent is a parsed event from the Redis Stream.
type TopologyEvent struct {
    Type    string          // "device.enrolled" | "device.revoked" | ...
    Payload json.RawMessage // type-specific JSON payload
}

// Apply processes a topology event and mutates the graph in-place.
//
// Apply takes an exclusive write lock for the duration of the mutation.
// It returns the new generation number. If the event is a no-op (e.g.
// revoking an already-revoked device), the generation is still bumped
// to signal that the event was received and processed.
func (g *TopologyGraph) Apply(ev *TopologyEvent) (int64, error) {
    g.mu.Lock()
    defer g.mu.Unlock()

    switch ev.Type {
    case "device.enrolled":
        return g.applyDeviceEnrolled(ev.Payload)
    case "device.revoked":
        return g.applyDeviceRevoked(ev.Payload)
    case "connector.attached":
        return g.applyConnectorAttached(ev.Payload)
    case "connector.detached":
        return g.applyConnectorDetached(ev.Payload)
    case "policy.compiled":
        return g.applyPolicyCompiled(ev.Payload)
    default:
        return g.generation, fmt.Errorf("topology: unknown event type %q", ev.Type)
    }
}

func (g *TopologyGraph) bumpGeneration() int64 {
    g.generation++
    return g.generation
}
```

Each `apply*` method:

1. Unmarshals the type-specific payload.
2. Performs the graph mutation (add/remove/update nodes and edges as described in §4.2).
3. Replaces edges atomically (delete old edge, insert new edge) -- never mutate an
   edge in-place (snapshot readers may hold a pointer to the old edge).
4. Calls `bumpGeneration()`.

### 5.4 Concurrency model

```
                    TopologyGraph
                    +-----------+
   Redis Stream --->|  Apply()  |  (exclusive write lock, RWMutex.Lock)
   Event Consumer   +-----------+
                          |
                    +-----------+
   buildMap() ----->| Snapshot()|  (shared read lock, RWMutex.RLock)
   (multiple calls) +-----------+
                          |
                    +-----------+
   PeersVisibleTo ->| GraphSnapshot |  (no locks -- immutable)
   (on snapshot)    +---------------+
```

**Key properties:**
- `Apply()` holds `mu.Lock()` -- at most one writer at a time. Event processing is
  serialized within a tenant, which is correct because Redis Streams deliver events
  in order.
- `Snapshot()` holds `mu.RLock()` -- multiple readers can take snapshots concurrently
  with each other, but not concurrently with a writer.
- Once a `GraphSnapshot` is obtained, all operations on it are **lock-free** -- the
  snapshot's maps and the `Node`/`Edge` pointers within them are immutable.
- `Apply()` never modifies existing `Node` or `Edge` structs in-place. When a field
  changes, it allocates a new struct and replaces the pointer in `g.nodes` or
  `g.adjOut`/`g.adjIn`. This is what makes snapshots safe: a snapshot reader holds
  pointers to the old structs, which remain valid even after the writer replaces them.

---

## 6. Consistency Guarantees

### 6.1 Read-your-writes within a single event

Within a single `Apply()` call, all mutations are applied before the lock is released.
A `Snapshot()` taken after the `Apply()` returns will see all mutations from that event.
This is guaranteed by the RWMutex: the writer releases `Lock()` before any reader's
`RLock()` can proceed.

### 6.2 Eventual consistency across event batches

Events are processed in Redis Stream order. The graph is eventually consistent with
Postgres: every event reflects a committed Postgres transaction. If the event consumer
falls behind (e.g., after a coordinator restart), it catches up by replaying events
from the last consumed stream ID.

### 6.3 Generation counter for stale snapshot detection

Every mutation bumps `TopologyGraph.generation` (monotonic `int64`). A `GraphSnapshot`
records the generation at snapshot time. A `buildMap()` caller can check
`snapshot.IsStale(graph.Generation())` to detect whether its snapshot is outdated and
optionally re-snapshot. This is useful for long-running map computations that want to
ensure they are working with fresh topology data.

### 6.4 Snapshot immutability guarantee

**Invariant:** `Node` and `Edge` structs, once inserted into the graph, are NEVER
mutated in-place. When a field changes (e.g., a connector's `AdvertisedPrefixes`
are updated), a NEW `Node` struct is allocated with the updated field, and the
pointer in `g.nodes[id]` is replaced. The old struct remains valid for any snapshot
reader still holding a reference to it.

This is the **copy-on-write** pattern applied at the struct level, not the page level.
It is cheap because `Node` and `Edge` are small (a few hundred bytes each), and the
number of concurrent snapshots is bounded by the number of concurrent `buildMap()`
calls (typically low, << 10).

---

## 7. Memory Bounds

### 7.1 Per-entity sizes

| Entity | Approximate size (Go) | Notes |
|---|---|---|
| `Node` struct | ~300 bytes | UUIDs (2x 16B), string headers, netip.Addr (16B), slices (24B header each) |
| `Edge` struct | ~200 bytes | UUIDs (2x 16B), EdgeAttrs (variable, ~100B avg) |
| Map entry overhead | ~80 bytes | Go map bucket overhead per key-value pair |
| Snapshot overhead | ~(N+E) * 8 bytes | Shallow copy: map entries hold pointers |

### 7.2 Tenant size estimates

| Tenant size | Nodes | Estimated edges | Estimated memory |
|---|---|---|---|
| Small (50 devices) | 50 | ~200 | <1 MiB |
| Medium (200 devices) | 200 | ~1,000 | ~3 MiB |
| Large (1,000 devices) | 1,000 | ~5,000 | ~8 MiB |
| X-Large (5,000 devices) | 5,000 | ~25,000 | ~40 MiB |

These estimates assume sparse ACLs (each device sees ~5 peers on average). With
dense ACLs (full mesh, every device sees every other), edge count grows as O(V^2)
and memory for 1,000 devices approaches ~50 MiB. The coordinator SHOULD monitor
graph memory and emit warnings when a tenant exceeds 50 MiB.

### 7.3 Garbage collection of removed nodes

When a node is revoked:
- It is **not deleted** from `g.nodes` -- it stays with `Status = NodeRevoked` for
  audit/history (the `revoked_at` timestamp is preserved).
- Its edges ARE removed from `g.adjOut` and `g.adjIn`.

When ALL snapshots that reference a replaced `Node` or `Edge` struct are released
(go out of scope), Go's garbage collector reclaims the old structs. Since snapshots
are short-lived (the duration of a `buildMap()` call, typically <100 ms), old structs
are reclaimed promptly.

The graph itself only grows when new devices enroll. For long-running coordinators
with high device churn, the node map may accumulate many revoked nodes. A future
optimization (P2) could periodically compact the node map by archiving revoked nodes
older than N days, but this is not needed for Phase 1.

---

## 8. Integration with buildMap (P1-071)

### 8.1 How the topology graph feeds into buildMap

```go
// internal/topology/buildmap.go

// BuildMap computes a NetworkMap for a given device from the topology snapshot.
//
// This is the coordinator's core function: it answers "what should device D's
// WireGuard configuration look like right now?"
func BuildMap(snap *GraphSnapshot, deviceID uuid.UUID, compiledPolicy *policy.CompiledPolicy) (*helixcore.NetworkMap, error) {
    node := snap.Node(deviceID)
    if node == nil {
        return nil, ErrDeviceNotFound
    }
    if node.Status == NodeRevoked {
        return nil, ErrDeviceRevoked
    }

    gateway := snap.Node(snap.GatewayID)
    if gateway == nil {
        return nil, ErrGatewayNotFound
    }

    // 1. Determine which peers are visible to this device.
    visiblePeers := PeersVisibleTo(snap, deviceID, compiledPolicy)

    // 2. Build the NetworkMap.
    nm := &helixcore.NetworkMap{
        Self: helixcore.SelfConfig{
            OverlayIP: node.OverlayIP.String(),
            Transport: "auto",
        },
        Gateway: helixcore.Gateway{
            Endpoint:  gatewayEndpoint(gateway),
            WgPubkey:  hex.EncodeToString(gateway.WgPubkey[:]),
            MasqueSNI: gatewayMasqueSNI(snap, gateway),
        },
        Peers: make([]helixcore.Peer, 0, len(visiblePeers)),
        DNS:   []string{snap.GatewayID.String()}, // placeholder: gateway overlay IP
    }

    for _, peer := range visiblePeers {
        // Filter out revoked peers (defense in depth -- policy should already exclude them).
        if peer.Status == NodeRevoked {
            continue
        }

        edge := snap.Edge(deviceID, peer.ID)
        allowedIPs := make([]string, 0)
        if edge != nil {
            for _, prefix := range edge.Attrs.AllowedIPs {
                allowedIPs = append(allowedIPs, prefix.String())
            }
        }

        // Always include the peer's own overlay IP.
        allowedIPs = append(allowedIPs, peer.OverlayIP.String()+"/128")

        nm.Peers = append(nm.Peers, helixcore.Peer{
            Name:       peer.Name,
            WgPubkey:   hex.EncodeToString(peer.WgPubkey[:]),
            AllowedIPs: allowedIPs,
        })
    }

    return nm, nil
}
```

### 8.2 PeersVisibleTo -- walks the graph filtered by ACL visibility

```go
// PeersVisibleTo returns the set of peers visible to the given device,
// as determined by the compiled policy (need-to-know, T5).
//
// It walks the snapshot's adjacency list for the device and filters by
// the CompiledPolicy.VisibleTo map.
func PeersVisibleTo(snap *GraphSnapshot, deviceID uuid.UUID, cp *policy.CompiledPolicy) []*Node {
    // Fast path: use the pre-computed visibility map from the policy compiler.
    visibleIDs, ok := cp.VisibleTo[deviceID]
    if !ok {
        // Device has no visibility entries -- it sees no peers (default-deny).
        return nil
    }

    peers := make([]*Node, 0, len(visibleIDs))
    for _, peerID := range visibleIDs {
        if peer := snap.Node(peerID); peer != nil && peer.Status == NodeRevoked {
            peers = append(peers, peer)
        }
    }

    // Sort by device name for deterministic output (P2 invariant from policies_spec).
    sort.Slice(peers, func(i, j int) bool {
        return peers[i].Name < peers[j].Name
    })

    return peers
}
```

### 8.3 Alternative: graph-walk visibility (without pre-computed policy)

If the policy compiler has NOT pre-computed `VisibleTo` (e.g., during boot before
the first policy compilation), `PeersVisibleTo` can fall back to a graph walk:

```go
// PeersVisibleToGraphWalk computes visibility by walking the topology graph
// and checking each edge's existence (no policy filtering -- raw topology).
//
// This is a development/debug fallback. In production, always use the
// CompiledPolicy.VisibleTo pre-computed map.
func PeersVisibleToGraphWalk(snap *GraphSnapshot, deviceID uuid.UUID) []*Node {
    edges := snap.EdgesFrom(deviceID)
    peers := make([]*Node, 0, len(edges))
    seen := make(map[uuid.UUID]bool)

    for _, e := range edges {
        if e.Kind != EdgeTunnel {
            continue
        }
        peerID := e.To
        if seen[peerID] {
            continue
        }
        seen[peerID] = true

        if peer := snap.Node(peerID); peer != nil && peer.Status == NodeRevoked {
            peers = append(peers, peer)
        }
    }

    sort.Slice(peers, func(i, j int) bool {
        return peers[i].Name < peers[j].Name
    })
    return peers
}
```

---

## 9. Edge Cases

### 9.1 Concurrent enroll + revoke for the same device

**Scenario:** Two events arrive in rapid succession: `device.enrolled` for device X,
then `device.revoked` for device X (or vice versa).

**Handling:** Events are processed in Redis Stream order. The second event sees the
state produced by the first and applies correctly:
- If enrolled-then-revoked: the enroll creates the node and edges; the revoke sets
  `Status = NodeRevoked` and removes edges. The final state is a revoked node with
  no edges -- correct.
- If revoked-then-enrolled: the revoke sees no existing node (no-op); the enroll
  creates a fresh node with `active` status. The final state is an active node --
  correct (the device was re-enrolled).

**Idempotency guarantee:** Both `applyDeviceEnrolled` and `applyDeviceRevoked` are
idempotent. If the same event is delivered twice (at-least-once delivery), the
second application is a no-op.

### 9.2 Rapid connector attach/detach

**Scenario:** A connector rapidly attaches, detaches, re-attaches with different
advertised prefixes.

**Handling:** Each event is processed in order. The graph state after each event is
internally consistent. A `buildMap()` caller that holds a snapshot from before the
first attach will see the old topology; a caller that snapshots after the last
attach sees the final topology. No intermediate inconsistent state is observable
because each `Apply()` is atomic (protected by the write lock).

### 9.3 Partial hydration failure

**Scenario:** During bootstrap hydration (§3.1), the devices query succeeds but the
advertised_prefixes query fails (Postgres connection drops).

**Handling:** The coordinator MUST NOT serve a partially-hydrated graph. It retries
the full bootstrap for the affected tenant. If retries are exhausted, the tenant is
marked DEGRADED. See §3.3.

### 9.4 Event consumer lag after restart

**Scenario:** The coordinator restarts after being down for 10 minutes. During the
outage, 50 events (device enrollments, policy changes) were published to the Redis
Stream.

**Handling:** On restart, the coordinator hydrates the graph from Postgres (which
reflects the CURRENT state -- all 50 events are already committed). It then sets
its Redis Stream consumer position to the latest entry. The 50 events are NOT
replayed because their effects are already captured in the Postgres state. This
is a "hydrate from source-of-truth, then subscribe to new events" pattern --
avoiding redundant replay of already-applied events.

### 9.5 Policy compilation failure during event processing

**Scenario:** A `policy.compiled` event arrives but the referenced policy version
fails to load from the policy store (e.g., the `policies` table row was deleted
between the event being published and the coordinator processing it).

**Handling:**
1. Log the error with the tenant ID and policy version.
2. Retry loading the policy (bounded, 3 attempts with backoff).
3. If retries exhausted, leave the graph as-is (with the previous policy version)
   and surface the tenant as DEGRADED.
4. The coordinator MUST NOT apply a partial policy update -- either the entire
   new policy is applied atomically, or none of it is.

### 9.6 Gateway node not found

**Scenario:** A tenant's graph has no gateway node (e.g., the gateway device row was
deleted or the tenant was misprovisioned).

**Handling:** The graph is in an invalid state. `buildMap()` returns
`ErrGatewayNotFound` for all devices in this tenant. The coordinator's health
endpoint reports the tenant as UNHEALTHY. This is a provisioning error that
requires operator intervention.

---

## 10. File Layout

```
helix-go/internal/topology/
  graph.go        -- TopologyGraph, Node, Edge, NodeKind, NodeStatus, EdgeKind,
                     EdgeAttrs, GraphSnapshot, AdvertisedPrefix, Via6Route types
  snapshot.go     -- GraphSnapshot methods: Node(), EdgesFrom(), Edge(), IsStale()
  apply.go        -- Apply(), applyDeviceEnrolled(), applyDeviceRevoked(),
                     applyConnectorAttached(), applyConnectorDetached(),
                     applyPolicyCompiled()
  hydrate.go      -- Hydrate(tenantID, db) -- bootstrap graph from Postgres
  events.go       -- TopologyEvent, Subscribe(tenantID, redisClient), event consumer loop
  buildmap.go     -- BuildMap(), PeersVisibleTo(), PeersVisibleToGraphWalk()
  graph_test.go   -- unit tests: hydration, incremental mutations, snapshots
  snapshot_test.go -- snapshot consistency tests (concurrent read + write)
  buildmap_test.go -- buildMap() output tests against fixtures
```

---

## 11. Frozen Contracts (Must Not Break)

| Contract | Where defined | What it means |
|---|---|---|
| `Node.ID` is the device UUID PK | `devices.id` column | The topology graph's primary key -- never change |
| `Edge.From` / `Edge.To` semantics | This document §2.2 | Directional: `From` is source, `To` is target |
| `GraphSnapshot` immutability | This document §6.4 | Node/Edge structs never mutated in-place after snapshot |
| `Apply()` serializes within a tenant | This document §5.4 | At most one writer per tenant at a time |
| Event type vocabulary | This document §4.2 | Closed set; new event types require a new `Apply()` case |
| `buildMap()` output shape matches `map.rs` `NetworkMap` | [map.rs] + [policies_spec §11] | `NetworkMap.peers[].allowed_ips` is `[]string` of CIDRs |
| Redis Stream key format | This document §4.1 | `helix:vpn:<tenant_id>:events` |

---

## 12. Sources Verified

| Source | URL / Reference | Date verified |
|---|---|---|
| HelixVPN NetworkMap schema (`map.rs`) | `submodules/helix_core/crates/helix-core/src/map.rs` | 2026-07-08 |
| HelixVPN Orchestrator (`orchestrator.rs`) | `submodules/helix_core/crates/helix-orch/src/orchestrator.rs` | 2026-07-08 |
| HelixVPN 4via6 IPAM Design | `docs/design/ipam/DESIGN.md` (Rev 1) | 2026-07-08 |
| HelixVPN Policy Spec Design | `docs/design/policies_spec/DESIGN.md` (Rev 1) | 2026-07-08 |
| Helix Go control-plane overview | `submodules/helix_go/CLAUDE.md` | 2026-07-08 |
| GoGraph sharded adjacency list | `github.com/FlavioCFOliveira/GoGraph/graph/adjlist` | 2026-07-08 |
| Go concurrent graph data structures (RWMutex patterns) | `pkg.go.dev/github.com/mstgnz/data-structures/graph` | 2026-07-08 |
| lora-db snapshot-based graph engine (ArcSwap pattern) | `github.com/lora-db/lora/docs/architecture/graph-engine.md` | 2026-07-08 |
| Tailscale -- How NAT traversal works (topology design) | `https://tailscale.com/blog/how-nat-traversal-works` | 2026-07-08 |
| Tailscale -- A deep dive into the Tailscale control plane | `https://tailscale.com/blog/how-tailscale-works` | 2026-07-08 |
| WireGuard whitepaper (Cryptokey Routing) | `https://www.wireguard.com/papers/wireguard.pdf` | 2026-07-08 |
| wg-meshconf -- WireGuard mesh configuration tool | `pkg.go.dev/github.com/intrand/wg-meshify` | 2026-07-08 |

---

*End of HelixVPN In-Memory Per-Tenant Topology Graph Design. For implementation, start with:
1. `internal/topology/graph.go` -- core types (Node, Edge, TopologyGraph, GraphSnapshot)
2. `internal/topology/hydrate.go` -- Postgres bootstrap
3. `internal/topology/apply.go` -- event-driven mutations
4. `internal/topology/buildmap.go` -- BuildMap() + PeersVisibleTo()
5. `internal/topology/events.go` -- Redis Streams consumer*