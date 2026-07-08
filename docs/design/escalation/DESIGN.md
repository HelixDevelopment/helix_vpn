# Transport Auto-Escalation Ladder — Design

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Status:** Draft
**Authority:** HVPN-P1-093

## 1. Problem Statement

Deep Packet Inspection (DPI) middleboxes increasingly block WireGuard traffic by matching the
Noise IK handshake signature in the first few UDP datagrams. When the target peer is behind such a
DPI device, a plain-UDP WireGuard tunnel never establishes — the initial handshake-initiation
message is dropped before it reaches the gateway.

MASQUE (RFC 9298 CONNECT-UDP over QUIC/HTTP/3) survives DPI because it presents as ordinary
HTTPS traffic to a CDN edge. But MASQUE introduces overhead: a QUIC handshake (1-RTT in the
best case, TLS 1.3), per-packet AEAD framing, and the CONNECT-UDP flow-establishment exchange.
For peers on a clean, non-DPI-filtered path, this overhead is pure waste — plain UDP is faster,
uses less CPU, and has a smaller per-packet header budget (28 bytes UDP/IPv4 vs. QUIC's
variable-length header + AEAD tag).

The system must therefore **start with the cheapest transport that works** and **escalate only
when the cheaper transport provably cannot reach the peer**.

## 2. TransportPolicy — Definition

A JSON-serializable policy that controls the escalation ladder. It lives alongside the
`NetworkMap` configuration and is the single source of truth for per-peer transport strategy.

### 2.1 JSON Schema

```json
{
  "order": ["plain-udp", "lwo", "masque-quic-standin"],
  "allow_downgrade": true,
  "backoff": {
    "initial": "1s",
    "max": "30s",
    "multiplier": 2.0
  },
  "probe_timeout": "5s",
  "max_attempts_per_transport": 3,
  "downgrade_probe_interval": "300s"
}
```

### 2.2 Field Reference

| Field | Type | Default | Description |
|---|---|---|---|
| `order` | `[]string` | `["plain-udp","lwo","masque-quic-standin"]` | Transport names tried left-to-right. Names match `Transport::name()` / `Connection::kind()`. |
| `allow_downgrade` | `bool` | `true` | When `true`, the system periodically probes cheaper transports even after escalating; if a cheaper transport becomes reachable, it downgrades. |
| `backoff.initial` | `duration` | `"1s"` | Wait before the first retry of a given transport. |
| `backoff.max` | `duration` | `"30s"` | Maximum backoff cap — no single wait between transport attempts exceeds this. |
| `backoff.multiplier` | `float64` | `2.0` | Exponential backoff multiplier. Wait `initial * multiplier^(attempt-1)`, capped at `max`. |
| `probe_timeout` | `duration` | `"5s"` | How long to wait for a single `dial()` to succeed before counting it as failed. |
| `max_attempts_per_transport` | `uint` | `3` | Number of dial attempts before declaring this transport dead and escalating. |
| `downgrade_probe_interval` | `duration` | `"300s"` | When `allow_downgrade` is true, how often to probe cheaper transports. |

### 2.3 Go Type (coordinator/config layer)

