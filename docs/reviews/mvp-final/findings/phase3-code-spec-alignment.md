# Phase 3 — Code/Spec Alignment Report

**Scope:** Reconcile all Helix code submodules against the final MVP spec in `docs/research/mvp/final/`.
**Date:** 2026-07-05
**Author:** senior systems engineer (subagent)
**Status:** findings + aligned artifacts delivered; commit deferred to coordinator.

---

## Executive summary

| Submodule | Languages / build | Real code? | Tests | Spec gaps |
|---|---|---|---|---|
| `helix_core` | Rust, Cargo workspace | **Yes** — transport, WG, orchestrator, MASQUE, TUN, map reconciler | 119 passed, 2 ignored | Trait shape differs from frozen spec; no end-to-end tunnel; MASQUE is a labeled stand-in |
| `helix_edge` | Rust, Cargo | **Yes** — MASQUE listener, decoy, gateway relay | 11 passed | No kernel-WG integration; decoy is TCP-only static HTML; no privileged :443 binding |
| `helix_go` | Go, Go modules | **Partial** — real quic-go + masque-go CONNECT-UDP edge spike | 2 passed (race enabled) | No control-plane services (identity, IPAM, policy, coordinator) |
| `helix_proto` | Protobuf | **No** (scaffolding only before this task) | N/A | **Filled by this deliverable** — 5 `.proto` files + buf config + Go stubs |
| `helix_transport` | Rust (declared) | No source | N/A | Entirely scaffolding; intended to merge into `helix_core::helix-transport` |
| `helix_ui` | Dart/Flutter (declared) | No source | N/A | Scaffolding only |
| `helix_shims` | Multi-platform (declared) | No source | N/A | Scaffolding only |
| `helix_design` | Tokens (declared) | No source | N/A | Scaffolding only |

**Key finding:** `helix_core` + `helix_edge` have genuine Phase-0 primitives and several passing integration proofs, but none of the submodules yet implements the full MVP. The biggest single gap is the control plane (`helix_go`), followed by the absence of schema-generated stubs for Dart/Rust and the MASQUE stand-in in Rust.

---

## Per-submodule inventory

### `helix_core` — Rust client/connector core

**Languages / build:** Rust (edition 2024), Cargo workspace (`cargo test --all-targets`).

**Key files:**
- `Cargo.toml` — workspace root
- `crates/helix-transport/src/lib.rs` — `Transport`/`Connection`/`Listener` traits + `TransportRegistry`
- `crates/helix-transport/src/plain.rs` — plain-UDP transport
- `crates/helix-wg/src/lib.rs` — WireGuard key types, `boringtun` wrapper, config
- `crates/helix-wg/src/device.rs` — peer/device lifecycle
- `crates/helix-wg/src/handshake.rs` — Noise IK handshake state helpers
- `crates/helix-wg/src/noise.rs` — `boringtun::noise::Tunn` wrapper
- `crates/helix-wg/src/timers.rs` — WG timer state
- `crates/helix-orch/src/lib.rs`, `orchestrator.rs`, `event.rs`, `wg_session.rs` — three-loop orchestrator + event bus + real WG handshake driver
- `crates/helix-masque/src/lib.rs`, `quic.rs`, `connect.rs`, `datagram.rs`, `proxy.rs` — MASQUE transport over `quinn`
- `crates/helix-tun/src/lib.rs` — Linux TUN abstraction
- `crates/helix-core/src/lib.rs`, `cli.rs`, `map.rs` — re-exports, `map.json` reconciler
- `crates/helix-core/src/bin/helix-client.rs`, `helix-connector.rs` — CLI binary stubs

