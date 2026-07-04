# Phase 1 (MVP) Subtask Deepening — epic → task → subtask (closes refinement R5)

**Revision:** 2
**Last modified:** 2026-07-04T12:00:00Z
**Rev 2:** Independent gap-analysis pass — spot-checked subtask acceptance criteria
against `07-phase1-mvp-wbs.md` parent tasks (E01/E02/E03 sampled in depth); all
carry falsifiable, captured-evidence acceptance and a §11.4.169 test-type set. No
placeholder/TBD text found. Confirmed this document is what closes R5 for Phase 1
(the parent WBS itself stays at epic→task; the third tier lives here, as designed).

> Volume 7 (Phase Execution), document 3 of 5. This spec **closes refinement
> item R5** (`REFINEMENT_NOTES.md`): Phase 0 (`06-…`) formalizes a full
> epic→task→subtask 3-tier breakdown, but Phase 1 (`07-phase1-mvp-wbs.md`) stops
> at epic→task. Here every Phase-1 task `HVPN-P1-NNN` is decomposed into
> PR-sized subtasks `HVPN-P1-NNN.k` (§11.4.93/.54 ids), each carrying a concrete
> **acceptance criterion** (falsifiable, captured-evidence per §11.4.5/.69/.107),
> the **§11.4.169 test types** that gate it, and an **estimated complexity**
> (XS/S/M/L T-shirt, sizing-only `TARGET` — never a date, §11.4.6). It is
> **spec-only**: it describes the subtasks and how "done" is proven; it does not
> build them. Every subtask traces to its parent task's Desc/Deliverable/
> Acceptance in `07-…`; nothing is invented beyond decomposing the stated work.
> These `.k` rows feed the `workable-items` DB (`workable-items-model.md` §7
> docs_chain context). Companion: `dependency-graph.md` (the DAG these subtasks
> inherit), `subtask-deepening-p2.md`, `subtask-deepening-p3.md`.

---

## Table of contents

