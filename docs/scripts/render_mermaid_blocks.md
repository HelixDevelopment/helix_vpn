# render_mermaid_blocks.py — companion guide

**Revision:** 2
**Last modified:** 2026-07-04T16:45:00Z

## Overview

`scripts/testing/render_mermaid_blocks.py` pre-processes a Markdown file so
that every ` ```mermaid ` fenced block is replaced by a reference to a rendered
**PNG** image, **before** the file is handed to pandoc. This closes the
Constitution **§11.4.168** gap: pandoc has no mermaid filter and weasyprint
cannot execute JavaScript, so without pre-rendering the diagram *source* would
ship as literal text in the HTML/PDF exports.

It is invoked automatically by `scripts/testing/sync_all_markdown_exports.sh`;
it can also be run standalone.

## Prerequisites

- `python3` (stdlib only)
- `mmdc` (mermaid-cli) on `PATH` — `npm i -g @mermaid-js/mermaid-cli`
- Config: `scripts/testing/mermaid.config.json`

## Usage

```
render_mermaid_blocks.py <input.md> <output.md> <png_dir> [<cache_dir>]
```

- `<input.md>` — source (never modified)
- `<output.md>` — pre-processed copy; each fence → `![diagram](<abs-png>)`
- `<png_dir>` — where per-block PNGs are written
- `<cache_dir>` — optional content-addressed cache (sha256 of source+config);
  a cache hit reuses the PNG instead of re-rendering. The driver wrapper uses
  `<repo>/.mermaid-cache` (gitignored; regen mechanism = the wrapper, §11.4.77).

Prints `mermaid: rendered=N reused=M failed=K`.

## Why PNG (not SVG)

Empirically (`§11.4.6`): weasyprint does **not** render mermaid's default-config
SVG `<foreignObject>` text — it vanishes in the PDF. An `htmlLabels:false` SVG
keeps native text but breaks `<br/>` labels. Rendering to PNG lets Chromium
(inside `mmdc`) do the full rendering — every label feature — and weasyprint
simply embeds the raster, which it can never fail to display. OCR of the
rendered PDF page confirms readability (the §11.4.168 validation oracle).
Proven across all 8 diagram types the spec uses.

## Edge cases

- **A diagram fails to render** (syntax error): the script exits **1** and the
  output carries a `**[MERMAID RENDER FAILED — block #N]**` marker — NEVER raw
  source. The wrapper marks the file FAILED. Fix the diagram source (see
  `validate_mermaid_blocks.md`) — a render failure is a real defect, not a
  tooling glitch.
- **No mermaid fences**: the input is copied verbatim.
- **Every diagram fails with a bare, unhelpful "[object Object]" / "Failed to
  launch the browser process!"** (2026-07-04 root-caused via a direct
  `puppeteer.launch({dumpio: true})` probe — not guessed): `mmdc`'s bundled/
  cached Puppeteer-downloaded Chromium build can fail to launch on some hosts
  (observed cause on this host: the cached build under `~/.cache/puppeteer`
  would not start; a system-package-installed `chromium` with its shared-lib
  deps resolved by the OS package manager launched fine). The script now
  auto-discovers a system `chromium`/`chromium-browser`/`google-chrome`/
  `google-chrome-stable` on `PATH` at import time and, if found, writes a
  throwaway Puppeteer config (`executablePath` + `--no-sandbox`) passed to
  `mmdc -p`; if none is found, it falls back to `mmdc`'s own default
  resolution unchanged (so hosts where the bundled Chrome already works are
  unaffected). Never hardcodes an absolute browser path in a tracked file —
  discovered fresh via `PATH` every run, per the project's no-guessing /
  resolve-by-stable-name discipline.

## Related scripts

- `scripts/testing/validate_mermaid_blocks.py` — render-validate every diagram
- `scripts/testing/sync_all_markdown_exports.sh` — the export driver
- `scripts/testing/mermaid.config.json` — mermaid render config

_Last verified: 2026-06-26 (rendered all 356 spec diagrams; 0 raw-source leaks
into PDF confirmed via pdftotext + OCR)._
