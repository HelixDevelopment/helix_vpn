#!/usr/bin/env bash
# Helix VPN — Phase 0 5-minute demo script (HVPN-P0-078)
#
# Purpose:  Walk an operator through the Phase-0 vertical slice
#           (HelixVPN-Phase0-Spike.md §1 "The vertical slice" + §2 "Spike
#           milestones" + §11 deliverable 8: "plain-UDP slice -> block WG
#           -> MASQUE slice survives -> flip a map entry -> new network
#           reachable, all narrated"), end to end, with real commands and
#           real captured evidence at every beat -- no slideware, no
#           narrated-but-not-executed steps. This script adds NO new
#           capability: it orchestrates the EXISTING Phase-0 tools
#           (submodules/helix_core's `g2-dpi-probe` binary + its own
#           `scripts/spike/g2_dpi_masque_unpriv.sh`, the
#           `g6_map_reconcile_integration` test, and this repo's own
#           `scripts/rig/*.sh` + `scripts/bench/edge_ab.sh` +
#           `scripts/spike.sh`) and narrates around their real output.
# Usage:    ./scripts/demo.sh [--skip-g4] [--skip-build]
#           Run as a normal user (default path, no root — everything
#           below works root-free in this sandbox) OR with sudo (EUID==0
#           additionally unlocks the real 2-netns routed-LAN reachability
#           beat via scripts/rig/*.sh, mirroring scripts/spike.sh's own
#           EUID check).
# Inputs:   SKIP_G4 (boolean, default false) -- skip the bonus G4
#             edge A/B beat to shorten the demo further.
#           SKIP_BUILD (boolean, default false) -- skip (re)building
#             g2-dpi-probe; use only if you already built it this
#             session (submodules/helix_core/target/release/g2-dpi-probe).
# Outputs:  Narrated console log of each beat + PASS/FAIL/SKIP per beat;
#           JSON/JSONL evidence under
#             submodules/helix_core/target/spike-evidence/g2/<run-id>/
#           (that submodule's own gitignored build-artifact dir).
# Side-effects: Builds submodules/helix_core's `g2-dpi-probe` release bin
#               (cargo) unless --skip-build. Runs helix_core's OWN
#               unmodified scripts/spike/g2_dpi_masque_unpriv.sh, which
#               creates one throwaway `unshare --net --user` namespace
#               for its own duration only (no host network state
#               touched). Runs `cargo test` for the G6 reconcile
#               integration test. Optionally (--skip-g4 unset) builds +
#               runs scripts/bench/edge_ab.sh's loopback-only tools.
#               With real root (EUID==0), additionally sets up + tears
#               down the scripts/rig/ netns topology.
# Dependencies: bash 4+, cargo, jq, unshare (util-linux), nft (nftables),
#               python3 -- all already required by the tools this script
#               drives; see each tool's own doc-comment for specifics.
# Cross-refs: docs/research/mvp/04_VPN_CLD/HelixVPN-Phase0-Spike.md §1/§2/§11,
#             scripts/bench/unified_harness.sh (HVPN-P0-077, the sibling
#             measurement harness this demo's numbers are a narrated
#             subset of), submodules/helix_core/G2-RESULTS.md.

set -uo pipefail   # deliberately NOT -e: some of the real tools this
                   # script drives (g2-dpi-probe) use their exit code to
                   # report a measured PASS/FAIL VERDICT, not a crash --
                   # see scripts/bench/unified_harness.sh's long comment
                   # on this for the fully-investigated root cause. This
                   # script checks each step's result explicitly instead
                   # of relying on `set -e`.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
HELIX_CORE_DIR="${ROOT_DIR}/submodules/helix_core"
PROBE_BIN="${HELIX_CORE_DIR}/target/release/g2-dpi-probe"

SKIP_G4=false
SKIP_BUILD=false
for a in "$@"; do
  case "$a" in
    --skip-g4) SKIP_G4=true ;;
    --skip-build) SKIP_BUILD=true ;;
  esac
done

BEATS_OK=0
BEATS_TOTAL=0
beat_pass() { BEATS_TOTAL=$((BEATS_TOTAL+1)); BEATS_OK=$((BEATS_OK+1)); echo -e "\033[0;32m[BEAT PASS]\033[0m $*"; }
beat_fail() { BEATS_TOTAL=$((BEATS_TOTAL+1));                          echo -e "\033[0;31m[BEAT FAIL]\033[0m $*"; }
beat_skip() { echo -e "\033[1;33m[BEAT SKIP]\033[0m $*"; }
narrate()   { echo ""; echo -e "\033[1;36m### $* \033[0m"; }
say()       { echo "    $*"; }

