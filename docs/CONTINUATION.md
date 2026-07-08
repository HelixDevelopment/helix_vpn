# Helix VPN — Session Continuation File

**Revision:** 21
**Last modified:** 2026-07-07T20:49:00Z

> Helix Constitution §11.4.131 — standing session-resumption artifact.
> Re-read this file at the start of any new session before touching code.
> A fresh session can resume with ONLY this file's path plus
> `.remember/remember.md` — both are kept byte-accurate against live
> `git rev-parse`/`workable-items validate` output, never against
> in-context memory or a prior handoff's text alone (see the
> multi-session lesson below).

---

## ROUND 7: PHASE-0 CLOSEOUT + PHASE-1 GATE-HARDENING KICK-OFF (2026-07-07)

**Started:** 2026-07-07T19:00:00Z (session resumed mid-context; found the repo
had already progressed through Rounds 5/6 via prior background work this
session had no in-context memory of — re-verified everything from git/DB
state before acting, per this file's own multi-session lesson below).
**Landed:** 2026-07-07T20:49:00Z (this entry).
**Trigger:** operator's repeated "continue endless autonomous loop, 3-4
parallel subagents, rock-solid evidence, no bluff" directive.

**What happened, in order:**

1. **Landed uncommitted work found sitting in the tree**: a 13-run
   statistical re-check of the G4 Rust-vs-Go edge benchmark
   (`scripts/bench/g4-statistical-analysis-2026-07-06.md`) plus its
   matching decision-log update — resolves the earlier single-sample
   ranking-flip ambiguity: **Go wins throughput 13/13** (mean +489 Mbps)
   while **Rust wins CPU-efficiency and connection-churn 13/13** — a
   genuine three-axis trade-off, not a coin flip. G4's edge-language call
   remains an explicit open operator decision (not closed by this data).
   Root commit `b218a26`.
2. **Retried HVPN-P0-055/058 (Android) and HVPN-P0-061/064 (iOS)** — the
   previous round's attempts at these had been lost when their scratch
   worktrees got wiped by a process restart before committing (confirmed:
   both branches were still at their unmodified base commit). Re-dispatched
   with an explicit "commit after every increment" instruction this time.
   - **iOS**: real `apple/helix-ios-ffi` C-ABI crate + genuine `cbindgen`
     header generation (fixing a real cbindgen `Option<fn>` codegen bug
     along the way) + complete Swift `NEPacketTunnelProvider` skeleton.
     Rust cross-compile to `aarch64-apple-ios` confirmed blocked at
     `ring`/`boringtun`'s `build.rs` (no Apple SDK on Linux — exact error
     captured, an unavoidable wall for any non-Mac host). **A background
     security review of the committed Swift file found 3 real
     memory-safety/lifecycle issues** (use-after-free from an unretained
     C-side self-pointer; the inbound callback and the outbound pump loop
     both continuing to act after `stopTunnel`) — the controller traced
     the exact C ABI contract in `engine.rs`, confirmed
     `helix_core_stop()` genuinely blocks until the engine thread joins,
     and fixed all 3 (verified by careful manual read-through only — this
     file cannot be compiled without Xcode). `helix_shims` commit
     `9eb51bb`.
   - **Android**: real Gradle app module (`HelixVpnService`, JNI decls,
     config parsing), 20/20 unit tests, a signed debug APK genuinely
     assembled. Rust cross-compile blocked on an isolated-worktree
     environment gap (no sibling `helix_core` checkout — honestly
     captured, not worked around by bypassing sandbox permissions).
     **A genuine emulator run** (KVM-accelerated AVD via the `containers`
     submodule's own orchestrator — the fully-containerized image path
     was blocked by a private-registry 401, honestly reported and
     substituted) **found and fixed a real crash**: `UnsatisfiedLinkError`
     is a `LinkageError` (an `Error`, not an `Exception`), so an existing
     `catch (Exception)` never caught it during cleanup, killing the app
     process — fixed via TDD (RED reproduced with Robolectric, then
     GREEN) with a `nativeLoaded` guard flag. Never described as a real
     device — "emulator" throughout. `helix_shims` commit `3a3ba03`.
3. **Independent adversarial audit** of the freshest, highest-regression-risk
   work (the G4 statistics, the `HVPN-P0-011` boringtun-fixture fix, the
   `HVPN-P0-049` FFI-projector fix) — re-ran everything from scratch
   (independently recomputed the G4 statistics bit-for-bit from raw CSVs,
   did the WireGuard fixture byte-math by hand against boringtun's actual
   vendored source, reproduced both original failures by reverting just
   the fix). **All 3 CONFIRMED**, no bluffs found.
4. **`HVPN-P1-001` (G1 gate, Phase-1 rigor)** — deliberately chosen as this
   round's Phase-1 entry point because the workable-items DB shows it has
   zero dependencies (the actual root of the `P1-010→P1-005→P1-001` chain)
   — hardening it does not jump ahead of any still-open Phase-0 gate.
   Real statistical BENCH (RTT percentiles + sustained throughput) +
   E2E (50 verified round trips, simulated drop, fresh handshake/roam,
   50 more verified round trips — recovery 8-11ms vs a <3s bar).
   `helix_core` commit `338fae1`.
5. **Closing `HVPN-P1-001` unblocked `HVPN-P1-002`/`005`/`006`** (all three
   listed it as their only dependency) — dispatched in parallel alongside
   the Android retry, bringing the round back up to 4 concurrent streams:
   - **`HVPN-P1-002` (G2)**: real BENCH (QUIC latency/throughput, honest
     RFC 9221 congestion-ceiling measurement), 6 adversarial SEC tests,
     a sustained 20-round E2E survival test. **Found and fixed 2 real bugs**
     while building the E2E test: a `quinn` connection-drop race, and a
     stale-connector-reuse bug that silently turned a negative control
     into a no-op. `helix_core` commit `f45fd29`.
   - **`HVPN-P1-005` (G5)**: UI-facing contract hardened (14→23 tests) —
     late subscription, concurrent subscribers, dead-subscriber safety,
     concurrent start/stop correctness, 3-cycle E2E. Honest scope note:
     no Dart/Flutter toolchain in this sandbox, so "UI" was tested as the
     contract a UI would consume, not faked. Real positive finding: a
     stopped session's sink genuinely closes (→ Dart's `onDone`), not
     just goes silent — a prior test assumption was corrected.
     `helix_shims` commit `a323196`.
   - **`HVPN-P1-006` (G6)**: 3 new scenarios (rapid successive edits,
     malformed intermediate `map.json` write, 4-peer no-cross-contamination)
     — no bug found, existing reconciler design already correct.
     `helix_core` commit `c653011`.

**Every merge in this round was independently re-verified by the
controller** (re-ran the relevant crate's/module's test suite fresh,
post-merge, before pushing) — not just trusted from the implementing
subagent's own report. Two accidental compiled-binary commits from an
earlier round's `.gitignore` filename mismatch were also caught and fixed
forward (never rewriting history).

**DB state after this round:** 485 items, `validate` PASS/0 issues. P0: 29
`Completed`, 3 `Fixed`, 1 `In progress` (`HVPN-P0-052`, still blocked on
`libgtk+3-devel`/root), 4 `Queued` (`HVPN-P0-067` memory-soak + the
`S5`/`S6`/`S7` milestone rollups — all genuinely hardware-blocked). P1: 4
`Completed` (`001`/`002`/`005`/`006`), 206 `Queued`.

**Still open / needs an operator decision:**
- **G4 edge-language call** (Rust vs Go) — a real three-axis trade-off
  (throughput vs CPU-efficiency/churn), not resolvable by more
  benchmarking alone; flagged to the operator, not decided unilaterally.
- **`HVPN-P0-052`** (Flutter-Linux toggle) — blocked on `libgtk+3-devel`
  needing root, confirmed no passwordless sudo in this environment.
- **`HVPN-P0-058`/`064`/`067`-class real-device work** — Android/iOS
  hardware and Mac/Xcode are genuinely required for the remaining scope;
  precisely enumerated in `android/README.md` and `apple/README.md` for
  whoever has the physical devices.
- **`HVPN-P1-003`** (G3 gate at Phase-1 rigor) — same hardware block as
  Phase-0's G3, not attempted again this round.

---

## ROUND 6: 4-STREAM PARALLEL PHASE-0 CLOSEOUT + 2 REAL REGRESSIONS FOUND (2026-07-06)

**Started:** 2026-07-06T19:04:00Z (after fixing Round 5's stale resumption
prompt — see the entry directly below)
**Landed:** 2026-07-06T16:39:13Z (this entry)
**Trigger:** operator instruction to run the endless autonomous loop with
≥3-4 parallel subagent streams on every genuinely parallelizable workable
item, with a hard "rock-solid evidence, no bluff" requirement.

**What happened:** dispatched 4 parallel subagent tracks (disjoint file
scope) plus did direct controller-level DB work between their reports.
All 4 hit an external Claude-API session-limit wall mid-flight (reset 7pm
Europe/Moscow) and were resumed via targeted continuations (not blind
restarts) once it cleared — no work was lost, all 4 had crashed before
writing any files.

**Track A — HVPN-P0-052 (Flutter-Linux toggle, `submodules/helix_ui`):**
real `app_access` Flutter project scaffolded (`flutter create
--platforms=linux`), G5 happy-path toggle+status-chip screen wired to a
swappable `HelixCoreBridge` interface whose only production
implementation honestly throws `UnimplementedError` (never fabricates
data) since no real Rust FFI wiring exists yet. 9/9 widget tests passing,
`flutter analyze` 0 issues, deterministic across 3 runs. **Confirmed
blocker**: `flutter build linux` needs `libgtk+3-devel`, which needs root
(no sudo password available) — an environment constraint, not a design
gap. Separately discovered `helix-ffi`'s FRB wiring isn't possible
regardless: hand-rolled `frb_stub::StreamSink`, no `cdylib` crate-type —
**and it doesn't currently compile at all** (see the regression below).
Item left `In progress`, not closed. `helix_ui` commit `1751701`, pointer
bumped in the root repo at `1747ba8`.

