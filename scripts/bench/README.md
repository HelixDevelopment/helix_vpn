# Helix VPN — Benchmark Harness

## Overview

The benchmark harness covers three measurement suites:

1. **netns-rig suite** (`run.sh`'s `latency_test`/`throughput_test`) —
   **latency**, **throughput**, **packet loss**, and **jitter** through the
   Helix VPN test rig (network namespace topology with client, server, and
   bridge namespaces per `scripts/rig/`). Requires root + `iperf3`. Writes
   `timestamp,test_type,metric,value,unit`.
2. **G4 edge A/B suite** (`edge_ab.sh`, HVPN-P0-045) — drives the REAL Rust
   `helix-edge` and Go `go-edge` MASQUE-termination binaries through the
   identical protocol from `HelixVPN-Phase0-Spike.md` §7.2 (throughput at
   1/10/100 concurrent flows, CPU-per-Gbps, added latency p50/p99,
   connection-churn handshakes/sec, memory-under-churn) so the §7.3
   decision matrix can be filled mechanically via `decision_matrix.sh`.
   Runs entirely on loopback — **no root, no iperf3 needed**. `run.sh`
   invokes it automatically (disable with `--skip-edge-ab`). Writes
   `timestamp,test_type,metric,value,unit`.
3. **Unified G1/G2/G4 harness** (`unified_harness.sh`, HVPN-P0-077) — the
   "one harness ... for every transport x edge combination" §8 calls for.
   See its own section below. Writes a DIFFERENT, wider schema:
   `timestamp,gate,transport,edge,metric,value,unit,pass_bar,verdict,
   method,note`.

The first two suites share one schema so their files can be
diffed/compared/tabulated uniformly (`compare.sh`); the unified harness's
CSV is intentionally wider (it needs `gate`/`pass_bar`/`verdict`/`method`
columns to be self-describing across G1/G2/G4 in one file) and is its own
format — read it directly or with `awk -F,`/`jq`-over-`csv`, not with
`compare.sh`.

## Unified G1/G2/G4 measurement harness (`unified_harness.sh`)

```bash
# Full run: G1 (plain-UDP baseline) + G2 (MASQUE through a DPI-style
# block) + G4 (Rust-vs-Go edge A/B), one CSV
./scripts/bench/unified_harness.sh

# Faster / narrower runs
./scripts/bench/unified_harness.sh --skip-g4                 # G1+G2 only, ~35s
./scripts/bench/unified_harness.sh --skip-loss                # skip the ~25s tc-netem loss-resilience phase
./scripts/bench/unified_harness.sh --out-csv /path/to/out.csv
```

This is `edge_ab.sh` generalized to also cover G1 and G2, per
`HelixVPN-Phase0-Spike.md` §8: "One harness, run for every transport x
edge combination so results are comparable." It does **not** reimplement
any transport/probe/edge logic — it drives `submodules/helix_core`'s
`g2-dpi-probe` binary (via its `overhead`/`dpi-survival`/`loss-resilience`
subcommands) + that submodule's own `scripts/spike/g2_dpi_masque_unpriv.sh`
sandboxed rig (real `nft` DPI block inside `unshare --net --user`, real
`AF_PACKET` wire-fingerprint capture) + this directory's own `edge_ab.sh`,
and normalizes their real JSON/CSV output into one comparable CSV.

**Verdict vocabulary** (the `verdict` column, closed set):

| Verdict | Meaning |
|---|---|
| `PASS` / `FAIL` | The row's real measured value was checked against its `pass_bar` and genuinely passed/failed. |
| `RECORDED` | Spec §8 says "record" for this metric (no PASS/FAIL bar) — the number is captured, not judged. |
| `NOT_APPLICABLE` | This metric's pass-bar text doesn't apply to this gate (e.g. `wire_fingerprint` for G1 — plain WireGuard isn't hiding). |
| `NOT_MEASURED` | This run's tooling doesn't exercise this metric for this gate (e.g. `edge_ab.sh` has no loss-impairment step) — see the row's `note` for the project-wide source of truth. |
| `SKIP` | Genuinely unmeasurable in this Phase-0 codebase/sandbox right now (e.g. `reconnect_roam` — no real up tunnel + flappable interface exists yet). |
| `UNMEASURED_VS_BAR` | The number itself is real, but no valid comparator exists to judge it against its bar (e.g. G1's own throughput number IS the closest thing to a "bare link" reference — there's nothing distinct to compute a percentage against). |

**Honest scope** (read before trusting a number this script produces):
this Phase-0 codebase has **no real end-to-end WireGuard dataplane wired
up yet** (no TUN device, no client-gateway-connector process chain running
simultaneously — only crate-level tests and probe binaries exist). So
G1/G2's `through_tunnel_throughput_mbps` and `added_latency_ms` are
loopback **transport-primitive** numbers (the same code a future tunnel
will carry WG datagrams over), not real tunnel measurements — every such
row's `note` column says so explicitly. `iperf3`, `tshark`, and `tcpdump`
are absent from the sandbox this harness was authored in (confirmed via
`command -v`); the loss-resilience, throughput, and wire-fingerprint
numbers reuse the real hand-rolled stand-ins the G2 work already built
(paced offered-load goodput comparison, `AF_PACKET` sniffer, `nft`-in-
`unshare` DPI block) rather than reinventing them.

**A real bug found (and worked around, not silently patched over) while
building this harness:** `g2_dpi_masque_unpriv.sh`'s own internal
loss-resilience phase (in `submodules/helix_core`, out of this task's file
scope) runs both its offered-load sub-tests as plain statements under
`set -euo pipefail`. `g2-dpi-probe`'s `loss-resilience` subcommand's exit
code communicates its measured PASS/FAIL **verdict** (exits 1 on a real
"FAIL", the exact same convention its `overhead`/`dpi-survival`
subcommands use) — and a "FAIL" is the historically expected, honestly
documented real outcome of that specific test (see
`submodules/helix_core/G2-RESULTS.md` §5). So the wrapper aborts before
its second sub-test and its own `tc` cleanup whenever that happens —
reproduced directly (running it unmodified left only one of the two
expected evidence files on disk, with no explicit error text). This
harness works around it by re-running both offered-load sub-tests itself,
reusing the identical technique (same binary, same `unshare`, same `tc
netem loss 5% delay 40ms 10ms`) with tolerant error handling, rather than
editing that out-of-scope script.

## G4 Edge A/B benchmark (`edge_ab.sh`)

```bash
# Build + run both edges through the full 1/10/100 sweep (default)
./scripts/bench/edge_ab.sh
# or: make bench-edge-ab

# Fill the §7.3 decision matrix from the resulting CSV
./scripts/bench/decision_matrix.sh bench-results/edge_ab-<timestamp>.csv
# or: make decision-matrix
```

Implementation: `scripts/bench/tools/rust_edge_bench/` (standalone Rust
binary, path-deps on `submodules/helix_edge` + `submodules/helix_core`'s
`helix-masque`/`helix-transport` crates) and
`scripts/bench/tools/go_edge_bench/` (standalone Go module, `replace`
directive onto `submodules/helix_go`) drive the REAL, already-built
`helix_edge::edge::spawn_edge` entry point and the REAL
`masqueedge.NewServer`/`masque-go`/`quic-go` stack — the exact production
code each edge's own `main.rs`/`main.go` calls. Neither tool modifies
`helix_core`, `helix_edge`, or `helix_go`.

Each tool runs as two separate OS processes (`--role server` / `--role
client`, mirroring `iperf3 -s`/`-c`) so `edge_ab.sh` can attribute CPU
(`/proc/<pid>/stat`) and peak RSS (`/proc/<pid>/status`) to the edge
process alone. Metrics land as `test_type=edge_ab`,
`metric=<edge>.<mode>.c<concurrency>.<submetric>`, e.g.
`rust.throughput.c10.sink_mbps`, `go.churn.c100.handshakes_per_sec`.

**Honest scope** — read before trusting a number: neither edge has a real
kernel-WireGuard/boringtun gateway-socket integration yet (this is a
Phase-0 spike gap independent of any sandbox constraint — see each edge's
own README/doc comments). This benchmark therefore measures the MASQUE
termination + gateway-relay hand-off data path itself against a real
loopback UDP sink, not an end-to-end WireGuard tunnel. Throughput is
reported from two angles: the client's own "offered" rate (informational
— a QUIC unreliable-datagram sender's local accept-into-queue rate, not
proof of delivery) and the edge's loopback sink's actually-received byte
count (`sink_mbps` — authoritative, positive sink-side evidence).

## netns-rig suite: metrics collected

| Metric         | Tool / source | Unit  | Description                                     |
|----------------|---------------|-------|-------------------------------------------------|
| Latency (avg)  | `ping`        | ms    | Round-trip time to the server address           |
| Latency (min)  | `ping`        | ms    | Minimum RTT observed                            |
| Latency (max)  | `ping`        | ms    | Maximum RTT observed                            |
| Latency (mdev) | `ping`        | ms    | Standard deviation (jitter proxy)               |
| Packet loss    | `ping`        | %     | Percentage of packets lost                      |
| Throughput     | `iperf3`      | Mbps  | TCP stream throughput (preferred)               |
| Throughput     | `ncat` + `pv` | B/s   | Fallback when `iperf3` is unavailable           |
| Jitter         | `iperf3`      | ms    | UDP jitter (when using UDP test mode)           |
| Datagram loss  | `iperf3`      | %     | UDP datagram loss rate (when using UDP mode)    |

## Output format

Results are written to a CSV file at the path specified by `--output`
(default `./bench-results/bench-<timestamp>.csv`).

```
timestamp,test_type,metric,value,unit
2025-07-04T12:00:00+00:00,latency,avg,12.34,ms
2025-07-04T12:00:00+00:00,latency,min,10.01,ms
2025-07-04T12:00:00+00:00,latency,max,15.67,ms
2025-07-04T12:00:00+00:00,throughput,tcp,94.20,Mbps
2025-07-04T12:00:00+00:00,jitter,avg,0.87,ms
2025-07-04T12:00:00+00:00,packet_loss,percent,0.0,%
```

## Usage

### Running a benchmark suite

```bash
# Default — 30-second tests, output to ./bench-results/
./scripts/bench/run.sh

# Custom duration and output directory
./scripts/bench/run.sh --duration 60 --output /tmp/my-bench

# Custom server address (default: 10.0.240.3)
./scripts/bench/run.sh --server-addr 10.0.240.10

# Via Make
make bench
```

### Comparing results

```bash
# Compare two CSV files
./scripts/bench/compare.sh bench-results/bench-001.csv bench-results/bench-002.csv

# Via Make (compares last two CSV files in bench-results/)
make bench-compare
```

## Adding new benchmarks

Edit `run.sh` and add a new `test_type` section following the existing
pattern. Each test must call `log_result` with the 4-tuple:
`(test_type, metric, value, unit)`.

## Requirements

### netns-rig suite

- `ping` (standard on all systems)
- `iperf3` — optional, enables TCP/UDP throughput tests
- `ncat` + `pv` — fallback path when `iperf3` is missing
- Root (sudo) — required to run within the `hx-server` network namespace

### G4 edge A/B suite (`edge_ab.sh`)

- `cargo` (to build `rust_edge_bench` + the real `helix-edge`/`helix-masque`
  dependency tree) and `go` (to build `go_edge_bench` + the real
  `masque-go`/`quic-go` dependency tree)
- `bc`, `awk`, `getconf`, `/proc` (Linux) — for CPU/RSS sampling and the
  sink-throughput/CPU-per-Gbps computation
- No root, no `iperf3` — both edges terminate MASQUE on loopback
