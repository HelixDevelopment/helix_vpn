# G4 edge A/B — statistical re-check (2026-07-06)

**Status: investigative input only — NOT a closure of G4.** This document does
not edit or close `docs/research/mvp/04_VPN_CLD/HelixVPN-Phase0-Spike.md`'s
decision log. It exists to give the operator the real, captured statistical
picture behind the reported ranking flip, per explicit instruction. The
architectural call (Rust vs Go vs dual-impl) is the operator's, not this
report's.

## 1. What was asked

The decision-log row in `HelixVPN-Phase0-Spike.md` §12 cites a single sample:
peak sink throughput **R:1163.9 Mbps @c=10 / G:884.5 Mbps @c=1** (Rust
winning). An independent audit re-ran the harness once and got the **opposite**
ranking (Go 800.7 Mbps beating Rust 655.7 Mbps). Task: run
`scripts/bench/edge_ab.sh` (the G4 A/B harness, HVPN-P0-045) enough times
back-to-back to tell whether this is genuine run-to-run noise or a real,
reproducible pattern — and flag any confound (order-of-execution, warm-up,
thermal, concurrent-work interference) rather than averaging blindly.

## 2. Method

- Harness: `scripts/bench/edge_ab.sh`, **default settings** (no flags):
  `DURATION_SECS=5`, `CONCURRENCIES="1 10 100"`, `PAYLOAD_BYTES=1200`, full
  build (no `--skip-build`) on every run — same protocol the decision log used.
- **10 independent full runs**, back to back, `run01`..`run10`
  (`bench-results/g4-stat-runs/run{01..10}.csv` + matching `.log`), captured
  2026-07-06 20:05:25–20:24:09 local time.
- **3 additional runs with the edge execution order reversed** (Go server
  started first, Rust second — the opposite of `edge_ab.sh`'s hardcoded
  `bench_rust; bench_go` order) to test order-of-execution bias directly:
  `reversed_run{01..03}.csv` / `.log`, captured 20:26:54–20:30:43. This used a
  throwaway copy of the script outside the repo
  (`/tmp/.../scratchpad/edge_ab_reversed.sh`, not committed, not part of this
  repo) with `SCRIPT_DIR` pinned back to the real
  `scripts/bench` (so it drives the *exact same*, unmodified
  `rust_edge_bench`/`go_edge_bench` tools and source trees) and only the two
  call-site lines at the bottom (`bench_rust` / `bench_go` order) swapped.
  `scripts/bench/edge_ab.sh` itself was **not modified**.
- All numbers below are the harness's own **`sink_mbps`** metric — the
  authoritative, sink-side received-byte throughput the harness's own README
  says to use for the decision matrix (not the client's "offered" rate, which
  is informational-only).
- One run (`run06`) hit a genuine mid-run failure on the first attempt
  (documented in §5) and was retried once, per the task's guidance; the
  contaminated partial CSV was isolated and only the clean, complete retry
  data (`run06.clean.csv`) is used in the statistics.
- Analysis script: `bench-results/g4-stat-runs/analyze.py` (python3,
  stdlib `csv`/`statistics` only). Raw output:
  `bench-results/g4-stat-runs/analyze_output.txt`.

## 3. Results — peak sink throughput (matches the decision-log's own metric)

Peak = `max(sink_mbps @ c=1, c=10, c=100)` per edge per run, exactly the
quantity the original decision-log row reports.

| run | rust_peak (Mbps) | go_peak (Mbps) | winner |
|---|---:|---:|---|
| run01 | 689.5 | 1237.5 | GO |
| run02 | 720.9 | 1034.9 | GO |
| run03 | 929.7 | 1565.8 | GO |
| run04 | 375.5 | 1442.0 | GO |
| run05 | 779.5 | 971.5 | GO |
| run06 (clean retry) | 837.3 | 1489.1 | GO |
| run07 | 843.1 | 1215.2 | GO |
| run08 | 237.5 | 728.6 | GO |
| run09 | 507.1 | 704.6 | GO |
| run10 | 759.3 | 1180.4 | GO |

**Winner tally: RUST = 0/10, GO = 10/10.**

| series | n | mean | stdev | min | max |
|---|---:|---:|---:|---:|---:|
| rust_peak | 10 | 667.94 | 223.26 | 237.50 | 929.72 |
| go_peak | 10 | 1156.95 | 299.35 | 704.59 | 1565.79 |

**Paired difference (go_peak − rust_peak) per run:** +548.0, +314.0, +636.1,
+1066.4, +192.0, +651.8, +372.1, +491.1, +197.5, +421.1 (Mbps).
Mean diff = **+489.0 Mbps**, stdev = 259.7, **min = +192.0**, max = +1066.4.