```go
package escalation

import "time"

// TransportPolicy governs the auto-escalation ladder for a single peer.
type TransportPolicy struct {
	Order                  []string      `json:"order"`
	AllowDowngrade         bool          `json:"allow_downgrade"`
	Backoff                BackoffConfig `json:"backoff"`
	ProbeTimeout           time.Duration `json:"probe_timeout"`
	MaxAttemptsPerTransport uint          `json:"max_attempts_per_transport"`
	DowngradeProbeInterval  time.Duration `json:"downgrade_probe_interval"`
}

// BackoffConfig controls the exponential backoff between dial attempts.
type BackoffConfig struct {
	Initial    time.Duration `json:"initial"`
	Max        time.Duration `json:"max"`
	Multiplier float64       `json:"multiplier"`
}

// DefaultPolicy returns the canonical default escalation policy.
func DefaultPolicy() TransportPolicy {
	return TransportPolicy{
		Order:                  []string{"plain-udp", "lwo", "masque-quic-standin"},
		AllowDowngrade:         true,
		Backoff:                BackoffConfig{Initial: 1 * time.Second, Max: 30 * time.Second, Multiplier: 2.0},
		ProbeTimeout:           5 * time.Second,
		MaxAttemptsPerTransport: 3,
		DowngradeProbeInterval:  5 * time.Minute,
	}
}

// Validate returns an error describing the first policy invariant violation,
// or nil if valid. Mandatory checks per §11.4.6 (no guessing on defaults):
//   - order is non-empty
//   - every entry in order matches a known transport name
//   - initial > 0 && initial <= max
//   - multiplier >= 1.0
//   - probe_timeout > 0
//   - max_attempts_per_transport >= 1
//   - downgrade_probe_interval > 0 (if allow_downgrade)
func (p *TransportPolicy) Validate(knownTransports []string) error {
	// Implementation: walk each invariant, return descriptive error on first violation.
	// Never guess — every rejected value cites which invariant broke.
	return nil // placeholder — real implementation per §11.4.6
}
```

### 2.4 Rust Type (transport runtime layer)

```rust
use serde::{Deserialize, Serialize};
use std::time::Duration;

/// Transport escalation policy for a single peer.
///
/// Lives in `helix-core` (alongside `NetworkMap`) so both the reconciler
/// and the orchestrator can reference it without a dependency cycle.
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct TransportPolicy {
    /// Transport names tried left-to-right. Each entry MUST match a
    /// `Transport::name()` / `Connection::kind()` value registered in the
    /// `TransportRegistry`.
    pub order: Vec<String>,

    /// When true, periodically probe cheaper transports and downgrade if
    /// one becomes reachable.
    #[serde(default = "default_allow_downgrade")]
    pub allow_downgrade: bool,

    /// Exponential backoff configuration.
    #[serde(default)]
    pub backoff: BackoffConfig,

    /// Per-dial-attempt timeout.
    #[serde(default = "default_probe_timeout")]
    pub probe_timeout: Duration,

    /// Number of dial attempts before escalating to the next transport.
    #[serde(default = "default_max_attempts")]
    pub max_attempts_per_transport: u32,

    /// Interval between downgrade probes (only meaningful when
    /// `allow_downgrade` is true).
    #[serde(default = "default_downgrade_probe_interval")]
    pub downgrade_probe_interval: Duration,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct BackoffConfig {
    /// Initial backoff duration.
    #[serde(default = "default_backoff_initial")]
    pub initial: Duration,

    /// Maximum backoff duration — individual waits never exceed this.
    #[serde(default = "default_backoff_max")]
    pub max: Duration,

    /// Exponential multiplier applied after each failed attempt:
    /// wait = min(initial * multiplier^(attempt-1), max).
    #[serde(default = "default_backoff_multiplier")]
    pub multiplier: f64,
}

// --- Default implementations (canonical values, never guessed) ---

fn default_allow_downgrade() -> bool { true }
fn default_probe_timeout() -> Duration { Duration::from_secs(5) }
fn default_max_attempts() -> u32 { 3 }
fn default_downgrade_probe_interval() -> Duration { Duration::from_secs(300) }
fn default_backoff_initial() -> Duration { Duration::from_secs(1) }
fn default_backoff_max() -> Duration { Duration::from_secs(30) }
fn default_backoff_multiplier() -> f64 { 2.0 }

impl Default for BackoffConfig {
    fn default() -> Self {
        Self {
            initial: default_backoff_initial(),
            max: default_backoff_max(),
            multiplier: default_backoff_multiplier(),
        }
    }
}

impl Default for TransportPolicy {
    fn default() -> Self {
        Self {
            order: vec![
                "plain-udp".into(),
                "lwo".into(),
                "masque-quic-standin".into(),
            ],
            allow_downgrade: default_allow_downgrade(),
            backoff: BackoffConfig::default(),
            probe_timeout: default_probe_timeout(),
            max_attempts_per_transport: default_max_attempts(),
            downgrade_probe_interval: default_downgrade_probe_interval(),
        }
    }
}

impl BackoffConfig {
    /// Compute the wait duration for attempt `n` (1-indexed).
    /// `wait = min(initial * multiplier^(n-1), max)`
    /// All inputs are non-negative and multiplier >= 1.0 by construction
    /// (enforced at policy validation time — see TransportPolicy::validate).
    pub fn wait_for_attempt(&self, attempt: u32) -> Duration {
        if attempt == 0 {
            return Duration::ZERO; // first attempt = no prior wait
        }
        let factor = self.multiplier.powi((attempt - 1) as i32);
        let dur = Duration::from_secs_f64(self.initial.as_secs_f64() * factor);
        dur.min(self.max)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn backoff_wait_first_attempt() {
        let cfg = BackoffConfig::default();
        assert_eq!(cfg.wait_for_attempt(0), Duration::ZERO);
    }

    #[test]
    fn backoff_wait_exponential() {
        let cfg = BackoffConfig::default(); // initial=1s, multiplier=2.0
        assert_eq!(cfg.wait_for_attempt(0), Duration::ZERO);    // no wait before first try
        assert_eq!(cfg.wait_for_attempt(1), Duration::from_secs(1));  // 1 * 2^0
        assert_eq!(cfg.wait_for_attempt(2), Duration::from_secs(2));  // 1 * 2^1
        assert_eq!(cfg.wait_for_attempt(3), Duration::from_secs(4));  // 1 * 2^2
        assert_eq!(cfg.wait_for_attempt(4), Duration::from_secs(8));  // 1 * 2^3
    }

    #[test]
    fn backoff_wait_respects_max() {
        let cfg = BackoffConfig {
            initial: Duration::from_secs(1),
            max: Duration::from_secs(5),
            multiplier: 2.0,
        };
        // attempt 4 would be 1*2^3=8s, capped at 5s
        assert_eq!(cfg.wait_for_attempt(4), Duration::from_secs(5));
        // attempt 5 would be 1*2^4=16s, capped at 5s
        assert_eq!(cfg.wait_for_attempt(5), Duration::from_secs(5));
    }
}
```

