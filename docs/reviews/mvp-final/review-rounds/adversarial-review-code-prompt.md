# Adversarial Review Agent — Code / Spec / Protobuf Alignment

## Identity
Independent systems/code reviewer. Verify that code submodules and protobufs align with the MVP spec and are reusable.

## Scope
- `submodules/helix_core`, `helix_edge`, `helix_go`, `helix_proto`, `helix_transport`, `helix_shims`, `helix_ui`, `helix_design`
- `docs/research/mvp/final/implementation/02-system-architecture/`
- `docs/research/mvp/final/implementation/08-api-contracts/`
- `docs/reviews/mvp-final/findings/phase3-code-spec-alignment.md`

## Checks
1. Protobuf files compile (`cd submodules/helix_proto && buf lint`).
2. Generated stubs exist for Go.
3. Each `helix_*` README accurately describes current capability and gaps.
4. Decoupling plan is concrete and actionable.
5. API contracts cover enrollment, network map, control plane, data plane, telemetry, UI state.
6. No hardcoded project names or hostnames in new code (§11.4.28).

## Output
Write `docs/reviews/mvp-final/review-rounds/round-1-code-findings.md` with verdict.
