# HelixVPN MVP — Source-Coverage Ledger (§11.4.118 proof-of-completeness)

**Revision:** 1
**Last modified:** 2026-06-25T00:00:00Z
**Status:** active
**Authority:** Constitution §11.4.118 (discovery-pressure / enumerated-coverage), §11.4.6 (no-guessing — gaps surfaced, never hidden)
**Scope:** Enumerated proof that every one of the 16 source research documents under
`docs/research/mvp/` (11 LLM analyses + 5 `04_VPN_CLD` refined docs) was read and that its
distinctive content is accounted for in the `final/` specification set — OR is named as a
gap. Evidence base: `scratchpad/kb/SYNTHESIS.md` + citation-grep across all 13 produced
`final/*.md` docs.

---

## How to read this ledger

- **Source id** — the canonical id used in citations (`[04_ARCH §3]`, `[02_QWN]`, …).
- **Absorbed into** — which `final/` doc(s) carry its content (verified by source-id
  citation grep + topical reconciliation; the dominant landing site is **bolded**).
- **Unique contribution** — the distinctive idea(s) this source brought that the others did
  not (or brought first / most sharply).
- **Where it landed** — the concrete section/decision the contribution shaped.

`final/` doc shorthand: **00** product-scope · **01** data-plane · **02** control-plane ·
**03** client-core+UI · **04** security/privacy/PKI · **05** repo-layout/tooling/ecosystem ·
**06** Phase-0 WBS · **07** Phase-1 WBS · **08** Phase-2 WBS · **09** Phase-3 WBS ·
**10** testing/QA · **11** deep-research appendix · **SPEC** SPECIFICATION.md (master index).

---

## Coverage table — one row per source