## 3. Escalation Algorithm

The algorithm is **deterministic**: given the same `TransportPolicy`, same `TransportRegistry`,
and same network conditions, every run produces the same outcome (§11.4.50).

### 3.1 Pseudocode

```
function escalate(peer, policy, registry, event_bus):
    // ---- Phase 1: walk the ladder ----
    current_transport_idx = find_starting_index(policy.order, peer.desired_transport)

    for idx in [current_transport_idx .. policy.order.len()]:
        transport_name = policy.order[idx]
        transport = registry.resolve(transport_name)
        if transport is None:
            log WARNING: "transport {transport_name} not registered, skipping"
            continue

        event_bus.emit(EscalationEvent::TryingTransport{transport_name})

        for attempt in [1 .. policy.max_attempts_per_transport]:
            if attempt > 1:
                wait = policy.backoff.wait_for_attempt(attempt)
                sleep(wait)

            conn = transport.dial(peer.addr) with timeout policy.probe_timeout
            if conn succeeds:
                // ---- SUCCESS: lock onto this transport ----
                record_active_transport(peer, transport_name, policy)
                event_bus.emit(EscalationEvent::Connected{transport_name, attempt, rtt})
                if policy.allow_downgrade:
                    spawn_background: downgrade_probe_loop(peer, policy, registry, event_bus,
                                                           cheaper_than=idx)
                return conn

            // ---- FAILURE: record and continue ----
            event_bus.emit(EscalationEvent::DialFailed{transport_name, attempt, error})

        // ---- All attempts for this transport exhausted ----
        event_bus.emit(EscalationEvent::TransportExhausted{transport_name})

    // ---- Phase 2: all transports exhausted ----
    event_bus.emit(EscalationEvent::AllExhausted)
    return Err("all transports exhausted for peer {peer.name}")

function find_starting_index(order, desired_transport):
    if desired_transport == "auto":
        return 0  // start from cheapest
    idx = order.index_of(desired_transport)
    if idx is not None:
        return idx  // operator-specified starting point
    log WARNING: "{desired_transport} not in policy order, starting from 0"
    return 0

function downgrade_probe_loop(peer, policy, registry, event_bus, cheaper_than):
    loop every policy.downgrade_probe_interval:
        for idx in [0 .. cheaper_than]:  // only cheaper transports
            transport_name = policy.order[idx]
            transport = registry.resolve(transport_name)
            if transport is None: continue

            conn = transport.dial(peer.addr) with timeout policy.probe_timeout
            if conn succeeds:
                // Downgrade: switch active transport to this cheaper one.
                event_bus.emit(EscalationEvent::Downgraded{
                    from: current_transport,
                    to: transport_name,
                })
                switch_active_transport(peer, transport_name)
                // Close old (more expensive) connection.
                old_conn.close()
                // Re-spawn downgrade probe loop from new position.
                spawn_background: downgrade_probe_loop(peer, policy, registry, event_bus,
                                                       cheaper_than=idx)
                return  // old loop exits; new loop takes over

            // Probe failed — cheaper transport still unreachable. Stay put.
```

