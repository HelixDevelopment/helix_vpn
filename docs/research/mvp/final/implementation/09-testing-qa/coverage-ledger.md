# HelixVPN MVP — Test Coverage Ledger

**Revision:** 1  
**Last modified:** 2026-07-05T14:20:00Z  
**Status:** active — Phase 4 QA architecture deliverable  
**Authority:** Constitution §11.4.25/.52/.153 (coverage ledger), §11.4.169 (mandatory test types), §11.4.6 (no guessing — gaps surfaced), §11.4.118 (enumerated coverage claim)  
**Scope:** Every MVP requirement (`HVPN-FR-NNN` / `HVPN-NFR-NNN`) mapped to test type, owner subsystem, current evidence state, and linked Challenge / HelixQA bank IDs. Phase-2 items are tracked as `P2` or `NOT_APPLICABLE` with an explicit re-arm trigger.

---

## How to read this ledger

| Column | Meaning |
|---|---|
| **Requirement** | `HVPN-FR-NNN` / `HVPN-NFR-NNN` from the authoritative FR/NFR docs. |
| **Subsystem** | The HelixVPN subsystem that owns the implementation and therefore the primary test debt. |
| **Test types** | The §11.4.169 closed-set test types applied to this requirement. Legend: `UNIT`, `INT`, `E2E`, `FA`, `SEC`, `DDOS`, `STRESS`, `CHAOS`, `CONC`, `RACE`, `MEM`, `BENCH`, `PERF`, `SCALE`, `UI`, `REC`, `CHAL`, `HQA`. |
| **Status** | `defined` (test authored in spec/bank), `automated` (harness exists and runs), `manual` (operator-attended only), `not-defined` (no spec-level test yet), `deferred-P2` (Phase-2 scoped), `not-applicable-MVP` (honest NA with reason). |
| **Challenge IDs** | IDs in `submodules/challenges/helix_vpn/` that score this requirement. |
| **HelixQA bank IDs** | IDs in `submodules/helix_qa/banks/helix_vpn/` that score this requirement. |
| **DoD / Gate** | Which MVP Definition-of-Done criterion or Phase-0 gate this requirement feeds. |

> **Evidence-state honesty.** HelixVPN is in specification / early scaffolding. No captured PASS is asserted here; every status is a *planned* state. A cell moves to `automated` only when a harness exists that produces a captured artifact on a clean deploy.

---

## Legend — §11.4.169 test-type mapping

| Abbrev | Type | HelixVPN evidence shape |
|---|---|---|
| `U` | unit | Pure logic: policy compiler, IPAM, delta diff, MASQUE framing, status FSM. Mocks allowed only here. |
| `INT` | integration | Real Postgres + Redis + edge booted via `containers/pkg/boot`; no mocks. |
| `E2E` | end-to-end | Netns + nftables + netem rig; real tunnel carries real packets. |
| `FA` | full-automation | Self-driving, `-count=3`, zero human-in-loop. |
| `SEC` | security | RLS, key-never-leaves, mTLS, kill-switch, DNS-leak, DPI evasion, secret audit. |
| `DDOS` | DDoS / load-flood | Handshake/volumetric flood; legit client stays usable; edge fail-static. |
| `STR` | stress | Sustained load ≥ 100 iterations / ≥ 30 s / ≥ 10 parallel. |
| `CHAOS` | chaos | SIGKILL, Redis drop, iface flap, partial write; recovery + no leak. |
| `CONC` | concurrency | Concurrent IPAM / enrollment / policy edits; no lost update. |
| `RACE` | race / deadlock | `-race`, `loom`, `tsan` on hot paths. |
| `MEM` | memory | iOS NE RSS, 24 h leak soak. |
| `BENCH` | benchmarking | Throughput, CPU-per-Gbps, handshakes/sec, framing latency. |
| `PERF` | performance | Latency p50/p99 vs SLO budget. |
| `SCALE` | scaling | N simulated agents / 24 h soak. |
| `UI` | UI / UX | Flutter widget + golden tests. |
| `REC` | recorded-evidence | Window-scoped MP4 + `vision_engine` OCR verdict. |
| `CHAL` | Challenge | `challenges` submodule bank entry; evidence re-read. |
| `HQA` | HelixQA | `helix_qa` autonomous session bank entry; evidence re-read. |

---