| Source id | Absorbed into (final/) | Unique contribution | Where it landed |
|---|---|---|---|
| **00** (VPN_Initial_Res) | **00**, 01, 02, 03, 04, **06**, **07**, 08, 09, 10, 11, SPEC | The founding constraint: 1 user → **N** joined private networks, **no inbound port-forward** (internal hosts dial outbound; reverse tunnel). Original problem statement + glossary all docs anchor to. | Product floor (00 §Problem), reverse-tunnel hub-and-spoke (01), threaded as the requirement source through every WBS. |
| **01_DSK** | 00, 02, 11, SPEC | **Go-core** camp (D2 Camp B): Go client core reusing Hysteria2's Go impl — the simpler, single-language alternative to the Rust core. Lean tunnel-first MVP stance. | Decision **D2** (client-core language) Camp B in 11-appendix + 00; lean-MVP argument feeding Phase-0/Phase-1 scoping. |
| **02_QWN** | 00, 02, 03, 09, 10, 11, SPEC | Detailed control-plane modeling; **Redis Streams-for-MVP → NATS-at-scale** staging (D3 Camp A); event-bus + real-time push depth. | Decision **D3** (event bus, Redis MVP) in 02 + 11; control-plane module decomposition (02). |
| **03_ZAI** | 00 (1×), 11 (1×); DR/KMS content → **04**, 08 | **KMS-encrypted automated backups**, **Terraform disaster-recovery (RTO < 15 min)**, GitOps config management, multi-arch CI/CD. Enterprise resilience/DR framing. | KMS backups + RTO in **04-security** (RTO present); HA/multi-region + GitOps in 08-Phase-2. CI/CD content deliberately dropped per §11.4.156 (no active CI). See gap G1. |
| **05_YBO** | **00**, 01, 02, 03, 04, 05, 07, 09, 10, 11, SPEC | The **operator mandate**: Go + Gin + PostgreSQL + Redis + rootless Podman stack floor; full platform list (iOS, Android, Aurora OS, HarmonyOS NEXT, Windows, Linux, macOS, Web); obfuscation mandatory "especially QUIC"; three apps. | Settled stack floor (00 §Stack), platform matrix (03 + 09), constitution-binding alignment (§11.4.76/.161). The most-cited mandate after 00/04_ARCH. |
| **06_GRK** | 00 (1×), 11 (1×); protocol/hardening content → 01, 04, 08 | **sing-box** as the universal transport-multiplexer framework; **KMP (Kotlin Multiplatform)** client-core alternative; detailed Podman-rootless firewall/netns hardening scripts; advanced Hysteria2 + Salamander configs. | KMP captured as a considered alternative in **03**/11; Hysteria2/Salamander → D1 Camp B (01/08); rootless hardening → 04. **sing-box framework itself not adopted** (custom Rust helix-transport chosen instead) — see gap G2. |
| **07_GMI** | 00, 03, 08, 09, 10, 11, SPEC | **CGNAT 100.64/10 1:1-per-network** answer to the IP-subnet-collision problem (D4 Camp B); Go-core leaning (D2 Camp B with DSK). | Decision **D4** (subnet-collision) Camp B surfaced in 11 + IPAM discussion (02/08); one of only three sources that actually solved D4. |
| **08_CPL** | 00 (2×), 11 (1×); connector mechanics → 01 | Pragmatic **minimal connector implementation**: read routes from a file, enable forwarding, NAT LAN → tunnel — the smallest-viable network-side agent. | Subsumed into the Connector data-path + route-advertisement design in **01-data-plane**; no distinctive idea left unreflected (fully generalized). |
| **09_GCT** | 00 (1×), 11 (1×); Go-only + SSE → 02, 03 | **Go-as-single-language** stance; server-side kill-switch; **SSE** real-time channel; **Fyne** desktop UI toolkit; Postgres schema sketch (Russian-language source). | Go-only → D2 Camp B (11); SSE → real-time transport options (02 control-plane offers WS/SSE); server-side kill-switch reflected. **Fyne explicitly overruled by Flutter** and not recorded as a considered-then-rejected option — see gap G3. |
| **10_KMI** | 00, 02, 03, 08, 09, 10, 11, SPEC | **NATS-JetStream-from-start** (D3 Camp B); **CGNAT-per-network** (D4 Camp B, with GMI); full "Connectivity-OS" ambition (D7 Camp B). | Decisions **D3** Camp B + **D4** Camp B + **D7** Camp B in 11; scale/HA NATS path in 08-Phase-2; ambition framing in 00. |
| **11_MST** | 00, 01, 02, 03, 08, 09, 10, 11, SPEC | **Asymmetric per-leg transport** (D6): Hysteria2/QUIC user↔gateway, WireGuard gateway↔networks — best-fit protocol per leg. Largest/most-detailed analysis. | Decision **D6** (transport topology) in 11 + 01-data-plane (per-leg transport design); woven through Phase-2 transport set (08). |
| **04_ARCH** (CLD refined) | **00**, **01**, 02, 03, **04**, 05, 06, 08, 09, 10, **11**, SPEC | The settled **architecture spine**: pitch, WireGuard-as-crypto-core + pluggable-obfuscation-beneath-WG, **MASQUE/QUIC = WG-over-HTTP-3** (D1 Camp A = true Mullvad parity), **IPv6 ULA /48 + 4via6** (D4 Camp A), repo layout, security invariants, no-logging-as-code. | The dominant citation across the whole set (50× in 00, 39× in 01, 32× in 04, 67× in 11). Anchors D1 Camp A, D4 Camp A, the repo layout (05), and the security model (04). |
| **04_UI** (CLD refined) | 00, **03**, 05, 09, 10, 11, SPEC | **`helix-core` (Rust) + `helix-ui` (Flutter)** shared-codebase strategy; `helix_design` system + signature components; three flavors from one tree (`runHelixApp`); Melos monorepo; per-platform `TunnelPlatform` shim. | The client architecture core of **03-client-core-and-ui**; design-system + platform-shim sections; Phase-3 platform builds (09). |
| **04_P0** (CLD refined) | 00, **01**, 03, **06**, 07, **10**, 11, SPEC | **Phase-0 spike** with exit gates **G1–G6** (G3 iOS NE memory ceiling = make-or-break; G4 Go-vs-Rust edge benchmark); surviving interfaces (`Transport` trait, `helix-wg`, orchestrator, FFI); Linux-netns + nftables-DPI + tc-netem test rig. | The spine of **06-Phase-0-WBS**; gate definitions also seed 10-testing; decision **D5** (edge language) gate G4 in 01/11. |
| **04_P1** (CLD refined) | 00, **02**, 03, 04, 05, **07**, 08, 09, 10, 11, SPEC | **Phase-1 MVP**: Go modular monolith module list; Postgres+RLS data model (no connection/traffic tables, CI-lint-enforced); protobuf `Coordinator` + server-streaming `WatchNetworkMap` (p99 <1s); policy compiler (default-deny → AllowedIPs/nftables); **8-criterion MVP DoD**. | The spine of **07-Phase-1-WBS** + **02-control-plane**; the 8 acceptance criteria → 10-testing; no-logging schema lint → 04-security. |
| **04_P2** (CLD refined) | 00, 01, 04, **08**, 09, 11, SPEC | **Phase-2 parity**: full transport set (+Shadowsocks, UDP-over-TCP, hardened LWO, auto-ladder); **DAITA via maybenot**; **direct P2P + NAT traversal + DERP-style relay**; **multi-hop**; **post-quantum** (ML-KEM hybrid / Rosenpass); HA multi-region. | The spine of **08-Phase-2-WBS**; PQ + DAITA → 04-security; multi-region/HA → 08 + Patroni/NATS references. |

