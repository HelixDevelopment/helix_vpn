# buildMap -- Per-Node NetworkMap Computation Design (HVPN-P1-071)

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Description:** Concrete design for the `buildMap` function that computes a
per-node `NetworkMap` from topology, policy, and identity state. This is the
coordinator-side function that produces the desired-state document consumed by
`map.rs`'s reconciler on every edge device.
**Authority:** design-authority
**Scope:** `helix-go/internal/coordinator/buildmap.go` (Go, control plane)

---

## 1. Problem

Every node in the HelixVPN overlay needs its own personal view of the network --
a `NetworkMap` that enumerates:

- **Itself** -- its own overlay identity (IP, hostname, public key, transport)
- **Its gateway** -- the bootstrap gateway for initial connection and internet egress
- **Its peers** -- every other node it is authorized to see, each annotated with
  the exact CIDR prefixes (WireGuard `AllowedIPs`) the policy grants
- **Its DNS configuration** -- the overlay DNS resolver addresses

The map must be **need-to-know**: a node never learns about peers it is not
authorized to reach (default-deny, P1 from [policies_spec]). The map must be
**deterministic**: same inputs produce byte-identical output, so the reconciler
can diff against the previous map and apply only the delta. And it must be
**fast**: < 100 ms for a 1,000-device tenant so the coordinator can push maps
to every affected node within the convergence SLO (p99 < 1 s).

---

## 2. Function signature

```go
// Package coordinator implements the per-node NetworkMap builder.
package coordinator

import (
    "context"
    "net/netip"

    "github.com/google/uuid"
    "github.com/helixdevelopment/helix-go/internal/policy"
    "github.com/helixdevelopment/helix-go/internal/topology"
)

// buildMap computes the per-node NetworkMap for a single device.
//
// The returned NetworkMap is the desired-state document that the
// WatchNetworkMap RPC stream pushes to the node. The Rust edge
// (helix_core map.rs) reconciles its local state against this
// desired state, adding/removing WireGuard peers as needed.
//
// Pure function -- no side effects, no DB access inside the hot path.
// All data is pre-resolved by the caller (topology graph already
// built, CompiledPolicy already compiled, Node already hydrated from
// the devices table + IPAM).
func buildMap(
    node   *topology.Node,
    graph  *topology.Graph,
    policy *policy.CompiledPolicy,
) (*NetworkMap, error)
```

### 2.1 Input types

| Parameter | Type | Source | Description |
|---|---|---|---|
| `node` | `*topology.Node` | `devices` table + IPAM | The node this map is FOR (the "self" of the map) |
| `graph` | `*topology.Graph` | Event-sourced in-memory graph | Full topology -- all nodes, their endpoints, their advertised prefixes |
| `policy` | `*policy.CompiledPolicy` | `policies.spec` compiler | Pre-compiled ACL output: who-can-see-whom + per-pair AllowedIPs + exit-node set |

### 2.2 Return type