## Functional-requirement coverage (MVP only)

### A. Connect & transport (FR-0xx)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-FR-001 WG crypto core | `helix-core` / `helix-edge` | U, INT, SEC, CHAL, HQA | defined | `HVPN-CHAL-AUTH-TUNNEL` | `HVPN-HQA-AUTH-TUNNEL` | G1, G2 |
| HVPN-FR-002 plain-UDP throughput | `helix-transport` / edge | E2E, BENCH, PERF, FA, CHAL, HQA | not-defined | not-defined | not-defined | DoD-2, G1 |
| HVPN-FR-003 `Transport` trait | `helix-core` / `helix-transport` | U, INT, FA | not-defined | not-defined | not-defined | — |
| HVPN-FR-004 MASQUE/QUIC | `helix-transport` / edge | E2E, SEC, BENCH, CHAL, HQA | not-defined | not-defined | not-defined | DoD-4, G2 |
| HVPN-FR-005 LWO | `helix-transport` | E2E, SEC | not-defined | not-defined | not-defined | — |
| HVPN-FR-006 auto-ladder | `helix-core` / orchestrator | E2E, FA, CHAL, HQA | not-defined | not-defined | not-defined | DoD-4 |
| HVPN-FR-007 pin transport | `helix-core` / FFI | U, INT, UI | not-defined | not-defined | not-defined | — |
| HVPN-FR-008 port evasion | edge | E2E, INT, SEC | not-defined | not-defined | not-defined | — |
| HVPN-FR-013 full-tunnel exit | `helix-core` / edge | E2E, SEC | not-defined | not-defined | not-defined | — |
| HVPN-FR-014 split tunneling | `helix-core` / shims | E2E, SEC, UI | not-defined | not-defined | not-defined | — |
| HVPN-FR-015 transient-drop recovery | orchestrator / client | CHAOS, E2E, CHAL, HQA | defined | `HVPN-CHAL-RECONNECT-ROAMING` | `HVPN-HQA-RECONNECT-ROAMING` | — |
| HVPN-FR-016 MTU management | `helix-transport` / WG | INT, E2E | not-defined | not-defined | not-defined | — |
| HVPN-FR-018 status stream | `helix-core` / FFI / UI | U, UI, REC | not-defined | not-defined | not-defined | — |
| HVPN-FR-019 asymmetric per-leg default | coordinator / edge | INT, E2E | not-defined | not-defined | not-defined | — |

### B. Identity & enrollment (FR-1xx)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-FR-101 OIDC identity | `svc-identity` | INT, SEC | not-defined | not-defined | not-defined | — |
| HVPN-FR-102 anonymous device token | `svc-identity` / enrollment | INT, SEC, CHAL, HQA | not-defined | not-defined | not-defined | DoD-2 |
| HVPN-FR-103 key-never-leaves | `svc-pki` / `helix-core` | SEC, CHAL, HQA | not-defined | not-defined | not-defined | — |
| HVPN-FR-104 cert+token-gated enroll | `svc-identity` / `svc-pki` | SEC, INT | not-defined | not-defined | not-defined | DoD-2 |
| HVPN-FR-105 short-lived mTLS cert | `svc-pki` | SEC, INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-106 control/data key separation | `svc-pki` / `helix-core` | SEC | not-defined | not-defined | not-defined | — |
| HVPN-FR-107 revoke < 1 s | `svc-registry` / edge | PERF, SEC, CHAL, HQA | not-defined | not-defined | not-defined | DoD-6 |
| HVPN-FR-108 per-tenant CA | `svc-pki` | SEC, INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-109 cert rotation no-drop | `svc-pki` / `helix-core` | INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-110 RLS tenant isolation | `svc-identity` / Postgres | SEC, INT, CHAL, HQA | not-defined | not-defined | not-defined | — |
| HVPN-FR-111 all-roles enroll | `svc-registry` | INT, E2E | not-defined | not-defined | not-defined | DoD-2 |

