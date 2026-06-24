#!/usr/bin/env bash
# scripts/post_constitution_update.sh
# §11.4.164 (Constitution.md line 9593) wrapper: invoke after:
#   git submodule update --remote constitution
# Calls constitution/scripts/post_update_hook.sh then runs the full
# validation sweep (scripts/verify-all-constitution-rules.sh).
# Usage: bash scripts/post_constitution_update.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
HOOK="${REPO_ROOT}/constitution/scripts/post_update_hook.sh"
VERIFY="${REPO_ROOT}/scripts/verify-all-constitution-rules.sh"

echo "=== §11.4.164 post-constitution-update ==="
echo "Repo root: ${REPO_ROOT}"
echo ""

# ---------------------------------------------------------------------------
# Step 1: invoke the constitution's own post-update hook
# ---------------------------------------------------------------------------
if [ -f "${HOOK}" ]; then
    echo "--- Running post_update_hook.sh ---"
    bash "${HOOK}"
    echo "--- post_update_hook.sh completed ---"
else
    echo "WARN: post_update_hook.sh not found at ${HOOK}" >&2
    echo "      Skipping hook; proceeding to validation sweep." >&2
fi
echo ""

# ---------------------------------------------------------------------------
# Step 2: run the full validation sweep
# ---------------------------------------------------------------------------
echo "--- Running verify-all-constitution-rules.sh ---"
bash "${VERIFY}"