echo ""
echo "=============================================================="
echo "  Helix VPN — Phase 0, 5-minute demo"
echo "  (HelixVPN-Phase0-Spike.md §1/§2/§11 vertical slice, narrated)"
echo "=============================================================="
echo "Started: $(date -Iseconds)"
echo ""
echo "The vertical slice this spike proves (spec §1): ONE client, ONE"
echo "gateway (edge + kernel WG/netns router), ONE connector, ONE LAN"
echo "host. This demo shows it working over plain UDP, surviving a"
echo "DPI-style block of plain WireGuard by switching to MASQUE/QUIC,"
echo "and reconciling a live network-map change without a restart --"
echo "the exact four beats spec §11 deliverable 8 calls for."
echo ""

# -----------------------------------------------------------------------
# Beat 0: prerequisites (mirrors scripts/spike.sh's own [S0])
# -----------------------------------------------------------------------
narrate "Beat 0 — prerequisites"
if rustc --version &>/dev/null; then
  say "rustc: $(rustc --version 2>&1)"
else
  beat_fail "rustc missing — cannot run any of this demo's Rust probes"
  exit 1
fi
if ! $SKIP_G4 && ! go version &>/dev/null; then
  say "go: MISSING — the bonus G4 beat will be skipped honestly (edge_ab.sh needs it for the Go edge)"
fi
say "OS: $(uname -srm)"
if [[ ! -x "${PROBE_BIN}" ]] && $SKIP_BUILD; then
  beat_fail "--skip-build given but ${PROBE_BIN} does not exist yet — build it first (drop --skip-build)"
  exit 1
fi
if ! $SKIP_BUILD; then
  say "Building submodules/helix_core's g2-dpi-probe (cargo build --release -p helix-core --bin g2-dpi-probe)..."
  if (cd "${HELIX_CORE_DIR}" && cargo build --release -p helix-core --bin g2-dpi-probe 2>&1 | tail -5); then
    beat_pass "g2-dpi-probe built"
  else
    beat_fail "g2-dpi-probe build failed"
    exit 1
  fi
else
  say "SKIP_BUILD set — using existing ${PROBE_BIN}"
fi

# -----------------------------------------------------------------------
# Beat 1: plain-UDP slice (G1) — a real WireGuard handshake, nothing
# blocking it. This is the "before censorship" baseline.
# -----------------------------------------------------------------------
narrate "Beat 1 — plain-UDP slice (G1): real WireGuard handshake, nothing blocked"
say "Driving a REAL boringtun Noise-IK WireGuard handshake"
say "(helix_orch::wg_session — the exact code helix-client/helix-connector"
say "drive) over plain UDP, with the DPI block INACTIVE:"
say "  \$ g2-dpi-probe dpi-survival --block-active false --evidence-file <path>"
say "(--masque-port 8443 here, not the well-known 443: binding :443"
say "needs CAP_NET_BIND_SERVICE, which Beat 2 gets for free from its"
say "unshare --map-root-user namespace -- confirmed directly: running"
say "this probe against :443 outside that namespace fails with a real"
say "'Permission denied (os error 13)', not a WireGuard/MASQUE defect."
say "Beat 1 only needs to prove the handshake itself, so a high port"
say "avoids that unrelated privilege requirement; the :443 disguise is"
say "the point of Beat 2, where it's actually exercised.)"
BEAT1_EV="$(mktemp -d)/beat1-noblock.json"
"${PROBE_BIN}" dpi-survival --evidence-file "${BEAT1_EV}" --block-active false \
  --plain-port 51822 --masque-port 8443 --wg-reference-port 51823 --probe-timeout-secs 8 \
  > /dev/null 2>&1
BEAT1_RC=$?
if [[ -s "${BEAT1_EV}" ]]; then
  PLAIN_OK="$(jq -r '.plain_udp_wg.success' "${BEAT1_EV}")"
  PLAIN_MS="$(jq -r '.plain_udp_wg.elapsed_ms' "${BEAT1_EV}")"
  say "plain_udp_wg.success=${PLAIN_OK}  elapsed_ms=${PLAIN_MS}  (spec §8 bar: <1000ms)"
  if [[ "${PLAIN_OK}" == "true" ]]; then
    beat_pass "real WireGuard handshake completed over plain UDP in ${PLAIN_MS}ms (evidence: ${BEAT1_EV})"
  else
    beat_fail "plain WireGuard handshake did NOT succeed with no block active — real defect, not narrated over"
  fi
else
  beat_fail "no evidence file written (rc=${BEAT1_RC}) — see stderr above"
fi

