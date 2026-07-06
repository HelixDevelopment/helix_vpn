# 99 — Source Coverage Ledger

**Revision:** 1
**Last modified:** 2026-07-05T14:20:00Z
**Status:** Draft — consolidated from `99-source-coverage-ledger.md`.

---

## 1. Position

This section is the §11.4.118 proof-of-completeness for the MVP documentation consolidation. It maps every canonical source document in `docs/research/mvp/final/` to the implementation section that carries its distinctive content, and it surfaces any source ideas that are not yet fully reflected.

The authoritative version lives at [`../../99-source-coverage-ledger.md`](../../99-source-coverage-ledger.md); this README is the navigable consolidation.

---

## 2. Source → implementation mapping

| Source | Implementation section(s) | Distinctive contribution |
|---|---|---|
| `SPECIFICATION.md` | [00 — Executive Summary](../00-executive-summary/README.md), [02 — System Architecture](../02-system-architecture/README.md) | Architectural spine, roles, principles, roadmap, decision register |
| `00-product-scope-and-principles.md` | [01 — Product Scope](../01-product-scope/README.md) | Product definition, personas, scope, parity matrix, decisions D1–D8 |
| `01-data-plane.md` | [03 — Data Plane](../03-data-plane/README.md) | `Transport` trait, transports, routing, multihop, MTU, DAITA |
| `02-control-plane.md` | [04 — Control Plane](../04-control-plane/README.md), [08 — API Contracts](../08-api-contracts/README.md) | Go monolith, data model, events, coordinator, API, reconciliation |
| `03-client-core-and-ui.md` | [05 — Client Core & UI](../05-client-core-ui/README.md) | FFI surface, Flutter UI, platform shims, state management |
| `04-security-privacy-pki.md` | [06 — Security, Privacy & PKI](../06-security-privacy-pki/README.md) | Zero-trust, identity, PKI, no-logging, kill-switch, PQ, threat model |
| `05-repo-layout-tooling-and-helix-ecosystem.md` | [07 — Infrastructure & DevOps](../07-infrastructure-devops/README.md) | Repo layout, codegen, `helixvpnctl`, substrates, ecosystem |
| `06-phase0-spike-wbs.md` | [11 — Guides & FAQs](../11-guides-faqs/README.md) | Phase-0 gates G1–G6, milestones S0–S8 |
| `07-phase1-mvp-wbs.md` | [11 — Guides & FAQs](../11-guides-faqs/README.md) | Phase-1 MVP WBS, 8-criteria DoD, SLOs |
| `08-phase2-parity-wbs.md` | [11 — Guides & FAQs](../11-guides-faqs/README.md) | Phase-2 parity WBS, P2-AC + SLOs |
| `09-phase3-reach-wbs.md` | [11 — Guides & FAQs](../11-guides-faqs/README.md) | Phase-3 extended-reach WBS, gates G20–G26 |
| `10-testing-acceptance-and-qa.md` | [09 — Testing & QA](../09-testing-qa/README.md) | §11.4.169 test types, evidence model, acceptance gates |
| `11-deep-research-appendix.md` | [12 — Appendix: Research](../12-appendix-research/README.md) | Consolidated cited research (10 angles) |
| `99-source-coverage-ledger.md` | this section | Source-coverage proof and residual gaps |
| `MASTER_INDEX.md` | [00 — Executive Summary](../00-executive-summary/README.md) | Full document tree, statuses, quality items |
| `REFINEMENT_NOTES.md` | [00 — Executive Summary](../00-executive-summary/README.md) | Pass-1→pass-N punch-list |

### Nano-detail volumes

| Volume | Dir | Implementation section |
|---|---|---|
| V0 Meta | `v00-meta/` | [00](../00-executive-summary/README.md), this section |
| V1 Product | `v01-product/` | [01](../01-product-scope/README.md), [08](../08-api-contracts/README.md) |
| V2 Data Plane | `v02-data-plane/` | [03](../03-data-plane/README.md) |
| V3 Control Plane | `v03-control-plane/` | [04](../04-control-plane/README.md), [08](../08-api-contracts/README.md) |
| V4 Clients | `v04-client/` | [05](../05-client-core-ui/README.md) |
| V5 Security | `v05-security/` | [06](../06-security-privacy-pki/README.md) |
| V6 Deploy | `v06-deploy/` | [07](../07-infrastructure-devops/README.md) |
| V7 Execution | `v07-execution/` | [11](../11-guides-faqs/README.md) |
| V8 Testing | `v08-testing/` | [09](../09-testing-qa/README.md) |
| V9 Research | `v09-research/` | [12](../12-appendix-research/README.md) |
| V10 Design | `v10-design/` | [10](../10-design-system/README.md) |

### External design-system index

| File | Implementation section |
|---|---|
| `docs/design/README.md` | [10 — Design System](../10-design-system/README.md) |

