# HelixVPN MVP — Testing & QA Implementation Strategy

**Revision:** 1  
**Last modified:** 2026-07-05T14:20:00Z  
**Status:** active — Phase 4 QA architecture deliverable  
**Authority:** `docs/research/mvp/final/10-testing-acceptance-and-qa.md` (canonical QA contract)  
**Scope:** Full testing strategy, test pyramid, CI/CD integration plan, and links to HelixVPN-specific Challenge / HelixQA banks.

---

## 1. One-sentence rule

> A green test suite is not proof the feature works. It only counts as proof if a real artifact — a packet capture, a throughput number, a database rowset, a screen recording — was captured *while the feature ran* and shows the user-visible outcome actually happened.

This document operationalizes that rule for HelixVPN.

---

## 2. Test pyramid

```
                    ┌─────────────┐
                    │   REC/UI    │  ← window-scoped MP4 + vision_engine OCR verdict
                    │  (top gate) │
                   ┌┴─────────────┴┐
                   │ CHAL + HQA   │  ← challenges + helix_qa banks score DoD ACs
                  ┌┴───────────────┴┐
                  │ SEC·STRESS·CHAOS│  ← kill-switch, DPI, reconnect, fail-static
                 ┌┴─────────────────┴┐
                 │       E2E         │  ← netns + nftables + netem rig
                ┌┴───────────────────┴┐
                │        INT          │  ← real PG + Redis + edge via containers
               ┌┴─────────────────────┴┐
               │        UNIT           │  ← pure logic; mocks allowed only here
               └───────────────────────┘
```

### Layer responsibilities

| Layer | Proves | Harness | Mocks? |
|---|---|---|---|
| **UNIT** | Policy compiler, IPAM, delta diff, MASQUE framing, status FSM | `cargo test`, `go test`, `dart test` | Only here |
| **INT** | Enrollment, RLS, reconcile, cert rotation, presence TTL | `containers/pkg/boot` PG+Redis+edge | No |
| **E2E** | Real tunnel carries real packets; authorized reach; default-deny | Netns + nftables + netem rig | No |
| **SEC/STR/CHAOS** | Kill-switch, DNS leak, DPI evasion, fail-static, reconnect | `rig/`, `scripts/stress_chaos.sh` | No |
| **CHAL/HQA** | DoD ACs scored on captured evidence; defeats B1–B5 | `submodules/challenges/`, `submodules/helix_qa/` | No |
| **REC/UI** | User-visible flows rendered and match core FSM | `panoptic` + `vision_engine` | No |

---

## 3. Test-type taxonomy (§11.4.169)

| Code | Type | MVP status | Owner subsystem |
|---|---|---|---|
| `UNIT` | Unit | required | `helix-core`, `helix-go`, `helix-ui` |
| `INT` | Integration | required | `svc-*`, `helix-core` |
| `E2E` | End-to-end | required | netns rig / edge |
| `FA` | Full-automation | required | `make spike` / `make qa` |
| `SEC` | Security | required | security / `helix-core` |
| `DDOS` | DDoS / load-flood | **parked** (GAP-6 closed) | `svc-api` (NFR-413), `helix-edge` (NFR-414) |
| `STRESS` | Stress | required | `svc-coordinator`, `helix-edge` |
| `CHAOS` | Chaos | required | `svc-coordinator`, `helix-edge` |
| `CONC` | Concurrency | required | `svc-ipam`, `svc-coordinator` |
| `RACE` | Race / deadlock | required | `helix-core`, `helix-go` |
| `MEM` | Memory | required | `shim-apple`, `svc-coordinator` |
| `BENCH` | Benchmarking | required | `bench.sh`, `criterion` |
| `PERF` | Performance | required | `svc-coordinator`, `helix-edge` |
| `SCALE` | Scaling | partial MVP / full P2 | `svc-coordinator` |
| `UI` | UI / UX | required | `helix-ui` |
| `REC` | Recorded evidence | required | `panoptic` |
| `CHAL` | Challenges | required (DoD) | `submodules/challenges/helix_vpn/` |
| `HQA` | HelixQA | required (DoD) | `submodules/helix_qa/banks/helix_vpn/` |

---

