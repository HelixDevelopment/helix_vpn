#!/usr/bin/env bash
# Helix VPN — Unified G1/G2/G4 measurement harness (HVPN-P0-077)
#
# Purpose:  HelixVPN-Phase0-Spike.md §8 calls for "one harness, run for
#           every transport x edge combination, producing a comparable
#           CSV" against the pass-bar table (through-tunnel throughput,
#           added latency, loss resilience, handshake time, reconnect/
#           roam, edge CPU/Gbps, core RSS, wire fingerprint) — for gates
#           G1 (plain-UDP baseline), G2 (MASQUE through a DPI-style
#           block), and G4 (Rust-vs-Go edge A/B). This script is that
#           harness: it does NOT reimplement any transport/probe/edge
#           logic — it drives the REAL, already-existing Phase-0 tools
#           (helix_core's `g2-dpi-probe` binary + its own
#           `g2_dpi_masque_unpriv.sh` sandboxed rig, and this repo's own
#           `edge_ab.sh`) and normalizes their JSON/CSV output into ONE
#           CSV schema so every gate's numbers land in the same table.
# Usage:    ./scripts/bench/unified_harness.sh [--out-csv PATH]
#               [--skip-g1] [--skip-g2] [--skip-g4] [--skip-build]
#               [--overhead-rounds N] [--overhead-chunk-bytes N]
#               [--edge-duration-secs N] [--edge-concurrencies "1 10"]
#               [--skip-loss]
# Inputs:   OUT_CSV (path, default ./bench-results/unified-<ts>.csv)
#           OVERHEAD_ROUNDS (default 20000 — enough wall-clock for a
#             handful of 0.2s-spaced RSS samples, still sub-second)
#           OVERHEAD_CHUNK_BYTES (default 1200 — MTU-sized, matches §8/G2)
#           EDGE_DURATION_SECS / EDGE_CONCURRENCIES — passed through to
#             edge_ab.sh, kept small by default so this harness stays a
#             few tens of seconds, not minutes (see edge_ab.sh directly
#             for the full 1/10/100 sweep already used for the G4
#             decision-log row).
#           SKIP_LOSS — passed through to helix_core's
#             g2_dpi_masque_unpriv.sh (skips its tc-netem loss-resilience
#             phase, which needs an extra ~12s).
# Outputs:  ONE CSV at OUT_CSV, schema:
#             timestamp,gate,transport,edge,metric,value,unit,pass_bar,
#             verdict,method,note
#           verdict is one of: PASS, FAIL, RECORDED, NOT_APPLICABLE,
#           NOT_MEASURED, SKIP, UNMEASURED_VS_BAR (see README.md for the
#           full vocabulary + what each one honestly means).
# Side-effects: Builds submodules/helix_core's `g2-dpi-probe` release
#               bin (cargo) unless --skip-build. Builds+runs
#               scripts/bench/tools/{rust_edge_bench,go_edge_bench} via
#               edge_ab.sh (unmodified). Runs helix_core's
#               scripts/spike/g2_dpi_masque_unpriv.sh UNMODIFIED, which
#               creates one throwaway `unshare --net --user` namespace
#               for its own duration (no host network state touched).
#               Writes JSON/JSONL evidence under
#               submodules/helix_core/target/spike-evidence/g2/<run-id>/
#               (that submodule's own gitignored build-artifact dir —
#               nothing is written outside it by this harness).
# Dependencies: bash 4+, cargo, jq, python3, bc, /proc (Linux), unshare
#               (util-linux), nft (nftables) — all already required by
#               the tools this harness drives; see each tool's own
#               doc-comment for specifics.
#
# # Honest scope (read before trusting a number this script produces)
#
# This Phase-0 codebase has NO real end-to-end WireGuard dataplane wired
# up yet (no TUN device, no client-gateway-connector process chain up
# simultaneously) — only individual crate-level tests and probe
# binaries exist (confirmed directly: `grep`/`find` across
# submodules/helix_core, submodules/helix_edge, submodules/helix_go
# turned up no runnable "bring up a real tunnel and route traffic
# through it" entry point outside the netns rig, which itself needs
# root not available in this sandbox — see scripts/rig/common.sh's
# `require_root`). So "through-tunnel throughput" and "added latency"
# for G1/G2 in this harness are loopback TRANSPORT-PRIMITIVE numbers
# (the same code the future tunnel will carry WG datagrams over), not
# actual tunnel measurements — every row says so explicitly in its
# `note` column, per this project's anti-bluff discipline (no silent
# relabeling of a proxy as the real thing). `iperf3`, `tshark`, and
# `tcpdump` are absent from this sandbox (confirmed via `command -v` /
# `find`); the loss-resilience, throughput, and wire-fingerprint
# numbers come from hand-rolled stand-ins the prior G2 work already
# built and this script REUSES rather than reinvents (real
# `AF_PACKET` sniffer, real `nft`-in-`unshare` DPI block, real paced
# offered-load goodput comparison) — see
# submodules/helix_core/G2-RESULTS.md for the full methodology
# writeup those stand-ins were validated against.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
HELIX_CORE_DIR="${ROOT_DIR}/submodules/helix_core"
G2_UNPRIV_SCRIPT="${HELIX_CORE_DIR}/scripts/spike/g2_dpi_masque_unpriv.sh"

