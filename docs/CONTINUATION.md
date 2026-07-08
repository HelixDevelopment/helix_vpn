# Helix VPN — Session Continuation File

**Revision:** 9
**Last modified:** 2026-07-08T18:14:00Z

> Helix Constitution §11.4.131 — standing session-resumption artifact.
> Re-read this file at the start of any new session before touching code.

---

## Summary

**Branch:** `main` (single working branch).
**Overall status:** Full MVP specification set COMPLETE. **Design System COMPLETE**. **Phase 0 Implementation — 7 Rust crates, 186 tests passing, G1+G2 gates proven, boringtun fixed, MASQUE/QUIC wired, orchestrator+reconciler integrated.**

**Active work (2026-07-08):**
1. ✅ MVP spec set — 11 vols, 126 md/html/pdf, all synced
2. ✅ Constitution + Governance — fully integrated, pre-commit active
3. ✅ Design System — OpenDesign, 26 files, 30+ components, 18+ screens
4. ✅ docs_chain — 3 contexts healthy
5. ✅ **P0-011: boringtun fixed** — test fixtures corrected (not a crypto bug), 2 ignored → 48 passing
6. ✅ **P0-031: MASQUE CONNECT-UDP** — quinn+h3 wired, connect.rs+datagram.rs, 2 benchmarks, adversarial tests
7. ✅ **P0-018: Orchestrator wired** — WG session management (wg_session.rs), Transport trait Connection::kind/effective_mtu, ReconcilerHandle
8. ✅ **P0-035: G2 DPI-block survival gate** — MASQUE survives DPI block (plain-WG blocked), clean QUIC wire fingerprint, 4 spike scripts + pkt sniffer
9. ✅ **P0-074: G6 reconcile** — live map-delta reconciliation (map.rs, reconciler.rs, integration tests)
10. ✅ **P1-090.1-.4: Transport + MASQUE hardening** — real UDP accept(), effective_mtu, context-id filtering, goodput benchmark
11. ✅ **3 CLI binaries** — helix-client, helix-connector, g2-dpi-probe
12. ✅ **helix_core governance** — CLAUDE.md, AGENTS.md, CONSTITUTION.md, QWEN.md, helix-deps.yaml
13. ✅ helix_core pushed to all upstreams (0e13a1a)

**Workspace: 7 crates, 186 tests, 0 failures, 0 errors, 0 ignored**

**Next queue (2026-07-08):**
1. **Fill G2 bulk-throughput gap** — G2-RESULTS.md §4 honest gap: 37.5% measured on non-bulk proxy metric; needs proper saturating benchmark (iperf3 or Rust-based)
2. **Create main repo test_g2.sh** — wrap helix_core G2 spike scripts as proper gate (following test_g1.sh pattern)
3. **Platform adapter research** — Android VpnService / iOS NEPacketTunnelProvider / Windows WFP
4. **G3 Platform-adapter benchmark gate** — define and build
5. **Submodule audit** — helix_qa, docs_chain dirty state; pull changes
6. **Phase 1 planning** — platform adapters, FFI surface, DNS, config crate

### Honest G2 gaps (from helix_core G2-RESULTS.md)
- Bulk throughput: 37.5% measured on non-bulk proxy metric — INCONCLUSIVE, needs saturating benchmark
- Canonical 3-netns rig test written but NOT executed (requires real root; unprivileged variant was executed)
- Both gaps root-caused and documented, not glossed over

**Locations:** spec: `docs/research/mvp/final/` | design: `docs/design/` | Rust: `submodules/helix_core/` (7 crates, 186 tests) | rig: `scripts/rig/` (4 gate scripts) | gates: `scripts/rig/test_g1.sh` (`test_g2.sh` pending) | G2 evidence: `submodules/helix_core/G2-RESULTS.md` + `scripts/spike/g2_*.sh`

---

## Completed Work (highlights)

### 1. Constitution submodule + mandatory submodules
- `constitution/` → `HelixDevelopment/HelixConstitution.git` (branch `main`)
- 11 own-org repos under `submodules/<name>` (flat, lowercase snake_case)
- `install_upstreams` run in each; `.helix-manifest.yaml` audit record
- Pre-commit hook, CI DISABLED (§11.4.156), local enforcement active

### 2. Full MVP specification set (V0–V10)
- 11 volumes, ~140 nano-detail documents, ~11.7K lines
- 46 Mermaid diagrams, SQL DDL, Podman/Docker/K8s manifests
- Every volume adversarial-reviewed (§11.4.142) + reconciled to GO (§11.4.134)
- 126 `.md` / 126 `.html` / 126 `.pdf` — all synced (§11.4.65)

### 3. Workable-items SQLite DB (§11.4.93)
- `docs/workable_items.db` — 484 items (P0: 36, P1: 210, P2: 132, P3: 96)
- Loader: `scripts/workable_items_loader.py` (md-to-db, bidirectional)

### 4. Design System
- 26 files, ~6,700 LOC — OpenDesign, tokens, components, screens, interaction/animation
- PDF/HTML/PNG/Figma exports complete

### 5. Phase 0 Implementation — helix_core (0e13a1a)
- **7 Rust crates, 186 tests, 0 failures**
  - helix-transport: Connection trait + UDP transport (30 tests + 1 bench + g1_integration)
  - helix-tun: async Linux TUN device (6 tests)
  - helix-wg: WireGuard+boringtun (48 tests, NO ignored — P0-011 fixed)
  - helix-orch: orchestrator + WG sessions + EventBus (17 tests)
  - helix-masque: MASQUE/QUIC transport (29 tests + 2 benches + g2_sec_adversarial)
  - helix-core: top-level crate + binaries + g1_orch_wg_integration + g6_map_reconcile_integration (40+5 tests)
