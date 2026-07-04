# RESEARCH hysteria2

**Revision:** 1
**Last modified:** 2026-07-04T12:00:00Z

Deep web research for a Mullvad-parity self-hosted VPN spec. Topic: Hysteria2 +
Salamander vs MASQUE; sing-box; AmneziaWG; udp2tcp/Shadowsocks; which obfuscation
survives which censorship regime. All facts cite a source URL + access date.
Web access: **YES** (WebSearch reachable; WebFetch blocked on a few domains —
`v2.hysteria.network`, `mullvad.net` — worked around via mirrors/search summaries).
Access date for all entries unless noted: **2026-06-25**.

---

## 1. Hysteria2 + Salamander / Gecko obfuscation

**What it is.** Hysteria is a TCP & UDP proxy built on **QUIC (RFC 9000) with the
Unreliable Datagram Extension (RFC 9221)**. The protocol shipped since v2.0.0 is
internally "v4". It is designed for speed + censorship resistance, with a custom
congestion-control called **Brutal** that deliberately *ignores packet loss* to
maximise throughput on lossy/throttled links (the "bandwidth cheating" behaviour —
the client tells the server a fixed bandwidth and Brutal sends at that rate
regardless of loss signals). [hysteria protocol docs; sing-box hysteria2 docs]

**Salamander obfuscation.** Encapsulates *all* QUIC packets. The obfuscator
computes a **BLAKE2b-256** hash of a randomly-generated **8-byte salt** appended to
a **user-provided pre-shared key**, and XOR/scrambles every packet into
seemingly-random bytes with no pattern. Purpose: defeat networks that specifically
fingerprint/block **QUIC or HTTP/3** (but not UDP in general) — it hides the QUIC
Initial packet structure so DPI cannot see a QUIC handshake or SNI. It does **not**
help where UDP itself is blocked. [hysteria protocol docs]

**Gecko (experimental).** Builds on Salamander and *additionally fragments the QUIC
handshake packets into randomly-sized, randomly-padded chunks* (configurable
`min_packet_size` / `max_packet_size`). This directly targets the GFW's
single-datagram QUIC-Initial inspection weakness (see §5). Added to both apernet
hysteria and sing-box in 2025–2026. [sing-box changelog; hysteria changelog]

