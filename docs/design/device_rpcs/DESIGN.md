# HelixVPN -- AdvertisePrefixes + ReportStatus Device RPCs Design (HVPN-P1-083)

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Status:** active -- concrete design for the two device-facing RPCs that feed the topology/buildMap pipeline
**Authority:** This document is the binding design for the `AdvertisePrefixes` and `ReportStatus`
RPCs on the `Coordinator` gRPC service. It synthesizes the existing coordinator proto stubs
(`helix.coordinator.v1`), the IPAM 4via6 design, the topology graph design, the buildMap
pipeline design, and the identity enrollment flow.
**Scope:** `submodules/helix_proto/proto/helix/coordinator/v1/coordinator.proto` (proto definitions)
+ `submodules/helix_go/internal/coordinator/` (Go implementation)

---

## 1. Problem Statement

Devices (clients and connectors) need two operational RPCs beyond the initial `Enroll` handshake:

1. **AdvertisePrefixes** -- Connectors MUST tell the coordinator which LAN subnets
   they serve, so the coordinator can compute 4via6 route prefixes, incorporate
   them into the per-peer `allowed_ips` in the `NetworkMap`, and push the updated
   map to every authorized peer. Without this RPC, a connector's LAN hosts are
   unreachable from the overlay.

2. **ReportStatus** -- Every device MUST periodically report its health to the
   coordinator so the coordinator can detect offline/dead peers, surface
   operational metrics, trigger health alerts, and maintain an accurate topology
   view. Without this RPC, the coordinator cannot distinguish a silently-failed
   device from an idle one.

Both RPCs feed into the **topology graph** (`topology.Apply`) and the
**buildMap pipeline** (`buildMap`), which together compute and push per-node
`NetworkMap` documents to every affected edge device.

### 1.1 Position in the system

```
Device (client / connector)
  |
  |-- Enroll (once) ──────────> coordinator ──> identity registry + IPAM
  |
  |-- WatchNetworkMap (stream) ──> coordinator ──> per-device NetworkMap push
  |
  |-- AdvertisePrefixes ───────> coordinator ──> topology.Apply ──> buildMap ──> WatchNetworkMap push
  |     [connectors only]             |
  |                                   +──> advertised_prefixes table (Postgres)
  |                                   +──> ipam.Via6RoutesFor (4via6 derivation)
  |
  |-- ReportStatus ────────────> coordinator ──> topology node status update + health alerts
        [periodic, 30s default]
```

---

## 2. AdvertisePrefixes RPC

### 2.1 Purpose

Connectors declare which LAN CIDRs they serve. The coordinator:

1. Validates the prefixes (no overlap with other devices in the same site, within
   allowed ranges, syntactically valid CIDRs).
2. Stores them in the `advertised_prefixes` table.
3. Derives 4via6 overlay route prefixes via `ipam.Via6RoutesFor(tenantID, connectorID)`.
4. Updates the topology graph (`connector.prefixes_changed` event).
5. Triggers `buildMap` for every peer authorized to reach this connector.
6. Pushes updated `NetworkMap` documents via `WatchNetworkMap` streams.

The RPC is **declarative** (the COMPLETE set, not a diff) and **idempotent**
(re-sending the same set is a no-op). The connector sends its current full prefix
set on every attach and on every change.

### 2.2 Protobuf definition

```protobuf
// AdvertisePrefixes — declarative prefix advertisement (connectors only).
//
// The connector sends the COMPLETE set of LAN CIDRs it currently serves.
// Re-sending the same set is idempotent (no-op). An empty list means
// "this connector serves no LANs" (valid state for a connector that has
// not yet been configured or has had all prefixes removed).
//
// Only callable by devices whose Kind == CONNECTOR. The server rejects
// calls from CLIENT devices with FAILED_PRECONDITION.
rpc AdvertisePrefixes(AdvertisePrefixesRequest) returns (AdvertisePrefixesResponse);

message AdvertisePrefixesRequest {
  // device_id of the connector. MUST equal the device bound to the mTLS
  // cert presented on this connection. The server cross-validates and
  // rejects mismatches with PERMISSION_DENIED.
  string device_id = 1;

  // The COMPLETE set of LAN CIDRs this connector serves.
  // Examples: ["192.168.1.0/24", "10.0.0.0/8", "172.16.0.0/12"].
  // IPv6 LAN prefixes are also accepted (shipped as plain allowed_ips
  // CIDRs without 4via6 translation).
  // Empty list = connector serves no LANs (valid, not an error).
  // Maximum: 64 prefixes per connector (configurable server-side).
  repeated string prefixes = 2;
}

message AdvertisePrefixesResponse {
  // Whether all prefixes were accepted and stored.
  bool accepted = 1;

  // Human-readable validation/overlap messages.
  // When accepted == true, this is empty.
  // When accepted == false, this enumerates the specific conflicts
  // (e.g. "192.168.1.0/24 overlaps with connector-xyz's 192.168.0.0/16").
  repeated string conflicts = 2;

  // The derived 4via6 route prefixes for each advertised CIDR.
  // Only populated when accepted == true. The connector can use these
  // for local validation and debugging.
  // Map key = the advertised IPv4 CIDR, value = the derived via6_prefix.
  // Example: {"192.168.1.0/24": "fd12:3456:789a:7::/96"}
  map<string, string> via6_routes = 3;

  // Monotonic generation counter bumped by this mutation. The connector
  // can compare this against the generation in its WatchNetworkMap stream
  // to confirm its advertisement has been compiled into the map.
  int64 topology_generation = 4;
}
```

