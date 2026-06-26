# docs_chain_md_to_db.sh

**Revision:** 1
**Last modified:** 2026-06-26T15:30:00Z

## Overview

Wrapper script that bridges the docs_chain `exec` transform to the
`workable_items_loader.py` script. Called by docs_chain when the WBS
source markdown files change (content-hash drift detected).

## Prerequisites

- Python 3.10+ (for the loader)
- docs_chain binary (`submodules/docs_chain/docs_chain`)
- WBS source docs at canonical paths

## Usage

```bash
# Called automatically by docs_chain sync:
submodules/docs_chain/docs_chain sync workable-items

# Or directly:
./scripts/docs_chain_md_to_db.sh
```

## How it works

1. docs_chain detects content-hash drift in WBS source nodes
2. docs_chain invokes this script as the `exec` transform
3. The script calls `workable_items_loader.py` which reads all WBS files
4. The loader clears and re-populates the SQLite DB
5. docs_chain records the new content hash

## Cross-references

- Constitution §11.4.106 (docs_chain)
- Constitution §11.4.93 (workable-items SSoT)
- Loader: `scripts/workable_items_loader.py`
- Context: `.docs_chain/contexts/workable-items.yaml`