**Current version (load-bearing fact):** apernet/hysteria latest = **app/v2.9.2,
released 2026-05-23**. That release adds the **Gecko** obfuscation, fixes a UDP
ACL-bypass security issue, and OOM hardening. ("important security fixes … strongly
encourage everyone to upgrade.") [github.com/apernet/hysteria/releases]

**Auth.** Hysteria2 uses a password/userpass over an HTTP/3-style auth exchange
inside the QUIC tunnel; combined with Salamander PSK this gives two independent
secrets (PSK for the wire obfuscation, password for authz).

---

## 2. MASQUE (CONNECT-UDP, RFC 9298) — and Mullvad's 2025 deployment

**What it is.** **RFC 9298 "Proxying UDP in HTTP" (Aug 2022)** defines CONNECT-UDP:
the client sends an Extended CONNECT with `:protocol = connect-udp`, encodes
target host/port in a URI template, and the proxy maps **QUIC DATAGRAM frames ↔ UDP
packets** to the target. Over HTTP/3 it is native-UDP; **RFC 9298 + RFC 9484
(CONNECT-IP) require HTTP/2 fallback** when QUIC is unavailable (works in
maximally-restrictive nets, but loses native-UDP semantics → TCP-over-TCP-ish).
[ietf-wg-masque draft / http.dev/masque]

**Mullvad's production use (the parity target).** Mullvad shipped **"QUIC Obfuscation
for WireGuard"** on **2025-09-09, app version 2025.9**, on all desktop platforms
(Android/iOS announced as forthcoming). It **tunnels WireGuard's UDP through an
HTTP/3 proxy using MASQUE (RFC 9298)** so the traffic "resemble[s] standard web
activity." Default behaviour: the desktop app **automatically tries QUIC after
failed connection attempts** (can be forced permanently in settings/CLI). It is
positioned for "regions/networks where WireGuard or Mullvad's other obfuscation
methods face restrictions." Mullvad published **no performance/overhead numbers**
and stated **no explicit threat it does not defend against** in the launch post.
[mullvad blog 2025/9/9; alternativeto.net 2025-09]

**Tooling.** `quic-go/masque-go` is the reference Go implementation of RFC 9298
(MASQUE over HTTP/3) — directly relevant if building a self-hosted MASQUE proxy in
Go. [github.com/quic-go/masque-go]

---

## 3. sing-box protocol landscape (2025–2026)

sing-box supports the relevant censorship-resistant set: **Hysteria2, TUIC, VLESS+
Reality, ShadowTLS, AnyTLS, Shadowsocks, Trojan, WireGuard**. Key facts:

- **Hysteria2** — QUIC + Brutal; "measurably higher throughput than TCP-based
  alternatives on stable connections"; **Gecko** obfuscation added as a new QUIC
  obfs type with `min/max_packet_size`. Caveat: UDP-based, so **blocked in
  corporate/restricted nets that drop UDP** → best as one option in a fallback
  chain, not a sole solution. [sing-box hysteria2 docs; sing-box changelog]
- **VLESS + Reality** — the standout for DPI evasion: borrows the **TLS fingerprint
  of a real high-traffic site**, making connections indistinguishable from normal
  HTTPS to DPI. TCP-based → survives UDP blocks. [curevpn comparison; sing-box]
- **AnyTLS** (added to sing-box ~**March 2025**) — designed to mitigate TLS-proxy
  traffic-shape characteristics with a new multiplexing scheme. [sing-box changelog]
- **ShadowTLS** — wraps an inner protocol inside a real TLS handshake to a real
  site so the handshake passes as genuine TLS. [sing-box]

Architectural takeaway: a Mullvad-parity self-host wants **a QUIC/UDP primary
(Hysteria2 or MASQUE) + a TLS-camouflage TCP fallback (Reality / ShadowTLS /
AnyTLS) + a UDP-block fallback (udp2tcp/Phantun)** rather than one protocol.

---

## 4. AmneziaWG — obfuscated WireGuard (the WireGuard-native alternative)

WireGuard fork adding protocol-level obfuscation against DPI. The standard
WireGuard fingerprint DPI exploits: **fixed 32-bit message types (1–4) + invariant
packet sizes** (handshake init always **148 bytes**, response always **92 bytes**).
AmneziaWG breaks both:

- **Junk packets:** before each handshake, send `Jc` junk packets of random size
  `Jmin..Jmax` (noise; client-side recommended only).
- **Header/size obfuscation (AmneziaWG 2.0):** dynamic header ranges instead of
  static 1–4, random padding on all message types, extended signature/junk
  ("CPS") packets before handshakes to mimic other UDP protocols. **Every server
  effectively speaks its own dialect → no universal DPI signature.**
- **Performance:** ~3% throughput cost on an uncensored net (WireGuard 95 Mbps vs
  AmneziaWG 92 Mbps in a cited benchmark).
- Still **UDP-based** → does not survive a hard UDP block on its own (pair with
  udp2tcp/Phantun). [docs.amnezia.org/amnezia-wg; dev.to AmneziaWG 2.0; deepwiki
  amneziawg-go]

Relevance to a Mullvad-parity self-host: AmneziaWG is the *lowest-overhead* DPI
evasion when the threat is **WireGuard fingerprinting, not UDP blocking**; MASQUE/
Hysteria2 are heavier but survive QUIC-permitting nets and HTTP-camouflage regimes.

---

## 5. Which obfuscation survives which censorship regime

### (a) China GFW — SNI-based QUIC censorship (USENIX Security 2025)
- GFW began **blocking QUIC by SNI since ~2024-04-07**; independently confirmed by
  USENIX Sec '25 ("Exposing and Circumventing SNI-based QUIC Censorship of the
  GFW") and UPB-SysSec (discovered **Jan 2025**, tested **Mar–Apr 2025**).
- Mechanism: GFW **derives the QUIC-Initial decryption key from the packet header**
  (the key is header-derivable per RFC 9001), extracts the **SNI**, matches a
  blocklist. Weekly it blocked an avg of **43.8K FQDNs** (Oct 2024–Jan 2025).
- **Critical weakness for circumvention:** the GFW **does NOT reassemble QUIC
  Initials split across >1 UDP datagram.** Chrome (since 2024-09-13) made Initials
  too large for one datagram → GFW less effective; Hysteria/sing-box/Xray added
  GFW-QUIC workarounds **almost immediately**. Firefox (Apr 2025) splits SNI to
  slip past. → **Gecko's handshake fragmentation directly defeats this.**
- Enforcement is **residual** (3-min block after a trigger; ~500ms to start;
  58% 3-tuple / 37% 4-tuple). Long-lived QUIC tunnels are disrupted; short-lived
  may complete in the uncensored window. **Salamander (full QUIC scramble) hides
  the SNI/Initial entirely → survives this regime**; plain QUIC does not.
- Researchers explicitly warn QUIC-reliant tools to "anticipate further GFW efforts
  to block QUIC and increased residual censorship." [gfw.report/usenixsecurity25;
  upb-syssec.github.io/blog/2025/quic-china; theregister 2025-08-04]

