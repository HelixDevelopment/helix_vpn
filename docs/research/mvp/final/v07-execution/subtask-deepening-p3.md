# Phase 3 (Extended Reach) Subtask Deepening — epic → task → subtask (closes R5)

**Revision:** 2
**Last modified:** 2026-07-04T12:00:00Z
**Rev 2:** Independent gap-analysis pass — structure and honesty stance
(`PENDING_DEVICE:`/`Operator-blocked` vocabulary) verified consistent with
`09-phase3-reach-wbs.md` and the sibling Phase-1/Phase-2 deepening docs. No
contradictions found.

> Volume 7 (Phase Execution), document 5 of 5. Companion to
> `subtask-deepening-p1.md` / `-p2.md`, applying the R5 deepening
> (`REFINEMENT_NOTES.md`) to Phase 3: every task `HVPN-P3-NNN` in
> `09-phase3-reach-wbs.md` is decomposed into PR-sized subtasks `HVPN-P3-NNN.k`
> (§11.4.93/.54), each with a concrete **acceptance** (falsifiable,
> captured-evidence §11.4.5/.69/.107), the **§11.4.169 test types** (the `09-…`
> §1 vocabulary incl. the Phase-3-specific `REPRO`), and an **estimated
> complexity** (engineer-day T-shirt, **widest error bars in the programme**,
> sizing-only `TARGET`, §11.4.6). **Spec-only.** Phase 3 is the one phase where
> *"ship what's provable, flag the rest honestly"* is the correct stance (`09-…`
> §13): the two native shims (E21 HarmonyOS, E22 Aurora) are real native work on
> toolchains the project does not yet own hardware/CI for, so device-gated
> acceptances are marked **`PENDING_DEVICE:`** (§11.4.3 `hardware_not_present`)
> and engagement-gated ones **`Operator-blocked`** (§11.4.21/.101) — never a
> faked pass. Every subtask traces to its parent task in `09-…`. These `.k` rows
> feed the `workable-items` DB (`workable-items-model.md` §7).

---

## Table of contents