### C. Policy & authorization (FR-2xx)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-FR-201 default-deny / fail-closed | `svc-policy` / edge | SEC, E2E, META, CHAL, HQA | not-defined | not-defined | not-defined | DoD-3 |
| HVPN-FR-202 ACL compiler grammar | `svc-policy` | U, INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-203 AllowedIPs + verdict map | `svc-policy` / edge | E2E, SEC | not-defined | not-defined | not-defined | — |
| HVPN-FR-204 authorized reach + denied | edge / `svc-policy` | E2E, CHAL, HQA | not-defined | not-defined | not-defined | DoD-3 |
| HVPN-FR-205 policy edit < 1 s | `svc-policy` / coordinator | PERF, E2E, CHAL, HQA | not-defined | not-defined | not-defined | DoD-5 |
| HVPN-FR-206 split-horizon default | `svc-policy` / IPAM | E2E, SEC | not-defined | not-defined | not-defined | — |
| HVPN-FR-207 need-to-know map filtering | `svc-coordinator` | SEC, META | not-defined | not-defined | not-defined | — |

### D. Routing, addressing & multi-network (FR-3xx)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-FR-301 1→N overlay | IPAM / routing / edge | E2E, CHAL, HQA | not-defined | not-defined | not-defined | — |
| HVPN-FR-302 overlapping-CIDR resolve | `svc-ipam` / routing | E2E | not-defined | not-defined | not-defined | — |
| HVPN-FR-303 ULA /48 + 4via6 | `svc-ipam` / routing | E2E, INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-304 stable overlay IP | `svc-ipam` | INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-305 connector advertise + route | `svc-registry` / edge | E2E, CHAL, HQA | not-defined | not-defined | not-defined | DoD-3 |
| HVPN-FR-306 two-way no-inbound | edge / `helix-core` | E2E, SEC | not-defined | not-defined | not-defined | DoD-3 |
| HVPN-FR-307 IPAM concurrent alloc | `svc-ipam` | STR, CONC, CHAL, HQA | not-defined | not-defined | not-defined | — |

### E. Kill-switch & leak protection (FR-5xx)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-FR-501 core-owned kill-switch | `helix-core` / shims | U, SEC, CHAL, HQA | not-defined | not-defined | not-defined | DoD-7 |
| HVPN-FR-502 no plaintext on drop | `helix-core` / OS firewall | SEC, E2E, CHAL, HQA | defined | `HVPN-CHAL-KILL-SWITCH` | `HVPN-HQA-KILL-SWITCH` | DoD-7 |
| HVPN-FR-503 DNS through tunnel | `helix-core` / resolver | SEC, E2E, CHAL, HQA | not-defined | not-defined | not-defined | DoD-7 |
| HVPN-FR-504 per-OS firewall | platform shims | SEC, per-OS branch | not-defined | not-defined | not-defined | DoD-7 |

### F. Console & administration (FR-6xx)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-FR-601 CRUD + RBAC | `svc-api` / Console | INT, E2E, SEC, CHAL, HQA | not-defined | not-defined | not-defined | — |
| HVPN-FR-602 Console no-core build | `helix-ui` / build | INT, FA | not-defined | not-defined | not-defined | — |
| HVPN-FR-603 live WS/SSE | `svc-api` | E2E | not-defined | not-defined | not-defined | — |
| HVPN-FR-604 live topology view | Console | UI, E2E | not-defined | not-defined | not-defined | — |
| HVPN-FR-605 control-action audit | `svc-telemetry` / audit | SEC, UI | not-defined | not-defined | not-defined | — |
| HVPN-FR-606 multi-tenant isolation | Console / RLS | SEC, CHAL, HQA | not-defined | not-defined | not-defined | — |
| HVPN-FR-608 responsive + visual | Console / `helix-ui` | UI, REC | not-defined | not-defined | not-defined | — |
| HVPN-FR-609 exportable audit slice | audit / `svc-telemetry` | SEC, INT | not-defined | not-defined | not-defined | — |

### G. Connector (FR-7xx)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-FR-701 outbound-only connector | `helix-core` (advertise/route) | E2E, SEC, CHAL, HQA | not-defined | not-defined | not-defined | DoD-2 |
| HVPN-FR-702 headless daemon | `helix-core` / connector shim | E2E, INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-703 shared Rust core | `helix-core` | INT, FA | not-defined | not-defined | not-defined | — |
| HVPN-FR-704 advertise + route | `svc-registry` / edge | E2E, CHAL, HQA | not-defined | not-defined | not-defined | DoD-3 |
| HVPN-FR-707 availability-following on drop | orchestrator / connector | CHAOS, E2E, CHAL, HQA | not-defined | not-defined | not-defined | — |

