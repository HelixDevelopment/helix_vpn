# Helix VPN — Benchmark Harness

## Overview

The benchmark harness covers two independent measurement suites, both
writing into the same CSV schema (`timestamp,test_type,metric,value,unit`)
so one file can be diffed/compared/tabulated uniformly:

1. **netns-rig suite** (`run.sh`'s `latency_test`/`throughput_test`) —
   **latency**, **throughput**, **packet loss**, and **jitter** through the
   Helix VPN test rig (network namespace topology with client, server, and
   bridge namespaces per `scripts/rig/`). Requires root + `iperf3`.
2. **G4 edge A/B suite** (`edge_ab.sh`, HVPN-P0-045) — drives the REAL Rust
   `helix-edge` and Go `go-edge` MASQUE-termination binaries through the
   identical protocol from `HelixVPN-Phase0-Spike.md` §7.2 (throughput at
   1/10/100 concurrent flows, CPU-per-Gbps, added latency p50/p99,
   connection-churn handshakes/sec, memory-under-churn) so the §7.3
   decision matrix can be filled mechanically via `decision_matrix.sh`.
   Runs entirely on loopback — **no root, no iperf3 needed**. `run.sh`
   invokes it automatically (disable with `--skip-edge-ab`).

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
