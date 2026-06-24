#!/usr/bin/env bash
# test_all.sh
# Orchestrator: runs all constitution-inheritance test steps in order.
#
# Steps:
#   1. tests/pre_build_verification.sh       — 5-invariant gate
#   2. tests/test_constitution_inheritance.sh — comprehensive host-side test
#   3. scripts/testing/meta_test_false_positive_proof.sh — anti-bluff mutation proof
#
# Prints a clear per-step result; exits non-zero if any step fails.
# NOTE: Step 3 mutates the constitution submodule transiently and restores it.
#       Run this script only when it is safe to transiently dirty the submodule.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

step_pass=0
step_fail=0

# ---------------------------------------------------------------------------
# run_step <number> <label> <script_path>
# ---------------------------------------------------------------------------
run_step() {
    local num="$1"
    local label="$2"
    local script="$3"

    echo "------------------------------------------------------------"
    echo "Step ${num}: ${label}"
    echo "------------------------------------------------------------"

    if [ ! -f "${script}" ]; then
        echo "STEP ${num} FAIL — script not found: ${script}"
        step_fail=$((step_fail + 1))
        return
    fi

    bash "${script}"
    local exit_code=$?

    echo ""
    if [ "${exit_code}" -eq 0 ]; then
        echo "STEP ${num} RESULT: PASS"
        step_pass=$((step_pass + 1))
    else
        echo "STEP ${num} RESULT: FAIL (exit ${exit_code})"
        step_fail=$((step_fail + 1))
    fi
    echo ""
}

# ---------------------------------------------------------------------------
# Run steps in order (each step always runs — we collect results at the end)
# ---------------------------------------------------------------------------
run_step 1 "Core invariant gate (pre_build_verification.sh)" \
    "${SCRIPT_DIR}/tests/pre_build_verification.sh"

run_step 2 "Comprehensive inheritance test (test_constitution_inheritance.sh)" \
    "${SCRIPT_DIR}/tests/test_constitution_inheritance.sh"

run_step 3 "Anti-bluff mutation proof (meta_test_false_positive_proof.sh)" \
    "${SCRIPT_DIR}/scripts/testing/meta_test_false_positive_proof.sh"

# ---------------------------------------------------------------------------
# Overall summary
# ---------------------------------------------------------------------------
echo "============================================================"
echo "OVERALL RESULTS: ${step_pass} step(s) passed, ${step_fail} step(s) failed."

if [ "${step_fail}" -gt 0 ]; then
    echo "OVERALL STATUS: FAIL"
    exit 1
fi

echo "OVERALL STATUS: PASS"
exit 0
