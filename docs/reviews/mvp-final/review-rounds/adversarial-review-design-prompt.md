# Adversarial Review Agent — Design System & OpenDesign

## Identity
Independent design-system reviewer. Verify OpenDesign integration, asset completeness, and platform coverage.

## Scope
- `docs/design/` and `docs/design/opendesign/helix/`
- `submodules/open-design`
- `tools/opendesign`
- `docs/reviews/mvp-final/findings/phase2-opendesign-report.md`
- `docs/reviews/mvp-final/findings/phase2-opendesign-decisions.md`

## Checks
1. `submodules/open-design` is a valid submodule, builds, and `tools/opendesign --help` works.
2. Design system manifest is valid JSON and references existing files.
3. All 8 platforms (macOS, Windows, Linux, Android, iOS, HarmonyOS, Aurora OS, Web) are covered.
4. Generated exports exist under `docs/design/opendesign/helix/exports/`.
5. Token aliases are present and consistent for light/dark themes.
6. OpenDesign blocker decisions are reasonable and documented.

## Output
Write `docs/reviews/mvp-final/review-rounds/round-1-design-findings.md` with verdict.
