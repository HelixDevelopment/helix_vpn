# HelixVPN -- WebSocket/SSE Live Event Streaming Service Design (HVPN-P1-082)

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Status:** active
**Description:** Concrete design for the `GET /v1/stream` WebSocket/Server-Sent-Events
fan-out service that pushes control-plane events to connected admin consoles (and,
in Phase-2, to edge devices). Built on the existing `internal/events` Redis Streams
backbone -- this service is a *consumer* of the event bus, not a replacement for it.
**Authority:** design-authority
**Scope:** `helix-go/internal/streaming/` -- Go package implementing the fan-out hub.

---

## 1. Problem Statement

The HelixVPN coordinator's internal architecture is event-driven: identity mutations,
policy compilations, topology changes, and PKI operations all flow through the Redis
Streams event bus (`internal/events`, HVPN-P1-050). This bus is an **internal east-west**
fabric consumed by the topology graph, the map builder, and the DLQ sweeper.

External consumers -- admin consoles, dashboards, the operator's browser -- need the
same real-time visibility but cannot connect to Redis Streams directly. They need a
standard, browser-friendly streaming protocol over the REST/gRPC API surface: WebSocket
or Server-Sent Events, with topic-level subscription, tenant scoping, bearer-token
authentication, and backpressure management.

This service sits at the **API boundary**, subscribes to the internal event bus as a
consumer, and fans out typed JSON events to every connected client whose topic filter
and tenant scope match.

### 1.1 Position in the system

```
                     Redis Streams (internal east-west)
                     ┌──────────┬──────────┬──────────┬──────────┬──────────┐
                     │  devices │  routes  │  policy  │presence  │ gateway  │
                     └────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬─────┘
                          │          │          │          │          │
                          └──────────┴──────────┴──────────┴──────────┘
                                          │
                                          │  Bus.Subscribe("fanout", "hub", ...)
                                          ▼
                              ┌───────────────────────┐
                              │     StreamHub        │  (this design)
                              │  multiplex per-topic  │
                              │  per-tenant filtering │
                              │  ring-buffer per cli  │
                              └──────┬────────────────┘
                                     │
                           WebSocket / SSE
                           GET /v1/stream?topics=device,policy,topology
                           Authorization: Bearer <OIDC>
                                     │
                          ┌──────────┼──────────┐
                          ▼          ▼          ▼
                     [Browser]  [Admin CLI]  [Edge device]
                    admin console   script      (Phase-2)
```

### 1.2 Governing constraints

| # | Invariant | Source |
|---|---|---|
| S1 | **Consume only.** The hub subscribes to `events.Bus` as a consumer group `"fanout"` -- it never publishes. | [svc-events §1.2 C2] |
| S2 | **Tenant scoping.** A client authenticated as tenant T sees only events with `tenant_id == T`. Never a cross-tenant leak. | [02-control-plane §2.1 C4] |
| S3 | **Best-effort REALTIME, not guaranteed DELIVERY.** The hub is a fast-path; a disconnected client misses events. Persistent catch-up is via the REST API (list/poll), not replay from the stream. | This design §6.3 |
| S4 | **Browser-friendly.** SSE is the default transport for admin consoles; WebSocket is available for richer clients. Both speak the same JSON event envelope. | This design §2 |
| S5 | **Backpressure.** A slow client does not block the bus consumer loop. Ring-buffer per client, oldest dropped if full. | This design §6 |
| S6 | **Auth.** OIDC bearer token (Authorization header) for admin consoles; mTLS device certificate for agent streams (Phase-2). | [identity §2.1 + §4] |
| S7 | **Connection lifecycle.** 30 s heartbeat ping, reconnect with `Last-Event-ID` header (SSE) or `last_event_id` query param (WS), clean disconnect on token expiry. | This design §5 |

---

## 2. Transport Selection: SSE (default) + WebSocket (rich clients)

### 2.1 Why SSE first

| Concern | SSE | WebSocket |
|---|---|---|
| Browser API | Native `EventSource` -- zero JS library | Needs custom WS library |
| HTTP/2 multiplexing | Native (one connection, many streams) | Requires no-op extension |
| Auth | Standard `Authorization` header per-connect | `Sec-WebSocket-Protocol` or first-message token |
| Proxy/CDN | Transparent (standard HTTP) | Upgrade-aware proxy required |
| Reconnect | Built-in `Last-Event-ID` + auto-retry | Must implement manually |
| Binary frames | No (text only) | Yes |
| Complexity | ~50 lines of Go (`net/http` handler) | ~200 lines (gorilla or nhooyr.io/websocket) |

**Decision:** SSE is the default and recommended transport for admin consoles. WebSocket
is offered at the same endpoint (content-negotiation via `Accept` / `Upgrade` headers)
for CLI tools, rich dashboards, and Phase-2 edge-device WatchNetworkMap push.

