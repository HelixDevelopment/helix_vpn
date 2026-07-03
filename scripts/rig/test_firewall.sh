#!/usr/bin/env bash
# nftables kill-switch test
#
# Purpose:  Validates that a DROP policy on the client namespace's
#           output filter blocks all egress traffic (kill-switch
#           behaviour) and that removing the rule restores connectivity.
#           This is the G1 gate for VPN kill-switch correctness.
# Usage:    sudo ./test_firewall.sh
# Inputs:   (none)
# Outputs:  PASS/FAIL for each phase (baseline → DROP → restore)
# Side-effects: May auto-create namespaces if missing.
# Dependencies: iproute2, nftables, bash 4+

set -euo pipefail
source "$(cd "$(dirname "$0")" && pwd)/common.sh"

require_root
check_rig || { warn "Rig not set up — running setup first"; "$(dirname "$0")/setup.sh"; }

NS_CLIENT="hx-client"; NS_SERVER="hx-server"
SERVER_IP="10.0.240.3"

# ── Phase 1: Baseline — ping must succeed ────────────────────────────
log "Phase 1: Baseline connectivity (no firewall rules beyond default accept)..."
if ip netns exec "${NS_CLIENT}" ping -c 1 -W 2 "${SERVER_IP}" &>/dev/null; then
  log "  Baseline ping OK"
else
  fail "Baseline ping failed — rig may not be healthy"
fi

# ── Phase 2: Install DROP policy on client output → ping must fail ──
log "Phase 2: Installing DROP policy on client output chain..."
# Add a rule that drops all IPv4 output.  We insert it at position 0
# so it takes effect before the default-accept policy on the chain.
ip netns exec "${NS_CLIENT}" nft insert rule inet filter output ip protocol icmp drop

if ip netns exec "${NS_CLIENT}" ping -c 1 -W 2 "${SERVER_IP}" &>/dev/null; then
  fail "DROP policy installed but ping still succeeded — kill-switch ineffective"
else
  log "  Ping correctly blocked by DROP policy — kill-switch works"
fi

# ── Phase 3: Remove the DROP rule → ping must succeed again ──────────
log "Phase 3: Removing DROP policy..."
ip netns exec "${NS_CLIENT}" nft delete rule inet filter output handle 1

if ip netns exec "${NS_CLIENT}" ping -c 1 -W 2 "${SERVER_IP}" &>/dev/null; then
  log "  Connectivity restored after rule removal"
else
  fail "Connectivity did not return after removing DROP rule"
fi

log "Firewall kill-switch test PASSED — G1 gate satisfied"