```go
// NetworkMap is the per-node desired-state document.
// Serialized as JSON and pushed via WatchNetworkMap; deserialized
// by the Rust edge (helix_core/crates/helix-core/src/map.rs).
type NetworkMap struct {
    Self      SelfConfig   `json:"self"`
    Gateway   Gateway      `json:"gateway"`
    Peers     []Peer       `json:"peers"`
    DNS       []string     `json:"dns"`
    Transport Transport    `json:"transport"`
}

// SelfConfig describes this node's own overlay identity.
type SelfConfig struct {
    NodeID    string   `json:"node_id"`     // UUID v7, e.g. "0192abcd-..."
    OverlayIP string   `json:"overlay_ip"`  // e.g. "fd12:3456:789a::4/128"
    Hostname  string   `json:"hostname"`    // human-readable, e.g. "alice-laptop"
    PublicKey string   `json:"public_key"`  // hex-encoded WireGuard public key (64 hex chars)
}

// Gateway describes the bootstrap gateway node.
type Gateway struct {
    NodeID    string `json:"node_id"`     // gateway's device UUID
    Endpoint  string `json:"endpoint"`    // e.g. "gw.example.com:443"
    PublicKey string `json:"public_key"`  // hex-encoded WG public key
    MasqueSNI string `json:"masque_sni"`  // SNI for MASQUE transport, e.g. "cdn.example.com"
    OverlayIP string `json:"overlay_ip"`  // gateway's overlay address, e.g. "fd12:3456:789a::1"
}

// Peer describes a single overlay peer visible to this node.
type Peer struct {
    ID         string   `json:"id"`          // device UUID
    PublicKey  string   `json:"public_key"`  // hex-encoded WG public key
    Endpoints  []string `json:"endpoints"`   // reachable underlay endpoints (host:port)
    AllowedIPs []string `json:"allowed_ips"` // CIDR prefixes for WG AllowedIPs (sorted, deduped)
    Transport  string   `json:"transport"`   // "plain-udp" or "masque-quic-standin"
}

// Transport is an enum of supported transport protocols.
type Transport string

const (
    TransportPlainUDP       Transport = "plain-udp"
    TransportMasqueQUIC     Transport = "masque-quic-standin"
    TransportAuto           Transport = "auto"
)
```

### 2.3 Relationship to map.rs schema

The Rust `map.rs` `NetworkMap` (Phase-0 spike) is a **subset** of this schema:

| map.rs field | Go field | Notes |
|---|---|---|
| `self.overlay_ip` | `Self.OverlayIP` | Same |
| `self.transport` | `Self.Transport` (moved to top-level `transport`) | Phase-0 had it nested; Phase-1 promotes it |
| `gateway.endpoint` | `Gateway.Endpoint` | Same |
| `gateway.wg_pubkey` | `Gateway.PublicKey` | Renamed for clarity |
| `gateway.masque_sni` | `Gateway.MasqueSNI` | Same |
| `peer.name` | `Peer.ID` + `Self.Hostname` | Phase-0 used name; Phase-1 uses stable UUID + separate hostname |
| `peer.wg_pubkey` | `Peer.PublicKey` | Renamed for clarity |
| `peer.allowed_ips` | `Peer.AllowedIPs` | Same (Vec<String> of CIDRs) |
| (absent) | `Peer.Endpoints` | New -- underlay reachability info |
| (absent) | `Peer.Transport` | New -- per-peer transport selection |
| `dns` | `DNS` | Same |

The Phase-1 `map.rs` will be extended to match this schema before
`WatchNetworkMap` goes live. The reconciler (`reconcile()`) continues to work
on the subset of fields it already understands; new fields are additive.

---

## 3. Peer computation algorithm

### 3.1 High-level algorithm

```
func buildMap(node, graph, policy) -> NetworkMap:
    1.  self    = buildSelf(node)
    2.  gateway = resolveGateway(graph, node.TenantID)
    3.  peers   = []Peer{}
    4.  visibleSet = policy.VisibleTo[node.ID]   // pre-computed by policy compiler
    5.  allowedMap = policy.AllowedIPs[node.ID]  // map[peerID][]netip.Prefix

    6.  for each candidate in graph.Nodes:
            if candidate.ID == node.ID:   continue    // skip self
            if candidate.ID not in visibleSet: continue  // ACL denied

            allowedIPs = resolveAllowedIPs(candidate, allowedMap[candidate.ID])
            isExit     = policy.ExitNodes contains candidate.ID

            if isExit:
                allowedIPs = append(allowedIPs, "0.0.0.0/0", "::/0")

            allowedIPs = deduplicateAndSort(allowedIPs)

            peer = Peer{
                ID:         candidate.ID,
                PublicKey:  candidate.WgPubkeyHex(),
                Endpoints:  candidate.EndpointCandidates(),
                AllowedIPs: allowedIPs,
                Transport:  selectTransport(node, candidate),
            }
            peers = append(peers, peer)

    7.  sort peers by ID (UUID byte-order) for deterministic output
    8.  dns = []string{gateway.OverlayIP}   // overlay DNS resolver at gateway

    9.  return NetworkMap{self, gateway, peers, dns, node.Transport}
```