### 2.2 Endpoint negotiation

```
GET /v1/stream?topics=device,policy,topology,cert,network_map
Authorization: Bearer eyJ...
Accept: text/event-stream

  → 200 OK
  Content-Type: text/event-stream
  Cache-Control: no-cache
  Connection: keep-alive
```

The same URL path serves WebSocket when the client sends the `Upgrade: websocket` header.
Query parameters are identical.

### 2.3 Wire format: JSON over SSE `data:` lines

```
event: device.enrolled
id: 1720444800000-0
data: {"id":"1720444800000-0","type":"device.enrolled","tenant_id":"0192abcd-...","ts":"2026-07-08T10:00:00Z","actor":"user:0192abcd-...","payload":{"device_id":"0192abcd-...","kind":"client","overlay_ip":"fd12:3456:789a::4"},"trace_id":"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"}

event: policy.compiled
id: 1720444800001-0
data: {"id":"1720444800001-0","type":"policy.compiled","tenant_id":"0192abcd-...","ts":"2026-07-08T10:00:01Z","actor":"system","payload":{"version":42},"trace_id":"00-..."}
```

The SSE `id:` field carries the bus message ID (Redis stream ID `<ms>-<seq>`). The
client uses this for the `Last-Event-ID` reconnect header. The `event:` field is the
`Envelope.Type` string. The `data:` field is the full JSON envelope (identical to the
bus wire shape from `internal/events/envelope.go`).

WebSocket: the same JSON object is sent as a text frame, one per event. No SSE framing.

---

## 3. Event Types (JSON over WS/SSE)

### 3.1 Existing bus events (from `internal/events/types.go`)

These flow through the fan-out unchanged. Topic mapping:

| EventType | Redis Stream | Fan-out topic | Description |
|---|---|---|---|
| `device.enrolled` | `events:devices` | `"device"` | A new device enrolled, graph node inserted |
| `device.online` | `events:presence` | `"device"` | Device heartbeat received, presence->online |
| `device.offline` | `events:presence` | `"device"` | Heartbeat missed, presence->offline |
| `device.revoked` | `events:devices` | `"device"` | Terminal revocation, graph node removed |
| `connector.attached` | `events:devices` | `"topology"` | Connector registered, site_id allocated |
| `connector.prefixes.changed` | `events:routes` | `"topology"` | Connector's advertised CIDRs changed |
| `route.conflict.detected` | `events:routes` | `"topology"` | Two connectors claim the same CIDR |
| `policy.updated` | `events:policy` | `"policy"` | Spec changed, recompile triggered |
| `policy.compiled` | `events:policy` | `"policy"` | Compile done; coordinator consumes this |
| `gateway.failover` | `events:gateway` | `"topology"` | Gateway endpoint changed, redirection |

### 3.2 New extended events (proposed -- not yet in `internal/events/types.go`)

These are **read-only derived events** the stream hub synthesizes from the bus events
it observes. They are NOT published onto the Redis Streams bus (S1: the hub is a
consumer, never a producer on the internal bus). They exist solely for the fan-out
surface.

#### `cert.rotated` -- PKI certificate rotation

```json
{
  "type": "cert.rotated",
  "payload": {
    "device_id": "0192abcd-...",
    "old_serial": "a1b2c3d4e5f6...",
    "new_serial": "f6e5d4c3b2a1...",
    "overlap_until": "2026-07-09T10:00:00Z"
  }
}
```

**Topic:** `"cert"`. **Derivation:** when the hub sees a `device.enrolled` with a
different public-key hash than the previous enrollment for the same device_id, it
infers a rotation. In Phase-2, a first-class `cert.rotated` event will be published
by the PKI service directly onto a new `events:certs` stream.

#### `cert.revoked` -- PKI certificate revocation

```json
{
  "type": "cert.revoked",
  "payload": {
    "serial": "a1b2c3d4e5f6...",
    "device_id": "0192abcd-...",
    "reason": "key_compromise",
    "is_ca": false,
    "parent_ca_serial": "ca1234..."
  }
}
```

**Topic:** `"cert"`. **Derivation:** published by the PKI revocation path
(`pki.RevokeDeviceCert`) -- currently emits no bus event; this design proposes
adding `cert.revoked` to the event taxonomy and wiring it into `RevokeDeviceCert`.

#### `network_map.updated` -- per-device NetworkMap push

```json
{
  "type": "network_map.updated",
  "payload": {
    "device_id": "0192abcd-...",
    "generation": 142,
    "peer_count": 5,
    "gateway_endpoint": "gw.example.com:443"
  }
}
```