**Current capability:**
- `Transport`/`Connection`/`Listener` trait system and registry.
- Plain-UDP loopback transport with real send/recv.
- Real `boringtun`-based X25519 key generation and Noise IK handshake between client and connector over loopback UDP.
- Event bus (`EventBus`) with `TunnelEvent`/`TunnelState`/`TunnelStats`.
- Orchestrator skeleton with three async loops (stats, connection, keepalive).
- Real `quinn` QUIC endpoints with TLS/SNI and RFC 9221 unreliable datagrams.
- MASQUE transport that implements `Transport`/`Connection`/`Listener` using a labeled simplified CONNECT-UDP stand-in over `quinn` streams + RFC-9297-style HTTP-Datagram framing.
- Linux TUN device creation via `tun` crate.
- Pure `map.json` diff/reconcile function matching the future `WatchNetworkMap` shape.

**Gaps vs. MVP spec (`01-data-plane.md`, `03-client-core-and-ui.md`):**
- The frozen `Transport` trait in the spec exposes `send`/`recv`/`kind`/`effective_mtu`/`health`/`close` and construction via a free `dial()` function. The current trait exposes `dial`/`listen`/`name` and `Connection` exposes `send`/`recv`/`close`/`local_addr`/`peer_addr`. This is a **shape mismatch** that must be reconciled before the auto-ladder and MTU contracts can be honored.
- No actual TUN ↔ WG ↔ transport packet pump. The WG handshake driver exists but does not read/write data-path packets or wire up `helix-tun`.
- `helix-masque` CONNECT-UDP flow establishment is explicitly a **simplified stand-in**, not real HTTP/3 Extended CONNECT / RFC 9298. The `h3` crate was evaluated and deemed not reasonably achievable for this Phase-0 spike.
- No DAITA, Shadowsocks, UDP-over-TCP, LWO, Hysteria2, or connect-ip transports.
- No auto-ladder / `TransportPolicy` failure budget / per-network memory.
- No `RouteMap` application to live orchestrator state.
- No FFI surface (`flutter_rust_bridge` / UniFFI) for Flutter.

**Decoupling/reusability:**
- The crate is deliberately project-agnostic: no hardcoded consumer-project name, hostname, or tenant assumption. Config is injected.
- `helix-transport` and `helix-wg` are already reusable library crates.
- The main coupling risk is the trait-shape mismatch: if external consumers adopt the current `Transport` trait, migrating to the frozen spec trait will be a breaking change.

---

### `helix_edge` — Rust data-plane edge gateway

**Languages / build:** Rust, Cargo (`cargo test --all-targets`).

**Key files:**
- `Cargo.toml`
- `src/lib.rs`, `edge.rs`, `gateway.rs`, `decoy.rs`, `main.rs`
- `tests/dependency_resolution_smoke.rs`, `decoy_probe_serves_believable_page.rs`, `edge_decoy_and_masque_flow_coexist.rs`, `masque_gateway_relay_byte_identical.rs`, `wg_handshake_through_masque_and_gateway.rs`

**Current capability:**
- Consumes `helix_core` crates via Cargo path dependencies (`../helix_core/crates/*`).
- Binds a real `helix-masque` server-role listener and a TCP decoy on the same port number.
- Relays decapsulated MASQUE datagrams to a configurable gateway UDP socket.
- Serves a static, non-revealing decoy HTML page to TCP probes.
- Integration proof: a real WireGuard Noise IK handshake completes through MASQUE + relay + loopback responder.

**Gaps vs. MVP spec (`01-data-plane.md`):**
- No real kernel-WireGuard or `boringtun` gateway socket integration. The gateway UDP socket is a loopback stand-in.
- Cannot bind privileged `:443` in this environment; tests use ephemeral high ports.
- Decoy is static HTTP/1.1 only. No HTTP/3-shaped decoy over QUIC.
- No verdict-map compiler / nftables/eBPF installation.
- No multi-hop, P2P, or DERP relay.

**Decoupling/reusability:**
- All addresses, hostnames, and certificates are caller-configured. The crate is reusable as a generic MASQUE-termination + decoy scaffolding module.
- Strong dependency on `helix_core` path layout; this is documented and tested.

---

### `helix_go` — Go control plane

**Languages / build:** Go 1.26.2, Go modules (`go test ./pkg/masqueedge/... -race -count=1 -v`).

**Key files:**
- `go.mod`
- `cmd/go-edge/main.go`
- `pkg/masqueedge/doc.go`, `client.go`, `server.go`, `gateway.go`, `tls.go`, `masqueedge_test.go`