OUT_CSV=""
SKIP_G1=false
SKIP_G2=false
SKIP_G4=false
SKIP_BUILD=false
SKIP_LOSS=false
OVERHEAD_ROUNDS=20000
OVERHEAD_CHUNK_BYTES=1200
EDGE_DURATION_SECS=2
EDGE_CONCURRENCIES="1 10"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out-csv) OUT_CSV="$2"; shift 2 ;;
    --skip-g1) SKIP_G1=true; shift ;;
    --skip-g2) SKIP_G2=true; shift ;;
    --skip-g4) SKIP_G4=true; shift ;;
    --skip-build) SKIP_BUILD=true; shift ;;
    --skip-loss) SKIP_LOSS=true; shift ;;
    --overhead-rounds) OVERHEAD_ROUNDS="$2"; shift 2 ;;
    --overhead-chunk-bytes) OVERHEAD_CHUNK_BYTES="$2"; shift 2 ;;
    --edge-duration-secs) EDGE_DURATION_SECS="$2"; shift 2 ;;
    --edge-concurrencies) EDGE_CONCURRENCIES="$2"; shift 2 ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

if [[ -z "${OUT_CSV}" ]]; then
  OUT_DIR="${ROOT_DIR}/bench-results"
  mkdir -p "${OUT_DIR}"
  OUT_CSV="${OUT_DIR}/unified-$(date +%Y%m%d-%H%M%S).csv"
fi
mkdir -p "$(dirname "${OUT_CSV}")"
echo "timestamp,gate,transport,edge,metric,value,unit,pass_bar,verdict,method,note" > "${OUT_CSV}"

WORKDIR="$(mktemp -d)"
trap 'rm -rf "${WORKDIR}"' EXIT

log() { echo "[unified_harness] $*" >&2; }

# sanitize a free-text field for a single CSV cell: commas -> semicolons,
# newlines -> spaces, so the schema's 11 columns never shift.
san() { printf '%s' "$1" | tr ',\n' ';\ ' | tr -s ' '; }

emit() {
  # emit <gate> <transport> <edge> <metric> <value> <unit> <pass_bar> <verdict> <method> <note>
  local gate="$1" transport="$2" edge="$3" metric="$4" value="$5" unit="$6"
  local pass_bar="$7" verdict="$8" method="$9" note="${10}"
  local ts; ts=$(date -Iseconds)
  echo "${ts},${gate},${transport},${edge},${metric},${value},${unit},$(san "${pass_bar}"),${verdict},$(san "${method}"),$(san "${note}")" >> "${OUT_CSV}"
  echo "  [$(date +%H:%M:%S)] ${gate}.${transport}.${edge}.${metric} = ${value} ${unit} [${verdict}]"
}

NO_TUNNEL_NOTE="NOT a through-tunnel measurement: no real end-to-end WG dataplane (TUN + encrypt/decrypt + routed peer) is wired up yet anywhere in this Phase-0 codebase (confirmed: only crate-level tests/probe binaries exist; the netns rig that WOULD carry a real tunnel needs root, unavailable in this sandbox) -- this is the closest sandboxed proxy: the SAME transport-primitive code the future tunnel will carry WG datagrams over, measured directly on loopback."
NO_ROAM_NOTE="SKIPPED, not silently omitted: reconnect/roam requires a real up tunnel + a flappable interface, neither of which exists yet in this Phase-0 codebase (no client-gateway-connector process chain is wired end-to-end). Tracked as Phase-0 follow-up alongside the S1/S2 milestones."

# ---------------------------------------------------------------------------
# RSS sampler (same idiom as edge_ab.sh's sample_rss_bg/max_of_file, kept
# local to this script rather than sourced cross-file per this project's
# established pattern of small per-tool reimplementations, e.g.
# g2_pkt_sniffer.py's own doc-comment on why it re-implements rather than
# imports the Rust classifier).
# ---------------------------------------------------------------------------
proc_rss_kb() { awk '/VmRSS/{print $2}' "/proc/$1/status" 2>/dev/null || echo 0; }
proc_cpu_ticks() { awk '{print $14+$15}' "/proc/$1/stat" 2>/dev/null || echo 0; }

sample_rss_bg() {
  local pid="$1" samples_file="$2" stop_file="$3"
  : > "${samples_file}"
  while [[ ! -e "${stop_file}" ]]; do
    proc_rss_kb "${pid}" >> "${samples_file}"
    sleep 0.2
  done
}
max_of_file() { sort -n "$1" 2>/dev/null | tail -1 || echo 0; }

# ---------------------------------------------------------------------------
# G1 (plain-UDP baseline) + G2 overhead (no impairment) — both come out of
# ONE g2-dpi-probe `overhead` run, since that subcommand always measures
# plain-UDP and MASQUE back-to-back in the same process for a fair delta.
# ---------------------------------------------------------------------------
PROBE_BIN="${HELIX_CORE_DIR}/target/release/g2-dpi-probe"

build_probe() {
  log "Building g2-dpi-probe (cargo build --release -p helix-core --bin g2-dpi-probe)..."
  (cd "${HELIX_CORE_DIR}" && cargo build --release -p helix-core --bin g2-dpi-probe 2>&1 | tail -8)
}

