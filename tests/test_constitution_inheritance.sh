#!/usr/bin/env bash
# tests/test_constitution_inheritance.sh
# Comprehensive host-side test for all 5 constitution-inheritance invariants.
# Also asserts:
#   - each parent pointer file is non-empty (size > 0)
#   - the .gitmodules entry for 'constitution' exists
#
# Prints per-invariant PASS/FAIL; exits 0 only if all pass.
# Delegates core invariant checks to pre_build_verification.sh (calls the gate),
# then adds the extra assertions that go beyond the gate.
#
# Usage: bash tests/test_constitution_inheritance.sh
# Paths are resolved from SCRIPT location, not $PWD.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
GATE="${SCRIPT_DIR}/pre_build_verification.sh"

pass_count=0
fail_count=0

# ---------------------------------------------------------------------------
# Logging helpers
# ---------------------------------------------------------------------------
pass() { echo "PASS  $*"; pass_count=$((pass_count + 1)); }
fail() { echo "FAIL  $*"; fail_count=$((fail_count + 1)); }

# ---------------------------------------------------------------------------
# PART 1 — Delegate to the gate (all 5 invariants)
# ---------------------------------------------------------------------------
echo "=== Part 1: Core invariants (via gate) ==="

if [ ! -f "${GATE}" ]; then
    fail "Gate script not found: ${GATE}"
    echo ""
    echo "Results: ${pass_count} passed, ${fail_count} failed."
    echo "STATUS: FAIL"
    exit 1
fi

GATE_OUTPUT="$(bash "${GATE}" 2>&1)"
GATE_EXIT=$?

# Print gate output lines individually, prefixing to show they came from gate
echo "${GATE_OUTPUT}"

if [ "${GATE_EXIT}" -eq 0 ]; then
    pass "Gate (pre_build_verification.sh): all 5 invariants passed"
else
    fail "Gate (pre_build_verification.sh): one or more invariants failed (exit ${GATE_EXIT})"
fi

# ---------------------------------------------------------------------------
# PART 2 — Extra assertions beyond the gate
# ---------------------------------------------------------------------------
echo ""
echo "=== Part 2: Extra assertions ==="

# 2a. Parent pointer files are non-empty
CLAUDE_PARENT="${REPO_ROOT}/CLAUDE.md"
AGENTS_PARENT="${REPO_ROOT}/AGENTS.md"
GUIDE_PARENT="${REPO_ROOT}/docs/guides/HELIX_VPN_CONSTITUTION.md"

for fpath in "${CLAUDE_PARENT}" "${AGENTS_PARENT}" "${GUIDE_PARENT}"; do
    label="${fpath#${REPO_ROOT}/}"
    if [ ! -f "${fpath}" ]; then
        fail "Extra-2a: ${label} — file does not exist"
    elif [ ! -s "${fpath}" ]; then
        fail "Extra-2a: ${label} — file exists but is empty (size = 0)"
    else
        pass "Extra-2a: ${label} is non-empty"
    fi
done

# 2b. .gitmodules entry for 'constitution' exists
GITMODULES="${REPO_ROOT}/.gitmodules"
if [ ! -f "${GITMODULES}" ]; then
    fail "Extra-2b: .gitmodules file not found at ${GITMODULES}"
elif grep -qF '[submodule "constitution"]' "${GITMODULES}"; then
    pass "Extra-2b: .gitmodules contains [submodule \"constitution\"] entry"
else
    fail "Extra-2b: .gitmodules exists but has no [submodule \"constitution\"] entry"
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "Results: ${pass_count} passed, ${fail_count} failed."

if [ "${fail_count}" -gt 0 ]; then
    echo "STATUS: FAIL"
    exit 1
fi

echo "STATUS: PASS"
exit 0
