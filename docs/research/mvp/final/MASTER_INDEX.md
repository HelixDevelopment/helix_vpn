# HelixVPN — Master Specification Index (the full document tree)

**Revision:** 5
**Last modified:** 2026-07-04T12:00:00Z
**Status:** active — the navigation + generation blueprint for the full HelixVPN technical specification
**Rev 5 (2026-07-04 hardening pass):** Independent deep gap-analysis + direct hardening executed
across **every** volume (V0–V10, ~140 nano-detail docs + the 16 top-level `final/` chapters + all 5
`04_VPN_CLD` source docs) via 16 parallel review-and-fix passes, closing the previously-owed
adversarial reviews for V0/V1/V6/V7/V8 (V2/V3/V4/V5/V10 had already passed review pre-Rev-5; this
pass re-verified them independently rather than trusting the "GO" label, and found + fixed several
real defects even in the already-reviewed volumes — see below). Real cross-document defects found
and fixed (not padding): (1) a security-relevant PKI bug — CA rotation had conflated *scheduled*
rotation with *compromise-triggered* rotation, meaning a compromised issuing CA would have stayed
trusted for up to 24h (`v03-control-plane/svc-pki.md`, now split into two distinct paths); (2) a
real Rust contradiction — the FFI boundary claimed both `catch_unwind` panic recovery *and*
workspace-wide `panic=abort`, which are mutually exclusive (`v04-client/helix-core-rust.md`, now an
honest corrected contract); (3) a missing QUIC anti-amplification limit on the public MASQUE edge
listener — a real DDoS-vector gap (`v02-data-plane/transport-masque-quic.md`); (4) a dangling
reconcile-reliability gap — no schema existed to prevent a silently-dropped reconcile trigger on a
crash between the Postgres commit and the Redis `XADD` (`v03-control-plane/data-model-ddl.md`, now
an `outbox` table + sweeper); (5) rate-limiting/DDoS resilience and a resource-sizing/cost model
were genuinely absent anywhere in the 131-file corpus — both added
(`05-repo-layout-tooling-and-helix-ecosystem.md`, `v06-deploy/ha-and-multiregion.md`); (6) R5
(Phase 1–3 WBS task/subtask tier asymmetry, tracked in `REFINEMENT_NOTES.md`) resolved — the
already-existing `v07-execution/subtask-deepening-p1/p2/p3.md` docs were orphaned (nothing in
`07/08/09-*-wbs.md` pointed to them); now cross-referenced; (7) `D-PKI-CA-TIER` and `D-OD-1` (both
noted below as open in Rev 3/4) are **operator-confirmed** per `v00-meta/decision-register.md`
Rev 2 — the Rev-3/4 notes below are superseded, kept for history; (8) a numeric contradiction
between `SPECIFICATION.md` §8.2 ("< 30s" failover) and `08-phase2-parity-wbs.md`'s measured
**P2-SLO3** ("< 3s") reconciled in `SPECIFICATION.md` Rev 2 (the WBS's tighter, instrumented figure
wins). Every touched file carries its own Revision bump recording its specific changes; consult
each file's header for the full per-doc changelog. `§11.4.168` Mermaid-pandoc-render tooling gap
(item (b) below) remains open — it is a build-tooling fix, not a markdown-content fix, and stays
out of this pass's scope.
**Rev 4:** Re-marked the generated `vNN-*` nano-detail rows `[GEN]`→`[DONE]` across Volumes 0–10 (the
expansion docs now exist on disk with synced siblings); added the `[DONE]` status to the legend. The
`[P1]` overview rows stay `[P1]`; `[RES]` preserved (e.g. `[DONE][RES]`).
**Rev 3:** All 11 volumes (V0–V10) expanded — generation COMPLETE (~140 nano-detail docs, all with
synced HTML/PDF siblings). Marked the wave order done; recorded the tracked pre-tag quality items
(owed volume reviews, §11.4.168 Mermaid-render, V9 dossiers, D-PKI-CA-TIER reversibility).
**Rev 2:** Added **Volume 10 — Design System (OpenDesign)** as a decoupled reusable submodule
(`vasic-digital/helix_design`) per operator mandate 2026-06-25 (§11.4.162); repointed the V4
design row to the client-integration view; recorded the resumed wave order.

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
- **[DONE]** generated + on disk + HTML/PDF siblings synced (an expansion-wave doc that has landed).
- **[RES]** research-backed (cited corpus in `11-deep-research-appendix.md`).