run_overhead_phase() {
  local evidence_file="${WORKDIR}/overhead.json"
  local rss_samples="${WORKDIR}/overhead_rss.samples" stop_file="${WORKDIR}/overhead_rss.stop"
  rm -f "${stop_file}"

  log "Running g2-dpi-probe overhead (rounds=${OVERHEAD_ROUNDS} chunk=${OVERHEAD_CHUNK_BYTES}B, no impairment, no root)..."
  "${PROBE_BIN}" overhead --evidence-file "${evidence_file}" \
    --rounds "${OVERHEAD_ROUNDS}" --chunk-size "${OVERHEAD_CHUNK_BYTES}" \
    > "${WORKDIR}/overhead_stdout.log" 2>&1 &
  local probe_pid=$!

  sample_rss_bg "${probe_pid}" "${rss_samples}" "${stop_file}" &
  local sampler_pid=$!

  wait "${probe_pid}" || true
  touch "${stop_file}"; wait "${sampler_pid}" 2>/dev/null || true

  if [[ ! -s "${evidence_file}" ]]; then
    log "FAIL: overhead evidence file not written"; cat "${WORKDIR}/overhead_stdout.log" >&2
    return 1
  fi

  local rss_peak_kb; rss_peak_kb="$(max_of_file "${rss_samples}")"
  echo "${rss_peak_kb:-0}" > "${WORKDIR}/overhead_rss_peak_kb"
  echo "${evidence_file}"
}

emit_g1_and_g2_overhead() {
  local ev; ev="$(run_overhead_phase)"
  local rss_peak_kb; rss_peak_kb="$(cat "${WORKDIR}/overhead_rss_peak_kb" 2>/dev/null || echo 0)"

  # --- pull the fields we need out of the real JSON (jq, no guessing) ---
  local plain_hs plain_rt_us plain_cpu_us plain_goodput_bps
  plain_hs="$(jq -r '.plain_udp.handshake_ms' "${ev}")"
  plain_rt_us="$(jq -r '.plain_udp.avg_round_trip_us' "${ev}")"
  plain_cpu_us="$(jq -r '.plain_udp.cpu_us_per_round_trip' "${ev}")"
  plain_goodput_bps="$(jq -r '.plain_udp.round_trip_goodput_bytes_per_sec' "${ev}")"

  local masque_hs masque_rt_us masque_cpu_us masque_goodput_bps ratio
  masque_hs="$(jq -r '.masque.handshake_ms' "${ev}")"
  masque_rt_us="$(jq -r '.masque.avg_round_trip_us' "${ev}")"
  masque_cpu_us="$(jq -r '.masque.cpu_us_per_round_trip' "${ev}")"
  masque_goodput_bps="$(jq -r '.masque.round_trip_goodput_bytes_per_sec' "${ev}")"
  ratio="$(jq -r '.throughput_ratio_masque_vs_plain' "${ev}")"

  local plain_mbps masque_mbps
  plain_mbps="$(echo "scale=4; ${plain_goodput_bps} * 8 / 1000000" | bc -l)"
  masque_mbps="$(echo "scale=4; ${masque_goodput_bps} * 8 / 1000000" | bc -l)"

  # cores-of-CPU-per-Gbps, same formula edge_ab.sh's cpu_per_gbps uses:
  # (cpu_seconds / elapsed_seconds) / (Mbps/1000). Here cpu_us_per_round_trip
  # and avg_round_trip_us are already PER round trip, so cpu/elapsed cancels
  # to cpu_us / round_trip_us directly.
  local plain_cpu_per_gbps masque_cpu_per_gbps
  plain_cpu_per_gbps="$(echo "scale=6; (${plain_cpu_us} / ${plain_rt_us}) / (${plain_mbps} / 1000)" | bc -l 2>/dev/null || echo 0)"
  masque_cpu_per_gbps="$(echo "scale=6; (${masque_cpu_us} / ${masque_rt_us}) / (${masque_mbps} / 1000)" | bc -l 2>/dev/null || echo 0)"

  local overhead_note="g2-dpi-probe overhead subcommand, ${OVERHEAD_ROUNDS} rounds x ${OVERHEAD_CHUNK_BYTES}B, request-response (no pipelining) -- NOT an iperf3-equivalent saturating bulk transfer (iperf3 unavailable in this sandbox, confirmed via 'command -v iperf3')."

  # ---- G1 rows ----
  emit G1 plain-udp n/a baseline_round_trip_latency_ms "$(echo "scale=4; ${plain_rt_us}/1000" | bc -l)" ms \
    "n/a (this IS the baseline other transports are measured against)" RECORDED \
    "${overhead_note}" "loopback UDP transport RTT, no WG encryption layered on top yet in this codebase."

  emit G1 plain-udp n/a through_tunnel_throughput_mbps "${plain_mbps}" Mbps \
    ">=80% of bare link (iperf3)" UNMEASURED_VS_BAR \
    "${overhead_note}" "${NO_TUNNEL_NOTE} No separate 'bare link' number exists to compute a percentage against -- this raw transport goodput IS effectively the bare-link number in this sandbox."

  emit G1 plain-udp n/a edge_cpu_per_gbps "${plain_cpu_per_gbps}" cores_per_gbps \
    "record (cost-to-serve)" RECORDED "${overhead_note}" \
    "'edge' column reused per spec §8 naming; no separate edge-relay process exists on the G1 path in this codebase -- this is the client/connector transport-primitive process's own cost."

  emit G1 plain-udp n/a core_rss_peak_kb "${rss_peak_kb}" KiB \
    "record (§6.3 iOS gate is the platform-specific instance)" RECORDED \
    "linux /proc/<pid>/status VmRSS sampled every 0.2s" \
    "Same process instance also ran the G2 masque overhead phase sequentially right after -- this is the PEAK across both phases, not isolated per-transport."

  # ---- G2 rows (no-impairment overhead half) ----
  local hs_verdict_str="FAIL"; [[ "$(echo "${masque_hs} < 2000" | bc -l)" == "1" ]] && hs_verdict_str="PASS"
  emit G2 masque-h3 n/a handshake_time_ms "${masque_hs}" ms "<2s MASQUE (spec §8)" "${hs_verdict_str}" \
    "real quinn QUIC/TLS handshake dial() time, no impairment, no DPI block active" \
    "This is the pure-handshake number (dial() only); see the second handshake_time_ms row (method=dpi-survival) for the handshake+echo elapsed under an ACTIVE DPI block."

  local thr_verdict="FAIL"; [[ "$(echo "${ratio} >= 0.5" | bc -l)" == "1" ]] && thr_verdict="PASS"
  emit G2 masque-h3 n/a through_tunnel_throughput_mbps "${masque_mbps}" Mbps \
    ">=50% of plain-UDP (spec §8)" "${thr_verdict}" "${overhead_note}" \
    "${NO_TUNNEL_NOTE} ratio_masque_vs_plain=${ratio} (computed by g2-dpi-probe itself from the same run's plain-UDP number above)."

  local added_latency_ms; added_latency_ms="$(echo "scale=4; (${masque_rt_us} - ${plain_rt_us}) / 1000" | bc -l)"
  local lat_verdict="FAIL"; [[ "$(echo "${added_latency_ms} < 15" | bc -l)" == "1" ]] && lat_verdict="PASS"
  emit G2 masque-h3 n/a added_latency_ms "${added_latency_ms}" ms "<15ms added (spec §8)" "${lat_verdict}" \
    "delta between masque and plain-udp average round-trip WITHIN the same overhead run (same process, same loopback, back-to-back phases)" \
    "This is a transport-vs-transport delta (masque RTT minus plain-UDP RTT), not a tunnel-vs-bare-link delta -- the most direct 'added by MASQUE' proxy available without a real dataplane."

  emit G2 masque-h3 n/a edge_cpu_per_gbps "${masque_cpu_per_gbps}" cores_per_gbps \
    "record (cost-to-serve)" RECORDED "${overhead_note}" \
    "'edge' column reused per spec §8 naming; no separate edge-relay process exists on this path -- see gate=G4 rows for the real Rust/Go edge-relay process cost."

  emit G2 masque-h3 n/a core_rss_peak_kb "${rss_peak_kb}" KiB \
    "record (§6.3 iOS gate is the platform-specific instance)" RECORDED \
    "linux /proc/<pid>/status VmRSS sampled every 0.2s" \
    "Same process instance also ran the G1 plain-udp overhead phase sequentially right before -- this is the PEAK across both phases, not isolated per-transport."
}