---

## Decision-coverage cross-check (the 7 D-decisions are all surfaced)

| Decision | Camp A source(s) | Camp B source(s) | Surfaced in final/ |
|---|---|---|---|
| D1 obfuscating transport | 04_ARCH/04_P0 (MASQUE/QUIC) | plurality of 10 (Hysteria2+Salamander) | 01, 08, 11 |
| D2 client-core language | 04_*/QWN/ZAI/CPL/KMI/MST (Rust) | DSK/GMI/GCT (Go) | 03, 11 |
| D3 event bus | 04_P1/QWN/MST (Redis→NATS) | DSK/KMI (NATS-from-start) | 02, 08, 11 |
| D4 subnet collision | 04_ARCH (ULA/4via6) | GMI/KMI (CGNAT 100.64/10) | 02, 08, 11 |
| D5 edge language | 04_P0 (Rust) | (Go quic-go) | 01, 06 (gate G4), 11 |
| D6 transport topology | single-protocol | 11_MST (asymmetric per-leg) | 01, 11 |
| D7 MVP ambition | QWN/DSK/04_P0 (lean) | KMI/ZAI (Connectivity-OS) | 00, 11 |

All seven decisions from SYNTHESIS §3 are present as explicit decisions-with-recommendations.

---

## Coverage gaps

The following distinctive source ideas are **not yet fully reflected** in `final/`. They are
surfaced here per §11.4.6/§11.4.118 (no silent omission) rather than hidden. Each is tagged
**deliberate-divergence** (consciously not adopted; rationale exists) or **open-gap** (should
be added/recorded in a future revision).

- **G1 — [03_ZAI] Disaster-recovery completeness — partial / open-gap.**
  ZAI's KMS-encrypted automated backups landed (04, 02, 07) and an RTO target appears in
  04-security, but ZAI's *named* DR posture — explicit **RTO/RPO budget**, Terraform-driven
  region-failover runbook, and "disaster recovery" as a first-class operational section — is
  not consolidated anywhere; the pieces are scattered. **Recommendation:** add an explicit
  HA/DR + RTO/RPO subsection to 08-Phase-2 (or 05-tooling) citing `[03_ZAI]`.

- **G2 — [06_GRK] sing-box transport framework — deliberate-divergence (record the rationale).**
  GRK proposed **sing-box** as the universal transport-multiplexer (Hysteria2/Shadowsocks/
  TUIC/etc. via one Go framework). The protocols GRK wanted are reflected (Hysteria2 → D1
  Camp B; Shadowsocks + UDP-over-TCP → 08 Phase-2 transport set), but the **sing-box framework
  itself was not adopted** — the spec chose a custom Rust `helix-transport` crate (04_ARCH).
  This is a conscious divergence, but the *reason sing-box was rejected* is **not written down**
  in any final/ doc. **Recommendation:** add a one-paragraph "considered-and-rejected: sing-box
  (Go framework conflicts with Rust-core D2/iOS-memory-ceiling)" note to 01-data-plane or the
  11-appendix.

- **G3 — [09_GCT] Fyne desktop UI — deliberate-divergence (rejected alternative not recorded).**
  GCT proposed **Fyne** for the desktop/web client. Flutter (04_UI) won unanimously and Fyne
  is correctly absent from the build plan, but it is **not recorded as a considered-then-rejected
  option**, so a future reader cannot see it was evaluated. **Recommendation:** one line in
  03-client-core-and-ui's UI-toolkit decision noting Fyne/`[09_GCT]` was evaluated and dropped
  in favor of Flutter (cross-platform mobile reach + shared design system).

No source is **wholly** unabsorbed: every one of the 16 ids is cited in at least `00` and the
`11` deep-research appendix, and the three gaps above are scoped, named, and recommendation-
tagged. The thinly-cited sources (03_ZAI, 06_GRK, 08_CPL, 09_GCT) are the "full-ecosystem /
Go-or-sing-box camp" whose protocol/ops ideas were largely **generalized** into the chosen
Rust-core + custom-transport design rather than carried verbatim.

---

## Provenance of this ledger

- Synthesis evidence base: `scratchpad/kb/SYNTHESIS.md` (digests of all 16 sources).
- Citation grep across all 13 `final/*.md` docs (source-id frequency table, 2026-06-25).
- Distinctive-term presence check (`sing-box`, `KMS`, `Fyne`, `KMP`, `RTO`, `GitOps`,
  `Terraform`, `Patroni`, `reproducible build`) confirming G1–G3.