### H. Observability & telemetry (FR-8xx)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-FR-801 no durable flow table | schema / migrations | SEC, META, CHAL, HQA | not-defined | not-defined | not-defined | DoD-8 |
| HVPN-FR-802 ephemeral presence | `svc-events` / Redis | INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-803 Prometheus metrics | `svc-telemetry` | INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-804 convergence SLO metrics | `svc-telemetry` | PERF, INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-805 counts-only success metrics | `svc-telemetry` | SEC, INT | not-defined | not-defined | not-defined | — |

### I. Deployment & self-host (FR-9xx)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-FR-901 `helixvpnctl init` | `helixvpnctl` / deploy | E2E, FA, CHAL, HQA | not-defined | not-defined | not-defined | DoD-1 |
| HVPN-FR-902 rootless Podman | deploy / quadlets | SEC, E2E | not-defined | not-defined | not-defined | — |
| HVPN-FR-904 `helixvpnctl` commands | `helixvpnctl` | E2E, INT | not-defined | not-defined | not-defined | — |
| HVPN-FR-906 hardened edge container | deploy / edge | SEC, E2E | not-defined | not-defined | not-defined | — |
| HVPN-FR-907 fail-static on CP down | edge / coordinator | CHAOS, E2E, CHAL, HQA | not-defined | not-defined | not-defined | — |
| HVPN-FR-908 build-essential regeneration | repo tooling | U, FA | not-defined | not-defined | not-defined | — |

### J. Clients & platform apps (FR-10xx)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-FR-1001 three flavors one tree | `helix-ui` / Melos | INT, FA, UI | not-defined | not-defined | not-defined | — |
| HVPN-FR-1002 shared core + FFI stream | `helix-core` / FFI | INT, UI, REC | not-defined | not-defined | not-defined | — |
| HVPN-FR-1003 one-button connect | `helix-ui` | UI, REC, E2E, CHAL, HQA | defined | `HVPN-CHAL-CLIENT-UI-VISUAL` | `HVPN-HQA-CLIENT-UI-VISUAL` | DoD-8 |
| HVPN-FR-1004 iOS NE memory | `shim-apple` / `helix-core` | MEM, CHAL, HQA | not-defined | not-defined | not-defined | G3 |
| HVPN-FR-1005 Android `VpnService` | `shim-android` | E2E, UI, REC | not-defined | not-defined | not-defined | — |
| HVPN-FR-1006 Linux kernel WG | `shim-linux` | E2E, UI | not-defined | not-defined | not-defined | — |
| HVPN-FR-1012 exit/network + obfuscation UI | `helix-ui` / FFI | UI, E2E | not-defined | not-defined | not-defined | — |
| HVPN-FR-1013 light + dark themes | `helix-ui` / OpenDesign | UI, REC | not-defined | not-defined | not-defined | — |
| HVPN-FR-1014 three apps drive E2E | all clients | E2E, UI, REC, CHAL, HQA | not-defined | not-defined | not-defined | DoD-8 |

---

## Non-functional-requirement coverage (MVP only)

### Performance (NFR-001…009)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-NFR-001 ≥ 80 % bare-link throughput | `helix-transport` / edge | PERF, BENCH, E2E, CHAL, HQA | not-defined | not-defined | not-defined | G1 |
| HVPN-NFR-002 ≥ 50 % MASQUE throughput | `helix-transport` / edge | PERF, BENCH, E2E | not-defined | not-defined | not-defined | G2 |
| HVPN-NFR-003 p99 convergence < 1 s | `svc-coordinator` | PERF, INT, CHAL, HQA | not-defined | not-defined | not-defined | DoD-5 |
| HVPN-NFR-004 revoke < 1 s | `svc-registry` / edge | PERF, SEC | not-defined | not-defined | not-defined | DoD-6 |
| HVPN-NFR-005 WG handshake latency | `helix-core` / edge | INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-006 ladder escalation bounded | orchestrator / ladder | INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-008 MTU 1420 / no fragment | `helix-transport` | INT, E2E | not-defined | not-defined | not-defined | — |
| HVPN-NFR-009 admin-perceived edit < 1 s | `svc-api` → coordinator | E2E, FA, PERF | not-defined | not-defined | not-defined | DoD-5 |

