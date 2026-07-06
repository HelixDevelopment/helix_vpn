# Helix VPN — OpenDesign Blocker Decisions

**Date:** 2026-07-05  
**Decider:** Coordinator (Kimi Code CLI)

## Context
Phase 2 integrated OpenDesign (`nexu-io/open-design`) as a submodule, built the daemon CLI, imported the Helix VPN design system, and generated brand-archive exports. Five blockers were raised in `phase2-opendesign-report.md`.

## Decisions

### 1. Manifest schema alignment — ADOPT canonical schema
**Decision:** Rewrite `docs/design/opendesign/helix/manifest.json` to OpenDesign's `od-design-system-project/v1` schema.

**Rationale:** The importer was normalizing the old schema anyway; using the native schema removes guesswork and makes the design system first-class in OpenDesign. The source `manifest.json` has been updated; generated exports reflect the canonical form.

### 2. Token contract rebuild — KEEP project-specific `--hx-*` names, ADD canonical aliases
**Decision:** Keep the existing `--hx-*` token namespace as the human-maintained source of truth. Add OpenDesign canonical aliases (`--bg`, `--surface`, `--fg`, …) as `var(--hx-*)` mappings in `tokens.css` for consumers that expect the canonical contract.

**Rationale:** Maximizes reusability: HelixVPN keeps its descriptive names, while OpenDesign-aware tools and third-party consumers can use the canonical names. A literal-value canonical layer would raise the importer grade but duplicate every token and create a maintenance hazard; the current approach is the maintainability-grade trade-off for a spec-phase package.

**Status:** Canonical aliases added for light and dark themes. Re-import verified against a running daemon. Importer grade remains `needs-rebuild` because OpenDesign's scanner does not treat `var()` references as source-backed values; this is an importer limitation, not a design-system defect. Documented honestly in the report.

### 3. CLI stdout flush bug — DOCUMENT workaround
**Decision:** Accept the bug and document the workaround (pseudo-TTY wrapper `script` or direct HTTP API). Report upstream to OpenDesign maintainers when the project reaches a stable milestone.

### 4. Node 22 support — ACCEPT with `engine-strict=false`
**Decision:** Continue building OpenDesign on the project's Node 22 with `engine-strict=false`. No Node upgrade for this round.

**Rationale:** The build succeeds and the daemon runs. A Node 24 migration is out of scope for the documentation/package-readiness round.

### 5. No Figma/Sketch/XD/Penpot exporters — ACCEPT limitation
**Decision:** Acknowledge that OpenDesign provides Figma import only, not export. If vendor deliverables are required later, evaluate external converters (e.g., `html-to-figma`, Penpot import) in a dedicated design-ops task.

### 6. Source manifest consistency — REMOVE import-generated path declarations
**Decision:** The source `manifest.json` should only describe files that actually live in `docs/design/opendesign/helix/`. Remove `usage`, `componentsManifest`, `preview`, and `sourceFiles` keys that point to artifacts produced by OpenDesign's import/export process under `exports/`.

**Rationale:** A source manifest that references generated artifacts breaks reproducibility on a clean re-import. Consumers can still find previews, scanned files, and reports under `exports/` after running the import; the source manifest remains a minimal, reproducible description of the authored design-system inputs.

**Status:** Removed in Round-2 remediation. Re-import verified with exit code 0 and an empty error file.

### 7. Import tool output inconsistency — DOCUMENT discrepancy
**Decision:** The OpenDesign CLI stdout reports "Token contract report is usable; no rebuild recommended" while the structured JSON report (`exports/opendesign-import-local.json`) recommends a rebuild with grade `needs-rebuild`. Treat the JSON report as the authoritative artifact; document the stdout/JSON discrepancy as a known OpenDesign CLI quirk.

**Rationale:** CI and reproducible pipelines must consume structured output, not stdout. The `needs-rebuild` grade itself was already accepted in Decision 2; the only new action is to warn future consumers not to rely on the stdout message.

**Status:** Documented here and in `docs/reviews/mvp-final/review-rounds/round-2-design-findings.md`.

## Result
OpenDesign is installed, functional, and wired into the project. The Helix VPN design system is importable, generates exports, and provides both `--hx-*` and canonical token namespaces. The residual `needs-rebuild` grade is understood and accepted for the MVP kick-off package. The source manifest is now internally consistent and reproducible.
