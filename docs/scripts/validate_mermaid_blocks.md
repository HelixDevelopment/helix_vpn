# validate_mermaid_blocks.py — companion guide

**Revision:** 1
**Last modified:** 2026-06-26T00:00:00Z

## Overview

`scripts/testing/validate_mermaid_blocks.py` render-validates every
` ```mermaid ` block under a path using `mmdc`. A block that fails to render is
a Constitution **§11.4.168** defect — under the export pipeline it would ship as
RAW SOURCE in the HTML/PDF instead of an image. This is the anti-bluff gate
proving every diagram in the spec actually renders, and the enumerated-discovery
tool (**§11.4.118**) for finding broken diagrams.

## Prerequisites

- `python3` (stdlib), `mmdc` (mermaid-cli), `scripts/testing/mermaid.config.json`

## Usage

```
validate_mermaid_blocks.py [PATH]
```

- `PATH` — file or directory. Default: `docs/research/mvp/final`.

Output: one `FAIL <file>:<line> [<type>] -> <error>` per broken diagram, then
`mermaid-validate: total=N ok=M failed=K`.

Exit: `0` all render · `1` ≥1 failed · `2` mmdc absent.

## Common mermaid syntax faults (empirically confirmed in this spec)

1. **`;` in a sequence/state message or note** — mermaid treats `;` as a
   statement separator. Fix: `;` → `,` (or split). The validator stops at the
   *first* error per block, so a block may need several passes.
2. **A second `:` in a stateDiagram-v2 transition label** (first `:` is the
   label delimiter). Fix: reword, e.g. `:53` → `port 53`.
3. **erDiagram attribute block inline on one line with `;`**. Fix: one
   attribute per line inside `{ }`, format `TYPE name [PK|FK] ["comment"]`, no
   `;`. Note: `PK_FK` is invalid — use `PK,FK`.
4. **`splitLineToFitWidth does not support newlines`** on a `<br/>` label — if
   it persists, replace `<br/>` with a space.
5. **Parens in a flowchart edge pipe-label** (`|advertise()|`) — quote it:
   `|"advertise()"|`.

To debug one block: write it to `/tmp/x.mmd` and run
`mmdc -i /tmp/x.mmd -o /tmp/x.png -b white -s3 -c scripts/testing/mermaid.config.json`.

## Anti-bluff (§1.1)

A deliberately-broken diagram MUST be reported `failed=1` (exit 1); a good one
`failed=0` (exit 0). Verified 2026-06-26.

## Related scripts

- `scripts/testing/render_mermaid_blocks.py` — the renderer used by the export
- `scripts/testing/sync_all_markdown_exports.sh` — the export driver

_Last verified: 2026-06-26 (full spec tree: 356 diagrams, failed=0)._