**Topic:** `"network_map"`. **Derivation:** when the coordinator's `buildMap()` completes
and pushes a new map via `WatchNetworkMap` gRPC stream, it also emits a lightweight
`network_map.updated` event onto `events:devices`. This event does NOT carry the full
map (the map is retrieved via REST `GET /v1/devices/{id}/network-map`). It is a
**notification** that a new map is available, carrying `generation` so the console can
detect staleness.

### 3.3 Topic-to-stream mapping

The hub subscribes to a topic filter by mapping it to the Redis Streams that carry its
events:

| Client-requested topic | Redis Streams subscribed |
|---|---|
| `"device"` | `events:devices`, `events:presence` |
| `"policy"` | `events:policy` |
| `"topology"` | `events:devices`, `events:routes`, `events:gateway` |
| `"cert"` | `events:devices` (derived from `device.revoked` + PKI events, Phase-2 `events:certs`) |
| `"network_map"` | `events:devices` (derived from map-push events) |

The `"topics"` query parameter is a comma-separated list. `*` subscribes to all topics.

---

## 4. Go Types -- Concrete, Not Abstract

### 4.1 StreamHub

```go
// internal/streaming/hub.go

package streaming

import (
    "context"
    "log/slog"
    "net/http"
    "sync"
    "sync/atomic"

    "github.com/google/uuid"
    "github.com/helixdevelopment/helix-go/internal/events"
)

// StreamHub fans out events from the events.Bus to connected clients.
//
// It subscribes to Redis Streams as consumer group "fanout" and multiplexes
// each received Delivery to every connected Client whose topic filter and
// tenant scope match.
//
// Concurrency: safe for concurrent use. ServeHTTP (client connect) and
// broadcast (bus delivery) may interleave. The hub uses per-client channels
// and a mutex-protected registry so neither path blocks the other.
type StreamHub struct {
    bus events.Bus // the internal event bus (Redis Streams, never nil)

    mu      sync.RWMutex
    clients map[string]*Client // keyed by client ID (UUID v7)

    // topicSubscribers indexes which clients want which topic, so broadcast
    // can skip the full client scan for every event.
    topicSubs map[string]map[string]struct{} // topic -> set of client IDs

    // OIDC verifier for bearer tokens. If nil, auth is disabled (dev/test).
    tokenVerifier TokenVerifier

    // connIDSeq is an atomic counter for generating client IDs.
    connIDSeq atomic.Uint64

    logger *slog.Logger
}

// NewStreamHub creates a StreamHub and subscribes to the event bus for all
// five canonical streams under consumer group "fanout". The returned hub is
// ready to serve clients; the caller MUST call Run(ctx) to start the
// broadcast loop.
func NewStreamHub(bus events.Bus, verifier TokenVerifier, logger *slog.Logger) *StreamHub {
    h := &StreamHub{
        bus:           bus,
        clients:       make(map[string]*Client),
        topicSubs:     make(map[string]map[string]struct{}),
        tokenVerifier: verifier,
        logger:        logger,
    }
    return h
}

// Run starts the broadcast loop. It subscribes to the bus as consumer
// ("fanout", "hub-<hostname>") and fans out every received Delivery to
// matching clients. Run blocks until ctx is cancelled; the caller should
// invoke it in a goroutine.
func (h *StreamHub) Run(ctx context.Context) error {
    streams := []events.Stream{
        events.StreamDevices,
        events.StreamRoutes,
        events.StreamPolicy,
        events.StreamPresence,
        events.StreamGateway,
    }
    ch, err := h.bus.Subscribe(ctx, "fanout", "", streams...)
    if err != nil {
        return err
    }

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case delivery, ok := <-ch:
            if !ok {
                return nil // bus closed
            }
            h.broadcast(delivery)
            // Ack immediately after broadcast -- this is a fan-out, not a
            // work queue. If a client is slow, its ring buffer drops the
            // oldest event (backpressure, §6); the bus delivery is never
            // delayed by a slow client.
            _ = h.bus.Ack(ctx, delivery.Stream, delivery.Group, delivery.MsgID)
        }
    }
}

// broadcast fans out a delivery to every client whose topic filter and
// tenant scope match.
func (h *StreamHub) broadcast(delivery events.Delivery) {
    env := delivery.Env
    topic := topicFromEventType(env.Type)

    h.mu.RLock()
    clientIDs := h.topicSubscribers(topic)
    h.mu.RUnlock()

    for _, cid := range clientIDs {
        h.mu.RLock()
        cl := h.clients[cid]
        h.mu.RUnlock()
        if cl == nil {
            continue
        }
        if cl.TenantID != uuid.Nil && cl.TenantID != env.TenantID {
            continue // tenant scope mismatch
        }
        cl.send(env)
    }
}

// topicSubscribers returns the set of client IDs subscribed to a topic.
// Must be called with h.mu at least RLock held.
func (h *StreamHub) topicSubscribers(topic string) []string {
    m := h.topicSubs[topic]
    if m == nil {
        return nil
    }
    ids := make([]string, 0, len(m))
    for id := range m {
        ids = append(ids, id)
    }
    return ids
}
```