### 2.3 Validation rules

| Rule | Check | Error on failure |
|---|---|---|
| Caller is a connector | `device.kind == CONNECTOR` | `FAILED_PRECONDITION`: "device is not a connector" |
| Device ID matches mTLS cert | `req.device_id == cert.cn` | `PERMISSION_DENIED`: "device_id does not match mTLS identity" |
| Device is active (not revoked) | `device.revoked_at IS NULL` | `PERMISSION_DENIED`: "device is revoked" |
| CIDRs are syntactically valid | `netip.ParsePrefix(c)` succeeds | `INVALID_ARGUMENT`: "invalid CIDR: ..." |
| CIDRs are within allowed ranges | RFC 1918 (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`) or other private ranges | OK with warning logged; public IP ranges rejected with `INVALID_ARGUMENT` |
| No overlap with other connectors in same site | `advertised_prefixes` table query for overlapping CIDRs | `ALREADY_EXISTS` w/ `conflicts` detailing the overlapping connector |
| Max prefix count | `len(prefixes) <= 64` | `RESOURCE_EXHAUSTED`: "too many prefixes" |

### 2.4 Flow (server-side)

```
AdvertisePrefixes(req)
  |
  --> 1. Authenticate: extract device_id from mTLS cert, cross-validate with req.device_id
  --> 2. Authorize: confirm device.kind == CONNECTOR, device is active (not revoked)
  --> 3. Validate: parse all CIDRs, check RFC 1918 ranges, check max count
  --> 4. Conflict-check: query advertised_prefixes for overlapping CIDRs from OTHER
         connectors in the SAME tenant (self-overlap is allowed -- declarative resend)
  --> 5. If conflicts: return AdvertisePrefixesResponse{accepted=false, conflicts=[...]}
  --> 6. If clean:
        a. BEGIN tx
        b. DELETE FROM advertised_prefixes WHERE connector_id = $1
        c. INSERT INTO advertised_prefixes (connector_id, cidr) VALUES ...
        d. ipam.Via6RoutesFor(tenantID, connectorID) -> []Via6Route
        e. COMMIT tx
        f. topology.Apply(connector.prefixes_changed event)
        g. affected = graph.AffectedNodes(event)
        h. for each node in affected (parallel, worker-pool):
             nm = buildMap(node, graph, policy)
             stream.Push(node.ID, nm)
        i. return AdvertisePrefixesResponse{
             accepted=true,
             via6_routes=...,
             topology_generation=graph.Generation()
           }
```

### 2.5 Convergence SLO

From `AdvertisePrefixes` RPC return to the last affected edge receiving its
updated `NetworkMap`: **p99 < 1 second** (per the IPAM design SS5.4 and the
buildMap design SS8.3).

| Step | Budget |
|---|---|
| Validation + DB tx | < 50 ms |
| `topology.Apply` | < 5 ms |
| `buildMap` per affected node (parallel) | < 100 ms |
| `stream.Push` (gRPC send) | < 50 ms |
| Network + edge deserialize + reconcile | < 300 ms |
| **Total p99 target** | **< 1 s** |

---

## 3. ReportStatus RPC

### 3.1 Purpose

Every device periodically reports its operational health to the coordinator.
The coordinator uses this data to:

1. **Detect offline/dead peers** -- if a device stops reporting, it is marked
   as disconnected after a grace period (default: 3 missed heartbeats = 90 s).
2. **Surface operational metrics** -- bytes sent/received, connected peers,
   uptime, active transport.
3. **Trigger health alerts** -- sustained disconnection, anomalous traffic
   patterns, transport degradation.
4. **Maintain topology accuracy** -- the coordinator's view of which peers
   are reachable through which transports.

The RPC is a **periodic heartbeat** with a default interval of 30 seconds.

### 3.2 Privacy posture

The existing skeletal `StatusReport` in `coordinator.proto` intentionally
carried ONLY `transport` + `rtt_ms` with the comment "Carries NO bytes /
flows / destinations / packet counts -- no-logging by construction."

This design EXPANDS the report to include operational telemetry
(`bytes_sent`, `bytes_recv`, `connected_peers`, `uptime_seconds`) while
maintaining the privacy stance through two mechanisms:

1. **Configurable reporting level** -- the device's local config controls
   which fields are populated. A privacy-maximal deployment sets
   `reporting_level = MINIMAL` and the device sends only `connected` +
   `active_transport` + `uptime_seconds` (no traffic data). A full-
   telemetry deployment sets `reporting_level = FULL` and sends all fields.

2. **Aggregation, not inspection** -- `bytes_sent` / `bytes_recv` are
   aggregate interface counters (total since boot / last reset), NOT
   per-flow or per-destination. The coordinator never learns WHAT was sent,
   only HOW MUCH.

| Reporting Level | Fields sent |
|---|---|
| `MINIMAL` | `connected`, `active_transport`, `uptime_seconds` |
| `STANDARD` | MINIMAL + `connected_peers` (peer IDs, not IPs) |
| `FULL` | STANDARD + `bytes_sent`, `bytes_recv` (aggregate counters) |

The coordinator's `ReportStatus` handler accepts any level; it is the
DEVICE that decides what to report. The coordinator never demands a higher
level than the device is configured to provide.

**Honest boundary (SS11.4.6):** Even at FULL level, this is NOT a
privacy-invasive telemetry stream. It does not carry flow logs, DNS
queries, SNI values, per-packet metadata, or destination IPs. Aggregate
counters + peer-ID lists are the maximum surface. Deployments requiring
stricter privacy use MINIMAL.

### 3.3 Protobuf definition

```protobuf
// ReportStatus — periodic device health heartbeat.
//
// Sent by every enrolled device (client, connector, or gateway) on a
// periodic interval (default 30 s). The coordinator uses this to maintain
// an accurate liveness view of the overlay and to surface operational metrics.
//
// NOT a traffic-inspection stream — carries only aggregate counters and
// peer-identity lists. The device controls reporting granularity via its
// local config (MINIMAL / STANDARD / FULL).
rpc ReportStatus(ReportStatusRequest) returns (ReportStatusResponse);

message ReportStatusRequest {
  // device_id of the reporting device. MUST equal the device bound to the
  // mTLS cert on this connection.
  string device_id = 1;

  // Current device health/status snapshot.
  DeviceStatus status = 2;
}

message DeviceStatus {
  // Whether the device considers itself connected to the overlay.
  // false = the device believes it has lost connectivity (WG handshake
  // failed, transport unreachable, etc.).
  bool connected = 1;

  // Currently-active transport protocol label.
  // Examples: "plain-udp", "masque-quic-standin", "wireguard".
  // Empty string = no active transport (device is disconnected or
  // still probing).
  string active_transport = 2;

  // Aggregate bytes sent over the tunnel interface since boot or last
  // counter reset. Only populated when reporting_level >= FULL.
  // Omitting (zero value) means "not reported at this level."
  uint64 bytes_sent = 3;

  // Aggregate bytes received over the tunnel interface since boot or last
  // counter reset. Only populated when reporting_level >= FULL.
  uint64 bytes_recv = 4;

  // Device IDs of peers this device currently has an active WireGuard
  // session with (recent handshake). This tells the coordinator which
  // peers are reachable through which transports.
  // Only populated when reporting_level >= STANDARD.
  repeated string connected_peers = 5;

  // Device uptime in seconds (since boot or since the agent started).
  // Always populated regardless of reporting level.
  double uptime_seconds = 6;
}

message ReportStatusResponse {
  // Acknowledgment. The coordinator echoes back the topology generation
  // so the device can detect staleness without opening a WatchNetworkMap
  // stream (useful for lightweight clients).
  int64 topology_generation = 1;

  // If non-empty, the coordinator is instructing the device to perform
  // an action. Currently defined actions:
  //   "RESYNC" — the device should re-open its WatchNetworkMap stream
  //              (the coordinator's generation advanced past a threshold
  //              and the device may have a stale map).
  repeated string actions = 2;
}
```

### 3.4 Heartbeat lifecycle

```
Device connects                             Coordinator
     |                                          |
     |── ReportStatus{connected=true} ───────>  |
     |                                          |── update topology.Node.Status = active
     |                                          |── reset missed-heartbeat counter
     |                                          |── return ReportStatusResponse{...}
     |                                          |
     |── ... (30 s interval) ...                |
     |── ReportStatus{connected=true} ───────>  |
     |                                          |── update topology (no-op if unchanged)
     |                                          |
     |── [device loses connectivity]            |
     |   (stops sending ReportStatus)           |
     |                                          |
     |                                          |── 30 s: 1st missed heartbeat (log warning)
     |                                          |── 60 s: 2nd missed heartbeat
     |                                          |── 90 s: 3rd missed heartbeat
     |                                          |── mark Node.Status = disconnected
     |                                          |── emit device.disconnected event
     |                                          |── trigger health alert
     |                                          |
     |── ReportStatus{connected=false} ──────>  |
     |   (device regains transport,             |
     |    but WG handshake still failing)        |
     |                                          |── mark Node.Status = degraded
     |                                          |── log: "device connected at transport but WG down"
     |                                          |
     |── ReportStatus{connected=true} ───────>  |
     |                                          |── mark Node.Status = active
     |                                          |── emit device.reconnected event
```

### 3.5 Flow (server-side)

```
ReportStatus(req)
  |
  --> 1. Authenticate: extract device_id from mTLS cert, cross-validate with req.device_id
  --> 2. Authorize: confirm device is active (not revoked)
  --> 3. Update topology.Node status fields:
        node.Status = req.status.connected ? NodeActive : NodeDisconnected
        node.ActiveTransport = req.status.active_transport
        node.LastSeen = now
  --> 4. If newly disconnected (was active, now !connected):
        a. Increment missed_heartbeat counter
        b. If missed >= 3:
             emit device.disconnected event
             trigger health alert (log + metrics counter)
             affected = graph.AffectedNodes(disconnect event)
             for each affected node: buildMap + push
  --> 5. If newly reconnected (was disconnected, now connected):
        emit device.reconnected event
        affected = graph.AffectedNodes(reconnect event)
        for each affected node: buildMap + push
  --> 6. Return ReportStatusResponse{
        topology_generation = graph.Generation(),
        actions = [...]
      }
```

### 3.6 Health alert triggers

| Condition | Alert | Severity |
|---|---|---|
| 3 consecutive missed heartbeats | `device.disconnected` | WARNING |
| 10 consecutive missed heartbeats (5 min) | `device.unreachable` | CRITICAL |
| Transport flip (3+ changes in 5 min) | `transport.unstable` | WARNING |
| Connected but zero connected_peers (connector, 5+ min) | `connector.isolated` | WARNING |
| bytes_sent growing but bytes_recv == 0 (5+ min) | `device.tx_only` | INFO |

---

## 4. Go Implementation Sketch

### 4.1 File layout

```
helix-go/internal/coordinator/
  advertise.go         -- AdvertisePrefixes handler
  advertise_test.go    -- tests: validation, idempotency, overlap detection
  report_status.go     -- ReportStatus handler
  report_status_test.go -- tests: heartbeat lifecycle, disconnect detection, alert triggers
  rpc_helpers.go       -- shared auth, validation, mTLS cert extraction
```

### 4.2 AdvertisePrefixes handler

```go
// Package coordinator implements the Coordinator gRPC service handlers.
package coordinator

import (
    "context"
    "fmt"
    "net/netip"

    "github.com/google/uuid"
    coordinatorv1 "github.com/vasic-digital/helix_proto/gen/go/helix/coordinator/v1"
)

// AdvertisePrefixes handles a connector's declarative prefix advertisement.
//
// Validation: caller must be a CONNECTOR, device_id must match mTLS cert,
// device must be active, all CIDRs must parse, no overlap with other
// connectors in the same site, within max count.
//
// On success: stores prefixes in advertised_prefixes table, derives 4via6
// routes via IPAM, updates the topology graph, triggers buildMap for
// affected peers, and pushes updated NetworkMap documents.
//
// Idempotent: re-sending the same prefix set is a no-op.
func (s *CoordinatorServer) AdvertisePrefixes(
    ctx context.Context,
    req *coordinatorv1.AdvertisePrefixesRequest,
) (*coordinatorv1.AdvertisePrefixesResponse, error) {
    // 1. Authenticate + authorize
    device, err := s.authenticateConnector(ctx, req.DeviceId)
    if err != nil {
        return nil, err
    }

    // 2. Parse and validate CIDRs
    prefixes, err := parseAndValidatePrefixes(req.Prefixes, device.TenantID)
    if err != nil {
        return nil, status.Error(codes.InvalidArgument, err.Error())
    }

    // 3. Conflict check (overlap with OTHER connectors in same tenant)
    conflicts, err := s.db.CheckPrefixOverlaps(ctx, device.TenantID, device.ID, prefixes)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "overlap check: %v", err)
    }
    if len(conflicts) > 0 {
        return &coordinatorv1.AdvertisePrefixesResponse{
            Accepted:  false,
            Conflicts: conflicts,
        }, nil
    }

    // 4. Store + derive 4via6 routes
    via6Routes, err := s.storePrefixesAndDeriveRoutes(ctx, device, prefixes)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "store prefixes: %v", err)
    }

    // 5. Update topology graph
    event := s.topology.ApplyPrefixesChanged(device.ID, prefixes, via6Routes)

    // 6. Compute affected nodes + push updated maps
    affected := s.graph.AffectedNodes(event)
    s.pushMapDeltas(ctx, affected)

    // 7. Build via6_routes map for response
    via6Map := make(map[string]string, len(via6Routes))
    for _, vr := range via6Routes {
        via6Map[vr.IPv4CIDR] = vr.Via6Prefix
    }

    return &coordinatorv1.AdvertisePrefixesResponse{
        Accepted:           true,
        Via6Routes:         via6Map,
        TopologyGeneration: s.graph.Generation(),
    }, nil
}