- [0. Deepening conventions + the honesty stance](#0-deepening-conventions--the-honesty-stance)
- [1. E20 — Reach CI fabric (gate G20)](#1-e20--reach-ci-fabric-gate-g20)
- [2. E21 — HarmonyOS NEXT build (G21, highest risk)](#2-e21--harmonyos-next-build-g21-highest-risk)
- [3. E22 — Aurora OS build (G22)](#3-e22--aurora-os-build-g22)
- [4. E23 — WASM browser-scoped MASQUE proxy (G23)](#4-e23--wasm-browser-scoped-masque-proxy-g23)
- [5. E24 — Billing-optional multi-tenant (G24)](#5-e24--billing-optional-multi-tenant-g24)
- [6. E25 — Third-party security audit (G25)](#6-e25--third-party-security-audit-g25)
- [7. E26 — Reproducible builds (G26)](#7-e26--reproducible-builds-g26)
- [8. E27 — Reach l10n, governance, release](#8-e27--reach-l10n-governance-release)
- [9. Subtask roll-up + honest-gap ledger](#9-subtask-roll-up--honest-gap-ledger)
- [Sources verified](#sources-verified)

---

## 0. Deepening conventions + the honesty stance

Columns as in `-p1`/`-p2` §0: `id` (`HVPN-P3-NNN.k`) · **Subtask** (≥6 words
§11.4.91) · **Acceptance** · **Tests** (§11.4.169 codes, `09-…` §1 vocab incl.
`REPRO`) · **Cx** (XS/S/M/L engineer-days, sizing `TARGET`).

**Phase-3 honesty stance (§11.4.6, `09-…` §0/§13).** Where a subtask's acceptance
genuinely cannot be proven without the target device or an external engagement,
the acceptance carries the literal `PENDING_DEVICE:` (device-gated, §11.4.3
`hardware_not_present`) or the item's status is `Operator-blocked` (engagement /
spend, §11.4.21/.101) with the §11.4.148 D3 unblock condition. A gate is **never**
marked `pass` on metadata/config-only evidence; the runtime signature (§11.4.108)
for the native shims is asserted on a **clean install on the real device** — an
APK/HAP/RPM that *builds* is not *done*. No active CI (§11.4.156): the "fork
runners" are local/self-hosted rootless-Podman build pods, never GitHub/GitLab
workflows.

---

## 1. E20 — Reach CI fabric (gate G20)

*Self-hosted rootless Podman build pods (§11.4.76/.161/.156); a fork lag never blocks mainline.*

**HVPN-P3-200 — Pin OpenHarmony-SIG Flutter fork + DevEco toolchain in a pod** (`09-…` §5; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `harmonyos.container` quadlet building a HAP from the pinned `flutter_flutter` `ohos` SHA | `podman build` produces a HAP from the sample app; toolchain SHA recorded | UNIT,FA | M |
| `.2` | DevEco signing material mounted read-only as secrets (§11.4.10) | secrets never in the image layer (SEC leak audit green) | SEC | S |
| `.3` | REPRO: same SHA → same toolchain digest | identical toolchain digest on rebuild (REPRO) | REPRO | S |

**HVPN-P3-201 — Pin OMP Aurora `flutter-aurora` fork + RPM signing pod** (`09-…` §5; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `aurora.container` quadlet producing a signed Aurora RPM | signed RPM of the sample app builds; `rpm -K` verifies the signature | UNIT,FA | M |
| `.2` | Mirror-pin the Russian-hosted SDK (§11.4.77 regen) + isolated egress allow-list | SDK tarball SHA-256 matches the recorded manifest; egress allow-list holds | SEC,REPRO | S |

**HVPN-P3-202 — Reach release-artifact registry + provenance manifest** (`09-…` §5; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Signed `reach_artifacts.json` (source SHA + toolchain digest + SBOM pointer per HAP/RPM/.wasm) | manifest entry per build, schema-validated, links resolve | UNIT,FA | S |

**HVPN-P3-203 — G20 readiness certification** (`09-…` §5; XS)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `make reach-ci-gate` builds+signs both fork artefacts, emits the G20 verdict | HAP + signed RPM both produced from a clean checkout in two `-count=2` runs; gate `outcome=pass` | FA,CHAL | XS |

---

## 2. E21 — HarmonyOS NEXT build (G21, highest risk)

*ArkTS VPN ability + NAPI bridge to the Rust `helix-core` `.so`; the UI ports for free, the tunnel ability is real native work.*

**HVPN-P3-210 — `helix-core` cross-compile to OpenHarmony + NAPI surface** (`09-…` §6; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Build `helix-ffi` for `aarch64-unknown-linux-ohos`; emit `libhelix_ohos.so` NAPI module (re-exports the platform-neutral core, §11.4.28) | `.so` loads in an ArkTS test harness; `helix_version()` round-trips | UNIT,INT | M |
| `.2` | `helix_start(fd,cfg)`/`helix_stop`/`helix_subscribe` NAPI surface over the unchanged core | NAPI surface callable from ArkTS (INT) | UNIT,INT | M |
| `.3` | MEM/BENCH on emulator | RSS ceiling sampled (`PENDING_DEVICE:` on hardware); throughput benched on emulator | MEM,BENCH | M |

**HVPN-P3-211 — Network Kit `VpnExtensionAbility` + lifecycle + kill-switch** (`09-…` §6; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | ArkTS ability: VPN consent → `VpnConnection` (tun fd/routes/DNS, MTU 1280) → hand fd to core | ability creates the tunnel fd + drives the core (INT) | UNIT,INT | M |
| `.2` | Drive connect/disconnect/kill-switch/auto-escalate from the core status stream | `PENDING_DEVICE:` enroll→UP→`curl` reaches authorized LAN host on a HarmonyOS NEXT device; window-scoped MP4 + DevEco heap capture; §11.4.3 SKIP `hardware_not_present` until G20 provisions a device | E2E,MEM,UX,REC | L |
| `.3` | Tunnel-drop blanks plaintext (kill-switch per §04) | `PENDING_DEVICE:` kill-switch blanks plaintext on drop (device capture) | SEC,CHAL | M |

**HVPN-P3-212 — Flutter HAP build (Access + Connector, OHOS channel)** (`09-…` §6; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Signed HAPs via `runHelixApp(flavor,…)` on the SIG fork; ArkTS↔Dart MethodChannel wired | app installs + drives the shim; golden UI tests pass on the OHOS theme | UI | M |
| `.2` | UX walkthrough MP4 (`PENDING_DEVICE`) | UX MP4 vision-verified (§11.4.159), `PENDING_DEVICE:` until a device | UX,REC | S |

**HVPN-P3-213 — DevEco signing + market metadata + l10n (zh-Hans first-tier)** (`09-…` §6; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | DevEco signing profile + AppGallery metadata + `intl_zh.arb` first-tier locale | signed HAP verifies; zh-Hans strings render with no overflow (UI golden) | FA,UI | S |

**HVPN-P3-214 — G21 device certification + honest gap doc** (`09-…` §6; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `challenges`/`helix_qa` bank: enroll→UP→reach→kill-switch on device, PASS only on captured evidence (§11.4.27/.107) | gate `outcome=pass` with device evidence, **OR** `outcome=pending_device` with the §11.4.148 D3 unblock condition (provision a HarmonyOS NEXT device + add to the §11.4.128 tracked-device set) — never a faked pass | E2E,FA,CHAL,REC | M |

---

## 3. E22 — Aurora OS build (G22)

*OMP Russia Flutter fork → signed RPM; tunnel backend in Qt/C++ linking `helix-core` as a C library. Enterprise/government SKU.*

**HVPN-P3-220 — `helix-core` C ABI for Aurora + cbindgen header** (`09-…` §7; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Build `helix-ffi` for the `aarch64`/`armv7hl` Sailfish target; `cbindgen` → `helix.h` (no Aurora specifics inside the core, §11.4.28) | header compiles in a Qt test; `helix_version()` round-trips | UNIT,INT | M |
| `.2` | MEM/BENCH on the Aurora target | RSS sampled (`PENDING_DEVICE:`); throughput benched | MEM,BENCH | M |

**HVPN-P3-221 — Qt/C++ `tun` lifecycle + Friflex Flutter bridge + kill-switch** (`09-…` §7; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Open/route/DNS the Sailfish `tun`; `helix_start` over the C ABI; status marshalled onto the Qt event loop | the backend drives the core; status events surface (INT) | UNIT,INT | M |
| `.2` | Friflex-style plugin bridge connect/disconnect/status to Flutter + kill-switch/auto-escalate | `PENDING_DEVICE:` enroll→UP→reach on an Aurora device; kill-switch blanks plaintext; MP4 + RSS capture; §11.4.3 SKIP until a device | E2E,MEM,UX,REC,CHAL | L |

**HVPN-P3-222 — Flutter signed-RPM build (Access + Connector, OMP fork)** (`09-…` §7; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Signed Aurora RPMs of the two flavors + `ru` first-tier l10n | `rpm -K` verifies; `ru` strings render with no overflow; install on the Aurora emulator (`PENDING_DEVICE:` for hardware reach) | UI,UX,REC,SEC | M |

**HVPN-P3-223 — G22 device certification + enterprise-SKU ops doc** (`09-…` §7; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Bank entry enroll→UP→reach→kill-switch on Aurora + ops doc (isolated runner + Mos.Hub signing §11.4.10) | gate `outcome=pass` with device evidence, **OR** honest `pending_device`; toolchain-provenance recorded `UNCONFIRMED:` until an Aurora device is in hand (§11.4.6) | E2E,FA,CHAL,REC | M |

---

## 4. E23 — WASM browser-scoped MASQUE proxy (G23)

*Browser-scoped proxy of the browser's own traffic — NOT a system-wide tunnel; the reuse pillar: `helix-transport` → wasm32 over WebTransport.*

**HVPN-P3-230 — `helix-transport` wasm32 build + WebTransport binding** (`09-…` §8; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Feature-gate `helix-transport` to compile to `wasm32-unknown-unknown`; bind browser WebTransport for the H3/datagram path (same MASQUE state machine, only the I/O leaf differs) | `.wasm` loads in headless Chromium; a MASQUE `CONNECT-UDP` session establishes to a test edge | UNIT,INT | L |
| `.2` | Throughput benched | proxy MB/s captured (BENCH) | BENCH | S |

**HVPN-P3-231 — Service-worker / fetch-shim integration + scope isolation** (`09-…` §8; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Service worker routing *only* explicitly-proxied origins through MASQUE; everything else normal browser networking | a non-proxied origin never touches the gateway (SEC, captured) | UNIT,INT,E2E | M |
| `.2` | Hard scope: only policy-authorized hosts (server-side need-to-know); die on tab/worker close | a non-authorized host is unreachable; tab-close terminates the session, no lingering tunnel (SEC) | SEC | M |

**HVPN-P3-232 — Console UX: explicit "browser-scoped, not a system VPN" affordance** (`09-…` §8; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `ShieldIndicator` variant + copy stating the limitation plainly; per-network toggle; live reachability | UX MP4 vision-verified that the scope wording is present + unambiguous (§11.4.159) — guards an over-claim bluff | UI,UX,REC | S |

**HVPN-P3-233 — G23 certification + threat note (WASM origin model)** (`09-…` §8; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Bank: browser reaches an authorized host, denied an unauthorized one, leaks nothing beyond scope; threat note (short-lived token, no WG private key in browser) | gate `outcome=pass` with captured evidence; the "no key in browser" invariant proven by a SEC assertion | E2E,SEC,CHAL | S |

---

## 5. E24 — Billing-optional multi-tenant (G24)

*Billing must not break no-logging — metering is aggregate counters only; billing OFF by default = zero behaviour change.*

**HVPN-P3-240 — Billing schema (plans/subs/entitlements/aggregate meters, RLS + no-log lint)** (`09-…` §9; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | FORCE-RLS tenant-scoped billing tables; `usage_counter` daily-aggregate only (no src/dst/host/port/flow, no sub-day ts) | cross-tenant read denied (RLS); aggregate schema passes the lint | UNIT,INT,SEC | M |
| `.2` | Extend the no-log CI lint to reject a billing column matching the traffic-log shape + paired §1.1 mutation | the lint FAILs a planted `usage_flows(src,dst,port,ts)` table, passes the aggregate schema | SEC | S |

**HVPN-P3-241 — Aggregate metering pipeline (edge counters → daily upsert)** (`09-…` §9; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Metering consumer folds edge rx/tx counters into `usage_counter` daily upserts (idempotent by tenant/device/day) | exactly-once bucket under a SIGKILL mid-fold (CHAOS) | UNIT,INT,CHAOS | M |
| `.2` | Bounded-memory at scale | 10k devices, bounded memory (STRESS) | STRESS | S |

**HVPN-P3-242 — Entitlement enforcement in the coordinator** (`09-…` §9; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Plan limits → admission checks at enroll + map build; over-quota = deny with reason (D-P3-2 fail-open) | an over-device-quota enroll is denied with a clear reason; bytes-quota breach degrades per the chosen policy | UNIT,INT,SEC | M |
| `.2` | PERF: check adds < 1 ms to map build | captured map-build delta < 1 ms | PERF | S |

**HVPN-P3-243 — Optional payment-provider adapter (interface + one reference impl, flagged)** (`09-…` §9; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Narrow `PaymentProvider` interface + one reference adapter (provider behind a flag, no name in core §11.4.28) | billing-OFF path never calls the adapter (captured) | UNIT,INT,SEC | M |
| `.2` | Public webhook: signature-verify + replay-reject + rate-limit (first public surface → DDOS applicable) | webhook signature verified; replay rejected; rate-limit holds under flood (DDOS) | DDOS,SEC | S |

**HVPN-P3-244 — G24 certification (billing ON works, OFF is a perfect no-op, no-log preserved)** (`09-…` §9; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Bank: (a) billing-ON meter+quota+payment flow; (b) billing-OFF byte-identical to pre-E24 (§11.4.108 runtime-signature diff); (c) no-log lint still green + planted traffic-log still FAILs | gate `outcome=pass`; the no-op proof captured as a §11.4.135 regression guard | E2E,SEC,FA,CHAL | S |

---

## 6. E25 — Third-party security audit (G25)

*A self-audit is not an audit; the verdict is the external party's. Engagement is operator-gated (spend → §11.4.101/.66).*

**HVPN-P3-250 — Audit scope, threat model, asset inventory** (`09-…` §10; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | STRIDE per surface (crypto core, control plane, the four reach surfaces, no-log invariant) | every shipped surface mapped to ≥1 threat + ≥1 audit objective; no surface unlisted (§11.4.118) | SEC | M |

**HVPN-P3-251 — SBOM + reproducible-source snapshot for the auditor** (`09-…` §10; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | CycloneDX SBOM per artefact + a pinned rebuildable source snapshot | SBOM covers 100% of shipped deps; snapshot SHA recorded; `endor`/`aikido`-class scan green or every finding triaged | SEC,REPRO | S |

**HVPN-P3-252 — Engage auditor + run the engagement (operator-gated)** (`09-…` §10; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Select + contract an independent firm (operator decision, §11.4.66 `AskUserQuestion`), share scope+snapshot, receive the report | report received; status `Operator-blocked` with the §11.4.21 unblock detail until the engagement is funded — honest, not faked | SEC,CHAL | L |

**HVPN-P3-253 — Remediation loop to zero Critical/High (§11.4.134/.146)** (`09-…` §10; L)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Per finding: §11.4.102 root cause → §11.4.115 RED-on-vulnerable → fix → §11.4.146 extend → permanent §11.4.135 guard | zero unresolved Critical/High; each closure carries a falsifiable regression test (no console-mark-is-fixed bluff) | UNIT,INT,SEC,CHAL,REC | L |
| `.2` | Re-audit fixed items until the firm signs off (§11.4.134 iterate-to-GO) | the firm signs off after re-audit | SEC,CHAL | M |

**HVPN-P3-254 — G25 certification + public disclosure** (`09-…` §10; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Publish the report + remediation summary; gate verdict | gate `outcome=pass` only when Critical/High = 0 + report published; else honest `fail`/`operator_blocked` with the open-finding list | SEC,CHAL | S |

---

## 7. E26 — Reproducible builds (G26)

*Table stakes for a privacy product: a user must verify the binaries match the source.*

**HVPN-P3-260 — Determinize Go + Rust builds** (`09-…` §11; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Pin toolchains, `-trimpath`/`--remap-path-prefix`, fix `SOURCE_DATE_EPOCH`, strip non-determinism, build in a pinned `containers` image | two clean rebuilds of `helix-go`+`helix-edge`+`helix-core` on two hosts → identical SHA-256 (REPRO) | REPRO,FA | M |

**HVPN-P3-261 — Determinize the Flutter app builds** (`09-…` §11; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Reproducible Flutter AOT (iOS/Android/desktop/web) + fork HAP/RPM; deterministic asset + ARB ordering | two rebuilds of each flavor → identical artefact (signature stripped before compare); divergences root-caused (§11.4.102) to zero | REPRO,FA | M |
| `.2` | Honest boundary: signed installers differ by signature → compare the unsigned payload digest | the honest boundary recorded (§11.4.6); unsigned-payload digests identical | REPRO | S |

**HVPN-P3-262 — WASM + native-shim reproducibility** (`09-…` §11; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Pin OHOS/Aurora cross-toolchains; rebuild `.wasm` + `libhelix_ohos.so` + Aurora `libhelix.a` | two rebuilds → identical SHA-256, **OR** `PENDING_TOOLCHAIN:` honestly where a fork toolchain is not yet deterministic (a finding, not a pass) | REPRO | M |

**HVPN-P3-263 — Public verification script + transparency doc** (`09-…` §11; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `verify-reproducible.sh` + `reproducible_builds.md` (model + honest signature exceptions) | a third party (the §11.4.165 independent verifier) runs it + confirms identity on every non-signature artefact (captured) | REPRO,FA,CHAL | S |

**HVPN-P3-264 — G26 certification** (`09-…` §11; XS)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Gate verdict over the full artefact matrix | gate `outcome=pass` only when every non-signature artefact reproduces bit-identically on two independent hosts; any `PENDING_TOOLCHAIN` keeps the gate honestly open | REPRO,FA,CHAL | XS |

---

## 8. E27 — Reach l10n, governance, release

**HVPN-P3-270 — First-tier `ru` + `zh-Hans` l10n completeness + overflow audit** (`09-…` §12; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Complete `helix_l10n` ARBs for `ru`/`zh-Hans`; no overflow/overlap (§11.4.162); state announced not just colored (a11y, §04) | golden UI green in both locales on Access/Connector/Console; UX MP4 vision-verified | UI,UX,REC | M |

**HVPN-P3-271 — Phase-3 feature Status + Status_Summary set (§11.4.153) with per-surface video confirmation** (`09-…` §12; M)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | `docs/features/Status.md` + `Status_Summary.md` enumerating every Phase-3 surface (Implementation/Wiring/Validation/Video-confirmation), four-format export (HTML+PDF+DOCX), docs-chain-synced | every confirmed row has a real-use MP4 (`PENDING_DEVICE` rows honestly marked); §11.4.86 fingerprint matches | REC,CHAL | M |

**HVPN-P3-272 — docs-chain sync + items-DB load + reproducible-release tag** (`09-…` §12; S)

| id | Subtask | Acceptance | Tests | Cx |
|---|---|---|---|---|
| `.1` | Load all `HVPN-P3-*` rows into `docs/workable_items.db` (§11.4.93/.95), sync exports (§11.4.65), cut `<prefix>-3.0.0-…` (§11.4.151) + fan out via merge-onto-latest-main (§11.4.113, never force-push) | `workable-items validate` green; tag prefix correct across main + every owned submodule; artefacts reproduce (E26) | FA,REPRO | S |

---

## 9. Subtask roll-up + honest-gap ledger

| Epic | Parent tasks | Deepened subtasks | Honest-gap class |
|---|---|---|---|
| E20 reach CI fabric | 4 | 7 | toolchain access (medium) |
| E21 HarmonyOS NEXT | 5 | 11 | **`PENDING_DEVICE` — make-or-break** |
| E22 Aurora OS | 4 | 6 | **`PENDING_DEVICE` + Russian toolchain `UNCONFIRMED`** |
| E23 WASM proxy | 4 | 6 | browser API maturity (medium) |
| E24 billing-optional | 5 | 9 | no-log invariant must hold |
| E25 third-party audit | 5 | 7 | **`Operator-blocked` — external spend** |
| E26 reproducible builds | 5 | 7 | fork-toolchain `PENDING_TOOLCHAIN` |
| E27 reach l10n/release | 3 | 3 | low |
| **Total** | **35** | **56** | — |

**Honest-gap ledger (§11.4.6, `09-…` §13/§14 risk register).** Phase 3 is the
phase where some acceptances cannot be self-proven; the deepening preserves that
honesty in the subtask acceptances rather than hiding it:

| Subtask | Gap | Honest marker | Unblock (§11.4.148 D3) |
|---|---|---|---|
| `211.2/.3`, `214.1` HarmonyOS device reach | no HarmonyOS NEXT device/CI | `PENDING_DEVICE:` / gate `pending_device` | provision a device + add to the §11.4.128 tracked set (via G20) |
| `221.2`, `223.1` Aurora device reach | no Aurora device + Russian toolchain | `PENDING_DEVICE:` + `UNCONFIRMED:` provenance | provision an Aurora device; mirror-pin + audit the SDK |
| `252.1` audit engagement | external spend | `Operator-blocked` (§11.4.101/.66) | fund + contract an independent audit firm |
| `262.1`, `264.1` fork-toolchain reproducibility | fork toolchain not yet deterministic | `PENDING_TOOLCHAIN:` keeps G26 honestly open | root-cause the fork non-determinism to zero |

None of these may be marked `pass` on metadata/config-only evidence (`09-…` §13).
A device-/engagement-gated gate that cannot be proven is `pending_device`/
`operator_blocked` with its unblock condition — the release notes state plainly
which reach targets are *certified* vs *built-but-pending-device*. Complexity
sums to ≈ `09-…` §14's ~172 engineer-days — the widest error bars in the
programme, sizing only, never a date (§11.4.6).

---

## Sources verified

- `09-phase3-reach-wbs.md` §2.1 gates G20–G26, §5–§12 (every task Desc/Deliverable/Acceptance/Effort/Tests), §13 traceability + honest-gap rule, §14 risk register, §15 open decisions D-P3-* — read 2026-06-26.
- Sibling `workable-items-model.md` (§3/§6/§9 incl. `gates` table + `pending_device`/`operator_blocked` outcomes), `dependency-graph.md` (§6 Phase-3 DAG), `subtask-deepening-p1.md`/`-p2.md` (§0 conventions) — authored this volume.
- Constitution anchors §11.4.3/.6/.10/.21/.27/.28/.40/.54/.66/.76/.86/.91/.93/.101/.102/.107/.108/.113/.115/.118/.128/.134/.135/.146/.151/.153/.156/.159/.161/.162/.165/.169 — read 2026-06-26.

> Honest boundary (§11.4.6): subtasks decompose stated Phase-3 work. Device- and
> engagement-gated subtasks carry `PENDING_DEVICE:` / `UNCONFIRMED:` /
> `Operator-blocked` markers verbatim — a gate is never `pass` on metadata-only
> evidence. All complexity is sizing `TARGET`, never a date; Phase-3 carries the
> programme's widest error bars.
