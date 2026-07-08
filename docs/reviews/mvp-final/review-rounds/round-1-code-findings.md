# Round 1 — Adversarial Code/Spec Review

**Scope:** `helix_core`, `helix_edge`, `helix_go`, `helix_proto`, `helix_transport`, `helix_shims`, `helix_ui`, `helix_design`; system architecture and API-contract docs; Phase 3 alignment report.  
**Date:** 2026-07-05  
**Reviewer:** independent adversarial reviewer (subagent)  
**Verdict:** **GO-with-conditions**  

---

## Executive verdict

The MVP package is **not ready for an unconditional release**, but it is **honest, testable, and architecturally coherent**. The protobuf contracts are complete and lint-clean, the Rust Phase-0 primitives build and pass tests, and the documentation accurately discloses what is missing. The blockers are integration and completeness gaps, not correctness lies or hidden regressions.

| Area | Status |
|---|---|
| Protobuf schema | Lint passes; covers all required domains |
| Generated Go stubs | Exist, but **not a consumable Go module** |
| Rust core (`helix_core`) | **119 passed, 2 ignored** — real WG/MASQUE/TUN primitives |
| Rust edge (`helix_edge`) | **11 passed** — decoy + MASQUE relay integration proofs |
| Go control plane (`helix_go`) | Only `pkg/masqueedge` spike exists; **no production control plane** |
| README honesty | Good; scaffolding submodules are explicitly marked |
| Decoupling plan | Concrete module boundaries and dependency-inversion measures documented |
| Hostname/project leaks (READMEs) | No hardcoded service hostnames or tenant assumptions found |

---

## Commands run and evidence

### 1. Protobuf lint

```bash
cd submodules/helix_proto && buf lint
```

Result: `buf` is installed at `/home/milosvasic/go/bin/buf`; lint completed with exit code 0 and no warnings.

### 2. Generated Go stubs

```bash
find submodules/helix_proto/gen/go -type f -name '*.go' | head
```

Confirmed generated files exist:

- `gen/go/helix/coordinator/v1/coordinator.pb.go`
- `gen/go/helix/coordinator/v1/coordinatorv1connect/coordinator.connect.go`
- `gen/go/helix/session/v1/session.pb.go`
- `gen/go/helix/session/v1/sessionv1connect/session.connect.go`
- `gen/go/helix/tunnel/v1/tunnel.pb.go`
- `gen/go/helix/telemetry/v1/telemetry.pb.go`
- `gen/go/helix/telemetry/v1/telemetryv1connect/telemetry.connect.go`
- `gen/go/helix/ui/v1/ui.pb.go`

`buf generate` also re-ran cleanly (exit 0).

### 3. Rust test suites

`helix_core`:

```bash
cd submodules/helix_core && cargo test --all-targets
```

Aggregated result: **119 passed, 0 failed, 2 ignored**.

Breakdown per test target:

- `helix-transport` unittests: 12 passed
- `g1_integration.rs`: 3 passed
- `helix-tun` unittests: 5 passed
- `helix-wg` unittests: 34 passed, 2 ignored (`test_encrypt_decrypt_bidirectional`, `test_encrypt_decrypt_roundtrip` — labeled `HVPN-P0-011 need boringtun transport key alignment`)
- Additional workspace targets: 65 passed across the remaining crates/integration tests

`helix_edge`:

```bash
cd submodules/helix_edge && cargo test --all-targets
```

Aggregated result: **11 passed, 0 failed, 0 ignored**.

Breakdown:

- `decoy::tests::decoy_body_is_html_and_non_revealing`: 1 passed
- `gateway::tests::gateway_error_converts_from_io_and_transport`: 1 passed
- `decoy_probe_serves_believable_page.rs`: 2 passed
- `dependency_resolution_smoke.rs`: 4 passed
- `edge_decoy_and_masque_flow_coexist.rs`: 1 passed
- `masque_gateway_relay_byte_identical.rs`: 1 passed
- `wg_handshake_through_masque_and_gateway.rs`: 1 passed

### 4. Go MASQUE edge spike

```bash
cd submodules/helix_go && go test ./pkg/masqueedge/... -count=1 -race
```

Result: **ok `github.com/vasic-digital/helix_go/pkg/masqueedge` 5.505s**.

`go build ./...` in `helix_go` also succeeds.

### 5. README review

All eight `helix_*/README.md` files were inspected. Findings:

- Each README contains an explicit, honest **MVP status** section listing what is implemented and what is not.
- Scaffolding submodules (`helix_transport`, `helix_shims`, `helix_ui`, `helix_design`) are clearly labeled as scaffolding only.
- No hardcoded **service hostnames**, **tenant assumptions**, or **asset paths** were found. The only hostname-like strings are:
  - `example.com` and `www.example.com` used in unit/integration tests as RFC 2606 placeholder hostnames (acceptable).
  - `github.com/vasic-digital/helix_go` in `go.mod` and imports, which is the Go module path, not a runtime hostname.
- The decoy HTML in `helix_edge/src/decoy.rs` is deliberately bland and contains no VPN/MASQUE/WireGuard/tunnel vocabulary; this is enforced by a unit test.

### 6. Decoupling plan

`docs/research/mvp/final/implementation/02-system-architecture/decoupling-plan.md` names concrete module boundaries and dependency-inversion measures:

- `helix-transport` / `Connection` / `Listener` trait boundary
- `PacketIO` trait for platform TUN shims
- `PkiService` interface for CA-backend abstraction
- Per-crate public interfaces and hidden implementations
- Explicit coupling-risk table with severity and mitigations

### 7. Protobuf contract coverage