// parseAndValidatePrefixes parses string CIDRs into netip.Prefix values
// and validates them against the allowed-range policy.
func parseAndValidatePrefixes(cidrs []string, tenantID uuid.UUID) ([]netip.Prefix, error)

// storePrefixesAndDeriveRoutes stores the prefixes in the database
// (delete-then-insert in a single tx) and derives 4via6 route prefixes.
func (s *CoordinatorServer) storePrefixesAndDeriveRoutes(
    ctx context.Context,
    device *Device,
    prefixes []netip.Prefix,
) ([]Via6Route, error)
```

### 4.3 ReportStatus handler

```go
// ReportStatus handles a periodic device health heartbeat.
//
// Updates the topology node status (active/connected vs disconnected),
// resets the missed-heartbeat counter, triggers health alerts on state
// transitions, and optionally instructs the device to resync its map.
//
// Default heartbeat interval is 30 s; disconnect detection fires after
// 3 missed heartbeats (90 s).
func (s *CoordinatorServer) ReportStatus(
    ctx context.Context,
    req *coordinatorv1.ReportStatusRequest,
) (*coordinatorv1.ReportStatusResponse, error) {
    // 1. Authenticate
    device, err := s.authenticateDevice(ctx, req.DeviceId)
    if err != nil {
        return nil, err
    }

    // 2. Update topology node status
    prevStatus := device.Status
    s.topology.UpdateNodeStatus(device.ID, NodeStatusUpdate{
        Connected:       req.Status.Connected,
        ActiveTransport: req.Status.ActiveTransport,
        ConnectedPeers:  req.Status.ConnectedPeers,
        LastSeen:        time.Now(),
    })

    // 3. Handle state transitions
    var actions []string

    if req.Status.Connected {
        // Reset missed-heartbeat counter on successful report.
        s.heartbeats.Reset(device.ID)

        if prevStatus == NodeDisconnected {
            // Device reconnected -- emit event + push maps.
            s.events.Emit(DeviceReconnected{DeviceID: device.ID})
            affected := s.graph.AffectedNodes(DeviceReconnected{DeviceID: device.ID})
            s.pushMapDeltas(ctx, affected)
        }
    } else {
        // Device reports itself as disconnected.
        s.topology.MarkDisconnected(device.ID)
        s.events.Emit(DeviceDisconnected{DeviceID: device.ID})
    }

    // 4. Check for resync instruction
    if s.graph.Generation() - device.LastKnownGeneration > ResyncThreshold {
        actions = append(actions, "RESYNC")
    }

    return &coordinatorv1.ReportStatusResponse{
        TopologyGeneration: s.graph.Generation(),
        Actions:            actions,
    }, nil
}

