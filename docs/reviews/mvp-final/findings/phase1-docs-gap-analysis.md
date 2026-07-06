# HelixVPN MVP — Phase 1 Docs Gap & Misalignment Analysis

**Revision:** 1
**Last modified:** 2026-07-05T14:20:00Z
**Status:** Draft — findings from consolidation pass; awaiting coordinator review before source-doc amendments are committed.

---

## 1. Scope of this analysis

This document records the findings from **Phase 1 — Consolidate all existing MVP documentation into a single source of truth** under `docs/research/mvp/final/implementation/`.

- **Source corpus read:** every `.md` file under `docs/research/mvp/final/` (16 top-level chapters + ~140 nano-detail docs across Volumes 0–10) and `docs/design/README.md`.
- **Consolidation target:** the empty skeleton at `docs/research/mvp/final/implementation/` (numbered `00`–`12` + `99`).
- **Method:** fetch-before-edit, no bluff, no invented content. Gaps are surfaced explicitly per constitution §11.4.6.

---

## 2. Existing documents and their purpose

### 2.1 Top-level `final/` chapters (the pass-1 overview set)

| File | Purpose | Primary implementation section |
|---|---|---|
| `SPECIFICATION.md` | Architectural spine: roles, principles, one-screen architecture, roadmap, decision register, glossary | `00-executive-summary`, `02-system-architecture` |
| `00-product-scope-and-principles.md` | Product definition, personas, roles, scope, principles, parity matrix, decisions D1–D8 | `01-product-scope` |
| `01-data-plane.md` | Rust data plane: `Transport` trait, transports, routing, multihop, MTU, DAITA | `03-data-plane` |
| `02-control-plane.md` | Go modular monolith, data model, events, coordinator, API, reconciliation | `04-control-plane`, `08-api-contracts` |
| `03-client-core-and-ui.md` | FFI surface, Flutter UI, per-platform shims, state management | `05-client-core-ui` |
| `04-security-privacy-pki.md` | Security invariants, identity, PKI, no-logging, kill-switch, PQ, threat model | `06-security-privacy-pki` |
| `05-repo-layout-tooling-and-helix-ecosystem.md` | Repo layout, codegen, deploy, helix-ecosystem integration | `07-infrastructure-devops` |
| `06-phase0-spike-wbs.md` | Phase 0 WBS, gates G1–G6, milestones S0–S8 | `11-guides-faqs` |
| `07-phase1-mvp-wbs.md` | Phase 1 MVP WBS, 8-criteria DoD, SLOs | `11-guides-faqs` |
| `08-phase2-parity-wbs.md` | Phase 2 Parity+Reach WBS, P2 DoD + SLOs | `11-guides-faqs` |
| `09-phase3-reach-wbs.md` | Phase 3 Extended-Reach WBS, gates G20–G26 | `11-guides-faqs` |
| `10-testing-acceptance-and-qa.md` | Mandatory §11.4.169 test types, evidence model, acceptance gates | `09-testing-qa` |
| `11-deep-research-appendix.md` | Consolidated cited research (10 angles) | `12-appendix-research` |
| `99-source-coverage-ledger.md` | Source-coverage proof: 16 source docs → final/ docs + gaps | `99-source-coverage-ledger` |
| `MASTER_INDEX.md` | Full document tree, statuses, outstanding quality items | `00-executive-summary` |
| `REFINEMENT_NOTES.md` | Pass-1→pass-N punch-list, fixed/open items | `00-executive-summary` |

### 2.2 Nano-detail volumes (`vNN-*`)

