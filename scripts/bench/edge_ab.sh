#!/usr/bin/env bash
# Helix VPN — G4 edge A/B benchmark (HVPN-P0-045)
#
# Purpose:  Drive BOTH MASQUE-termination edges (Rust helix-edge and Go
#           go-edge) through the identical measurement protocol from
#           HelixVPN-Phase0-Spike.md §7.2 (throughput at 1/10/100
#           concurrent flows, CPU-per-Gbps, p50/p99 added latency,
#           connection-churn handshakes/sec, memory-under-churn) and
#           append the results as CSV rows so the §7.3 decision matrix
#           can be filled mechanically from real, captured numbers.
# Usage:    ./scripts/bench/edge_ab.sh [--out-csv PATH] [--duration-secs N]
#                                      [--concurrencies "1 10 100"]
#                                      [--payload-bytes N] [--skip-build]
# Inputs:   OUT_CSV (path, default ./bench-results/edge_ab-<ts>.csv)
#           DURATION_SECS (per-test seconds, default 5)
#           CONCURRENCIES (space-separated list, default "1 10 100")
#           PAYLOAD_BYTES (default 1200 — MTU-sized WG-shaped datagram)
# Outputs:  CSV rows appended to OUT_CSV in the SAME schema
#           scripts/bench/run.sh uses: timestamp,test_type,metric,value,unit
#           metric is "<edge>.<mode>.c<N>.<submetric>" — see README.md.
# Side-effects: Builds scripts/bench/tools/{rust_edge_bench,go_edge_bench}
#               (cargo build --release / go build); spawns + kills
#               short-lived loopback server processes for each edge.
# Dependencies: cargo, go, /proc (Linux), bash 4+, getconf, bc
#
# # Honest scope (read before trusting a number this script produces)
#
# Both edges are Phase-0 spikes with NO real kernel-WireGuard/boringtun
# gateway-socket integration yet (confirmed directly from each edge's own
# source/README — see rust_edge_bench's and go_edge_bench's own doc
# comments). This script therefore benchmarks the MASQUE termination +
# gateway-relay hand-off data path itself (real QUIC/TLS handshake, real
# CONNECT-UDP-flow establishment, real datagram relay to a real loopback
# UDP sink) — NOT an end-to-end WireGuard tunnel, because that full slice
# does not exist yet for either edge, independent of any sandbox
# constraint. Everything runs on 127.0.0.1 — no root, no iperf3, no real
# kernel WG needed for THIS specific benchmark (unlike the netns rig in
# scripts/rig/, which does need root).
#
# Throughput is reported from TWO angles: the client's own "offered"
# rate (informational — a QUIC unreliable-datagram sender's local
# accept-into-queue rate, NOT delivery proof) and the edge's own
# loopback sink's actually-received byte count (authoritative —
# positive sink-side evidence of real relayed goodput). Use the sink
# number for the decision matrix.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
TOOLS_DIR="${SCRIPT_DIR}/tools"
RUST_DIR="${TOOLS_DIR}/rust_edge_bench"
GO_DIR="${TOOLS_DIR}/go_edge_bench"

OUT_CSV=""
DURATION_SECS=5
CONCURRENCIES="1 10 100"
PAYLOAD_BYTES=1200
SKIP_BUILD=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out-csv) OUT_CSV="$2"; shift 2 ;;
    --duration-secs) DURATION_SECS="$2"; shift 2 ;;
    --concurrencies) CONCURRENCIES="$2"; shift 2 ;;
    --payload-bytes) PAYLOAD_BYTES="$2"; shift 2 ;;
    --skip-build) SKIP_BUILD=true; shift ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ -z "${OUT_CSV}" ]]; then
  OUT_DIR="${SCRIPT_DIR}/../../bench-results"
  mkdir -p "${OUT_DIR}"
  OUT_CSV="${OUT_DIR}/edge_ab-$(date +%Y%m%d-%H%M%S).csv"
fi
mkdir -p "$(dirname "${OUT_CSV}")"
if [[ ! -f "${OUT_CSV}" ]]; then
  echo "timestamp,test_type,metric,value,unit" > "${OUT_CSV}"
fi