// HeartbeatTracker tracks per-device heartbeat state for disconnect detection.
type HeartbeatTracker struct {
    mu     sync.Mutex
    states map[uuid.UUID]*HeartbeatState
}

type HeartbeatState struct {
    LastSeen   time.Time
    Missed     int       // consecutive missed heartbeats
    Status     NodeStatus
}

// Reset resets the missed-heartbeat counter for a device (called on
// successful ReportStatus with connected=true).
func (ht *HeartbeatTracker) Reset(deviceID uuid.UUID)

// IncrementMissed increments the missed counter and returns whether
// the disconnect threshold (3) has been reached.
func (ht *HeartbeatTracker) IncrementMissed(deviceID uuid.UUID) bool

// StartBackgroundChecker runs a background goroutine that periodically
// (every 5 s) checks for devices that have missed heartbeats and updates
// their status accordingly.
func (ht *HeartbeatTracker) StartBackgroundChecker(ctx context.Context, graph *topology.TopologyGraph, events EventEmitter)
```

### 4.4 Shared authentication helpers

```go
// authenticateConnector authenticates the caller as a CONNECTOR device.
//
// Extracts the device_id from the mTLS client certificate presented on
// the gRPC connection, cross-validates it against req.device_id, loads
// the device from the topology graph (or DB as fallback), and verifies:
//   - Device exists and is active (not revoked)
//   - Device.Kind == CONNECTOR
//
// Returns the authenticated Device on success, or a gRPC status error:
//   - UNAUTHENTICATED: no valid mTLS cert
//   - PERMISSION_DENIED: device_id mismatch or device is revoked
//   - FAILED_PRECONDITION: device is not a connector
func (s *CoordinatorServer) authenticateConnector(
    ctx context.Context,
    deviceID string,
) (*Device, error)