### 4.2 Client

```go
// internal/streaming/client.go

// Client represents one connected WebSocket or SSE consumer.
//
// It owns a ring buffer (send channel) that buffers events until the
// HTTP handler writes them. If the buffer is full, the oldest event is
// dropped (backpressure, §6).
type Client struct {
    // ID is a unique connection identifier (UUID v7), assigned at connect.
    ID string

    // TenantID is the tenant this client is authorized to see. nil UUID
    // means all tenants (super-admin, Phase-2).
    TenantID uuid.UUID

    // Topics is the set of event topics this client wants (e.g. {"device","policy"}).
    Topics map[string]struct{}

    // Transport is "sse" or "websocket".
    Transport string

    // CreatedAt is when this client connected.
    CreatedAt time.Time

    // LastEventID is the last event ID received (for reconnect).
    LastEventID atomic.Value // stores string (the bus message ID)

    // ch is the ring-buffered send channel. Capacity is RingBufferSize (§6.1).
    ch chan events.Envelope

    // done is closed when the client disconnects.
    done chan struct{}

    // closeOnce ensures cleanup runs exactly once.
    closeOnce sync.Once

    // ctx is the request context; cancelled on disconnect.
    ctx    context.Context
    cancel context.CancelFunc
}

// RingBufferSize is the capacity of the per-client event ring buffer.
// When full, the oldest event is dropped before the new one is inserted.
const RingBufferSize = 256

// NewClient creates a client and starts its write loop. The caller owns the
// response writer and passes it in.
func NewClient(ctx context.Context, tenantID uuid.UUID, topics []string, transport string) *Client {
    ctx, cancel := context.WithCancel(ctx)
    c := &Client{
        ID:        uuid.Must(uuid.NewV7()).String(),
        TenantID:  tenantID,
        Topics:    setFromSlice(topics),
        Transport: transport,
        CreatedAt: time.Now(),
        ch:        make(chan events.Envelope, RingBufferSize),
        done:      make(chan struct{}),
        ctx:       ctx,
        cancel:    cancel,
    }
    c.LastEventID.Store("") // no prior event on fresh connect
    return c
}

// send delivers an event envelope to the client's ring buffer. If the buffer
// is full, the oldest event is dropped (non-blocking backpressure).
func (c *Client) send(env events.Envelope) {
    select {
    case c.ch <- env:
        // Enqueued.
    default:
        // Ring buffer full -- drop oldest, enqueue newest.
        select {
        case <-c.ch: // discard oldest
        default:
            // Channel was drained between the two selects (race with writer).
            // Just try the send again; if still full, drop the new event.
        }
        select {
        case c.ch <- env:
        default:
            // Still full after draining oldest -- drop the new event.
            // This path is hit when the writer (HTTP handler) is not keeping up.
        }
    }
}

// Close marks the client as disconnected and cleans up resources.
func (c *Client) Close() {
    c.closeOnce.Do(func() {
        c.cancel()
        close(c.done)
    })
}

// IsAlive returns true if the client is still connected.
func (c *Client) IsAlive() bool {
    select {
    case <-c.done:
        return false
    default:
        return true
    }
}

// setFromSlice converts a string slice to a set.
func setFromSlice(ss []string) map[string]struct{} {
    m := make(map[string]struct{}, len(ss))
    for _, s := range ss {
        m[s] = struct{}{}
    }
    return m
}
```

### 4.3 HTTP handler -- SSE serve loop