- **3 CLI binaries:** helix-client, helix-connector, g2-dpi-probe
- **G2 gate proven:** MASQUE survives DPI block, clean QUIC fingerprint
- **G6 reconcile:** map-delta reconciliation via ReconcilerHandle
- **Governance:** CLAUDE.md, AGENTS.md, CONSTITUTION.md, QWEN.md, helix-deps.yaml

### 6. Test Rig + Infra (main repo)
- 7 rig scripts (common, setup, teardown, test_reach, test_firewall, test_netem, README)
- G1 gate: `scripts/rig/test_g1.sh`
- Makefile (11 targets), spike.sh, bench scripts

---

## What Remains

### Done (all subagents completed)
- **helix_core advancement** — 19 commits merged (P0-011, P0-031, P0-018, P0-035, P0-074, P1-090.1-.4, benchmarks, security, governance)
- All previous subagents (repo creation, design, Go binary, DOCX exports, design review)

### Known issues
- `install_upstreams` recipe format mismatch — remotes configured manually
- `helix_qa` nested submodules (docling) still dirty — pre-existing
- `docs_chain` submodule has dirty tracked file — pre-existing
- OpenDesign CLI (`od`) is GNU octal dump — no local OpenDesign agent for Figma generation

### Deferred
- **G2 bulk-throughput benchmark gap** — proper saturating benchmark needed
- **Main repo test_g2.sh** — wrap helix_core G2 spike scripts
- **Platform adapters** — Android VpnService, iOS NEPacketTunnelProvider, Windows WFP, Linux nftables — needs helix_core FFI stable
- **G3 Gate** — platform-adapter benchmark
- **Figma design file generation** — requires OpenDesign CLI or Figma MCP auth
- **UI implementation** — requires core transport layer stable first

---

## Evidence Locations

| Artifact | Path |
|----------|------|
| MVP spec set | `docs/research/mvp/final/` (126 md/html/pdf) |
| Design System | `docs/design/` (26 files, ~6,700 LOC) |
| helix_core workspace | `submodules/helix_core/` (7 crates, 186 tests, 0 failures) |
| G2 Results | `submodules/helix_core/G2-RESULTS.md` |
| G2 spike scripts | `submodules/helix_core/scripts/spike/g2_*.sh` (4 scripts) |
| G2 pkt sniffer | `submodules/helix_core/scripts/spike/g2_pkt_sniffer.py` |
| G1 Gate | `scripts/rig/test_g1.sh` |
| Test rig | `scripts/rig/` (7 scripts) |
| Benchmarks | `submodules/helix_core/crates/*/benches/*.rs` (3 benches) |
| Integration tests | `submodules/helix_core/crates/*/tests/*.rs` (3 test files) |
| helix-deps.yaml | `submodules/helix_core/helix-deps.yaml` (§11.4.31) |
| Workable-items DB | `docs/workable_items.db` (§11.4.93/.95) |
| DB loader | `scripts/workable_items_loader.py` |
| docs_chain contexts | `.docs_chain/contexts/*.yaml` |
| Pre-build gate | `tests/pre_build_verification.sh` (8 invariants) |
| Constitution | `constitution/` (submodule) |
| Submodule audit | `.helix-manifest.yaml` |

---

## Resumption prompt (§11.4.127)

### SHORT variant

> Continue work on `main` in `/home/milos/Factory/projects/tools_and_research/helix_vpn`; read `docs/CONTINUATION.md` first, then continue the Phase 0→1 work queue: G2 throughput gap, test_g2.sh, platform adapter research, submodule audit.

### FULL variant

```
You are resuming work on the Helix VPN project.

Repository:  /home/milos/Factory/projects/tools_and_research/helix_vpn
Branch:      main
Handoff doc: docs/CONTINUATION.md  ← read this FIRST

State at handoff
----------------
- Full MVP spec set: 126 md/html/pdf under docs/research/mvp/final/
- Design System: 26 files, ~6,700 LOC
- Phase 0 Implementation: 7 Rust crates, 186 tests PASS (0 ignored)
  - helix_core workspace at submodules/helix_core/ (Rust 1.96.1, ed. 2024)
  - helix_core HEAD: 0e13a1a (19 commits ahead from last session state)
- G1 Gate: scripts/rig/test_g1.sh (100-round UDP echo, JSON evidence)
- G2 Gate: PROVEN (helix_core G2-RESULTS.md + 4 spike scripts)
  - Honest gap: bulk-throughput benchmark INCONCLUSIVE, 3-netns variant not executed
- G6 Reconcile: map-delta reconciliation via ReconcilerHandle
- 3 CLI binaries: helix-client, helix-connector, g2-dpi-probe
- Test rig: scripts/rig/ (7 scripts)
- Latest main commit: b632680 — "feat: G1 gate script + helix_core submodule advanced"

Active work queue
-----------------
1. Fill G2 bulk-throughput gap (proper saturating benchmark)
2. Create main repo scripts/rig/test_g2.sh (wrap helix_core G2 scripts)
3. Platform adapter research — Android VpnService / iOS NEPacketTunnelProvider / Windows WFP
4. Submodule audit — helix_qa, docs_chain pull + fix dirty state
5. G3 Platform-adapter benchmark gate

First actions
-------------
1. git fetch --all --prune
2. git submodule update --remote --init
3. cargo test --all-targets (expect 186 pass, 0 ignored)
4. Continue from highest-priority workable item
```