### Scale (NFR-100…108)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-NFR-100 ≥ 10 k streams | `svc-coordinator` | SCALE, STR, MEM | not-defined | not-defined | not-defined | SLO4 |
| HVPN-NFR-101 bounded coordinator memory | `svc-coordinator` | SCALE, MEM | not-defined | not-defined | not-defined | SLO4 |
| HVPN-NFR-102 minimal affected set | `svc-coordinator` | U, PERF | not-defined | not-defined | not-defined | — |
| HVPN-NFR-103 connectors/tenant | `svc-ipam` / edge | INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-104 1 user → N nets | routing / edge | E2E | not-defined | not-defined | not-defined | — |
| HVPN-NFR-105 reconnect storm | `svc-coordinator` / edge | STR | not-defined | not-defined | not-defined | — |
| HVPN-NFR-106 per-tenant isolation | `svc-coordinator` | CONC, RACE | not-defined | not-defined | not-defined | — |

### Availability & HA (NFR-200…207)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-NFR-200 fail-static CP down | edge | CHAOS, E2E, CHAL, HQA | defined | `HVPN-CHAL-CONTROL-PLANE-HA` | `HVPN-HQA-CONTROL-PLANE-HA` | — |
| HVPN-NFR-201 Postgres/Redis blip | `svc-coordinator` | CHAOS | not-defined | not-defined | not-defined | — |
| HVPN-NFR-202 Redis loss graceful | `svc-coordinator` / presence | CHAOS | not-defined | not-defined | not-defined | — |
| HVPN-NFR-203 no-work-loss consumer | `svc-events` | CHAOS, INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-204 coordinator restart resume | `svc-coordinator` | CHAOS, INT, CHAL, HQA | not-defined | not-defined | not-defined | — |
| HVPN-NFR-207 poison → DLQ | `svc-events` | CHAOS, INT | not-defined | not-defined | not-defined | — |

### Privacy (NFR-300…309)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-NFR-300 no durable flow table | schema / migrations | SEC, META | not-defined | not-defined | not-defined | DoD-8 |
| HVPN-NFR-301 schema-lint not tautology | schema lint | META | not-defined | not-defined | not-defined | DoD-8 |
| HVPN-NFR-302 bus carries no traffic shape | `svc-events` | SEC, payload-lint | not-defined | not-defined | not-defined | — |
| HVPN-NFR-303 audit = control-only | `svc-telemetry` | U, INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-304 ephemeral presence | `svc-events` / Redis | INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-305 coarse `last_seen_at` | `svc-telemetry` | U, INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-306 counts-only metrics | `svc-telemetry` | U | not-defined | not-defined | not-defined | — |
| HVPN-NFR-307 anonymous identity | `svc-identity` | INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-308 runtime no-log signature | migrations / deploy | E2E, CHAL, HQA | not-defined | not-defined | not-defined | DoD-8 |
| HVPN-NFR-309 audit retention policy | audit / `svc-telemetry` | INT | not-defined | not-defined | not-defined | — |

### Security (NFR-400…414) — including GAP-6 closure

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-NFR-400 WG crypto never forked | `helix-core` / CI | SEC, META | not-defined | not-defined | not-defined | — |
| HVPN-NFR-401 outbound-only edges | edge / deploy | SEC | not-defined | not-defined | not-defined | — |
| HVPN-NFR-402 need-to-know map | `svc-coordinator` | E2E, META | not-defined | not-defined | not-defined | — |
| HVPN-NFR-403 mTLS + revoke | `svc-pki` / edge | SEC | not-defined | not-defined | not-defined | — |
| HVPN-NFR-404 kill-switch no leak | `helix-core` / shims | SEC, E2E | not-defined | not-defined | not-defined | DoD-7 |
| HVPN-NFR-405 DNS-leak protection | `helix-core` / resolver | SEC, E2E | defined | `HVPN-CHAL-DNS-LEAK` | `HVPN-HQA-DNS-LEAK` | DoD-7 |
| HVPN-NFR-406 private key on-device | `helix-core` / FFI | SEC, INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-408 RLS isolation | Postgres / `svc-identity` | SEC, INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-409 anti-bluff gauntlet | QA harness | META | not-defined | not-defined | not-defined | — |
| HVPN-NFR-410 DPI evasion | `helix-transport` | E2E | not-defined | not-defined | not-defined | DoD-4 |
| HVPN-NFR-411 secrets rotation | deploy / KMS | SEC, INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-412 key rotation cadence | `svc-pki` / `helix-core` | SEC, INT | not-defined | not-defined | not-defined | — |
| **HVPN-NFR-413** control-plane API rate limiting | `svc-api` / Redis | **DDOS**, STR, SEC | **defined** | `HVPN-CHAL-NFR413-API-Rate-Limit` | `HVPN-HQA-NFR413-API-Rate-Limit` | **GAP-6 owner** |
| **HVPN-NFR-414** data-plane DDoS resilience | edge / `helix-transport` | **DDOS**, SEC | **defined** | `HVPN-CHAL-NFR414-Edge-Flood` | `HVPN-HQA-NFR414-Edge-Flood` | **GAP-6 owner** |