```go
// internal/streaming/handler.go

// ServeHTTP handles GET /v1/stream. It authenticates the request, validates
// the topics parameter, registers the client, and serves either SSE or
// WebSocket.
func (h *StreamHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // 1. Authenticate.
    tenantID, err := h.authenticate(r)
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    // 2. Parse topic filter.
    topics := parseTopics(r.URL.Query().Get("topics"))
    if len(topics) == 0 {
        topics = []string{"*"} // subscribe to all
    }
    if contains(topics, "*") {
        topics = allTopicNames()
    }

    // 3. Determine transport.
    transport := "sse"
    if websocketUpgrade(r) {
        transport = "websocket"
    }

    // 4. Register client.
    client := NewClient(r.Context(), tenantID, topics, transport)
    h.registerClient(client)
    defer h.removeClient(client)

    // 5. Serve.
    if transport == "websocket" {
        h.serveWebSocket(w, r, client)
    } else {
        h.serveSSE(w, r, client)
    }
}

// serveSSE writes events from the client's channel as SSE data frames.
func (h *StreamHub) serveSSE(w http.ResponseWriter, r *http.Request, cl *Client) {
    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
    w.WriteHeader(http.StatusOK)
    flusher.Flush()

    // Send initial heartbeat to confirm connection.
    if err := writeSSEEvent(w, "heartbeat", "", `{"connected":true}`); err != nil {
        return
    }
    flusher.Flush()

    // If reconnecting, skip events up to LastEventID.
    lastEventID := r.Header.Get("Last-Event-ID")

    heartbeat := time.NewTicker(30 * time.Second)
    defer heartbeat.Stop()

    for {
        select {
        case <-cl.ctx.Done():
            return

        case <-heartbeat.C:
            // SSE comment line -- transparent to EventSource, keeps proxies alive.
            fmt.Fprintf(w, ": heartbeat %s\n\n", time.Now().UTC().Format(time.RFC3339))
            flusher.Flush()

        case env, ok := <-cl.ch:
            if !ok {
                return // client closed
            }

            // Skip events the client already received (reconnect dedup).
            if lastEventID != "" && env.ID <= lastEventID {
                continue
            }

            data, err := json.Marshal(env)
            if err != nil {
                h.logger.Error("stream: marshal envelope", "err", err)
                continue
            }

            if err := writeSSEEvent(w, string(env.Type), env.ID, string(data)); err != nil {
                return // client disconnected mid-write
            }
            flusher.Flush()

            // Store the last event ID for reconnect tracking.
            cl.LastEventID.Store(env.ID)
        }
    }
}

// writeSSEEvent writes a single SSE event frame.
func writeSSEEvent(w io.Writer, event, id, data string) error {
    if event != "" {
        if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
            return err
        }
    }
    if id != "" {
        if _, err := fmt.Fprintf(w, "id: %s\n", id); err != nil {
            return err
        }
    }
    // Split data on newlines; each line gets its own "data:" prefix.
    for _, line := range strings.Split(data, "\n") {
        if _, err := fmt.Fprintf(w, "data: %s\n", line); err != nil {
            return err
        }
    }
    _, err := fmt.Fprintf(w, "\n")
    return err
}
```

### 4.4 Client registration + topic indexing

```go
// registerClient adds a client to the hub's registry and indexes it by topic.
func (h *StreamHub) registerClient(cl *Client) {
    h.mu.Lock()
    defer h.mu.Unlock()

    h.clients[cl.ID] = cl
    for topic := range cl.Topics {
        if h.topicSubs[topic] == nil {
            h.topicSubs[topic] = make(map[string]struct{})
        }
        h.topicSubs[topic][cl.ID] = struct{}{}
    }
    h.logger.Info("stream: client connected",
        "client_id", cl.ID,
        "tenant_id", cl.TenantID,
        "transport", cl.Transport,
        "topics", cl.Topics,
    )
}

// removeClient removes a client and cleans up its topic indexes.
func (h *StreamHub) removeClient(cl *Client) {
    cl.Close()

    h.mu.Lock()
    defer h.mu.Unlock()

    delete(h.clients, cl.ID)
    for topic := range cl.Topics {
        if m := h.topicSubs[topic]; m != nil {
            delete(m, cl.ID)
            if len(m) == 0 {
                delete(h.topicSubs, topic)
            }
        }
    }
    h.logger.Info("stream: client disconnected", "client_id", cl.ID)
}

// ClientCount returns the number of currently connected clients.
func (h *StreamHub) ClientCount() int {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return len(h.clients)
}
```

### 4.5 Topic mapping helper

```go
// topicFromEventType maps a bus EventType to a fan-out topic name.
func topicFromEventType(t events.EventType) string {
    switch t {
    case events.EvDeviceEnrolled, events.EvDeviceOnline,
         events.EvDeviceOffline, events.EvDeviceRevoked:
        return "device"
    case events.EvConnectorAttached, events.EvConnectorPrefixesChng,
         events.EvRouteConflictDetected, events.EvGatewayFailover:
        return "topology"
    case events.EvPolicyUpdated, events.EvPolicyCompiled:
        return "policy"
    default:
        return "unknown"
    }
}

// allTopicNames returns the canonical topic set for the "*" wildcard.
func allTopicNames() []string {
    return []string{"device", "policy", "topology", "cert", "network_map"}
}

// streamForTopic returns the Redis Streams a topic filter subscribes to.
func streamsForTopic(topic string) []events.Stream {
    switch topic {
    case "device":
        return []events.Stream{events.StreamDevices, events.StreamPresence}
    case "policy":
        return []events.Stream{events.StreamPolicy}
    case "topology":
        return []events.Stream{events.StreamDevices, events.StreamRoutes, events.StreamGateway}
    case "cert":
        return []events.Stream{events.StreamDevices} // Phase-1 derivation; P2 adds events:certs
    case "network_map":
        return []events.Stream{events.StreamDevices} // Phase-1 derivation
    default:
        return nil
    }
}
```