**Current capability:**
- Real RFC 9298 CONNECT-UDP edge terminator using `quic-go` + `masque-go`.
- Server, client, loopback gateway stand-in, and self-signed TLS helper.
- Integration proof: payload round-trips byte-identical through CONNECT-UDP, including concurrent isolated flows.

**Gaps vs. MVP spec (`02-control-plane.md`):**
- **No control-plane services:** identity, registry, IPAM, PKI, policy compiler, coordinator, event bus, telemetry, REST/WS/SSE API.
- No Postgres + RLS data model.
- No Redis Streams backbone.
- No `Coordinator.WatchNetworkMap` server implementation.
- No `helixvpnctl` CLI.
- The `masqueedge` package is a Phase-0 spike, not the production edge.

**Decoupling/reusability:**
- `pkg/masqueedge` is project-agnostic reusable infrastructure: no hardcoded hostnames/ports.
- The rest of the repo is scaffolding.

---

### `helix_proto` — Protobuf + OpenAPI + codegen

**Languages / build:** Protobuf, `buf`, `protoc`.

**Key files (created/updated by this deliverable):**
- `buf.yaml`
- `buf.gen.yaml`
- `proto/helix/coordinator/v1/coordinator.proto`
- `proto/helix/session/v1/session.proto`
- `proto/helix/tunnel/v1/tunnel.proto`
- `proto/helix/ui/v1/ui.proto`
- `proto/helix/telemetry/v1/telemetry.proto`
- `gen/go/...` — generated Go message + Connect stubs

**Current capability:**
- Canonical agent contract (`Coordinator` service: Enroll, WatchNetworkMap, AdvertisePrefixes, ReportStatus) aligned with `v03-control-plane/protobuf-spec.md`.
- Session, tunnel/UI, and telemetry contracts defined.
- `buf lint` passes.
- Go stubs generated successfully.

**Gaps:**
- Dart and Rust stubs **not generated** in this environment because `protoc-gen-dart` and `protoc-gen-prost`/`protoc-gen-tonic` are not installed. Commands documented in `buf.gen.yaml` and `README.md`.
- No OpenAPI schema for the REST/WS surface yet.

---

### `helix_transport`, `helix_ui`, `helix_shims`, `helix_design`

These submodules are currently **scaffolding only** (README, AGENTS.md, constitution files, upstream recipes). They declare intent but contain no source code or tests.

**Disposition:**
- `helix_transport`: The spec intended this to be the standalone Rust transport crate, but the actual transport code lives inside `helix_core/crates/helix-transport` and `helix_core/crates/helix-masque`. Decision needed: either populate `helix_transport` as a thin re-export/redirect or remove/merge it. **Not deleted** in this pass per constraints.
- `helix_ui`, `helix_shims`, `helix_design`: Out of scope for Phase 3 code reconciliation; marked as future work in READMEs.

---

## API surface inventory and protobuf coverage