### (b) Hard UDP block / UDP throttling (common in China domestic + corporate)
- QUIC/Hysteria2/MASQUE-over-H3/WireGuard/AmneziaWG **all fail** — they need UDP.
- Survivors: **TCP-camouflage** (VLESS+Reality, ShadowTLS, AnyTLS) **or**
  **udp2tcp**: `dndx/phantun` masquerades UDP as a *fake* TCP stream through L3/L4
  NAT/firewalls, **12-byte overhead**, no real TCP retransmit/flow-control penalty
  (user-mode TCP state machine, 100% safe Rust). Standard pairing: WireGuard/
  AmneziaWG **over Phantun** for UDP-blocked nets. MASQUE's **HTTP/2 fallback** is
  the in-protocol analogue. [github.com/dndx/phantun; deepwiki phantun; mullvad]

### (c) SNI filtering / TLS-based DPI (TCP 443)
- Survivors: **VLESS+Reality** (real-site TLS fingerprint), **ShadowTLS** (real TLS
  handshake to a real host), **AnyTLS**. Salamander-obfuscated QUIC also has no SNI
  on the wire. Plain TLS proxies with a self-issued cert are detectable. [sing-box]

### (d) Active probing (Shadowsocks-class)
- GFW passively flags suspicious connections (partly by **packet-size**
  distribution), then **actively probes the server** to confirm. Mitigation:
  randomise packet sizes; AEAD Shadowsocks + probe-resistant front. Hysteria2's
  password auth + Salamander make active probing return nothing useful (no
  protocol response to an unauthenticated prober). [gfw.report/blog/gfw_shadowsocks;
  gfw.report/blog/ss_advise]

---

## 6. Hysteria2-primary vs MASQUE-primary — the tradeoff (current facts)