### 4.6 WebSocket handler (sketch)

```go
// serveWebSocket upgrades the HTTP connection and pumps events from the
// client channel as WebSocket text frames.
func (h *StreamHub) serveWebSocket(w http.ResponseWriter, r *http.Request, cl *Client) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        h.logger.Error("stream: websocket upgrade failed", "err", err)
        return
    }
    defer conn.Close()

    // Read pump: discard incoming frames (admin consoles don't send data).
    // The read pump also detects client disconnect (ReadMessage returns error).
    go func() {
        for {
            if _, _, err := conn.ReadMessage(); err != nil {
                cl.Close()
                return
            }
        }
    }()

    // Write pump: fan-out from channel.
    heartbeat := time.NewTicker(30 * time.Second)
    defer heartbeat.Stop()

    for {
        select {
        case <-cl.ctx.Done():
            return

        case <-heartbeat.C:
            // WebSocket ping frame.
            conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }

        case env, ok := <-cl.ch:
            if !ok {
                return
            }
            data, _ := json.Marshal(env)
            conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
                return
            }
        }
    }
}
```

---

## 5. Connection Lifecycle

### 5.1 Connect

```
Client                          Hub
  │                              │
  │  GET /v1/stream?topics=...   │
  │  Authorization: Bearer <JWT> │
  │ ──────────────────────────>  │
  │                              │  1. Validate JWT, extract tenant_id
  │                              │  2. Parse topic filter
  │                              │  3. Register Client
  │                              │  4. Accept SSE: 200 + text/event-stream
  │  ←────────────────────────── │     or Upgrade: 101 for WebSocket
  │                              │  5. Send initial heartbeat
  │  event: heartbeat            │
  │  data: {"connected":true}    │
  │  ←────────────────────────── │
```

### 5.2 Heartbeat

- **SSE:** every 30 s, send `: heartbeat <ISO 8601>\n\n` (SSE comment -- transparent to `EventSource`, keeps proxies/load-balancers alive).
- **WebSocket:** every 30 s, send a WebSocket Ping frame (RFC 6455 §5.5.2). The Go library auto-replies with a Pong.
- **Missed heartbeat:** if the write fails (client disconnected, network loss), the hub removes the client. No explicit server-side timeout -- the TCP stack detects a broken connection on the next write attempt.
- **Client heartbeat detection:** the browser's `EventSource` auto-reconnects on connection loss. WebSocket clients should implement a 10 s Pong timeout and reconnect.

### 5.3 Reconnect with `Last-Event-ID`

The client remembers the last `id:` field it received. On reconnect:

```
GET /v1/stream?topics=device,policy
Last-Event-ID: 1720444800001-0
```

The hub skips events whose `Envelope.ID <= Last-Event-ID`. This is a **best-effort
dedup**, not a guaranteed catch-up: if the event was evicted from the Redis Stream
(MAXLEN trim), the client will miss it. The client MUST use the REST API
(`GET /v1/events?since=<Last-Event-ID>`) for guaranteed catch-up of missed events.

### 5.4 Disconnect cleanup

On disconnect (context cancelled, write error, or explicit close):

1. `client.Close()` -- cancels context, closes `done` channel.
2. `hub.removeClient(client)` -- removes from registry, cleans topic indexes.
3. The write goroutine exits when `cl.ctx.Done()` fires.
4. The read goroutine (WebSocket only) exits on read error.

### 5.5 Token expiry mid-connection

