#!/usr/bin/env bash
# G1 Milestone Gate — Plain-UDP transport reachability test (HVPN-P0-025)
#
# Purpose:  Verifies bidirectional UDP data flow through the netns test rig.
#           Runs 100 UDP echo rounds between client and server namespaces,
#           measures round-trip time (mean, min, max, stddev), and checks
#           against PASS/FAIL criteria (mean RTT < 50 ms, loss < 5%).
#           Outputs structured JSON evidence to qa-results/g1/.
# Usage:    sudo ./test_g1.sh
# Inputs:   (none; topology is auto-created if missing)
# Outputs:  PASS/FAIL on stdout; JSON evidence file under qa-results/g1/
# Side-effects: May auto-create namespaces if rig is not up.
# Dependencies: iproute2, python3, jq, bash 4+

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/common.sh"

# ── Configuration ──────────────────────────────────────────────────────────────
EVIDENCE_DIR="${SCRIPT_DIR}/../../qa-results/g1"
TIMESTAMP="$(date +%Y%m%d-%H%M%S)"
EVIDENCE_FILE="${EVIDENCE_DIR}/g1-${TIMESTAMP}.json"
SERVER_PORT=24001
NS_CLIENT="hx-client"
NS_SERVER="hx-server"
SERVER_IP="10.0.240.3"
COUNT=100

# ── Pre-flight ─────────────────────────────────────────────────────────────────
require_root
require_tools ip python3 jq timeout

mkdir -p "${EVIDENCE_DIR}"

if ! check_rig; then
  log "Setting up test rig..."
  bash "${SCRIPT_DIR}/setup.sh"
fi

# ── Phase 1: Start UDP echo server in server namespace ─────────────────────────
log "Starting UDP echo server on ${SERVER_IP}:${SERVER_PORT}..."

ip netns exec "${NS_SERVER}" python3 -c "
import socket, sys
sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.bind(('${SERVER_IP}', ${SERVER_PORT}))
while True:
    try:
        data, addr = sock.recvfrom(1500)
        sock.sendto(data, addr)
    except OSError:
        break
" &
SERVER_PID=$!
sleep 0.5

# Verify server is alive
if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
  fail "UDP echo server failed to start in namespace ${NS_SERVER}"
fi

# ── Phase 2: Run echo client from client namespace ─────────────────────────────
log "Running ${COUNT} UDP echo rounds from ${NS_CLIENT}..."

RESULTS_DIR="$(mktemp -d)"

cleanup() {
  kill "${SERVER_PID}" 2>/dev/null || true
  rm -rf "${RESULTS_DIR}" 2>/dev/null || true
}
trap cleanup EXIT

ip netns exec "${NS_CLIENT}" python3 -c "
import json, socket, time, sys

sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
sock.settimeout(3)

count = ${COUNT}
server_addr = ('${SERVER_IP}', ${SERVER_PORT})
rows = []

for i in range(count):
    payload = f'HelixVPN-G1-Echo-{i}-{time.monotonic_ns()}'.encode()
    start = time.monotonic_ns()
    sock.sendto(payload, server_addr)
    try:
        data, _ = sock.recvfrom(1500)
        end = time.monotonic_ns()
        rtt_ms = (end - start) / 1_000_000.0
        rows.append((i, rtt_ms, 'ok'))
    except socket.timeout:
        rows.append((i, 0.0, 'lost'))

received = [r for r in rows if r[2] == 'ok']
lost_count = count - len(received)
loss_pct = lost_count * 100.0 / count

if received:
    rtt_vals  = [r[1] for r in received]
    mean      = sum(rtt_vals) / len(rtt_vals)
    sq_sum    = sum((v - mean) ** 2 for v in rtt_vals)
    stddev    = (sq_sum / len(rtt_vals)) ** 0.5
    rtt_min   = min(rtt_vals)
    rtt_max   = max(rtt_vals)
else:
    mean = 0.0; stddev = 0.0; rtt_min = 0.0; rtt_max = 0.0

sock.close()

output = {
    'count':      count,
    'sent':       count,
    'received':   len(received),
    'lost':       lost_count,
    'loss_pct':   round(loss_pct, 2),
    'rtt_ms':     {
        'min':    round(rtt_min, 2),
        'max':    round(rtt_max, 2),
        'mean':   round(mean, 2),
        'stddev': round(stddev, 2)
    }
}
with open('${RESULTS_DIR}/results.json', 'w') as f:
    json.dump(output, f)
print(json.dumps(output))
"

# ── Phase 3: Read and analyse results ──────────────────────────────────────────
RESULTS_JSON="$(cat "${RESULTS_DIR}/results.json" 2>/dev/null || echo '{"count":0,"sent":0,"received":0,"lost":0,"loss_pct":100,"rtt_ms":{"min":0,"max":0,"mean":0,"stddev":0}}')"

SENT="$(echo "${RESULTS_JSON}" | jq -r '.sent')"
RECEIVED="$(echo "${RESULTS_JSON}" | jq -r '.received')"
LOST="$(echo "${RESULTS_JSON}" | jq -r '.lost')"
LOSS_PCT="$(echo "${RESULTS_JSON}" | jq -r '.loss_pct')"
MEAN="$(echo "${RESULTS_JSON}" | jq -r '.rtt_ms.mean')"
MIN="$(echo "${RESULTS_JSON}" | jq -r '.rtt_ms.min')"
MAX="$(echo "${RESULTS_JSON}" | jq -r '.rtt_ms.max')"
STDDEV="$(echo "${RESULTS_JSON}" | jq -r '.rtt_ms.stddev')"

# Determine verdict
MAX_LOSS_OK=5.0
MAX_MEAN_RTT_OK=50

if (( $(echo "${LOSS_PCT} >= ${MAX_LOSS_OK}" | bc -l) )) || (( $(echo "${MEAN} >= ${MAX_MEAN_RTT_OK}" | bc -l) )); then
  VERDICT="FAIL"
else
  VERDICT="PASS"
fi

# ── Phase 4: Write evidence JSON ───────────────────────────────────────────────
cat > "${EVIDENCE_FILE}" << EOM
{
  "gate": "G1",
  "test": "Plain-UDP transport reachability (HVPN-P0-025)",
  "timestamp": "${TIMESTAMP}",
  "transport": "plain-udp",
  "topology": {
    "client_ns": "${NS_CLIENT}",
    "server_ns": "${NS_SERVER}",
    "server_addr": "${SERVER_IP}:${SERVER_PORT}"
  },
  "results": {
    "count": ${COUNT},
    "sent": ${SENT},
    "received": ${RECEIVED},
    "lost": ${LOST},
    "loss_pct": ${LOSS_PCT},
    "rtt_ms": {
      "min": ${MIN},
      "max": ${MAX},
      "mean": ${MEAN},
      "stddev": ${STDDEV}
    }
  },
  "verdict": "${VERDICT}",
  "criteria": {
    "max_loss_pct": ${MAX_LOSS_OK},
    "max_mean_rtt_ms": ${MAX_MEAN_RTT_OK}
  }
}
EOM

# ── Phase 5: Report ────────────────────────────────────────────────────────────
if [ "${VERDICT}" = "PASS" ]; then
  log "G1 PASS — loss=${LOSS_PCT}%, mean RTT=${MEAN} ms, min=${MIN} ms, max=${MAX} ms, σ=${STDDEV} ms"
else
  fail "G1 FAIL — loss=${LOSS_PCT}% (max ${MAX_LOSS_OK}%), mean RTT=${MEAN} ms (max ${MAX_MEAN_RTT_OK} ms)"
fi

log "Evidence saved: ${EVIDENCE_FILE}"