| Axis | **Hysteria2-primary** (+ Salamander/Gecko) | **MASQUE-primary** (RFC 9298, à la Mullvad) |
|---|---|---|
| Standardisation | Custom protocol, single main impl (apernet) + sing-box | **IETF RFC 9298**, multiple impls (masque-go, Cloudflare, Apple Private Relay) |
| Throughput on lossy/throttled links | **Best** — Brutal ignores loss, "bandwidth cheating" | Standard QUIC congestion control (fair, lower on lossy links) |
| Looks like normal web traffic | Salamander = random noise (hides QUIC, but *is* unusual UDP); Gecko fragments | **Strong** — genuinely *is* HTTP/3 to an HTTP proxy; blends with web |
| GFW QUIC-SNI (2024–25) | Survives via Salamander (no SNI) / Gecko (frag) | Survives **iff** the H3 proxy's own SNI is unblocked/fronted; otherwise SNI-filtered like any QUIC |
| Hard UDP block | Fails (needs UDP) — add Phantun | **Has in-protocol HTTP/2 fallback** (degraded but connects) |
| Self-host maturity | Very mature, one-click installers, low-RAM | Newer for self-host; masque-go exists; more moving parts |
| WireGuard compatibility | Tunnels arbitrary TCP/UDP incl. WG | **Designed to carry WireGuard UDP** (Mullvad's exact model) |
| Ecosystem clients | sing-box, mihomo, NekoBox, etc. | sing-box (less mature), Mullvad app, browsers' H3 stacks |

**Synthesis for a Mullvad-parity self-host:**
- **Primary** should be a **QUIC/UDP obfuscated transport**. Mullvad chose **MASQUE**
  because it is an **IETF standard that genuinely looks like HTTP/3 web traffic** and
  carries WireGuard UDP — strongest "blend in" + in-protocol HTTP/2 fallback.
- **Hysteria2 + Salamander/Gecko** beats MASQUE on **raw throughput over
  throttled/lossy links** and on **out-of-the-box GFW-QUIC evasion (Gecko frag)**,
  with far more mature self-host tooling — but it is a non-standard protocol whose
  obfuscation is *random noise* rather than *genuine web traffic*.
- **Neither survives a hard UDP block alone** → both need a TCP fallback
  (MASQUE: native HTTP/2 fallback; Hysteria2: pair with **Phantun udp2tcp** or a
  Reality/ShadowTLS sibling).
- **Recommended layered design:** MASQUE **or** Hysteria2/Gecko as QUIC primary →
  AmneziaWG for low-overhead WG-fingerprint evasion where UDP is allowed → Phantun
  (udp2tcp) + VLESS-Reality/ShadowTLS as the UDP-blocked / SNI-filtered fallback
  chain. This mirrors Mullvad's "auto-try obfuscation after failure" model.

---

## Sources verified
- https://v2.hysteria.network/docs/developers/Protocol/ — Hysteria2 protocol, Salamander/Gecko, QUIC RFC 9000 + datagram ext (accessed 2026-06-25, via WebSearch summary; direct fetch blocked)
- https://github.com/apernet/hysteria/releases — latest app/v2.9.2, 2026-05-23, Gecko + UDP ACL security fix (accessed 2026-06-25)
- https://v2.hysteria.network/docs/Changelog/ — Hysteria changelog (accessed 2026-06-25, via search)
- https://sing-box.sagernet.org/configuration/outbound/hysteria2/ — sing-box Hysteria2 (accessed 2026-06-25)
- https://sing-box.sagernet.org/changelog/ — sing-box changelog, AnyTLS Mar 2025, Gecko (accessed 2026-06-25)
- https://mullvad.net/en/blog/2025/9/9/introducing-quic-obfuscation-for-wireguard — Mullvad MASQUE/QUIC obfuscation, RFC 9298, v2025.9, 2025-09-09 (accessed 2026-06-25, via alternativeto mirror; direct fetch blocked)
- https://alternativeto.net/news/2025/9/mullvad-vpn-adds-quic-obfuscation-for-wireguard-to-bypass-censorship/ — Mullvad launch details + dates (accessed 2026-06-25)
- https://ietf-wg-masque.github.io/draft-ietf-masque-connect-udp/draft-ietf-masque-connect-udp.html — RFC 9298 CONNECT-UDP (accessed 2026-06-25)
- https://http.dev/masque — MASQUE explained, HTTP/2 fallback requirement (accessed 2026-06-25)
- https://github.com/quic-go/masque-go — masque-go reference impl RFC 9298 (accessed 2026-06-25)
- https://docs.amnezia.org/documentation/amnezia-wg/ — AmneziaWG obfuscation params (accessed 2026-06-25)
- https://dev.to/bivlked/amneziawg-20-self-host-an-obfuscated-wireguard-vpn-that-bypasses-dpi-4692 — AmneziaWG 2.0 junk packets, 148/92-byte fingerprint, ~3% perf (accessed 2026-06-25)
- https://deepwiki.com/amnezia-vpn/amneziawg-go — AmneziaWG obfuscation features (accessed 2026-06-25)
- https://gfw.report/publications/usenixsecurity25/en/ — USENIX Sec '25 SNI-based QUIC censorship of GFW (accessed 2026-06-25)
- https://upb-syssec.github.io/blog/2025/quic-china/ — GFW QUIC-Initial SNI key derivation, residual censorship 3-tuple/4-tuple, timeline (accessed 2026-06-25)
- https://www.theregister.com/2025/08/04/china_great_firewall_quic_security_flaws/ — GFW QUIC upgrade analysis (accessed 2026-06-25)
- https://gfw.report/blog/gfw_shadowsocks/ — Shadowsocks active probing + packet-size detection (accessed 2026-06-25)
- https://gfw.report/blog/ss_advise/en/ — defend against GFW active probing (accessed 2026-06-25)
- https://github.com/dndx/phantun — Phantun udp2tcp, 12-byte overhead, fake-TCP (accessed 2026-06-25)
- https://deepwiki.com/dndx/phantun — Phantun internals (accessed 2026-06-25)
- https://www.wireguard.com/known-limitations/ — WireGuard UDP-only limitation (accessed 2026-06-25)
