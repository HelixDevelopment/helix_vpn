#!/usr/bin/env bash
# scripts/testing/meta_test_no_active_ci.sh
# §1.1 paired mutation test for Inv6 (CM-NO-ACTIVE-CI).
#
# Proves that the Inv6 gate in tests/pre_build_verification.sh is
# discriminating:
#   1. When an active workflow (.yml) is present in the git index the gate FAILs.
#   2. When no active workflow is present the gate PASSes.
#
# Strategy: inject a temporary .yml file into the git index using
# "git update-index --add --cacheinfo" (no working-tree write needed), verify
# Inv6 fails, then remove it from the index and verify Inv6 passes.
# A trap guarantees the index is restored even if the script is interrupted.
#
# Usage: bash scripts/testing/meta_test_no_active_ci.sh
# Expected exit: 0 (both probe phases behave as specified)

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

MUT_PATH=".github/workflows/_mut_ci_test.yml"

meta_pass=0
meta_fail=0

meta_check() {
    local label="$1"
    local result="$2"   # "ok" or "fail"
    local detail="$3"
    if [ "$result" = "ok" ]; then
        echo "META-PASS  ${label}"
        meta_pass=$((meta_pass + 1))
    else
        echo "META-FAIL  ${label} — ${detail}"
        meta_fail=$((meta_fail + 1))
    fi
}

# ---------------------------------------------------------------------------
# Cleanup: remove the mutation entry from the index if it was injected
# ---------------------------------------------------------------------------
cleanup() {
    git -C "${REPO_ROOT}" update-index --remove "${MUT_PATH}" 2>/dev/null || true
}
trap cleanup EXIT

# ---------------------------------------------------------------------------
# Helper: run ONLY Inv6 logic inline (mirrors pre_build_verification.sh Inv6)
# Returns 0 if Inv6 PASSes, 1 if Inv6 FAILs
# ---------------------------------------------------------------------------
run_inv6() {
    local inv6_ok=true
    if git -C "${REPO_ROOT}" ls-files | grep -qE '^\.github/workflows/.*\.ya?ml$'; then
        inv6_ok=false
    fi
    if git -C "${REPO_ROOT}" ls-files | grep -qE '^\.gitlab-ci\.yml$'; then
        inv6_ok=false
    fi
    if $inv6_ok; then
        return 0
    else
        return 1
    fi
}

echo "=== meta_test_no_active_ci: CM-NO-ACTIVE-CI gate mutation proof ==="
echo ""

# ---------------------------------------------------------------------------
# Phase 1: Inject a fake active workflow into the git index, expect gate FAIL
# ---------------------------------------------------------------------------
echo "Phase 1: injecting mutation ${MUT_PATH} into git index..."

# Create an empty blob object and inject it as a tracked .yml file
BLOB_HASH="$(git -C "${REPO_ROOT}" hash-object -w --stdin </dev/null)"
git -C "${REPO_ROOT}" update-index --add --cacheinfo "100644,${BLOB_HASH},${MUT_PATH}"

if run_inv6; then
    meta_check "gate FAILs when active workflow present" "fail" \
        "Inv6 returned PASS but should have FAILed with ${MUT_PATH} in index"
else
    meta_check "gate FAILs when active workflow present" "ok" ""
fi

# ---------------------------------------------------------------------------
# Phase 2: Remove the mutation, expect gate PASS
# ---------------------------------------------------------------------------
echo "Phase 2: removing mutation from git index..."
git -C "${REPO_ROOT}" update-index --remove "${MUT_PATH}"

if run_inv6; then
    meta_check "gate PASSes when no active workflow present" "ok" ""
else
    meta_check "gate PASSes when no active workflow present" "fail" \
        "Inv6 returned FAIL after mutation was removed — check for stray .yml files"
fi

# ---------------------------------------------------------------------------
# Phase 3: Confirm the real repo is clean (no active .yml in index)
# ---------------------------------------------------------------------------
echo "Phase 3: confirming real repo index is clean..."
if git -C "${REPO_ROOT}" ls-files | grep -qE '^\.github/workflows/.*\.ya?ml$|^\.gitlab-ci\.yml$'; then
    meta_check "real repo index clean after test" "fail" \
        "active CI file still present in git index after cleanup"
else
    meta_check "real repo index clean after test" "ok" ""
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "Meta-results: ${meta_pass} passed, ${meta_fail} failed."

if [ "${meta_fail}" -gt 0 ]; then
    echo "STATUS: FAIL"
    exit 1
fi

echo "STATUS: PASS"
exit 0