WORKDIR="$(mktemp -d)"
trap 'kill_all; rm -rf "${WORKDIR}"' EXIT

SERVER_PID=""
kill_all() {
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  SERVER_PID=""
}

log() { echo "[edge_ab] $*"; }

emit() {
  # emit <edge> <mode> <concurrency> <submetric> <value> <unit>
  local edge="$1" mode="$2" conc="$3" sub="$4" value="$5" unit="$6"
  local ts; ts=$(date -Iseconds)
  echo "${ts},edge_ab,${edge}.${mode}.c${conc}.${sub},${value},${unit}" >> "${OUT_CSV}"
  echo "  [$(date +%H:%M:%S)] ${edge}.${mode}.c${conc}.${sub} = ${value} ${unit}"
}

CLK_TCK="$(getconf CLK_TCK)"

proc_rss_kb() {
  local pid="$1"
  awk '/VmRSS/{print $2}' "/proc/${pid}/status" 2>/dev/null || echo 0
}

proc_cpu_ticks() {
  local pid="$1"
  # fields 14 (utime) + 15 (stime), per proc(5)
  awk '{print $14+$15}' "/proc/${pid}/stat" 2>/dev/null || echo 0
}

# Samples RSS every 200ms into $1 (a file) until $2 (a "stop" sentinel
# file) appears. Run in the background; the caller `wait`s on its PID
# and then reads the max value from the samples file.
sample_rss_bg() {
  local pid="$1" samples_file="$2" stop_file="$3"
  : > "${samples_file}"
  while [[ ! -e "${stop_file}" ]]; do
    proc_rss_kb "${pid}" >> "${samples_file}"
    sleep 0.2
  done
}

max_of_file() {
  sort -n "$1" 2>/dev/null | tail -1 || echo 0
}

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
build_rust() {
  log "Building rust_edge_bench (cargo build --release)..."
  (cd "${RUST_DIR}" && cargo build --release 2>&1 | tail -5)
}

build_go() {
  log "Building go_edge_bench (go build)..."
  (cd "${GO_DIR}" && go build -o "${WORKDIR}/go_edge_bench" . 2>&1 | tail -5)
}

RUST_BIN="${RUST_DIR}/target/release/rust_edge_bench"
GO_BIN="${WORKDIR}/go_edge_bench"

if ! $SKIP_BUILD; then
  build_rust
  build_go
else
  log "--skip-build: expecting pre-built binaries"
  GO_BIN="${GO_DIR}/go_edge_bench"
fi

# ---------------------------------------------------------------------------
# Rust edge driver
# ---------------------------------------------------------------------------
bench_rust() {
  local cert="${WORKDIR}/rust.der" log_file="${WORKDIR}/rust_server.log"
  log "Starting Rust helix-edge server role..."
  "${RUST_BIN}" --role server --bind-ip 127.0.0.1 --sni-host edge-bench.invalid \
    --cert-out "${cert}" > "${log_file}" 2>&1 &
  SERVER_PID=$!

  local waited=0
  while [[ ! -s "${log_file}" ]] && (( waited < 100 )); do sleep 0.1; waited=$((waited+1)); done
  if ! grep -q '^READY' "${log_file}"; then
    log "FAIL: rust edge server did not print READY within 10s"; cat "${log_file}" >&2
    kill_all; return 1
  fi
  local ready_line edge_addr echo_sink count_sink
  ready_line="$(grep '^READY' "${log_file}")"
  edge_addr="$(echo "${ready_line}"  | grep -oP 'edge=\K\S+')"
  echo_sink="$(echo "${ready_line}"  | grep -oP 'echo_sink=\K\S+')"
  count_sink="$(echo "${ready_line}" | grep -oP 'count_sink=\K\S+')"
  log "Rust edge ready: edge=${edge_addr} echo_sink=${echo_sink} count_sink=${count_sink} pid=${SERVER_PID}"

  run_edge_matrix "rust" "${RUST_BIN}" "${edge_addr}" "${echo_sink}" "${count_sink}" \
    "--cert ${cert} --sni-host edge-bench.invalid"

  kill_all
}