## 4. CI/CD integration plan

> **Constraint:** `docs/research/mvp/final/10-testing-acceptance-and-qa.md` §9 / §11.4.156 — no active remote CI. All gates are local.

### Local gate layers

```
pre-commit hook
├── sh -n on shell scripts
├── schema-lint (no durable flow table)
├── no-logging payload lint
├── mutation-residue scan
└── UNIT (fast)

pre-push hook
├── UNIT (full)
├── INT (containers boots PG+Redis)
├── SEC subset (secret audit, RLS fixture)
└── propagation gates (buf breaking-change lint)

make test
├── UNIT
├── INT
├── E2E (netns rig)
├── SEC
├── STRESS + CHAOS
└── report to qa-results/

make qa
├── CHAL/HQA bank run
├── REC/UI bank run
└── coverage-ledger regeneration
```

### CI/CD parity (future)

When remote CI is re-enabled, the same local targets are wrapped:

```yaml
# .github/workflows/qa.yml (future)
- run: make test
- run: make qa
- run: scripts/verify-coverage-ledger.sh
```

No new test logic lives in CI; CI only orchestrates the existing local harness.

---

## 5. Evidence model

Every feature class maps to a §11.4.69 sink-side evidence class.

| Feature class | Evidence class | Artifact |
|---|---|---|
| tunnel carries traffic | `network_throughput` | `iperf3 -J` + pcap with ≥1 inner packet |
| authorized reachability | `network_connectivity` | `curl` 200 + pcap SYN→SYN-ACK |
| default-deny | `network_connectivity` (negative) | pcap SYN out, zero SYN-ACK |
| kill-switch / DNS leak | `wifi_link` / negative | host pcap: zero plaintext / zero :53 |
| transport escalation | `network_connectivity` | `StatusReport.transport=="masque-h3"` + tshark H3 |
| policy reconcile latency | `counter_delta` | `helix_reconcile_seconds` p99 < 1 s |
| revoke latency | `counter_delta` | revoke→edge-enforcement timing CSV |
| key-never-leaves | negative | FFI boundary scan + log scan |
| iOS NE memory | `MEM` | Instruments `.trace` / `footprint` RSS |
| UI connect flow | `REC` | window-scoped MP4 + vision verdict |

---

## 6. HelixVPN-specific banks

### Challenges

Path: `submodules/challenges/helix_vpn/`

| Challenge ID | Critical path | Evidence |
|---|---|---|
| `HVPN-CHAL-AUTH-TUNNEL` | Auth + tunnel establishment | pcap + iperf3 JSON |
| `HVPN-CHAL-RECONNECT-ROAMING` | Reconnect / roaming | pcap + status log |
| `HVPN-CHAL-KILL-SWITCH` | Kill-switch | gap pcap |
| `HVPN-CHAL-DNS-LEAK` | DNS leak prevention | DNS pcap + resolvers JSON |
| `HVPN-CHAL-CONTROL-PLANE-HA` | Control-plane HA | HA pcap + CP down log |
| `HVPN-CHAL-CLIENT-UI-VISUAL` | Client UI visual proof | MP4 + vision verdict |

### HelixQA

Path: `submodules/helix_qa/banks/helix_vpn/`

| Test-case ID | Critical path | Evidence |
|---|---|---|
| `HVPN-HQA-AUTH-TUNNEL` | Auth + tunnel establishment | pcap + iperf3 |
| `HVPN-HQA-RECONNECT-ROAMING` | Reconnect / roaming | pcap + status log |
| `HVPN-HQA-KILL-SWITCH` | Kill-switch | gap pcap |
| `HVPN-HQA-DNS-LEAK` | DNS leak prevention | DNS pcap |
| `HVPN-HQA-CONTROL-PLANE-HA` | Control-plane HA | HA pcap + log |
| `HVPN-HQA-CLIENT-UI-VISUAL` | Client UI visual proof | MP4 + vision verdict |
| `HVPN-HQA-NFR413-API-Rate-Limit` | GAP-6: API rate limiting | latency + counter CSV |
| `HVPN-HQA-NFR414-Edge-Flood` | GAP-6: edge flood resilience | liveness + legit latency CSV |

---

## 7. GAP-6 closure