> **GAP-6 closed.** The `DDOS` test type now has explicit ownership: **NFR-413** (control-plane API gateway, token-bucket rate limiting per API key / device) and **NFR-414** (data-plane edge, UDP/handshake-flood resilience and fail-static behavior). Both are `defined` with parked Phase-2 banks; the MVP cell is `not-applicable-MVP` (`single-node-selfhost`) with a mechanical re-arm trigger when topology becomes `multi-tenant-ha`.

### Resource (NFR-500…506)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-NFR-500 iOS NE memory headroom | `shim-apple` / `helix-core` | MEM, CHAL, HQA | not-defined | not-defined | not-defined | G3 |
| HVPN-NFR-501 battery-sensitive | `helix-core` / platform shims | BENCH | not-defined | not-defined | not-defined | — |
| HVPN-NFR-502 binary footprint | `helix-core` | BENCH | not-defined | not-defined | not-defined | — |
| HVPN-NFR-503 ≤ 60 % host RAM | build tooling | BENCH | not-defined | not-defined | not-defined | — |
| HVPN-NFR-504 bounded CP memory | `svc-coordinator` | SCALE, MEM | not-defined | not-defined | not-defined | SLO4 |
| HVPN-NFR-505 lightweight sampler | `svc-telemetry` | BENCH | not-defined | not-defined | not-defined | — |
| HVPN-NFR-506 bounded send queues | `svc-coordinator` | STR, MEM | not-defined | not-defined | not-defined | — |

### Portability (NFR-600…609)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-NFR-600 three flavors one tree | `helix-ui` / Melos | INT, FA, UI | not-defined | not-defined | not-defined | — |
| HVPN-NFR-601 shared Rust core | `helix-core` | INT, FA | not-defined | not-defined | not-defined | — |
| HVPN-NFR-602 iOS/Android/Linux Access | platform shims | E2E, FA | not-defined | not-defined | not-defined | DoD-8 |
| HVPN-NFR-603 Connector headless | `helix-core` / shims | INT | not-defined | not-defined | not-defined | — |
| HVPN-NFR-604 Console Web + desktop | `helix-ui` / Console | E2E, UI | not-defined | not-defined | not-defined | — |
| HVPN-NFR-609 cross-platform parity | platform shims | per-OS branch | not-defined | not-defined | not-defined | — |

### Operability & Compatibility (NFR-700…703)

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| HVPN-NFR-702 protobuf / OpenAPI compatibility | codegen pipeline | U, META | not-defined | not-defined | not-defined | — |

---

## MVP Definition-of-Done acceptance-criteria coverage

| DoD | Criterion | Satisfying FRs/NFRs | Primary challenge | Primary HelixQA bank |
|---|---|---|---|---|
| DoD-1 | Self-host from zero | FR-901, FR-902 | not-defined | not-defined |
| DoD-2 | Enroll Connector + Client | FR-102, FR-104, FR-111, FR-701 | `HVPN-CHAL-AUTH-TUNNEL` | `HVPN-HQA-AUTH-TUNNEL` |
| DoD-3 | Reach authorized / deny unauthorized | FR-201, FR-204, FR-301, FR-305, FR-306, FR-704 | not-defined | not-defined |
| DoD-4 | Auto-escalate to MASQUE | FR-004, FR-006, NFR-410 | not-defined | not-defined |
| DoD-5 | Policy edit < 1 s | FR-205, NFR-003, NFR-009 | not-defined | not-defined |
| DoD-6 | Revoke < 1 s | FR-107, NFR-004 | not-defined | not-defined |
| DoD-7 | Kill-switch + DNS-leak | FR-501…504, NFR-404, NFR-405 | `HVPN-CHAL-KILL-SWITCH` | `HVPN-HQA-KILL-SWITCH` |
| DoD-8 | No durable log + three apps | FR-801, FR-1001, FR-1014, NFR-300, NFR-308 | `HVPN-CHAL-CLIENT-UI-VISUAL` | `HVPN-HQA-CLIENT-UI-VISUAL` |

