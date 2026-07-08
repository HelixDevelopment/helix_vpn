# Phase 4 — QA Architecture & Coverage Report

**Project:** HelixVPN  
**Date:** 2026-07-05  
**Author:** QA Architecture subagent  
**Scope:** Define full testability for the MVP, author HelixVPN-specific Challenge and HelixQA banks, close GAP-6, and surface remaining blockers.

---

## Executive summary

Phase 4 delivered the HelixVPN MVP testability layer:

- A complete **Test Coverage Ledger** mapping 91 MVP requirements (53 FRs + 38 NFRs) to §11.4.169 test types, owner subsystems, statuses, Challenge IDs, and HelixQA bank IDs.
- A **HelixVPN Challenge bank** (`submodules/challenges/helix_vpn/`) with six critical-path challenges and driver skeletons.
- A **HelixQA test bank** (`submodules/helix_qa/banks/helix_vpn/`) with eight test cases (six critical paths + two GAP-6 DDoS rows).
- A full **implementation README** covering the test pyramid, evidence model, CI/CD integration plan, and acceptance gates.
- **GAP-6 closed** by explicitly assigning `DDOS` ownership to NFR-413 (control-plane API rate limiting) and NFR-414 (data-plane edge flood resilience).

All deliverables are **spec-level** (defined, not yet automated) because HelixVPN is still in scaffolding/spec phase. No captured PASS is asserted.

---

## Files created / modified

| Path | Type | Purpose |
|---|---|---|
| `docs/research/mvp/final/implementation/09-testing-qa/coverage-ledger.md` | new | Full FR/NFR × test-type coverage ledger; closes GAP-6 |
| `docs/research/mvp/final/implementation/09-testing-qa/README.md` | new | Testing strategy, pyramid, CI/CD plan, gate mapping |
| `submodules/challenges/helix_vpn/README.md` | new | Challenge suite documentation |
| `submodules/challenges/helix_vpn/helix_vpn_challenges.json` | new | Challenge bank (JSON) |
| `submodules/challenges/helix_vpn/helix_vpn_challenges.yaml` | new | Challenge bank (YAML) |
| `submodules/challenges/helix_vpn/drivers/*.sh` | new | 6 driver skeletons |
| `submodules/helix_qa/banks/helix_vpn/README.md` | new | HelixQA bank documentation |
| `submodules/helix_qa/banks/helix_vpn/helix_vpn_bank.yaml` | new | HelixQA bank (YAML) |
| `submodules/helix_qa/banks/helix_vpn/helix_vpn_bank.json` | new | HelixQA bank (JSON) |
| `docs/reviews/mvp-final/findings/phase4-qa-coverage-report.md` | new | This report |

No existing files were modified.

---

## Coverage summary

| Metric | Result |
|---|---|
| MVP functional requirements mapped | 53 |
| MVP non-functional requirements mapped | 38 |
| Total test-type cells mapped | 400+ |
| Challenge IDs minted | 91 (critical paths + per-FR/NFR rows) |
| HelixQA bank IDs minted | 91 |
| MVP DoD criteria covered | 8 / 8 |
| Phase-0 gates covered | 6 / 6 |
| GAP-6 (DDOS ownership) | closed |

### Test-type distribution (MVP cells)

| Type | Count of mapped cells | Notes |
|---|---|---|
| UNIT | ~35 | Policy compiler, IPAM, FSM, schema-diff |
| INT | ~55 | PG+Redis+edge real infra |
| E2E | ~45 | Netns rig |
| SEC | ~40 | Kill-switch, DNS leak, RLS, mTLS |
| CHAOS | ~20 | CP down, Redis drop, iface flap |
| STRESS | ~15 | Reconnect storm, sustained load |
| CONC/RACE | ~15 | IPAM, coordinator hot paths |
| MEM | ~8 | iOS NE, coordinator soak |
| BENCH/PERF | ~25 | Throughput, convergence, revoke |
| SCALE | ~5 | 10 k stream soak (partial MVP) |
| UI/REC | ~20 | Flutter, visual proof |
| CHAL/HQA | 91 | DoD + per-requirement scoring |
| DDOS | 2 | NFR-413, NFR-414 (parked Phase 2) |

---

## Critical-path banks

### Challenges (`submodules/challenges/helix_vpn/`)

| ID | Critical path | Evidence | DoD / Gate |
|---|---|---|---|
| `HVPN-CHAL-AUTH-TUNNEL` | Auth + tunnel establishment | pcap + iperf3 | DoD-2 |
| `HVPN-CHAL-RECONNECT-ROAMING` | Reconnect / roaming | pcap + status log | — |
| `HVPN-CHAL-KILL-SWITCH` | Kill-switch no leak | gap pcap | DoD-7 |
| `HVPN-CHAL-DNS-LEAK` | DNS leak prevention | DNS pcap | DoD-7 |
| `HVPN-CHAL-CONTROL-PLANE-HA` | Control-plane HA | HA pcap + log | — |
| `HVPN-CHAL-CLIENT-UI-VISUAL` | Client UI visual proof | MP4 + vision verdict | DoD-8, §11.4.170 |

### HelixQA (`submodules/helix_qa/banks/helix_vpn/`)

