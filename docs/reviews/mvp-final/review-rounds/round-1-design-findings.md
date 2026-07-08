# Round 1 — Independent Design-System Review: OpenDesign Package

**Reviewer:** Independent adversarial design-system reviewer (Kimi Code CLI subagent)  
**Scope:** `docs/design/`, `docs/design/opendesign/helix/`, `submodules/open-design`, `tools/opendesign`, `docs/reviews/mvp-final/findings/phase2-opendesign-report.md`, `docs/reviews/mvp-final/findings/phase2-opendesign-decisions.md`  
**Date:** 2026-07-05  
**Verdict:** **GO-with-conditions**

---

## Executive summary

The Helix VPN OpenDesign package is structurally sound and the tooling chain is operational. The CLI wrapper works, the submodule is populated and built, the design system imports successfully, and a full brand archive plus preview exports have been generated. Canonical light/dark token aliases are present, all eight target platforms are documented, and the blocker-decisions document is reasonable.

However, the source manifest references two files that do not exist in the source tree, the token contract still grades `needs-rebuild` (31/100), and there is a naming drift between the requested "Web" platform and the manifest's `web-extension` / DESIGN.md's "Web Extension". These issues are blockers to calling the package fully production-ready and must be resolved or explicitly accepted by the coordinator before final sign-off.

---

## Check results

### 1. CLI wrapper — PASS

```bash
bash tools/opendesign --help
```

Output: full usage banner displayed (daemon, tools, artifacts, MCP, plugins, diagnostics, export, etc.). The wrapper script at `tools/opendesign` correctly resolves `submodules/open-design/apps/daemon/bin/od.mjs` and executes with `node`.

### 2. Submodule population — PASS

`submodules/open-design` exists and contains:

- `.git` (submodule pointer present)
- `package.json` (`name: open-design`, `version: 0.12.1`, `private: true`)
- `apps/daemon/bin/od.mjs` (executable, 496 bytes)
- `node_modules/` and `pnpm-lock.yaml` from completed `pnpm install`

### 3. Source manifest validity — PARTIAL FAIL

`docs/design/opendesign/helix/manifest.json` is valid JSON (`python3 -m json.tool` parses cleanly). It references the following files:

| Key | Referenced path | Exists in source? | Exists in exports? |
|-----|-----------------|-------------------|--------------------|
| `files.design` | `DESIGN.md` | ✅ | ✅ |
| `files.tokens` | `tokens.css` | ✅ | ✅ |
| `files.designTokens` | `design-tokens.json` | ❌ | ✅ |
| `files.tailwind` | `tailwind-v4.css` | ❌ | ✅ |
| `files.components` | `components.html` | ✅ | ✅ |

**Finding:** The source design-system directory is missing `design-tokens.json` and `tailwind-v4.css`, even though the canonical manifest declares them. They are only present as generated artifacts under `docs/design/opendesign/helix/exports/`. A manifest that points to non-existent source files breaks reproducibility: re-importing from source would not produce the same OpenDesign contract.

### 4. Eight-platform documentation — PASS with naming caveat

All eight platforms are explicitly covered:

- **macOS** — `DESIGN.md` §1.3 table + tray icon specs
- **Windows** — `DESIGN.md` §1.3 table + tray icon specs
- **Linux** — `DESIGN.md` §1.3 table + tray icon specs
- **Android** — `DESIGN.md` §1.3 + §5.2
- **iOS** — `DESIGN.md` §1.3 + §5.2
- **HarmonyOS** — `DESIGN.md` §1.3 + §5.2
- **Aurora OS** — `DESIGN.md` §1.3 + §5.2
- **Web** — described as "Web Extension" in `DESIGN.md`; manifest lists `web-extension`

**Naming caveat:** The review scope asked for "Web". The manifest encodes it as `web-extension` and `DESIGN.md` labels it "Web Extension". Functionally this is the Web platform, but the naming is inconsistent with the canonical platform list and could confuse consumers or downstream tooling.

### 5. Generated exports — PASS

Exports exist under `docs/design/opendesign/helix/exports/`:

- **Archive:** `archive/helix-vpn-opendesign-archive.zip` (48,563 bytes)
- **Manifest:** `manifest.json` (OpenDesign-normalized)
- **Tokens:** `tokens.css`, `design-tokens.json`
- **Components:** `components.html`, `components.manifest.json`
- **Preview:** `preview/colors.html`, `preview/typography.html`, `preview/spacing.html`, `preview/components-buttons.html`, `preview/components-inputs.html`, `preview/app.html`, plus additional palette/specimen pages

The import error file `exports/opendesign-import-local.err` is 0 bytes.

### 6. Light/dark canonical token aliases — PASS

`docs/design/opendesign/helix/tokens.css` contains canonical OpenDesign aliases under both `:root` (light) and `[data-theme="dark"]`:

```css
:root {
  --bg: var(--hx-bg-primary);
  --surface: var(--hx-bg-secondary);
  --fg: var(--hx-text-primary);
  --muted: var(--hx-text-tertiary);
  --accent: var(--hx-primary-500);
  ...
}

[data-theme="dark"] {
  --bg: var(--hx-bg-primary);
  --surface: var(--hx-bg-secondary);
  --fg: var(--hx-text-primary);
  --muted: var(--hx-text-tertiary);
  --accent-hover: var(--hx-primary-400);
  ...
}
```