### 3.2 Visibility filter (need-to-know)

The `policy.VisibleTo[node.ID]` map is **pre-computed** by the policy compiler
(`Compile()` in [policies_spec §7]). It is a `map[uuid.UUID][]uuid.UUID` where
`VisibleTo[source]` is the set of peer device IDs the source is authorized to
see. This is an O(1) map lookup per candidate -- no ACL rule evaluation happens
inside `buildMap`.

A peer NOT in `VisibleTo[node.ID]` is **silently absent** from the map. The
node never learns it exists. This is need-to-know enforced by construction:
the map is the ONLY source of peer information the node receives, and an absent
peer is unreachable regardless of what the node might guess.

### 3.3 AllowedIPs resolution

For each visible peer, the `AllowedIPs` are resolved from two sources:

**Source A -- compiled policy CIDRs:**

```go
// allowedMap is CompiledPolicy.AllowedIPs[node.ID][peer.ID]
// It is a []netip.Prefix pre-computed by the policy compiler.
//
// Example: if an ACL rule grants "group:admins" -> "warehouse-cams:*",
// and warehouse-cams resolves to connectorX advertising 10.10.0.0/24,
// then AllowedIPs[aliceDevice][connectorX] = [10.10.0.0/24]
func resolveAllowedIPs(
    peer       *topology.Node,
    policyCIDRs []netip.Prefix,
) []string {
    cidrs := make([]string, 0, len(policyCIDRs))

    for _, p := range policyCIDRs {
        cidrs = append(cidrs, p.String())
    }

    // If the peer is a connector, also include its 4via6 overlay prefix
    // so WG routes 4via6-destined packets to this peer.
    if peer.Kind == topology.KindConnector && peer.Via6Prefix.IsValid() {
        cidrs = append(cidrs, peer.Via6Prefix.String())
    }

    // Always include the peer's own node overlay address (/128)
    // so control-plane traffic (DNS, health checks) reaches it.
    cidrs = append(cidrs, peer.OverlayIP.String())

    return cidrs
}
```

**Source B -- 4via6 overlay prefix (connectors only):**

When a peer is a connector (advertises LAN prefixes), its 4via6 `/96` site
prefix is always included in `AllowedIPs`. This is the mechanism by which
4via6-destined packets are routed through the correct connector:

```
Connector warehouse, site ID 7:
  AllowedIPs includes: "fd12:3456:789a:7::/96"
  → Any packet destined for fd12:3456:789a:7::c0a8:10a (4via6 of 192.168.1.10)
    matches this /96 and is encrypted to the connector's WG public key.
```

The 4via6 prefix is always included regardless of policy -- it is the
**routing** layer, not the **authorization** layer. Authorization is enforced
by the policy compiler: if a node is not authorized to reach a connector's
LAN, the connector is simply not in `VisibleTo[node.ID]` at all.

**Source C -- exit node default routes:**

If the peer is in `policy.ExitNodes`, the default routes `0.0.0.0/0` and
`::/0` are added. This tells WireGuard to route ALL internet-destined traffic
through this peer, implementing the full-tunnel VPN exit node pattern.

### 3.4 Deduplication and sorting

