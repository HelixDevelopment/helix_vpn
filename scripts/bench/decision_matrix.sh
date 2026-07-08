#!/usr/bin/env bash
# Helix VPN — G4 decision-matrix filler (HVPN-P0-045)
#
# Purpose:  Read an edge_ab-*.csv produced by scripts/bench/edge_ab.sh and
#           mechanically render the HelixVPN-Phase0-Spike.md §7.3 decision
#           matrix table with the real measured numbers — no hand-waving,
#           no placeholder "?" where a real number exists.
# Usage:    ./scripts/bench/decision_matrix.sh <edge_ab-*.csv>
# Inputs:   One CSV path, schema timestamp,test_type,metric,value,unit
#           (metric = "<edge>.<mode>.c<N>.<submetric>")
# Outputs:  A markdown table on stdout, plus the raw per-edge numbers used
#           to fill each cell (so a reviewer can trace every cell back to
#           its source row).
# Side-effects: none
# Dependencies: bash 4+, awk

set -euo pipefail

CSV="${1:-}"
if [[ -z "${CSV}" || ! -f "${CSV}" ]]; then
  echo "Usage: $0 <edge_ab-*.csv>"
  exit 1
fi

get() {
  # get <metric-exact-string>
  awk -F, -v m="$1" '$3==m {print $4}' "${CSV}" | tail -1
}

max_metric_across_concurrency() {
  # max_metric_across_concurrency <edge> <mode> <submetric>
  local edge="$1" mode="$2" sub="$3"
  awk -F, -v e="${edge}" -v mo="${mode}" -v s="${sub}" '
    $3 ~ ("^"e"\\."mo"\\.c[0-9]+\\."s"$") { print $4 }
  ' "${CSV}" | sort -n | tail -1
}

concurrency_of_max() {
  local edge="$1" mode="$2" sub="$3"
  awk -F, -v e="${edge}" -v mo="${mode}" -v s="${sub}" '
    $3 ~ ("^"e"\\."mo"\\.c[0-9]+\\."s"$") {
      split($3, a, ".c"); split(a[2], b, "."); print $4, b[1]
    }
  ' "${CSV}" | sort -n -k1,1 | tail -1 | awk '{print $2}'
}

for edge in rust go; do
  declare -g "${edge}_peak_sink_mbps=$(max_metric_across_concurrency "${edge}" throughput sink_mbps)"
  declare -g "${edge}_peak_sink_c=$(concurrency_of_max "${edge}" throughput sink_mbps)"
  declare -g "${edge}_cpu_per_gbps_c10=$(get "${edge}.throughput.c10.cpu_per_gbps")"
  declare -g "${edge}_p50_ms=$(get "${edge}.latency.c1.p50_ms")"
  declare -g "${edge}_p99_ms=$(get "${edge}.latency.c1.p99_ms")"
  declare -g "${edge}_handshake_c100_ms=$(get "${edge}.throughput.c100.handshake_setup_ms")"
  declare -g "${edge}_churn_c10=$(get "${edge}.churn.c10.handshakes_per_sec")"
  declare -g "${edge}_churn_peak=$(max_metric_across_concurrency "${edge}" churn handshakes_per_sec)"
  declare -g "${edge}_mem_churn_peak_kb=$(max_metric_across_concurrency "${edge}" churn mem_peak_kb)"
done

echo ""
echo "## §7.3 Decision matrix — filled from ${CSV}"
echo ""
echo "| Criterion | Weight | Rust | Go |"
echo "|---|---|---|---|"
echo "| Reuse / single-impl guarantee | high | ✅ (shares helix-transport byte-for-byte with clients) | ✗ (separate implementation) |"
printf "| MASQUE implementation effort | high | %s | %s |\n" \
  "hand-rolled non-HTTP/3 CONNECT-UDP stand-in (helix-masque/src/connect.rs) — h3 crate judged not reasonably achievable (open datagram/stream-assoc bug, no CONNECT-UDP builder)" \
  "real RFC 9298 CONNECT-UDP via turnkey quic-go + masque-go — genuine HTTP/3, no hand-rolled framing"
printf "| Throughput and CPU/Gbps | high | peak sink %s Mbps @c=%s; %s cores/Gbps @c=10 | peak sink %s Mbps @c=%s; %s cores/Gbps @c=10 |\n" \
  "${rust_peak_sink_mbps}" "${rust_peak_sink_c}" "${rust_cpu_per_gbps_c10}" \
  "${go_peak_sink_mbps}" "${go_peak_sink_c}" "${go_cpu_per_gbps_c10}"
printf "| p99 latency under load | med | idle p99=%sms; handshake-setup @c=100 = %sms | idle p99=%sms; handshake-setup @c=100 = %sms |\n" \
  "${rust_p99_ms}" "${rust_handshake_c100_ms}" "${go_p99_ms}" "${go_handshake_c100_ms}"
echo "| Fits existing Go control plane | med | ✗ | ✅ |"
printf "| Team velocity (proxy: handshakes/sec sustained, higher = cheaper connection churn to serve) | med | %s/s @c=10 (peak %s/s) | %s/s @c=10 (peak %s/s) |\n" \
  "${rust_churn_c10}" "${rust_churn_peak}" "${go_churn_c10}" "${go_churn_peak}"
echo ""
echo "### Supporting raw numbers (traceable to CSV rows above)"
echo ""
echo "| Metric | Rust | Go |"
echo "|---|---|---|"
printf "| Peak sink-measured throughput | %s Mbps @ c=%s | %s Mbps @ c=%s |\n" "${rust_peak_sink_mbps}" "${rust_peak_sink_c}" "${go_peak_sink_mbps}" "${go_peak_sink_c}"
printf "| CPU cost @ c=10 | %s cores/Gbps | %s cores/Gbps |\n" "${rust_cpu_per_gbps_c10}" "${go_cpu_per_gbps_c10}"
printf "| Idle RTT p50 / p99 (loopback) | %s / %s ms | %s / %s ms |\n" "${rust_p50_ms}" "${rust_p99_ms}" "${go_p50_ms}" "${go_p99_ms}"
printf "| Handshake setup time @ c=100 (contended) | %s ms | %s ms |\n" "${rust_handshake_c100_ms}" "${go_handshake_c100_ms}"
printf "| Connection churn @ c=10 (peak) | %s/s (%s/s) | %s/s (%s/s) |\n" "${rust_churn_c10}" "${rust_churn_peak}" "${go_churn_c10}" "${go_churn_peak}"
printf "| Memory peak under churn | %s KiB | %s KiB |\n" "${rust_mem_churn_peak_kb}" "${go_mem_churn_peak_kb}"
echo ""
echo "Honest scope: both edges run loopback-only, real MASQUE-termination +"
echo "gateway-relay data path (no real kernel-WireGuard integration exists yet"
echo "for either edge — see each tool's own doc comments). Loopback RTT numbers"
echo "are NOT representative of real network latency; they measure this"
echo "sandbox's MASQUE-processing + relay overhead only."
