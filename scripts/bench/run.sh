#!/usr/bin/env bash
# Helix VPN — Benchmark runner
#
# Purpose:  Measure latency, throughput, packet loss, and jitter through the
#           Helix VPN test rig. Records results to a timestamped CSV file.
# Usage:    ./scripts/bench/run.sh [--duration N] [--output DIR] [--server-addr IP]
# Inputs:   DURATION (seconds, default 30)
#           OUTPUT_DIR (path, default ./bench-results/)
#           SERVER_ADDR (IP, default 10.0.240.3)
# Outputs:  Timestamped CSV file under OUTPUT_DIR
# Side-effects: Creates OUTPUT_DIR if it does not exist
# Dependencies: bash 4+, ping, iperf3 (optional), ncat (optional), pv (optional)

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

DURATION=30
OUTPUT_DIR="${SCRIPT_DIR}/../../bench-results"
SERVER_ADDR="10.0.240.3"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --duration) DURATION="$2"; shift 2 ;;
    --output) OUTPUT_DIR="$2"; shift 2 ;;
    --server-addr) SERVER_ADDR="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

mkdir -p "${OUTPUT_DIR}"
CSV_FILE="${OUTPUT_DIR}/bench-$(date +%Y%m%d-%H%M%S).csv"
echo "timestamp,test_type,metric,value,unit" > "${CSV_FILE}"

log_result() {
  local ts; ts=$(date -Iseconds)
  echo "${ts},$1,$2,$3,$4" >> "${CSV_FILE}"
  echo "[$(date +%H:%M:%S)] $1: $2 = $3 $4"
}

cleanup() {
  echo ""
  echo "Cleaning up background processes..."
  jobs -p 2>/dev/null | xargs -r kill 2>/dev/null || true
}

trap cleanup EXIT

# ---------------------------------------------------------------------------
# Latency + packet loss via ping
# ---------------------------------------------------------------------------
latency_test() {
  echo ""
  echo "--- Latency & Packet Loss ---"

  local count=10
  local ping_out
  ping_out=$(ping -c "${count}" -W 2 "${SERVER_ADDR}" 2>&1 || true)

  local loss pkt_rx pkt_tx
  pkt_tx=$(echo "${ping_out}" | grep -oP '\d+(?= packets transmitted)' || echo "0")
  pkt_rx=$(echo "${ping_out}" | grep -oP '\d+(?= (received|packets received))' || echo "0")
  if [[ "${pkt_tx}" -gt 0 ]]; then
    loss=$(echo "scale=1; (${pkt_tx} - ${pkt_rx}) / ${pkt_tx} * 100" | bc -l)
    loss=$(printf "%.1f" "${loss}")
    log_result "packet_loss" "percent" "${loss}" "%"
  else
    log_result "packet_loss" "percent" "FAIL" "%"
    loss="FAIL"
  fi

  local rtt_line
  rtt_line=$(echo "${ping_out}" | grep -oP 'rtt min/avg/max/mdev = [0-9./]+' || true)
  if [[ -n "${rtt_line}" ]]; then
    local rtt_values
    rtt_values=$(echo "${rtt_line}" | sed 's/.*= //')
    local rtt_min rtt_avg rtt_max rtt_mdev
    rtt_min=$(echo "${rtt_values}" | awk -F/ '{print $1}')
    rtt_avg=$(echo "${rtt_values}" | awk -F/ '{print $2}')
    rtt_max=$(echo "${rtt_values}" | awk -F/ '{print $3}')
    rtt_mdev=$(echo "${rtt_values}" | awk -F/ '{print $4}')

    log_result "latency" "avg"  "${rtt_avg}"  "ms"
    log_result "latency" "min"  "${rtt_min}"  "ms"
    log_result "latency" "max"  "${rtt_max}"  "ms"
    log_result "latency" "mdev" "${rtt_mdev}" "ms"
  else
    for m in avg min max mdev; do
      log_result "latency" "${m}" "FAIL" "ms"
    done
  fi
}