All required contract areas are present in `submodules/helix_proto/proto/helix/`:

| Required area | Proto package | Evidence |
|---|---|---|
| Enrollment | `helix.coordinator.v1` | `Coordinator.Enroll`, `EnrollRequest`, `EnrollResponse`, `DeviceKind` |
| Network map | `helix.coordinator.v1` | `Coordinator.WatchNetworkMap`, `NetworkMap`, `MapDelta`, `Peer`, `Via6Route`, `GatewayInfo` |
| Control plane | `helix.coordinator.v1` + `helix.session.v1` | `Coordinator` service, `Session` service (auth/validate/revoke) |
| Data plane | `helix.tunnel.v1` | `TunnelState`, `TransportKind`, `TunnelEvent`, `TunnelCommand`, `TransportHealth` |
| Telemetry | `helix.telemetry.v1` | `Telemetry.SubmitMetrics`, `Telemetry.SubmitLog`, `MetricsBatch`, `LogEntry`, `EdgeHealthReport` |
| UI state | `helix.ui.v1` | `UiState`, `ConnectionCard`, `NetworkList`, `Notification` |

---

## Top 3 findings

### 1. Generated Go stubs are not a consumable Go module (HIGH)

**Evidence:**

- The `go_package` option in every `.proto` file uses `github.com/vasic-digital/helix-go/...` (**hyphen**).
- The actual Go control-plane repository is `github.com/vasic-digital/helix_go` (**underscore**) per `submodules/helix_go/go.mod`.
- `submodules/helix_proto` has **no `go.mod`**, so the generated files in `gen/go/` are not a standalone, importable Go module.
- Cross-imports inside the generated stubs reference the hyphen path, e.g. `ui.pb.go` imports `github.com/vasic-digital/helix-go/gen/helix/tunnel/v1`.

**Impact:** The schema-first promise is blocked. The control plane cannot import these stubs without either (a) adding a `go.mod` to `helix_proto` with the hyphen module path, or (b) aligning `go_package` to the underscore path and embedding the stubs in `helix_go`.

**Condition before GO:** Decide the canonical Go module path, align all `go_package` options, and make the generated stubs `go build`-able from a real module.

### 2. Control plane is effectively absent despite complete proto contracts (HIGH)

**Evidence:**

- `helix_go` contains only `pkg/masqueedge` (an RFC 9298 CONNECT-UDP edge spike) and `cmd/go-edge`.
- No implementation exists for: identity/enrollment, device/tenant registry, IPAM, PKI, policy compiler, coordinator (`Coordinator.WatchNetworkMap` server), Redis Streams event bus, telemetry collector, Gin REST/WebSocket/SSE gateway, or `helixvpnctl`.
- The `pkg/masqueedge` tests pass, but the package is explicitly labeled a Phase-0 spike, not the production edge.

**Impact:** The MVP cannot enroll devices, stream network maps, allocate overlay IPs, or enforce policy. The protobuf contracts are sound, but they are currently "specs without a server."

**Condition before GO:** Implement the Go control-plane services behind the trait boundaries already documented in the decoupling plan, starting with `Coordinator` (enrollment + `WatchNetworkMap`) and the registry/IPAM/PKI foundations.

### 3. `Transport` trait shape mismatches the frozen spec (MEDIUM-HIGH)

**Evidence:**

- The frozen spec trait expects `send`/`recv`/`kind`/`effective_mtu`/`health`/`close` methods on `Transport` itself, plus a free `dial(TransportConfig)` function.
- The current code in `helix_core/crates/helix-transport/src/lib.rs` defines a `Transport` trait with `dial`/`listen`/`name`, and a separate `Connection` trait with `send`/`recv`/`close`/`local_addr`/`peer_addr`.
- The decoupling plan §7 explicitly flags this as a **High** severity risk and states the plan is to converge on the frozen spec trait.

**Impact:** External consumers of `helix-transport` will face a breaking change when the trait is reconciled. The auto-ladder, MTU contracts, and transport-health telemetry cannot be fully honored until the shapes match.

**Condition before GO:** Reconcile the trait to the frozen spec, migrate all current implementations and tests, and update the decoupling plan's risk table to reflect closure.

---

## Additional findings

| # | Finding | Severity | Notes |
|---|---|---|---|
| 4 | `helix_transport`, `helix_ui`, `helix_shims`, `helix_design` are scaffolding only | MEDIUM | Honest in READMEs, but four declared submodules contain no source or tests. |
| 5 | Rust MASQUE CONNECT-UDP flow is a labeled stand-in, not RFC 9298 | MEDIUM | Documented honestly; `helix_go/pkg/masqueedge` is the only RFC 9298 implementation today. |
| 6 | Dart and Rust proto stubs not generated | MEDIUM | `buf.gen.yaml` has plugins commented out; generators not installed in this environment. |
| 7 | No `buf breaking` CI gate yet | LOW | Documented as pending in README; needed to protect frozen `v1` field numbers. |
| 8 | No end-to-end TUN ↔ WG ↔ transport packet pump | MEDIUM | WG handshake and MASQUE relay are proven, but the data path is not closed. |

---

## Conditions for unconditional GO

1. Fix the canonical Go module path for generated stubs and make them importable/compilable.
2. Implement the Go control-plane MVP services (`identity`, `registry`, `ipam`, `pki`, `policy`, `coordinator`) behind the documented trait boundaries.
3. Reconcile the Rust `Transport` trait with the frozen spec and migrate all consumers.

---

## Files touched

- `docs/reviews/mvp-final/review-rounds/round-1-code-findings.md` (created)

No source files were modified. No commits or pushes were performed.
