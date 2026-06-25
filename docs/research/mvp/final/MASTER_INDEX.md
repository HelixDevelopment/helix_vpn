# HelixVPN — Master Specification Index (the full document tree)

**Revision:** 1
**Last modified:** 2026-06-25T00:00:00Z
**Status:** active — the navigation + generation blueprint for the full HelixVPN technical specification

> This index decomposes the HelixVPN specification into **volumes** of **nano-detail
> documents** (dozens of documents, thousands of pages target). `SPECIFICATION.md` is the
> architectural spine; this index is the complete map. Pass-1 docs (the `NN-*.md` topical
> set) are the **volume overviews**; each volume's `vNN-<area>/` subdirectory holds the
> deep per-topic documents generated in expansion waves. No false results, no bluff: every
> deep doc cites sources `[04_ARCH §N]`/`[research-<angle>]` and marks anything unproven
> `UNVERIFIED` (§11.4.6).

## Status legend
- **[P1]** authored in pass 1 (exists) — serves as the volume overview, to be deepened.
- **[GEN]** to be generated in an expansion wave (nano-detail).
- **[RES]** research-backed (cited corpus in `11-deep-research-appendix.md`).

---

## Volume 0 — Spine, meta & governance
| Doc | Status | Scope |
|---|---|---|
| `SPECIFICATION.md` | [P1] | Architectural spine: roles, principles, one-screen architecture, roadmap, decision register, glossary |
| `MASTER_INDEX.md` | [P1] | **This file** — the full document tree |
| `99-source-coverage-ledger.md` | [P1] | Source-coverage proof (16 docs → where absorbed), gaps |
| `REFINEMENT_NOTES.md` | [P1] | Pass-1→pass-N punch-list |
| `v00-meta/glossary.md` | [GEN] | Full glossary (every term, acronym, protocol, RFC) |
| `v00-meta/decision-register.md` | [GEN] | D1–D8 expanded + the gate that resolves each + reversal criteria |
| `v00-meta/requirements-traceability.md` | [GEN] | Requirement → component → test cross-reference matrix |

## Volume 1 — Product & requirements
| Doc | Status | Scope |
|---|---|---|
| `00-product-scope-and-principles.md` | [P1] | Overview: product def, personas, roles, scope, principles, parity matrix, decisions |
| `v01-product/product-vision-and-positioning.md` | [GEN] | Vision, market, prior-art deep-dive, differentiators, non-goals |
| `v01-product/personas-and-roles.md` | [GEN] | Each persona: jobs-to-be-done, journeys, pain points, success |
| `v01-product/functional-requirements.md` | [GEN] | Every FR, numbered (HVPN-FR-NNN), with acceptance |
| `v01-product/nonfunctional-requirements.md` | [GEN] | Perf/scale/availability/privacy/security SLOs (HVPN-NFR-NNN) |
| `v01-product/use-cases-and-journeys.md` | [GEN] | End-to-end use cases with sequence diagrams |
| `v01-product/success-metrics.md` | [GEN] | KPIs, telemetry-derived (counts only, no flows) |