// authenticateDevice authenticates the caller as ANY active device
// (client, connector, or gateway). Same extraction + cross-validation
// as authenticateConnector but without the kind check.
func (s *CoordinatorServer) authenticateDevice(
    ctx context.Context,
    deviceID string,
) (*Device, error)

// extractDeviceIDFrommTLS extracts the device UUID from the CN field
// of the mTLS client certificate presented on the gRPC connection.
func extractDeviceIDFrommTLS(ctx context.Context) (uuid.UUID, error)
```

### 4.5 pushMapDeltas helper

```go
// pushMapDeltas fans out buildMap computations for a set of affected
// device IDs, computes per-node NetworkMap documents, and pushes them
// via the WatchNetworkMap gRPC streams.
//
// Calls buildMap for each affected node in parallel via a bounded
// worker pool (capped at min(len(affected), runtime.NumCPU())).
// Each buildMap call reads from an immutable topology snapshot.
//
// Non-blocking for the RPC handler: the handler returns to the caller
// after enqueuing the push; the actual gRPC sends happen on background
// goroutines. The convergence SLO (p99 < 1 s) covers the full path
// from RPC return to last stream delivery.
func (s *CoordinatorServer) pushMapDeltas(
    ctx context.Context,
    affected []uuid.UUID,
)
```

---

## 5. Integration with topology.Apply + buildMap Pipeline

### 5.1 Event flow

```
AdvertisePrefixes RPC                    ReportStatus RPC
        |                                       |
        v                                       v
  Validate + store prefixes              Update node status
        |                                       |
        v                                       v
  ipam.Via6RoutesFor(tenant, conn)       Check state transition
        |                                 (active->disconnected /
        v                                  disconnected->active)
  topology.Apply(                         |
    connector.prefixes_changed)           v
        |                           topology.UpdateNodeStatus(...)
        v                                       |
  graph.AffectedNodes(event)             graph.AffectedNodes(event)
  -> all peers authorized to             -> all peers that had this
     reach this connector's LANs            device as a peer
        |                                       |
        v                                       v
  for each affected node (parallel):     for each affected node (parallel):
    nm = buildMap(node, graph, policy)     nm = buildMap(node, graph, policy)
        |                                       |
        v                                       v
  stream.Push(node.ID, nm)               stream.Push(node.ID, nm)
        |                                       |
        v                                       v
  Edge receives via WatchNetworkMap       Edge receives via WatchNetworkMap
  map.rs reconcile() applies delta        map.rs reconcile() applies delta