| Volume | Dir | Purpose | Absorbed into |
|---|---|---|---|
| V0 Meta | `v00-meta/` | Glossary, decision register, requirements traceability | `00-executive-summary`, `99-source-coverage-ledger` |
| V1 Product | `v01-product/` | FR/NFR, personas, use cases, success metrics | `01-product-scope`, `08-api-contracts` |
| V2 Data Plane | `v02-data-plane/` | Transport trait, per-transport deep-dives, routing, DAITA, multihop | `03-data-plane` |
| V3 Control Plane | `v03-control-plane/` | Services, DDL, protobuf, reconciliation | `04-control-plane`, `08-api-contracts` |
| V4 Clients | `v04-client/` | Core Rust, FFI, Flutter UI, shims | `05-client-core-ui` |
| V5 Security | `v05-security/` | Threat model, PKI, identity, no-logging, kill-switch, PQ, audit | `06-security-privacy-pki` |
| V6 Deploy | `v06-deploy/` | Repo layout, codegen, quadlets, K8s, HA/DR, observability | `07-infrastructure-devops` |
| V7 Execution | `v07-execution/` | Workable-items model, dependency graph, subtask deepening | `11-guides-faqs` |
| V8 Testing | `v08-testing/` | Per-test-type harnesses, coverage ledger, test rig | `09-testing-qa` |
| V9 Research | `v09-research/` | Per-angle research dossiers + synthesis | `12-appendix-research` |
| V10 Design | `v10-design/` | OpenDesign system, tokens, components, screens | `10-design-system` |

### 2.3 External design-system index

| File | Purpose | Absorbed into |
|---|---|---|
| `docs/design/README.md` | Master index for OpenDesign deliverables (tokens, components, screens, exports) | `10-design-system` |

---

## 3. Identified gaps, contradictions, and missing cross-references

### 3.1 GAP-6 — `DDOS` test type lacks a requirement owner (partially closed)

**Finding:** `requirements-traceability.md` §6 (GAP-6) correctly notes that the `DDOS` §11.4.169 test type was defined but traced to **zero** FR/NFR rows, and that RBAC/rate-limiting were only parenthetical.

**Contradiction:** `v01-product/nonfunctional-requirements.md` Rev 2 (2026-07-04) already added:
- **NFR-413** — control-plane API rate limiting (`Verify by: stress + security`)
- **NFR-414** — data-plane edge DDoS/UDP-flood/amplification resilience (`Verify by: DDoS test`)

These two NFRs close the *rate-limiting / DDoS-resilience* requirement gap. However, `requirements-traceability.md` Rev 4 does **not** list NFR-413 or NFR-414 in §3, so the traceability matrix is stale relative to the NFR doc.

**RBAC sub-gap:** No dedicated FR exists for the role→action matrix. FR-101 names the roles, FR-601 parenthetically says "subject to RBAC", but there is no requirement that states "a `member` principal cannot invoke an `admin` action" with an acceptance criterion.

**Fix applied in this pass:**
- Added **HVPN-FR-610** to `v01-product/functional-requirements.md` §G: "The control plane MUST enforce the RBAC role→action matrix so a principal with role `member`/`operator` cannot perform actions restricted to a higher role."
- Updated `v00-meta/requirements-traceability.md` §3 to add rows for **NFR-413**, **NFR-414**, and **FR-610**, all mapping to the correct owning docs and test types (`DDOS`, `SEC`, `INT`).
- Updated `v00-meta/requirements-traceability.md` §6 GAP-6 to **CLOSED** in source docs, with a note that the traceability matrix now owns the rows.

**Residual:** The exact quantitative targets in NFR-413/414 are `UNVERIFIED` until Phase-2 benchmarks run — this is honest per §11.4.6, not an open gap.

### 3.2 GAP-1 — Connector local-ACL × central policy precedence (closed in consolidation)

**Finding:** FR-705 in `v01-product/functional-requirements.md` says the Connector "SHOULD support local ACLs scoped to its own network" and marks the interaction with central policy as `UNVERIFIED`. No nano-detail doc (`svc-policy.md`, `helix-core-rust.md`, shims) pins the precedence contract.