---

## 3. Residual coverage gaps

The following distinctive source ideas are deliberately not adopted or are recorded as open.

### G1 — `[03_ZAI]` disaster-recovery completeness — ✅ RESOLVED

`v06-deploy/disaster-recovery.md` now consolidates the RTO/RPO budget, KMS-encrypted backups, and Terraform-driven region-failover runbook that the original ledger called for. Independently re-verified 2026-07-04. No further action.

> **Traceability note:** This resolves the documentation side of requirements-traceability **GAP-3** (NFR-205 DR RTO/RPO targets are still `UNVERIFIED` pending the first **CHAOS region-failover drill in Phase 2** — a measurement deferral, not a missing doc).

### G2 — `[06_GRK]` sing-box transport framework — deliberate divergence

**Proposed:** use sing-box as the universal transport multiplexer.
**Decision:** **not adopted**. The protocols it carries (Hysteria2, Shadowsocks, UDP-over-TCP) are reflected in the Phase-2 transport set, but the framework itself conflicts with the Rust-core decision and the iOS NE memory ceiling. The custom Rust `helix-transport` crate gives byte-for-byte client↔edge reuse.

**Rationale recorded:** `99-source-coverage-ledger.md` §G2 and [`03 — Data Plane](../03-data-plane/README.md).

### G3 — `[09_GCT]` Fyne desktop UI — deliberate divergence

**Proposed:** use Fyne for desktop/web client.
**Decision:** **not adopted**. Flutter reaches all 8 required platforms (including Aurora OS and HarmonyOS NEXT) with one codebase and a shared `helix_design` system. Fyne lacks mobile/niche-OS reach.

**Rationale recorded:** `99-source-coverage-ledger.md` §G3 and [`05 — Client Core & UI](../05-client-core-ui/README.md).

### GAP cross-check (`requirements-traceability.md` register)

The independent GAP-1..GAP-6 register in `v00-meta/requirements-traceability.md` is kept current with this consolidation:

| GAP | Status | Residual |
|---|---|---|
| GAP-1 | Closed in source docs | local-ACL × central-policy precedence pinned in `v03-control-plane/svc-policy.md` and `v04-client/helix-core-rust.md` |
| GAP-2 | Intentional / documented | FR-1103 (Rosenpass) is an evaluation, not a runtime capability; no runtime test type required |
| GAP-3 | Doc-level closed | NFR-205 RTO/RPO targets remain `UNVERIFIED` until the first Phase-2 CHAOS region-failover drill measures them |
| GAP-4 | Closed in source docs | Single owning doc created: `v04-client/connector.md` consolidates FR-701..707 and the advertise/route/registry contracts |
| GAP-5 | Closed at doc level | All 15 `v08-testing/` docs + `coverage-ledger-schema.md` authored; evidence states are `PENDING` in spec phase |
| GAP-6 | Closed in source docs | NFR-413/NFR-414/FR-610 added to pin `DDOS`/RBAC/rate-limiting owners; quantitative targets remain `UNVERIFIED` until Phase-2 benchmarks |

---

## 4. Decision coverage cross-check

All eight key decisions are surfaced explicitly in `final/`:

| Decision | Camp A | Camp B | Surfaced in |
|---|---|---|---|
| D1 obfuscation | MASQUE/QUIC | Hysteria2 + Salamander | `01-data-plane.md`, `08-phase2-parity-wbs.md`, `11-deep-research-appendix.md` |
| D2 client core | Rust | Go | `03-client-core-and-ui.md`, `11-deep-research-appendix.md` |
| D3 event bus | Redis → NATS | NATS from start | `02-control-plane.md`, `08-phase2-parity-wbs.md` |
| D4 subnet collision | ULA/4via6 | CGNAT | `02-control-plane.md`, `08-phase2-parity-wbs.md` |
| D5 edge language | Rust | Go | `01-data-plane.md`, `06-phase0-spike-wbs.md` |
| D6 transport topology | single protocol | asymmetric per-leg | `01-data-plane.md`, `11-deep-research-appendix.md` |
| D7 ambition | lean | Connectivity-OS | `00-product-scope-and-principles.md`, `SPECIFICATION.md` |
| D8 licensing | source-available + commercial | pure OSS | `SPECIFICATION.md` §9 |

---

## 5. Cross-references

- Authoritative ledger → [`../../99-source-coverage-ledger.md`](../../99-source-coverage-ledger.md)
- Synthesis evidence base → [`../../v09-research/_SYNTHESIS.md`](../../v09-research/_SYNTHESIS.md)
- Gap analysis → [`../../../../../reviews/mvp-final/findings/phase1-docs-gap-analysis.md`](../../../../../reviews/mvp-final/findings/phase1-docs-gap-analysis.md)

---

*Source: `docs/research/mvp/final/99-source-coverage-ledger.md`.*