**Track B — HVPN-P0-077 + HVPN-P0-078 (measurement harness + decision
log + demo script):** `scripts/bench/unified_harness.sh` (603 lines) —
one harness producing a comparable CSV across G1/G2/G4 per §8's
pass-bar contract, reusing existing tools rather than reimplementing
probes; every honestly-unmeasurable metric labeled with a reason, never
fabricated. `scripts/demo.sh` (304 lines) — the 5-minute narrated
vertical-slice walkthrough. §12's G1 decision-log row backfilled with
real, independently-re-run evidence (was an empty placeholder). Real bug
found (out of scope, in `helix_core`): `g2_dpi_masque_unpriv.sh`'s
loss-resilience phase runs under `set -e` while the probe's exit code
communicates its measured verdict, silently truncating the script —
worked around without editing the out-of-scope file. Both items closed
`Completed`. Root commits `1a00311` (DB) + `9a7b3bb` (scripts/docs).

**Track D — audit + close verified-complete milestones:** independently
re-verified every constituent child of `HVPN-P0-S0`/`S2`/`S3`/`S4`
(re-ran cited tests, confirmed cited commits exist) before closing each
milestone — **and correctly refused to close `HVPN-P0-S1`**, having found
a real gap in `HVPN-P0-011` (below). Controller closed `HVPN-P0-S8`
separately once its last 2 children (`HVPN-P0-077`/`078`) landed from
Track B.

