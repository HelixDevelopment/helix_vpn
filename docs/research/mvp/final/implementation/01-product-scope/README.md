# 01 — Product Scope

**Revision:** 1
**Last modified:** 2026-07-05T14:20:00Z
**Status:** Draft — consolidated from `00-product-scope-and-principles.md` and Volume 1 (`v01-product/*`).

---

## 1. Product definition

HelixVPN is a **self-hostable overlay network with a privacy-VPN front end**.

It unifies four lineages:

| Borrowed from | What HelixVPN takes |
|---|---|
| **Cloudflare Tunnel + WARP** | Connector dials out; MASQUE/QUIC transport |
| **Tailscale / Headscale** | Coordination plane, network map, push-don't-poll, ACL-compiled topology |
| **Mullvad** | Obfuscation + privacy bar (MASQUE, Shadowsocks, UoT, LWO, DAITA, multi-hop, kill-switch, no-logging) |
| **NetBird** | Sanity-check OSS shape (WG + management/signal server, self-hosted) |

**Founding constraint:** a remote user must obtain full, policy-scoped access to **one or many internal/home/lab networks without any inbound port-forward**.

## 2. What HelixVPN is / is NOT

### 2.1 It IS

1. A WireGuard-crypto-core overlay; obfuscation is a pluggable layer **beneath** WG.
2. A three-role system: Connector, Gateway, Client.
3. Self-hostable by one person, scaling to HA multi-region with the same images.
4. Cross-platform on iOS, Android, Aurora OS, HarmonyOS NEXT, Windows, Linux, macOS, Web.
5. Event-driven, push-based control plane.
6. No-logging-by-construction — CI-enforced, not a toggle.

### 2.2 It is NOT

| NOT | Boundary |
|---|---|
| A forked or re-rolled WireGuard crypto | Transport spec owns wire shape only; CI lint forbids touching WG crypto primitives. |
| A separate "QUIC protocol" alongside WG | Mullvad's QUIC mode **is** WG-over-MASQUE/HTTP-3. |
| A system VPN in a browser | Web = Console management + optional browser-scoped WASM MASQUE proxy. |
| A logging / lawful-intercept / DLP appliance | Control actions are audited; traffic never is. |
| A generic L7 API gateway / service mesh | L3 IP overlay + L4 port policy only. |
| A consumer "free VPN" ad-funded service | Default is self-hosted; managed SKU is optional later. |

## 3. Roles and apps

### 3.1 Three roles

| Role | Runs | Direction | Reuses |
|---|---|---|---|
| **Connector** | Inside a private network | Outbound-only | `helix-core` advertise/route mode |
| **Gateway** | Public VPS | Accepts dials | `helix-core` edge + `helix-go` control plane |
| **Client** | End-user device | Outbound-only | `helix-core` + `helix-ui` |

### 3.2 Three app classes

| App | Persona | Primary jobs | Cores used |
|---|---|---|---|
| **Helix Access** | End user | Connect, pick exit/network, toggle obfuscation, kill-switch, split-tunnel | `helix-ui` + `helix-core` + shim |
| **Helix Connector** | Network operator | Onboard network, advertise CIDRs, local ACLs, run headless | `helix-core` (advertise/route) + optional `helix-ui` |
| **Helix Console** | Admin | Tenants, users, devices, networks, routes, policies, audit, billing-optional | `helix-ui` admin flavor only — **no tunnel core** |

All three build from one Flutter tree via `runHelixApp(flavor, home, capabilities)`.

## 4. "Two-way" and "multi-network"

- **Two-way:** both the network-side leg (Connector → Gateway) and the user-side leg (Client → Gateway) are outbound; the Gateway stitches and polices between them.
- **Multi-network:** `1 user → N joined private networks`, each policy-scoped. This requires solving:
  1. Overlapping RFC1918 ranges (D4).
  2. Per-user ACL routing (default-deny).
  3. Split horizon (connectors/clients cannot reach ungranted networks).

## 5. Prior art and positioning

| System | Self-host | Obfuscation | Multi-network | Cross-platform apps |
|---|---|---|---|---|
| Mullvad | ✗ | ✅ best | ✗ | ✅ |
| Tailscale | partial | ✗ | ✅ | ✅ |
| NetBird | ✅ | limited | ✅ | ✅ |
| Cloudflare Tunnel+WARP | ✗ | MASQUE | ✅ | ✅ |
| **HelixVPN** | ✅ | ✅ full | ✅ | ✅ 8 platforms, one codebase |

## 6. Hosting, licensing, and commercial stance

- **Self-hosted / home-lab first.** A managed SKU may come later using the same images.
- **Licensing (D8)** is an open decision; recommendation is **L-C**: permissive (Apache-2.0/MIT) reusable cores (`helix-core`, `helix-ui`, `helix-proto`) + source-available (BSL-class) managed Console layer. Owner-gated before public release.
- **No-logging is invariant** regardless of hosting shape.

## 7. Seven non-negotiable principles

1. **Control plane and data plane are strictly separated** — fail-static.
2. **WireGuard is the crypto core; transports are pluggable beneath it.**
3. **Outbound-only from edges** — no inbound holes.
4. **Push, don't poll** — `WatchNetworkMap` streaming deltas.
5. **One core per concern, reused everywhere**.
6. **Self-hostable by one person, scalable to many gateways**.
7. **No-logging by construction** — CI schema-lint fails on durable flow tables.

