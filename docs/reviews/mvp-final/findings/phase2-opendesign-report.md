# Phase 2 — OpenDesign Integration Report

**Project:** Helix VPN  
**Workspace:** `/run/media/milosvasic/DATA4TB/Projects/helix_vpn`  
**Date:** 2026-07-05  
**Engineer:** DevOps/frontend integration engineer (Kimi Code CLI subagent)

---

## Executive summary

OpenDesign was added as a git submodule, built, and exercised against the existing `docs/design/opendesign/helix/` design system. The daemon starts, the design system imports successfully, and a full brand archive export was generated. The CLI is functional for the core `design-systems import-local` flow, but several practical blockers were discovered:

- CLI stdout does not flush reliably when piped/redirected (use a TTY or the REST API for automation).
- The imported token contract scores **31/100** and is graded **needs-rebuild** because many A1 tokens fall back to importer defaults.
- The existing `manifest.json` uses a non-OpenDesign schema and key names; OpenDesign normalizes it on import.
- No Figma/Sketch/Adobe/XD/Penpot **exporters** exist in OpenDesign; only Figma **import** is provided.

---

## Environment

| Tool | Version / Path |
|------|----------------|
| Node.js | v22.19.0 |
| pnpm | 10.33.2 (installed globally with `npm install -g pnpm@10.33.2`) |
| Go | 1.26.2 (project runtime, not used for OpenDesign) |
| OpenDesign CLI version | `0.12.1` |
| OpenDesign commit | `50b38c4fb0c31f8a58184cee68e2bb549de17f97` |
| OpenDesign submodule path | `submodules/open-design` |
| Project wrapper | `tools/opendesign` |

---

## Exact commands used

### 1. Add the submodule

```bash
cd /run/media/milosvasic/DATA4TB/Projects/helix_vpn
git submodule add -b main git@github.com:nexu-io/open-design.git submodules/open-design
# Fallback tried when SSH timed out:
# git submodule add -b main https://github.com/nexu-io/open-design.git submodules/open-design
git submodule update --init --recursive submodules/open-design
```

### 2. Install pnpm and build OpenDesign

```bash
npm install -g pnpm@10.33.2
cd submodules/open-design
pnpm config set engine-strict false
pnpm install
```

`pnpm install` completed in ~126s. The `postinstall` script built the daemon and all workspace packages. Node 22 emits an `Unsupported engine` warning, but the build succeeds.

### 3. Verify the CLI

```bash
./tools/opendesign --help
./tools/opendesign version   # 0.12.1
```

### 4. Start the daemon

```bash
./tools/opendesign --no-open --port 7456
# Listening log: [od] listening on http://127.0.0.1:7456
```

### 5. Import the Helix VPN design system

```bash
./tools/opendesign design-systems import-local \
  docs/design/opendesign/helix --name "Helix VPN" --json
```

Captured output: `docs/design/opendesign/helix/exports/opendesign-import-local.json`  
Assigned design system id: `user:helix-vpn-2` (a second import because `user:helix-vpn` already existed from earlier testing).

### 6. List and inspect design systems

```bash
./tools/opendesign design-systems list --json
./tools/opendesign design-systems show user:helix-vpn --json
```

Because of the CLI stdout-flush issue, the JSON was captured with a pseudo-TTY:

```bash
script -q -c '/run/media/milosvasic/DATA4TB/Projects/helix_vpn/tools/opendesign design-systems list --json' /dev/null
```

Captured output: `docs/design/opendesign/helix/exports/opendesign-design-systems-list.json`

### 7. Export the full brand archive

HTTP API (daemon must be running):

```bash
curl -s -o docs/design/opendesign/helix/exports/helix-vpn-opendesign-archive.zip \
  http://127.0.0.1:7456/api/design-systems/user:helix-vpn/archive
```

Archive size: **48,563 bytes**. It was extracted into `docs/design/opendesign/helix/exports/`.

### 8. Direct HTTP inspection

```bash
curl -s http://127.0.0.1:7456/api/design-systems/user:helix-vpn | python3 -m json.tool
```

---

## Files created / modified

### New / updated project files

| File | Purpose |
|------|---------|
| `submodules/open-design` | New git submodule (OpenDesign repository) |
| `tools/opendesign` | Project wrapper script for the OpenDesign CLI |
| `docs/design/README.md` | Added **OpenDesign Integration** section |
| `docs/reviews/mvp-final/findings/phase2-opendesign-report.md` | This report |

### Generated exports

All outputs are under `docs/design/opendesign/helix/exports/`:

| Path | Description |
|------|-------------|
| `archive/helix-vpn-opendesign-archive.zip` | Full brand export from OpenDesign |
| `manifest.json` | OpenDesign-normalized manifest (`schemaVersion: od-design-system-project/v1`) |
| `DESIGN.md` | Rewritten design system spec |
| `tokens.css` | Canonical CSS custom properties |
| `design-tokens.json` | Structured token export |
| `tailwind-v4.css` | Tailwind v4 theme mapping |
| `colors_and_type.css` | Color/type quick-reference stylesheet |
| `components.html` | Component reference page |
| `components.manifest.json` | Component manifest |
| `index.html` | Design-system landing page |
| `preview/*.html` | Pre-rendered preview pages (colors, typography, spacing, buttons, inputs, app) |
| `source/token-contract.report.json` | Token-contract validation report |
| `source/evidence.md` | Importer evidence notes |
| `source/scanned-files.json` | List of scanned source files |
| `source/tokens.source.json` | Extracted token sources |
| `ui_kits/app/components/*` | Generated UI kit components |
| `opendesign-import-local.json` | Captured CLI output from import |
| `opendesign-design-systems-list.json` | Captured CLI output from design-systems list |
| `opendesign-version.txt` | CLI version output |