---

## Volume 0 — Spine, meta & governance
| Doc | Status | Scope |
|---|---|---|
| `SPECIFICATION.md` | [P1] | Architectural spine: roles, principles, one-screen architecture, roadmap, decision register, glossary |
| `MASTER_INDEX.md` | [P1] | **This file** — the full document tree |
| `99-source-coverage-ledger.md` | [P1] | Source-coverage proof (16 docs → where absorbed), gaps |
| `REFINEMENT_NOTES.md` | [P1] | Pass-1→pass-N punch-list |
| `v00-meta/glossary.md` | [DONE] | Full glossary (every term, acronym, protocol, RFC) |
| `v00-meta/decision-register.md` | [DONE] | D1–D8 expanded + the gate that resolves each + reversal criteria |
| `v00-meta/requirements-traceability.md` | [DONE] | Requirement → component → test cross-reference matrix |

## Volume 1 — Product & requirements
| Doc | Status | Scope |
|---|---|---|
| `00-product-scope-and-principles.md` | [P1] | Overview: product def, personas, roles, scope, principles, parity matrix, decisions |
| `v01-product/product-vision-and-positioning.md` | [DONE] | Vision, market, prior-art deep-dive, differentiators, non-goals |
| `v01-product/personas-and-roles.md` | [DONE] | Each persona: jobs-to-be-done, journeys, pain points, success |
| `v01-product/functional-requirements.md` | [DONE] | Every FR, numbered (HVPN-FR-NNN), with acceptance |
| `v01-product/nonfunctional-requirements.md` | [DONE] | Perf/scale/availability/privacy/security SLOs (HVPN-NFR-NNN) |
| `v01-product/use-cases-and-journeys.md` | [DONE] | End-to-end use cases with sequence diagrams |
| `v01-product/success-metrics.md` | [DONE] | KPIs, telemetry-derived (counts only, no flows) |

## Volume 2 — Data plane (Rust)
| Doc | Status | Scope |
|---|---|---|
| `01-data-plane.md` | [P1] | Overview: layering, transports, routing, multihop |
| `v02-data-plane/wireguard-core.md` | [DONE][RES] | boringtun vs kernel WG per platform, Noise IK, AllowedIPs, key mgmt, MTU |
| `v02-data-plane/transport-trait.md` | [DONE] | The `Transport` trait — full Rust signature, invariants, lifecycle, errors |
| `v02-data-plane/transport-plain-udp.md` | [DONE] | Plain-UDP transport: impl, MTU 1420, fast path |
| `v02-data-plane/transport-masque-quic.md` | [DONE][RES] | MASQUE CONNECT-UDP (RFC 9298/9297/9221), quinn+h3, hand-rolled framing, DPI camouflage |
| `v02-data-plane/transport-shadowsocks.md` | [DONE][RES] | Shadowsocks-wrap (AEAD-TCP), shadowsocks-rust |
| `v02-data-plane/transport-udp-over-tcp.md` | [DONE] | UoT last-resort, udp2tcp framing |
| `v02-data-plane/transport-lwo.md` | [DONE] | Lightweight obfuscation (keyed WG-header obfs + padding) |
| `v02-data-plane/transport-selection-ladder.md` | [DONE][RES] | Auto-escalation, per-network memory, regional priors, handshake-failure events |
| `v02-data-plane/obfuscation-and-dpi.md` | [DONE][RES] | DPI/censorship landscape (GFW probing, SNI, UDP block), which transport survives what |
| `v02-data-plane/routing-and-addressing.md` | [DONE] | ULA /48, 4via6 overlapping CIDR, AllowedIPs + nftables/eBPF verdict maps |
| `v02-data-plane/multihop.md` | [DONE] | Nested WireGuard, per-hop keys, entry/exit jurisdiction |
| `v02-data-plane/daita.md` | [DONE][RES] | DAITA via maybenot: packet sizing, cover traffic, timing, overhead |
| `v02-data-plane/orchestrator-and-state.md` | [DONE] | helix-core loops, tokio::broadcast status enum, reconnection state machine |

