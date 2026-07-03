#!/usr/bin/env bash
# Helix VPN — Phase 0 one-shot spike
#
# Purpose:  Run a complete Phase 0 verification spike covering prerequisites
#           check, workspace build, test rig setup, reachability, and
#           benchmarks. Designed as the single entry point for `make spike`.
# Usage:    sudo ./scripts/spike.sh [--fast]
#           --fast  Skip benchmarks (prereqs + build + rig only)
# Inputs:   FAST (boolean, default false)
# Outputs:  Console log of each milestone result
#           Benchmark CSV when --fast is not set
# Side-effects: Creates bench-results/ and test rig namespaces
# Dependencies: bash 4+, rustc, go, iperf3 (optional), ping

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

FAST=false
[[ "${1:-}" == "--fast" ]] && FAST=true

PASS=0
FAIL=0
pass()  { PASS=$((PASS + 1)); echo -e "  \033[0;32mPASS\033[0m $*"; }
fail()  { FAIL=$((FAIL + 1)); echo -e "  \033[0;31mFAIL\033[0m $*"; }

echo ""
echo "=============================================="
echo "  Helix VPN — Phase 0 Spike"
echo "=============================================="
echo "Started: $(date -Iseconds)"
echo "Fast mode: ${FAST}"
echo ""

# ----[S0] Prerequisites ------------------------------------------------
echo "---[S0] Prerequisites ---"

if rustc --version &>/dev/null; then
  pass "rustc: $(rustc --version 2>&1)"
else
  fail "rustc: MISSING"
fi

if go version &>/dev/null; then
  pass "go: $(go version 2>&1)"
else
  fail "go: MISSING"
fi

echo "  OS:   $(uname -srm)"

# ----[S1] Build workspace ----------------------------------------------
echo ""
echo "---[S1] Build workspace ---"

HELIX_CORE="${PROJECT_ROOT}/submodules/helix_core"
if [[ -d "${HELIX_CORE}" ]]; then
  if (cd "${HELIX_CORE}" && cargo check --all-targets 2>&1); then
    pass "helix_core: cargo check --all-targets"
  else
    fail "helix_core: cargo check --all-targets"
  fi
else
  fail "helix_core directory not found at ${HELIX_CORE}"
fi

# ----[S2] Test rig ----------------------------------------------------
echo ""
echo "---[S2] Test rig ---"

RIG_DIR="${SCRIPT_DIR}/rig"
if [[ -d "${RIG_DIR}" ]]; then
  if [[ -f "${RIG_DIR}/setup.sh" ]]; then
    if [[ $EUID -eq 0 ]]; then
      if bash "${RIG_DIR}/setup.sh" 2>&1; then
        pass "rig setup"
      else
        fail "rig setup"
      fi
    else
      echo "  SKIP — rig setup requires root (run with sudo)"
    fi
  else
    echo "  SKIP — rig/setup.sh not found"
  fi

  if [[ -f "${RIG_DIR}/test_reach.sh" ]]; then
    if [[ $EUID -eq 0 ]]; then
      # Check if rig is up before testing
      source "${RIG_DIR}/common.sh" 2>/dev/null || true
      if check_rig 2>/dev/null; then
        if bash "${RIG_DIR}/test_reach.sh" 2>&1; then
          pass "rig reachability"
        else
          fail "rig reachability"
        fi
      else
        echo "  SKIP — rig namespaces not present (setup not run)"
      fi
    else
      echo "  SKIP — reachability test requires root"
    fi
  else
    echo "  SKIP — rig/test_reach.sh not found"
  fi
else
  echo "  SKIP — rig directory not found"
fi

# ----[S3] Benchmarks --------------------------------------------------
echo ""
echo "---[S3] Benchmarks ---"

if ! $FAST; then
  if [[ -f "${SCRIPT_DIR}/bench/run.sh" ]]; then
    if bash "${SCRIPT_DIR}/bench/run.sh" --duration 15 2>&1; then
      pass "benchmarks"
    else
      fail "benchmarks"
    fi
  else
    fail "bench/run.sh not found"
  fi
else
  echo "  SKIP — --fast mode enabled"
fi

# ----[S4] Containers submodule check ----------------------------------
echo ""
echo "---[S4] Containers submodule ---"

CONTAINERS_DIR="${PROJECT_ROOT}/submodules/containers"
if [[ -d "${CONTAINERS_DIR}" ]]; then
  echo "  Found at: ${CONTAINERS_DIR}"
  # Check for executable scripts
  exec_count=0
  while IFS= read -r -d '' f; do
    if [[ -x "$f" ]]; then
      exec_count=$((exec_count + 1))
    fi
  done < <(find "${CONTAINERS_DIR}" -name '*.sh' -type f -print0 2>/dev/null)
  echo "  Executable scripts: ${exec_count}"
  if [[ -f "${CONTAINERS_DIR}/bin/boot" ]]; then
    boot_sz=$(stat -c%s "${CONTAINERS_DIR}/bin/boot" 2>/dev/null || echo "?")
    echo "  boot binary: ${boot_sz} bytes"
    if [[ -x "${CONTAINERS_DIR}/bin/boot" ]]; then
      pass "containers/boot is executable"
    else
      fail "containers/boot is NOT executable"
    fi
  fi
  pass "containers submodule present"
else
  fail "containers submodule NOT found"
fi

# ---- Summary ---------------------------------------------------------
echo ""
echo "=============================================="
echo "  Results: ${PASS} pass, ${FAIL} fail"
echo "  Finished: $(date -Iseconds)"
echo "=============================================="

exit "${FAIL}"
