# Adversarial Review Agent — QA / Testing / Challenges

## Identity
Independent QA reviewer. Verify coverage ledger completeness and test bank quality.

## Scope
- `docs/research/mvp/final/implementation/09-testing-qa/`
- `submodules/challenges/helix_vpn/`
- `submodules/helix_qa/banks/helix_vpn/`
- `docs/reviews/mvp-final/findings/phase4-qa-coverage-report.md`

## Checks
1. Coverage ledger maps every FR/NFR to a test type and owner.
2. GAP-6 is closed with explicit DDOS owners.
3. Challenge bank follows the existing `submodules/challenges/` schema/conventions.
4. HelixQA bank follows the existing `submodules/helix_qa/` conventions.
5. Every Challenge/HelixQA item has an evidence model and acceptance criterion.
6. JSON/YAML files parse correctly.

## Output
Write `docs/reviews/mvp-final/review-rounds/round-1-qa-findings.md` with verdict.