# ---------------------------------------------------------------------------
# G2 DPI-block survival + wire fingerprint: reruns helix_core's OWN,
# already-established, UNMODIFIED sandboxed rig (g2_dpi_masque_unpriv.sh,
# living in submodules/helix_core -- explicitly out of this task's file
# scope, so it is invoked, never edited) -- this is the "reuse whatever
# sandboxed techniques the prior G2 work already established" instruction,
# applied literally: we invoke that script rather than re-implement its
# unshare/nft/AF_PACKET wiring a second time.
#
# --skip-loss is ALWAYS passed here (regardless of this harness's own
# --skip-loss flag, which instead gates run_g2_loss_resilience_phase
# below) because of a real, reproduced bug found via direct
# investigation (not guessed, §11.4.102/§11.4.6): g2_dpi_masque_unpriv.sh's
# own internal loss-resilience phase runs BOTH offered-load sub-tests as
# plain top-level statements under `set -euo pipefail`, but
# g2-dpi-probe's `loss-resilience` subcommand's exit code communicates
# its comparison VERDICT (exits 1 whenever the real measured result is
# "FAIL", by the exact same convention as its dpi-survival/overhead
# subcommands -- confirmed by reading g2-dpi-probe.rs's main(): `Ok(false)
# => ExitCode::FAILURE`). A "FAIL" is the historically EXPECTED, honestly
# documented outcome of this specific test (see helix_core's own
# G2-RESULTS.md sec 5: "MASQUE did not beat the UoT-over-TCP strawman at
# either offered load tested") -- so `set -e` aborts the wrapper BEFORE
# its second sub-test and before its own tc-netem cleanup, REPRODUCED
# directly in this sandbox: running the wrapper unmodified left only
# loss-resilience-300kbps.json on disk, never -2mbps.json, with no
# explicit error text (confirmed via a standalone repro: invoking
# `g2-dpi-probe loss-resilience --target-bitrate-bps 2000000` directly
# under the identical unshare+tc-netem conditions completes successfully
# and writes real evidence, but exits 1 because its own verdict is
# "FAIL"). Fix belongs in that submodule (out of this task's scope) --
# worked around here by running BOTH offered-load sub-tests ourselves,
# reusing the SAME technique (same probe binary, same
# `unshare --net --user --map-root-user`, same `tc netem loss 5% delay
# 40ms 10ms` impairment profile from spec §3/§8) with tolerant error
# handling, in run_g2_loss_resilience_phase below.
# ---------------------------------------------------------------------------
run_g2_dpi_and_loss() {
  local stdout_log="${WORKDIR}/g2_unpriv_stdout.log"
  local extra_flags=(--skip-overhead --skip-loss)  # overhead: covered by run_overhead_phase; loss: see comment above

  log "Running helix_core's g2_dpi_masque_unpriv.sh ${extra_flags[*]} (unshare --net --user, no root)..."
  bash "${G2_UNPRIV_SCRIPT}" "${extra_flags[@]}" > "${stdout_log}" 2>&1 || true
  cat "${stdout_log}" >&2

  local evidence_dir
  evidence_dir="$(grep -m1 '^Evidence dir:' "${stdout_log}" | sed 's/^Evidence dir: //')"
  if [[ -z "${evidence_dir}" || ! -d "${evidence_dir}" ]]; then
    log "FAIL: could not locate g2_dpi_masque_unpriv.sh evidence dir from its stdout"
    return 1
  fi
  echo "${evidence_dir}"
}

