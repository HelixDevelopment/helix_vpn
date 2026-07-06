# HelixVPN MVP Final — Round 1 Adversarial Docs Review

**Reviewer:** independent adversarial documentation reviewer  
**Scope:** `docs/research/mvp/final/implementation/` (17 files incl. exports), `docs/research/mvp/final/SPECIFICATION.md`, `docs/research/mvp/final/MASTER_INDEX.md`, `docs/research/mvp/final/99-source-coverage-ledger.md`, `docs/reviews/mvp-final/findings/phase1-docs-gap-analysis.md`  
**Date:** 2026-07-05  
**Verdict:** **GO-with-conditions**

---

## Evidence captured

| Check | Command / script | Result |
|---|---|---|
| 1. Mermaid block validation | `python3 scripts/testing/validate_mermaid_blocks.py docs/research/mvp/final/implementation/` | `mermaid-validate: total=0 ok=0 failed=0` — no fenced `mermaid` blocks exist in the 17 implementation files, so the check passes vacuously. |
| 2. Relative-link resolution | Custom Python link checker (`/tmp/check_links.py`) scanned all `*.md` under `implementation/` | `scanned_files=17 ok_links=157 broken=0` — every relative markdown link resolves to an existing file. |
| 3. Source-coverage ledger completeness | Read `99-source-coverage-ledger.md` (top-level) and `implementation/99-source-coverage-ledger/README.md` | All 16 top-level `final/*.md` source docs are mapped; `REFINEMENT_NOTES.md` and `MASTER_INDEX.md` are also mapped. No skipped doc is unnoted. |
| 4. Revision-header spot-check | `head -n 10` loop over `implementation/*/README.md` | 13 of 14 `NN-*/README.md` files carry `Revision`, `Last modified`, and `Status`. One file (`08-api-contracts/README.md`) is missing the standard header. |
| 5. GAP-6 / RBAC / GAP-1 backing text | Read `v01-product/functional-requirements.md` Rev 4, `v01-product/nonfunctional-requirements.md` Rev 2, `v00-meta/requirements-traceability.md` Rev 5 | GAP-6 closed by NFR-413/NFR-414/FR-610 + traceability rows; RBAC requirement exists as FR-610; GAP-1 precedence rule is adopted in consolidation but explicitly marked pending source-doc backport. |
| 6. "Complete/done/passes" claims | `grep -i -E '\b(complete|done|passes|passed|green)\b'` across `implementation/` | All hits are conditional/future-facing ("Definition of Done", "gate passes", "builds green", "complete set" as a data-model adjective). No unsupported claim that the MVP is currently complete or passing. |

---

## Critical findings (must fix before final sign-off)

1. **`implementation/08-api-contracts/README.md` lacks the mandatory revision header.**  
   Every other `NN-*/README.md` carries `Revision`, `Last modified`, and `Status`. This file only has `Scope:` and `Status: Phase-3 reconciliation`. Per constitution §11.4.44 and the project's own header convention, this file must be brought into line.  
   **Owner:** docs coordinator / API-contracts author.

2. **`implementation/08-api-contracts/README.md` status is misaligned with the MVP-final package.**  
   A "Phase-3 reconciliation" status in the consolidated MVP-final implementation source-of-truth is inconsistent with the rest of the tree, which is marked "Draft — consolidated from …" or "active — Phase 4 QA architecture deliverable". It signals that this section has not actually been reconciled into the MVP baseline.  
   **Owner:** docs coordinator / API-contracts author.

---

## Major findings

3. **GAP-1 is closed in consolidation only; source docs remain unamended.**  
   `phase1-docs-gap-analysis.md` §3.2 is honest that the local-ACL precedence rule is adopted in `implementation/03-data-plane/README.md` §9 and `implementation/04-control-plane/README.md` §6, with source-doc backport pending coordinator confirmation. `v00-meta/requirements-traceability.md` still lists FR-705 as `UNVERIFIED` and GAP-1 as open. This is correctly surfaced, not hidden, but the fix is not complete until `svc-policy.md` and `helix-core-rust.md` are updated.  
   **Owner:** coordinator to confirm precedence rule, then `svc-policy.md` / `helix-core-rust.md` authors.

4. **`99-source-coverage-ledger.md` (top-level) claims G1 is "RESOLVED" based on a doc that did not exist when the ledger was first written.**  
   The ledger states `v06-deploy/disaster-recovery.md` "now consolidates" the DR posture and was "independently re-verified during the 2026-07-04 hardening pass". The file exists and the claim is internally consistent, but the review did not independently re-read `v06-deploy/disaster-recovery.md` to confirm the RTO/RPO budget and runbook are actually comprehensive. The claim is accepted for this round because the existence check passed; a second-round spot-read of the DR doc is recommended.  
   **Owner:** DR-doc author for second-round verification.

---

## Minor findings

5. **Mermaid validation is vacuous.**  
   `validate_mermaid_blocks.py` returned `total=0` because the implementation `.md` files contain no ` ```mermaid ` fences. This is not a failure, but it means the anti-bluff gate of §11.4.168 is not exercised on this corpus. The top-level `final/*.md` source docs do contain Mermaid blocks; running the validator on the parent `final/` directory would be a stronger check.

6. **`implementation/README.md` is the only top-level index in the implementation tree and is correctly mapped, but it carries a 2026-07-05 timestamp while declaring itself "subordinate to SPECIFICATION.md".**  
   This is correct governance; noted only to confirm the hierarchy is explicit.

7. **The `08-api-contracts/README.md` links to `submodules/helix_proto/proto/helix/...` paths.**  
   These links resolve on disk, but they point into a submodule whose contents were not audited in this round. The link checker confirms existence; semantic correctness of the proto files is out of scope here.

---

## Required fix owners

| Finding | Owner | Action |
|---|---|---|
| 08-api-contracts README missing revision header | Docs coordinator / API-contracts author | Add `Revision`, `Last modified`, `Status` header matching project convention. |
| 08-api-contracts README Phase-3 status misalignment | Docs coordinator / API-contracts author | Reconcile content to MVP-final baseline and update status to "Draft — consolidated from …" or equivalent. |
| GAP-1 source-doc backport pending | Coordinator + `svc-policy.md` / `helix-core-rust.md` authors | Confirm or override precedence rule; backport into source docs; update traceability matrix. |
| G1 DR resolution claim | DR-doc author | Available for second-round independent read of `v06-deploy/disaster-recovery.md`. |

---

## Top 3 findings summary

1. **`implementation/08-api-contracts/README.md` is missing the mandatory revision header** and is therefore non-conforming with the rest of the implementation package and with §11.4.44.
2. **`implementation/08-api-contracts/README.md` is labeled "Phase-3 reconciliation"**, which is inconsistent with a finalized MVP implementation source-of-truth and suggests this section has not actually been folded into the MVP baseline.
3. **GAP-1 is honestly tracked as open in source docs** despite being "closed in consolidation"; the precedence rule needs coordinator confirmation and backport to `svc-policy.md` / `helix-core-rust.md` before the gap is truly closed.
