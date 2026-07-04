# Phase 2 (Parity + Reach) Subtask Deepening — epic → task → subtask (closes R5)

**Revision:** 2
**Last modified:** 2026-07-04T12:00:00Z
**Rev 2:** Independent gap-analysis pass — structure and id-convention verified
consistent with `subtask-deepening-p1.md` and `08-phase2-parity-wbs.md`. No
contradictions found.

> Volume 7 (Phase Execution), document 4 of 5. Companion to
> `subtask-deepening-p1.md`, applying the same R5 deepening
> (`REFINEMENT_NOTES.md`) to Phase 2: every task `HVPN-P2-NNN` in
> `08-phase2-parity-wbs.md` is decomposed into PR-sized subtasks
> `HVPN-P2-NNN.k` (§11.4.93/.54), each with a concrete **acceptance** (falsifiable,
> captured-evidence §11.4.5/.69/.107), the **§11.4.169 test types** (the `08-…`
> §1 vocabulary, now including `DDOS` — re-armed for the public multi-region
> surface), and an **estimated complexity** (XS/S/M/L, sizing-only `TARGET`,
> §11.4.6). **Spec-only.** Phase 2's load-bearing premise — *everything is
> additive* (`08-…` §0) — means most subtasks are "more impls of an existing
> seam", and two epics (E21 DAITA, E24 PQ) carry the *measure-don't-assert*
> privacy-claim discipline (self-validated analyzers, §11.4.107(10)). Every
> subtask traces to its parent task in `08-…`; nothing is invented beyond
> decomposing stated work. These `.k` rows feed the `workable-items` DB
> (`workable-items-model.md` §7).

---

## Table of contents