### 3.2 Event Vocabulary

New `TunnelEvent` variants (additive to the existing `event.rs` enum, matching the pattern of
`TransportSwitched`, `RouteAdded`, etc.):

```rust
/// An escalation-ladder lifecycle event.
EscalationEvent {
    /// Peer name this event pertains to.
    peer: String,
    /// The specific escalation sub-event.
    event: EscalationSubEvent,
}

enum EscalationSubEvent {
    /// The ladder is trying a new transport.
    TryingTransport { transport: String },
    /// A single dial attempt failed.
    DialFailed { transport: String, attempt: u32, error: String },
    /// All attempts for this transport exhausted; moving to the next.
    TransportExhausted { transport: String },
    /// Connected successfully on this transport.
    Connected { transport: String, attempt: u32, rtt_ms: u64 },
    /// Downgraded from a more expensive transport to a cheaper one.
    Downgraded { from: String, to: String },
    /// Every transport in the policy order has been exhausted.
    AllExhausted,
}
```

### 3.3 Deterministic Behavior Guarantee (§11.4.50)

The algorithm is deterministic by construction:

1. **Transport ordering** is fixed by `policy.order` — no random shuffle, no "fastest-first" heuristic.
2. **Backoff timing** is purely a function of `backoff.initial`, `backoff.multiplier`, and the
   attempt counter — wall-clock variation in `sleep()` does not affect the decision (a transport
   either connects or it does not; the wait only spaces out retries to avoid network congestion).
3. **Probe timeout** is a fixed duration — a `dial()` that takes longer than `probe_timeout`
   is treated as a failure regardless of whether it *might* have succeeded given more time.
4. **Downgrade probes** fire on a fixed interval — the system does not probe "when idle" or on
   some heuristic trigger.

Given the same `TransportPolicy` and the same network state, two independent runs produce the
same active transport choice at every point in the timeline. The backoff sleeps are the only
non-deterministic wall-clock component and they do not change the decision outcome.

## 4. LWO — LightWire Obfuscation Transport

### 4.1 Rationale

Between "plain UDP" (zero overhead, trivially fingerprintable) and "MASQUE/QUIC" (full TLS 1.3
encapsulation, highest overhead), there is a middle ground that defeats simple DPI without paying
the QUIC tax.

Most DPI devices that block WireGuard do so by matching the first few bytes of the Noise IK
handshake initiation message — specifically the `type` field (`0x01`) and the `reserved` zero
bytes at known offsets. These are **static, protocol-mandated values** that appear in every
WireGuard handshake-initiation message regardless of keypair.

LWO (LightWire Obfuscation) XORs the first **N** bytes (default 64) of every outbound datagram
with a per-connection symmetric key derived from the peer's WireGuard public key. The receiver
applies the same XOR to recover the original datagram. This:

- **Defeats simple DPI** — the handshake signature bytes (`0x01 0x00 0x00 0x00` at the
  handshake-initiation message start) are no longer visible in plaintext.
- **Adds negligible overhead** — XOR is a single CPU instruction per word; there is no extra
  framing, no extra round-trips, no AEAD.