## 8. Mullvad parity matrix

| # | Feature | Implementation | Phase |
|---|---|---|---|
| F1 | WireGuard-only crypto | `helix-core` WG (kernel fast path, `boringtun` fallback) | P0/P1 |
| F2 | QUIC obfuscation (MASQUE) | `helix-transport` CONNECT-UDP via `quinn`+`h3`; edge `:443/udp` | P0/P1 |
| F3 | CONNECT-IP | `quinn`+`h3` CONNECT-IP path | P2 |
| F4 | Shadowsocks | `helix-transport` Shadowsocks wrap | P2 |
| F5 | UDP-over-TCP | `helix-transport` UoT | P2 |
| F6 | LWO | `helix-transport` LWO (XOR/padding) | P1 |
| F7 | Automatic obfuscation | Client escalation ladder | P1 |
| F8 | Custom WG port / 443 / 53 | Edge multi-listener + port-hopping | P1 |
| F9 | DAITA | Optional shaping above WG (`maybenot`-style) | P2 |
| F10 | Multi-hop | Nested WG, control-plane orchestrated | P2 |
| F11 | Kill-switch | OS firewall rules driven by core state machine | P1 |
| F12 | Split tunneling | Per-route `AllowedIPs` + per-app rules | P1 |
| F13 | DNS leak protection | Core sets tunnel DNS; blocks plaintext :53 off-tunnel | P1 |
| F14 | No-logging | Ephemeral Redis presence; no durable table | P1 |
| F15 | Anonymous account | Anonymous device-token enrollment alongside OIDC | P1 |
| F16 | Post-quantum handshake | WG PQ PSK / ML-KEM; hybrid never PQ-only | P2 |
| F17 | Per-device management | Registry + Console; revoke <1s | P1 |

**HelixVPN differentiators:** X1 multi-network overlay, X2 outbound-only onboarding, X3 self-hostable Mullvad-class stack, X4 direct P2P + NAT traversal + relay, X5 8-platform reach.

## 9. Scope per phase

| Phase | In scope | Out of scope |
|---|---|---|
| **P0 Spike** | G1–G6: plain-UDP slice, MASQUE through DPI, iOS NE memory, Go-vs-Rust edge, FFI, static-map reconcile | Product UIs, persistence, multi-tenancy, HA |
| **P1 MVP** | Go monolith, Postgres+RLS, `WatchNetworkMap`, Redis Streams, identity/PKI, policy compiler, Gin REST/WS/SSE, `helixvpnctl`, Podman quadlets, auto-ladder, iOS/Android/Linux Access, Connector, Console (web) | Shadowsocks/UoT/DAITA/P2P/multi-hop/PQ, desktop Windows/macOS, HA, Aurora/HarmonyOS/WASM |
| **P2 Parity+Reach** | Full transport set, DAITA, P2P/NAT/relay, multi-hop, PQ, Windows/macOS desktop, GitOps, HA/multi-region | Aurora/HarmonyOS/WASM, billing, audit, reproducible builds |
| **P3 Extended Reach** | HarmonyOS, Aurora, WASM proxy, billing-optional multi-tenant, third-party audit, reproducible builds | L7 inspection, lawful-intercept, ad-funded free tier |

## 10. Open architectural decisions D1–D8

| ID | Decision | Recommendation | Gate |
|---|---|---|---|
| D1 | Primary obfuscating transport | **MASQUE/QUIC** (Hysteria2 as ladder rung) | Phase-0 G2 |
| D2 | Shared client-core language | **Rust** core + Flutter UI | Phase-0 G3 (make-or-break) |
| D3 | Event bus | **Redis Streams** MVP → **NATS** Phase 2 | Phase-2 HA work |
| D4 | Subnet collision | **ULA /48 + 4via6** (NAT fallback) | Phase-1 IPAM design |
| D5 | Gateway edge language | **Rust** (`quinn`+`h3`) | Phase-0 G4 benchmark |
| D6 | Transport topology | **Asymmetric per-leg** via one crate | Phase-1 coordinator design |
| D7 | MVP ambition | **Lean spike → 8-criteria MVP** | Settled |
| D8 | Licensing model | **L-C** (permissive reusable cores + source-available managed layer) | Owner-gated before public release |

## 11. Cross-references

- Functional requirements → [`../../v01-product/functional-requirements.md`](../../v01-product/functional-requirements.md)
- Non-functional requirements → [`../../v01-product/nonfunctional-requirements.md`](../../v01-product/nonfunctional-requirements.md)
- Personas and RBAC → [`../../v01-product/personas-and-roles.md`](../../v01-product/personas-and-roles.md)
- System architecture → [02 — System Architecture](../02-system-architecture/README.md)
- Data plane → [03 — Data Plane](../03-data-plane/README.md)
- Control plane → [04 — Control Plane](../04-control-plane/README.md)

---

*Sources: `docs/research/mvp/final/00-product-scope-and-principles.md` §1–§13, `v01-product/functional-requirements.md`, `v01-product/nonfunctional-requirements.md`, `v01-product/personas-and-roles.md`.*