```go
func deduplicateAndSort(cidrs []string) []string {
    seen := make(map[string]struct{}, len(cidrs))
    unique := make([]string, 0, len(cidrs))

    for _, c := range cidrs {
        // Normalize: parse and re-stringify so "10.0.0.0/24" and
        // "10.0.0.0/24" (identical) and "10.0.0.0/24" vs
        // "10.0.0.0/24" (trailing space) collapse to one entry.
        prefix, err := netip.ParsePrefix(strings.TrimSpace(c))
        if err != nil {
            continue // skip malformed CIDRs (should never happen from compiled policy)
        }
        normalized := prefix.String()
        if _, ok := seen[normalized]; !ok {
            seen[normalized] = struct{}{}
            unique = append(unique, normalized)
        }
    }

    // Sort lexicographically for deterministic output.
    // IPv6 prefixes sort after IPv4 because 'f' > '0' in ASCII,
    // but within each family the sort is prefix-length-aware
    // (shorter prefix = broader = sorts first).
    sort.Strings(unique)
    return unique
}
```

### 3.5 Transport selection

Each peer carries a `transport` field indicating which protocol to use.
Selection logic:

```go
func selectTransport(self, peer *topology.Node) string {
    // If the peer is on the same underlay subnet, prefer plain UDP
    // (no need for MASQUE overhead).
    if sameUnderlaySubnet(self, peer) {
        return string(TransportPlainUDP)
    }

    // If the peer has a MASQUE endpoint, prefer MASQUE.
    if peer.HasMasqueEndpoint() {
        return string(TransportMasqueQUIC)
    }

    // Fall back to plain UDP (NAT traversal via STUN).
    return string(TransportPlainUDP)
}
```

The `self.transport` field (top-level) is a **preference** hint ("auto",
"wireguard", "masque"), not a hard constraint on peers. Each peer's transport
is selected independently based on reachability. The top-level hint is used
when the connector/dialer has no peer-specific preference.

### 3.6 Gateway resolution

```go
func resolveGateway(graph *topology.Graph, tenantID uuid.UUID) Gateway {
    gw := graph.GatewayNode(tenantID)
    if gw == nil {
        // A tenant without a gateway has no overlay -- every node
        // is isolated. Return an empty Gateway; the caller handles
        // this as a SKIP-with-reason (tenant not yet provisioned).
        return Gateway{}
    }

    return Gateway{
        NodeID:    gw.ID.String(),
        Endpoint:  gw.PrimaryEndpoint(),     // e.g. "gw.example.com:443"
        PublicKey: gw.WgPubkeyHex(),
        MasqueSNI: gw.MasqueSNI,             // from gateway config, e.g. "cdn.example.com"
        OverlayIP: gw.OverlayIP.String(),    // e.g. "fd12:3456:789a::1"
    }
}
```

---

## 4. Exit node selection

When a peer is in `policy.ExitNodes`, the default routes `0.0.0.0/0` and
`::/0` are appended to its `AllowedIPs`. WireGuard interprets these as
"route all traffic through this peer."

### 4.1 Exit node constraints

- **Connectors are excluded.** The policy compiler rejects exit-node selectors
  that resolve to connectors (`EXIT_NODE_IS_CONNECTOR` error in
  [policies_spec §2.6]). `buildMap` asserts this as a defense-in-depth check:
  if `peer.Kind == KindConnector && policy.ExitNodes contains peer.ID`, the
  function returns an error rather than silently adding default routes to a
  connector.

- **Revoked devices are excluded.** The policy compiler silently drops revoked
  devices from the exit-node set. `buildMap` double-checks: if
  `peer.RevokedAt != nil`, the peer is never added as an exit node regardless
  of what `policy.ExitNodes` says.

- **Self is excluded.** A node cannot be its own exit node. The self-skip at
  algorithm step 6 handles this implicitly.

### 4.2 Multiple exit nodes

When multiple devices are in the exit-node set, each gets `0.0.0.0/0` and
`::/0` in its `AllowedIPs`. The client's WireGuard configuration will have
multiple peers with overlapping default routes. WireGuard's `AllowedIPs`
longest-prefix-match selects the peer with the most specific route; since
both default routes are `/0`, the kernel's route selection (metric, table
order) determines which exit node is used. The HelixVPN client MAY implement
exit-node preference ordering in a future phase; Phase 1 accepts
kernel-default behavior.