- [0. Deepening conventions](#0-deepening-conventions)
- [1. E19 — Phase-1 seam certification (entry)](#1-e19--phase-1-seam-certification-entry)
- [2. E20 — Full transport / obfuscation set](#2-e20--full-transport--obfuscation-set)
- [3. E21 — DAITA traffic-analysis defense](#3-e21--daita-traffic-analysis-defense)
- [4. E22 — Direct P2P + NAT traversal + relay](#4-e22--direct-p2p--nat-traversal--relay)
- [5. E23 — Multi-hop (nested WireGuard)](#5-e23--multi-hop-nested-wireguard)
- [6. E24 — Post-quantum handshake (ML-KEM PSK)](#6-e24--post-quantum-handshake-ml-kem-psk)
- [7. E25 — Desktop apps (Windows, macOS)](#7-e25--desktop-apps-windows-macos)
- [8. E26 — Policy-as-code & GitOps](#8-e26--policy-as-code--gitops)
- [9. E27 — HA + multi-region fleet](#9-e27--ha--multi-region-fleet)
- [10. E28 — Observability additions](#10-e28--observability-additions)
- [11. E29 — QA · SLOs · DoD certification](#11-e29--qa--slos--dod-certification)
- [12. E30 — Governance & release](#12-e30--governance--release)
- [13. Subtask roll-up + measure-don't-assert note](#13-subtask-roll-up--measure-dont-assert-note)
- [Sources verified](#sources-verified)

---

## 0. Deepening conventions

Identical to `subtask-deepening-p1.md` §0: `id` (`HVPN-P2-NNN.k`) · **Subtask**
(≥6 words §11.4.91) · **Acceptance** (falsifiable, captured-evidence) · **Tests**
(§11.4.169 codes, `08-…` §1 vocabulary incl. `DDOS`) · **Cx** (XS/S/M/L sizing
`TARGET`). A subtask is `complete` only with a `test_diary` evidence path
(`workable-items-model.md` §9). **Two anti-bluff escalations specific to Phase 2**
(`08-…` §0): (a) any "direct path" claim is `Direct` only when a non-gateway src
is observed at the peer (§11.4.107 — no false direct); (b) DAITA + PQ privacy
claims MUST be *measured* with a self-validated analyzer (golden-bad fixture must
FAIL the analyzer), never asserted. Complexity sums to ≈ parent effort (`08-…`
§19, ~259 person-days excl. entry seams) — sizing only, never a date.

---

## 1. E19 — Phase-1 seam certification (entry)

Entry gate (not Phase-2 work; tracks that the reserved seams exist, `08-…` §5).

**HVPN-P2-190 — Transport-trait seam present** (XS) · **191 — `Peer.endpoint`
candidate-list seam** (XS) · **192 — federated Coordinator + PKI hooks** (XS).
Each is a single seam-audit subtask:

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `190.1` | Register a no-op stub transport + walk the ladder unchanged | the ladder walks the stub with zero changes to the trait | UNIT,INT | XS |
| `191.1` | Audit `NetworkMap.Peer.endpoint` is a `repeated EndpointCandidate` | the field is repeated; a coordinator delta can populate it | UNIT | XS |
| `192.1` | Audit coordinator statelessness + a PKI PSK-injection hook | coordinator hydrates from store+events; PKI exposes a PSK hook | UNIT,SEC | XS |

---

## 2. E20 — Full transport / obfuscation set

*Every new transport is just another impl of the unchanged Phase-1 `Transport` trait.*

**HVPN-P2-200 — Shadowsocks-wrap transport** (`08-…` §6; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Wrap WG datagrams in a Shadowsocks AEAD stream (reuse `shadowsocks-rust` primitives) | tunnel survives `udp drop`+`quic drop` nft rules (E2E) | UNIT,INT,E2E | M |
| `.2` | DPI-gauntlet classifier check | `tshark` sees no WG/QUIC signature (SEC) | SEC,CHAL | S |
| `.3` | Goodput record with TCP-HoL caveat | goodput captured + HoL noted (BENCH) | BENCH | S |

**HVPN-P2-201 — UDP-over-TCP (UoT) transport** (`08-…` §6; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Length-prefix WG over a single TCP conn; ladder ranks last | tunnel comes up under `udp drop` (all ports); reachability preserved (E2E) | UNIT,E2E | S |
| `.2` | HoL penalty measured + documented | HoL overhead captured (BENCH) | BENCH | XS |

**HVPN-P2-202 — LWO hardening (per-session keyed header obfuscation)** (`08-…` §6; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Per-session-keyed obfuscation of WG header signature bytes + randomized padding | a WG-signature DPI rule that drops plain-WG fails to classify LWO-v2 (SEC) | UNIT,SEC,CHAL | M |
| `.2` | Overhead < 2% vs plain-UDP | captured overhead < 2% (BENCH) | BENCH | XS |

**HVPN-P2-203 — Refined auto-escalation ladder + per-network memory** (`08-…` §6; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Ladder FSM with per-step failure budget (N handshakes/T seconds) | client converges to `shadowsocks` on a UDP+QUIC-blocked network (E2E) | UNIT,E2E | M |
| `.2` | `network_memory` per SSID/gateway fingerprint | on the next connect to the same network, escalation latency is skipped (captured timing delta) | FA,CHAL | M |

**HVPN-P2-204 — Coordinator-pushed regional priors** (`08-…` §6; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `TransportPolicy.region_prior` map in `NetworkMap`; region-keyed reorder | a censored-region-resolved client gets a reordered ladder + connects on the prior's head transport | UNIT,INT,E2E | S |

**HVPN-P2-205 — Aggregate censorship-evasion telemetry** (`08-…` §6; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Record aggregate "transport X after N escalations in region R" (no per-user data) + Grafana panel | dashboard renders transport-mix by region | UNIT,CHAL | S |
| `.2` | No-log schema-lint mutation: a per-user evasion table FAILs | the planted per-user table FAILs the build (SEC) | SEC | XS |

**HVPN-P2-206 — Hysteria2/Salamander interop evaluation (D1/D6)** (`08-…` §6; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | A/B benchmark Hysteria2+Salamander vs MASQUE/H3 (goodput under loss, DPI evasion parity) | recorded §20 decision-log row + CSV; rec: keep MASQUE primary, Hysteria2 optional rung | BENCH,E2E | S |

---

## 3. E21 — DAITA traffic-analysis defense

*Adopt `maybenot` (the engine behind Mullvad DAITA) — do not roll your own.*

**HVPN-P2-210 — maybenot engine integration** (`08-…` §7; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Embed `maybenot::Framework` as a shaping stage (`on_packet` pad, `tick` cover) over WG datagrams | outgoing datagrams pad to uniform sizes; cover packets emit on schedule (capture: size histogram collapses) | UNIT,INT | M |
| `.2` | Machines-as-config (data, not code) loading | a known padding machine loads + drives shaping | UNIT,BENCH | S |

**HVPN-P2-211 — Machines-as-data distribution via NetworkMap** (`08-…` §7; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `NetworkMap.daita` field carrying signed maybenot machine defs | a coordinator-pushed machine update changes shaping live (no rebuild) | UNIT,INT,E2E | S |
| `.2` | Signature-verify machine blobs before load | a tampered blob is rejected (captured) | SEC,INT | S |

**HVPN-P2-212 — DAITA opt-in toggle + honest cost surface** (`08-…` §7; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `DaitaToggle` widget (off by default) + Riverpod binding + live `cover_ratio` readout | UX MP4 shows toggling DAITA, the cost note, the live overhead (§11.4.159 vision-verified) | UI,UX,REC | S |

**HVPN-P2-213 — Closed-world fingerprinting efficacy harness (the privacy claim)** (`08-…` §7; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Closed-world WF classifier trained on captured traces; DAITA-off vs DAITA-on | DAITA-on accuracy < DAITA-off by the documented margin on the project's own fixtures | FA,SEC,BENCH | L |
| `.2` | Self-validated analyzer (golden-good reduces accuracy; golden-bad no-op machine does NOT, §11.4.107(10)) | the golden-bad no-op machine FAILs the analyzer (proves no-bluff) | CHAL,SEC | M |
| `.3` | Cover-traffic overhead recorded | overhead captured + surfaced honestly | BENCH | S |

---

## 4. E22 — Direct P2P + NAT traversal + relay

*The gateway becomes a coordinator + relay-of-last-resort; traffic goes direct where NAT allows.*

**HVPN-P2-220 — STUN-like endpoint discovery** (`08-…` §8; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Host + server-reflexive candidate discovery via the gateway STUN endpoint → `ReportStatus` | a node behind NAT reports a correct server-reflexive `ip:port` (INT) | UNIT,INT,E2E | M |
| `.2` | STUN endpoint refuses amplification (response ≤ request, rate-limited) | flood test: no amplification, rate-limit holds (DDOS) | DDOS | S |

**HVPN-P2-221 — Candidate distribution via WatchNetworkMap** (`08-…` §8; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Reuse `WatchNetworkMap` as the signaling channel: push each peer the other's candidates as a `Peer.endpoint` delta | both peers receive candidates within the SLO1 budget (captured) | UNIT,INT,E2E | S |
| `.2` | Candidates policy-filtered (need-to-know) | an unauthorized peer never appears in candidates (SEC) | INT,SEC | S |

**HVPN-P2-222 — UDP hole punching across the NAT-type matrix** (`08-…` §8; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Simultaneous WG handshake probes to all candidates; WG roaming latches the working src | direct datagrams captured bypassing the gateway for full/restricted/port-restricted pairs | UNIT,E2E,STRESS | L |
| `.2` | Symmetric-on-both-ends → clean relay fallback | symmetric pairs fall through to relay; no false "direct" claim (§11.4.107) | SEC,CHAL | M |

**HVPN-P2-223 — helix-relay (DERP-style encrypted relay fallback)** (`08-…` §8; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Edge relay mode keyed by destination pubkey (below the WG crypto boundary) | a symmetric-NAT pair communicates via relay (E2E) | UNIT,INT,E2E | L |
| `.2` | Relay holds only ciphertext, no per-flow durable record; tenant/region-fungible | memory/pcap inspection proves only pubkey-keyed ciphertext (no plaintext) | SEC,STRESS | M |
| `.3` | Relay forwarding throughput | captured forwarding BENCH | BENCH | S |

**HVPN-P2-224 — Path selection: start-relay, upgrade-to-direct** (`08-…` §8; M · SLO1)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Start on relay, attempt direct in background, upgrade seamlessly; report `path` | SLO1: direct-path establishment (non-symmetric NAT) > 90% within 10 s (histogram) | UNIT,E2E,PERF,FA | M |
| `.2` | No connectivity gap on upgrade | relay is the instant fallback; no user-visible reconnect | FA | S |

**HVPN-P2-225 — Continuous direct-path health-check + downgrade** (`08-…` §8; S · SLO2)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Health-loop: relay→direct upgrade < 15 s, no reconnect; direct→relay downgrade on degradation | SLO2 < 15 s upgrade (captured); injected degradation downgrades within the window | UNIT,E2E,PERF,CHAOS | S |

**HVPN-P2-226 — Path indicator UI + UX** (`08-…` §8; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `PathIndicator` bound to the core status stream | UX MP4 shows relay→direct on upgrade and direct→relay on injected degradation (§11.4.159) | UI,UX,REC | S |

---

## 5. E23 — Multi-hop (nested WireGuard)

*`Client → Entry → Exit` so no single node sees both who you are and where you go.*

**HVPN-P2-230 — Hop-chain computation in the coordinator** (`08-…` §9; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Compute the hop chain (respect `exitNodes` + user selection), per-hop keys, `hops` list in `NetworkMap` | Entry=EU/Exit=US yields a two-hop map with distinct per-hop keys (INT) | UNIT,INT | M |
| `.2` | Enforce `exitNodes` policy | an unauthorized exit is never offered (SEC) | SEC | S |

**HVPN-P2-231 — Nested-session construction in helix-core** (`08-…` §9; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Build nested WG (outer Client↔Entry obfuscated, inner Client↔Exit plain) — transport/DAITA on outer only | end-to-end reachability through Entry→Exit (E2E) | UNIT,E2E | M |
| `.2` | Second-hop overhead measured | goodput/latency overhead captured + documented | BENCH | S |

**HVPN-P2-232 — Hop-isolation capture assertions (the privacy proof)** (`08-…` §9; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Pcap at each hop: Entry sees client IP not dest; Exit sees dest not client IP | P2-AC4 both isolation invariants proven by pcap | SEC,E2E,CHAL | M |
| `.2` | Mutation: collapse to single hop FAILs the assertion | the single-hop-collapse mutation FAILs | CHAL,SEC | S |

**HVPN-P2-233 — Multi-hop picker UI** (`08-…` §9; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `MultiHopPicker` (region/jurisdiction labelled) bound to the policy-permitted hop set | UX MP4 shows selecting Entry/Exit + the two-hop connection with jurisdiction labels (§11.4.159) | UI,UX,REC | S |

---

## 6. E24 — Post-quantum handshake (ML-KEM PSK)

*Close harvest-now-decrypt-later via a PQ-derived PSK mixed into WG — without forking WG crypto.*

**HVPN-P2-240 — ML-KEM PSK exchange over the control channel** (`08-…` §10; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | ML-KEM (FIPS 203) exchange over the authenticated `Coordinator` RPC (no new listener) → HKDF → WG PSK | two PQ-capable peers establish a WG session whose PSK is ML-KEM-derived (captured) | UNIT,INT,SEC | L |
| `.2` | PSK rotates on each rekey; exchange at setup/rekey only | PSK rotation captured; no per-packet KEM | SEC,BENCH | M |

**HVPN-P2-241 — Hybrid combiner (never PQ-only) + capability negotiation** (`08-…` §10; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `PSK = HKDF(classical \|\| pq)` combiner + capability flags | PQ-capable↔classical-only pairs interoperate (negotiate down without error) | UNIT,SEC,E2E | M |
| `.2` | Mutation: PQ-only combine FAILs | a PQ-only combine FAILs the unit + mutation gate | UNIT,SEC | S |

**HVPN-P2-242 — Downgrade-safety guard** (`08-…` §10; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Downgrade FSM: a PQ failure → classical-or-fail-closed, never weaker | injecting a PQ failure yields classical-or-fail-closed, never weaker-than-classical (captured) | SEC,E2E,CHAL | M |
| `.2` | MITM strip-PQ detection logged as a control event | a strip-PQ attempt is detected/logged (not as traffic) | SEC | S |

**HVPN-P2-243 — PQ default-on where supported + adoption metric** (`08-…` §10; S · SLO5)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | PQ default-on for capable pairs + `PqChip` + `helix_pq_sessions_ratio` | SLO5: PQ adoption among capable pairs > 95% (dashboard) | UNIT,PERF,REC | S |
| `.2` | Zero steady-state per-packet cost | per-packet benchmark ≈ 0 delta vs classical | PERF | S |

**HVPN-P2-244 — Rosenpass alternative evaluation (decision)** (`08-…` §10; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Evaluate Rosenpass vs in-house ML-KEM (audit pedigree, integration cost) | recorded §20 decision row; rec: ship in-house ML-KEM, keep a Rosenpass `KemSuite` adapter | BENCH,SEC | S |

---

## 7. E25 — Desktop apps (Windows, macOS)

*Everything above the shim is the same `helix-ui` + `helix-core`.*

**HVPN-P2-250 — Windows privileged service + wireguard-nt shim** (`08-…` §11; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Windows service hosting `helix-core` over `wireguard-nt`/`wintun` + named-pipe IPC | connect → reach an authorized LAN host from Windows; app unprivileged, service holds tunnel | INT,E2E,REC | L |
| `.2` | Code-signed driver + service (Authenticode) | driver + service signature verified | SEC | M |

**HVPN-P2-251 — macOS NEPacketTunnelProvider shim** (`08-…` §11; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | macOS NE hosting `helix-core` (desktop memory budget), notarized + signed | connect → reach from macOS; notarization + signature verified; UX recording (§11.4.159) | INT,E2E,SEC,REC | M |

**HVPN-P2-252 — Desktop split-tunnel (per-app routing)** (`08-…` §11; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Windows WFP filters + macOS app-bound rules, include/exclude set from the shared UI | a designated app routes outside the tunnel while the rest stays inside (pcap proves the split per-app on both OSes) | UNIT,E2E,SEC | L |
| `.2` | Split-tunnel UX recording | UX MP4 of the split-tunnel screen (§11.4.159) | UX,REC | S |

**HVPN-P2-253 — Desktop kill-switch + DNS-leak parity** (`08-…` §11; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | WFP (Win) + NE rules (mac) kill-switch wired to core state | on forced drop, host-side pcap shows zero plaintext + zero DNS leak on both OSes (§11.4.107) | SEC,E2E,REC | S |

---

## 8. E26 — Policy-as-code & GitOps

*Git becomes the source of truth; the apply runner is the tenant's own runner (§11.4.156, no active server-side CI).*

**HVPN-P2-260 — `helixvpnctl policy compile --dry-run` effect-diff** (`08-…` §12; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Render the compiled *effect* delta (which devices gain/lose which access) | a PR adding one ACL line produces a human-readable effect-diff naming the exact devices/targets | UNIT,INT | M |
| `.2` | Mutation: a hidden granted edge FAILs the diff completeness check | the hidden-edge mutation FAILs | SEC,CHAL | S |

**HVPN-P2-261 — GitOps apply runner + tenant-token auth** (`08-…` §12; M · P2-AC7)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | On merge, tenant runner `helixvpnctl policy apply` (token-auth) → persist version → emit `policy.compiled` → converge | P2-AC7: a merged PR converges the fleet < 1 s (`helix_reconcile_seconds` p99) with the effect-diff reviewed | INT,E2E,FA | M |
| `.2` | No secrets in the repo (declarative intent only; sops/age refs) | secret-leak audit green (§11.4.10) | SEC | S |

**HVPN-P2-262 — GitOps round-trip safety (drift detect + rollback)** (`08-…` §12; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `helixvpnctl policy diff`/`rollback`; idempotent apply | an out-of-band change is flagged as drift; a revert PR restores the prior compiled state byte-identically (captured) | INT,CHAOS,SEC | S |

---

## 9. E27 — HA + multi-region fleet

*Same images, only topology changes (`08-…` §13/§17).*

**HVPN-P2-270 — Stateless coordinator fleet + stream re-pin** (`08-…` §13; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | N stateless coordinators per region; rebuildable cache from PG+NATS | coordinator holds no local durable state (§11.4.108 clean-restart) | UNIT,INT,SCALE | M |
| `.2` | Stream re-pin: agent re-pins to a sibling, resumes from `known_version` | killing a coordinator mid-stream re-pins + resumes with zero missed deltas (captured) | CHAOS | M |

**HVPN-P2-271 — Postgres HA (Patroni) + regional read replicas** (`08-…` §13; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Single logical primary + sync replicas + Patroni auto-failover | killing the PG primary fails over with no committed-write loss (CHAOS) | INT,CHAOS,SCALE | L |
| `.2` | Regional read replicas serving coordinator hydration; RLS/no-log on replicas | regional coordinators hydrate from local replicas; invariants hold (SEC) | SEC,SCALE | M |

**HVPN-P2-272 — NATS JetStream cross-region event mesh (Redis swap)** (`08-…` §13; L · SLO4)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `NatsJetStreamBus` impl behind the unchanged `EventBus` interface (durable subjects = Phase-1 taxonomy 1:1) | the taxonomy is identical to Phase 1 (no event renamed) | UNIT,INT | M |
| `.2` | Cross-region propagation p99 < 2 s | SLO4 cross-region p99 < 2 s (captured) | E2E,PERF,SCALE | M |
| `.3` | Region partition + heal shows no event loss | durable subjects replay; no loss (CHAOS) | CHAOS | M |

**HVPN-P2-273 — Edge/relay fleet + client steering (Anycast/GeoDNS + RTT list)** (`08-…` §13; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Anycast + GeoDNS fallback; `NetworkMap.gateways` candidate list; client picks by RTT | a client picks the nearest edge by measured RTT; Anycast routes UDP/QUIC to the closest PoP | INT,E2E,PERF | M |
| `.2` | STUN/edge handshake flood rate-limited | flood shed without dropping legit tunnels (DDOS) | DDOS | S |

**HVPN-P2-274 — Regional failover flow + floating IPs** (`08-…` §13; M · P2-AC8/SLO3)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Health prober → `gateway.failover{from,to}` → `NetworkMap` delta → client re-handshake; floating IPs | SLO3: gateway failover → client reconnected p99 < 3 s | E2E,CHAOS,PERF | M |
| `.2` | Existing direct-P2P sessions unaffected by region loss | P2-AC8: region kill reconnects clients < 3 s, zero policy/identity loss, direct sessions survive (chaos capture) | CHAOS | M |

---

## 10. E28 — Observability additions

*Counts + health only, never destinations/flows (`08-…` §14).*

**HVPN-P2-280 — Reach/evasion KPI dashboards** (`08-…` §14; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Transport-mix/region, escalation depth, direct-vs-relay ratio, hole-punch success by NAT-type, PQ adoption %, DAITA overhead % (aggregate) | dashboards render every KPI from aggregate counters | INT,CHAL | M |
| `.2` | No-log lint mutation: a per-flow source table FAILs | the per-flow mutation FAILs the no-log lint (§11.4 / §02 §2.4) | SEC | S |

**HVPN-P2-281 — Multi-region health (NATS lag, PG replication, convergence p99)** (`08-…` §14; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Per-region coordinator health + NATS consumer lag + PG replication lag + cross-region convergence-p99 panels + alerts | convergence-p99 panel reflects P2-SLO4 + alerts on breach; lag alerts fire under an injected partition (captured) | INT,PERF,SCALE | S |

---

## 11. E29 — QA · SLOs · DoD certification

*Each surface is a measure-don't-assert gate; privacy/efficacy claims carry self-validated analyzers (§11.4.107(10)).*

| Parent task | Subtask focus (`.1`/`.2`) | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `290` NAT-traversal matrix | `.1` full/restricted/port-restricted/symmetric harness | every theoretically-direct pair achieves direct (pcap bypass); symmetric relays cleanly; no false direct | E2E,STRESS,FA,CHAL | L |
| `291` obfuscation gauntlet | `.1` each transport through its DPI block + `tshark` classifier | P2-AC1: each transport survives its block + the classifier confirms the disguise | E2E,SEC,CHAL | M |
| `292` DAITA efficacy cert | `.1` accuracy-drop with the self-validated analyzer wired into meta-test | P2-AC2: DAITA-on accuracy drops materially; golden-bad no-op FAILs the analyzer; cost surfaced | FA,SEC,CHAL | M |
| `293` PQ interop + downgrade cert | `.1` classical↔PQ negotiation + downgrade safety | P2-AC5: all capability pairs interoperate; every injected PQ failure → classical-or-fail-closed (captured) | SEC,E2E,CHAL | M |
| `294` multi-hop isolation cert | `.1` per-hop pcap assertions | P2-AC4: both isolation invariants proven; single-hop-collapse mutation FAILs | SEC,E2E,CHAL | S |
| `295` HA/chaos cert | `.1` kill coordinator/PG primary/region | P2-AC8 + SLO3 (failover p99<3s) + SLO4 (cross-region p99<2s) green; no control-plane data loss; direct sessions survive | CHAOS,SCALE,PERF,FA | L |
| `296` DDoS resilience cert | `.1` STUN + edge handshake flood | STUN response ≤ request (no amplification); edge sheds a flood without dropping legit tunnels below SLO | DDOS,SEC,STRESS | M |
| `297` Phase-2 DoD cert + discovery sweep | `.1` full-suite retest + enumerated discovery (§11.4.118) | all 8 P2-AC + 5 P2-SLO green with evidence; the discovery set surfaces zero release-blockers (else §11.4.129) | FA,CHAL,REC | M |

---

## 12. E30 — Governance & release

**HVPN-P2-300 — Phase-2 items DB projection + docs-chain sync** (`08-…` §16; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Project the Phase-2 leaves into `docs/workable_items.db` via the reused Phase-1 loader + `.docs_chain/contexts/wbs-p2.yaml` | md↔db byte-identical round-trip (§11.4.93); a leaf edit re-syncs exports; `verify` is the gate | UNIT,INT,FA | M |

**HVPN-P2-301 — Phase-2 release tagging + multi-upstream publish** (`08-…` §16; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Cut `<HELIX_RELEASE_PREFIX>-2.0.0-parity` (§11.4.151) on main + every owned submodule; publish via merge-onto-latest-main (no force-push §11.4.113) | `git tag -l '<PREFIX>-2.*'` enumerates the whole release; HVPN-P2-297 cert is the tag's evidence; reachable on every remote | FA | S |

---

## 13. Subtask roll-up + measure-don't-assert note

| Epic | Parent tasks | Deepened subtasks | Risk note |
|---|---|---|---|
| E19 entry seams | 3 | 3 | prereq audits |
| E20 transport set | 7 | 15 | additive impls of one trait |
| E21 DAITA | 4 | 8 | **measure-don't-assert** |
| E22 P2P/NAT | 7 | 14 | **no-false-direct (§11.4.107)** |
| E23 multi-hop | 4 | 7 | isolation proven by pcap |
| E24 PQ | 5 | 9 | **never-weaker-than-classical** |
| E25 desktop | 4 | 6 | per-app split-tunnel |
| E26 GitOps | 3 | 5 | effect-diff completeness |
| E27 HA/multi-region | 5 | 11 | failover SLOs |
| E28 observability | 2 | 4 | aggregate-only, no-log holds |
| E29 QA/DoD | 8 | 8 | self-validated analyzers |
| E30 governance | 2 | 2 | items-DB + tag |
| **Total** | **54** | **92** | — |

The two privacy-claim epics (E21 DAITA, E24 PQ) and the NAT-traversal epic (E22)
carry the load-bearing anti-bluff escalations (`08-…` §0/§19): a green pass that
*asserts* a privacy claim without the measurement is a §11.4 PASS-bluff; a
"direct" path claimed without a non-gateway src observed at the peer is the same
class. Every subtask's acceptance is a captured-evidence assertion
(§11.4.5/.69/.107); the golden-bad fixture FAILing the analyzer is the proof the
harness cannot bluff (§11.4.107(10)). Complexity sums to ≈ `08-…` §19's ~259
person-days — sizing only, never a date (§11.4.6).

---

## Sources verified

- `08-phase2-parity-wbs.md` §5–§16 (every task Desc/Deliverable/Acceptance/Effort/Tests), §17 HA topology, §18 traceability, §19 critical path, §20 open decisions — read 2026-06-26.
- Sibling `workable-items-model.md` (§3/§6/§9), `dependency-graph.md` (§5 Phase-2 DAG), `subtask-deepening-p1.md` (§0 conventions) — authored this volume.
- Constitution anchors §11.4.5/.6/.10/.40/.54/.58/.69/.85/.91/.93/.107/.108/.113/.118/.129/.132/.151/.156/.159/.169 — read 2026-06-26.

> Honest boundary (§11.4.6): subtasks decompose stated Phase-2 work. DAITA/PQ
> efficacy subtasks are *measure-don't-assert* — their acceptance is a measured
> delta with a self-validated analyzer, never an asserted claim. Complexity is
> sizing `TARGET` only; `DDOS` subtasks are now applicable (public multi-region
> surface), the Phase-1 `NOT_APPLICABLE` deferral re-armed.
