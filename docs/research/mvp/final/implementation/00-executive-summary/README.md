# 00 — Executive Summary

**Revision:** 1
**Last modified:** 2026-07-05T14:20:00Z
**Status:** Draft — consolidated from `SPECIFICATION.md` and `MASTER_INDEX.md`.

---

## One-sentence pitch

HelixVPN is a **self-hostable overlay network with a privacy-VPN front end**: Cloudflare Tunnel + WARP, rebuilt as Tailscale-style coordination, with Mullvad's obfuscation stack, fully self-hostable, on one shared codebase.

## Why it exists

Remote users need full, policy-scoped access to **one or many** internal/home/lab networks **without any inbound port-forward**. Internal hosts dial *outbound* to a public gateway; the gateway relays and routes.

## Lead differentiators

1. **Self-hosted + Mullvad-grade obfuscation** (MASQUE/QUIC, Shadowsocks, UDP-over-TCP, LWO, DAITA).
2. **One user → N joined private networks** with bidirectional gateway and ACL routing.
3. **8-platform reach** including Aurora OS and HarmonyOS NEXT on one codebase.
4. **One shared Rust + Flutter codebase** — client, connector, and edge reuse the same core.
5. **Event-driven real-time control plane** — sub-second convergence, push-not-poll.

## The three roles

| Role | Runs | Direction | Reuses |
|---|---|---|---|
| **Connector** | Inside a private network | Outbound-only | `helix-core` advertise/route mode |
| **Gateway** | Public VPS | Accepts dials | `helix-core` edge + `helix-go` control plane |
| **Client** | End-user device | Outbound-only | `helix-core` + `helix-ui` |

## The three app classes

| App | User | Platforms |
|---|---|---|
| **Helix Access** | End user | iOS, Android, Aurora, HarmonyOS, Windows, Linux, macOS, Web (limited) |
| **Helix Connector** | Network operator | Linux/Windows/macOS daemon (+ optional UI); Android/embedded appliance |
| **Helix Console** | Admin | Web + Desktop Flutter (API client only, no tunnel core) |

## Non-negotiable principles

1. Control plane and data plane are strictly separated — **fail-static**.
2. **WireGuard** is the cryptographic core; transports are pluggable *beneath* it.
3. **Outbound-only** from edges — no inbound holes.
4. **Push, don't poll** — `WatchNetworkMap` streaming deltas.
5. **One core per concern, reused everywhere**.
6. **Self-hostable by one person, scalable to many gateways**.
7. **No-logging by construction** — CI schema-lint fails on durable flow tables.
8. **Schema-first, zero drift** — protobuf + OpenAPI, generated clients.
9. **Decoupled, reusable components** per constitution §11.4.28/§11.4.74.

## Phase roadmap

| Phase | Goal | Hard gates |
|---|---|---|
| **Phase 0 — Spike** (~3–4 weeks) | Prove the hard parts on production interfaces | G1–G6 (throughput, MASQUE, iOS memory, Go-vs-Rust edge, FFI, reconcile) |
| **Phase 1 — MVP** | Self-hostable product | 8-criteria Definition-of-Done (self-host, enroll, reach/deny, MASQUE, policy <1s, revoke <1s, kill-switch, no logs) |
| **Phase 2 — Parity + Reach** | Full Mullvad parity + HA/multi-region | P2-AC1–AC8 + P2-SLO1–SLO5 (P2P, DAITA, multi-hop, PQ, desktop, GitOps, regional failover <3s) |
| **Phase 3 — Extended Reach** | HarmonyOS, Aurora, WASM proxy, billing, audit, reproducible builds | G20–G26 |

## Open decisions (D1–D8)

See [01 — Product Scope](../01-product-scope/README.md) §7 for the full decision register with options + recommendations. Key decisions:

- **D1** — Primary obfuscating transport: MASQUE/QUIC (recommended).
- **D2** — Shared client-core language: Rust (make-or-break G3).
- **D3** — Event bus: Redis Streams MVP → NATS JetStream Phase 2.
- **D4** — Subnet collision: IPv6 ULA /48 + Tailscale 4via6.
- **D5** — Gateway edge language: Rust if G4 benchmark holds.
- **D6** — Transport topology: best-fit-per-leg as policy.
- **D7** — MVP ambition: lean tunnel-first.
- **D8** — Licensing / positioning: decide before public release.

## Where to read next

- Product definition and scope → [01 — Product Scope](../01-product-scope/README.md)
- C4 architecture and components → [02 — System Architecture](../02-system-architecture/README.md)
- Detailed roadmap and WBS → [11 — Guides & FAQs](../11-guides-faqs/README.md)
- Gap analysis → [`docs/reviews/mvp-final/findings/phase1-docs-gap-analysis.md`](../../../../../reviews/mvp-final/findings/phase1-docs-gap-analysis.md)

---

*Sources: `docs/research/mvp/final/SPECIFICATION.md` §1–§9, `MASTER_INDEX.md`, `00-product-scope-and-principles.md` §1–§11.*