# Standalone loss-resilience runner (see the long comment above for why
# this exists instead of relying on g2_dpi_masque_unpriv.sh's own Phase
# 4): same probe binary + same unshare/tc-netem technique, but tolerant
# of the probe's expected non-zero exit on a real "FAIL" verdict.
run_g2_loss_resilience_phase() {
  local out_dir="${WORKDIR}/loss_resilience"
  mkdir -p "${out_dir}"
  local inner_script="${WORKDIR}/loss_inner.sh"
  cat > "${inner_script}" <<'INNER'
set -uo pipefail
PROBE_BIN="$1"; OUT_DIR="$2"
ip link set lo up
tc qdisc add dev lo root netem loss 5% delay 40ms 10ms
tc qdisc show dev lo
for pair in "300kbps:300000" "2mbps:2000000"; do
  label="${pair%%:*}"; bitrate="${pair##*:}"
  echo "--- loss-resilience offered load: ${label} (${bitrate} bps) ---"
  "$PROBE_BIN" loss-resilience --evidence-file "${OUT_DIR}/loss-resilience-${label}.json" \
    --duration-secs 5 --chunk-size 1200 --target-bitrate-bps "${bitrate}"
  rc=$?
  echo "  (exit ${rc} -- non-zero here means the REAL measured verdict was FAIL, not a crash; evidence file is still written -- see g2-dpi-probe.rs's exit-code convention)"
done
tc qdisc del dev lo root
CLEAN="$(tc qdisc show dev lo)"
if echo "$CLEAN" | grep -q netem; then
  echo "FATAL: netem qdisc still present after cleanup" >&2
  exit 1
fi
INNER
  log "Running standalone loss-resilience phase (unshare --net --user, real tc netem loss 5% delay 40ms 10ms on lo)..."
  unshare --net --user --map-root-user -- bash "${inner_script}" "${PROBE_BIN}" "${out_dir}" >&2
  echo "${out_dir}"
}

