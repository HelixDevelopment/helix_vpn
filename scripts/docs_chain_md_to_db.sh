#!/usr/bin/env bash
# docs_chain md-to-db wrapper
# Called by docs_chain exec transform: $1=input $2=output
# We ignore both — the loader reads all WBS files directly and writes to the DB.
# The wrapper exists so docs_chain can trigger the loader on source drift.
exec python3 "$(cd "$(dirname "$0")" && pwd)/workable_items_loader.py"
