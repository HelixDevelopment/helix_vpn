# 04 — Control Plane

**Revision:** 1
**Last modified:** 2026-07-05T14:20:00Z
**Status:** Draft — consolidated from `02-control-plane.md` and `v03-control-plane/*`.

---

## 1. Position

The control plane is the **Go brain** that holds identity, topology, and policy truth, compiles ACLs, and pushes desired-state network maps to every edge. It **never sits in the packet path**.

**Stack:** Go + Gin + PostgreSQL + Redis + Podman (rootless).

## 2. Modular monolith layout

```text
helix-go/
├── cmd/
│   ├── helixd/              # control-plane binary
│   └── helixvpnctl/         # bootstrap + ops CLI
├── internal/
│   ├── identity/            # tenants, users, OIDC, anonymous device tokens
│   ├── registry/            # devices, connectors, advertised prefixes, presence
│   ├── ipam/                # overlay IP allocation (ULA /48 per tenant)
│   ├── pki/                 # short-lived device certs, rotation, revoke
│   ├── policy/              # ACL model + pure, versioned compiler
│   ├── coordinator/         # in-mem topology graph, deltas, WatchNetworkMap streams
│   ├── events/              # bus abstraction (Redis Streams MVP → NATS JetStream P2)
│   ├── telemetry/           # Prometheus metrics + control-action audit sink
│   ├── api/                 # Gin REST + Connect (agents) + WS/SSE
│   └── store/               # Postgres pool, RLS tenant-tx helper, migrations
├── proto/                   # helix/coordinator/v1/*.proto
├── openapi/                 # REST contract
└── migrations/              # goose SQL — schema authority
```

## 3. Wiring rules

- **R1** — no cross-store imports; modules read each other through exported interfaces.
- **R2** — `coordinator` owns no durable tables; only in-memory graph + streams.
- **R3** — every state mutation emits an event (transactional outbox pattern).
- **R4** — every DB access is tenant-scoped under RLS.

## 4. Data model

Core tenant-scoped tables:

- `tenants`, `users` (with `user_role` enum: admin/operator/member)
- `devices`, `connectors`, `advertised_prefixes`
- `groups`, `group_members`
- `policies` (jsonb spec, monotonic version, active flag)
- `overlay_pools` (ULA /48 per tenant)
- `device_certs` (short-lived, revocable)
- `audit_events` (control actions only — never traffic)
- `outbox` (transactional event staging)

**Absent by design:** no `connections`, `sessions`, `flows`, `traffic`, `packets` tables. Their absence is CI-lint-enforced.

## 5. Row-Level Security

```sql
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE devices FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON devices
  USING (tenant_id = current_setting('app.tenant_id')::uuid);
```

The app connects as a non-superuser `helix_app` role; tenant isolation is enforced at the database even if app-layer RBAC is bypassed.

## 6. Policy service and compiler (with GAP-1 consolidation)

The `policy` package is a **pure, deterministic, versioned compiler**:

- Input: Tailscale-ACL-flavored JSON document (`policies.spec`).
- Output:
  - `VisibleTo` — need-to-know peer set.
  - `AllowedIPs` — coarse WG routes.
  - `Verdicts` — port/proto-granular edge map.
  - `Via6` — 4via6 mappings for overlapping IPv4 LANs.
  - `ExitNodes` — full-tunnel grants.

**GAP-1 — connector local-ACL precedence (adopted in consolidation):**

The `policy` compiler accepts an optional `connector.local_denylist` input. The precedence rule is:

1. Local-deny overrides central-allow.
2. Central-deny overrides local-allow.
3. Output = central policy minus local-deny.

> **Honesty note:** backported into `v03-control-plane/svc-policy.md` and `v04-client/helix-core-rust.md`; GAP-1 CLOSED.

## 7. Event backbone

- **MVP:** Redis Streams.
- **Phase 2:** NATS JetStream (mechanical swap via `Bus` interface).

Key streams: `device.enrolled`, `device.revoked`, `route.advertised`, `policy.updated`, `policy.compiled`.

Reliable publish via transactional `outbox` + sweeper.

## 8. Coordinator and `WatchNetworkMap`

The coordinator maintains an **in-memory per-tenant topology graph** and streams:

- `Snapshot` — full desired-state on first connect or version gap.
- `Delta` — minimal change set for the affected agent.

Peers are **pre-policy-filtered** before they hit the wire (need-to-know).

**SLOs:**
- Convergence p99 < 1 s (event → delta-on-wire).
- Revoke < 1 s (peer dropped everywhere).

## 9. API surface

- **REST (Gin):** admin / Console CRUD, OpenAPI-generated clients.
- **Connect-RPC / gRPC:** agent `WatchNetworkMap` and enrollment.
- **WebSocket / SSE:** live Console state feed.

Auth: OIDC session or API token + RBAC (`admin` / `operator` / `member`), with RLS as the backstop.

## 10. Reconciliation flow

```text
Admin/Operator → REST/CLI
        │
        ▼
   Postgres commit
        │
   outbox row (same tx)
        │
   sweeper → Redis Streams XADD
        │
   coordinator XREADGROUP
        │
   recompute graph + minimal affected set
        │
   stream.Send(MapDelta) to affected agents
        │
   helix-core reconciler applies routes / peers / verdict map
```

## 11. Cross-references

- Data plane → [03 — Data Plane](../03-data-plane/README.md)
- API contracts → [08 — API Contracts](../08-api-contracts/README.md)
- Security/PKI → [06 — Security, Privacy & PKI](../06-security-privacy-pki/README.md)
- Client core → [05 — Client Core & UI](../05-client-core-ui/README.md)
- Detailed DDL → [`../../v03-control-plane/data-model-ddl.md`](../../v03-control-plane/data-model-ddl.md)
- Detailed protobuf → [`../../v03-control-plane/protobuf-spec.md`](../../v03-control-plane/protobuf-spec.md)
- Detailed coordinator → [`../../v03-control-plane/svc-coordinator.md`](../../v03-control-plane/svc-coordinator.md)

---

*Sources: `docs/research/mvp/final/02-control-plane.md`, `v03-control-plane/*.md`.*