---

## 5. Determinism

Same inputs MUST produce byte-identical `NetworkMap`. This is load-bearing for
the reconciler: if two calls to `buildMap` with the same node/graph/policy
produce different JSON (different key order, different peer order), the
reconciler sees a diff and applies a no-op change, causing churn on every
map push.

### 5.1 Determinism guarantees

| Aspect | Mechanism |
|---|---|
| Peer ordering | Sort by `Peer.ID` (UUID bytes, not string) before marshaling |
| AllowedIPs ordering | Sort lexicographically after dedup (see §3.4) |
| Endpoints ordering | Sort lexicographically (host:port strings) |
| JSON key ordering | `encoding/json` sorts keys alphabetically by default in Go |
| Map iteration | Never iterate over `map[K]V` directly -- always collect keys, sort, then iterate |
| UUID formatting | Always use `uuid.UUID.String()` (canonical 8-4-4-4-12 hex format) |
| IP formatting | Always use `netip.Prefix.String()` / `netip.Addr.String()` (canonical form) |

### 5.2 Property test

```go
func TestBuildMapIsDeterministic(t *testing.T) {
    node  := fixtureNode("alice-laptop")
    graph := fixtureThreeNodeGraph()
    policy := fixtureCompiledPolicy(graph)

    map1, err1 := buildMap(node, graph, policy)
    map2, err2 := buildMap(node, graph, policy)

    require.NoError(t, err1)
    require.NoError(t, err2)

    json1, _ := json.MarshalIndent(map1, "", "  ")
    json2, _ := json.MarshalIndent(map2, "", "  ")

    assert.Equal(t, json1, json2, "buildMap must be deterministic")
}
```

---

## 6. Performance

### 6.1 Target: < 100 ms for 1,000-device tenant

| Operation | Complexity | Notes |
|---|---|---|
| Visibility check (per candidate) | O(1) | Map lookup in `VisibleTo[node.ID]` |
| AllowedIPs lookup (per visible peer) | O(1) | Map lookup in `AllowedIPs[node.ID]` |
| CIDR dedup + sort (per peer) | O(m log m) | m = number of CIDRs per peer (typically < 10) |
| Peer sort (final) | O(n log n) | n = visible peers (≤ total nodes, typically << 1,000) |
| **Total** | **O(N + V log V + V * M log M)** | N = total nodes, V = visible nodes, M = avg CIDRs/peer |

For N=1,000 with V=500 and M=5: roughly 500 map lookups + 500 small sorts +
one 500-element sort. Well under 100 ms on any modern CPU.

### 6.2 Allocation strategy

```go
func buildMap(node *topology.Node, graph *topology.Graph, policy *policy.CompiledPolicy) (*NetworkMap, error) {
    visibleSet := policy.VisibleTo[node.ID]
    allowedMap := policy.AllowedIPs[node.ID]

    // Pre-allocate: worst case every node except self is visible.
    peers := make([]Peer, 0, len(graph.Nodes)-1)

    for _, candidate := range graph.Nodes {
        if candidate.ID == node.ID {
            continue
        }
        if _, ok := visibleSet[candidate.ID]; !ok {
            continue
        }
        // ... build peer, append to peers
    }

    // Sort once before returning.
    sort.Slice(peers, func(i, j int) bool {
        return peers[i].ID < peers[j].ID
    })

    return &NetworkMap{...}, nil
}
```

### 6.3 Concurrency

`buildMap` is called **per-affected-node** after a topology event. Since each
call is independent (pure function of its inputs), the coordinator fans out
calls across a worker pool:

