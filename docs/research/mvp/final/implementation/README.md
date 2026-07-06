# HelixVPN MVP — Consolidated Implementation Source of Truth

**Revision:** 1
**Last modified:** 2026-07-05T14:20:00Z
**Status:** Draft — Phase 1 consolidation; subordinate to `docs/research/mvp/final/SPECIFICATION.md`.

---

## What this directory is

This is the **single source of truth for HelixVPN MVP implementation guidance**. It merges the 16 top-level `final/` chapters, the ~140 nano-detail docs across Volumes 0–10, and the OpenDesign system index into a numbered, navigable set.

Each section is self-contained. Use this index to jump to the area you need; each section links back to the authoritative nano-detail docs in the parent tree.

---

## Navigation

| # | Section | What it covers | Source docs |
|---|---|---|---|
| **00** | [Executive Summary](./00-executive-summary/README.md) | One-page product, vision, differentiators, roadmap | `SPECIFICATION.md`, `MASTER_INDEX.md` |
| **01** | [Product Scope](./01-product-scope/README.md) | What it is/is not, roles, scope, principles, parity matrix, decisions | `00-product-scope-and-principles.md`, `v01-product/*` |
| **02** | [System Architecture](./02-system-architecture/README.md) | C4 view, components, data/control-plane split, repo layout | `SPECIFICATION.md` §5–§6, `02-control-plane.md`, `01-data-plane.md` |
| **03** | [Data Plane](./03-data-plane/README.md) | `Transport` trait, transports, WG core, orchestrator, routing, policy, DAITA, multihop | `01-data-plane.md`, `v02-data-plane/*` |
| **04** | [Control Plane](./04-control-plane/README.md) | Go services, DDL/RLS, events, coordinator, API, reconciliation | `02-control-plane.md`, `v03-control-plane/*` |
| **05** | [Client Core & UI](./05-client-core-ui/README.md) | `helix-core` FFI, Flutter UI, platform shims, state management | `03-client-core-and-ui.md`, `v04-client/*`, `v04-client/connector.md` |
| **06** | [Security, Privacy & PKI](./06-security-privacy-pki/README.md) | Zero-trust, identity, PKI, no-logging, kill-switch, PQ, threat model | `04-security-privacy-pki.md`, `v05-security/*` |
| **07** | [Infrastructure & DevOps](./07-infrastructure-devops/README.md) | Repo layout, codegen, `helixvpnctl`, Podman/Compose/K8s, ecosystem | `05-repo-layout-tooling-and-helix-ecosystem.md`, `v06-deploy/*` |
| **08** | [API Contracts](./08-api-contracts/README.md) | REST/WS/SSE, Connect-RPC, protobuf, FFI surface | `02-control-plane.md` §7, `v03-control-plane/svc-api.md`, `protobuf-spec.md`, `v04-client/ffi-surface.md`; GAP-1 precedence rule pinned in `v01-product/functional-requirements.md`, `v03-control-plane/svc-policy.md`, `v04-client/helix-core-rust.md` |
| **09** | [Testing & QA](./09-testing-qa/README.md) | §11.4.169 test types, evidence model, acceptance gates, rigs | `10-testing-acceptance-and-qa.md`, `v08-testing/*` |
| **10** | [Design System](./10-design-system/README.md) | OpenDesign, tokens, components, screens, exports | `docs/design/README.md`, `v10-design/*` |
| **11** | [Guides & FAQs](./11-guides-faqs/README.md) | Phase roadmap, WBS summary, operational runbooks, FAQ | `06/07/08/09-phase*-wbs.md`, `v07-execution/*`, `v06-deploy/disaster-recovery.md` |
| **12** | [Appendix: Research](./12-appendix-research/README.md) | Cited external research per angle | `11-deep-research-appendix.md`, `v09-research/*` |
| **99** | [Source Coverage Ledger](./99-source-coverage-ledger/README.md) | Source doc → implementation section mapping + residual gaps | `99-source-coverage-ledger.md`, this tree |

---

## How this consolidation relates to the source docs

- **Authoritative detail remains in the parent `final/` tree.** This consolidation is the navigable entry point; if a detail is missing here, read the owning nano-detail doc.
- **Cross-references use relative paths** from each `implementation/NN-*/README.md` back to sibling sections (`../03-data-plane/README.md`) and up to the parent nano-detail docs (`../../v02-data-plane/transport-trait.md`).
- **Known gaps closed in this pass:** GAP-1 (precedence rule backported), GAP-3 (DR RTO/RPO targets/runbooks documented; measurement pending Phase-2 drill), GAP-4 (single `v04-client/connector.md` owning doc), GAP-5 (`v08-testing/` 15 nano-detail docs authored; residual evidence-state `PENDING` is the honest spec-phase state), GAP-6 (DDOS/RBAC/rate-limiting owners), RBAC ownership. The only residual seams are quantitative targets that must be measured once code is built. See `docs/reviews/mvp-final/signoffs/gap-closure-summary.md`.

---

*End of master navigation index.*