## Volume 2 — Data plane (Rust)
| Doc | Status | Scope |
|---|---|---|
| `01-data-plane.md` | [P1] | Overview: layering, transports, routing, multihop |
| `v02-data-plane/wireguard-core.md` | [GEN][RES] | boringtun vs kernel WG per platform, Noise IK, AllowedIPs, key mgmt, MTU |
| `v02-data-plane/transport-trait.md` | [GEN] | The `Transport` trait — full Rust signature, invariants, lifecycle, errors |
| `v02-data-plane/transport-plain-udp.md` | [GEN] | Plain-UDP transport: impl, MTU 1420, fast path |
| `v02-data-plane/transport-masque-quic.md` | [GEN][RES] | MASQUE CONNECT-UDP (RFC 9298/9297/9221), quinn+h3, hand-rolled framing, DPI camouflage |
| `v02-data-plane/transport-shadowsocks.md` | [GEN][RES] | Shadowsocks-wrap (AEAD-TCP), shadowsocks-rust |
| `v02-data-plane/transport-udp-over-tcp.md` | [GEN] | UoT last-resort, udp2tcp framing |
| `v02-data-plane/transport-lwo.md` | [GEN] | Lightweight obfuscation (keyed WG-header obfs + padding) |
| `v02-data-plane/transport-selection-ladder.md` | [GEN][RES] | Auto-escalation, per-network memory, regional priors, handshake-failure events |
| `v02-data-plane/obfuscation-and-dpi.md` | [GEN][RES] | DPI/censorship landscape (GFW probing, SNI, UDP block), which transport survives what |
| `v02-data-plane/routing-and-addressing.md` | [GEN] | ULA /48, 4via6 overlapping CIDR, AllowedIPs + nftables/eBPF verdict maps |
| `v02-data-plane/multihop.md` | [GEN] | Nested WireGuard, per-hop keys, entry/exit jurisdiction |
| `v02-data-plane/daita.md` | [GEN][RES] | DAITA via maybenot: packet sizing, cover traffic, timing, overhead |
| `v02-data-plane/orchestrator-and-state.md` | [GEN] | helix-core loops, tokio::broadcast status enum, reconnection state machine |

## Volume 3 — Control plane (Go)
| Doc | Status | Scope |
|---|---|---|
| `02-control-plane.md` | [P1] | Overview: modular monolith, data model, coordinator, events, API |
| `v03-control-plane/architecture-and-wiring.md` | [GEN] | Package boundaries, interface/event wiring rules, no-cross-store-import |
| `v03-control-plane/svc-identity.md` | [GEN] | Tenants/users/SSO-OIDC/anon-device-token; full API + state |
| `v03-control-plane/svc-registry.md` | [GEN] | Devices/connectors/prefixes/overlay-IP registration |
| `v03-control-plane/svc-ipam.md` | [GEN] | Overlay pool allocation, ULA /48, host allocation, 4via6 mapping |
| `v03-control-plane/svc-pki.md` | [GEN][RES] | Tenant CA, short-lived mTLS device certs, rotation, revocation, PQ material |
| `v03-control-plane/svc-policy.md` | [GEN] | ACL model, compiler algorithm → AllowedIPs + verdict maps, fail-closed |
| `v03-control-plane/svc-coordinator.md` | [GEN] | In-mem topology graph, minimal-delta computation, <1s SLO |
| `v03-control-plane/svc-events.md` | [GEN][RES] | Redis Streams, envelope, taxonomy, consumer groups, XAUTOCLAIM DLQ |
| `v03-control-plane/svc-telemetry.md` | [GEN] | Counters/health/audit (no traffic logs), Prometheus metrics |
| `v03-control-plane/svc-api.md` | [GEN][RES] | Gin REST routes, WS/SSE, Connect-RPC, authz, OpenAPI |
| `v03-control-plane/data-model-ddl.md` | [GEN] | Full Postgres DDL, per-table, RLS policies, non-superuser role, migrations |
| `v03-control-plane/protobuf-spec.md` | [GEN] | Complete `.proto` (helix.coordinator.v1), every message + field + semantics |
| `v03-control-plane/reconciliation-flow.md` | [GEN] | Event→delta→enforce end-to-end, sequence diagrams, idempotency |