## Volume 3 — Control plane (Go)
| Doc | Status | Scope |
|---|---|---|
| `02-control-plane.md` | [P1] | Overview: modular monolith, data model, coordinator, events, API |
| `v03-control-plane/architecture-and-wiring.md` | [DONE] | Package boundaries, interface/event wiring rules, no-cross-store-import |
| `v03-control-plane/svc-identity.md` | [DONE] | Tenants/users/SSO-OIDC/anon-device-token; full API + state |
| `v03-control-plane/svc-registry.md` | [DONE] | Devices/connectors/prefixes/overlay-IP registration |
| `v03-control-plane/svc-ipam.md` | [DONE] | Overlay pool allocation, ULA /48, host allocation, 4via6 mapping |
| `v03-control-plane/svc-pki.md` | [DONE][RES] | Tenant CA, short-lived mTLS device certs, rotation, revocation, PQ material |
| `v03-control-plane/svc-policy.md` | [DONE] | ACL model, compiler algorithm → AllowedIPs + verdict maps, fail-closed |
| `v03-control-plane/svc-coordinator.md` | [DONE] | In-mem topology graph, minimal-delta computation, <1s SLO |
| `v03-control-plane/svc-events.md` | [DONE][RES] | Redis Streams, envelope, taxonomy, consumer groups, XAUTOCLAIM DLQ |
| `v03-control-plane/svc-telemetry.md` | [DONE] | Counters/health/audit (no traffic logs), Prometheus metrics |
| `v03-control-plane/svc-api.md` | [DONE][RES] | Gin REST routes, WS/SSE, Connect-RPC, authz, OpenAPI |
| `v03-control-plane/data-model-ddl.md` | [DONE] | Full Postgres DDL, per-table, RLS policies, non-superuser role, migrations |
| `v03-control-plane/protobuf-spec.md` | [DONE] | Complete `.proto` (helix.coordinator.v1), every message + field + semantics |
| `v03-control-plane/reconciliation-flow.md` | [DONE] | Event→delta→enforce end-to-end, sequence diagrams, idempotency |

## Volume 4 — Clients (Rust core + Flutter UI + shims)
| Doc | Status | Scope |
|---|---|---|
| `03-client-core-and-ui.md` | [P1] | Overview: shared-codebase strategy, FFI, UI, shims |
| `v04-client/helix-core-rust.md` | [DONE] | Crate layout, reconciler, kill-switch + DNS state machine, memory/size strategy |
| `v04-client/ffi-surface.md` | [DONE][RES] | flutter_rust_bridge v2 / UniFFI, start/stop/status_stream, TunnelStatus, threading |
| `v04-client/helix-ui-flutter.md` | [DONE][RES] | Melos monorepo, 3 flavors via runHelixApp, capability gating |
| `v04-client/design-system.md` | [DONE] | **Client-integration view only** — how `helix-ui` consumes `helix_design`; the full design system is **Volume 10** (operator mandate 2026-06-25) |
| `v04-client/state-management.md` | [DONE] | Riverpod providers, status-stream→UI pure-function, Console WS/SSE folding |
| `v04-client/shim-apple.md` | [DONE][RES] | iOS/macOS NEPacketTunnelProvider, the memory ceiling, Swift↔Rust |
| `v04-client/shim-android.md` | [DONE][RES] | VpnService + JNI, builder/protect/fd handoff, background-kill |
| `v04-client/shim-windows.md` | [DONE] | wireguard-nt/wintun + privileged service, named-pipe IPC, WFP split-tunnel |
| `v04-client/shim-linux.md` | [DONE] | kernel WG/tun, systemd integration |
| `v04-client/shim-harmonyos.md` | [DONE][RES] | OpenHarmony flutter fork, Network Kit VPN ability, ArkTS→NAPI→.so |
| `v04-client/shim-aurora.md` | [DONE][RES] | omprussia flutter, Qt/C++ tun shim, signed RPM |
| `v04-client/web-console.md` | [DONE] | Console-only build, no core_ffi, optional WASM MASQUE proxy caveat |