| ID | Critical path | Evidence | DoD / Gate |
|---|---|---|---|
| `HVPN-HQA-AUTH-TUNNEL` | Auth + tunnel establishment | pcap + iperf3 | DoD-2 |
| `HVPN-HQA-RECONNECT-ROAMING` | Reconnect / roaming | pcap + status log | — |
| `HVPN-HQA-KILL-SWITCH` | Kill-switch no leak | gap pcap | DoD-7 |
| `HVPN-HQA-DNS-LEAK` | DNS leak prevention | DNS pcap | DoD-7 |
| `HVPN-HQA-CONTROL-PLANE-HA` | Control-plane HA | HA pcap + log | — |
| `HVPN-HQA-CLIENT-UI-VISUAL` | Client UI visual proof | MP4 + vision verdict | DoD-8, §11.4.170 |
| `HVPN-HQA-NFR413-API-Rate-Limit` | GAP-6: API rate limiting | latency + counter CSV | NFR-413 |
| `HVPN-HQA-NFR414-Edge-Flood` | GAP-6: edge flood resilience | liveness + legit latency CSV | NFR-414 |

---

## GAP-6 closure details

**Original gap:** `requirements-traceability.md` Rev 4 noted that the `DDOS` §11.4.169 test type was defined in the legend but traced to zero requirements.

**Resolution:**

1. The NFR document already minted **HVPN-NFR-413** (control-plane API rate limiting) and **HVPN-NFR-414** (data-plane edge DDoS resilience) in Rev 2.
2. This Phase-4 pass:
   - Mapped both NFRs to the `DDOS` test type in the coverage ledger.
   - Assigned owner subsystems: `svc-api` for NFR-413; `helix-edge` / `helix-transport` for NFR-414.
   - Created Challenge IDs and HelixQA bank IDs for both.
   - Marked the MVP cells `not-applicable-MVP` with reason `single-node-selfhost`.
   - Documented the mechanical re-arm trigger: `deployment.topology == "multi-tenant-ha"` in Phase 2.

---

## Verification performed

1. **Read and reconciled** `10-testing-acceptance-and-qa.md`, `99-source-coverage-ledger.md`, `functional-requirements.md`, `nonfunctional-requirements.md`, and `requirements-traceability.md`.
2. **Inspected** `submodules/challenges/banks/examples/` and `submodules/challenges/banks/yole/` to derive the Challenge JSON/YAML schema.
3. **Inspected** `submodules/helix_qa/banks/*.yaml` (especially `ddos-ratelimit-comprehensive.yaml` and `cli-agent-e2e-flow.yaml`) to derive HelixQA bank conventions.
4. **Confirmed** `v08-testing/ddos.md` exists and provides the DDoS harness spec referenced by NFR-414.
5. **Confirmed** no existing `helix_vpn/` directories existed in the submodules before creation.
6. **Validated** all new JSON files are syntactically well-formed (confirmed during write).

---

## Blockers requiring coordinator decision

| # | Blocker | Impact | Recommended next step |
|---|---|---|---|
| **B1** | **No runnable harness exists yet.** All challenges and HelixQA cases are spec-level skeletons; drivers are placeholders. | Coverage is `defined`, not `automated`. | Coordinate Phase-0/Phase-1 implementation to fill in `rig/`, `helixvpnctl`, and `vision_engine` integration. |
| **B2** | **DDOS design-attack-rate and SLO budgets are unmeasured.** NFR-413/NFR-414 targets are `UNVERIFIED`. | Phase-2 DDOS gate cannot be calibrated now. | Run the Phase-2 soak rig once the multi-tenant HA topology exists; set concrete `ATTACK_PPS`, legit-latency budget, and supervisor restart budget. |
| **B3** | **iOS memory gate requires real hardware.** Simulator does not reproduce the jetsam ceiling per QA-D3. | MEM gate for FR-1004/NFR-500 cannot be fully automated on CI-less hosts. | Procure on-device test hardware or record honest `SKIP: hardware_not_present` per §11.4.3. |
| **B4** | **Existing submodules are git submodules.** Files created inside `submodules/challenges/helix_vpn/` and `submodules/helix_qa/banks/helix_vpn/` modify the submodule worktree but are not committed. | Coordinator must decide whether to commit these additions inside the submodule repos or to keep the HelixVPN-specific content in the parent project only. | This is a repo-layout decision; current files follow the deliverable instructions. |

---

## Remaining work

1. **Implement driver skeletons** under `submodules/challenges/helix_vpn/drivers/` once the netns rig, `helixvpnctl`, and core CLI are available.
2. **Wire HelixQA executor** to the new bank once the `helix_qa` runner integration is ready.
3. **Generate coverage-ledger SQL projection** into `docs/.workable_items.db` when the `doc_processor` tooling is operational.
4. **Calibrate DDOS budgets** in Phase 2 and flip NFR-413/NFR-414 cells from `not-applicable-MVP` to `REQUIRED`.
5. **Add golden-good / golden-bad fixtures** for every analyzer referenced in the banks.

---

## Conclusion

Phase 4 closes the MVP testability specification. Every MVP requirement is mapped to a test type, owner, Challenge ID, and HelixQA bank ID. GAP-6 is closed by assigning DDOS ownership to NFR-413 and NFR-414. The critical-path banks are authored and parked, ready to be automated as the implementation phases deliver the underlying harness and CLI.

No code changes were committed per instructions; all work is staged for coordinator review.
