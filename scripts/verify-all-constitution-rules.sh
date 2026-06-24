#!/usr/bin/env bash
# scripts/verify-all-constitution-rules.sh
# §11.4.32 (Constitution.md line 2019) canonical sweep: re-runs ALL governance
# gates and reports an aggregate result.
# Usage: bash scripts/verify-all-constitution-rules.sh
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

pass_count=0
fail_count=0

pass() { echo "PASS  $*"; pass_count=$((pass_count+1)); }
fail() { echo "FAIL  $*"; fail_count=$((fail_count+1)); }

echo "=== §11.4.32 Constitution governance sweep ==="
echo "Repo root: ${REPO_ROOT}"
echo ""

# ---------------------------------------------------------------------------
# Gate 1: pre_build_verification.sh
# ---------------------------------------------------------------------------
echo "=== Gate 1: pre_build_verification.sh ==="
if bash "${REPO_ROOT}/tests/pre_build_verification.sh"; then
    pass "pre_build_verification.sh"
else
    fail "pre_build_verification.sh"
fi
echo ""

# ---------------------------------------------------------------------------
# Gate 2: test_constitution_inheritance.sh
# ---------------------------------------------------------------------------
echo "=== Gate 2: test_constitution_inheritance.sh ==="
if bash "${REPO_ROOT}/tests/test_constitution_inheritance.sh"; then
    pass "test_constitution_inheritance.sh"
else
    fail "test_constitution_inheritance.sh"
fi
echo ""

# ---------------------------------------------------------------------------
# Gate 3: meta_test_false_positive_proof.sh
# ---------------------------------------------------------------------------
echo "=== Gate 3: meta_test_false_positive_proof.sh ==="
if bash "${REPO_ROOT}/scripts/testing/meta_test_false_positive_proof.sh"; then
    pass "meta_test_false_positive_proof.sh"
else
    fail "meta_test_false_positive_proof.sh"
fi
echo ""

# ---------------------------------------------------------------------------
# Submodule cleanliness check
# ---------------------------------------------------------------------------
echo "=== Submodule cleanliness check ==="
SUBMOD_STATUS="$(git -C "${REPO_ROOT}/constitution" status --porcelain 2>/dev/null)"
if [ -z "${SUBMOD_STATUS}" ]; then
    pass "constitution submodule: working tree clean"
else
    fail "constitution submodule: working tree not clean"
    echo "${SUBMOD_STATUS}"
fi
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