```go
// In the coordinator event loop:
func (c *Coordinator) pushMapDeltas(affected []uuid.UUID) {
    var wg sync.WaitGroup
    for _, deviceID := range affected {
        wg.Add(1)
        go func(id uuid.UUID) {
            defer wg.Done()
            node   := c.graph.Node(id)
            policy := c.policyCache.Get(c.graph.TenantID(id))
            nm, err := buildMap(node, c.graph, policy)
            if err != nil {
                c.log.Error("buildMap failed", "device", id, "error", err)
                return
            }
            c.streams.Push(id, nm)
        }(deviceID)
    }
    wg.Wait()
}
```

Fan-out is bounded by `GOMAXPROCS`; the worker pool size is capped at
`min(len(affected), runtime.NumCPU())`.

---

## 7. Example walkthrough

### 7.1 Topology

Three-node tenant "acme-corp":

| Device | Kind | Overlay IP | WG Pubkey | Advertises |
|---|---|---|---|---|
| `gw-1` | gateway | `fd12:3456:789a::1` | `gwkey123...` | (none -- gateway) |
| `connectorA` | connector | `fd12:3456:789a::2` | `peerkeyA...` | `10.10.0.0/24` (site 1) |
| `connectorB` | connector | `fd12:3456:789a::3` | `peerkeyB...` | `10.20.0.0/24` (site 2) |
| `alice-laptop` | client | `fd12:3456:789a::4` | `alicekey...` | (none -- client) |

### 7.2 ACL policy

```json
{
  "acls": [
    { "action": "accept", "src": ["alice@corp.com"], "dst": ["warehouse-cams:*"] }
  ],
  "hosts": {
    "warehouse-cams": "10.10.0.0/24"
  }
}
```