## Volume 5 — Security & privacy
| Doc | Status | Scope |
|---|---|---|
| `04-security-privacy-pki.md` | [P1] | Overview: zero-trust, identity, PKI, no-logging, PQ |
| `v05-security/threat-model.md` | [DONE] | STRIDE/LINDDUN, attacker classes, trust boundaries, mitigations |
| `v05-security/zero-trust-and-default-deny.md` | [DONE] | Policy enforcement points, need-to-know peer filtering |
| `v05-security/identity-and-enrollment.md` | [DONE] | OIDC + anon device tokens, device keygen, key-never-leaves |
| `v05-security/pki-and-certs.md` | [DONE][RES] | Cert lifecycle, short-lived mTLS, CA hierarchy, rotation, revocation <1s |
| `v05-security/no-logging-as-code.md` | [DONE] | CI schema-lint, ephemeral Redis presence, audit-only-control-actions |
| `v05-security/kill-switch-and-dns-leak.md` | [DONE] | Per-OS firewall state machine, DNS-forced-through-tunnel |
| `v05-security/post-quantum.md` | [DONE][RES] | ML-KEM/FIPS-203 PSK, hybrid-never-PQ-only, Rosenpass evaluation |
| `v05-security/audit-and-compliance.md` | [DONE] | audit_events, control-action audit, compliance posture |

## Volume 6 — Deployment, tooling & operations
| Doc | Status | Scope |
|---|---|---|
| `05-repo-layout-tooling-and-helix-ecosystem.md` | [P1] | Overview: repo layout, codegen, deploy, ecosystem |
| `v06-deploy/repo-layout-and-decoupling.md` | [DONE] | Monorepo + planned vasic-digital component repos, §11.4.28/.29/.74 |
| `v06-deploy/codegen-pipeline.md` | [DONE] | buf → Go/Dart/Rust, OpenAPI → Dart/TS, zero-drift |
| `v06-deploy/helixvpnctl.md` | [DONE] | Cobra CLI: init/keys/enroll-token/policy/revoke — full command spec |
| `v06-deploy/podman-quadlets.md` | [DONE][RES] | Rootless quadlet units, NET_ADMIN, :443/udp, one-pod, read-only rootfs |
| `v06-deploy/docker-compose.md` | [DONE][RES] | Equivalent Docker Compose stack |
| `v06-deploy/kubernetes.md` | [DONE][RES] | Deployment/StatefulSet/Service/NetworkPolicy manifests |
| `v06-deploy/ha-and-multiregion.md` | [DONE] | Stateless coordinators, Patroni PG, NATS JetStream, anycast/geoDNS |
| `v06-deploy/observability.md` | [DONE] | Prometheus/Grafana-as-code, convergence + event-lag SLOs |
| `v06-deploy/disaster-recovery.md` | [DONE] | RTO/RPO budget, backup, region-failover runbook (closes ledger gap G1) |
| `v06-deploy/helix-ecosystem-integration.md` | [DONE] | containers/helix_qa/challenges/docs_chain/security/vision_engine wiring |
| `v06-deploy/remote-testing-infra.md` | [DONE] | nezha.local heavy-testing node, containers-submodule distribution (PARKED until greenlit) |

## Volume 7 — Phase execution (work breakdown → tasks → subtasks)
| Doc | Status | Scope |
|---|---|---|
| `06-phase0-spike-wbs.md` | [P1] | Phase 0 WBS (HVPN-P0-NNN), gates G1–G6, milestones S0–S8 |
| `07-phase1-mvp-wbs.md` | [P1] | Phase 1 MVP WBS (HVPN-P1-NNN), DoD + SLOs |
| `08-phase2-parity-wbs.md` | [P1] | Phase 2 WBS (HVPN-P2-NNN) |
| `09-phase3-reach-wbs.md` | [P1] | Phase 3 WBS (HVPN-P3-NNN) |
| `v07-execution/workable-items-model.md` | [DONE] | §11.4.93 SQLite mapping: every task/subtask → DB row schema + docs_chain sync |
| `v07-execution/dependency-graph.md` | [DONE] | Cross-phase dependency DAG, critical path |
| `v07-execution/subtask-deepening-p1.md` | [DONE] | Phase 1 epic→task→subtask deepening (closes R5) |
| `v07-execution/subtask-deepening-p2.md` | [DONE] | Phase 2 subtask deepening |
| `v07-execution/subtask-deepening-p3.md` | [DONE] | Phase 3 subtask deepening |

