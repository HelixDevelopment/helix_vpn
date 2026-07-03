# Helix VPN — Benchmark Harness

## Overview

The benchmark harness measures **latency**, **throughput**, **packet loss**, and
**jitter** through the Helix VPN test rig (network namespace topology with
client, server, and bridge namespaces). All results are recorded to
timestamped CSV files for automated comparison and historical tracking.

## Metrics collected

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

- `ping` (standard on all systems)
- `iperf3` — optional, enables TCP/UDP throughput tests
- `ncat` + `pv` — fallback path when `iperf3` is missing
- Root (sudo) — required to run within the `hx-server` network namespace