# ---------------------------------------------------------------------------
# Go edge driver
# ---------------------------------------------------------------------------
bench_go() {
  local cert="${WORKDIR}/go.der" log_file="${WORKDIR}/go_server.log"
  log "Starting Go go-edge server role..."
  "${GO_BIN}" --role server --bind-ip 127.0.0.1 --cert-out "${cert}" > "${log_file}" 2>&1 &
  SERVER_PID=$!

  local waited=0
  while [[ ! -s "${log_file}" ]] && (( waited < 100 )); do sleep 0.1; waited=$((waited+1)); done
  if ! grep -q '^READY' "${log_file}"; then
    log "FAIL: go edge server did not print READY within 10s"; cat "${log_file}" >&2
    kill_all; return 1
  fi
  local ready_line edge_addr echo_sink count_sink
  ready_line="$(grep '^READY' "${log_file}")"
  edge_addr="$(echo "${ready_line}"  | grep -oP 'edge=\K\S+')"
  echo_sink="$(echo "${ready_line}"  | grep -oP 'echo_sink=\K\S+')"
  count_sink="$(echo "${ready_line}" | grep -oP 'count_sink=\K\S+')"
  log "Go edge ready: edge=${edge_addr} echo_sink=${echo_sink} count_sink=${count_sink} pid=${SERVER_PID}"

  run_edge_matrix "go" "${GO_BIN}" "${edge_addr}" "${echo_sink}" "${count_sink}" \
    "--cert ${cert}"

  kill_all
}