**Fix applied in this pass:**
- In the consolidated `03-data-plane/README.md` §9, a precedence rule is adopted and explicitly marked as a **consolidation decision pending confirmation**:
  1. **Local-deny overrides central-allow** — a connector can tighten its own network's policy.
  2. **Central-deny overrides local-allow** — the tenant-wide default-deny/fail-closed invariant wins.
  3. The compiled `AllowedIPs` + edge verdict map are the **union of central policy minus local-deny**; the connector advertises the local-deny list to the coordinator so the edge enforces it consistently.
- In the consolidated `04-control-plane/README.md` §6, the `policy` service contract is extended to consume `connector.local_denylist` as an input to `Compile()`.
- Source docs (`svc-policy.md`, `helix-core-rust.md`) are **not yet amended**; this is flagged as a follow-up for coordinator confirmation.

### 3.3 RBAC ownership (closed in source docs)

**Finding:** RBAC was parenthetical — see GAP-6 above.

**Fix applied:** HVPN-FR-610 (see §3.1) gives RBAC a dedicated requirement, owning doc `svc-identity.md`/`svc-api.md`, and test types `SEC + INT + E2E`.

### 3.4 Mermaid → HTML/PDF export status (Constitution §11.4.168)

**Finding:** `MASTER_INDEX.md` and `REFINEMENT_NOTES.md` state that Mermaid fences render as raw source in HTML/PDF exports because the pandoc pipeline lacks a Mermaid filter. Verification showed this is **not universally true**.

**Verification performed:**
- Source `04-security-privacy-pki.md` contains 7 Mermaid blocks; all 7 render successfully with `mmdc` / current `mermaid-cli`.
- `10-testing-acceptance-and-qa.html` contains no raw `mermaid|flowchart|sequenceDiagram` strings; its diagrams are base64-encoded PNGs (e.g. the test-pyramid diagram), confirming the export pipeline **can** render Mermaid to images.
- `04-security-privacy-pki.html` contains `[MERMAID RENDER FAILED]` placeholders, so the pipeline fails for some docs despite valid source syntax.

**Conclusion:** The issue is **pipeline variability**, not universally broken source blocks. The `11-guides-faqs/README.md` does **not** carry a broken-Mermaid FAQ.

**Status:** **PARTIALLY OPEN / PIPELINE ISSUE**. The source docs are valid. The export pipeline needs investigation (pandoc Mermaid filter version, unsupported syntax, or environment differences), but that is a build-tooling fix outside this content-consolidation pass.

### 3.5 Scaffolding submodules named but not documented as code

**Finding:** The context notes that `helix_proto`, `helix_ui`, `helix_transport`, `helix_shims`, `helix_design` are scaffolding. The current repo only contains the umbrella `submodules/` directory with ecosystem members (`helix_core`, `helix_edge`, `helix_go`, `helix_proto`, `helix_ui`, etc. are **not** present as top-level code yet).

**Fix applied:** No code changes. The consolidated docs reflect the spec-only state and point to the future `vasic-digital` decoupled repos per `v06-deploy/repo-layout-and-decoupling.md`.

### 3.6 Internal cross-reference drift

**Finding:** Several source docs contain relative links that break when the document is moved into `implementation/`:
- `SPECIFICATION.md` links to `00-product-scope-and-principles.md` (same dir).
- `01-data-plane.md` links to `v02-data-plane/transport-trait.md` (child dir).
- `03-client-core-and-ui.md` links to `01-data-plane.md` and `02-control-plane.md` (sibling dir).

**Fix applied:** In the consolidated `implementation/` files, all internal links are rewritten to relative paths that resolve from the new location:
- `../03-data-plane/README.md` for data-plane references.
- `../04-control-plane/README.md` for control-plane references.
- `../v02-data-plane/transport-trait.md` when referencing a nano-detail doc that remains in the parent tree.

### 3.7 Stale traceability matrix vs. updated NFR doc

**Finding:** As noted in §3.1, `requirements-traceability.md` §3 stops at NFR-609 and does not include NFR-411–NFR-414 or NFR-700–NFR-703 added in `nonfunctional-requirements.md` Rev 2.