| Contract area | Spec owner | Proto file | Service / messages | Coverage |
|---|---|---|---|---|
| Enrollment | `02-control-plane.md` §9, `protobuf-spec.md` §4.1 | `coordinator.proto` | `Coordinator.Enroll`, `EnrollRequest`, `EnrollResponse`, `DeviceKind` | ✅ Full |
| WatchNetworkMap | `02-control-plane.md` §4, `protobuf-spec.md` §1, §3 | `coordinator.proto` | `Coordinator.WatchNetworkMap`, `WatchRequest`, `MapUpdate`, `NetworkMap`, `MapDelta`, `Peer`, `Via6Route`, `GatewayInfo`, `DnsConfig`, `TransportPolicy` | ✅ Full |
| Prefix advertisement | `02-control-plane.md` §7, `protobuf-spec.md` §1 | `coordinator.proto` | `Coordinator.AdvertisePrefixes`, `AdvertiseRequest`, `AdvertiseResponse` | ✅ Full |
| Status/presence | `02-control-plane.md` §6, `protobuf-spec.md` §4.8 | `coordinator.proto` | `Coordinator.ReportStatus`, `StatusReport`, `StatusAck`, `KeepAlive` | ✅ Full |
| Authentication/session | `04-security-privacy-pki.md` | `session.proto` | `Session.Authenticate`, `Session.ValidateToken`, `Session.RevokeToken` | ✅ MVP-aligned |
| Data-plane / tunnel | `01-data-plane.md` §5.2, `03-client-core-and-ui.md` §1 | `tunnel.proto` | `TunnelState`, `TransportKind`, `TunnelEvent`, `StatsUpdate`, `TunnelCommand`, `TransportHealth` | ✅ MVP-aligned |
| Client UI state | `03-client-core-and-ui.md` | `ui.proto` | `UiState`, `ConnectionCard`, `NetworkList`, `Notification` | ✅ MVP-aligned |
| Telemetry/observability | `02-control-plane.md` §5, `10-testing-acceptance-and-qa.md` | `telemetry.proto` | `Telemetry.SubmitMetrics`, `Telemetry.SubmitLog`, `MetricsBatch`, `LogEntry`, `EdgeHealthReport` | ✅ MVP-aligned |

**API surface gaps:**
- The `Transport` trait is not yet expressed in `.proto` (it is a Rust trait, not a wire contract). A future `transport_config.proto` could materialize `TransportConfig` for cross-language transport policy if the ladder is driven from Dart/Go.
- No proto for the compiled policy / verdict map (edge-side representation). This is currently a Rust data structure.
- No proto for kill-switch/DNS-leak state machine events; partially covered by `TunnelState.KILL_SWITCH_ACTIVE`.

---

## Decoupling/reusability assessment

| Component | Reusable today? | How other projects can reuse | Blockers |
|---|---|---|---|
| `helix-transport` crate | Yes | Depend on the crate; implement `Transport`/`Connection`/`Listener` for new carriers | Trait shape mismatch with frozen spec; breaking change likely |
| `helix-wg` crate | Yes | Use key types, handshake helpers, and `boringtun` wrapper | None significant |
| `helix-orch` crate | Partially | Use `EventBus` + `TunnelEvent`; WG handshake driver is usable | Orchestrator loops are still skeletal |
| `helix-masque` crate | Partially | Use `quinn` QUIC endpoints and HTTP-Datagram framing | CONNECT-UDP stand-in is not real RFC 9298; do not reuse as a spec-compliant MASQUE proxy |
| `helix-tun` crate | Yes | Linux-only TUN abstraction | Non-Linux platforms return `UnsupportedPlatform` |
| `helix_edge` lib | Partially | Use `spawn_edge` + decoy for MASQUE-termination scaffolding | No kernel-WG handoff |
| `helix_go/pkg/masqueedge` | Yes | Drop-in RFC 9298 CONNECT-UDP edge terminator using `quic-go`/`masque-go` | Not integrated with Helix control plane |
| `helix_proto` | Yes | Generated stubs for Go; Dart/Rust once plugins installed | Dart/Rust generators not installed here |

**Key decoupling actions needed:**
1. Reconcile the `Transport` trait to the frozen spec shape (`send`/`recv`/`kind`/`effective_mtu`/`health`/`close` + `dial()` constructor).
2. Move/merge `helix_transport` scaffolding decision.
3. Add platform abstraction layer so `helix-tun` can be implemented for Android/iOS/Windows/HarmonyOS/Aurora without forking the core.
4. Generate Dart + Rust stubs before any Flutter/Rust consumer commits to hand-written structs.

---

## Build / test results

### `helix_core`

```
cargo test --all-targets
```

Result: **PASS** — all crates green.

| Crate | Passed | Failed | Ignored |
|---|---|---|---|
| helix-core (lib) | 20 | 0 | 0 |
| helix-core bins | 2 | 0 | 0 |
| helix-masque | 27 | 0 | 0 |
| helix-orch | 16 | 0 | 0 |
| helix-transport (lib) | 12 | 0 | 0 |
| helix-transport integration | 3 | 0 | 0 |
| helix-tun | 5 | 0 | 0 |
| helix-wg | 34 | 0 | 2 |
| **Total** | **119** | **0** | **2** |