emit_g2_dpi_survival_and_loss() {
  local evdir; evdir="$(run_g2_dpi_and_loss)"

  # --- DPI-block survival (core G2 criterion, spec §5.3.1/§0 -- bonus
  #     context row, not literally one of the §8 metric names but the
  #     headline this whole gate exists to prove) ---
  local blocked_json="${evdir}/dpi-survival-blocked.json"
  if [[ -s "${blocked_json}" ]]; then
    local verdict plain_ok plain_ms masque_ok masque_elapsed_ms
    verdict="$(jq -r '.verdict' "${blocked_json}")"
    plain_ok="$(jq -r '.plain_udp_wg.success' "${blocked_json}")"
    plain_ms="$(jq -r '.plain_udp_wg.elapsed_ms' "${blocked_json}")"
    masque_ok="$(jq -r '.masque.success' "${blocked_json}")"
    masque_elapsed_ms="$(jq -r '.masque.elapsed_ms' "${blocked_json}")"
    local verdict_str="FAIL"; [[ "${verdict}" == PASS* ]] && verdict_str="PASS"
    emit G2 masque-h3 n/a dpi_block_survival "$([[ ${verdict_str} == PASS ]] && echo 1 || echo 0)" boolean \
      "plain-WG blocked (fails) AND masque survives (spec §0/§5.3.1)" "${verdict_str}" \
      "real nft DROP rule inside unshare --net --user --map-root-user, real boringtun handshake vs real quinn QUIC handshake" \
      "plain_udp_wg.success=${plain_ok} elapsed=${plain_ms}ms; masque.success=${masque_ok} elapsed=${masque_elapsed_ms}ms; full evidence: ${blocked_json}"

    local hs_verdict="FAIL"; [[ "$(echo "${masque_elapsed_ms} < 2000" | bc -l)" == "1" ]] && hs_verdict="PASS"
    emit G2 masque-h3 n/a handshake_time_ms "${masque_elapsed_ms}" ms "<2s MASQUE (spec §8)" "${hs_verdict}" \
      "dial+send+recv-echo elapsed under an ACTIVE real nft DPI block on the plain-WG port (dpi-survival subcommand)" \
      "This is the handshake proven under REAL censorship conditions (the scenario G2 exists to test), as opposed to the no-impairment pure-handshake row above."
  else
    log "WARNING: dpi-survival-blocked.json not found under ${evdir} -- skipping DPI-survival rows honestly"
  fi

  # --- wire fingerprint (spec §5.3.2/§8) ---
  local capture_jsonl="${evdir}/capture.jsonl"
  if [[ -s "${capture_jsonl}" ]]; then
    local tally; tally="$(python3 -c "
import json, sys
path = sys.argv[1]
port = 443
total = quic = wg = 0
with open(path) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        rec = json.loads(line)
        if rec.get('dport') == port:
            total += 1
            kind = rec.get('wire_signature', {}).get('kind')
            if kind in ('QuicLongHeader', 'QuicShortHeaderCandidate'):
                quic += 1
            elif kind == 'WireGuard':
                wg += 1
print(f'{total} {quic} {wg}')
" "${capture_jsonl}")"
    local total quic wg
    read -r total quic wg <<< "${tally}"
    local quic_frac="0"
    [[ "${total}" -gt 0 ]] && quic_frac="$(echo "scale=4; ${quic} / ${total}" | bc -l)"
    local fp_verdict="FAIL"; [[ "${wg}" == "0" && "${quic}" -gt 0 ]] && fp_verdict="PASS"
    emit G2 masque-h3 n/a wire_fingerprint_quic_fraction "${quic_frac}" fraction_of_port443_packets \
      "classified as HTTP/3 (QUIC framing), zero WireGuard signature (spec §8)" "${fp_verdict}" \
      "hand-rolled AF_PACKET raw-socket sniffer (submodules/helix_core/scripts/spike/g2_pkt_sniffer.py) -- no tshark/tcpdump in this sandbox (confirmed via 'command -v')" \
      "${total} UDP datagrams captured on dst port 443; ${quic} classified QUIC (long/short-header framing), ${wg} classified WireGuard. Classifies QUIC wire framing, does NOT itself confirm the HTTP/3 CONNECT-UDP semantic layer above it. Full capture: ${capture_jsonl}"
  else
    log "WARNING: capture.jsonl not found under ${evdir} -- skipping wire-fingerprint row honestly"
  fi

  # --- loss resilience (spec §5.3.3/§8) ---
  if $SKIP_LOSS; then
    emit G2 masque-h3 n/a loss_resilience 0 n/a "MASQUE/QUIC > UoT strawman (spec §8)" SKIP \
      "--skip-loss passed to this harness run" "tc netem loss-resilience phase intentionally skipped this run"
  else
    local loss_dir; loss_dir="$(run_g2_loss_resilience_phase)"
    for load in 300kbps 2mbps; do
      local f="${loss_dir}/loss-resilience-${load}.json"
      if [[ -s "${f}" ]]; then
        local masque_bps uot_bps masque_ratio uot_ratio offered
        masque_bps="$(jq -r '.masque_goodput_bps' "${f}")"
        uot_bps="$(jq -r '.uot_goodput_bps' "${f}")"
        masque_ratio="$(jq -r '.masque_delivery_ratio' "${f}")"
        uot_ratio="$(jq -r '.uot_delivery_ratio' "${f}")"
        offered="$(jq -r '.offered_load_bps' "${f}")"
        local lr_verdict="FAIL"; [[ "$(echo "${masque_bps} > ${uot_bps}" | bc -l)" == "1" ]] && lr_verdict="PASS"
        emit G2 masque-h3 n/a "loss_resilience.offered_${load}" "${masque_bps}" bytes_per_sec \
          "MASQUE/QUIC > UoT strawman (spec §8/§5.3.3)" "${lr_verdict}" \
          "real tc netem loss 5% delay 40ms 10ms on lo inside unshare, paced offered-load comparison (MASQUE vs a UDP-over-TCP strawman)" \
          "offered_load_bps=${offered}; masque_goodput_bps=${masque_bps} (delivery_ratio=${masque_ratio}); uot_goodput_bps=${uot_bps} (delivery_ratio=${uot_ratio}). Full evidence: ${f}"
      else
        log "WARNING: loss-resilience-${load}.json not found under ${loss_dir} -- skipping honestly"
      fi
    done
  fi

  emit G2 masque-h3 n/a reconnect_roam_s 0 s "<3s (spec §8)" SKIP "n/a" "${NO_ROAM_NOTE}"
}

emit_g1_not_applicable_rows() {
  emit G1 plain-udp n/a loss_resilience 0 n/a "MASQUE/QUIC > UoT strawman (spec §8)" NOT_APPLICABLE \
    "n/a" "This pass-bar's text is MASQUE-vs-UoT specific; see gate=G2 rows for the real measurement."
  emit G1 plain-udp n/a wire_fingerprint_quic_fraction 0 n/a "classified as HTTP/3, no WG signature (spec §8)" NOT_APPLICABLE \
    "n/a" "Fingerprinting exists to prove MASQUE is indistinguishable from ordinary HTTPS traffic; plain-UDP-WG is not trying to hide, so this bar does not apply to G1."
  emit G1 plain-udp n/a reconnect_roam_s 0 s "<3s (spec §8)" SKIP "n/a" "${NO_ROAM_NOTE}"
}