**Gap:** `DDOS` test type had no owning FR/NFR.

**Fix:**
- **NFR-413** — control-plane API rate limiting (token bucket per API key/device) → `svc-api` owns it.
- **NFR-414** — data-plane edge DDoS/UDP-flood resilience → `helix-edge` / `helix-transport` owns it.

At MVP the runnable cells are `not-applicable-MVP` (`single-node-selfhost`); the banks are authored-now-parked and mechanically re-arm when `deployment.topology == "multi-tenant-ha"` in Phase 2.

---

## 8. Acceptance gates

### Phase 0 — spike exit gates

| Gate | Bar | Evidence | Test types |
|---|---|---|---|
| G1 | plain-UDP ≥ 80 % bare link | pcap + iperf3 CSV | E2E, BENCH, CHAL |
| G2 | MASQUE ≥ 50 % plain-UDP through DPI block | pcap + tshark + CSV | E2E, SEC, BENCH |
| G3 | iOS NE memory headroom ≥ 30 % | Instruments `.trace` | MEM |
| G4 | Go-vs-Rust edge decision | `edge_compare.csv` | BENCH, PERF, RACE |
| G5 | flutter_rust_bridge FFI | FFI round-trip log + UI recording | UNIT, UI |
| G6 | push-based reconcile | reconcile event log | INT, E2E |

### Phase 1 — MVP DoD

| AC | Criterion | Evidence | Primary challenge |
|---|---|---|---|
| AC1 | self-host from zero | terminal recording + healthchecks | `HVPN-CHAL-FR901-Init-From-Zero` |
| AC2 | enroll connector+client; reach authorized host | E2E netns + pcap | `HVPN-CHAL-AUTH-TUNNEL` |
| AC3 | deny unauthorized host | negative E2E pcap | `HVPN-CHAL-FR201-Default-Deny` |
| AC4 | auto-escalate to MASQUE | `StatusReport.transport` + tshark H3 | `HVPN-CHAL-FR006-Auto-Ladder` |
| AC5 | policy edit < 1 s | `helix_reconcile_seconds` p99 | `HVPN-CHAL-FR205-Policy-Reconcile` |
| AC6 | revoke < 1 s | revoke→edge timing CSV | `HVPN-CHAL-FR107-Revoke-Speed` |
| AC7 | kill-switch + DNS leak | host pcap zero leak | `HVPN-CHAL-KILL-SWITCH`, `HVPN-CHAL-DNS-LEAK` |
| AC8 | no durable log + 3 apps | schema-lint + UX recordings | `HVPN-CHAL-FR801-No-Flow-Table`, `HVPN-CHAL-CLIENT-UI-VISUAL` |

---

## 9. Risk ordering

The suite runs highest-risk first:

1. kill-switch / DNS leak (AC7)
2. default-deny / RLS (AC3)
3. revoke < 1 s (AC6)
4. key-never-leaves (FR-103)
5. authorized reach (AC2)
6. transport escalation (AC4)
7. policy reconcile (AC5)
8. UI / recorded evidence (AC9)

Only after the irreversible-security floor is GREEN does the rest of the pyramid run.

---

## 10. Determinism & mutation

- Every PASS runs N=3 (normal) / N=10 (cycle-validation) against the same artifact MD5 + same rig.
- Divergent runs are auto-FAIL.
- Every gate has a paired §1.1 mutation that disables the protection and expects RED.
- Every analyzer ships with golden-good + golden-bad fixtures.

---

## 11. Links

- Full coverage ledger: [`coverage-ledger.md`](coverage-ledger.md)
- Canonical QA contract: [`../../10-testing-acceptance-and-qa.md`](../../10-testing-acceptance-and-qa.md)
- Challenges bank: [`submodules/challenges/helix_vpn/`](../../../../../../submodules/challenges/helix_vpn/)
- HelixQA bank: [`submodules/helix_qa/banks/helix_vpn/`](../../../../../../submodules/helix_qa/banks/helix_vpn/)
- Phase 4 findings report: [`docs/reviews/mvp-final/findings/phase4-qa-coverage-report.md`](../../../../../reviews/mvp-final/findings/phase4-qa-coverage-report.md)
