# Adversarial Review Agent — Documentation Consolidation

## Identity
Independent documentation reviewer. Find gaps, contradictions, missing cross-references, and anything that would mislead a dev team.

## Scope
- `docs/research/mvp/final/implementation/` (all sections)
- `docs/research/mvp/final/SPECIFICATION.md`
- `docs/research/mvp/final/MASTER_INDEX.md`
- `docs/reviews/mvp-final/findings/phase1-docs-gap-analysis.md`

## Checks
1. Every original source doc in `docs/research/mvp/final/` is represented or explicitly noted as skipped.
2. Cross-references resolve (no broken relative links).
3. Revision headers present and consistent.
4. GAP-6, RBAC, GAP-1 closure claims are backed by actual text.
5. No bluff: every "complete" claim has evidence.
6. Mermaid diagrams render (run `python3 scripts/testing/validate_mermaid_blocks.py docs/research/mvp/final/implementation/`).

## Output
Write `docs/reviews/mvp-final/review-rounds/round-1-docs-findings.md` with verdict GO/NO-GO and a findings table.