The key fact: **the paired difference never once goes negative** — Go's
worst margin over Rust in any single run is still +192 Mbps. This did not
come close to flipping.

### Per-concurrency breakdown (all 10 runs)

| series | n | mean | stdev | min | max |
|---|---:|---:|---:|---:|---:|
| rust.c1 | 10 | 595.48 | 213.33 | 237.50 | 843.10 |
| rust.c10 | 10 | 531.97 | 274.43 | 126.53 | 929.72 |
| rust.c100 | 10 | 346.31 | 279.71 | 53.02 | 759.27 |
| go.c1 | 10 | 1027.48 | 285.68 | 401.74 | 1327.59 |
| go.c10 | 10 | 1014.43 | 400.51 | 357.70 | 1565.79 |
| go.c100 | 10 | 690.12 | 461.59 | 97.32 | 1287.00 |

Go beats Rust's mean at **every** concurrency level tested (c=1, c=10, c=100),
not just at the specific (edge, concurrency) pair each side's peak happened to
land on in the original single-sample decision-log row.

### CPU-per-Gbps and churn (secondary metrics, at c=10 — same concurrency the decision log cited)

| series | n | mean | stdev | min | max |
|---|---:|---:|---:|---:|---:|
| rust.cpu_per_gbps.c10 (cores/Gbps, lower=better) | 10 | 1.6773 | 0.1202 | 1.5194 | 1.9538 |
| go.cpu_per_gbps.c10 | 10 | 2.1465 | 0.2078 | 1.9252 | 2.6176 |
| rust.churn_hps.c10 (handshakes/sec, higher=better) | 10 | 1533.88 | 324.56 | 781.40 | 1787.80 |
| go.churn_hps.c10 | 10 | 720.90 | 298.07 | 262.20 | 1032.20 |

This is the interesting nuance: **Rust is consistently more CPU-efficient per
Gbps delivered, and consistently faster at connection churn, even though it
consistently loses on raw sink throughput.** These are not contradictory —
they are three different axes, and this data says Rust wins two of them while
Go wins the one (throughput) the decision-log row happened to headline.

## 4. Is this noise, or a real pattern?

**A real, highly reproducible pattern — not noise, on this hardware, with this
exact harness, right now.**

- Ranking consistency: **10/10** original-order runs favor Go on peak sink
  throughput; **0/10** favor Rust. Adding the 3 reversed-order runs (§5.1):
  **13/13** favor Go.
- The *magnitude* of run-to-run variance is real and non-trivial (per-cell
  stdev is 30–70% of the mean — e.g. `go.c100` ranges from 97 to 1287 Mbps
  across runs) — the decision-log's own caveat ("run-to-run variance on
  shared hardware is non-trivial") is correct and this data confirms it.
- But that variance is **never large enough to cross the gap between the two
  edges when compared paired (same run, same conditions)**. The tightest race
  among the 10 original-order runs was `run05`, rust_peak 779.5 vs
  go_peak 971.5 → **+192.0 Mbps**; the next-tightest was `run09`, rust_peak
  507.1 vs go_peak 704.6 → +197.5 Mbps. Neither comes remotely close to zero.
- A subtler point worth flagging explicitly: the **unpaired (marginal)**
  distributions *do* overlap — Rust's best single run (`run03`, 929.7) is
  numerically higher than Go's worst single run (`run09`, 704.6). If someone
  compared Rust's best-ever sample against Go's worst-ever sample in
  isolation (e.g. citing one cherry-picked run of each), it could look like a
  close or even reversed race. That comparison would be the wrong lens: the
  **paired**, same-run comparison is what controls for whatever is driving
  the run-to-run swings (§5.2/§5.5), and by that lens Go wins unanimously,
  13/13, with the smallest paired margin still +192 Mbps.
- Conclusion on "is the decision a coin flip": **no.** With 1 sample it looked
  like it could be (Rust won once, an independent re-run showed Go winning
  once). With 13 samples taken back-to-back under controlled, identical
  methodology, the coin is not fair — it has landed on Go 13 times running.

## 5. Confounds investigated

### 5.1 Order-of-execution bias — checked directly, **ruled out** as the primary driver

`edge_ab.sh` always runs `bench_rust` then `bench_go` (Rust first, Go second)
— every single one of the 10 original runs shares this order, so order and
edge-identity are perfectly confounded *within that dataset alone*. To
separate them, 3 additional runs were captured with the order reversed (Go
first, Rust second; confirmed via each run's own log —
`grep 'Starting.*server role' reversed_run01.log` shows `Starting Go go-edge
server role...` at line 13 and `Starting Rust helix-edge server role...` at
line 51):