**Track E — independent adversarial evidence audit of all closed P0
items:** re-verified 20 items' evidence by actually re-running cited
tests (in isolated git worktrees for historical commits, cleaned up
after), re-executing live probes (`containers_check`, the G2 DPI rig
twice from scratch), and static analysis where told not to touch a
tree another track owned. 19/20 CONFIRMED or PLAUSIBLE. **Independently
rediscovered the `HVPN-P0-011` gap on its own** (force-ran the same
ignored tests, got the same `decapsulate error: InvalidPacket`) before
learning Track D had already caught and reopened it — a genuine
cross-validation, not a copy. Flagged one citation-precision nit
(`HVPN-P0-001`'s evidence cites a commit with 2 crates for a "6 crates"
claim true only of a later commit — not a functional defect, just a
citation to tighten next time it's touched).

**Two real regressions found and REOPENED this round (§11.4.34
Reopened-Details attached, not silently left closed):**
- **`HVPN-P0-049`** (`helix-ffi`) — no longer compiles. `helix_core`
  added 3 new `TunnelEvent` variants (`RouteAdded`/`RouteRemoved`/
  `TransportSwitched`) that `helix-ffi`'s match statement doesn't cover
  (`E0004` non-exhaustive match). Found by Track A, confirmed
  independently by the controller.
- **`HVPN-P0-011`** (`helix-wg` boringtun wrapper) — 2 tests were
  `#[ignore]`d for a vague, unverified "transport key alignment" reason;
  force-running them reproduced a real, reproducible `decapsulate error:
  InvalidPacket` failure. **At the time this was first written up (just
  above), that was read as "the WG data-plane encrypt/decrypt path is
  broken" — see the correction directly below; that reading was wrong.**
  Found independently by BOTH Track D and Track E, corroborated by an
  already-existing finding in
  `docs/reviews/mvp-final/findings/phase3-code-spec-alignment.md`.

**Flagged for operator attention — NOT yet actioned, needs a decision:**
Track E found a freshly-generated benchmark CSV (from Track B's harness
run) showing **Go beating Rust on G4 throughput** (800.7 vs 655.7 Mbps)
— the **opposite ranking** from the decision-log's cited numbers (Rust
1163.9 vs Go 884.5 Mbps). This is a full ranking flip, not just noise,
on the exact metric the still-open G4 edge-language decision hinges on.
The decision log's own "run-to-run variance on shared hardware is
non-trivial" caveat already anticipated volatility, but a flip this
large is worth resolving (more samples, or explicitly widening the
decision's confidence framing) before G4 is finally closed.

### Both regressions fixed this round, same session — including a self-correction

Two more parallel tracks were dispatched immediately after the finding
above, and both landed real, independently-verified fixes:

**`HVPN-P0-011` — CORRECTED root cause, then fixed.** The actual root
cause was **not** a crypto/transport bug — it was two malformed test
fixtures. `boringtun::noise::Tunn::decapsulate` runs every decrypted
payload through its own `validate_decapsulated_packet` sanity check,
which requires the payload to parse as a well-formed IPv4/IPv6 packet
whose header-declared total length equals the packet's actual byte
length (WireGuard's data plane only ever tunnels genuine IP packets;
this is boringtun's built-in defense against garbage/attack payloads).
`test_encrypt_decrypt_roundtrip`'s crafted packet declared an IP Total
Length of 46 and a UDP Length of 26 but was only actually 44 bytes long
— an internally-inconsistent fixture, correctly rejected.
`test_encrypt_decrypt_bidirectional` fed raw ASCII (`b"hello from A"`)
as a "plaintext IP packet," whose first byte's version-nibble made
boringtun attempt IPv6 parsing on a 12-byte message. **Neither failure
ever touched the Noise IK handshake or session-key cross-pairing** —
those were already proven correct by the never-ignored
`test_handshake_completes`. Fixed by correcting the fixtures to be
genuine, length-consistent IPv4/UDP packets (verified byte-for-byte
against boringtun's own vendored source); both tests now pass for the
right reason, `#[ignore]` removed. `helix-wg`: 36/36 passing (was 34/36
+ 2 ignored), full `helix_core` workspace green. `helix_core` commit
`b7e6b248`, pushed, confirmed matching `origin/main`. Independently
re-verified by the controller (re-read the diff's byte-length math,
re-ran the tests fresh). **Lesson: the earlier write-up in this same
file overstated the defect** ("the WG data-plane encrypt/decrypt path
is broken") before the actual root cause was known — corrected here
rather than silently left wrong, per this project's own no-guessing/
no-bluff discipline applying equally to its own documentation.

**`HVPN-P0-049` — fixed with real design justification, not a wildcard.**
Read `helix_core`'s G6 map-reconciliation tests first to confirm the
reconciler converges routes/transport **without restarting the tunnel**
(`TunnelState` stays `Connected` throughout). Decision: `RouteAdded`/
`RouteRemoved` → genuine no-op (pure route-table plumbing, no
corresponding slot in the Dart-facing 5-variant `TunnelStatus`).
`TransportSwitched` → re-projects `Connected` with the new transport,
honestly **reusing the last genuinely-measured RTT** rather than
fabricating a new one (no fresh handshake occurs on a transport switch,
so no new RTT exists to report — inventing one would itself be a
§11.4.6/§11.4.107 bluff). 6 new tests (4 unit + 2 integration) added
TDD-first. `helix-ffi`: 10/10 unit + 4/4 integration passing, full
crate builds clean. Confirmed the Dart-side `tunnel_status.dart` model
(built by the earlier Flutter track) still matches unchanged.
`helix_shims` commit `734ead0`, pushed, confirmed matching
`origin/main`. Independently re-verified by the controller (re-ran all
14 tests fresh).

Both closed `Fixed (→ Fixed.md)` in the tracker with real evidence.
`HVPN-P0-S1` milestone remains `Queued` — its own closure wasn't
re-attempted this round (P0-011 is now fixed, so S1 may be closable on
a future pass; not verified here to avoid re-opening this same file
mid-edit).

**DB state (after both fixes):** 485 items, `validate` PASS/0 issues,
`diff` PASS/0 divergences. 26 P0 items `Completed`, 3 `Fixed`
(`P0-084`, `P0-011`, `P0-049`), 1 `In progress` (`P0-052`), 9 `Queued`
(the genuinely device-blocked items + `S1`/`S5`/`S6`/`S7` milestones).

**Remaining queued P0 work:** `HVPN-P0-055`/`058` (Android, needs real
phone), `HVPN-P0-061`/`064` (iOS, needs real device), `HVPN-P0-067`
(memory soak, needs a real tunnel up) — all genuinely `Operator-blocked`
per §11.4.21, not silently skipped. The G4 throughput ranking-flip
above still needs an operator decision. `HVPN-P0-052` (Flutter toggle)
remains blocked on `libgtk+3-devel` (needs root) for a full build/run,
though its FFI dependency (`helix-ffi`) now compiles again after the
`P0-049` fix.

---

## ROUND 5: PHASE 0 SPIKE KICK-OFF — G2/G4/G5/G6 GATES CLOSED (2026-07-06)

**Started:** 2026-07-06T15:15:00Z
**Landed:** 2026-07-06T15:40:00Z (this entry)
**Trigger:** operator instruction to kick off real development against the Round-4
documentation package, then a follow-up demand for ≥3-4 parallel autonomous subagent
streams on parallelizable workable items.

**What happened:** dispatched 4 parallel subagents against the 13 remaining queued
Phase-0 (`P0`) tasks in `docs/workable_items.db`, picking the 4 that were genuinely
independent (different repos/crates, no shared file scope): two in isolated
`git worktree`s of `helix_core` (avoiding index/lock contention on the same repo per
§11.4.176/§11.4.179), one in `helix_shims` (previously empty scaffolding), one in the
root repo's `scripts/bench`+`scripts/spike` + `submodules/containers`. All 4 returned
real, independently-controller-reverified results (re-ran test suites / a bounded
bench sample myself, not just trusted the subagent reports) — no bluffed PASSes.

**Closed this round (all `Completed (→ Fixed.md)` in the workable-items DB):**
- `HVPN-P0-074` — **G6 pass.** Live `map.json` file-watch reconciliation wired into
  `helix-orch::Orchestrator` (poll-based, no new crate dep). Real test: peer
  unreachable → edit `map.json` → reachable within 5s, no restart, unrelated peer
  undisturbed. `helix_core` commit `c2e815e` (merged to main at `02c3636`).
- `HVPN-P0-049` — **G5 pass.** New `helix-ffi` crate in `helix_shims` (bootstrapped
  from empty scaffolding) implementing the flutter_rust_bridge v2 surface
  (`ClientConfig`/`start`/`stop`/`status_stream`/`TunnelStatus`), driving a **real**
  WireGuard handshake through `helix-orch`. TDD caught a real race (late stream
  subscription missing the `Connecting` event) and fixed it. `helix_shims` commit
  `4958072`, merged to its `main` and pushed. **Honest gap:** Dart/Flutter codegen
  not run — no `dart`/`flutter` toolchain in this environment; tracked as a
  follow-up, not silently skipped.
- `HVPN-P0-045` + `HVPN-P0-080` — **G4 measured, decision deferred.**
  `scripts/bench/edge_ab.sh` + `decision_matrix.sh` now drive both the Rust
  (`helix-edge`) and Go (`masqueedge`) MASQUE edges through an identical
  throughput/CPU/latency/churn protocol into a CSV; `scripts/spike.sh` S4 now proves
  genuine `containers` submodule runtime connectivity instead of just checking an
  executable bit. **Honest caveat surfaced by the subagent and preserved in the
  decision log:** Rust's edge currently uses a hand-rolled non-HTTP/3 CONNECT-UDP
  stand-in (per `helix-masque`'s own docs — the `h3` crate isn't yet viable for
  this), while Go's is the real RFC 9298 stack — so the numbers aren't yet an
  apples-to-apples MASQUE-conformance comparison. Root repo commits `f58d079` +
  `edc80b0` (the second fixes two compiled binaries — `helix_vpn_go_edge_bench`,
  `helix_vpn_containers_check` — that slipped past a `.gitignore` filename mismatch
  into the first commit; caught during controller review and fixed forward, not by
  rewriting history).
- `HVPN-P0-035` — **G2 core claim pass, 2 quantitative sub-bars fail (root-caused,
  not hidden).** Real nftables DROP on plain-WG UDP + ACCEPT on :443/udp (run
  unprivileged via `unshare --net --user`, since no root/sudo was available in this
  sandbox): the real boringtun handshake timed out while the real MASQUE/QUIC
  connection succeeded and moved traffic — core survival proven. Wire fingerprint:
  hand-rolled AF_PACKET capture (no tshark in sandbox) classified 16/16 `:443`
  packets as QUIC, 0 WireGuard signatures. **But** loss-resilience under
  `netem loss 5%` did NOT beat a UDP-over-TCP strawman — traced to RFC 9221 (QUIC
  DATAGRAM frames are congestion-controlled, so a fresh connection under immediate
  loss has no inherent raw-throughput edge; QUIC's real advantage here is avoiding
  head-of-line blocking, a latency property the goodput-only metric used didn't
  capture). `helix_core` commit `c257a7e` (merged to main at `02c3636`).

**Full decision log (G1–G6):** see
`docs/research/mvp/04_VPN_CLD/HelixVPN-Phase0-Spike.md` §12 — G2/G4/G5/G6 rows
filled this round with evidence citations; G1/G3 remain from earlier rounds (G1
marked done in the DB but its §12 row was never backfilled with evidence — a
pre-existing gap, not introduced this round; G3 needs a real iOS device, out of
autonomous reach).

**Verification performed by the controller (not just trusted subagent claims):**
independently re-ran `cargo test` for both `helix_core` merges (40/40 green) and
`helix_shims`'s `helix-ffi` (9/9 green), and ran a bounded (`--duration-secs 2`)
real invocation of `scripts/bench/edge_ab.sh` producing real, if differently-valued,
numbers — confirming the harness is genuinely reproducible, not a one-off.

**Remaining queued P0 tasks (untouched this round):** `HVPN-P0-052` (Flutter-Linux
toggle, G5 close — now unblocked since `helix-ffi` exists), `HVPN-P0-055`/`058`
(Android VpnService+JNI, real-device E2E — needs a physical phone),
`HVPN-P0-061`/`064` (iOS staticlib + NEPacketTunnelProvider — needs a real device
per §6.3, Simulator is explicitly non-representative), `HVPN-P0-067` (memory soak,
depends on a real tunnel being up), `HVPN-P0-077` (unified measurement harness —
partially subsumed by `edge_ab.sh`), `HVPN-P0-078` (decision log + demo script —
depends on all other gates closing first).

**Known pre-existing gap (not introduced this round, noted honestly):**
`workable-items diff` reports 43 MD↔DB divergences, all on `milestone`/`epic`-kind
items across all four WBS docs — the `db-to-md` generator does not appear to render
these kinds at all (only `task`/`subtask`). Not investigated further this round;
flagged for whoever next touches the docs_chain/generator tooling.

---

## ROUND 4: FINAL MVP DOCUMENTATION & PLATFORM READINESS — LANDED

**Started:** 2026-07-05T11:57:05Z  
**Landed:** 2026-07-06T13:40:00Z  
**Main repo commit:** `d8b9fc1d087bfffa2ba871685bf8ac89687a8740`  
**Handoff report:** `docs/reviews/mvp-final/signoffs/mvp-final-handoff-report.md`

**Goal achieved:** The consolidated MVP implementation source-of-truth is authored,
reviewed, gap-closed, committed, pushed, and verified. The package is ready for
development-team kick-off.

**Deliverables landed:**
1. `docs/research/mvp/final/implementation/` — 13 numbered sections + source-coverage
   ledger, all with HTML/PDF/DOCX siblings.
2. `docs/design/opendesign/helix/` — OpenDesign design system (manifest, tokens,
   components, exports); imports cleanly (exit 0).
3. `submodules/helix_proto/` — five `.proto` packages, generated Go stubs,
   `go.mod`/`go.sum`; `go build ./gen/...` and `buf lint` pass.
4. `submodules/challenges/helix_vpn/` + `submodules/helix_qa/banks/helix_vpn/` —
   8 Challenges + 8 HelixQA cases, bidirectionally traceable with the coverage ledger.
5. `docs/research/mvp/final/v04-client/connector.md` — single owning nano-detail doc
   consolidating FR-701..707.
6. `docs/reviews/mvp-final/review-rounds/round-{1,2}-{docs,design,code,qa}-findings.md` —
   two full rounds of independent adversarial review.
7. `docs/reviews/mvp-final/signoffs/gap-closure-summary.md` — GAP-1..GAP-6 status,
   owner, residual, and kick-off blocker flag.

**Round-2 adversarial verdicts:**
- Docs: **GO-with-conditions** → post-review conditions fixed.
- Design: **GO-with-conditions** → source-manifest consistency fixed; token-contract
  grade `needs-rebuild` explicitly accepted and documented.
- Code: **GO**.
- QA: **GO**.

**Key fixes closed in this round:**
- `helix_proto` `go_package` paths aligned with generated directory layout so Go stubs
  are importable.
- GAP-1 connector local-ACL × central-policy precedence rule backported from the
  consolidation READMEs into `v03-control-plane/svc-policy.md` and
  `v04-client/helix-core-rust.md`; `requirements-traceability.md` and
  `functional-requirements.md` updated to CLOSED/VERIFIED.
- QA coverage ledger made bidirectionally traceable to the actual bank contents; 8
   Challenge IDs and 8 HelixQA IDs minted; DDoS entries added for NFR-413/NFR-414.
- OpenDesign source manifest stripped of import-generated path declarations so the
   source package is reproducible.
- GAP-3 closed at doc level: `v06-deploy/disaster-recovery.md` RTO/RPO targets and
  runbooks authored; measurement pending Phase-2 CHAOS region-failover drill.
- GAP-4 closed: `v04-client/connector.md` created as the single owning doc for
  FR-701..707; traceability matrix repointed.
- GAP-5 closed at doc level: all 15 `v08-testing/` nano-detail docs authored and
  mapped; evidence states honestly PENDING until build.
- GAP-6 closed: DDOS/RBAC/rate-limiting owners pinned; quantitative DDoS targets
  defined in `v08-testing/ddos.md` §10.

**Verification:**
- `tests/pre_build_verification.sh` — PASS.
- Mermaid validation — 361/361 blocks rendered ok across `docs/research/mvp/final/`.
- `go build ./gen/...` and `buf lint` in `submodules/helix_proto` — PASS.
- All submodule commits verified by `git rev-parse HEAD` equality against every
  configured remote.
- Main repo push verified by `git rev-parse HEAD` equality against `github`,
  `origin`, and `upstream`.

**Honest residual measurement gaps (not kick-off blockers):** NFR-205 RTO/RPO
numbers pending Phase-2 drill; NFR-413/NFR-414 DDoS `ATTACK_PPS` and legit-handshake
SLO pending Phase-2 benchmarks; OpenDesign token-contract grade `needs-rebuild`
accepted. All are tracked in the coverage ledger, review reports, and handoff
report.

---

## ROUND 3.1: FULLY LANDED (2026-07-05T11:29:45Z)

Follow-up clean-up round for deferred items discovered during the Round 3
second decoupling audit. Committed and pushed; verified via direct
`git rev-parse` equality against every remote for the main repo AND every
touched submodule:

- `docs_chain` — `99ad270` — fixed broken `[Constitution.md](Constitution.md)`
  self-links in `CONSTITUTION.md` / `AGENTS.md` / `CLAUDE.md` / `QWEN.md`;
  renamed `Upstreams/` → `upstreams/` and `GitHub.sh`/`GitLab.sh` →
  `github.sh`/`gitlab.sh`; updated `install_upstreams.sh`. Pushed to
  `origin`/`github`/`gitlab`/`upstream` on `main`.
- `llms_verifier` — `0e7d6949` — fixed broken self-links in `QWEN.md`;
  lowercase upstream scripts. Pushed to `origin`/`github`/`gitlab`/`upstream`
  on `main`.
- `panoptic` — `c6b6c49` — fixed broken self-links in `CRUSH.md`; lowercase
  upstream scripts. Pushed to `origin`/`github`/`upstream` on `main`.
- `challenges` — `2711bf0` — added containers-style package metadata table to
  `README.md`; lowercase upstream scripts. Pushed to `origin`/`github`/
  `gitlab`/`upstream`/`vasicdigitalgithub` on `main`.
- `security` — `318c8c7` — added containers-style package metadata table to
  `README.md`; lowercase upstream scripts. Pushed to `origin`/`github`/
  `gitlab`/`upstream` on `main`.
- `containers` — `df980b3` — lowercase upstream scripts. Pushed to
  `origin`/`github`/`gitlab`/`upstream`/`vasicdigitalgitlab` on `main`.
- `doc_processor` — `4e98523` — lowercase upstream scripts. Pushed to
  `origin`/`github`/`gitlab`/`upstream`/`vasicdigitalgithub`/
  `vasicdigitalgitlab` on `master`.
- `helix_qa` — `04e12e4` — lowercase upstream scripts. Pushed to
  `origin`/`github`/`gitlab`/`upstream`/`vasicdigitalgithub` on `main`.
- `llm_orchestrator` — `4aa7219` — lowercase upstream scripts. Pushed to
  `origin`/`github`/`upstream` on `master`.
- `llm_provider` — `084d56f` — lowercase upstream scripts. Pushed to
  `origin`/`github`/`gitlab`/`upstream`/`vasicdigitalgithub` on `master`.
- `vision_engine` — `9553a31` — lowercase upstream scripts; rewrote stale
  `push-all.sh` to iterate all configured remotes and removed four broken
  single-remote push scripts. Pushed to `origin`/`github`/`upstream` on
  `master`.
- `constitution` — `eae531a` — lowercase upstream scripts. Merged gitflic/main
  updates before push (no force-push, per §11.4.113). Pushed to `gitflic`/
  `github`/`gitlab`/`gitverse`/`origin`/`upstream`/`vasicdigitalgithub`/
  `vasicdigitalgitlab` on `main`.
- Main repo — `ea9677a` — renamed `upstreams/GitHub.sh` → `upstreams/github.sh`;
  bumped all touched submodule pointers.

**Remaining queued (not yet dispatched):** `helix_qa`'s nested third-party
`tools/opensource/docling` / `tools/opensource/skyvern` working-tree drift
(remains untouched; third-party vendored code, exempt per §11.4.28).

**Next action:** none in flight. Await new operator instructions.

---

## ROUND 3: FULLY LANDED (2026-07-05T07:00:00Z, re-verified this pass)

All Round 3 work is committed and pushed, re-confirmed via direct
`git rev-parse` equality against every remote for the main repo AND
every touched submodule (never trusted a push-log message or a prior
handoff's text alone, per §11.4.88/§11.4.6):

- `helix_core` — `992e1be` (engineering batch), on `origin/main`.
  Re-verified fresh this pass: `cargo test --workspace` → 120 tests,
  0 failed.
- `helix_edge` — `08d6e18` (first real crate), on `origin/main`.
  Re-verified fresh: `cargo test --all-targets` → 11 tests, 0 failed.
- `helix_go` — `57d4972` (first real Go module + a `.gitignore` fix —
  a stray `/pkg/` ignore rule was silently excluding `pkg/masqueedge`
  from version control; caught before push), on `origin/main`.
  Re-verified fresh: `go build ./...` clean, `go test ./...` → ok.
- `llm_orchestrator` — `ef73c3a` (two-pass CONSTITUTION.md rewrite — a
  review caught the first pass as incomplete), pushed to `master` on
  all 3 remotes (this repo's parent-tracked lineage is `master`, not
  `main`).
- `vision_engine` — `2f22942`, reviewed GO, pushed to `master`.
- `llms_verifier` — `17b4bfb6` (HelixCode + leftover-Lava fix at
  `9281cae2`, **fast-forwarded** past 3 unrelated upstream commits from
  a separate workstream — a semantic-code-visibility exit-code fix +
  a CONST-069 mandate + a reconciliation merge — found and merged
  during this pass, no force-push, no conflict), on `origin/main`.
- `panoptic` — `31aaceb` (cascaded CONST-048/050/051/052/056
  boilerplate fixed on a second pass), on `origin/main`.
- `containers` — `a432efa` (real `os.UserHomeDir()` fix + a synthetic-
  `$HOME` regression test + 33-package doc-table correction),
  re-verified.
- `helix_qa` — `c1c2513` (routine nested opensource-tool submodule
  pointer advancement, confirmed ordinary upstream drift). Its nested
  third-party tools (`tools/opensource/docling`, `.../skyvern`) show
  their OWN working-tree drift (a modified test-data file, a modified
  nested `integrations/n8n` pointer) — deliberately left untouched
  (third-party vendored code, exempt from equal-engineering per
  §11.4.28; origin/intent of the drift was not investigated).
- Main repo — `26b4b2a` (bumps `llms_verifier`'s pointer to the
  fast-forwarded tip above; supersedes the earlier `4d338cb`/`e96410b`/
  `f1de366` pointer-bump sequence). Confirmed identical across
  `origin`/`github`/`upstream` via `git rev-parse`.

Every commit above passed an independent adversarial code-review pass
(§11.4.125/§11.4.134) before being accepted — three rounds initially
returned NO-GO or hit a mid-review session-limit error (llm_orchestrator's
CONSTITUTION.md; the first llms_verifier+panoptic HelixCode pass; the
helix_go review) — each was confirmed genuinely completed via git state
before being trusted, per the no-guessing mandate, not re-done blindly.

**Multi-session lesson (new this pass, keep this note until it's no
longer novel):** mid-session, this exact working directory was found to
have been advanced by a PARALLEL Claude Code session operating the same
checkout concurrently — commits landed and `.remember/remember.md` was
overwritten with a handoff describing work outside the then-current
conversation's own context. The correct response was to treat git state
as ground truth and re-verify everything (`git log`, `git rev-parse`
against every remote, re-run test suites fresh) rather than trust the
handoff text or the in-context conversation memory of "pending work."
This generalizes §11.4.37's fetch-before-edit doctrine to a stronger
claim: verify against the filesystem even when your OWN memory feels
current — a parallel session can invalidate it without warning.

`docs/workable_items.db`: HVPN-P0-018/028/031/039/042/071 closed
`Completed (→ Fixed.md)` with real evidence citations. `validate`: PASS,
484 items, 0 issues (re-confirmed this pass).

**Remaining queued (not yet dispatched, carried forward to a later
round):** `helix_qa`'s nested third-party tool drift noted above. The
other deferred items (broken `Constitution.md` self-links in `docs_chain`,
stale package tables in `challenges`/`security`, and PascalCase upstream
scripts) were dispatched and completed in **Round 3.1** above.

**Next action**: none in flight. Await new operator instructions
(original mandate: notify when fully committed/pushed so the user can
send mvp4 instructions — done), or action the queued follow-up round
above.

---

## ROUND 3 — Phase-0 Rust/Go engineering + fleet decoupling audit (historical log, landed above)

Round 2 (MVP gap-analysis) + Round 2.1 (decoupling audit) are complete
and pushed — see "ROUND 2 + 2.1: FULLY LANDED" below. Round 3 landed
real Phase-0 engineering across two submodules plus a second, deeper
fleet-wide decoupling audit that surfaced serious findings now being
remediated in parallel.

### Landed and independently reviewed GO (uncommitted, staged for one batch)

All in `submodules/helix_core` unless noted:

- **HVPN-P0-018/020/021** (orchestrator client/connector binaries) —
  real WireGuard Noise IK handshake over loopback via `helix-wg`/
  boringtun, driven by new `crates/helix-core/src/bin/{helix-client,
  helix-connector}.rs` + `crates/helix-orch/src/wg_session.rs`. First
  review returned **NO-GO** (2 findings: a false "nft/iptables not
  installed" claim — both ARE installed, `nft` runs unprivileged
  inside an isolated netns; a private key accepted via `--private-key`
  CLI argv, leaking via `ps`/`/proc/<pid>/cmdline`). Both fixed
  (env-var-only key resolution via new `cli::read_private_key_from_env`;
  corrected doc comment/runtime message to the real constraint — root-
  owned namespace access to the host's actual LAN NIC, not missing
  tooling). Re-review: **GO**.
- **HVPN-P0-028/029/030** (quinn+h3 QUIC connection) —
  `crates/helix-masque/src/quic.rs` taken from stub to real: genuine
  `quinn::Endpoint` client+server, real hostname/SAN cert verification
  (no skip-verification shortcut), RFC 9221 datagram round-trip.
  Review: **GO**.
- **HVPN-P0-031/032/033** (MASQUE CONNECT-UDP + HTTP-Datagram framing)
  — new `crates/helix-masque/src/{datagram,connect}.rs`. Deep-researched
  `h3`'s real RFC 9298 support (found genuinely immature — open bugs in
  its own datagram/quarter-stream-ID handling) and honestly built a
  labeled simplified stand-in instead of claiming false RFC compliance.
  Not yet independently reviewed as a standalone item (folded into the
  same batch as P0-018/028, which were).
- **HVPN-P0-071/072/073** (map.json schema + reconciler) — new
  `crates/helix-core/src/map.rs`. Pure diff engine, idempotent, panic-
  free on adversarial malformed/duplicate-peer inputs. Review: **GO**
  (two non-blocking follow-ups noted for Phase-1: canonicalize
  `allowed_ips` ordering before live wiring; add an explicit CONC/RACE
  test per the WBS's own declared test-types).
- **HVPN-P0-042/043/044** (Go edge, `quic-go`+`masque-go`) — bootstrapped
  `submodules/helix_go`'s first real Go module (`pkg/masqueedge` +
  `cmd/go-edge`). `masque-go` proved genuinely turnkey (real CONNECT-UDP
  server+client wired from its own test-suite pattern, no hand-rolled
  framing needed) — concrete evidence for the Go-vs-Rust edge-language
  decision, in Go's favor for this specific protocol layer. Not yet
  independently reviewed.
- **HVPN-P0-039/040/041** (Rust edge `helix-edge`) — bootstrapped
  `submodules/helix_edge`'s first real Cargo binary, path-depending on
  the sibling `helix_core` crates. A **complete real WireGuard
  handshake traverses MASQUE-client → edge-relay →
  `helix_orch::wg_session` responder**. A genuine bug was found and
  fixed during TDD: `send_datagram()` only enqueues (doesn't flush) —
  an early test closed the connection immediately after, racing
  quinn's async flush and dropping the handshake's 3rd message; fixed
  by removing the premature close, stable across 5 reruns. A decoy
  HTML responder coexists with the real MASQUE flow on the same port
  number (TCP vs UDP/QUIC — independent kernel namespaces). Review
  in progress.
- Fixed a real, pre-existing (predates this session, from commit
  `405db88`) decoupling violation the earlier audit missed:
  `helix-masque`'s `MasqueConfig`/`QuicConfig` default values hardcoded
  `proxy.helixvpn.io` — directly contradicting the submodule's own
  README/CLAUDE.md claim of "no HelixVPN-specific hostnames." Fixed to
  the generic `proxy.example` (RFC 2606) across all 6 occurrences;
  27/27 `helix-masque` tests still pass.
- Fixed a bare-pinned `hex = "0.4"` dependency (used identically by two
  crates, `helix-wg` and `helix-core`) into `[workspace.dependencies]`
  per this project's own written convention ("never a separately-
  pinned version") — flagged by the map.rs reviewer as a minor
  drive-by finding from the P0-018 batch.

**Honest environment constraint** (re-confirmed multiple times this
round via direct probes, never assumed): no passwordless sudo
(`sudo -n true` fails) → real kernel WireGuard, real network
namespaces reaching the host's actual LAN interface, and binding
privileged port 443 are NOT autonomously achievable here. Every item
above is honestly scoped to what's provable via loopback + unprivileged
high ports, exactly as this project's own anti-bluff discipline
requires — no faked privilege, no silently-skipped acceptance criteria.

### Second fleet-wide decoupling audit — severe findings, remediation in progress

A deeper audit (beyond Round 2.1's pass) found:

1. **Wrong-project contamination (worse than the earlier "Lava §6.AD"
   dangling-sentence bug)**: `llm_orchestrator`, `vision_engine`,
   `llms_verifier`, `panoptic` have their ENTIRE `CLAUDE.md`/`AGENTS.md`
   /`CONSTITUTION.md` bodies describing a different, unrelated project
   called "HelixCode" (22-62 grep hits each) — nonexistent directories,
   wrong package structures, wrong Makefile targets, wrong module
   names. Each submodule's own `README.md` is correct and was used as
   ground truth for the fix. Remediation dispatched (2 parallel
   subagents, one per submodule pair).
2. **`llms_verifier` still carries leftover "Lava"-project content**
   beyond what the earlier session-wide fix removed — a full section
   ("§6.X — Container-Submodule Emulator Wiring Mandate") explicitly
   referencing "the parent Lava repo." Being removed/genericized as
   part of the same remediation pass.
3. **`containers/pkg/remote/compose_detector.go:76` hardcodes this
   operator's home directory in PRODUCTION CODE** (not just docs) as
   a `podman-compose` lookup candidate — silently never matches on any
   other machine. Fix dispatched: real `os.UserHomeDir()` resolution +
   a test that would have caught the original bug.
4. Broken `[Constitution.md](Constitution.md)` self-links (should be
   `CONSTITUTION.md`, uppercase) in `docs_chain`/`llms_verifier`/
   `panoptic`; stale package tables in `challenges`/`containers`/
   `security` (missing 3-16 real packages each); PascalCase
   `GitHub.sh`/`GitLab.sh` upstream scripts in 11 "borrowed" submodules
   (violates §11.4.29 lowercase-snake_case) — queued for a follow-up
   round, not yet dispatched.

### Next actions — all of Round 3's own steps are DONE; see "Remaining
queued" above for the one carried-forward follow-up round.

Milestones S4-S8's remaining subtasks (bench.sh A/B harness, decoy/
DPI-survival gates, FFI/mobile work) remain `Queued` — most are gated
behind the edge-implementation work above, which has now landed and
reviewed cleanly, so these are unblocked for a future round.

---

## ROUND 2 + 2.1: FULLY LANDED (2026-07-05T00:14:23Z, verified)

Both main-repo commits are confirmed pushed to all 3 remotes — local
HEAD, `github/main`, `origin/main`, `upstream/main` all equal
`20dc9a4` (verified via `git rev-parse`, not assumed). Round 2's first
push attempt (`e46a710`) stalled/died across a session/process
restart mid-transfer (~84MB of new binary content) and had to be
retried from scratch — the retry completed in seconds, confirming the
first attempt was genuinely stuck, not just slow. **Lesson for future
large pushes in this environment: verify completion by comparing
`git rev-parse HEAD` against every remote's tip, never trust a
"push started in background" message alone — nohup+disown does not
reliably survive a session/process restart in this environment.**

All 19 owned submodules verified in sync with their own remotes
(15 confirmed byte-identical local==origin; the 4 known-diverged ones
—doc_processor/llm_orchestrator/llm_provider/vision_engine— correctly
left untouched at the main-repo-pointer level, with llm_provider's
actual fix independently confirmed live on its real remote).

**Round 2 + 2.1 are now genuinely, verifiably complete — commits
pushed, submodule pointers current, nothing pending.**

## ROUND 2.1 — post-completion cleanup (2026-07-04T18:10-19:30, DONE, LANDED)

After round 2 was declared complete, two more real, operator-directed
fixes landed across the submodule fleet (main repo's own commit for
this is NOT yet made — main repo's huge round-2 push is still
uploading; the pointer bumps for these submodules will land in the
NEXT main-repo commit):

1. **"Lava §6.AD" dangling cross-project reference** — removed from
   19 submodules' `CLAUDE.md`/`AGENTS.md`/`CONSTITUTION.md` (a stray
   sentence claiming this project's root CLAUDE.md has a "§6.AD" about
   incorporating an unrelated project called "Lava" — it does not).
   **Important sub-finding**: 4 submodules (`doc_processor`,
   `llm_orchestrator`, `llm_provider`, `vision_engine`) have a LOCAL
   checkout on an independently-diverged `master`-lineage branch vs.
   their `origin/main` — confirmed via `merge-base --is-ancestor`
   (NOT corruption; operator confirmed this is expected/known). Fixed
   directly on `origin/main` (the branch that matters) via a clean
   detached worktree for the one repo that needed it there
   (`llm_provider` — the other 3's `main` was already clean of this
   specific issue); did NOT touch/merge/push the diverged `master`
   lineage for any of the 4. `llm_provider`'s `vasic-digital` mirror
   (github+gitlab) additionally rejected the push as non-fast-forward
   (yet another independent divergence) — left untouched, same
   "expected divergence" class per operator confirmation.
2. **Decoupling violation — hardcoded "Helix VPN" project name** —
   found (operator mandate: "Keep all Submodules fully decoupled! No
   Submodule can be parent project aware!!!") in the 8 freshly-created
   submodules' governance files AND in `helix_core`'s REAL source
   metadata (`Cargo.toml` `description` fields, `lib.rs` doc comments,
   `README.md` — pre-existing from an earlier session, not this
   round's governance work). Fixed all of it generically (e.g. "a
   consuming VPN/networking product"); verified `cargo check
   --workspace` still compiles clean (0 new warnings) after the
   text-only changes. Comprehensive fleet-wide `git grep` sweep across
   all 19 submodules' TRACKED files confirms zero remaining
   "helix vpn"/"helix_vpn" references anywhere.

All 9 affected submodules (`docs_chain` + the 8 fresh ones, plus
`llm_provider`'s main separately) committed + pushed to their primary
remotes. **Main repo's submodule-pointer bump for all of this is
still pending** — do not forget it in the next main-repo commit.

## ROUND 2 STATUS: COMPLETE (2026-07-04T18:10:00Z)

MVP gap-analysis/hardening round 2 (see full detail retained below) is
**DONE**: all 3 MVP corpora hardened + unified, independent review gate
passed (after a real fix cycle — 8 critical findings fixed, not
rubber-stamped), governance propagated to all 19 owned submodules (9
newly fixed, all committed+pushed to their own remotes), workable-items
DB reconciled with real `cargo test` evidence, all exports regenerated
(0 failures after fixing 4 real mermaid defects + 1 Puppeteer/Chrome
launch bug), main repo committed
(`feat: MVP gap-analysis + enterprise hardening round...`), push to all
main-repo upstreams running detached per §11.4.88 (check
`qa-results/push_failures/` for any failure log — absence = success).
**Task #6 (anti-bluff constitution propagation) is also now COMPLETE**:
root project was already compliant; all 19 owned submodules now carry
proper CLAUDE.md/AGENTS.md/QWEN.md/CONSTITUTION.md with the inheritance
pattern; the project's real Rust tests were confirmed genuine (not
mocked) during the DB-reconciliation work. **Nothing from this round's
scope remains open** except the explicitly-flagged, intentionally
out-of-scope items in §5 below (operator decisions) and the low-priority
"Lava §6.AD" fleet-wide cleanup item (tracked, not urgent).

**If resuming fresh: verify the push actually succeeded**
(`git log --oneline HEAD..@{u}` for each remote should be empty, or
check `qa-results/push_failures/` for a failure log) before assuming
this round is 100% externally visible — the commit itself is durable
either way.

---

## CURRENT ROUND (2026-07-04, round 2) — MVP gap-analysis, hardening, unification

**Operator mandate (verbatim intent):** analyze all three MVP corpora
(`docs/research/mvp/`, `docs/research/mvp2/`, `docs/research/mvp3/` +
`docs/research/mvp_final/`) for gaps / inconsistencies / shortcomings /
unfinished parts; close every gap with rock-solid, enterprise-grade,
scalable content; extend all docs/guides/plans/diagrams/OpenDesign
cross-references to make all MVP phases impeccable; commit + push
everything (main + all submodules, all upstreams); keep
CONTINUATION.md + exports in sync throughout; use subagent-driven
parallel execution; notify when fully committed/pushed so the
operator can send mvp4 instructions. **Follow-up mandate queued
right behind this one** (do NOT drop it — see "Queued follow-up"
below): propagate the anti-bluff testing/Challenges mandate into
this project's `Constitution.md`/`CLAUDE.md`/`AGENTS.md`/`QWEN.md`
and every owned submodule's equivalents, respecting the
HelixConstitution submodule inheritance rules (§11.4.35/§11.4.26).

**Status: 4 parallel subagents dispatched, IN PROGRESS (not yet returned as of this write):**
1. **Agent A** — deep gap analysis + direct hardening of `docs/research/mvp/` (Phase 0/1 control-plane corpus — note the actual numbered-volume spec lives under `docs/research/mvp/final/*.md`, plus `docs/research/mvp/04_VPN_CLD/*.md`; agent was told to `find docs/research/mvp -name '*.md'` first so it discovers the real paths itself).
2. **Agent B** — deep gap analysis + direct hardening of `docs/research/mvp2/` (Phase 2, 8-platform client-app corpus).
3. **Agent C** — defines Phase 3 (`docs/research/mvp3/MVP3_ENTERPRISE_SCALE.md`, new) and the GA/Final phase (`docs/research/mvp_final/MVP_FINAL_GA_READINESS.md`, new) from scratch (both were previously empty `TBD.md` placeholders), plus a new unifying `docs/research/UNIFIED_PHASE_ROADMAP.md` reconciling all phases into one index.
4. **Agent D** — cross-cutting, read-only-on-others audit: `docs/research/CROSS_CUTTING_GAP_ANALYSIS.md` (new) covering OpenDesign coverage across mvp2's 8 platforms, testing-philosophy consistency between mvp/ and mvp2/, and a diagram-completeness sweep across all corpora — produces recommendations only, does not edit mvp/mvp2/mvp3/mvp_final/design directly (disjoint file-scope from Agents A/B/C per §11.4.58/§11.4.20).

**Next actions once all 4 agents return (in priority order):**
1. Review each agent's final report; spot-check the diffs for quality/consistency (§11.4.92 multi-pass — at minimum Pass 1 main-task + Pass 2 blast-radius).
2. Action Agent D's recommendations that are cheap/obvious (e.g. missing OpenDesign cross-refs in mvp2 UI/UX spec) — either inline now or track as a fast-follow.
3. Re-sync exports: run whatever markdown→HTML/PDF/DOCX export pipeline this project uses for `docs/research/**` (check `scripts/testing/sync_all_markdown_exports.sh` — confirm it still exists and covers the new files before invoking) — per §11.4.65/§11.4.12, exports must stay in sync with sources.
4. If `docs/workable_items.db` / Issues.md tracking is meant to cover this round's work, reconcile per §11.4.93/§11.4.148 (check whether this project's tracker actually requires per-doc workable items for a docs-only round, or whether that's overkill here — do not force-fit a heavyweight tracker entry for a pure documentation-authoring pass unless the project's own convention already does so).
5. Commit + push: main repo via `scripts/commit_all.sh` (NOT raw `git commit`), and any owned submodule that was touched (none expected this round — all 4 agents were scoped to `docs/research/**`, which lives in the main repo, not a submodule) — push to all configured upstreams.
6. Report completion to the operator with a summary of every file created/modified.
7. **Then** address the queued follow-up mandate below (anti-bluff Constitution/CLAUDE.md/AGENTS.md/QWEN.md propagation) — do not let it drop; it was explicitly deferred, not abandoned.

### Workable-items DB reconciliation (DONE this round — real evidence, not a guess)

The DB (`docs/workable_items.db`, 484 items) was found 100% `Queued` —
stale against reality: `submodules/helix_core` is actually AHEAD of
what this file previously described. Verified with real commands
(not trusted from old notes): `cargo test --workspace` inside
`submodules/helix_core` (HEAD `405db88` — "WireGuard boringtun +
orchestrator + MASQUE stub + G1 tests"), crate source inspection for
stub/TODO markers, and workspace `Cargo.toml` inspection. **Actual
current state: 6 crates** (`helix-core`, `helix-masque`, `helix-orch`,
`helix-transport`, `helix-tun`, `helix-wg` — not 4), **72 unit/integration
tests passing** (helix_masque 12, helix_orch 13, helix_transport 12,
`tests/g1_integration.rs` 3, helix_tun 5, helix_wg 27; plus doctests) —
not the "39 tests" this file previously claimed. Using the canonical
`cmd/workable-items` Go CLI (`close --id ... --status ... --evidence
...`, never raw SQL), closed with cited evidence:
- **Completed (→ Fixed.md):** HVPN-P0-001 (workspace bootstrap),
  HVPN-P0-004 (Transport trait, 12 tests), HVPN-P0-008 (UDP transport,
  proven via G1 echo test), HVPN-P0-011 (real `boringtun=0.7.1` wrapper
  — genuine dependency + `Tunn`-based device/handshake/noise/timers,
  zero stub markers, 27 tests), HVPN-P0-015 (TUN device, 5+1 tests),
  HVPN-P0-022 (test rig, referenced directly by the G1 integration
  test's doc comment), HVPN-P0-025 (G1 gate — `g1_udp_loopback_echo`
  really round-trips 10 UDP datagrams with RTT assertions, 3/3 passing).
- **In progress (honest, NOT overclaimed):** HVPN-P0-018 (orchestrator
  three-loop core is real and tested — 393+226 lines, 13 tests — but
  zero `[[bin]]` targets exist yet, so the task's own "client/connector
  binaries" deliverable is unmet), HVPN-P0-028 + HVPN-P0-031 (the
  `helix-masque` crate's OWN doc comment says "This is a **research /
  stub pass**" — honored that self-assessment rather than closing).
- DB re-`validate`d clean (484/484, 0 issues) after every update;
  WAL-checkpointed per §11.4.95. This DB change + the corrected
  crate/test counts above still need to be committed alongside the
  rest of this round's work (see "Next actions" below — nothing has
  been committed yet as of this write).

### Independent-review gate + fix cycle (2026-07-04 ~17:00-18:00, DONE)

Per §11.4.125/§11.4.134 (code-review-before-build, iterate-until-clean-GO),
dispatched 3 independent (structurally-separated, not self-review) adversarial
review agents against everything this round produced, before considering it
committable:
1. Review of the 5 new top-level docs (`mvp3/MVP3_ENTERPRISE_SCALE.md`,
   `mvp_final/MVP_FINAL_GA_READINESS.md`, `UNIFIED_PHASE_ROADMAP.md`,
   `GOVERNANCE_INHERITANCE_AUDIT.md`, `CROSS_CUTTING_GAP_ANALYSIS.md`) — found
   3 wrong section-number citations in `MVP3_ENTERPRISE_SCALE.md` (§7.2 cited
   instead of §4 twice, §2.1 cited instead of §1.2 once) + confirmed both
   audit docs' headline findings had already been remediated by parallel work
   (accurate when written, stale by the time of review).
2. Review of the mvp/+mvp2/ hardening diffs — found 2 real defects in mvp/
   (a broken relative link the pass itself introduced in
   `v01-product/functional-requirements.md`; a stale "still needs fixing" note
   in `08-phase2-parity-wbs.md` describing a fix that had ALREADY landed in
   the same diff) and 3 in mvp2/ (a pre-existing broken table row in
   `MVP2_OVERVIEW.md` §9.2 left unfixed despite heavy editing; an unescaped
   pipe in `MVP2_SECURITY_PERFORMANCE.md` breaking a table; the self-flagged
   Phase-1-"COMPLETED" contradiction in `MVP2_OVERVIEW.md` §3.1 that
   `MVP2_WEB_CLIENT.md` had named but never actually annotated in the source
   file itself) + 1 minor (residual semantic-color drift in
   `MVP2_MOBILE_APPS.md` beyond just the primary/accent brand color).
3. Review of the 9-submodule governance propagation — **clean pass, zero
   critical findings** (already committed/pushed, see below).

**All Critical + the one worthwhile Minor finding were fixed directly
(myself, not delegated — small, well-specified, mechanical changes) and each
fix was empirically verified**, not just asserted: the broken link now
resolves (`test -f` confirmed); the stale note now says "already fixed" with
the real evidence; the `MVP2_OVERVIEW.md` table row now has the correct cell
count (verified via `awk -F'|'`); the pipe is escaped; the Phase-1
contradiction now carries an explicit flag note (matching the precedent
`UNIFIED_PHASE_ROADMAP.md` R-2 already set — flag, don't silently rewrite);
the 3 mvp3 citations point at the confirmed-correct sections; the two
now-stale audit docs (`GOVERNANCE_INHERITANCE_AUDIT.md`,
`CROSS_CUTTING_GAP_ANALYSIS.md`) each got a "STATUS UPDATE" callout so a
future reader doesn't re-do already-closed work; the semantic colors in
`MVP2_MOBILE_APPS.md` now cite `color.json`'s real hex values.

**Bonus, found independently of the 3 review agents, during my own export-sync
QA pass:** 3 MORE genuine mermaid syntax defects the review agents' text-only
reading couldn't have caught (they don't render diagrams) — all
root-caused via direct `mmdc` reproduction + bisection, never guessed:
(a) `07-phase1-mvp-wbs.md` — a literal `;` inside a sequence-diagram message
breaks mermaid's parser regardless of arrow syntax; (b) `MVP2_WEB_CLIENT.md`
— an unquoted flowchart decision-node label containing `(...)` needs
quoting; (c) `MVP2_SHARED_CORE.md` — mermaid's `stateDiagram-v2` transition-
label parser cannot handle a literal `::` (confirmed via a 6-way bisection
test matrix, not assumed); (d) `MVP3_ENTERPRISE_SCALE.md` — the
`A -. label .-> B` dotted-arrow-with-inline-label form isn't valid mermaid;
the pipe-delimited `A -.->|"label"| B` form is. All 4 verified via actual
successful PNG render before moving on, all 11 affected files' exports
re-synced with `--force` (0 failures).

**This is now genuinely done** — not just self-reported by the authoring
agents, but independently adversarially reviewed AND the findings fixed AND
re-verified. Ready to commit.

### Session-limit crash + recovery (2026-07-04 ~16:16 MSK)

Agents A (mvp/) and B (mvp2/, plus B's own 3 sub-dispatched helpers for
`MVP2_WEB_CLIENT.md`/`MVP2_MOBILE_APPS.md`/`MVP2_IMPLEMENTATION_ROADMAP.md`)
all failed simultaneously with "session limit resets 4:10pm
(Europe/Moscow)" — a hard external wall, not a design/logic bug. NO
WORK WAS LOST: their edits already landed on disk before the crash
(108 files / +2430 lines in `mvp/`, all 10 `MVP2_*.md` + siblings /
+4791 lines in `mvp2/` — verified via `git diff --stat`, not assumed).
Confirmed the reset had already passed by the time of investigation and
dispatched 2 continuation agents (NOT a repeat of the original giant
prompts — targeted at (a) the SPECIFIC dangling thread each crashed
agent was mid-sentence on when cut off, cited verbatim from their last
output, and (b) reconciling 3 unactioned findings from
`CROSS_CUTTING_GAP_ANALYSIS.md`: brand-color inconsistency across
mvp2/docs/design, missing anti-bluff testing philosophy in mvp2/, thin
diagram coverage in mvp2/). Both launched without error — confirms the
limit has cleared. **If a fresh session picks this up and finds these
still running or newly crashed again, re-dispatch using the SAME
targeted-continuation pattern (not the original full prompts) — check
`git diff --stat` first to see exactly what's already done.**

### Governance-file propagation (DONE + already committed/pushed — the ONE piece of this round that IS already on remotes)

The 8 zero-governance submodules (`helix_core`, `helix_design`,
`helix_edge`, `helix_go`, `helix_proto`, `helix_shims`,
`helix_transport`, `helix_ui`) each got `CLAUDE.md`/`AGENTS.md`/
`QWEN.md`/`CONSTITUTION.md`, replicating the exact pattern already used
by compliant siblings (`security`, `challenges`) — verified against
those two + spot-checked against `docs_chain`/`vision_engine`/
`llm_orchestrator` before writing anything. `panoptic` got its missing
`QWEN.md` + an inheritance pointer added to its existing
`CONSTITUTION.md`. **All 9 submodules were committed + pushed to their
own remotes already** (each is an independent git repo with its own
`origin`; `panoptic` additionally has `github`/`upstream` remotes, all
three identical URLs, pushed to all three). Commit hashes: helix_core
`f245b98`, helix_design `7fbe145`, helix_edge `3c7771a`, helix_go
`1eccd10`, helix_proto `24b41a0`, helix_shims `e7c6b43`, helix_transport
`cdce305`, helix_ui `bd2a495`, panoptic `8212dbe`.

**A flagged-but-deliberately-NOT-fixed finding from that pass:** a stray
"Lava §6.AD" sentence is duplicated verbatim across ALL previously-
compliant submodules' governance files (a pre-existing copy-paste
artifact from some unrelated project, predating this round) — it was
replicated into the 9 newly-touched submodules too, for fleet
consistency, rather than silently patched out mid-round. This is a
real, separate, low-priority cleanup item — track it, don't drop it,
but it is NOT part of the current round's scope.

**IMPORTANT — main repo NOT yet committed for this.** The main repo's
own commit (which needs to bump all 9 submodule pointers, plus land
the workable-items DB reconciliation + this file's edits +
GOVERNANCE_INHERITANCE_AUDIT.md/CROSS_CUTTING_GAP_ANALYSIS.md/
UNIFIED_PHASE_ROADMAP.md/mvp3+mvp_final content) is being held back on
PURPOSE: `scripts/commit_all.sh` does `git add -A`, and Agents A
(mvp/ hardening) and B (mvp2/ hardening) are STILL actively writing to
tracked files as of this write — running the full-repo commit wrapper
right now would risk sweeping up their in-progress, possibly-incomplete
edits (§11.4.84 working-tree-quiescence concern, generalized from
mutation-gates to any concurrent subagent write). **Wait for A and B to
fully complete before running `scripts/commit_all.sh` for the main
repo.**

### Queued follow-up (do not drop): anti-bluff testing/Challenges mandate propagation

The operator's anti-bluff covenant (tests + Challenges MUST prove real
end-user-usable functionality, not just green CI — verbatim historical
anchor already lives in `constitution/CLAUDE.md` §11.4 family, esp.
§11.4.1/.2/.5/.6/.27/.50/.52/.69/.98/.107/.123/.134/.142) needs to be
confirmed present (or added) in THIS project's own root
`Constitution.md`/`CLAUDE.md`/`AGENTS.md`/`QWEN.md` if this project
maintains project-level copies distinct from the inherited
`constitution/` submodule, AND cascaded to every owned submodule
under `submodules/*` per §11.4.28/§11.4.35 inheritance rules — universal
content belongs in the `constitution/` submodule (already there);
project-specific restatement/pointer belongs in this repo's own
governance files per the inheritance pattern in
`constitution/CLAUDE.md` "How inheritance works". Action: audit
whether helix_vpn's own `CLAUDE.md`/`AGENTS.md` (this repo's root
files, not the submodule) already carry the required inheritance
pointer + anti-bluff restatement; if any owned submodule
(`submodules/helix_core`, `helix_design`, `helix_edge`, `helix_go`,
`helix_proto`, `helix_qa`, `helix_shims`, `helix_transport`,
`helix_ui`, `llm_orchestrator`, `llm_provider`, `llms_verifier`,
`panoptic`, `security`) is missing its own constitution submodule
inheritance, fix per §11.4.26 (fetch+pull constitution submodule
first, in EACH affected submodule, before editing).

---

## Summary

**Branch:** `main` (single working branch).
**Overall status:** Full MVP specification set COMPLETE. **Design System COMPLETE** (26 files, ~6,700 LOC). **Phase 0 Implementation ADVANCED** — 6 Rust crates, 72 unit/integration tests passing (corrected 2026-07-04 — see "Workable-items DB reconciliation"), submodule pushed. **MVP gap-analysis/hardening round 2 IN PROGRESS** (see "CURRENT ROUND" above).

**Active work (2026-07-04):**
1. ✅ MVP spec set — 11 vols, 126 md/html/pdf, all synced
2. ✅ Constitution — fully integrated, pre-commit hooks active
3. ✅ Design System — OpenDesign, 30+ components, 18+ screens, 4 exports
4. ✅ docs_chain — 3 contexts, all doctor PASS
5. ✅ P0-001: Workspace skeleton — 4 crates, compiled
6. ✅ P0-022: Test rig — 7 scripts, netns+nftables+netem
7. ✅ P0-080/-077: Make + Bench + Spike — 11 targets
8. ✅ P0-004: Transport trait refinement — close(), local_addr(), peer_addr(), mock transport
9. ✅ P0-008: Plain-UDP transport — UdpTransport, UdpConnection, 12 tests
10. ✅ P0-015: Linux TUN device — helix-tun crate, 5+1 tests
11. ✅ P0-011 prep: WireGuard stub — helix-wg crate, 5 modules, 21 tests
12. ✅ Workspace: 39 tests total — 4 crates, 0 failures, 0 errors
13. ✅ Design review fixes — F8, F10, F11, PDFs re-exported
14. ✅ All committed + helix_core submodule advanced

**Next work queue (corrected 2026-07-04, round 2 — see "Workable-items DB
reconciliation" below; P0-011/-018/-025 were already substantively done
and are no longer next-up):**
1. HVPN-P0-018 remainder: client/connector `[[bin]]` targets wired to the
   already-implemented orchestrator core (the core loops + 13 tests exist;
   the binaries do not yet)
2. HVPN-P0-028/HVPN-P0-031: take `helix-masque` from its self-declared
   "research / stub pass" to production-grade QUIC/MASQUE (needed for G2)
3. Platform adapters — VpnService, NEPacketTunnelProvider, WFP

**Locations:** spec: `docs/research/mvp/final/` | design: `docs/design/` | Rust: `submodules/helix_core/` (4 crates, 39 tests) | rig: `scripts/rig/`

---

## Completed Work (highlights)

### 1. Constitution submodule + mandatory submodules
- `constitution/` → `HelixDevelopment/HelixConstitution.git` (branch `main`)
- 11 own-org repos under `submodules/<name>` (flat, lowercase snake_case)
- `install_upstreams` run in each; `.helix-manifest.yaml` audit record
- Pre-commit hook, CI DISABLED (§11.4.156), local enforcement active

### 2. Full MVP specification set (V0–V10)
- 11 volumes, ~140 nano-detail documents, ~11.7K lines in the spine + pass-1 set
- All 16 research docs cited; decisions D1–D8 surfaced
- 46 Mermaid diagrams, SQL DDL, Podman/Docker/K8s manifests
- Every volume adversarial-reviewed (§11.4.142) + reconciled to GO (§11.4.134)
- 126 `.md` / 126 `.html` / 126 `.pdf` — all synced (§11.4.65)

### 3. Workable-items SQLite DB (§11.4.93)
- `docs/workable_items.db` — 484 items (P0: 36, P1: 210, P2: 132, P3: 96)
- Schema: items, item_history, test_diary, gates, operator_block_details, obsolete_details, meta
- Loader: `scripts/workable_items_loader.py` (md-to-db, bidirectional)
- All items start as `Queued` / `Task` status

### 4. Research corpus
- `docs/research/mvp/` — 16 source docs (11 LLM analyses + 5 refined)
- `v09-research/` — 10 per-angle research dossiers (all cited, all verified except wireguard partially)

---

## What Remains

### Done (all subagents completed)
- **D-PKI-CA-TIER** — operator confirmed: two-tier issuing CA as MVP default
- **D-OD-1** — operator confirmed: OpenDesign authoring-layer interpretation
- **vasic-digital component repos** — 8 repos created on GitHub+GitLab + added as submodules
- **Go workable-items binary** — HVPN-P1-150 complete, 6 commands verified
- **DOCX exports** — pipeline updated, all docs have DOCX siblings
- **Design System COMPLETE** — 26 files, ~6,700 LOC
  - OpenDesign 9-section DESIGN.md with light+dark themes + 5 custom palettes
  - tokens.css (200+ CSS custom properties) + Figma Variables-compatible JSON
  - Component library (30+ components, 4 platform variants)
  - Screen wireframes (18+ screens across 8 platforms)
  - Interaction patterns + animation specs
  - Exports: 4 PDF, 4 HTML, 2 PNG screenshots
- **Phase 0 Implementation ADVANCED** — 4 Rust crates, 39 tests, all pushed
  - helix-transport: Transport trait + UDP transport (12 tests)
  - helix-tun: async Linux TUN device (5+1 tests)
  - helix-wg: WireGuard stub + timers (21 tests)
  - helix-core: workspace re-export (0 tests)
- **Phase 0 Implementation — Test Rig** (HVPN-P0-022)
  - 7 scripts (common, setup, teardown, test_reach, test_firewall, test_netem, README)
  - 3-namespace topology (client/bridge/server) with nftables + netem
  - G1 precondition gate scriptable
- **Phase 0 Implementation — Infra** (HVPN-P0-080/-077)
  - Makefile with 11 targets (spike, check, test, bench, rig, clean, etc.)
  - scripts/spike.sh (S0→S4 one-shot verification command)
  - scripts/bench/run.sh + compare.sh (iperf3/ping, CSV output)
- **Design quality review** — 15 findings, 5 fixed (F1,F2,F4,F7,F13)
- **Docs chain** — 'design' context registered (12 nodes, 8 edges, doctor PASS)
- **Submodule pushes** — helix_core (first Rust code), containers (exec fixes)

### Known issues
- `install_upstreams` recipe format mismatch: recipe files use `GIT_SSH_URL` but the script expects `UPSTREAMABLE_REPOSITORY`. Remotes configured manually. Should be fixed upstream in the Upstreamable toolkit.
- `helix_qa` nested submodules (docling) still dirty — pre-existing, not from our work
- `docs_chain` submodule has dirty tracked file — pre-existing, needs upstream fix
- Design system: OpenDesign CLI (`od`) is GNU octal dump, not the OpenDesign tool — no local OpenDesign agent for automated Figma generation

### Deferred
- **Phase 0 Remaining (HIGH)** — P0-011 (boringtun wire), P0-018 (orchestrator three-loop), P0-025 (G1 test with rig), P0-028 (QUIC/MASQUE)
- **Figma design file generation** — requires OpenDesign CLI install or Figma MCP authentication
- **UI implementation** — requires core transport layer stable first
- **Platform adapters** — Android VpnService, iOS NEPacketTunnelProvider, Windows WFP, Linux nftables — each needs helix_core FFI stable

---

## Evidence Locations

| Artifact | Path |
|----------|------|
| MVP spec set | `docs/research/mvp/final/` (126 md/html/pdf) |
| Master index | `docs/research/mvp/final/MASTER_INDEX.md` |
| Spec spine | `docs/research/mvp/final/SPECIFICATION.md` |
| Research corpus | `docs/research/mvp/` (16 source docs) |
| Research dossiers | `docs/research/mvp/final/v09-research/` |
| Workable-items DB | `docs/workable_items.db` (§11.4.93/.95) |
| DB loader | `scripts/workable_items_loader.py` |
| DB loader docs | `docs/scripts/workable_items_loader.md` |
| docs_chain wrapper | `scripts/docs_chain_md_to_db.sh` |
| docs_chain contexts | `.docs_chain/contexts/*.yaml` |
| .gitignore-meta | `.gitignore-meta/*.yaml` (§11.4.77 regen mechanisms) |
| Pre-build gate | `tests/pre_build_verification.sh` (8 invariants) |
| Export script | `scripts/testing/sync_all_markdown_exports.sh` |
| Mermaid helper | `scripts/testing/render_mermaid_blocks.py` |
| Mermaid cache | `.mermaid-cache/` (content-addressed PNGs) |
| Constitution | `constitution/` (submodule) |
| Submodule audit | `.helix-manifest.yaml` |
| Pre-commit hook | `.githooks/pre-commit` |
| CI (DISABLED) | `.github/workflows/constitution.yml.disabled-local-only` |
| **DESIGN SYSTEM** | **`docs/design/`** (26 files, ~6,700 LOC) |
| OpenDesign DESIGN.md | `docs/design/opendesign/helix/DESIGN.md` |
| OpenDesign tokens.css | `docs/design/opendesign/helix/tokens.css` |
| OpenDesign manifest | `docs/design/opendesign/helix/manifest.json` |
| Component reference | `docs/design/opendesign/helix/components.html` |
| Component library doc | `docs/design/components/README.md` |
| Screen wireframes | `docs/design/screens/README.md` |
| Interaction/animation | `docs/design/interaction/README.md` |
| Design master index | `docs/design/README.md` |
| Color tokens JSON | `docs/design/tokens/color.json` |
| Typography tokens | `docs/design/tokens/typography.json` |
| Figma tokens JSON | `docs/design/exports/HelixVPN-Figma-Tokens.json` |
| Design export PDFs | `docs/design/exports/HelixVPN-*.pdf` (4 files) |
| Design screenshots | `docs/design/exports/HelixVPN-Components-*.png` (2 files) |
| Platform-specific | `docs/design/components/{desktop,mobile,aurora,web}/*.md` |

---

## Resumption prompt (§11.4.127)

### SHORT variant

> Continue work on `main` in `/run/media/milosvasic/DATA4TB/Projects/helix_vpn`; read `docs/CONTINUATION.md` first (esp. "ROUND 6" at the top, including the "Both regressions fixed" follow-up sub-section — Rounds 1-6 are all complete and pushed). `HVPN-P0-011` (was a malformed test fixture, not a crypto bug) and `HVPN-P0-049` (helix-ffi TunnelEvent coverage) are both now `Fixed`. Still open: surface the G4 Rust-vs-Go throughput ranking-flip finding to the operator before that gate closes; `HVPN-P0-S1` milestone may now be closable (not re-checked yet); remaining P0-055/058/061/064/067 items are genuinely device-blocked.

### FULL variant

```
You are resuming work on the Helix VPN project.

Repository:  /run/media/milosvasic/DATA4TB/Projects/helix_vpn
Branch:      main
Handoff doc: docs/CONTINUATION.md  ← read this FIRST, especially
             "ROUND 6" at the top.

State at handoff (2026-07-06, Rounds 1-6 all complete and pushed;
HEAD 1747ba8 confirmed identical across origin/github/upstream via
direct git rev-parse, never trusted from memory or a prior handoff's
text alone — see the multi-session lesson further down this file)
-------------------------------------------------------------------
- Round 1 (complete): full MVP spec set, OpenDesign system, Phase-0
  Rust scaffolding, workable-items DB.
- Round 2 (complete): full gap-analysis + enterprise hardening across
  mvp/mvp2/mvp3/final docs, unified phase roadmap.
- Round 2.1 (complete): first fleet-wide decoupling audit.
- Round 3 + 3.1 (complete): real Phase-0 Rust/Go engineering across
  helix_core/helix_edge/helix_go; second, deeper decoupling audit
  (HelixCode contamination in 4 submodules, hardcoded home-dir in
  containers) fixed and re-reviewed to clean GO; fleet-wide
  lowercase-snake_case + broken-self-link cleanup.
- Round 4 (complete): consolidated MVP implementation doc set (13
  sections + coverage ledger), OpenDesign system, helix_proto stubs,
  Challenges/HelixQA banks, connector.md, two rounds of adversarial
  review to GO. Main repo commit d8b9fc1.
- Round 5 (complete): real Phase-0 spike engineering closing gates
  G2/G4/G5/G6 — HVPN-P0-074 (live map.json reconciliation),
  HVPN-P0-049 (helix-ffi crate + real WG handshake via flutter_rust_
  bridge v2 surface, Dart codegen honestly deferred — no toolchain),
  HVPN-P0-045/080 (edge_ab.sh A/B bench harness, Rust-vs-Go MASQUE
  edge, decision deferred pending apples-to-apples conformance),
  HVPN-P0-035 (G2 core survival proven, 2 quantitative sub-bars
  honestly failed and root-caused, not hidden). All independently
  re-verified by the controller (re-ran test suites, not just
  trusted subagent reports).
- Round 6 (complete): 4-stream parallel closeout — fixed the db-to-md
  generator's epic/milestone heading round-trip bug (HVPN-P0-084);
  closed milestones S0/S2/S3/S4/S8 with independently re-verified
  evidence; closed HVPN-P0-077/078 (unified measurement harness +
  decision-log G1 backfill + demo script); scaffolded a real Flutter
  app for HVPN-P0-052 (blocked on missing libgtk+3-devel, needs root).
  Found and REOPENED 2 real regressions: HVPN-P0-049 (helix-ffi no
  longer compiles against helix_core's new TunnelEvent variants) and
  HVPN-P0-011 (helix-wg's encrypt/decrypt path genuinely broken,
  found independently by TWO separate tracks). An independent
  adversarial audit (Track E) re-verified 19/20 already-closed P0
  items by actually re-running cited tests/probes, not trusting DB
  text. Flagged for operator decision: a G4 Rust-vs-Go throughput
  ranking flip in a fresh benchmark run. Main repo HEAD 1747ba8.

What's next
-----------
1. git fetch --all --prune && git submodule foreach --recursive 'git fetch --all --prune --quiet'
2. Read docs/CONTINUATION.md fully, starting at "ROUND 6" at the top.
3. Fix the 2 reopened regressions (both autonomously fixable, no
   device needed): HVPN-P0-011's helix-wg boringtun encrypt/decrypt
   bug (real crypto debugging, don't guess) and HVPN-P0-049's
   helix-ffi compile failure (add real match arms for the 3 new
   TunnelEvent variants, decide what each should map to for the
   Dart-facing TunnelStatus — don't just silence the compiler).
4. Surface the G4 throughput ranking-flip finding to the operator
   before that decision is finalized.
5. Remaining queued P0 items needing a real device (P0-055/058
   Android VpnService+JNI, P0-061/064 iOS staticlib+
   NEPacketTunnelProvider, P0-067 memory soak) are genuinely
   Operator-blocked per §11.4.21 — no physical phone/iPhone reachable
   autonomously in this environment.
6. Otherwise, await new operator instructions.
```
