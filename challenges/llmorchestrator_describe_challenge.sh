#!/usr/bin/env bash
# llmorchestrator_describe_challenge.sh — round-275 anti-bluff
# wrapper around the in-process Challenge runner
# (challenges/runner/main.go).
#
# Two-mode behaviour (CONST-050(A) paired-mutation; §1.1):
#
#   normal:    exits 0 only when the runner exits 0 (every invariant
#              passes). Any deviation FAILS.
#
#   mutate:    sets LLMORCH_MUTATE_RUNNER=1 which inverts invariant 2
#              inside the runner. The runner MUST then exit non-zero;
#              this wrapper rewrites that non-zero exit to 99
#              (paired-mutation success). If the runner exits 0 under
#              mutation, this wrapper FAILS — proving the runner
#              actually checks what it claims to check, not a
#              metadata-only PASS.
#
# Verbatim 2026-05-19 operator mandate (preserved per
# CONST-049 §11.4.17):
#   "all existing tests and Challenges do work in anti-bluff
#    manner - they MUST confirm that all tested codebase really
#    works as expected! ..."

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

MODE="${1:-normal}"
echo "=== LLMOrchestrator Describe Challenge (round-275) ==="
echo "  mode=${MODE}"
echo "  module=${MODULE_DIR}"

if ! command -v go >/dev/null 2>&1; then
    echo "SKIP-OK: #env-no-go-toolchain"
    echo "=== Describe Challenge: PASSED (SKIP-OK) ==="
    exit 0
fi

cd "${MODULE_DIR}"

case "${MODE}" in
    normal)
        unset LLMORCH_MUTATE_RUNNER
        out="$(go run ./challenges/runner/ 2>&1)"
        rc=$?
        echo "${out}" | tail -35
        if [[ "${rc}" -ne 0 ]]; then
            echo "=== Describe Challenge: FAILED (runner rc=${rc}) ==="
            exit 1
        fi
        # Belt-and-braces: assert the summary line carries FAIL=0 —
        # defends against an accidental exit-0 with buried FAILs.
        if ! echo "${out}" | grep -q "FAIL=0"; then
            echo "=== Describe Challenge: FAILED (no FAIL=0 line) ==="
            exit 1
        fi
        echo "=== Describe Challenge: PASSED ==="
        exit 0
        ;;
    mutate)
        export LLMORCH_MUTATE_RUNNER=1
        out="$(go run ./challenges/runner/ 2>&1)"
        rc=$?
        echo "${out}" | tail -10
        if [[ "${rc}" -eq 0 ]]; then
            echo "=== Describe Challenge: FAILED " \
                 "(mutation undetected — runner exited 0) ==="
            exit 1
        fi
        echo "=== Describe Challenge: MUTATION DETECTED " \
             "(runner rc=${rc} → exit 99) ==="
        exit 99
        ;;
    *)
        echo "usage: $0 [normal|mutate]"
        exit 2
        ;;
esac