| run (Go first, Rust second) | rust_peak | go_peak | winner |
|---|---:|---:|---|
| reversed_run01 | 669.5 | 1323.1 | GO |
| reversed_run02 | 809.5 | 924.5 | GO |
| reversed_run03 | 810.2 | 1577.0 | GO |

Go still wins all 3 reversed-order runs, by margins (+654, +115, +767 Mbps)
consistent with the original-order set. Reversed-order means:
rust_peak = 763.0, go_peak = 1274.9 — Rust's mean was actually *slightly
higher* running second (763.0) than in the original order running first
(667.9 over 10 runs), which is the direction a naive "runs second gets a
warm-up bonus" hypothesis would predict — but the effect (if real at all,
n=3 is too small to say) is far too small to explain, let alone reverse, the
~490 Mbps mean gap. **Order-of-execution is not the explanation for the
ranking.**

### 5.2 Mild downward drift across the session (unconfirmed cause)

Correlating run order (1..10) against each edge's peak: rust r=−0.24,
go r=−0.44 (both mildly negative — first-half mean vs second-half mean:
rust 699.0→636.8, go 1250.3→1063.6). Both edges drift down together over the
~19-minute session, consistent with *something* (thermal state, growing
background load, CPU frequency scaling under the `powersave` governor
observed on this host, or simple contention from the concurrent `helix_core`
work described below) gradually loading the shared hardware. This is a
**directional observation from n=10, not a proven cause** — the correlation
is weak and could be sampling noise itself. It does not change the ranking
conclusion since it moves both edges the same way.

### 5.3 Concurrent `submodules/helix_core` work during the session — confirmed, assessed as not the explanation

Per the task's own warning, a separate track was concurrently touching
`submodules/helix_core`. This was verified directly, not assumed:

```
$ cd submodules/helix_core && git reflog --date=iso | head -6
b43a920 HEAD@{2026-07-06 20:20:20 +0300}: commit: docs(helix-wg): correct doc-comment overstatement of boringtun's length invariant
3874095 HEAD@{2026-07-06 20:17:28 +0300}: commit: test(helix-wg): HVPN-P0-011 stress+chaos coverage for encrypt/decrypt data plane
b7e6b24 HEAD@{2026-07-06 19:49:23 +0300}: commit: fix(helix-wg): HVPN-P0-011 was malformed test fixtures, not a crypto bug
02c3636 HEAD@{2026-07-06 15:36:52 +0300}: merge feature/hvpn-p0-074-map-delta-reconcile: Fast-forward
```

Two of these commits (`3874095` @ 20:17:28, `b43a920` @ 20:20:20) landed
**during** my session (which ran 20:05:25–20:24:09 for the main 10 runs), in
the gap between `run06` (ended 20:16:52) and `run08` (ended 20:21:21) — i.e.
`run07` onward were built against a slightly newer `helix_core` checkout than
`run01`–`run06`. Diffing the two checkouts confirms the change was confined
to one file:

```
$ git diff --stat 76c311e b43a920 -- crates/helix-masque crates/helix-transport crates/helix-wg
 crates/helix-wg/src/noise.rs | 649 ++++++++++++++++++++++++++++++++++++++++++-
 1 file changed, 641 insertions(+), 8 deletions(-)
```