# -----------------------------------------------------------------------
# Beat 2: block WG, MASQUE survives (G2) — the censorship-evasion
# headline. Reuses helix_core's own established sandboxed rig
# unmodified (real nft DROP rule inside an unshare --net --user
# namespace, real AF_PACKET wire-fingerprint capture).
# -----------------------------------------------------------------------
narrate "Beat 2 — block plain WireGuard, MASQUE/QUIC survives (G2)"
say "Reusing helix_core's own established sandboxed rig (unmodified):"
say "  \$ scripts/spike/g2_dpi_masque_unpriv.sh --skip-overhead --skip-loss"
say "(real nft DROP rule on the plain-WG port + ACCEPT on :443/udp, run"
say "unprivileged via 'unshare --net --user --map-root-user' since no"
say "root/sudo is available in this sandbox; real AF_PACKET capture"
say "stands in for tshark/tcpdump, which are absent here too.)"
BEAT2_LOG="$(mktemp)"
bash "${HELIX_CORE_DIR}/scripts/spike/g2_dpi_masque_unpriv.sh" --skip-overhead --skip-loss > "${BEAT2_LOG}" 2>&1
BEAT2_RC=$?
EVDIR="$(grep -m1 '^Evidence dir:' "${BEAT2_LOG}" | sed 's/^Evidence dir: //')"
if [[ -n "${EVDIR}" && -s "${EVDIR}/dpi-survival-blocked.json" ]]; then
  BLOCKED="${EVDIR}/dpi-survival-blocked.json"
  PLAIN_OK="$(jq -r '.plain_udp_wg.success' "${BLOCKED}")"
  PLAIN_ERR="$(jq -r '.plain_udp_wg.error' "${BLOCKED}")"
  MASQUE_OK="$(jq -r '.masque.success' "${BLOCKED}")"
  MASQUE_MS="$(jq -r '.masque.elapsed_ms' "${BLOCKED}")"
  VERDICT="$(jq -r '.verdict' "${BLOCKED}")"
  say "WITH the block active:"
  say "  plain-UDP WireGuard: success=${PLAIN_OK}  error=\"${PLAIN_ERR}\"   <- BLOCKED, as expected"
  say "  MASQUE/QUIC:         success=${MASQUE_OK}  elapsed_ms=${MASQUE_MS}  <- SURVIVES, censorship evaded"
  say "  probe verdict: ${VERDICT}"
  if [[ "${PLAIN_OK}" == "false" && "${MASQUE_OK}" == "true" ]]; then
    beat_pass "plain WireGuard genuinely fails under the block while MASQUE/QUIC genuinely survives (evidence: ${BLOCKED})"
  else
    beat_fail "expected plain-WG to fail and MASQUE to survive under the block — did not observe that; real finding, not narrated over"
  fi
  if [[ -s "${EVDIR}/capture.jsonl" ]]; then
    TALLY="$(python3 -c "
import json
total=quic=wg=0
with open('${EVDIR}/capture.jsonl') as f:
    for line in f:
        rec=json.loads(line)
        if rec.get('dport')==443:
            total+=1
            k=rec.get('wire_signature',{}).get('kind')
            if k in ('QuicLongHeader','QuicShortHeaderCandidate'): quic+=1
            elif k=='WireGuard': wg+=1
print(f'{total} {quic} {wg}')
")"
    read -r C_TOTAL C_QUIC C_WG <<< "${TALLY}"
    say "Wire fingerprint (real AF_PACKET capture on :443): ${C_TOTAL} packets captured, ${C_QUIC} classified QUIC framing, ${C_WG} classified WireGuard."
    if [[ "${C_WG}" == "0" ]]; then
      beat_pass "zero WireGuard signature observed on the MASQUE port — the traffic is not distinguishable from ordinary QUIC/HTTP-3 (capture: ${EVDIR}/capture.jsonl)"
    else
      beat_fail "a WireGuard signature WAS observed on the MASQUE port — real finding, not narrated over"
    fi
  fi
else
  beat_fail "could not locate g2_dpi_masque_unpriv.sh's evidence (rc=${BEAT2_RC}) — see ${BEAT2_LOG}"
fi

# -----------------------------------------------------------------------
# Beat 3: flip a map entry, new network reachable (G6) — the push-based
# reconciliation model. Runs the REAL integration test, which drives a
# REAL Orchestrator + file-watch reconciler + a REAL temp map.json edit.
# -----------------------------------------------------------------------
narrate "Beat 3 — flip a network-map entry, new network becomes reachable, no restart (G6)"
say "Running the real G6 reconciliation integration test (drives a real"
say "Orchestrator + file-watch reconciler against a real temp map.json:"
say "peer unreachable -> edit map.json to add its prefix -> reachable"
say "within a poll-bounded wait, unrelated peer's route undisturbed,"
say "TunnelState::Connected unbroken throughout -- exactly spec §10's"
say "G6 pass criterion):"
say "  \$ cargo test -p helix-core --test g6_map_reconcile_integration -- --nocapture"
BEAT3_LOG="$(mktemp)"
(cd "${HELIX_CORE_DIR}" && cargo test -p helix-core --test g6_map_reconcile_integration -- --nocapture) \
  > "${BEAT3_LOG}" 2>&1