# ---------------------------------------------------------------------------
# G4 (edge A/B): reruns this repo's OWN edge_ab.sh (unmodified) with a
# bounded duration/concurrency set so this harness stays fast, then
# normalizes its CSV rows into the unified schema. edge_ab.sh's own CSV
# (metric = "<edge>.<mode>.c<N>.<submetric>") is left completely
# untouched on disk -- this only READS it.
# ---------------------------------------------------------------------------
run_g4_edge_ab() {
  local edge_csv="${WORKDIR}/edge_ab.csv"
  local edge_ab_log="${WORKDIR}/edge_ab_stdout.log"
  log "Running edge_ab.sh --duration-secs ${EDGE_DURATION_SECS} --concurrencies \"${EDGE_CONCURRENCIES}\" (bounded, for speed -- see edge_ab.sh directly for the full 1/10/100 sweep already used for the G4 decision-log row)..."
  bash "${SCRIPT_DIR}/edge_ab.sh" --out-csv "${edge_csv}" \
    --duration-secs "${EDGE_DURATION_SECS}" --concurrencies "${EDGE_CONCURRENCIES}" \
    > "${edge_ab_log}" 2>&1
  cat "${edge_ab_log}" >&2
  echo "${edge_csv}"
}

csv_get() { awk -F, -v m="$2" '$3==m {print $4}' "$1" | tail -1; }
csv_max_across_c() {
  # csv_max_across_c <csv> <edge> <mode> <submetric>  -> "value concurrency"
  awk -F, -v e="$2" -v mo="$3" -v s="$4" '
    $3 ~ ("^"e"\\."mo"\\.c[0-9]+\\."s"$") {
      split($3, a, ".c"); split(a[2], b, "."); print $4, b[1]
    }' "$1" | sort -n -k1,1 | tail -1
}

