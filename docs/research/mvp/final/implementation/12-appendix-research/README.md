# 12 — Appendix: Research

**Revision:** 1
**Last modified:** 2026-07-05T14:20:00Z
**Status:** Draft — consolidated from `11-deep-research-appendix.md` and `v09-research/*`.

---

## 1. Position

This appendix indexes the **external, citable findings** that shaped the HelixVPN master specification. Every research angle was investigated with live web access (2026-06-25) and is recorded in its own dossier under `v09-research/`. This README is a navigable summary; the dossiers and [`11-deep-research-appendix.md`](../../11-deep-research-appendix.md) retain the full sources and verbatim URLs.

---

## 2. Research angles

| Dossier | Topic | Key finding | Surfaced in |
|---|---|---|---|
| [`research-masque.md`](../../v09-research/research-masque.md) | MASQUE / CONNECT-UDP / HTTP Datagrams | WG-over-HTTP/3 is a stack of stable RFCs (9298, 9297, 9221, 9220); Mullvad shipped QUIC obfuscation in 2025.9 using `quinn` + MASQUE. | D1 Camp A, `01-data-plane`, `08-phase2-parity-wbs.md` |
| [`research-hysteria2.md`](../../v09-research/research-hysteria2.md) | Hysteria2 + Salamander / Gecko, sing-box | Hysteria2 v2.9.2 adds Gecko handshake fragmentation; `masque-go` is turnkey in Go; sing-box is a universal multiplexer but **not adopted**. | D1 Camp B, Phase-2 transport set, source gap G2 |
| [`research-wireguard.md`](../../v09-research/research-wireguard.md) | WireGuard protocol + implementations | Noise IK 1-RTT; fixed primitives (Curve25519, ChaCha20Poly1305, BLAKE2s, HKDF, SipHash24, XChaCha20Poly1305); exact wire sizes confirmed (148/92/64/32 B). | `01-data-plane`, `v02-data-plane/wireguard-core.md` |
| [`research-mullvad.md`](../../v09-research/research-mullvad.md) | Mullvad parity feature set | DAITA v1/v2 built on maybenot; constant packet size + cover traffic + pattern distortion; server-driven defenses in v2. | `04-security-privacy-pki`, Phase-2 DAITA track |
| [`research-daita_test.md`](../../v09-research/research-daita_test.md) | maybenot + VPN testing rigs | maybenot 2.2.2 core, MIT, used by Mullvad; Linux netns + nftables + tc netem is the standard test harness. | DAITA design, `10-testing-acceptance-and-qa.md` |
| [`research-pki_pq_nat.md`](../../v09-research/research-pki_pq_nat.md) | PKI, post-quantum, NAT traversal | SPIFFE/SPIRE-style short-lived mTLS over WG; ML-KEM hybrid PSK (Mullvad/Rosenpass); Tailscale-style STUN+hole-punch+DERP relay. | `04-security-privacy-pki`, `08-phase2-parity-wbs.md` |
| [`research-ios_android.md`](../../v09-research/research-ios_android.md) | Mobile native constraints | iOS NEPacketTunnelProvider limit ≈ 50 MiB (unofficial), plan to ~12–15 MB working set; Rust staticlib size optimizations mandatory. | `03-client-core-and-ui`, G3 gate |
| [`research-go_cp.md`](../../v09-research/research-go_cp.md) | Go control-plane stack | Gin 1.12.0, Connect-RPC 1.20.0, sqlc 1.31.1, protovalidate v1.0 — versions and best practices. | `02-control-plane`, `08-api-contracts` |
| [`research-flutter_ffi.md`](../../v09-research/research-flutter_ffi.md) | Rust ↔ Flutter / native bridges | flutter_rust_bridge 2.12.0 recommended; UniFFI-Dart experimental; per-platform staticlib/cdylib packaging. | `03-client-core-and-ui` |
| [`research-podman_k8s.md`](../../v09-research/research-podman_k8s.md) | Rootless Podman, quadlets, K8s | Quadlet is core in Podman 5.x; rootless unit search paths; `DropCapability=ALL` + `AddCapability=NET_ADMIN NET_RAW` + `/dev/net/tun`. | `07-infrastructure-devops` |

---

## 3. Cross-doc synthesis

The high-level synthesis in [`v09-research/_SYNTHESIS.md`](../../v09-research/_SYNTHESIS.md) distills the 16 original source documents into:

- The settled product floor (self-hostable overlay network + privacy-VPN front end).
- The near-unanimous stack floor (Go + Gin + Postgres + Redis + rootless Podman; WireGuard; Flutter).
- The eight key decisions D1–D8 (obfuscation transport, client-core language, event bus, subnet collision, edge language, transport topology, MVP ambition, licensing).
- The four-phase roadmap used by the WBS docs.

---

## 4. Decision coverage check

All eight decisions are surfaced explicitly somewhere in `final/`:

| Decision | Camp A | Camp B | Surfaced in |
|---|---|---|---|
| D1 obfuscation | MASQUE/QUIC (Mullvad parity) | Hysteria2 + Salamander | `01-data-plane`, `08-phase2-parity-wbs.md`, `11-deep-research-appendix.md` |
| D2 client core | Rust + Flutter | Go + Flutter | `03-client-core-and-ui.md`, `11-deep-research-appendix.md` |
| D3 event bus | Redis Streams → NATS | NATS from start | `02-control-plane.md`, `08-phase2-parity-wbs.md` |
| D4 subnet collision | IPv6 ULA /48 + 4via6 | CGNAT 100.64/10 | `02-control-plane.md`, `08-phase2-parity-wbs.md` |
| D5 edge language | Rust (quinn+h3) | Go (quic-go) | `01-data-plane.md`, G4 gate |
| D6 transport topology | single protocol | asymmetric per-leg | `01-data-plane.md`, `11-deep-research-appendix.md` |
| D7 ambition | lean tunnel-first | full Connectivity-OS | `00-product-scope-and-principles.md`, `SPECIFICATION.md` |
| D8 licensing | source-available + commercial | pure OSS | `SPECIFICATION.md` §9 (pre-Phase-3) |

---

## 5. Honest boundaries

- This appendix is an **index**; every factual claim is backed by the cited dossier and its sources.
- Some values (e.g. exact iOS NE memory limit, future OpenDesign exporter shapes) are marked `UNVERIFIED` in the source dossiers per §11.4.6.
- Tool versions are pinned to the access date (2026-06-25) and should be re-verified before procurement.

---

## 6. Cross-references

- Deep-research appendix → [`../../11-deep-research-appendix.md`](../../11-deep-research-appendix.md)
- Cross-doc synthesis → [`../../v09-research/_SYNTHESIS.md`](../../v09-research/_SYNTHESIS.md)
- Source coverage ledger → [99 — Source Coverage Ledger](../99-source-coverage-ledger/README.md)

---

*Sources: `11-deep-research-appendix.md`, `v09-research/*.md`.*