- **Preserves datagram boundaries** — the obfuscation is a pure per-datagram transform with no
  state carried between datagrams.

### 4.2 Key Derivation

```
lwo_key = HKDF-Expand(
    PRK = HKDF-Extract(salt = "helix-lwo-v1", ikm = peer_wg_pubkey),
    info = "lwo-xor-key",
    length = N   // default N = 64 bytes
)
```

`peer_wg_pubkey` is the 32-byte Curve25519 public key already known to both peers (it is in
the `NetworkMap.peers[].wg_pubkey` field). HKDF with a well-known salt prevents the derived
key from being secret — **LWO is obfuscation, not encryption**, and the key is derivable by
anyone who knows the peer's public key (which is not secret on the wire).

### 4.3 Transport Implementation Sketch (Rust)

LWO is **not a new protocol** — it is a lightweight shim that wraps an existing `Transport`
implementation and applies XOR obfuscation to the first N bytes of every datagram.

```rust
/// A transport shim that XOR-obfuscates the first N bytes of every datagram.
///
/// Wraps an inner `Transport` (typically `plain::UdpTransport`) and applies
/// per-datagram XOR with a key derived from the peer's WireGuard public key.
pub struct LwoTransport {
    inner: Arc<dyn Transport>,
    key: Vec<u8>,           // N bytes, derived via HKDF from peer_wg_pubkey
}

impl LwoTransport {
    /// Create a new LWO shim wrapping `inner`, deriving the obfuscation key
    /// from `peer_wg_pubkey` (32-byte Curve25519 public key, hex-encoded).
    pub fn new(inner: Arc<dyn Transport>, peer_wg_pubkey: &str, n_bytes: usize) -> Self {
        let pubkey_bytes = hex::decode(peer_wg_pubkey)
            .expect("peer_wg_pubkey must be valid hex");
        assert_eq!(pubkey_bytes.len(), 32, "Curve25519 public key is 32 bytes");
        let key = hkdf_expand(
            salt = b"helix-lwo-v1",
            ikm = &pubkey_bytes,
            info = b"lwo-xor-key",
            length = n_bytes,  // default 64
        );
        Self { inner, key }
    }
}

#[async_trait]
impl Transport for LwoTransport {
    async fn dial(&self, addr: SocketAddr) -> TpResult<Box<dyn Connection>> {
        let raw_conn = self.inner.dial(addr).await?;
        Ok(Box::new(LwoConnection {
            inner: raw_conn,
            key: self.key.clone(),
        }))
    }

    async fn listen(&self, addr: SocketAddr) -> TpResult<Box<dyn Listener>> {
        let raw_listener = self.inner.listen(addr).await?;
        Ok(Box::new(LwoListener {
            inner: raw_listener,
            key: self.key.clone(),
        }))
    }

    fn name(&self) -> &'static str { "lwo" }
}

/// An LWO-obfuscated connection. Every `send` XORs the first N bytes;
/// every `recv` reverses the XOR.
struct LwoConnection {
    inner: Box<dyn Connection>,
    key: Vec<u8>,
}

#[async_trait]
impl Connection for LwoConnection {
    async fn send(&self, buf: &[u8]) -> TpResult<usize> {
        let mut obfuscated = buf.to_vec();
        let n = self.key.len().min(obfuscated.len());
        for i in 0..n {
            obfuscated[i] ^= self.key[i];
        }
        self.inner.send(&obfuscated).await
    }

    async fn recv(&self, buf: &mut [u8]) -> TpResult<usize> {
        let n = self.inner.recv(buf).await?;
        let xor_len = self.key.len().min(n);
        for i in 0..xor_len {
            buf[i] ^= self.key[i];
        }
        Ok(n)
    }

    async fn close(&self) -> TpResult<()> { self.inner.close().await }
    fn local_addr(&self) -> TpResult<SocketAddr> { self.inner.local_addr() }
    fn peer_addr(&self) -> TpResult<SocketAddr> { self.inner.peer_addr() }
    fn kind(&self) -> &'static str { "lwo" }
    fn effective_mtu(&self) -> u16 {
        // LWO adds no framing overhead — XOR is in-place.
        // Same MTU as the inner transport (plain-udp: 1420).
        self.inner.effective_mtu()
    }
}
```