emit_g4_edge_ab() {
  local edge_csv; edge_csv="$(run_g4_edge_ab)"
  local plain_mbps; plain_mbps="$(jq -r '.plain_udp.round_trip_goodput_bytes_per_sec' "${WORKDIR}/overhead.json" 2>/dev/null | awk '{printf "%.4f", $1*8/1000000}' 2>/dev/null || echo "")"

  for edge in rust go; do
    local peak_line peak_mbps peak_c
    peak_line="$(csv_max_across_c "${edge_csv}" "${edge}" throughput sink_mbps)"
    peak_mbps="$(echo "${peak_line}" | awk '{print $1}')"
    peak_c="$(echo "${peak_line}" | awk '{print $2}')"
    [[ -z "${peak_mbps}" ]] && peak_mbps=0
    [[ -z "${peak_c}" ]] && peak_c="?"

    local cpu_c10 p50 p99
    cpu_c10="$(csv_get "${edge_csv}" "${edge}.throughput.c10.cpu_per_gbps")"
    p50="$(csv_get "${edge_csv}" "${edge}.latency.c1.p50_ms")"
    p99="$(csv_get "${edge_csv}" "${edge}.latency.c1.p99_ms")"
    local hs_line hs_ms hs_c
    hs_line="$(csv_max_across_c "${edge_csv}" "${edge}" throughput handshake_setup_ms)"
    hs_ms="$(echo "${hs_line}" | awk '{print $1}')"; [[ -z "${hs_ms}" ]] && hs_ms=0
    local mem_line mem_peak_kb
    mem_line="$(csv_max_across_c "${edge_csv}" "${edge}" churn mem_peak_kb)"
    mem_peak_kb="$(echo "${mem_line}" | awk '{print $1}')"; [[ -z "${mem_peak_kb}" ]] && mem_peak_kb=0

    local edge_label="Rust helix-edge (hand-rolled non-HTTP/3 CONNECT-UDP stand-in)"
    [[ "${edge}" == "go" ]] && edge_label="Go go-edge (real RFC 9298 masque-go/quic-go)"

    local thr_note="edge_ab.sh (HVPN-P0-045 protocol), sink-side authoritative goodput on loopback, bounded run (duration=${EDGE_DURATION_SECS}s, concurrencies=\"${EDGE_CONCURRENCIES}\") -- see edge_ab.sh's own README for the full 1/10/100 sweep numbers already used for the G4 decision-log row. ${edge_label}. Neither edge has real kernel-WireGuard/boringtun gateway-socket integration yet -- this measures the MASQUE-termination + gateway-relay hand-off data path itself, not an end-to-end tunnel."
    local thr_verdict="FAIL"
    if [[ -n "${plain_mbps}" ]] && (( $(echo "${plain_mbps} > 0" | bc -l) )); then
      local edge_vs_g1; edge_vs_g1="$(echo "scale=4; ${peak_mbps} / ${plain_mbps}" | bc -l)"
      [[ "$(echo "${edge_vs_g1} >= 0.5" | bc -l)" == "1" ]] && thr_verdict="PASS"
      thr_note="${thr_note} Cross-reference vs this SAME run's G1 plain-udp transport-primitive number (${plain_mbps} Mbps): ratio=${edge_vs_g1}. HONEST CAVEAT: this is an apples-to-oranges cross-reference -- G1's number is the raw client/connector transport primitive on loopback with no edge-relay hop, while this G4 number is the edge-relay hand-off hop in isolation with no client/connector transport layered on top; they are not measuring the same code path end-to-end (no real tunnel exists yet to make them comparable for real)."
    else
      thr_note="${thr_note} No G1 plain-udp cross-reference available this run (--skip-g1 was likely set)."
      thr_verdict="UNMEASURED_VS_BAR"
    fi
    emit G4 masque-h3 "${edge}" through_tunnel_throughput_mbps "${peak_mbps}" Mbps \
      ">=50% of plain-UDP (spec §8)" "${thr_verdict}" "peak sink_mbps @ c=${peak_c}" "${thr_note}"

    local lat_verdict="FAIL"; [[ -n "${p50}" ]] && [[ "$(echo "${p50} < 15" | bc -l)" == "1" ]] && lat_verdict="PASS"
    emit G4 masque-h3 "${edge}" added_latency_ms "${p50:-0}" ms "<15ms MASQUE added (spec §8)" "${lat_verdict}" \
      "edge_ab.sh idle single-flow p50, loopback" \
      "p50=${p50:-0}ms p99=${p99:-0}ms. Loopback RTT is NOT representative of real network latency (same caveat edge_ab.sh's own README states) -- this measures this sandbox's MASQUE-processing + relay overhead only, compared directly against the 15ms bar as the best available stand-in absent a real network."

    emit G4 masque-h3 "${edge}" loss_resilience 0 n/a "MASQUE/QUIC > UoT strawman (spec §8)" NOT_MEASURED \
      "n/a" "edge_ab.sh's protocol (spec §7.2) does not include a tc-netem loss-impairment step; gate=G2's loss_resilience rows (real tc netem 5% loss) are the project-wide source of truth for this bar."

    local hs_verdict="FAIL"; [[ "$(echo "${hs_ms} < 2000" | bc -l)" == "1" ]] && hs_verdict="PASS"
    emit G4 masque-h3 "${edge}" handshake_time_ms "${hs_ms}" ms "<2s MASQUE (spec §8)" "${hs_verdict}" \
      "edge_ab.sh handshake_setup_ms @ c=$(echo "${hs_line}" | awk '{print $2}')" \
      "Contended handshake-setup time at the highest tested concurrency, not an idle-single-flow number -- reports connection-churn cost under concurrent load, matching spec §7.2's own churn protocol."

    emit G4 masque-h3 "${edge}" edge_cpu_per_gbps "${cpu_c10:-0}" cores_per_gbps \
      "record (cost-to-serve, spec §8)" RECORDED "edge_ab.sh cpu_per_gbps @ c=10" \
      "Real /proc/<pid>/stat utime+stime attributed to the edge server process alone (edge_ab.sh's own per-process CPU accounting)."

    emit G4 masque-h3 "${edge}" core_rss_peak_kb "${mem_peak_kb:-0}" KiB \
      "record (§6.3 iOS gate is the platform-specific instance)" RECORDED \
      "edge_ab.sh /proc/<pid>/status VmRSS peak across churn phases" "Peak memory-under-connection-churn for the edge server process alone."

    emit G4 masque-h3 "${edge}" wire_fingerprint_quic_fraction 0 fraction_of_port443_packets \
      "classified as HTTP/3, no WG signature (spec §8)" NOT_MEASURED "n/a" \
      "edge_ab.sh runs entirely on loopback with no packet capture step; gate=G2's wire_fingerprint row (real AF_PACKET capture) is the project-wide source for the Rust path. HONEST GAP: the Go edge uses a genuinely different, real masque-go/quic-go MASQUE implementation not exercised by G2's capture run (which only drives helix-masque's Rust implementation) -- a Go-specific wire-fingerprint capture is flagged follow-up, not fabricated here."

    emit G4 masque-h3 "${edge}" reconnect_roam_s 0 s "<3s (spec §8)" SKIP "n/a" "${NO_ROAM_NOTE}"
  done
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
echo ""
echo "=== Helix VPN — unified G1/G2/G4 measurement harness (HVPN-P0-077) ==="
echo "Output CSV: ${OUT_CSV}"
echo ""

if ! $SKIP_BUILD; then
  build_probe
fi

if ! $SKIP_G1 || ! $SKIP_G2; then
  # the overhead run produces BOTH G1's plain-udp numbers AND G2's
  # no-impairment masque numbers in one pass -- run it once whenever
  # either gate is wanted (rows are tagged gate=G1/gate=G2 correctly
  # either way; --skip-g1 alone with --skip-g2 unset still yields a few
  # gate=G1 rows as a side effect of the shared run, which is honest
  # data, just not filtered out for that narrow combination).
  emit_g1_and_g2_overhead
  if ! $SKIP_G1; then
    emit_g1_not_applicable_rows
  fi
fi

if ! $SKIP_G2; then
  emit_g2_dpi_survival_and_loss
fi

if ! $SKIP_G4; then
  emit_g4_edge_ab
fi

echo ""
echo "Unified harness results written to ${OUT_CSV}"
if command -v column >/dev/null 2>&1; then
  echo ""
  echo "--- Summary (gate.transport.edge.metric = value unit [verdict]) ---"
  awk -F, 'NR>1 {printf "%-4s %-11s %-5s %-32s %14s %-22s %s\n", $2,$3,$4,$5,$6,$7,$9}' "${OUT_CSV}"
fi
echo "=== Unified harness complete: $(date -Iseconds) ==="
