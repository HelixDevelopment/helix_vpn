#!/usr/bin/env bash
# Common utilities for the Helix VPN test rig
#
# Purpose:  Shared functions for all rig scripts — logging, root/tool
#           prereq checks, and rig-state detection.
# Usage:    source "$(cd "$(dirname "$0")" && pwd)/common.sh"
# Inputs:   (none; helper library only)
# Outputs:  Exports log/warn/fail/require_root/require_tools/check_rig
# Side-effects: none
# Dependencies: bash 4+, standard POSIX tools (echo, id, grep, command)

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'

log()    { echo -e "${GREEN}[✓]${NC} $*"; }
warn()   { echo -e "${YELLOW}[!]${NC} $*"; }
fail()   { echo -e "${RED}[✗]${NC} $*"; exit 1; }

require_root() {
  [[ $EUID -eq 0 ]] || fail "This script requires root — run with sudo"
}

require_tools() {
  local missing=0
  for tool in "$@"; do
    command -v "$tool" &>/dev/null || { warn "Missing: $tool"; missing=1; }
  done
  [[ $missing -eq 0 ]] || fail "Install missing tools and retry"
}

check_rig() {
  local ns
  for ns in hx-client hx-server hx-bridge; do
    ip netns list | grep -q "$ns" || return 1
  done
  return 0
}