Alice is authorized to reach `warehouse-cams` (connectorA's LAN). She is NOT
authorized to reach connectorB.

### 7.3 Compiled policy (input to buildMap)

```
VisibleTo[alice-laptop] = { connectorA, gw-1 }
  -- connectorB is absent: Alice is not authorized to see it.

AllowedIPs[alice-laptop][connectorA] = [10.10.0.0/24]
  -- The compiled CIDR from the warehouse-cams host alias.

AllowedIPs[alice-laptop][gw-1] = []
  -- No explicit ACL grants to the gateway beyond its role as
     bootstrap/control node.

ExitNodes = []  (no exit nodes configured)
```

### 7.4 buildMap output for alice-laptop

```json
{
  "self": {
    "node_id": "0192abcd-1234-7abc-8901-234567890abc",
    "overlay_ip": "fd12:3456:789a::4/128",
    "hostname": "alice-laptop",
    "public_key": "alicekey..."
  },
  "gateway": {
    "node_id": "0192abcd-1234-7abc-8901-000000000001",
    "endpoint": "gw.example.com:443",
    "public_key": "gwkey123...",
    "masque_sni": "cdn.example.com",
    "overlay_ip": "fd12:3456:789a::1"
  },
  "peers": [
    {
      "id": "0192abcd-1234-7abc-8901-000000000002",
      "public_key": "peerkeyA...",
      "endpoints": ["10.0.0.5:51820"],
      "allowed_ips": [
        "10.10.0.0/24",
        "fd12:3456:789a:1::/96",
        "fd12:3456:789a::2/128"
      ],
      "transport": "plain-udp"
    },
    {
      "id": "0192abcd-1234-7abc-8901-000000000001",
      "public_key": "gwkey123...",
      "endpoints": ["gw.example.com:443"],
      "allowed_ips": [
        "fd12:3456:789a::1/128"
      ],
      "transport": "masque-quic-standin"
    }
  ],
  "dns": [
    "fd12:3456:789a::1"
  ],
  "transport": "auto"
}
```

### 7.5 What alice-laptop does NOT receive

- **connectorB** is absent from peers -- the ACL does not grant Alice access to
  connectorB's LAN, so `VisibleTo[alice-laptop]` excludes it. Alice never
  learns connectorB exists.
- **10.20.0.0/24** (connectorB's LAN) is absent from every peer's allowed_ips
  -- same reason.

### 7.6 What the reconciler does with this map

When the Rust edge receives this `NetworkMap` via `WatchNetworkMap`:

1. `parse_map(json)` deserializes into `map.rs`'s `NetworkMap` struct.
2. `reconcile(desired, actual)` computes:
   - `peers_to_add`: [connectorA, gw-1] (both new to alice-laptop)
   - `peers_to_remove`: [] (no stale peers)
   - `transport_changed`: true (if actual transport was different)
3. The orchestrator applies the delta: adds WireGuard peers for connectorA
   and gw-1 with their respective `AllowedIPs`.

---

## 8. Integration

### 8.1 Event-driven pipeline

```
Event Source                  Coordinator                    Edge Device
────────────                  ───────────                    ───────────
device.enrolled ──┐
device.revoked  ──┤
connector.attached ──┤
policy.updated  ──┘
                        │
                        v
                topology.Apply(event)
                        │
                        v
                affected = graph.AffectedNodes(event)
                        │
                        v
                for each node in affected:
                  nm = buildMap(node, graph, policy)
                        │
                        v
                stream.Push(node.ID, nm)
                        │
                        │  gRPC WatchNetworkMap
                        v
                ───────────────────────────────────>  map.rs reconcile()
                                                      │
                                                      v
                                                      WG peer add/remove
```

### 8.2 Event types and their affected sets

| Event | Affected nodes |
|---|---|
| `device.enrolled` | The new device itself (initial empty map -> full map) |
| `device.revoked` | ALL nodes that had the revoked device as a peer (peer removed from their maps) |
| `connector.attached` | ALL nodes authorized to reach the connector's LANs |
| `connector.prefixes_changed` | ALL nodes authorized to reach the connector -- their `AllowedIPs` change |
| `policy.updated` | ALL nodes in the tenant (every map may change) |
| `gateway.changed` | ALL nodes (gateway field changes) |

### 8.3 Convergence SLO

From event publish to the last affected edge receiving its updated map:
**p99 < 1 second** (per [ipam §5.4]).

| Step | Budget |
|---|---|
| Event bus delivery (Redis XADD -> coordinator consume) | < 50 ms |
| `topology.Apply` (in-memory graph mutation) | < 5 ms |
| `buildMap` per affected node (parallel, worker pool) | < 100 ms |
| `stream.Push` (gRPC send on open stream) | < 50 ms |
| Network round-trip + edge deserialize | < 200 ms |
| Edge `reconcile` + WG peer add/remove | < 50 ms |
| **Total target** | **< 1 s p99** |

### 8.4 Honest boundary

The convergence SLO is a **design target**, not a measured result (§11.4.6).
It becomes a measured SLO once the SS9 soak captures it on real hardware with
real network conditions. The figures above are derived from the component-level
targets in [ipam §5.4] and [identity §7]; they compose to the same
sub-second envelope.

---

## 9. Error handling

| Condition | Behaviour |
|---|---|
| `node` is nil | Return `ErrNilNode` (programmer error -- should never happen in event loop) |
| `graph` is nil | Return `ErrNilGraph` |
| `policy` is nil | Return `ErrNilPolicy` (policy is mandatory -- deny-all is an explicit empty policy, not nil) |
| Gateway not found for tenant | Return `NetworkMap` with empty `Gateway` fields; caller logs warning |
| Peer in `ExitNodes` but peer is a connector | Return error (defense-in-depth, policy compiler should have caught this) |
| Peer in `ExitNodes` but peer is revoked | Silently skip exit-node routes for that peer (defense-in-depth) |
| Malformed CIDR in compiled policy | Skip the CIDR (log warning); this is a policy compiler bug, not a `buildMap` bug |
| Empty `VisibleTo` set | Return map with zero peers (legitimate: node authorized to see no one) |

---

## 10. File layout

```
helix-go/internal/coordinator/
  buildmap.go        -- buildMap function, resolveAllowedIPs, deduplicateAndSort,
                        selectTransport, resolveGateway
  buildmap_test.go   -- unit tests: determinism, empty-policy, exit-node,
                        4via6 prefixes, sort order, error paths
  types.go           -- NetworkMap, SelfConfig, Gateway, Peer, Transport types
```

The `NetworkMap` types live alongside `buildMap` (they are the output of this
function). The protobuf definitions for `WatchNetworkMap` live in
`helix-proto/proto/helix/coordinator/v1/coordinator.proto` and are generated
into Go under `helix-proto/gen/go/coordinator/v1/`.

---

## 11. Frozen contracts (must not break)

| Contract | Defined in | What it means |
|---|---|---|
| `NetworkMap` JSON schema | This document §2.2 + `map.rs` | The wire format between coordinator and edge; changing field names or types breaks every edge |
| Peer ordering | This document §5 | `sort.Slice` by UUID bytes; changing sort key non-deterministically changes JSON output, causing reconciler churn |
| AllowedIPs as CIDR strings | `map.rs` `Peer.allowed_ips: Vec<String>` | Always `"10.0.0.0/24"` format, never binary/hex; changing format breaks `reconcile` (whole-struct equality) and WireGuard peer config |
| Need-to-know by construction | [policies_spec §7.4] + this document §3.2 | A peer absent from `VisibleTo` is absent from the map; no other mechanism may add it |
| Exit node = `0.0.0.0/0` + `::/0` | This document §4 | These exact strings; changing them breaks exit-node routing |

---

## Sources verified

- [map.rs] HelixVPN Rust `NetworkMap` schema -- `SelfConfig`, `Gateway`, `Peer`, `allowed_ips`, `reconcile()`. File: `submodules/helix_core/crates/helix-core/src/map.rs`. Read 2026-07-08.
- [policies_spec §3.6] Compiled output types -- `CompiledPolicy.VisibleTo`, `AllowedIPs`, `Verdicts`, `ExitNodes`, `NetworkMapPeer`. File: `docs/design/policies_spec/DESIGN.md`. Read 2026-07-08.
- [policies_spec §7.4] WireGuard peer mapping -- ACL denials become missing `allowed_ips`; need-to-know enforced by construction. File: `docs/design/policies_spec/DESIGN.md`. Read 2026-07-08.
- [policies_spec §7.5] Exit node routing -- `0.0.0.0/0` and `::/0` added to exit node's `allowed_ips`. File: `docs/design/policies_spec/DESIGN.md`. Read 2026-07-08.
- [ipam §3] 4via6 encoding -- ULA /48, site IDs, `Via6Route` protobuf, 4via6 prefix derivation. File: `docs/design/ipam/DESIGN.md`. Read 2026-07-08.
- [ipam §5.2] Peer protobuf -- `device_id`, `wg_pubkey`, `allowed_ips`, `is_connector`, `via6` routes. File: `docs/design/ipam/DESIGN.md`. Read 2026-07-08.
- [ipam §5.4] Convergence SLO -- p99 < 1 second from mutation to agent receiving delta. File: `docs/design/ipam/DESIGN.md`. Read 2026-07-08.
- [identity §4] Enroll RPC -- `EnrollResponse` delivers `device_id`, `overlay_ip`, `wg_pubkey`, `gateway`. File: `docs/design/identity/DESIGN.md`. Read 2026-07-08.
- [identity §1.1] Key invariants -- S2 (private keys never leave device), device identity model. File: `docs/design/identity/DESIGN.md`. Read 2026-07-08.
- Tailscale -- How NAT traversal works (4via6 addressing, WG `AllowedIPs`). URL: https://tailscale.com/blog/how-nat-traversal-works. Verified 2026-07-08.
- WireGuard whitepaper -- `AllowedIPs` cryptokey routing semantics. URL: https://www.wireguard.com/papers/wireguard.pdf. Verified 2026-07-08.

Date verified: 2026-07-08