This satisfies the decision to keep `--hx-*` as source-of-truth while adding canonical aliases.

### 7. Blocker decisions reasonableness — PASS

`docs/reviews/mvp-final/findings/phase2-opendesign-decisions.md` addresses all five blockers from the Phase 2 report:

1. **Manifest schema alignment** — Adopt canonical `od-design-system-project/v1`. Reasonable.
2. **Token contract rebuild** — Keep `--hx-*` names and add canonical aliases. Reasonable, though the resulting `needs-rebuild` grade is honestly acknowledged.
3. **CLI stdout flush bug** — Document workaround and defer upstream report. Reasonable for MVP.
4. **Node 22 support** — Accept with `engine-strict=false`. Reasonable for this round.
5. **No Figma/Sketch/XD/Penpot exporters** — Accept OpenDesign limitation. Reasonable if vendor deliverables are not in MVP scope.

---

## Detailed findings

### Finding 1 — Source manifest references missing files (HIGH)

**Evidence:**

```
$ python3 -c "import json,os; m=json.load(open('docs/design/opendesign/helix/manifest.json')); [print(k,v,os.path.exists(os.path.join('docs/design/opendesign/helix',v))) for k,v in m['files'].items()]"
design DESIGN.md True
tokens tokens.css True
designTokens design-tokens.json False
tailwind tailwind-v4.css False
components components.html True
```

`design-tokens.json` and `tailwind-v4.css` are present only in `exports/`, not in the source directory that the manifest describes. This makes the source package incomplete and means a clean re-import from `docs/design/opendesign/helix` would not have access to those files.

**Recommendation:** Either (a) copy or generate `design-tokens.json` and `tailwind-v4.css` into the source directory so the manifest is internally consistent, or (b) remove the `files.designTokens` and `files.tailwind` keys from the source manifest if they are intended to be import-generated artifacts only. Option (a) is preferred because the OpenDesign canonical manifest expects these files.

### Finding 2 — Token contract still grades `needs-rebuild` (MEDIUM)

**Evidence:**

```json
{
  "summary": {
    "totalTokens": 56,
    "sourceBackedTokens": 13,
    "sourceBackedA1": 6,
    "requiredA1": 26,
    "fallbackTokens": 43,
    "score": 31,
    "grade": "needs-rebuild",
    "recommendRebuild": true
  }
}
```

Although canonical aliases were added, OpenDesign's scanner does not treat `var(--hx-*)` references as source-backed literal values. As a result, many A1/A2 tokens still fall back to importer defaults. The decisions document acknowledges this and accepts it for MVP kick-off.

**Recommendation:** Accept for MVP only if no downstream consumer relies on OpenDesign's token contract score. For a higher grade, provide literal canonical values or run `od design-systems rebuild-token-contract` after adopting OpenDesign's expected token names.

### Finding 3 — "Web" platform naming is inconsistent (LOW)

**Evidence:**

- Manifest platforms array: `"web-extension"`
- `DESIGN.md` platform table: "Web Extension"
- Review scope requested: "Web"

**Recommendation:** Align naming with the canonical platform list. If "Web Extension" is the deliberate MVP scope, document that "Web" means "Browser Extension" in this context. Otherwise, rename to `web` in the manifest and "Web" in `DESIGN.md`.

### Finding 4 — CLI stdout flush bug remains (LOW, documented)

The Phase 2 report and decisions document note that CLI output is not reliably captured when piped/redirected. This was accepted with a documented workaround (`script` pseudo-TTY or direct HTTP API). No further action required for this round, but it remains a risk for CI automation.

---

## Verdict rationale

**GO-with-conditions** is chosen because:

- The OpenDesign toolchain is functional and the package can be imported and exported.
- All eight target platforms are documented.
- Canonical light/dark token aliases exist.
- The blocker decisions are reasonable and honestly document residual limitations.

The package is **not** a clean GO because the source manifest is internally inconsistent (missing referenced files) and the token contract grade is below acceptable thresholds for a consumer-grade design system. These are resolvable conditions.

---

## Conditions for promotion to GO

1. **Fix source manifest consistency:** Add `design-tokens.json` and `tailwind-v4.css` to `docs/design/opendesign/helix/`, or adjust the manifest to not declare them as source files.
2. **Resolve or accept token contract grade:** Either rebuild the token contract to achieve a passing grade, or obtain explicit coordinator sign-off that `needs-rebuild` (31/100) is acceptable for the MVP.
3. **Clarify Web platform naming:** Decide whether the platform is "Web", "Web Extension", or "Browser Extension" and align the manifest + `DESIGN.md` accordingly.

---

## Top 3 findings

1. **Source manifest references missing `design-tokens.json` and `tailwind-v4.css`.** These files only exist in generated exports, making the source design-system package incomplete and non-reproducible on re-import.
2. **Token contract scores 31/100 and grades `needs-rebuild`.** Canonical aliases were added, but OpenDesign does not count `var()` references as source-backed values, leaving most A1/A2 tokens on importer defaults.
3. **"Web" platform naming is inconsistent.** The review scope asks for "Web"; the manifest says `web-extension` and `DESIGN.md` says "Web Extension".