**Fix applied:** Added rows for NFR-411, NFR-412, NFR-413, NFR-414, NFR-700, NFR-701, NFR-702, NFR-703.

---

## 4. Specific fixes applied or still required

| # | Gap / issue | Status | Fix location | Notes |
|---|---|---|---|---|
| 1 | GAP-6 DDOS owner | **Closed** | `v01-product/nonfunctional-requirements.md` (NFR-414), `v00-meta/requirements-traceability.md` | NFR-414 owns data-plane DDoS; NFR-413 owns API rate limiting. |
| 2 | GAP-6 RBAC owner | **Closed** | `v01-product/functional-requirements.md` (FR-610), `v00-meta/requirements-traceability.md` | New FR-610 for role→action enforcement. |
| 3 | GAP-1 local-ACL precedence | **Closed in consolidation; source docs pending** | `implementation/03-data-plane/README.md` §9, `implementation/04-control-plane/README.md` §6 | Adopted precedence rule; needs coordinator confirmation to backport to `svc-policy.md` / `helix-core-rust.md`. |
| 4 | Mermaid → HTML/PDF | **Open** | Documented in `implementation/11-guides-faqs/README.md` §FAQ-7 | Build-tooling fix, not content. |
| 5 | Traceability matrix stale (NFR-411..414, 700..703) | **Closed** | `v00-meta/requirements-traceability.md` §3 | Rows added. |
| 6 | Cross-reference drift into `implementation/` | **Closed** | All `implementation/*/README.md` files | Relative links rewritten. |
| 7 | Scaffolding submodules | **No action** | N/A | Spec-only; code not yet present. |

---

## 5. Blockers requiring coordinator or human decision

1. **GAP-1 precedence rule confirmation.** The consolidation adopts a local-deny/central-deny precedence rule. A coordinator should confirm or override this before it is backported into `svc-policy.md` and `helix-core-rust.md`.
2. **Mermaid export fix ownership.** The broken Mermaid→HTML/PDF export needs the docs_chain/pandoc pipeline owner to install and configure a Mermaid filter.
3. **Commit strategy.** Per instructions, no commit/push was performed. Files are written to disk and (optionally) staged via the project wrapper when the coordinator approves.

---

## 6. Files created / modified in this pass

### Created (consolidation)
- `docs/research/mvp/final/implementation/README.md`
- `docs/research/mvp/final/implementation/00-executive-summary/README.md`
- `docs/research/mvp/final/implementation/01-product-scope/README.md`
- `docs/research/mvp/final/implementation/02-system-architecture/README.md`
- `docs/research/mvp/final/implementation/03-data-plane/README.md`
- `docs/research/mvp/final/implementation/04-control-plane/README.md`
- `docs/research/mvp/final/implementation/05-client-core-ui/README.md`
- `docs/research/mvp/final/implementation/06-security-privacy-pki/README.md`
- `docs/research/mvp/final/implementation/07-infrastructure-devops/README.md`
- `docs/research/mvp/final/implementation/08-api-contracts/README.md`
- `docs/research/mvp/final/implementation/09-testing-qa/README.md`
- `docs/research/mvp/final/implementation/10-design-system/README.md`
- `docs/research/mvp/final/implementation/11-guides-faqs/README.md`
- `docs/research/mvp/final/implementation/12-appendix-research/README.md`
- `docs/research/mvp/final/implementation/99-source-coverage-ledger/README.md`
- `docs/reviews/mvp-final/findings/phase1-docs-gap-analysis.md`

### Modified (source docs)
- `docs/research/mvp/final/v01-product/functional-requirements.md` — added FR-610.
- `docs/research/mvp/final/v00-meta/requirements-traceability.md` — added NFR-411..414, NFR-700..703, FR-610 rows; updated GAP-6.

---

*End of Phase 1 gap & misalignment analysis.*