```

### 5.2 Affected node computation

**For `AdvertisePrefixes`:**
All nodes in the tenant whose `CompiledPolicy.VisibleTo` includes the
advertising connector. (These are the nodes authorized to reach the
connector's LAN -- their `allowed_ips` change.)

**For `ReportStatus` (disconnect):**
All nodes that had the disconnected device as a peer in their most recent
`NetworkMap`. (The peer is removed from their maps.)

**For `ReportStatus` (reconnect):**
All nodes whose `CompiledPolicy.VisibleTo` includes the reconnected
device. (The peer is added back to their maps.)

### 5.3 Event types emitted

| RPC | Payload Change | Event Emitted | topology.Apply Method |
|---|---|---|---|
| `AdvertisePrefixes` | Prefixes added/changed/removed | `connector.prefixes_changed` | `ApplyPrefixesChanged(deviceID, prefixes, via6Routes)` |
| `ReportStatus` | `connected: true -> false` | `device.disconnected` | `MarkDisconnected(deviceID)` |
| `ReportStatus` | `connected: false -> true` | `device.reconnected` | `MarkReconnected(deviceID)` |
| `ReportStatus` | No change (steady heartbeat) | (none -- only counter reset) | (none) |

### 5.4 Event payload schemas (Redis Streams)

```json
// connector.prefixes_changed
{
  "type": "connector.prefixes_changed",
  "device_id": "0192abcd-1234-7abc-8901-234567890abc",
  "tenant_id": "0192abcd-1234-7abc-8901-000000000000",
  "prefixes": [
    {"ipv4_cidr": "192.168.1.0/24", "via6_prefix": "fd12:3456:789a:7::/96"}
  ]
}