- [0. Deepening conventions](#0-deepening-conventions)
- [1. E01 — Foundation (repo, proto, codegen)](#1-e01--foundation-repo-proto-codegen)
- [2. E02 — Store, RLS, no-log lint](#2-e02--store-rls-no-log-lint)
- [3. E03 — Identity, enrollment, PKI](#3-e03--identity-enrollment-pki)
- [4. E04 — IPAM](#4-e04--ipam)
- [5. E05 — Event backbone](#5-e05--event-backbone)
- [6. E06 — Policy model & compiler](#6-e06--policy-model--compiler)
- [7. E07 — Coordinator & WatchNetworkMap](#7-e07--coordinator--watchnetworkmap)
- [8. E08 — API surface + authz](#8-e08--api-surface--authz)
- [9. E09 — helix-core (Rust client/connector)](#9-e09--helix-core-rust-clientconnector)
- [10. E10 — helix-edge (data plane)](#10-e10--helix-edge-data-plane)
- [11. E11 — helix-ui (Flutter)](#11-e11--helix-ui-flutter)
- [12. E12 — Platform tunnel shims](#12-e12--platform-tunnel-shims)
- [13. E13 — Deploy](#13-e13--deploy)
- [14. E14 — QA, SLOs, DoD certification](#14-e14--qa-slos-dod-certification)
- [15. E15 — Governance & release](#15-e15--governance--release)
- [16. Subtask roll-up + coverage note](#16-subtask-roll-up--coverage-note)
- [Sources verified](#sources-verified)

---

## 0. Deepening conventions

Each parent task gets a subtask table: `id` (`HVPN-P1-NNN.k`, monotonic per
parent, §11.4.54) · **Subtask** (≥6-word title, §11.4.91) · **Acceptance**
(falsifiable, captured-evidence) · **Tests** (§11.4.169 codes — `UNIT INT E2E FA
SEC CHAOS STRESS PERF BENCH SCALE UI UX REC CHAL`, the `07-…` §1 vocabulary) ·
**Cx** (complexity XS≈0.5d, S≈1–2d, M≈2–3d, L≈3–5d — sizing `TARGET`).

A subtask is `complete` only when its test types are green with a `test_diary`
evidence path (`workable-items-model.md` §9). Subtask complexity sums to ≈ the
parent task's `07-…` Effort; it is a sizing aid, not a commitment (§11.4.6,
`07-…` §22). Phase-0 entry-gate items (`HVPN-P1-001..006`) are prerequisites
tracked in `06-…`; they are not re-deepened here.

---

## 1. E01 — Foundation (repo, proto, codegen)

**HVPN-P1-010 — Monorepo + submodule layout** (`07-…` §6; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Scaffold the snake_case component tree (`helix-core/ helix-edge/ helix-go/ helix-proto/ helix-ui/ shims/ deploy/`) | each dir present + decoupled (§11.4.28/.29); `tree` capture matches [SYNTHESIS §6] | UNIT,FA | S |
| `.2` | Wire `.gitmodules` + per-repo `helix-deps.yaml` (§11.4.31) for the six incorporated submodules | `git submodule status` lists containers/helix_qa/challenges/docs_chain/security/vision_engine | UNIT | XS |
| `.3` | `upstreams/` recipes + `install_upstreams` per repo (§11.4.36) | `git remote -v \| grep -c push` = expected count on a fresh clone | FA | XS |
| `.4` | Assert **no active CI** (§11.4.156): workflows disabled/renamed | `CM-NO-ACTIVE-CI` green; `git ls-files` finds no `.github/workflows/*.yml` | FA,SEC | XS |

**HVPN-P1-011 — `helix-proto` agent protobuf + buf codegen** (`07-…` §6; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Author `agent.proto` (`Coordinator`: Enroll/WatchNetworkMap/AdvertisePrefixes/ReportStatus) per [02 §4] | `buf lint` clean; service matches the [02 §4] contract | UNIT | S |
| `.2` | Wire `buf generate` → Go + Dart + Rust stubs | one source → three generated clients compile | UNIT,FA | S |
| `.3` | `buf breaking` gate on incompatible change | a removed field FAILs `buf breaking` (captured) | FA | XS |
| `.4` | Assert no hand-written client code exists (§8 [04_P1]) | grep finds zero hand-rolled RPC client; generated-only | SEC | XS |

**HVPN-P1-012 — REST OpenAPI + Dart/TS client gen** (`07-…` §6; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Define `openapi/helix.v1.yaml` (enroll-tokens/devices/policies/networks/stream) | OpenAPI 3.1 validates | UNIT | S |
| `.2` | Generate Dart + TS clients; round-trip a stub server | generated client round-trips; spec↔handler drift FAILs the gen check | UNIT,FA | S |

**HVPN-P1-013 — Local dev infra via `containers` submodule** (`07-…` §6; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `internal/testinfra` wrapper booting PG+Redis via `pkg/boot`/`pkg/compose`/`pkg/health` (§11.4.76/.161 rootless) | integration tests boot infra on demand; no ad-hoc docker | INT,FA | S |
| `.2` | `make test-infra-up/down` with orphan-free teardown (§11.4.14) | teardown leaves zero orphan containers (captured `podman ps`) | INT,FA | XS |

---

## 2. E02 — Store, RLS, no-log lint

*Risk-first (§11.4.132): the irreversible correctness floor — ships before any feature reads/writes data.*

**HVPN-P1-020 — Migrations, enums, DB roles** (`07-…` §7; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `goose` DDL: 11 tenant tables + enums per [02 §2.2] | migrations apply+rollback clean on fresh PG; `\dt` matches spec | UNIT,INT | M |
| `.2` | Three roles `helix_owner`/`helix_app`(non-super,non-bypass)/`helix_sys` | role grants verified; `helix_app` lacks rolsuper/rolbypassrls | INT,SEC | S |
| `.3` | Assert no connection/traffic table present (privacy floor) | `\dt` shows zero `connections\|flows\|traffic\|packets` table | SEC | XS |

**HVPN-P1-021 — `sqlc` typed queries + `store.WithTenant`** (`07-…` §7; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `sqlc`-generated compile-time-checked queries | generated `Queries` compiles; `sqlc vet` clean | UNIT | S |
| `.2` | `WithTenant(ctx,tid,fn)` tx-scoped `SET LOCAL app.tenant_id` + `WithSystem` | a query forgetting `WHERE tenant_id` still returns only the active tenant's rows (INT) | UNIT,INT | M |

**HVPN-P1-022 — FORCE RLS + tenant_isolation policies** (`07-…` §7; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `ENABLE`+`FORCE RLS` + `tenant_isolation USING/WITH CHECK` on all 10 tenant tables | every tenant table has the policy; `\d+` capture | UNIT,INT | M |
| `.2` | `store/rls_guard.go` startup abort if `helix_app` can bypass RLS | startup aborts under a super role (captured exit) | UNIT,SEC | S |
| `.3` | Cross-tenant denial proof | a crafted tenant-A→tenant-B read returns ZERO rows (captured) | SEC | S |
| `.4` | Mid-tx drop rollback (no leak) | connection drop mid-tx rolls back; no partial cross-tenant write | CHAOS | S |

**HVPN-P1-023 — No-logging CI schema-lint + mutation** (`07-…` §7; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `tools/schemalint` rejecting connection-log-shaped tables [02 §2.4] | green on the real schema; pre-build gate wired | UNIT,SEC | S |
| `.2` | Paired §1.1 mutation: inject `connections(src,dst,bytes,ts)` → lint FAILs | the planted table FAILs the lint (captured) | SEC | XS |

---

## 3. E03 — Identity, enrollment, PKI

**HVPN-P1-030 — Dual identity: OIDC + anonymous device tokens** (`07-…` §8; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | OIDC managed mode (Keycloak/Authentik) → tenant/role mapping | an OIDC user maps to a tenant/role (INT) | UNIT,INT | M |
| `.2` | Anonymous device enroll-tokens (no PII, Mullvad posture) | an anonymous device enrolls with zero PII persisted (SEC asserts DB/logs) | UNIT,SEC | S |
| `.3` | Token mint/store-hashed/consume-atomic | single-use, short-lived; concurrent double-consume rejected (SEC) | SEC,STRESS | S |

**HVPN-P1-031 — `Enroll` RPC (device key never leaves)** (`07-…` §8; M · SLO2)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Token verify → IPAM-allocate → issue cert → persist device row (public WG key only) | device row holds only the WG public key (SEC asserts no private key) | UNIT,INT,SEC | M |
| `.2` | Emit `device.enrolled`; first NetworkMap path | event well-formed; map issued | INT,E2E | S |
| `.3` | SLO2 timing: enroll → first NetworkMap < 2 s | captured histogram p99 < 2 s | E2E,PERF | S |

**HVPN-P1-032 — PKI: device-cert issuance + tenant CA** (`07-…` §8; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `pki.Issue/Rotate/Revoke` short-lived (24 h) mTLS certs vs per-tenant CA | a cert authenticates `WatchNetworkMap` (INT) | UNIT,INT | M |
| `.2` | CA-root secrecy (KMS/offline) + secret-leak audit (§11.4.10.A) | CA private key never in any app log (audit green) | SEC | S |

**HVPN-P1-033 — Cert rotation + revoke-<1 s** (`07-…` §8; M · SLO3, AC6)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Auto-rotate before expiry over the existing channel | rotation at the boundary works (captured) | UNIT,INT | S |
| `.2` | `device.revoke` → drop WG peer from every map + edge enforce + cert revoked | revoked cert rejected immediately (SEC) | SEC | M |
| `.3` | SLO3 timing: revoke → edge enforcement < 1 s | captured revoke→enforcement timing p99 < 1 s | PERF | S |

---

## 4. E04 — IPAM

**HVPN-P1-040 — ULA `/48` provisioning + `AllocOverlayIP`** (`07-…` §9; M, D4)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Provision `fd7a:helix:<rand>::/48` per tenant (`::1`=gateway) on tenant create | each tenant gets a distinct ULA /48 (INT) | UNIT,INT | S |
| `.2` | `AllocOverlayIP` deterministic + concurrency-safe | 1000 concurrent allocations: no duplicate/gap (STRESS, captured) | UNIT,STRESS | M |
| `.3` | Restart-stable re-hydration of `next_host` | allocation resumes after restart with no reuse | INT | S |

**HVPN-P1-041 — `4via6` for colliding IPv4 LANs** (`07-…` §9; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Stable site-id allocator per connector site | each site gets a stable id (INT) | UNIT,INT | S |
| `.2` | `4via6` route synthesis → `Peer.allowed_ips` | one client reaches the same IPv4 CIDR via two connectors, disambiguated by site (E2E curl each) | E2E | M |

---

## 5. E05 — Event backbone

**HVPN-P1-050 — `events.Bus` Redis Streams + envelope** (`07-…` §10; M, D3)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `events.Bus` interface + Redis Streams `XADD`/`XReadGroup`/`XACK` + §5.2 envelope | at-least-once delivery; idempotent reaction; replay harmless (INT) | UNIT,INT | M |
| `.2` | Consumer-crash re-delivery on restart | crash mid-handle re-delivers (CHAOS, captured) | CHAOS | S |

**HVPN-P1-051 — DLQ sweeper (`XAUTOCLAIM`) + metric** (`07-…` §10; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `XAUTOCLAIM` reclaim + dead-letter routing + `helix_events_dlq_total` | poisoned event lands in DLQ after N retries; metric increments (captured) | UNIT,INT,CHAOS | S |

**HVPN-P1-052 — Event taxonomy producers** (`07-…` §10; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Typed constructors for the [04_P1 §5.3] taxonomy (device.*, connector.*, route.conflict, policy.*, gateway.failover) | each producer emits a well-formed envelope on its trigger (INT capture) | UNIT,INT | S |

---

## 6. E06 — Policy model & compiler

**HVPN-P1-060 — Spec parse + group/host resolution** (`07-…` §11; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Parse `policies.spec` jsonb (groups/hosts/acls/exitNodes) | valid spec parses; malformed rejected | UNIT | S |
| `.2` | Resolve `group:*`→device-ids + `host`→CIDRs vs `advertised_prefixes` | unknown group/host rejected; ambiguity emits `route.conflict.detected` | UNIT,INT | M |
| `.3` | Reject an ACL granting a revoked device | revoked-device grant rejected (SEC) | SEC | S |

**HVPN-P1-061 — Two-artifact compiler (AllowedIPs + verdict map)** (`07-…` §11; L · AC2/AC3)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Pure default-deny `compile(tenant,spec)→{visible,allowedIPs,exitNodes}` | golden-table tests pass; deterministic | UNIT | M |
| `.2` | Port-level verdict map (nftables/eBPF granularity) | a contractor granted `554,80` cannot reach port 22 (verdict-map SEC) | UNIT,SEC | M |
| `.3` | Determinism property: same spec ⇒ byte-identical output | property test green over N runs (§11.4.50) | UNIT | S |
| `.4` | Perf: compile 1k-device tenant < 100 ms | captured timing < 100 ms | PERF | S |

**HVPN-P1-062 — Dry-run fail-closed validation + atomic activate/rollback** (`07-…` §11; M · AC5)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Dry-run compile rejects unknown ref / uncovered dst / revoked grant | a bad policy never reaches the active version (fail-closed, captured) | UNIT,SEC | M |
| `.2` | Atomic version bump + `policy.compiled` emit + instant rollback | rollback restores the prior compiled set atomically (INT) | INT | S |

---

## 7. E07 — Coordinator & WatchNetworkMap

*Critical path; the < 1 s convergence promise lives here.*

**HVPN-P1-070 — In-memory per-tenant topology graph** (`07-…` §12; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Hydrate per-tenant graph from PG on boot (no durable coordinator tables) | graph reconstructable from PG + events alone (INT) | UNIT,INT | M |
| `.2` | Event-driven incremental subgraph mutation | applying an event mutates only the affected subgraph (INT) | INT,SCALE | M |

**HVPN-P1-071 — `buildMap(node)` per-node map** (`07-…` §12; M · AC2/AC3)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `buildMap` → `NetworkMap{self,gateway,peers,dns,transport}` with `PeersVisibleTo` filter | a node never appears in another's map unless policy grants it (SEC, captured) | UNIT,SEC | M |
| `.2` | `relayEndpoint` via gateway (MVP hub-and-spoke) | reachability matches the Phase-0 slice | UNIT | S |

**HVPN-P1-072 — `WatchNetworkMap` stream loop** (`07-…` §12; L · AC5)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | mTLS-auth → snapshot/resume-from-`known_version` → delta+keepalive loop | reconnect with `known_version=N` resumes from deltas, no full resync (captured) | UNIT,INT,E2E | L |
| `.2` | Bounded per-stream send queue (slow-consumer shed) | a stalled consumer is shed without blocking others (CHAOS) | CHAOS | M |

**HVPN-P1-073 — Minimal-affected-set fan-out + reconcile metric** (`07-…` §12; M · AC5/SLO1)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Compute minimal affected open-stream set; per-stream deltas (no full resync) | a 1-device change touches only the affected streams (INT count assertion) | UNIT,INT | M |
| `.2` | `helix_reconcile_seconds` histogram: event → delta-on-wire | SLO1 p99 < 1 s (captured histogram) | PERF | S |

**HVPN-P1-074 — 24 h soak / 10k-stream memory bound** (`07-…` §12; M · SLO4)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | 10k simulated agents holding streams while flapping policies | SLO1 maintained under load (captured) | SCALE,STRESS | M |
| `.2` | 24 h RSS time-series, bounded no-leak | coordinator memory bounded, no growth over 24 h (SLO4 graph) | PERF,SCALE | M |

---

## 8. E08 — API surface + authz

**HVPN-P1-080 — Connect handlers on Gin + mTLS resolver** (`07-…` §13; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | One multiplexed server: Connect (HTTP/2) alongside Gin REST | native agent + gRPC-Web caller both reach `Coordinator` | UNIT,INT | M |
| `.2` | `authDevice(ctx)` resolves identity from mTLS cert | an unauthenticated agent RPC is rejected (SEC) | SEC | S |

**HVPN-P1-081 — REST routes + OpenAPI + RBAC** (`07-…` §13; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Gin routes (enroll-tokens/devices/policies/networks/audit) | routes match OpenAPI (INT) | UNIT,INT | M |
| `.2` | OIDC/session/API-token + RBAC (admin/operator/member) | a `member` cannot mint enroll-tokens (403, SEC) | SEC | S |
| `.3` | RLS backstop (defense-in-depth) | RLS blocks cross-tenant read even if RBAC bypassed (INT) | INT,SEC | S |

**HVPN-P1-082 — `GET /v1/stream` WS/SSE fan-out** (`07-…` §13; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Stream control-plane events to Console over WS/SSE | a policy apply surfaces "policy v N active" to a connected Console within the convergence window (E2E) | UNIT,INT,E2E | S |

**HVPN-P1-083 — `AdvertisePrefixes` + `ReportStatus` RPCs** (`07-…` §13; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `AdvertisePrefixes` accepts connector CIDRs → `connector.prefixes.changed` | advertised prefixes flow into the coordinator graph (INT) | UNIT,INT | S |
| `.2` | `ReportStatus` carries presence/transport/rtt only (no bytes/flows) | SEC asserts the message has no byte/flow/destination field | SEC | S |

---

## 9. E09 — helix-core (Rust client/connector)

*The shared client/connector/edge crate; durable evolution of the Phase-0 interfaces.*

**HVPN-P1-090 — `Transport` trait + plain-UDP + MASQUE** (`07-…` §14; L · AC4, D1)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Harden the trait (`send/recv/kind/effective_mtu`, unreliable datagrams) into `helix-transport` | same trait drives client/connector/edge byte-for-byte | UNIT | M |
| `.2` | Production `plain_udp` impl | throughput baseline captured (BENCH) | UNIT,BENCH | S |
| `.3` | Production `masque` (quinn+h3+hand-rolled CONNECT-UDP/HTTP-Datagram, RFC 9298/9297/9221) | MASQUE flow classifies HTTP/3, no WG signature (tshark, SEC) | UNIT,SEC,E2E | L |
| `.4` | MASQUE goodput vs plain ≥ 50% | captured BENCH ≥ 50% of plain | BENCH | S |

**HVPN-P1-091 — `helix-wg` boringtun wrapper** (`07-…` §14; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Wrap `Tunn` (handshake/encrypt/decrypt/4 verdicts) | in-process handshake + data round-trip identical plaintext | UNIT | M |
| `.2` | Sustained 1 GB transfer + 2-min rekey | no handshake stall (STRESS); rekey at boundary works (captured) | E2E,STRESS | M |

**HVPN-P1-092 — Orchestrator + network-map reconciler** (`07-…` §14; L · AC5)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Three-loop orchestrator (tun→wg→transport, transport→wg→tun, timer) | loops drive a plain-UDP slice (E2E) | UNIT,E2E | M |
| `.2` | Reconciler consuming streamed `MapUpdate` (snapshot/delta) without restart | a delta adds/removes a peer with no reconnect of unrelated state (FA) | FA | M |
| `.3` | Mid-stream coordinator-restart re-sync from `known_version` | re-syncs cleanly (CHAOS) | CHAOS | S |

**HVPN-P1-093 — Auto-escalation ladder** (`07-…` §14; M · AC4)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Walk `TransportPolicy.order` (plain-udp→lwo→masque-h3) on handshake failure, honor `allow_user_override` | with plain WG blocked, client escalates to MASQUE and stays up (E2E `transport=="masque-h3"`) | UNIT,E2E,SEC | M |
| `.2` | Deterministic ladder across N runs (§11.4.50) | identical escalation outcome over N=3 | UNIT | S |

**HVPN-P1-094 — Kill-switch + DNS-leak protection** (`07-…` §14; L · AC7)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Core state machine drives platform firewall on Down/Reconnecting | host-side pcap during a forced drop shows ZERO leaked packets (SEC) | UNIT,SEC | L |
| `.2` | DNS pinned to overlay resolver while up; no plaintext DNS on drop | zero plaintext DNS leaked (SEC capture) | SEC | M |
| `.3` | State survives an abrupt interface flap | kill-switch holds across flap (CHAOS) | CHAOS | S |

**HVPN-P1-095 — `helix-ffi` (flutter_rust_bridge v2) surface** (`07-…` §14; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `start(ClientConfig)`/`stop()` + codegen | Dart drives connect/disconnect; codegen clean | UNIT,FA | S |
| `.2` | `status_stream(StreamSink<TunnelStatus>)` mirroring the broadcast channel | Dart renders live status from the stream alone (UI golden + FA ×3) | UNIT,UI,FA | M |

---

## 10. E10 — helix-edge (data plane)

*Language fixed by Phase-0 G4 (default lean Rust, shares `helix-transport`).*

**HVPN-P1-100 — MASQUE termination on `:443/udp` + masquerade** (`07-…` §15; L · AC4)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Terminate MASQUE/H3, extract HTTP Datagrams → WG → kernel WG | a valid client tunnels through (E2E, captured) | UNIT,E2E | L |
| `.2` | Native decoy site for non-CONNECT-UDP probes | a scanner at `:443` sees a plain website (SEC) | SEC | M |
| `.3` | CPU/Gbps within the G4 budget | captured BENCH within budget | BENCH | S |

**HVPN-P1-101 — Kernel-WG peer programming + relay** (`07-…` §15; M · AC2)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Program kernel WG peers from the coordinator map | peers appear in `wg show` (INT) | UNIT,INT | M |
| `.2` | Hub-and-spoke relay peer↔peer via gateway | client reaches an authorized connector LAN host through the relay (E2E curl) | E2E | M |

**HVPN-P1-102 — Edge verdict-map enforcement + revoke drop** (`07-…` §15; M · AC3/AC6, SLO3)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Apply the compiler's port-level verdict map (nftables/eBPF) | an unauthorized port is dropped (AC3 SEC, captured) | UNIT,INT,SEC | M |
| `.2` | Drop a revoked device's sessions on `device.revoked` | revoke removes the kernel peer < 1 s (SLO3, captured); negligible p99 add (PERF) | PERF,SEC | S |

**HVPN-P1-103 — Edge hardening posture** (`07-…` §15; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Rootless Podman, read-only rootfs, seccomp, `NET_ADMIN`-only, no SSH | container runs unprivileged except `NET_ADMIN` (SEC) | SEC | S |
| `.2` | Fail-static: killing control plane doesn't drop data-plane tunnels | established tunnels survive a control-plane kill (CHAOS capture) | CHAOS | S |

---

## 11. E11 — helix-ui (Flutter)

*All UI under OpenDesign tokens (§11.4.162) — light+dark, no ad-hoc CSS.*

**HVPN-P1-110 — `helix_design` system (OpenDesign tokens, Material 3)** (`07-…` §16; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Brand-tokenized palette + signature components (ConnectButton/StatusChip/ExitPicker/ShieldIndicator/AdaptiveScaffold), light+dark | visual-regression goldens pass for every component in both themes (UI) | UNIT,UI | M |
| `.2` | No overlapping/overlaid labels (§11.4.162 layout invariant) | layout-audit golden clean; palette reacts to connection state (REC vision-verified §11.4.159) | UI,REC | S |

**HVPN-P1-111 — Shared shell `runHelixApp(flavor,…)` + Riverpod binding** (`07-…` §16; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | One tree → three flavors (Access/Connector/Console) | same widget tree renders all three flavors (UI) | UNIT,UI | M |
| `.2` | Riverpod providers = pure function of the FFI status stream | UI state deterministic over a replayed status stream (FA) | FA | S |

**HVPN-P1-112 — Access app screens (enroll → connect → reach)** (`07-…` §16; L · AC9)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Enroll (paste/scan token) → connect → ExitPicker/authorized-network list | golden tests for each screen (UI) | UI | M |
| `.2` | Real enroll→connect→reach-LAN journey window-scoped MP4 (§11.4.143/.159) | vision verdict confirms the authorized host loaded (UX/REC) | UX,E2E,REC | M |

**HVPN-P1-113 — Console (web) admin screens** (`07-…` §16; L · AC5/AC9)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Devices/enroll-tokens/policy editor (validate→activate→rollback) + live `/v1/stream` feed | a policy edit reflects on devices < 1 s with a live "policy v N active" feed (E2E/REC, AC5) | UI,E2E,REC | L |
| `.2` | RBAC hides admin actions from `member` | member sees no admin action (SEC) | SEC | S |

**HVPN-P1-114 — Connector daemon UI/headless control** (`07-…` §16; M · AC9)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Enroll + advertise CIDRs + show prefixes/reachability; headless daemon mode | a connector advertises a LAN; its prefixes appear in Console + a client's map (E2E) | UI,E2E,REC | M |

---

## 12. E12 — Platform tunnel shims

*MVP platforms only: iOS, Android, Linux. The shim is the only platform-specific code.*

**HVPN-P1-120 — Linux tun shim (Rust-native)** (`07-…` §17; S · AC1/AC2)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Linux core drives the TUN directly (reference for the netns rig) | the Access Linux build runs the full enroll→connect→reach journey on the netns rig (E2E) | E2E,FA | S |

**HVPN-P1-121 — Android `VpnService` + JNI shim** (`07-…` §17; M · AC7/AC9)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `VpnService` owns the TUN; JNI loads `helix-core.so` | tunnel up on a real phone (device REC) | E2E,UX,REC | M |
| `.2` | Kill-switch → `VpnService` always-on/lockdown | kill-switch blocks plaintext on drop (SEC capture) | SEC | S |

**HVPN-P1-122 — iOS `NEPacketTunnelProvider` shim (within G3 budget)** (`07-…` §17; L · AC7/AC9)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `NEPacketTunnelProvider` configures settings + pumps `packetFlow` ↔ Rust core | tunnel up on a real device (device REC) | E2E,UX,REC | L |
| `.2` | Honor the G3 memory ceiling over a sustained transfer | captured Instruments RSS within ceiling (PERF) | PERF | M |
| `.3` | Kill-switch + DNS-leak hold | zero plaintext/DNS leak on drop (SEC) | SEC | S |

---

## 13. E13 — Deploy

**HVPN-P1-130 — `helixvpnctl` bootstrap + ops CLI (Cobra)** (`07-…` §18; M · AC1)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `init` generates tenant/admin/CA/overlay-`/48`/creds/gateway-keys/quadlets | `helixvpnctl init` on a fresh host produces a runnable deploy (FA, captured) | UNIT,FA | M |
| `.2` | `gateway keys`/`enroll-token`/`policy apply`/`device revoke` subcommands | `policy apply ./policy.jsonc` validates + activates (E2E) | E2E | S |

**HVPN-P1-131 — Podman quadlets (rootless, systemd-managed)** (`07-…` §18; M · AC1)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Render `helixd`/`helix-edge`/`postgres`/`redis` quadlets, rootless, read-only rootfs | `systemctl --user start helixvpn-pod` brings the stack up rootless (FA) | INT,FA | M |
| `.2` | `helixd` connects to PG as the non-superuser role (RLS enforced) | the app role cannot bypass RLS (SEC) | SEC | S |

**HVPN-P1-132 — On-demand integration infra via `containers` submodule** (`07-…` §18; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | QA harness boots+tears down the full stack via the submodule (§11.4.76) | the E2E suite boots+tears down the stack itself (FA, captured) | INT,FA | S |

**HVPN-P1-133 — Compose dev-loop + K8s Phase-2 shape (reference)** (`07-…` §18; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Docker Compose dev-loop (local) + reference stateless-coordinator K8s manifest | dev compose boots PG+Redis+helixd; the K8s manifest is lint-valid (not deployed at MVP) | INT | S |

---

## 14. E14 — QA, SLOs, DoD certification

*Anti-bluff capstone: the §11.4.40 full-suite retest, §11.4.107 recorded evidence, §11.4.27 100%-type coverage.*

**HVPN-P1-140 — netns + DPI + netem E2E rig (CI-ready)** (`07-…` §19; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Promote the Phase-0 rig fed by the real control plane (lanA + nft WG-block + tc netem) | `make e2e` runs the full slice unattended | E2E,FA | M |
| `.2` | Re-runnable `-count=3` identically (§11.4.98) | three identical runs (captured) | FA | S |

**HVPN-P1-141 — SLO instrumentation + dashboards** (`07-…` §19; S · SLO1-4)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Prometheus metrics + Grafana-as-code + alert rules at the §2 SLO targets | all four SLOs measured + asserted in CI thresholds (captured histograms, alert-on-breach) | PERF,FA | S |

**HVPN-P1-142 — Security test suite** (`07-…` §19; L · AC3/AC6/AC7/AC8)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | RLS cross-tenant denial + key-never-leaves + mTLS-reject + kill-switch/DNS pcap + revoke timing + secret-leak audit (§11.4.10.A) | every assertion green with captured evidence | SEC,SCALE | L |
| `.2` | Paired §1.1 mutations (drop an RLS policy / weaken kill-switch) FAIL their gate | each mutation FAILs (captured) | SEC,CHAOS | M |

**HVPN-P1-143 — Stress + chaos suite (§11.4.85)** (`07-…` §19; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Sustained load + concurrency + boundary across the control plane | no deadlock/leak under 10× concurrency | STRESS,SCALE | M |
| `.2` | Failure injection (Redis kill / PG drop / coordinator SIGKILL / disk-full) | every fault recovers to a consistent state with a captured `recovery_trace.log` | CHAOS | M |

**HVPN-P1-144 — HelixQA Challenge bank + vision-verified recordings** (`07-…` §19; M · AC1-AC9)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `challenges`/`helix_qa` bank dispatching real user journeys, PASS only on captured evidence | the 8-AC journey set scores PASS with a vision-confirmed MP4 per AC (no frozen/bluff frame §11.4.107) | CHAL,REC,UX | M |
| `.2` | Analyzer self-validated (golden-good/bad); broken build FAILs | a deliberately broken build scores FAIL | CHAL | S |

**HVPN-P1-145 — Full-suite retest + DoD certification gate** (`07-…` §19; M · AC1-AC9/SLO1-4)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | §11.4.40 release-gate sweep on a clean baseline (pre/post-build + on-device + meta-mutation + Challenge + no-log lint + CONTINUATION sync) | all of §21 green simultaneously on a from-zero install (§11.4.108 fresh deploy) | FA,E2E,CHAL | M |

---

## 15. E15 — Governance & release

**HVPN-P1-150 — Workable-items DB projection + docs-chain sync** (`07-…` §20; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `cmd/workable-items/` loader: parse leaves → upsert `items`/`test_diary` (git-tracked DB §11.4.95) | md↔db byte-identical round-trip (§11.4.93) | UNIT,INT,FA | M |
| `.2` | `.docs_chain/contexts/wbs.yaml` registered (§11.4.106) | a leaf edit re-syncs exports out-of-the-box; `verify` is the CI gate | INT,FA | S |

**HVPN-P1-151 — Release tagging + multi-upstream publish** (`07-…` §20; S · AC1-AC9)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Cut `<HELIX_RELEASE_PREFIX>-1.0.0-mvp` (§11.4.151) on main + every owned submodule | `git tag -l '<PREFIX>-*'` enumerates the whole release across repos | FA | S |
| `.2` | Publish to all upstreams via merge-onto-latest-main (no force-push §11.4.113) | the tag is reachable on every remote; HVPN-P1-145 cert report is the tag's evidence | FA | S |

---

## 16. Subtask roll-up + coverage note

| Epic | Parent tasks | Deepened subtasks | Notes |
|---|---|---|---|
| E01 Foundation | 4 | 12 | repo/proto/codegen/dev-infra |
| E02 Store+RLS+lint | 4 | 11 | risk-first correctness floor |
| E03 Identity/PKI | 4 | 11 | revoke-<1s + key-never-leaves |
| E04 IPAM | 2 | 5 | ULA /48 + 4via6 (D4) |
| E05 Events | 3 | 4 | Redis Streams (D3) |
| E06 Policy compiler | 3 | 9 | two-artifact, default-deny |
| E07 Coordinator | 5 | 11 | <1s convergence + 24h soak |
| E08 API+authz | 4 | 8 | Connect+REST+RBAC+RLS backstop |
| E09 helix-core | 6 | 16 | Transport/WG/orchestrator/kill-switch/FFI |
| E10 helix-edge | 4 | 9 | MASQUE term + verdict-map + fail-static |
| E11 helix-ui | 5 | 10 | OpenDesign + 3 flavors |
| E12 shims | 3 | 6 | Linux/Android/iOS |
| E13 deploy | 4 | 7 | helixvpnctl + quadlets |
| E14 QA/DoD | 6 | 11 | rig/SLO/security/chaos/Challenge/cert |
| E15 governance | 2 | 4 | items-DB + release tag |
| **Total** | **59** | **134** | — |

Subtask complexity sums to ≈ the parent task effort (`07-…` §22, ~273
person-days excl. Phase-0 prereqs); it is a **sizing aid, not a commitment**
(§11.4.6). Every subtask is anti-bluff: its acceptance is a captured-evidence
assertion (§11.4.5/.69/.107), metadata-only PASS is forbidden (§11.4.1), and it
is `complete` only with a `test_diary` evidence path (`workable-items-model.md`
§9). The `.k` ids are monotonic per parent and never renumbered (§11.4.54). The
full set of these subtasks feeds the §11.4.93 DB via the
`subtask-deepening-p1.md` docs_chain source (`workable-items-model.md` §7).

---

## Sources verified

- `07-phase1-mvp-wbs.md` §6–§20 (every task Desc/Deliverable/Acceptance/Effort/Tests), §21 traceability, §22 effort roll-up — read 2026-06-26.
- `06-phase0-spike-wbs.md` §0.1 (the 3-tier breakdown R5 aligns to) — read 2026-06-26.
- `REFINEMENT_NOTES.md` R5 (subtask-tier asymmetry to close) — read 2026-06-26.
- Sibling `workable-items-model.md` (§3 field mapping, §6 test diary, §9 complete-definition), `dependency-graph.md` (the inherited DAG) — authored this volume.
- Constitution anchors §11.4.5/.6/.40/.54/.58/.69/.85/.91/.93/.98/.107/.108/.113/.132/.143/.151/.156/.159/.162/.169 — read 2026-06-26.

> Honest boundary (§11.4.6): subtasks decompose the *stated* parent-task work;
> no new scope is invented. All complexity figures are sizing `TARGET`s, never
> date commitments; device-bound subtasks (iOS HVPN-P1-122.*, Android
> HVPN-P1-121.*) carry the §11.4.3 SKIP-with-reason fallback where no device is
> reachable.
