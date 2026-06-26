# workable_items_loader.py

**Revision:** 1
**Last modified:** 2026-06-26T15:30:00Z

## Overview

Parses the four HelixVPN WBS Markdown documents and their subtask deepening
companions, then populates the §11.4.93 SQLite single-source-of-truth at
`docs/workable_items.db`. This is the **md-to-db** direction of the
bidirectional sync contract.

## Prerequisites

- Python 3.10+ (stdlib only — no external packages)
- SQLite 3 (via Python's `sqlite3` module)
- The WBS source docs must exist at their canonical paths

## Usage

```bash
# Full load (clears + re-creates all items)
python3 scripts/workable_items_loader.py

# Dry run (prints what would be loaded)
python3 scripts/workable_items_loader.py --dry-run
```

## Inputs

| File | Phase | Path |
|------|-------|------|
| Phase 0 WBS | P0 | `docs/research/mvp/final/06-phase0-spike-wbs.md` |
| Phase 1 WBS | P1 | `docs/research/mvp/final/07-phase1-mvp-wbs.md` |
| Phase 2 WBS | P2 | `docs/research/mvp/final/08-phase2-parity-wbs.md` |
| Phase 3 WBS | P3 | `docs/research/mvp/final/09-phase3-reach-wbs.md` |
| P1 deepening | P1 | `docs/research/mvp/final/v07-execution/subtask-deepening-p1.md` |
| P2 deepening | P2 | `docs/research/mvp/final/v07-execution/subtask-deepening-p2.md` |
| P3 deepening | P3 | `docs/research/mvp/final/v07-execution/subtask-deepening-p3.md` |

## Output

- `docs/workable_items.db` — SQLite database with tables: `items`, `item_history`,
  `test_diary`, `gates`, `operator_block_details`, `obsolete_details`, `meta`

## Item counts (expected)

| Phase | Epics/Milestones | Tasks | Subtasks | Total |
|-------|-----------------|-------|----------|-------|
| P0 | 9 | 27 | — | 36 |
| P1 | 15 | 65 | 130 | 210 |
| P2 | 11 | 54 | 77 | 142 |
| P3 | 8 | 35 | 53 | 96 |
| **Total** | **43** | **181** | **260** | **484** |

## Parsing patterns

- **Phase 0 tasks**: `### TASK HVPN-P0-NNN — Title` headings
- **Phase 1+ tasks**: `- **HVPN-P1-NNN — Title.**` inline bold format
- **Epics**: `## N. E01 — Title` or `## N. Milestone S0 — Title`
- **Subtasks**: Table rows in deepening docs: `| .k | Title | Acceptance | Tests | Cx |`

## Field extraction

Fields are extracted from task body text using regex patterns:
- `· module: <name>` → `module`
- `· gate: G1` → `gate`
- `· type: Task|Feature|Bug` → `type`
- `· deps: HVPN-P1-001` → `deps` (JSON array)
- `· effort: M(5)` or `· effort: 3 days` → `effort_days`
- `· tests: UNIT,INT` → `test_types` (normalized to canonical §11.4.169 codes)

## Cross-references

- Constitution §11.4.93 (SQLite-backed SSoT)
- Constitution §11.4.95 (DB tracked in git)
- Constitution §11.4.54 (HVPN-Pn-NNN id convention)
- Spec: `v07-execution/workable-items-model.md`
- Go replacement: `cmd/workable-items/` (HVPN-P1-150)