// device.disconnected
{
  "type": "device.disconnected",
  "device_id": "0192abcd-1234-7abc-8901-234567890abc",
  "tenant_id": "0192abcd-1234-7abc-8901-000000000000",
  "missed_heartbeats": 3,
  "last_seen": "2026-07-08T12:00:00Z"
}

// device.reconnected
{
  "type": "device.reconnected",
  "device_id": "0192abcd-1234-7abc-8901-234567890abc",
  "tenant_id": "0192abcd-1234-7abc-8901-000000000000",
  "active_transport": "plain-udp",
  "downtime_seconds": 45.2
}
```

---

## 6. Database Schema (advertised_prefixes table)

```sql
-- Already defined in the IPAM design; reproduced here for completeness.
CREATE TABLE advertised_prefixes (
    id           BIGSERIAL PRIMARY KEY,
    connector_id UUID    NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    tenant_id    UUID    NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    cidr         CIDR    NOT NULL,                         -- e.g. 192.168.1.0/24
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- A connector may re-advertise the same CIDR (idempotent resend).
    UNIQUE (connector_id, cidr)
);

CREATE INDEX idx_advertised_prefixes_tenant ON advertised_prefixes(tenant_id);
```

---

## 7. Relationship to Existing Proto Stubs

The existing `helix.coordinator.v1.Coordinator` service in
`submodules/helix_proto/proto/helix/coordinator/v1/coordinator.proto`
already declares both RPCs with skeletal message types:

| RPC | Existing message | This design's message |
|---|---|---|
| `AdvertisePrefixes` | `AdvertiseRequest` / `AdvertiseResponse` | `AdvertisePrefixesRequest` / `AdvertisePrefixesResponse` |
| `ReportStatus` | `StatusReport` / `StatusAck` | `ReportStatusRequest` / `ReportStatusResponse` |

The existing stubs were intentional placeholders -- minimal definitions
that established the RPC names and basic shapes. This design provides
the full, production-ready definitions.

**Key differences from the existing stubs:**

1. **AdvertisePrefixes:**
   - Existing: `AdvertiseRequest.cidrs` (field name)
   - This design: `AdvertisePrefixesRequest.prefixes` (more descriptive)
   - Existing: `AdvertiseResponse.accepted` + `conflicts`
   - This design: adds `via6_routes` (derived 4via6 prefixes) and
     `topology_generation` (monotonic generation counter)

2. **ReportStatus:**
   - Existing: `StatusReport.transport` + `rtt_ms` (minimal, "no-logging by construction")
   - This design: `DeviceStatus` with `connected`, `active_transport`,
     `bytes_sent`, `bytes_recv`, `connected_peers`, `uptime_seconds`,
     gated by configurable `reporting_level` (MINIMAL/STANDARD/FULL)
   - Existing: `StatusAck` (empty)
   - This design: `ReportStatusResponse.topology_generation` + `actions`

**Migration path:** The proto definitions in this document replace the
existing skeletal messages. The RPC names (`AdvertisePrefixes`,
`ReportStatus`) remain unchanged, so existing generated client stubs
continue to compile after regeneration. The message type changes are
backward-incompatible at the wire level, which is acceptable during
Phase-1 development (pre-GA). The `buf breaking` gate will flag the
change as `FIELD_SAME_TYPE` violations; these are acknowledged and
accepted for the Phase-1 development window.

---

## 8. Edge Cases

| # | Edge Case | Handling |
|---|---|---|
| EC-1 | Connector advertises the same CIDR as another connector in the same tenant | Accepted (no overlap error) -- different site IDs produce different 4via6 prefixes; the conflict is detected and surfaced as an informational event, not a block |
| EC-2 | Connector advertises an IPv6 LAN prefix | Accepted as-is; stored in `advertised_prefixes` but no 4via6 derivation (4via6 only applies to IPv4) |
| EC-3 | Connector sends an empty prefix list | Accepted; all existing prefixes for this connector are deleted; connector serves no LANs (valid state) |
| EC-4 | Connector is deleted and re-created | Cascade frees old `connector_sites` row; re-enroll allocates new site ID; old `advertised_prefixes` are cascade-deleted; new connector starts with empty prefixes |
| EC-5 | Device sends ReportStatus after being revoked | `PERMISSION_DENIED`; the mTLS cert is still valid (may not yet be on the CRL), but the device is revoked in the DB |
| EC-6 | ReportStatus with `connected=true` but `active_transport` empty | Accepted; the device is connected but hasn't yet selected a transport (transitional state during startup) |
| EC-7 | ReportStatus with `connected=false` but `active_transport` non-empty | Accepted; the transport layer is up but WG handshake is failing (degraded state) |
| EC-8 | Coordinator restart loses in-memory heartbeat counters | On restart, all devices are considered "unknown" until their first ReportStatus arrives; no false-disconnect alerts during the grace window |
| EC-9 | Rapid AdvertisePrefixes calls (connector flapping) | Each call is processed in order; idempotent when the set hasn't changed; topology generation bumps only when the stored set actually changes |
| EC-10 | AdvertisePrefixes called by a CLIENT device | `FAILED_PRECONDITION`: "device is not a connector" |

---

## 9. Frozen Contracts (Must Not Break)

| Contract | Where defined | What it means |
|---|---|---|
| RPC names | `coordinator.proto` `service Coordinator` | `AdvertisePrefixes` and `ReportStatus` are the canonical names; renaming breaks every generated client |
| `AdvertisePrefixesRequest.prefixes` | This document SS2.2 | Declarative (complete set, not a diff); changing to incremental-diff semantics breaks idempotency |
| `ReportStatusRequest.status.connected` | This document SS3.3 | The liveness signal; misinterpreting `false` (disconnected vs degraded) leads to false alerts |
| Heartbeat interval default: 30 s | This document SS3.1 | Changing the default changes the disconnect-detection window (3 x 30 = 90 s) |
| Disconnect threshold: 3 missed heartbeats | This document SS3.6 | Changing the threshold changes alert sensitivity |
| `DeviceStatus` reporting levels | This document SS3.2 | MINIMAL / STANDARD / FULL; adding a level that breaks the privacy floor violates the no-logging intent |
| 4via6 route derivation on AdvertisePrefixes | IPAM design SS5.1 | Advertised CIDRs produce 4via6 /96 prefixes via `ipam.Via6RoutesFor` |
| `topology.Apply` event vocabulary | This document SS5.3 | Closed set: `connector.prefixes_changed`, `device.disconnected`, `device.reconnected` |

---

## 10. Sources Verified

| Source | URL / Reference | Date verified |
|---|---|---|
| HelixVPN Coordinator proto (existing stubs) | `submodules/helix_proto/proto/helix/coordinator/v1/coordinator.proto` | 2026-07-08 |
| HelixVPN IPAM 4via6 Design | `docs/design/ipam/DESIGN.md` (Rev 1) SS5.1 Route Advertisement | 2026-07-08 |
| HelixVPN Topology Graph Design | `docs/design/topology/DESIGN.md` (Rev 1) SS4.2 Event types, SS8 Integration with buildMap | 2026-07-08 |
| HelixVPN buildMap Design | `docs/design/buildmap/DESIGN.md` (Rev 1) SS2 Function signature, SS8 Event-driven pipeline | 2026-07-08 |
| HelixVPN NetworkMap schema (map.rs) | `submodules/helix_core/crates/helix-core/src/map.rs` -- NetworkMap, Peer, reconcile() | 2026-07-08 |
| HelixVPN Identity / Enroll Design | `docs/design/identity/DESIGN.md` (Rev 1) SS4 Enroll RPC wire contract | 2026-07-08 |
| gRPC health checking pattern | https://github.com/grpc/grpc/blob/master/doc/health-checking.md | 2026-07-08 |
| Connect protocol (gRPC-compatible) | https://connectrpc.com/docs/protocol | 2026-07-08 |
| Protocol Buffers style guide | https://protobuf.dev/programming-guides/style/ | 2026-07-08 |
| Tailscale -- How NAT traversal works (4via6, WG AllowedIPs) | https://tailscale.com/blog/how-nat-traversal-works | 2026-07-08 |
| WireGuard whitepaper (Cryptokey Routing, AllowedIPs) | https://www.wireguard.com/papers/wireguard.pdf | 2026-07-08 |

Date verified: 2026-07-08

---

*End of HelixVPN AdvertisePrefixes + ReportStatus Device RPCs Design. For implementation, start with:*
1. *`submodules/helix_proto/proto/helix/coordinator/v1/coordinator.proto` -- replace skeletal messages with full definitions*
2. *`submodules/helix_go/internal/coordinator/advertise.go` -- AdvertisePrefixes handler*
3. *`submodules/helix_go/internal/coordinator/report_status.go` -- ReportStatus handler + HeartbeatTracker*
4. *`submodules/helix_go/internal/coordinator/rpc_helpers.go` -- shared mTLS auth extraction*
5. *`submodules/helix_go/internal/topology/apply.go` -- add ApplyPrefixesChanged, MarkDisconnected, MarkReconnected*