# ---------------------------------------------------------------------------
# Throughput via iperf3 (preferred) or ncat + pv (fallback)
# ---------------------------------------------------------------------------
throughput_test() {
  echo ""
  echo "--- Throughput ---"

  if command -v iperf3 &>/dev/null; then
    # TCP throughput test (reverse mode so the namespace-side instance sends)
    local iperf_out
    iperf_out=$(iperf3 -c "${SERVER_ADDR}" -t "${DURATION}" -O 2 -f m 2>&1 || true)

    local bw_line
    bw_line=$(echo "${iperf_out}" | grep -E '^\[SUM\]' | grep -oP '\d+\.?\d*\s*Mbits/sec' | tail -1 || true)
    if [[ -n "${bw_line}" ]]; then
      local bw_val
      bw_val=$(echo "${bw_line}" | awk '{print $1}')
      log_result "throughput" "tcp" "${bw_val}" "Mbps"
    else
      log_result "throughput" "tcp" "FAIL" "Mbps"
    fi

    # UDP throughput + jitter + datagram loss
    local udp_out
    udp_out=$(iperf3 -c "${SERVER_ADDR}" -u -t "$((DURATION / 2))" -b 100M -f m 2>&1 || true)
    local udp_bw jitter dg_loss
    udp_bw=$(echo "${udp_out}" | grep -E '^\[SUM\]' | grep -oP '\d+\.?\d*\s*Mbits/sec' | tail -1 | awk '{print $1}' || echo "FAIL")
    jitter=$(echo "${udp_out}" | grep -oP '\d+\.?\d*\s*ms' | tail -1 | awk '{print $1}' || echo "FAIL")
    dg_loss=$(echo "${udp_out}" | grep -oP '\d+/\d+ \([\d.]+%\)' | tail -1 | grep -oP '[\d.]+%' | tr -d '%' || echo "FAIL")

    log_result "throughput" "udp"   "${udp_bw}"  "Mbps"
    log_result "jitter"     "avg"   "${jitter}"  "ms"
    log_result "datagram_loss" "percent" "${dg_loss}" "%"

  elif command -v ncat &>/dev/null && command -v pv &>/dev/null; then
    # Fallback: ncat + pv throughput estimate
    echo "  iperf3 not found — using ncat + pv fallback"
    local tmp_file
    tmp_file=$(mktemp)
    dd if=/dev/zero bs=1M count=$((DURATION * 10)) of="${tmp_file}" 2>/dev/null &
    local dd_pid=$!

    local pv_out
    pv_out=$(timeout "${DURATION}" pv -b -t -f "${tmp_file}" 2>&1 | ncat -w 3 "${SERVER_ADDR}" 9999 2>/dev/null || true)
    kill "${dd_pid}" 2>/dev/null || true
    rm -f "${tmp_file}"

    if [[ -n "${pv_out}" ]]; then
      log_result "throughput" "tcp_fallback" "${pv_out}" "B/s"
    else
      log_result "throughput" "tcp_fallback" "FAIL" "B/s"
    fi
  else
    echo "  Neither iperf3 nor ncat+pv available — skipping throughput"
    log_result "throughput" "tcp" "SKIP" "Mbps"
  fi
}

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "=== Helix VPN Benchmark Suite ==="
echo "Output: ${CSV_FILE}"
echo "Server: ${SERVER_ADDR}"
echo "Duration: ${DURATION}s per test"
echo ""

if ((${EUID:-0} != 0)); then
  echo "WARNING: Not running as root. Some tests that require namespace"
  echo "         access (throughput via iperf3 to namespaced server) may fail."
  echo ""
fi

latency_test
throughput_test

echo ""
echo "Results saved to ${CSV_FILE}"
echo "=== Benchmark complete: $(date -Iseconds) ==="