## Volume 4 — Clients (Rust core + Flutter UI + shims)
| Doc | Status | Scope |
|---|---|---|
| `03-client-core-and-ui.md` | [P1] | Overview: shared-codebase strategy, FFI, UI, shims |
| `v04-client/helix-core-rust.md` | [GEN] | Crate layout, reconciler, kill-switch + DNS state machine, memory/size strategy |
| `v04-client/ffi-surface.md` | [GEN][RES] | flutter_rust_bridge v2 / UniFFI, start/stop/status_stream, TunnelStatus, threading |
| `v04-client/helix-ui-flutter.md` | [GEN][RES] | Melos monorepo, 3 flavors via runHelixApp, capability gating |
| `v04-client/design-system.md` | [GEN] | helix_design: tokens, connection-state palette, signature components, OpenDesign §11.4.162 |
| `v04-client/state-management.md` | [GEN] | Riverpod providers, status-stream→UI pure-function, Console WS/SSE folding |
| `v04-client/shim-apple.md` | [GEN][RES] | iOS/macOS NEPacketTunnelProvider, the memory ceiling, Swift↔Rust |
| `v04-client/shim-android.md` | [GEN][RES] | VpnService + JNI, builder/protect/fd handoff, background-kill |
| `v04-client/shim-windows.md` | [GEN] | wireguard-nt/wintun + privileged service, named-pipe IPC, WFP split-tunnel |
| `v04-client/shim-linux.md` | [GEN] | kernel WG/tun, systemd integration |
| `v04-client/shim-harmonyos.md` | [GEN][RES] | OpenHarmony flutter fork, Network Kit VPN ability, ArkTS→NAPI→.so |
| `v04-client/shim-aurora.md` | [GEN][RES] | omprussia flutter, Qt/C++ tun shim, signed RPM |
| `v04-client/web-console.md` | [GEN] | Console-only build, no core_ffi, optional WASM MASQUE proxy caveat |

## Volume 5 — Security & privacy
| Doc | Status | Scope |
|---|---|---|
| `04-security-privacy-pki.md` | [P1] | Overview: zero-trust, identity, PKI, no-logging, PQ |
| `v05-security/threat-model.md` | [GEN] | STRIDE/LINDDUN, attacker classes, trust boundaries, mitigations |
| `v05-security/zero-trust-and-default-deny.md` | [GEN] | Policy enforcement points, need-to-know peer filtering |
| `v05-security/identity-and-enrollment.md` | [GEN] | OIDC + anon device tokens, device keygen, key-never-leaves |
| `v05-security/pki-and-certs.md` | [GEN][RES] | Cert lifecycle, short-lived mTLS, CA hierarchy, rotation, revocation <1s |
| `v05-security/no-logging-as-code.md` | [GEN] | CI schema-lint, ephemeral Redis presence, audit-only-control-actions |
| `v05-security/kill-switch-and-dns-leak.md` | [GEN] | Per-OS firewall state machine, DNS-forced-through-tunnel |
| `v05-security/post-quantum.md` | [GEN][RES] | ML-KEM/FIPS-203 PSK, hybrid-never-PQ-only, Rosenpass evaluation |
| `v05-security/audit-and-compliance.md` | [GEN] | audit_events, control-action audit, compliance posture |

## Volume 6 — Deployment, tooling & operations
| Doc | Status | Scope |
|---|---|---|
| `05-repo-layout-tooling-and-helix-ecosystem.md` | [P1] | Overview: repo layout, codegen, deploy, ecosystem |
| `v06-deploy/repo-layout-and-decoupling.md` | [GEN] | Monorepo + planned vasic-digital component repos, §11.4.28/.29/.74 |
| `v06-deploy/codegen-pipeline.md` | [GEN] | buf → Go/Dart/Rust, OpenAPI → Dart/TS, zero-drift |
| `v06-deploy/helixvpnctl.md` | [GEN] | Cobra CLI: init/keys/enroll-token/policy/revoke — full command spec |
| `v06-deploy/podman-quadlets.md` | [GEN][RES] | Rootless quadlet units, NET_ADMIN, :443/udp, one-pod, read-only rootfs |
| `v06-deploy/docker-compose.md` | [GEN][RES] | Equivalent Docker Compose stack |
| `v06-deploy/kubernetes.md` | [GEN][RES] | Deployment/StatefulSet/Service/NetworkPolicy manifests |
| `v06-deploy/ha-and-multiregion.md` | [GEN] | Stateless coordinators, Patroni PG, NATS JetStream, anycast/geoDNS |
| `v06-deploy/observability.md` | [GEN] | Prometheus/Grafana-as-code, convergence + event-lag SLOs |
| `v06-deploy/disaster-recovery.md` | [GEN] | RTO/RPO budget, backup, region-failover runbook (closes ledger gap G1) |
| `v06-deploy/helix-ecosystem-integration.md` | [GEN] | containers/helix_qa/challenges/docs_chain/security/vision_engine wiring |
| `v06-deploy/remote-testing-infra.md` | [GEN] | nezha.local heavy-testing node, containers-submodule distribution (PARKED until greenlit) |