Note: 2 `helix-wg` tests are ignored with comment "HVPN-P0-011 need boringtun transport key alignment".

### `helix_edge`

```
cargo test --all-targets
```

Result: **PASS** — 11 passed, 0 failed.

| Test file | Passed |
|---|---|
| unit (lib + main) | 2 |
| decoy_probe_serves_believable_page | 2 |
| dependency_resolution_smoke | 4 |
| edge_decoy_and_masque_flow_coexist | 1 |
| masque_gateway_relay_byte_identical | 1 |
| wg_handshake_through_masque_and_gateway | 1 |

### `helix_go`

```
go test ./pkg/masqueedge/... -race -count=1 -v
```

Result: **PASS** — 2 tests passed (race detector enabled). Console shows benign `quic-go` UDP-buffer warnings and expected EOF-on-close errors.

### `helix_proto`

```
buf lint
buf generate
```

Result: **PASS** — lint clean; Go stubs generated under `gen/go/`.

---

## Open gaps needing follow-up

### Must-do before MVP
1. **Reconcile `Transport` trait shape** with the frozen spec in `v02-data-plane/transport-trait.md`.
2. **Implement the control plane in `helix_go`**: identity, registry, IPAM, PKI, policy compiler, coordinator, event bus, REST/Connect API.
3. **Wire TUN ↔ WG ↔ transport packet pump** in `helix-core` / `helix-orch`.
4. **Generate Dart + Rust proto stubs** (install `protoc-gen-dart`, `protoc-gen-prost`, `protoc-gen-tonic`).
5. **Replace MASQUE stand-in with real RFC 9298 CONNECT-UDP over HTTP/3** (decision D5: Rust vs Go benchmark still pending).

### Should-do for architecture integrity
6. Decide fate of `helix_transport` scaffolding (populate, redirect, or archive).
7. Add FFI surface (`flutter_rust_bridge` + UniFFI) to `helix_core`.
8. Implement platform tunnel shims (`helix_shims`) for at least Linux, Android, iOS, Windows.
9. Add policy/verdict-map proto and compiler tests.
10. Wire `WatchNetworkMap` delta stream into the orchestrator reconciler.

### Engineering-hygiene follow-up
11. Investigate and resolve the 2 ignored `helix-wg` encryption tests (transport key alignment).
12. Tune `quic-go` UDP buffer warning in `helix_go` tests or document as env constraint.
13. Add `buf breaking` gate against `main` once the proto is merged.

---

## Files created or modified by this deliverable

### Code submodules
- `submodules/helix_proto/buf.yaml`
- `submodules/helix_proto/buf.gen.yaml`
- `submodules/helix_proto/proto/helix/coordinator/v1/coordinator.proto`
- `submodules/helix_proto/proto/helix/session/v1/session.proto`
- `submodules/helix_proto/proto/helix/tunnel/v1/tunnel.proto`
- `submodules/helix_proto/proto/helix/ui/v1/ui.proto`
- `submodules/helix_proto/proto/helix/telemetry/v1/telemetry.proto`
- `submodules/helix_proto/gen/go/...` (generated Go stubs)
- `submodules/helix_core/README.md` (updated)
- `submodules/helix_edge/README.md` (updated)
- `submodules/helix_go/README.md` (updated)
- `submodules/helix_proto/README.md` (updated)
- `submodules/helix_transport/README.md` (updated)
- `submodules/helix_ui/README.md` (updated)
- `submodules/helix_shims/README.md` (updated)
- `submodules/helix_design/README.md` (updated)

### Docs
- `docs/reviews/mvp-final/findings/phase3-code-spec-alignment.md`
- `docs/research/mvp/final/implementation/02-system-architecture/README.md`
- `docs/research/mvp/final/implementation/02-system-architecture/decoupling-plan.md`
- `docs/research/mvp/final/implementation/08-api-contracts/README.md`