BEAT3_RC=$?
tail -6 "${BEAT3_LOG}" | sed 's/^/    /'
if [[ ${BEAT3_RC} -eq 0 ]] && grep -q "test result: ok" "${BEAT3_LOG}"; then
  beat_pass "live map-delta reconciliation proven: a new peer/network became reachable after editing map.json, with no restart and no disturbance to the existing peer (full log: ${BEAT3_LOG})"
else
  beat_fail "G6 reconcile test did not pass — real finding, not narrated over (log: ${BEAT3_LOG})"
fi

# -----------------------------------------------------------------------
# Bonus beat: G4 edge A/B (Rust vs Go MASQUE termination) — not one of
# the spec §11 deliverable-8 four beats, but part of the same Phase-0
# spike and it's an existing script this demo can narrate too.
# -----------------------------------------------------------------------
if ! $SKIP_G4; then
  narrate "Bonus beat — G4 edge A/B (Rust helix-edge vs Go go-edge MASQUE termination)"
  say "Running the existing G4 A/B bench with a short bounded duration"
  say "(see scripts/bench/edge_ab.sh directly for the full 1/10/100 sweep):"
  say "  \$ scripts/bench/edge_ab.sh --duration-secs 2 --concurrencies \"1 10\""
  BEAT4_CSV="$(mktemp -d)/edge_ab_demo.csv"
  BEAT4_LOG="$(mktemp)"
  bash "${SCRIPT_DIR}/bench/edge_ab.sh" --out-csv "${BEAT4_CSV}" --duration-secs 2 --concurrencies "1 10" \
    > "${BEAT4_LOG}" 2>&1
  BEAT4_RC=$?
  if [[ ${BEAT4_RC} -eq 0 && -s "${BEAT4_CSV}" ]]; then
    RUST_MBPS="$(awk -F, '$3=="rust.throughput.c10.sink_mbps"{print $4}' "${BEAT4_CSV}" | tail -1)"
    GO_MBPS="$(awk -F, '$3=="go.throughput.c10.sink_mbps"{print $4}' "${BEAT4_CSV}" | tail -1)"
    say "Rust helix-edge sink throughput @c=10: ${RUST_MBPS:-?} Mbps"
    say "Go go-edge sink throughput @c=10:      ${GO_MBPS:-?} Mbps"
    beat_pass "both MASQUE-termination edges built and benchmarked (CSV: ${BEAT4_CSV}; decision matrix already filled in spec §12/§7.3)"
  else
    beat_fail "edge_ab.sh did not complete cleanly (rc=${BEAT4_RC}, log: ${BEAT4_LOG})"
  fi
else
  beat_skip "bonus G4 beat (--skip-g4 given)"
fi

# -----------------------------------------------------------------------
# Optional real-root beat: the full 2-netns routed-LAN reachability
# rig (scripts/rig/*.sh) -- only runs with real root, mirroring
# scripts/spike.sh's own EUID check. Honestly SKIPPED otherwise, never
# faked.
# -----------------------------------------------------------------------
narrate "Optional beat — full 2-netns routed-LAN reachability (scripts/rig/)"
if [[ $EUID -eq 0 ]]; then
  say "Running as root — bringing up the real netns rig and testing reachability:"
  if bash "${SCRIPT_DIR}/rig/setup.sh" && bash "${SCRIPT_DIR}/rig/test_g1.sh"; then
    beat_pass "real 2-netns rig up, client reached the connector's simulated LAN host"
  else
    beat_fail "rig setup/reachability failed — real finding, not narrated over"
  fi
  bash "${SCRIPT_DIR}/rig/teardown.sh" || true
else
  beat_skip "requires real root (this sandbox: \`sudo -n true\` fails) — re-run this script with sudo for the fully-routed 2-netns proof; Beats 1-3 above already prove the transport + censorship-evasion + reconciliation mechanisms with real evidence, just not through the routed netns topology"
fi

# -----------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------
echo ""
echo "=============================================================="
echo "  Demo summary: ${BEATS_OK}/${BEATS_TOTAL} required beats passed"
echo "  Finished: $(date -Iseconds)"
echo "=============================================================="
echo ""
echo "For the full comparable-CSV measurement harness behind these"
echo "numbers (G1/G2/G4, HVPN-P0-077), run:"
echo "  ./scripts/bench/unified_harness.sh"
echo ""

[[ ${BEATS_OK} -eq ${BEATS_TOTAL} ]]