### Dotfiles / lockfile changes

- `.gitmodules` — added `submodules/open-design` entry.
- `submodules/open-design/pnpm-lock.yaml` / `node_modules` — created by `pnpm install` inside the submodule.

---

## Validation results

### Manifest schema conformance

The existing `docs/design/opendesign/helix/manifest.json` differs from the OpenDesign canonical manifest:

| Field | Existing file | OpenDesign canonical (generated) |
|-------|---------------|----------------------------------|
| `schemaVersion` | `"1.0"` | `"od-design-system-project/v1"` |
| `id` | `"helix-vpn-design-system"` | `"helix-vpn"` (truncated on import) |
| `category` | `"Security & Privacy"` | `"Imported"` (overridden by importer) |
| `files.designMd` | `"DESIGN.md"` | `files.design` |
| `files.tokensCss` | `"tokens.css"` | `files.tokens` |
| `files.componentsHtml` | `"components.html"` | `files.components` |
| Extra keys expected | — | `files.designTokens`, `files.tailwind`, `usage`, `componentsManifest`, `importMode`, `craft`, `preview`, `sourceFiles` |

**Result:** The importer normalizes these differences automatically, but the source manifest is **not natively conformant**. No CLI failure occurs.

### Token contract quality

Source report: `docs/design/opendesign/helix/exports/source/token-contract.report.json`

| Metric | Value |
|--------|-------|
| Total tokens | 56 |
| Source-backed tokens | 13 |
| Source-backed A1 | 6 / 26 required |
| Fallback tokens | 43 |
| Score | 31 |
| Grade | `needs-rebuild` |
| Recommend rebuild | `true` |

**Key gaps:**

- A1-structure layer: 0/18 source-backed.
- A2 layer: 3/26 source-backed.
- `--surface` (A1-identity) falls back to importer default `#ffffff`.
- `--muted` (A1-identity) falls back to importer default `#6b7280`.

**Root cause:** The source `tokens.css` uses a project-specific prefix (`--hx-*`) rather than OpenDesign's canonical token names, so the importer relies on heuristics and defaults.

### Components / screens

- `components.html` is recognized and included in the export.
- The importer did **not** detect standalone component files (Button/Input/Card/Nav/Sidebar) because they are embedded in the single HTML fixture.
- No dedicated screen files exist; OpenDesign generated preview pages from the imported assets, not from a screen manifest.

---

## Converters (Figma / Sketch / Adobe / XD / Penpot)

Investigated the OpenDesign repository for export converters.

| Format | Status | Evidence |
|--------|--------|----------|
| Figma | Import only | `od figma import --project <id> --file <path.fig>`; `apps/daemon/src/figma/figma-import.ts` |
| Sketch | Not available | No exporter found |
| Adobe XD | Not available | No exporter found |
| Adobe (other) | Not available | No exporter found |
| Penpot | Not available | No exporter found |

**Result:** No converter outputs were produced. This is a limitation of OpenDesign, not a project-level error.

---

## Blockers requiring coordinator decision

1. **Schema alignment**  
   Should `docs/design/opendesign/helix/manifest.json` be rewritten to match OpenDesign's `od-design-system-project/v1` schema? The importer currently normalizes it, but a native schema would improve validation scores and reduce importer guesswork.

2. **Token contract rebuild**  
   OpenDesign recommends a token-contract rebuild (`od design-systems rebuild-token-contract user:helix-vpn`). This is a design-time decision: should the team adopt OpenDesign's canonical token names (e.g., `--surface`, `--fg`, `--muted`) in addition to the existing `--hx-*` tokens, or keep the project-specific naming and accept the `needs-rebuild` grade?

3. **CLI stdout flush bug**  
   The CLI returns empty output when piped/redirected. This breaks scripted automation unless workarounds (`script`, direct HTTP API) are used. This should be reported upstream to OpenDesign. In the meantime, the README documents the workaround.

4. **Node 22 support**  
   OpenDesign declares Node `~24`. The build works on Node 22 with `engine-strict=false`, but this is an unsupported configuration. Coordinator should decide whether to upgrade Node or accept the warning.

5. **No design exporters**  
   If Figma/Sketch/XD/Penpot deliverables are required, an external tool or manual workflow is needed. OpenDesign cannot generate them.

---

## Is OpenDesign CLI functional?

**Yes, conditionally.**

- `od --help`, `od version`, `od design-systems import-local`, `od design-systems list`, and `od design-systems show` all execute.
- The daemon starts and responds on `http://127.0.0.1:7456`.
- The full brand archive exports successfully via the HTTP API.
- Output capture requires a TTY wrapper (`script`) or direct HTTP calls because of a stdout-flush issue when piped.

---

## Next recommended steps

1. Decide on manifest schema alignment (blocker #1).
2. Decide on token naming strategy (blocker #2); if canonical names are adopted, run `od design-systems rebuild-token-contract user:helix-vpn`.
3. Report the CLI stdout-flush issue to the OpenDesign maintainers.
4. If Figma/XD/etc. exports are needed, research external converters (e.g., `html-to-figma`, Penpot import) or manual handoff.

---

*Report generated automatically during Phase 2 integration work. No commits or pushes were performed.*