## Volume 7 — Phase execution (work breakdown → tasks → subtasks)
| Doc | Status | Scope |
|---|---|---|
| `06-phase0-spike-wbs.md` | [P1] | Phase 0 WBS (HVPN-P0-NNN), gates G1–G6, milestones S0–S8 |
| `07-phase1-mvp-wbs.md` | [P1] | Phase 1 MVP WBS (HVPN-P1-NNN), DoD + SLOs |
| `08-phase2-parity-wbs.md` | [P1] | Phase 2 WBS (HVPN-P2-NNN) |
| `09-phase3-reach-wbs.md` | [P1] | Phase 3 WBS (HVPN-P3-NNN) |
| `v07-execution/workable-items-model.md` | [GEN] | §11.4.93 SQLite mapping: every task/subtask → DB row schema + docs_chain sync |
| `v07-execution/dependency-graph.md` | [GEN] | Cross-phase dependency DAG, critical path |
| `v07-execution/subtask-deepening-p1.md` | [GEN] | Phase 1 epic→task→subtask deepening (closes R5) |
| `v07-execution/subtask-deepening-p2.md` | [GEN] | Phase 2 subtask deepening |
| `v07-execution/subtask-deepening-p3.md` | [GEN] | Phase 3 subtask deepening |

## Volume 8 — Testing & QA (per §11.4.169 mandatory test types)
| Doc | Status | Scope |
|---|---|---|
| `10-testing-acceptance-and-qa.md` | [P1] | Overview: all test types, helix_qa/challenges, acceptance gates |
| `v08-testing/unit.md` … `v08-testing/benchmarking.md` | [GEN] | One deep doc per §11.4.169 type: unit, integration, e2e, full-automation, challenges, helixqa, ddos, security, stress-chaos, concurrency, race-deadlock, memory, benchmarking — harness, fixtures, evidence, acceptance |
| `v08-testing/coverage-ledger-schema.md` | [GEN] | feature × test-type × evidence-state ledger |
| `v08-testing/test-rig.md` | [GEN][RES] | netns + nftables-DPI + tc-netem rig, iperf3 bars, leak tests |

## Volume 9 — Research appendix (cited)
| Doc | Status | Scope |
|---|---|---|
| `11-deep-research-appendix.md` | [P1][RES] | Consolidated cited research (10 angles); being rewritten from the corpus |
| `v09-research/research-<angle>.md` ×10 | [GEN][RES] | Per-angle full research dossier (wireguard, masque, hysteria2, mullvad, flutter_ffi, ios_android, go_cp, podman_k8s, pki_pq_nat, daita_test) |

---

## Generation plan (autonomous, batched, rate-limit-aware)
Documents are generated in **expansion waves** of 3–4 parallel subagents (§11.4.103), each
reading the volume's pass-1 overview + `kb/SYNTHESIS.md` + the relevant `research-<angle>.md`,
producing one nano-detail document with concrete interfaces/DDL/protobuf/skeletons + Mermaid
diagrams, citing sources, marking unproven facts `UNVERIFIED`. Each wave is committed +
pushed; an adversarial review gates each volume. Wave order: V2 data-plane → V3 control-plane →
V4 clients → V5 security → V6 deploy → V1 product → V7 execution → V8 testing → V0/V9 meta.