## Volume 8 — Testing & QA (per §11.4.169 mandatory test types)
| Doc | Status | Scope |
|---|---|---|
| `10-testing-acceptance-and-qa.md` | [P1] | Overview: all test types, helix_qa/challenges, acceptance gates |
| `v08-testing/unit.md` … `v08-testing/benchmarking.md` | [DONE] | One deep doc per §11.4.169 type: unit, integration, e2e, full-automation, challenges, helixqa, ddos, security, stress-chaos, concurrency, race-deadlock, memory, benchmarking — harness, fixtures, evidence, acceptance |
| `v08-testing/coverage-ledger-schema.md` | [DONE] | feature × test-type × evidence-state ledger |
| `v08-testing/test-rig.md` | [DONE][RES] | netns + nftables-DPI + tc-netem rig, iperf3 bars, leak tests |

## Volume 9 — Research appendix (cited)
| Doc | Status | Scope |
|---|---|---|
| `11-deep-research-appendix.md` | [P1][RES] | Consolidated cited research (10 angles); being rewritten from the corpus |
| `v09-research/research-<angle>.md` ×10 | [DONE][RES] | Per-angle full research dossier (wireguard, masque, hysteria2, mullvad, flutter_ffi, ios_android, go_cp, podman_k8s, pki_pq_nat, daita_test) |

## Volume 10 — Design System (OpenDesign) — `vasic-digital/helix_design` (decoupled reusable submodule)
> **Operator mandate (2026-06-25, §11.4.162).** OpenDesign is the **mandatory** design-and-refinement
> system for **every** HelixVPN user-facing application (Client, Console, Connector) on **all 8
> platforms**. The design system is a **fully decoupled, reusable submodule** (`vasic-digital/helix_design`,
> snake_case flat per §11.4.28/.29/.74) incorporable by any future app. This volume documents — in
> depth — the whole design system: every screen, every reusable component, light+dark color schemes,
> typography, and **all** design/UI/UX resources emitted in **every consumable form** (JSON / CSS /
> Dart / Swift / Kotlin / ArkTS / C-Qt) for **direct** incorporation. Every component ships light+dark
> variants; elements MUST NOT overlap/overlay labels; all UI changes carry visual-regression coverage
> (§11.4.162). NO ad-hoc CSS / one-off tools — OpenDesign tokens/themes only; missing patterns extend
> OpenDesign upstream per §11.4.74.

| Doc | Status | Scope |
|---|---|---|
| `v10-design/00-overview-and-submodule.md` | [DONE] | `helix_design` as a decoupled submodule: purpose, repo layout, semver, consumption model, how ANY app incorporates it (§11.4.28/.74/.162), upstreams/install_upstreams, decoupling invariants (no project-specific context) |
| `v10-design/opendesign-foundation.md` | [DONE][RES] | OpenDesign integration: install, theme/token engine, how it drives palette/type/spacing/component tokens; extend-upstream policy (§11.4.74); honest boundary vs functional/a11y testing (§11.4.162) |
| `v10-design/design-tokens.md` | [DONE] | Canonical token taxonomy + source-of-truth schema: color/type/spacing/radius/elevation/motion/z-index/breakpoints; naming, tiers (primitive→semantic→component), theming model |
| `v10-design/color-system.md` | [DONE] | Full brand palette **light+dark**, connection-state palette (status colors), semantic mapping, WCAG contrast proofs, no-overlap/no-overlay rule (§11.4.162) |
| `v10-design/typography-iconography-motion.md` | [DONE] | Type ramp + fonts (per-platform), icon set + asset forms, motion/animation tokens & curves |
| `v10-design/token-export-pipeline.md` | [DONE][RES] | Emit tokens in **all forms** — JSON (Style-Dictionary-class) → CSS vars, Dart `ThemeData`, SwiftUI, Compose, ArkTS, C/Qt; build pipeline, drift gate, §11.4.65/.168 export + visual validation |
| `v10-design/component-library.md` | [DONE] | **Every reusable component** (ConnectButton, StatusChip, ExitPicker, ShieldToggle, ServerList, PeerCard, TopologyGraph, dialogs, forms, nav) — anatomy, states, variants, tokens, a11y, light+dark |
| `v10-design/screens-client.md` | [DONE] | **All Client app screens** (onboarding, connect/home, exits, multihop, shields, account, settings, errors) — full specs + wireframes, responsive phone/tablet/desktop/TV |
| `v10-design/screens-console.md` | [DONE] | **All admin Console screens** (devices, policy editor, users/SSO, audit, topology, billing) — full specs + wireframes |
| `v10-design/screens-connector.md` | [DONE] | **All Connector appliance screens** (enroll, advertised routes, conflicts, health) — full specs |
| `v10-design/ux-flows-and-interaction.md` | [DONE] | End-to-end UX flows, navigation/IA, interaction patterns, motion, loading/empty/error states, i18n, full accessibility spec |
| `v10-design/platform-adaptation.md` | [DONE][RES] | Per-platform UI adaptation (Material/Cupertino/desktop/TV-leanback/HarmonyOS/Aurora) + capability gating, OpenDesign theme mapping per platform |
| `v10-design/visual-regression-and-qa.md` | [DONE] | §11.4.162 visual-regression suite, golden screenshots, token-drift tests, a11y/contrast assertions, §11.4.168 exported-doc visual validation |
| `v10-design/assets-and-deliverables.md` | [DONE] | **All design/UI/UX deliverables in all forms** ready for direct incorporation: token bundles, per-platform theme packages, icon/illustration asset forms, component packages, design-source — + the per-app pull recipe |

