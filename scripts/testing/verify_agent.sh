#!/usr/bin/env bash
# scripts/testing/verify_agent.sh
# §11.4.165 (Constitution.md line 9621) Universal Independent Verification Agent.
# Independent re-verification entrypoint; structurally separate from the author.
# Iterates to zero-finding GO per §11.4.134; self-validated by §1.1 mutation.
# Usage: bash scripts/testing/verify_agent.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

pass_count=0
fail_count=0

pass() { echo "PASS  $*"; pass_count=$((pass_count+1)); }
fail() { echo "FAIL  $*"; fail_count=$((fail_count+1)); }

echo "=== §11.4.165 Independent Verification Agent ==="
echo "Repo root: ${REPO_ROOT}"
echo ""

# ---------------------------------------------------------------------------
# Run the full governance sweep (verify-all-constitution-rules.sh)
# ---------------------------------------------------------------------------
echo "--- Running verify-all-constitution-rules.sh ---"
if bash "${REPO_ROOT}/scripts/verify-all-constitution-rules.sh"; then
    pass "verify-all-constitution-rules.sh aggregate"
else
    fail "verify-all-constitution-rules.sh aggregate"
fi
echo ""

# ---------------------------------------------------------------------------
# Independent structural checks
# ---------------------------------------------------------------------------
echo "--- Checking constitution/ submodule is initialized ---"
if [ -f "${REPO_ROOT}/constitution/Constitution.md" ]; then
    pass "constitution/Constitution.md present"
else
    fail "constitution/Constitution.md not present"
fi

echo "--- Checking .gitmodules entry ---"
if grep -qF '[submodule "constitution"]' "${REPO_ROOT}/.gitmodules" 2>/dev/null; then
    pass ".gitmodules has constitution entry"
else
    fail ".gitmodules missing constitution entry"
fi

# ---------------------------------------------------------------------------
# Check all four governance scripts exist
# ---------------------------------------------------------------------------
echo "--- Checking governance scripts exist ---"
for f in \
    "scripts/commit_all.sh" \
    "scripts/verify-all-constitution-rules.sh" \
    "scripts/post_constitution_update.sh" \
    "scripts/testing/verify_agent.sh"
do
    if [ -f "${REPO_ROOT}/${f}" ]; then
        pass "${f} exists"
    else
        fail "${f} missing"
    fi
done
echo ""

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo "Results: ${pass_count} passed, ${fail_count} failed."
if [ "${fail_count}" -gt 0 ]; then
    echo "STATUS: FAIL"
    exit 1
fi
echo "STATUS: PASS"
exit 0
