# sync_all_markdown_exports.sh

**Revision:** 1
**Last modified:** 2026-06-26T00:00:00Z

Companion user guide for `scripts/testing/sync_all_markdown_exports.sh`
(§11.4.18 script documentation mandate).

## Overview

Closes the §11.4.65 Universal Markdown export gap for the HelixVPN spec
tree. For every `*.md` document under `docs/research/mvp/final/`
(recursive) it generates two synchronized sibling artifacts:

- `<name>.html` — via `pandoc` (standalone, self-contained, embedded
  resources, document title taken from the source's first H1).
- `<name>.pdf` — via `weasyprint`, rendered from the generated HTML.

The script is idempotent: a sibling that is already newer than its source
`.md` is skipped unless `--force` is given. Conversion runs sequentially
(concurrency = 1) to respect the §12.6 60 % memory ceiling, with a
`timeout 60` per file per format.

## Prerequisites

- `pandoc` on `PATH` (HTML generation). Verified: pandoc 3.9.x.
- `weasyprint` on `PATH` (PDF generation). Verified: WeasyPrint 66.x.
- `bash`, `find`, and a `timeout`/`gtimeout` binary (optional — the
  script degrades gracefully to no per-call timeout if absent).

## Usage

```sh
# Whole spec tree (docs/research/mvp/final/, recursive)
bash scripts/testing/sync_all_markdown_exports.sh

# Limit to a sub-path (relative to repo root, or absolute)
bash scripts/testing/sync_all_markdown_exports.sh docs/research/mvp/final/v02-data-plane

# Force regeneration even when siblings are newer than the source
bash scripts/testing/sync_all_markdown_exports.sh --force
```

Final line of output:

```
generated=N skipped=M failed=K
```

Exit code `0` when `K == 0`, exit `1` when any file/format failed (the
failed source paths are also printed to stderr).

## Edge cases

- **Per-file failure does not abort the run.** A pandoc or weasyprint
  failure on one document is logged to stderr, the run continues, and the
  process exits non-zero at the end so the failure is not silently lost
  (§11.4.1 — no FAIL-bluff, no silent skip).
- **Concurrent spec generation.** The file list is snapshotted by a
  single `find` at start. If another process adds `.md` files mid-run,
  those late arrivals are not in the snapshot — simply re-run the script
  (idempotent) to generate siblings for them; already-current siblings
  are skipped.
- **mtime skip.** A sibling is skipped only when it is strictly newer than
  its `.md` source. Editing the source makes both siblings stale and they
  regenerate on the next run.
- **Title extraction.** The first ATX `# H1` line is used as the pandoc
  document title; if none is present the basename is used.
- **Mermaid code fences (§11.4.168 known limitation).** Plain pandoc does
  NOT render ```` ```mermaid ```` blocks into diagrams — they are emitted
  as `<pre class="mermaid">` source text in the HTML, and therefore appear
  as raw diagram source (not a rendered picture) in the PDF. Rendering
  Mermaid requires a pandoc filter (e.g. `mermaid-filter` /
  `mermaid-cli`), which is out of scope for this wrapper. Until such a
  filter is wired in, the HTML/PDF carry the prose faithfully but show
  Mermaid diagrams as code, not images. Do not claim visual diagram
  fidelity for these blocks.

## Internal behaviour

1. Resolve repo root from the script location, set scope root to
   `docs/research/mvp/final`.
2. Parse args: optional path-prefix (limit scope) and `--force`.
3. Verify `pandoc` + `weasyprint` are present; abort with a clear error
   otherwise.
4. `find` all `*.md` under the walk root into a newline-delimited temp
   file (spec doc paths contain no newlines), iterate with
   `while IFS= read -r`.
5. For each source: compute `.html`/`.pdf` sibling paths, apply the mtime
   skip unless `--force`, then run pandoc (HTML) and weasyprint (PDF)
   under `timeout 60`.
6. Tally `generated` / `skipped` / `failed`, print the summary, exit
   `0`/`1`.

The script is written POSIX-ish and parses cleanly under BOTH `sh -n` and
`bash -n` (§11.4.67) — no bash-only constructs (process substitution, NUL
handling, `[[ ]]`, arrays) are used.

## Related scripts

- This is a project-level instantiation of the §11.4.65 export discipline
  for the spec tree. A full-repo equivalent (`docs/**`, owned-submodule
  READMEs, etc.) is the broader §11.4.65 mandate; this wrapper covers the
  `docs/research/mvp/final/` MVP specification corpus specifically.

## Last verified date

2026-06-26 — full run over `docs/research/mvp/final/` produced
`generated=75 skipped=14 failed=0`; an idempotent follow-up run closed
the gap from concurrently-added spec files (md = html = pdf counts equal,
zero missing siblings). One generated PDF was confirmed to carry real
rendered prose (20 pages, title + revision header + body) via `pypdf`;
the matching HTML carried rendered `<h1>`/`<h2>` heading tree + ToC.
Mermaid blocks confirmed to remain as `<pre class="mermaid">` source (see
Edge cases).