### 4.4 LWO in the Escalation Order

The default `order` is `["plain-udp", "lwo", "masque-quic-standin"]`. This means:

1. **First try plain UDP** — zero overhead, works if DPI is absent or permissive.
2. **Then try LWO** — adds ~0 CPU overhead, defeats simple DPI that matches static WG handshake
   bytes, works if the DPI is signature-based but not doing statistical traffic analysis.
3. **Finally try MASQUE/QUIC** — full TLS 1.3 encapsulation, survives nearly all DPI, highest
   overhead.

A peer behind a DPI that drops the WG handshake signature will fail at step 1 (plain UDP —
handshake-initiation never elicits a response) and succeed at step 2 (LWO — the obfuscated
handshake-initiation passes the DPI, peer responds, tunnel establishes). The MASQUE step is
never reached, saving the QUIC overhead.

## 5. Integration with OrchConfig and NetworkMap

### 5.1 Per-Peer Transport Selection

Currently `OrchConfig.transport_name` is a single string — one transport for all peers.
The escalation ladder requires per-peer transport state:

```rust
// Addition to OrchConfig (helix-orch)
pub struct OrchConfig {
    // ... existing fields ...
    /// Default escalation policy used when a peer has no peer-specific override.
    pub default_transport_policy: TransportPolicy,
}
```

And a per-peer entry in the orchestrator's runtime state:

```rust
/// Per-peer transport state tracked by the escalator.
struct PeerTransportState {
    /// The transport currently in active use for this peer.
    active_transport: String,
    /// Index of active_transport in the policy order (for downgrade probing).
    active_transport_idx: usize,
    /// The escalation policy for this peer (may be default or per-peer override).
    policy: TransportPolicy,
    /// When the last downgrade probe ran (for allow_downgrade).
    last_downgrade_probe: Instant,
    /// Per-transport metrics for this peer.
    metrics: HashMap<String, TransportMetrics>,
}
```

### 5.2 Initial Transport from NetworkMap

The `NetworkMap.self.transport` field already carries the desired transport (`"auto"`, `"wireguard"`,
`"masque"`, etc.). The escalation ladder interprets these values:

| `self.transport` | Meaning |
|---|---|
| `"auto"` | Start from the cheapest transport in the policy order (`policy.order[0]`). `find_starting_index` returns 0. |
| `"plain-udp"` | Start from `"plain-udp"` if it is in the policy order; otherwise start from the beginning. |
| `"lwo"` | Start from `"lwo"` — skip plain UDP even if cheaper. Operator override for known-DPI environments. |
| `"masque"` | Start from `"masque-quic-standin"` — skip both plain and LWO. Operator override for hostile networks. |

The `find_starting_index()` function in the pseudocode (Section 3.1) implements this logic.

### 5.3 The Escalator Component

A new `Escalator` struct in `helix-orch` that lives alongside the existing `Orchestrator`:

```rust
/// The transport auto-escalation ladder.
///
/// Manages per-peer transport state, runs the escalation algorithm,
/// and emits `EscalationEvent`s on the event bus.
pub struct Escalator {
    /// Shared reference to the transport registry (same one the orchestrator uses).
    registry: Arc<TransportRegistry>,
    /// Per-peer transport state.
    peers: Arc<RwLock<HashMap<String, PeerTransportState>>>,
    /// Default policy for peers without an explicit override.
    default_policy: TransportPolicy,
    /// The event bus for emitting escalation lifecycle events.
    event_bus: EventBus,
}

impl Escalator {
    /// Create a new escalator with the given default policy and registry.
    pub fn new(default_policy: TransportPolicy, registry: Arc<TransportRegistry>, event_bus: EventBus) -> Self;

    /// Escalate for a single peer. Returns a dialed Connection on the first
    /// successful transport, or an error if all transports are exhausted.
    pub async fn connect(&self, peer_name: &str, peer_addr: SocketAddr) -> Result<Box<dyn Connection>, EscalationError>;

    /// Get the currently active transport for a peer.
    pub async fn active_transport(&self, peer_name: &str) -> Option<String>;

    /// Trigger an immediate downgrade probe for a peer (used by the background
    /// probe loop — see Section 3.1 pseudocode).
    pub async fn probe_downgrade(&self, peer_name: &str) -> Option<String>;
}
```