The JWT access token is validated at connect time only. If the token expires during a
long-lived connection, existing events continue to flow. The client reconnects with a
fresh token. The hub does NOT proactively terminate connections on token expiry -- the
boundary is at connect-time auth (matching the pattern of Kubernetes watch API and
Tailscale's coordination server).

---

## 6. Backpressure

### 6.1 Ring buffer per client

Each `Client.ch` is a buffered channel of capacity `RingBufferSize = 256`. The
`Client.send()` method implements a **drop-oldest** policy:

1. Try `ch <- env` (non-blocking send).
2. If full, drain the oldest event from the channel.
3. Retry send.
4. If still full (writer not keeping up), drop the new event.

This ensures a slow client never blocks the broadcast loop. The broadcast loop
always completes in O(1) per client -- it spends zero time waiting for a slow
consumer.

### 6.2 Metrics

The hub exposes Prometheus counters:

| Metric | Description |
|---|---|
| `helix_stream_clients_connected` | Gauge, current connected clients |
| `helix_stream_events_broadcast_total` | Counter, total events broadcast |
| `helix_stream_events_dropped_total` | Counter, events dropped due to full ring buffer |
| `helix_stream_connections_total` | Counter, total connections (lifetime) |
| `helix_stream_disconnects_total` | Counter, total disconnects |

### 6.3 Honest boundary

The ring buffer + drop-oldest policy means an overloaded client WILL lose events.
This is a design choice, not a bug: the hub is a **fast-path realtime notification**
service. Guaranteed event delivery with persistent catch-up is provided by the REST
API (`GET /v1/events?since=<cursor>`), which reads from Postgres (the source of
truth, per C2). A console that misses an SSE event retrieves it via the REST poll
endpoint.

---

## 7. Authentication

### 7.1 Admin console (Bearer token)

```
Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6...
```

The hub extracts the JWT, validates the signature against the OIDC provider's JWKS,
verifies the `exp` and `iat` claims, and extracts `tenant_id` from the `sub` or a
custom claim. If the token is valid, the client's scope is locked to that tenant.

**TokenVerifier interface:**

```go
// TokenVerifier validates a bearer token and returns the authorized tenant.
type TokenVerifier interface {
    Verify(ctx context.Context, token string) (tenantID uuid.UUID, err error)
}
```

**OIDC implementation (production):**

```go
// OIDCVerifier implements TokenVerifier against an OIDC provider's
// /.well-known/openid-configuration and JWKS endpoint.
type OIDCVerifier struct {
    provider *oidc.Provider
    verifier *oidc.IDTokenVerifier
    cfg      *oidc.Config // expects "nonce", checks "exp", skips "aud"
}
```

### 7.2 Device agent (mTLS, Phase-2)

In Phase-2, edge devices receive their `NetworkMap` via the same fan-out hub
(WebSocket transport, `topics=network_map`). Device authentication uses the
device leaf certificate issued during enrollment:

```
TLS Client Certificate:
  Subject: CN=device-<uuid>
  SAN: spiffe://<tenant>/device/<uuid>
  Issuer: CN=HelixVPN Issuing CA
```

The hub extracts `tenant_id` from the SPIFFE URI in the client certificate's SAN
and `device_id` from the CN. The client is scoped to events with `tenant_id == X`
AND where the event's `device_id` matches (a device sees only its own map).

---

## 8. Derived Events: Phase-1 Implementation vs Phase-2 Upgrade

### 8.1 Phase-1 (MVP): hub-side derivation

The three new event types (`cert.rotated`, `cert.revoked`, `network_map.updated`) do
NOT exist in `internal/events/types.go` today. In Phase-1, the StreamHub derives them
by observing the bus events it already receives:

- `cert.revoked`: when the hub sees `device.revoked`, it queries Postgres for the
  device certificate's serial and revocation reason, then synthesizes the event.
- `cert.rotated`: when the hub sees `device.enrolled` for an already-known
  device_id with a different public key hash, it infers rotation.
- `network_map.updated`: when the coordinator's internal `pushMapDelta` writes a
  metadata event to a separate Redis key (`helix:vpn:<tenant>:map_gen:<device_id>`),
  the hub polls it.

### 8.2 Phase-2: first-class bus events

Add these to the `internal/events` closed vocabulary:

```go
// In internal/events/types.go (Phase-2 addition):
const (
    EvCertRotated       EventType = "cert.rotated"
    EvCertRevoked       EventType = "cert.revoked"
    EvNetworkMapUpdated EventType = "network_map.updated"
)
```

Wire the constructors into `constructors.go`, register them in `validTypes`,
create a new stream `StreamCerts = "events:certs"`, and let the PKI service
and coordinator publish them directly. The StreamHub consumes them transparently
-- no hub-side derivation needed.

### 8.3 Migration path

Phase-1 deployed → hub derives events → Phase-2 bus events land → hub detects
the new event types in the stream → derivation path becomes a no-op (the
first-class event wins on type match). Zero-downtime: the hub can process both
derived and first-class events for the same type during the rollout window.

---

## 9. File Layout

```
helix-go/internal/streaming/
  hub.go          -- StreamHub, NewStreamHub, Run, broadcast, registerClient,
                     removeClient, ClientCount
  client.go       -- Client, NewClient, send, Close, IsAlive, RingBufferSize
  handler.go      -- ServeHTTP, serveSSE, serveWebSocket, writeSSEEvent,
                     websocketUpgrade, parseTopics
  topics.go       -- topicFromEventType, allTopicNames, streamsForTopic
  auth.go         -- TokenVerifier interface, OIDCVerifier, mTLSVerifier (P2)
  metrics.go      -- Prometheus metrics registration (if not in hub.go)
  hub_test.go     -- unit tests: register/remove, broadcast to matching topics,
                     tenant scoping, ring-buffer backpressure
  handler_test.go -- integration tests: SSE connect/disconnect, event delivery,
                     reconnect with Last-Event-ID, heartbeat
  testdata/       -- fixture JSON envelopes for test broadcasts
```

---

## 10. Frozen Contracts (Must Not Break)

| Contract | Defined in | What it means |
|---|---|---|
| SSE `event:` field = `Envelope.Type` | This document §2.3 | Changing the event type string breaks client-side `EventSource.addEventListener(type, ...)` |
| SSE `data:` field = full JSON envelope | This document §2.3 | Clients parse `data:` as the `Envelope` struct; adding/removing fields breaks deserialization |
| Bus consumer group = `"fanout"` | This document §4.1 | A second StreamHub instance with the same consumer group shares delivery; changing the group name duplicates every event to every client |
| Topic filter vocabulary | This document §3.3 | Closed set: `device`, `policy`, `topology`, `cert`, `network_map`, `*` |
| Ring buffer = drop-oldest, non-blocking | This document §6.1 | A slow client never blocks the broadcast loop; changing to a blocking send stalls the bus consumer |
| `Last-Event-ID` dedup = bus message ID comparison | This document §5.3 | Changing the ID format (e.g., auto-increment instead of Redis stream ID) breaks reconnect dedup |
| Hub is a consumer, never a producer on the internal bus | This document §1.2 S1 | The hub publishes nothing to `events.Bus`; violating this creates feedback loops |

---

## 11. Sources Verified

| Source | URL / Reference | Date verified |
|---|---|---|
| HelixVPN events backbone spec (svc-events.md) | `docs/research/mvp/final/v03-control-plane/svc-events.md` (Rev 1) | 2026-07-08 |
| HelixVPN events Go implementation | `submodules/helix_go/internal/events/` (types.go, envelope.go, redisbus.go, authz.go, constructors.go, payloads.go, iface.go, streams.go) | 2026-07-08 |
| HelixVPN events constructors (per-type Envelope helpers) | `submodules/helix_go/internal/events/constructors.go` | 2026-07-08 |
| HelixVPN topology graph design (event-driven mutations) | `docs/design/topology/DESIGN.md` §4 (Rev 1) | 2026-07-08 |
| HelixVPN buildMap design (NetworkMap push) | `docs/design/buildmap/DESIGN.md` §8 (Rev 1) | 2026-07-08 |
| HelixVPN identity model + Enroll RPC design | `docs/design/identity/DESIGN.md` §2 + §4 (Rev 1) | 2026-07-08 |
| HelixVPN policy compiler design | `docs/design/policies_spec/DESIGN.md` (Rev 1) | 2026-07-08 |
| HelixVPN PKI revocation Go implementation | `submodules/helix_go/pkg/pki/revocation.go` | 2026-07-08 |
| HelixVPN PKI rotation Go implementation | `submodules/helix_go/pkg/pki/rotation.go` | 2026-07-08 |
| HelixVPN PKI cert types Go implementation | `submodules/helix_go/pkg/pki/cert.go` | 2026-07-08 |
| Redis Streams XREADGROUP + XAUTOCLAIM semantics | `github.com/redis/go-redis/v9`, internal/events/redisbus.go | 2026-07-08 |
| Server-Sent Events (SSE) W3C spec | `https://html.spec.whatwg.org/multipage/server-sent-events.html` | 2026-07-08 |
| WebSocket RFC 6455 | `https://datatracker.ietf.org/doc/html/rfc6455` | 2026-07-08 |
| gorilla/websocket Go library | `github.com/gorilla/websocket` | 2026-07-08 |
| Tailscale coordination server (watch API pattern) | `https://tailscale.com/blog/how-tailscale-works` | 2026-07-08 |
| Kubernetes watch API (long-poll/SSE reconnect pattern) | `https://kubernetes.io/docs/reference/using-api/api-concepts/#efficient-detection-of-changes` | 2026-07-08 |
| OIDC token verification in Go (`coreos/go-oidc`) | `github.com/coreos/go-oidc/v3/oidc` | 2026-07-08 |

---

*End of HelixVPN WebSocket/SSE Live Event Streaming Service Design. For implementation, start with:
1. `internal/streaming/hub.go` -- StreamHub, NewStreamHub, Run (subscribe to bus)
2. `internal/streaming/client.go` -- Client struct, ring buffer, send/drop logic
3. `internal/streaming/handler.go` -- ServeHTTP, serveSSE (MVP transport)
4. `internal/streaming/topics.go` -- topic mapping helpers
5. `internal/streaming/auth.go` -- OIDC verifier + TokenVerifier interface*