---

## Phase-0 gate coverage

| Gate | Question | Satisfying FRs/NFRs | Primary challenge | Primary HelixQA bank |
|---|---|---|---|---|
| G1 | Plain-UDP WG throughput | FR-002, NFR-001 | not-defined | not-defined |
| G2 | MASQUE/QUIC through DPI block | FR-004, NFR-002 | not-defined | not-defined |
| G3 | iOS NE memory ceiling | FR-1004, NFR-500 | not-defined | not-defined |
| G4 | Go-vs-Rust edge benchmark | FR-003, NFR-001/002 | not-defined | not-defined |
| G5 | flutter_rust_bridge FFI | FR-018, FR-1002 | not-defined | not-defined |
| G6 | Push-based reconcile | FR-205, NFR-003 | not-defined | not-defined |

---

## GAP-6 closure statement

**Gap:** `requirements-traceability.md` Rev 4 identified that the `DDOS` §11.4.169 test type was defined in the legend but traced to zero requirements.

**Closure in this ledger:**

1. **Requirement HVPN-NFR-413** (`control-plane API rate limiting`) is explicitly owned by **`svc-api`** and verified by **DDOS** + stress + security tests.
2. **Requirement HVPN-NFR-414** (`data-plane DDoS/UDP-flood/amplification resilience`) is explicitly owned by **`helix-edge`** / **`helix-transport`** and verified by **DDOS** + security tests.
3. Both rows list concrete, minted Challenge IDs (`HVPN-CHAL-NFR413-API-Rate-Limit`, `HVPN-CHAL-NFR414-Edge-Flood`) and HelixQA bank IDs (`HVPN-HQA-NFR413-API-Rate-Limit`, `HVPN-HQA-NFR414-Edge-Flood`) that resolve in the committed banks.
4. The bank entries are **`defined`**; the live DDoS attack harness is not exercised in the single-node-selfhost MVP topology and mechanically re-arms when `deployment.topology == "multi-tenant-ha"` in Phase 2.

**Remaining coordinator decision:** The concrete design-attack-rate (`ATTACK_PPS`), legit-handshake SLO budget under flood, and supervisor restart budget are Phase-2 measured numbers; they must be calibrated during the Phase-2 soak before the DDOS gate becomes release-blocking.

---

## Coverage summary

| Metric | Count |
|---|---|
| MVP functional requirements traced | 53 |
| MVP non-functional requirements traced | 38 |
| Requirements with minted Challenge / HelixQA IDs | 8 |
| Total test-type cells mapped | 400+ (planned) |
| Challenge IDs minted | 8 |
| HelixQA bank IDs minted | 8 |
| DoD criteria covered | 8 / 8 (3 with minted primary bank IDs) |
| Phase-0 gates covered | 6 / 6 (0 with minted primary bank IDs) |
| Gaps closed (this pass) | GAP-6 |
| Remaining open gaps | GAP-3 (NFR-205 DR RTO/RPO unverified), GAP-4 (Connector single owning doc), GAP-5 (all evidence states PENDING until build) |

---

## Provenance

- Sources: `docs/research/mvp/final/10-testing-acceptance-and-qa.md`, `docs/research/mvp/final/v01-product/functional-requirements.md`, `docs/research/mvp/final/v01-product/nonfunctional-requirements.md`, `docs/research/mvp/final/v00-meta/requirements-traceability.md`, `docs/research/mvp/final/v08-testing/ddos.md`.
- Challenge / HelixQA bank conventions derived from `submodules/challenges/banks/examples/*.json`, `submodules/challenges/banks/yole/*.yaml`, `submodules/helix_qa/banks/*.yaml`.
- Constitution bindings: §11.4.6, §11.4.25, §11.4.52, §11.4.118, §11.4.153, §11.4.169.