### 5.4 Wire-in Point in Orchestrator::connect()

The current `Orchestrator::connect()` does a single `transport.dial(server_addr)`. With the
escalation ladder, it delegates to the `Escalator`:

```rust
impl Orchestrator {
    pub async fn connect(&self) -> Result<(), OrchError> {
        // ... existing state transitions ...

        let conn = self.escalator
            .connect(&self.current_peer_name, self.config.server_addr)
            .await
            .map_err(|e| OrchError::ConnectionFailed(e.to_string()))?;

        // Store the connection and the active transport name from the escalator.
        let active = self.escalator.active_transport(&self.current_peer_name).await
            .unwrap_or_else(|| self.config.transport_name.clone());
        self.switch_transport(active).await;

        // ... existing Connected event emission ...
    }
}
```

## 6. Metrics

### 6.1 Per-Transport Metrics

```rust
/// Metrics collected per transport, per peer.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct TransportMetrics {
    /// Total dial attempts.
    pub dial_attempts: u64,
    /// Successful dials (connection established).
    pub dial_successes: u64,
    /// Failed dials (timeout, connection refused, protocol error).
    pub dial_failures: u64,
    /// How many times this transport was the final successful choice in an
    /// escalation cycle (i.e. the ladder stopped here).
    pub escalation_wins: u64,
    /// How many times the system was on a more expensive transport and
    /// downgraded to this one (allow_downgrade).
    pub downgrades_to: u64,
    /// How many times the system was on this transport and escalated away
    /// (this transport failed and a more expensive one was chosen).
    pub escalations_from: u64,
    /// Cumulative time spent connected on this transport.
    pub total_connected_time: Duration,
    /// Last measured RTT on this transport.
    pub last_rtt_ms: u64,
}

impl TransportMetrics {
    /// Success rate as a fraction [0.0, 1.0]. Returns 0.0 if no attempts yet.
    pub fn success_rate(&self) -> f64 {
        if self.dial_attempts == 0 { return 0.0; }
        self.dial_successes as f64 / self.dial_attempts as f64
    }
}
```

### 6.2 Global Distribution

```rust
/// Snapshot of transport distribution across all peers.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct TransportDistribution {
    /// Number of peers currently using each transport.
    pub by_transport: HashMap<String, u64>,
    /// Total peers tracked.
    pub total_peers: u64,
    /// Peers that exhausted all transports (no connection).
    pub exhausted: u64,
}
```

### 6.3 Metrics Emission

The `Escalator` emits metrics snapshots on the event bus periodically (every `stats_interval`,
matching the existing stats loop in the orchestrator). A new event variant:

```rust
TunnelEvent::EscalationMetrics {
    per_peer: HashMap<String, HashMap<String, TransportMetrics>>,
    distribution: TransportDistribution,
}
```

## 7. Honest Assessment

### 7.1 LWO Limitations

LWO is **obfuscation, not encryption**. A determined DPI device can still detect LWO-obfuscated
WireGuard traffic through:

1. **Statistical traffic analysis** — WG handshake-initiation messages have a characteristic
   size (~148 bytes) and the response has a different characteristic size (~92 bytes). XOR does
   not change the datagram length or the timing pattern.

2. **Entropy analysis** — the Noise IK handshake payload (after XOR of the first N bytes) still
   contains Curve25519 ephemeral public keys (32 bytes of high-entropy random-looking data) at
   predictable offsets. XOR of a fixed key with a fixed plaintext produces a fixed ciphertext —
   the obfuscated handshake-initiation message is itself a static signature (different from the
   raw one, but equally static for a given peer keypair).

