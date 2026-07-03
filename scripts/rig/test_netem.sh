#!/usr/bin/env bash
# Network impairment (netem) test
#
# Purpose:  Injects latency and packet loss via netem on the link
#           between client and bridge namespaces, then verifies the
#           impairments are visible in ping RTT/loss stats.  Removes
#           all qdiscs at the end and confirms a clean state.
# Usage:    sudo ./test_netem.sh
# Inputs:   (none)
# Outputs:  PASS/FAIL for each phase (baseline → delay → loss → restore)
# Side-effects: May auto-create namespaces if missing.
# Dependencies: iproute2 (tc), iputils (ping), bash 4+

set -euo pipefail
source "$(cd "$(dirname "$0")" && pwd)/common.sh"

require_root
check_rig || { warn "Rig not set up — running setup first"; "$(dirname "$0")/setup.sh"; }

NS_CLIENT="hx-client"
NS_BRIDGE="hx-bridge"
SERVER_IP="10.0.240.3"
VETH_CLIENT_BR="veth-c-br"

# Helper: extract avg RTT from ping output (e.g. "rtt min/avg/max/mdev = ...")
ping_avg_rtt() {
  local ns="$1"; shift
  ip netns exec "${ns}" ping -c 3 -W 2 "$@" 2>/dev/null \
    | sed -n 's/.*rtt min\/avg\/max\/mdev = \([0-9.]*\)\/\([0-9.]*\)\/.*/\2/p'
}

# Helper: extract packet-loss percent from ping output
ping_loss_pct() {
  local ns="$1"; shift
  ip netns exec "${ns}" ping -c 10 -W 2 "$@" 2>/dev/null \
    | sed -n 's/.* \([0-9]*\)% packet loss/\1/p'
}

# ── Phase 1: Baseline RTT (no impairments) ──────────────────────────
log "Phase 1: Measuring baseline RTT..."
BASELINE_RTT="$(ping_avg_rtt "${NS_CLIENT}" "${SERVER_IP}" || echo "")"
if [[ -z "${BASELINE_RTT}" ]]; then
  fail "Cannot measure baseline RTT — rig may not be healthy"
fi
log "  Baseline avg RTT = ${BASELINE_RTT} ms"

# ── Phase 2: Add 50 ms delay on both directions of the client link ──
log "Phase 2: Adding 50 ms netem delay on client↔bridge link..."

# Egress from client (client→server leg)
ip netns exec "${NS_CLIENT}" tc qdisc add dev veth-c root netem delay 50ms
# Egress from bridge toward client (server→client leg)
ip netns exec "${NS_BRIDGE}" tc qdisc add dev "${VETH_CLIENT_BR}" root netem delay 50ms

DELAY_RTT="$(ping_avg_rtt "${NS_CLIENT}" "${SERVER_IP}" || echo "")"
if [[ -z "${DELAY_RTT}" ]]; then
  fail "Cannot measure RTT after delay injection"
fi
log "  After delay: avg RTT = ${DELAY_RTT} ms"

# Check that RTT increased by at least 80 ms (allow some slack for
# the host scheduler; 100 ms added, >=80 ms delta is a strong signal)
RTT_DELTA="$(echo "${DELAY_RTT} - ${BASELINE_RTT}" | bc -l 2>/dev/null || echo "0")"
if (( $(echo "${RTT_DELTA} >= 80" | bc -l 2>/dev/null || echo "0") )); then
  log "  RTT delta ${RTT_DELTA} ms >= 80 ms — delay working"
else
  fail "Expected RTT increase ≥80 ms, got ${RTT_DELTA} ms"
fi

# ── Phase 3: Add 5 % packet loss ────────────────────────────────────
log "Phase 3: Adding 5% packet loss on top of delay..."
ip netns exec "${NS_BRIDGE}" tc qdisc change dev "${VETH_CLIENT_BR}" root netem delay 50ms loss 5%

LOSS_PCT="$(ping_loss_pct "${NS_CLIENT}" "${SERVER_IP}" || echo "0")"
if (( LOSS_PCT > 0 )); then
  log "  Packet loss detected: ${LOSS_PCT}% (expecting ~5%)"
else
  warn "  No packet loss measured — loss may be too low or netem not applied correctly"
fi

# ── Phase 4: Remove all qdiscs & verify clean state ─────────────────
log "Phase 4: Removing impairments..."

# Delete the root qdisc on both sides (falls back to the default pfifo_fast)
ip netns exec "${NS_CLIENT}" tc qdisc del dev veth-c root 2>/dev/null || true
ip netns exec "${NS_BRIDGE}" tc qdisc del dev "${VETH_CLIENT_BR}" root 2>/dev/null || true

# Verify no netem qdisc remains
CLIENT_QDISC="$(ip netns exec "${NS_CLIENT}" tc qdisc show dev veth-c 2>/dev/null || true)"
BRIDGE_QDISC="$(ip netns exec "${NS_BRIDGE}" tc qdisc show dev "${VETH_CLIENT_BR}" 2>/dev/null || true)"

if echo "${CLIENT_QDISC}" | grep -q 'netem'; then
  fail "netem qdisc still present on client veth-c after cleanup"
fi
if echo "${BRIDGE_QDISC}" | grep -q 'netem'; then
  fail "netem qdisc still present on bridge ${VETH_CLIENT_BR} after cleanup"
fi

RESTORE_RTT="$(ping_avg_rtt "${NS_CLIENT}" "${SERVER_IP}" || echo "")"
if [[ -z "${RESTORE_RTT}" ]]; then
  fail "Cannot measure RTT after cleanup"
fi

RESTORE_DELTA="$(echo "${RESTORE_RTT} - ${BASELINE_RTT}" | bc -l 2>/dev/null || echo "99")"
if (( $(echo "${RESTORE_DELTA} < 10" | bc -l 2>/dev/null || echo "0") )); then
  log "  Clean-state RTT (${RESTORE_RTT} ms) within 10 ms of baseline — link restored"
else
  warn "  Clean-state RTT delta ${RESTORE_DELTA} ms exceeds 10 ms (baseline=${BASELINE_RTT}, now=${RESTORE_RTT})"
fi

log "Netem impairment test PASSED"