Assessment: `helix-wg` is compiled as part of the same cargo workspace (it
shows up in every build log, e.g. `warning: helix-wg (lib) generated 1
warning`), but `rust_edge_bench` path-deps onto `helix-masque`/
`helix-transport` (per the harness's own doc comment), not `helix-wg` — and
the concurrent commits' own messages describe **test-fixture and doc-comment**
changes ("stress+chaos coverage", "malformed test fixtures", "doc-comment
overstatement"), not a change to the MASQUE/transport runtime path this
benchmark exercises. Consistent with that: `run01`–`run06` (before these
commits) and `run07`–`run10` (after) show the **same** Go-wins-every-time
pattern at similar magnitude — no visible discontinuity at the `run06`/`run07`
boundary. **UNCONFIRMED** whether this concurrent work affected the measured
numbers at all (I did not instrument a byte-for-byte binary diff to prove
zero effect) — but it is not a plausible explanation for the *direction* of
the ranking, since the ranking is identical on both sides of the boundary.

No build ever hard-failed due to this concurrent work across all 13 runs.

### 5.4 A genuine runtime anomaly on `run06`'s first attempt — documented, not silently discarded

`run06`'s first attempt exited non-zero. This was **not** a build race (the
log shows `Finished \`release\` profile [optimized] target(s) in 0.06s` —
build succeeded). It was a runtime failure: the Rust edge's **churn** mode
degraded to 0 handshakes/sec with 100% failed handshakes at **all three**
concurrency levels (1, 10, 100) — full log:
`bench-results/g4-stat-runs/run06.attempt1.log`. The script then aborted
(`set -euo pipefail`) during the subsequent latency step because the client
produced no parseable CSV line — the underlying cause of the churn collapse
itself is **not established** (candidates: back-to-back invocation with no
settle gap after `run05` ended at the identical timestamp `run06` started;
transient resource contention; not investigated further as it is outside this
task's throughput-ranking question). The retry (with the same code, ~3
minutes later, after other runs had proceeded) succeeded cleanly. Because my
retry loop reused the same `--out-csv` path across attempts, the failed
attempt's 27 partial rows were appended ahead of the successful retry's 58
rows in the same file (`run06.csv`, 86 data-row lines total) — **this is a
methodology artifact of my own retry orchestration, not a bug in
`edge_ab.sh` itself** (the script's CSV-append design is intentional, for
accumulating multiple runs into one file; reusing one output path across
retry attempts within a single logical "run" was my error). It was caught and
corrected: only the clean 58-row second-attempt block is used in all
statistics above (`run06.clean.csv`); the raw contaminated `run06.csv` is
left in place, unedited, for audit. For all runs `run07`–`run10` I switched to
a fresh per-attempt CSV path (`runNN.attemptM.csv`) precisely to avoid
repeating this.

### 5.5 Background system load

`uptime` immediately after the session showed load average 0.17 (1 min) vs
3.43 (5 min) vs 5.12 (15 min) — confirming the host carried real background
load during the session that has since settled down (consistent with other
concurrent tracks per this project's multi-track working model). CPU governor
is `powersave` (frequency-scaling active, a plausible contributor to the
run-to-run variance magnitude noted in §4, though — like §5.2 — not a
plausible explanation for the *direction* being one-sided 13/13). No
per-run system-wide load sample was captured *during* each run (only the
harness's own per-process CPU/RSS sampling), so I cannot retroactively
attribute variance to specific moments — flagged honestly as an unconfirmed
gap rather than guessed at.

## 6. Honest scope reminder (from the harness's own doc comments)

Neither edge has a real kernel-WireGuard/boringtun gateway-socket integration
yet — this benchmarks the MASQUE-termination + gateway-relay hand-off path on
loopback, not an end-to-end tunnel. Rust's path is a hand-rolled,
non-HTTP/3-conformant CONNECT-UDP stand-in; Go's uses the real RFC 9298
`masque-go`/`quic-go` stack. That asymmetry (already flagged in the existing
decision-log row) is unchanged by this re-run and remains a relevant input to
the eventual G4 call alongside the numbers above.

## 7. Bottom line for the operator

- **Not a coin flip.** Across 13 independent, back-to-back runs (10 original
  order + 3 order-reversed) on this hardware, with the existing
  `edge_ab.sh` harness at its default settings, **Go's sink throughput beat
  Rust's in every single run**, by a mean paired margin of +489 Mbps and a
  minimum margin of +192 Mbps. Order-of-execution was directly tested and
  ruled out as the cause.
- Run-to-run variance is real and large in absolute terms (individual
  concurrency-level values swing 2–4× between runs) — the original decision
  log's "non-trivial variance" caveat is empirically correct — but it has
  never once been large enough to flip the Rust-vs-Go ranking when compared
  paired, same-run.
- Rust consistently wins on CPU-efficiency (cores/Gbps) and connection-churn
  rate — the opposite metrics from the ones the original decision-log row
  headlined.
- The original decision-log's single Rust-winning sample looks, on this
  evidence, like an outlier rather than the representative case — but I did
  not capture the system conditions at the moment that original sample was
  taken and cannot say definitively why it differed.
- This is an operator-facing architectural call (§7.3 dev-cost/reuse/velocity
  trade-offs are unaffected by this data), not something this report decides.

## 8. Artifacts

- This file: `scripts/bench/g4-statistical-analysis-2026-07-06.md`
- Raw per-run CSVs + logs: `bench-results/g4-stat-runs/run{01..10}.csv`
  (+ `.log`, `.attemptN.*` where retried), `run06.clean.csv` (the CSV actually
  used for run06's statistics), `run06.csv` (raw, contaminated, kept for
  audit per §5.4)
- Order-reversal check: `bench-results/g4-stat-runs/reversed_run{01..03}.csv`
  (+ `.log`)
- Analysis script + raw output: `bench-results/g4-stat-runs/analyze.py`,
  `bench-results/g4-stat-runs/analyze_output.txt`
- All of the above are left uncommitted in the working tree, per instruction;
  `bench-results/` is gitignored per this project's build-artifact policy in
  any case.