---

## Generation plan (autonomous, batched, rate-limit-aware)
Documents are generated in **expansion waves** of 3–4 parallel subagents (§11.4.103), each
reading the volume's pass-1 overview + `v09-research/_SYNTHESIS.md` + the relevant `research-<angle>.md`,
producing one nano-detail document with concrete interfaces/DDL/protobuf/skeletons + Mermaid
diagrams, citing sources, marking unproven facts `UNVERIFIED`. Each wave is committed +
pushed; an adversarial review gates each volume. Wave order (resumed 2026-06-25): V2 data-plane ✅
→ V3 control-plane ✅ → V4 clients ✅ → V10 design-system ✅ → V5 security ✅ (reviewed+reconciled→GO)
→ V6 deploy ✅ → V1 product ✅ → V7 execution ✅ → V8 testing ✅ → V0 meta ✅.

**Generation COMPLETE (2026-06-26): all 11 volumes (V0–V10) expanded** — ~140 nano-detail docs,
every `.md` carrying synced `.html`+`.pdf` siblings (§11.4.65, via `scripts/testing/sync_all_markdown_exports.sh`).

**Outstanding quality items — updated 2026-07-04 (Rev 5 hardening pass):**
- (a) **RESOLVED.** Adversarial reviews for V1/V6/V7/V8/V0 completed 2026-07-04 (independent
  gap-analysis + direct hardening, not a rubber-stamp — see Rev 5 changelog above for the specific
  defects found and fixed). All 11 volumes (V0–V10) are now independently reviewed at least once.
- (b) **STILL OPEN.** §11.4.168 — Mermaid fences render as raw source in HTML/PDF (needs a pandoc
  mermaid filter). This is a build/export-tooling gap, not a markdown-content gap, and is out of
  scope for a documentation hardening pass — tracked for whoever owns the `docs_chain`/pandoc
  export pipeline.
- (c) **RESOLVED (was already stale by 2026-07-04).** All 10 `v09-research/research-<angle>.md`
  per-angle dossiers exist on disk (confirmed by direct inspection during this pass), each with a
  `Revision` header (9 of 10 were missing the header and got it added in this pass) — this item's
  premise ("only wireguard split out") no longer matched the actual repository state.
- (d) **RESOLVED.** D-PKI-CA-TIER (two-tier issuing CA as MVP) is **operator-confirmed** per
  `v00-meta/decision-register.md` §4 (confirmed 2026-06-26, one day after this line was written —
  this Outstanding-items list had simply not been updated since). No veto exercised; proceed with
  the two-tier design as binding for MVP.
- (e) **NEW, tracked for a future pass.** `v03-control-plane/svc-coordinator.md` §10.1 (added in
  this hardening pass) surfaces a genuinely open design question for Phase-2 multi-replica
  coordinators: fan-out vs. graph-apply consumption semantics for the shared event log. Not
  resolved here — correctly left open per the no-guessing discipline (needs a Phase-2 spike, not a
  documentation-pass decision).