# ---------------------------------------------------------------------------
# Shared measurement matrix (same protocol for both edges — §7.2)
# ---------------------------------------------------------------------------
run_edge_matrix() {
  local edge="$1" bin="$2" edge_addr="$3" echo_sink="$4" count_sink="$5" extra_args="$6"
  local server_log="${WORKDIR}/${edge}_server.log"

  # --- Throughput + CPU-per-Gbps + memory, at 1/10/100 concurrent flows ---
  for c in ${CONCURRENCIES}; do
    log "[${edge}] throughput c=${c} duration=${DURATION_SECS}s payload=${PAYLOAD_BYTES}B"
    local cpu0 cpu1 rss_samples="${WORKDIR}/${edge}_rss_${c}.samples" stop_file="${WORKDIR}/${edge}_rss_${c}.stop"
    rm -f "${stop_file}"
    cpu0="$(proc_cpu_ticks "${SERVER_PID}")"
    sample_rss_bg "${SERVER_PID}" "${rss_samples}" "${stop_file}" &
    local sampler_pid=$!

    local before_bytes after_bytes t_start t_end
    before_bytes="$(grep COUNT_STATS "${server_log}" 2>/dev/null | tail -1 | grep -oP 'bytes=\K[0-9]+' || echo 0)"
    t_start=$(date +%s.%N)

    # shellcheck disable=SC2086
    local client_out
    client_out="$("${bin}" --role client --edge-addr "${edge_addr}" --target-addr "${count_sink}" \
      ${extra_args} --mode throughput --concurrency "${c}" \
      --duration-secs "${DURATION_SECS}" --payload-bytes "${PAYLOAD_BYTES}" 2>&1 || true)"

    t_end=$(date +%s.%N)
    cpu1="$(proc_cpu_ticks "${SERVER_PID}")"
    touch "${stop_file}"; wait "${sampler_pid}" 2>/dev/null || true

    after_bytes="$(grep COUNT_STATS "${server_log}" 2>/dev/null | tail -1 | grep -oP 'bytes=\K[0-9]+' || echo 0)"
    local elapsed sink_bytes sink_mbps cpu_secs cpu_per_gbps mem_peak_kb
    elapsed="$(echo "${t_end} - ${t_start}" | bc -l)"
    sink_bytes=$((after_bytes - before_bytes))
    sink_mbps="$(echo "scale=4; (${sink_bytes} * 8) / ${elapsed} / 1000000" | bc -l)"
    cpu_secs="$(echo "scale=4; (${cpu1} - ${cpu0}) / ${CLK_TCK}" | bc -l)"
    if (( $(echo "${sink_mbps} > 0" | bc -l) )); then
      cpu_per_gbps="$(echo "scale=6; (${cpu_secs} / ${elapsed}) / (${sink_mbps} / 1000)" | bc -l)"
    else
      cpu_per_gbps="0"
    fi
    mem_peak_kb="$(max_of_file "${rss_samples}")"

    local handshake_ms offered_mbps
    handshake_ms="$(echo "${client_out}" | grep '^CSV,' | grep 'handshake_setup_ms' | awk -F, '{print $6}' | tail -1)"
    offered_mbps="$(echo "${client_out}" | grep '^CSV,' | grep 'client_offered_mbps' | awk -F, '{print $6}' | tail -1)"

    emit "${edge}" throughput "${c}" sink_mbps "${sink_mbps}" Mbps
    emit "${edge}" throughput "${c}" client_offered_mbps "${offered_mbps:-0}" Mbps
    emit "${edge}" throughput "${c}" cpu_seconds "${cpu_secs}" cpu_seconds
    emit "${edge}" throughput "${c}" cpu_per_gbps "${cpu_per_gbps}" cores_per_gbps
    emit "${edge}" throughput "${c}" mem_peak_kb "${mem_peak_kb:-0}" KiB
    emit "${edge}" throughput "${c}" handshake_setup_ms "${handshake_ms:-0}" ms
  done

  # --- Connection churn (handshakes/sec) + memory-under-churn ---
  for c in ${CONCURRENCIES}; do
    log "[${edge}] churn c=${c} duration=${DURATION_SECS}s"
    local rss_samples="${WORKDIR}/${edge}_churn_rss_${c}.samples" stop_file="${WORKDIR}/${edge}_churn_rss_${c}.stop"
    rm -f "${stop_file}"
    sample_rss_bg "${SERVER_PID}" "${rss_samples}" "${stop_file}" &
    local sampler_pid=$!

    # shellcheck disable=SC2086
    local client_out
    client_out="$("${bin}" --role client --edge-addr "${edge_addr}" --target-addr "${echo_sink}" \
      ${extra_args} --mode churn --concurrency "${c}" --duration-secs "${DURATION_SECS}" 2>&1 || true)"

    touch "${stop_file}"; wait "${sampler_pid}" 2>/dev/null || true
    local mem_peak_kb hps failed
    mem_peak_kb="$(max_of_file "${rss_samples}")"
    hps="$(echo "${client_out}" | grep '^CSV,' | grep 'handshakes_per_sec' | awk -F, '{print $6}' | tail -1)"
    failed="$(echo "${client_out}" | grep '^CSV,' | grep 'failed_handshakes' | awk -F, '{print $6}' | tail -1)"

    emit "${edge}" churn "${c}" handshakes_per_sec "${hps:-0}" per_sec
    emit "${edge}" churn "${c}" failed_handshakes "${failed:-0}" count
    emit "${edge}" churn "${c}" mem_peak_kb "${mem_peak_kb:-0}" KiB
  done

  # --- Added latency (p50/p99), single idle flow ---
  log "[${edge}] latency (idle single flow, 200 pings)"
  # shellcheck disable=SC2086
  local client_out
  client_out="$("${bin}" --role client --edge-addr "${edge_addr}" --target-addr "${echo_sink}" \
    ${extra_args} --mode latency --payload-bytes 200 2>&1 || true)"
  local p50 p99
  p50="$(echo "${client_out}" | grep '^CSV,' | grep 'p50_ms' | awk -F, '{print $6}' | tail -1)"
  p99="$(echo "${client_out}" | grep '^CSV,' | grep 'p99_ms' | awk -F, '{print $6}' | tail -1)"
  emit "${edge}" latency 1 p50_ms "${p50:-0}" ms
  emit "${edge}" latency 1 p99_ms "${p99:-0}" ms
}

echo ""
echo "=== Helix VPN — G4 edge A/B benchmark (HVPN-P0-045) ==="
echo "Output CSV: ${OUT_CSV}"
echo "Duration per test: ${DURATION_SECS}s   Concurrencies: ${CONCURRENCIES}   Payload: ${PAYLOAD_BYTES}B"
echo ""

bench_rust
# server_log path used inside run_edge_matrix must match; rebind for go run.
bench_go

echo ""
echo "Edge A/B results written to ${OUT_CSV}"
echo "=== Edge A/B benchmark complete: $(date -Iseconds) ==="