3. **Traffic volume patterns** — WG data-plane messages have a characteristic cadence (keepalive
   every ~25 seconds, data bursts following user activity). LWO does nothing to change this.

4. **No forward secrecy** — the LWO key is static (derived from the peer's static public key).
   If the obfuscation key is ever recovered, all past and future datagrams for that peer are
   de-obfuscatable. This is acceptable because LWO provides zero security — it only provides
   obfuscation — so key recovery reveals nothing the attacker could not already see on the wire
   (the WG plaintext, which is itself encrypted by WireGuard's own Noise IK).

### 7.2 When LWO Is Sufficient

LWO is effective against DPI middleboxes that use **signature-based detection** — matching
static byte patterns at fixed offsets without performing full protocol state-machine analysis.
This describes the majority of consumer and enterprise DPI deployments today. Examples:

- An ISP that drops "unknown UDP protocol on port 51820" by matching the WG handshake type byte.
- A corporate firewall that blocks "VPN traffic" by matching the Noise IK message type and
  reserved zero fields.
- A country-level firewall that uses simple DPI rules rather than full stateful protocol analysis.

### 7.3 When LWO Is Insufficient — MASQUE Required

LWO will NOT defeat:

1. **Statistical/ML-based DPI** that classifies traffic by flow characteristics rather than
   static byte signatures (datagram size distribution, inter-arrival timing, entropy profile).
2. **Deep protocol analysis** that reassembles the obfuscated stream, identifies the Noise IK
   state machine by behavior (not bytes), and blocks regardless of obfuscation.
3. **Whitelist-only networks** that allow only TCP/443 (HTTPS) and drop all UDP — LWO is still
   UDP, and no amount of XOR changes the IP protocol number.

In these environments, MASQUE/QUIC (which presents as ordinary HTTPS to the network) is the
correct choice. The escalation ladder handles this automatically: LWO fails (UDP blocked
entirely), the ladder escalates to `masque-quic-standin`, QUIC-over-UDP-443 passes through.

### 7.4 MASQUE-quic-standin Status

The current MASQUE implementation is labeled `masque-quic-standin` — an explicitly-labeled
simplified stand-in, not a fully RFC 9298-conformant CONNECT-UDP/HTTP/3 implementation
(see `helix_masque::connect` module docs for the full research trail). The escalation ladder
uses this label honestly: when the ladder records "connected on masque-quic-standin," it means
the connection is on the current stand-in implementation, not on a production-grade MASQUE
proxy. This label will change to `masque-h3` when the implementation graduates to real
RFC 9298 compliance.

## 8. Sources Verified (§11.4.99)

| Source | URL | Fetched | Relevance |
|---|---|---|---|
| RFC 9298 — Proxying UDP in HTTP | https://www.rfc-editor.org/rfc/rfc9298.html | 2026-07-05 (helix-masque) | MASQUE CONNECT-UDP protocol definition |
| RFC 9297 — HTTP Datagrams | https://www.rfc-editor.org/rfc/rfc9297.html | 2026-07-05 (helix-masque) | HTTP-Datagram framing used by MASQUE |
| RFC 9221 — QUIC DATAGRAM frames | https://www.rfc-editor.org/rfc/rfc9221.html | 2026-07-05 (helix-masque) | Unreliable datagram extension to QUIC |
| RFC 9000 — QUIC | https://www.rfc-editor.org/rfc/rfc9000.html | 2026-07-05 (helix-masque) | QUIC transport protocol |
| RFC 5869 — HKDF | https://www.rfc-editor.org/rfc/rfc5869.html | 2026-07-08 | Key derivation for LWO key material |
| WireGuard Noise IK handshake format | https://www.wireguard.com/protocol/ | 2026-07-08 | Handshake-initiation message structure — identifies the static bytes LWO targets |
| Exponential backoff — AWS Architecture Blog | https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/ | 2026-07-08 | Canonical reference for exponential backoff algorithm used in `BackoffConfig` |
| XOR obfuscation — Shadowsocks and Obfsproxy design | https://github.com/shadowsocks/shadowsocks-org/issues/44 | 2026-07-08 | Precedent for XOR-based traffic obfuscation in circumvention tools |
