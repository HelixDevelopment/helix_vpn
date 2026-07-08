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

# ----[S4] Containers submodule — genuine runtime-connectivity check ---
#
# HVPN-P0-080 investigation (full reasoning in scripts/spike/containers_check
# /main.go's doc comment + the task's final report): the containers
# submodule orchestrates OCI containers (Docker/Podman/K8s — image pull,
# compose up, health-check, lifecycle). It has NO concept of Linux network
# namespaces, veth pairs, bridges, nftables, or tc netem — the exact kernel
# primitives HelixVPN-Phase0-Spike.md §3's rig topology needs. Forcing
# container-boot into that netns/veth/bridge/nftables/tc creation would
# misuse the submodule outside its design for zero real benefit.
# scripts/rig/*.sh (ip netns + veth + bridge + nftables + tc) therefore
# REMAINS the correct, sufficient tool for the rig's own topology — this
# step does not replace or wrap it. What genuinely fits the submodule's
# design is a FUTURE containerized stand-in for the connector-site LAN
# service (§3's "hello page" host, today an ad-hoc `python3 -m
# http.server`); this step proves that foundation — real runtime
# auto-detection + a live query against this host's actual container
# engine — is genuinely working, ahead of that future rig extension.
echo ""
echo "---[S4] Containers submodule (runtime-connectivity check) ---"

CONTAINERS_DIR="${PROJECT_ROOT}/submodules/containers"
CONTAINERS_CHECK_DIR="${SCRIPT_DIR}/spike/containers_check"
if [[ -d "${CONTAINERS_DIR}" ]]; then
  echo "  Found at: ${CONTAINERS_DIR}"
  pass "containers submodule present"

  if [[ -d "${CONTAINERS_CHECK_DIR}" ]] && go version &>/dev/null; then
    if (cd "${CONTAINERS_CHECK_DIR}" && go build -o containers_check . 2>&1); then
      if "${CONTAINERS_CHECK_DIR}/containers_check" 2>&1; then
        pass "containers submodule: runtime auto-detected + queried live"
      else
        fail "containers submodule: runtime-connectivity check reported FAIL (see output above — e.g. no container runtime installed on this host)"
      fi
    else
      fail "containers submodule: containers_check build failed"
    fi
  else
    echo "  SKIP — scripts/spike/containers_check not found or 'go' unavailable"
  fi
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
