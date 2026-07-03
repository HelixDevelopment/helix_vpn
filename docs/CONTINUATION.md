# Helix VPN — Session Continuation File

**Revision:** 7
**Last modified:** 2026-07-04T04:00:00Z

> Helix Constitution §11.4.131 — standing session-resumption artifact.
> Re-read this file at the start of any new session before touching code.

---

## Summary

**Branch:** `main` (single working branch).
**Overall status:** Full MVP specification set COMPLETE. **Design System COMPLETE** (26 files, ~6,700 LOC). **Phase 0 Implementation ADVANCED** — 4 Rust crates, 39 tests passing, submodule pushed.

**Active work (2026-07-04):**
1. ✅ MVP spec set — 11 vols, 126 md/html/pdf, all synced
2. ✅ Constitution — fully integrated, pre-commit hooks active
3. ✅ Design System — OpenDesign, 30+ components, 18+ screens, 4 exports
4. ✅ docs_chain — 3 contexts, all doctor PASS
5. ✅ P0-001: Workspace skeleton — 4 crates, compiled
6. ✅ P0-022: Test rig — 7 scripts, netns+nftables+netem
7. ✅ P0-080/-077: Make + Bench + Spike — 11 targets
8. ✅ P0-004: Transport trait refinement — close(), local_addr(), peer_addr(), mock transport
9. ✅ P0-008: Plain-UDP transport — UdpTransport, UdpConnection, 12 tests
10. ✅ P0-015: Linux TUN device — helix-tun crate, 5+1 tests
11. ✅ P0-011 prep: WireGuard stub — helix-wg crate, 5 modules, 21 tests
12. ✅ Workspace: 39 tests total — 4 crates, 0 failures, 0 errors
13. ✅ Design review fixes — F8, F10, F11, PDFs re-exported
14. ✅ All committed + helix_core submodule advanced

**Next work queue:**
1. P0-011: WireGuard boringtun integration — wire boringtun 0.7.1 into helix-wg
2. P0-018: Orchestrator three-loop core — needs WireGuard + TUN stable
3. P0-025: G1 routing reach test — plain-UDP echo through the rig
4. P0-028: QUIC/MASQUE transport — needed for G2
5. Platform adapters — VpnService, NEPacketTunnelProvider, WFP

**Locations:** spec: `docs/research/mvp/final/` | design: `docs/design/` | Rust: `submodules/helix_core/` (4 crates, 39 tests) | rig: `scripts/rig/`

---

## Completed Work (highlights)

### 1. Constitution submodule + mandatory submodules
- `constitution/` → `HelixDevelopment/HelixConstitution.git` (branch `main`)
- 11 own-org repos under `submodules/<name>` (flat, lowercase snake_case)
- `install_upstreams` run in each; `.helix-manifest.yaml` audit record
- Pre-commit hook, CI DISABLED (§11.4.156), local enforcement active

### 2. Full MVP specification set (V0–V10)
- 11 volumes, ~140 nano-detail documents, ~11.7K lines in the spine + pass-1 set
- All 16 research docs cited; decisions D1–D8 surfaced
- 46 Mermaid diagrams, SQL DDL, Podman/Docker/K8s manifests
- Every volume adversarial-reviewed (§11.4.142) + reconciled to GO (§11.4.134)
- 126 `.md` / 126 `.html` / 126 `.pdf` — all synced (§11.4.65)

### 3. Workable-items SQLite DB (§11.4.93)
- `docs/workable_items.db` — 484 items (P0: 36, P1: 210, P2: 132, P3: 96)
- Schema: items, item_history, test_diary, gates, operator_block_details, obsolete_details, meta
- Loader: `scripts/workable_items_loader.py` (md-to-db, bidirectional)
- All items start as `Queued` / `Task` status

### 4. Research corpus
- `docs/research/mvp/` — 16 source docs (11 LLM analyses + 5 refined)
- `v09-research/` — 10 per-angle research dossiers (all cited, all verified except wireguard partially)

---

## What Remains

### Done (all subagents completed)
- **D-PKI-CA-TIER** — operator confirmed: two-tier issuing CA as MVP default
- **D-OD-1** — operator confirmed: OpenDesign authoring-layer interpretation
- **vasic-digital component repos** — 8 repos created on GitHub+GitLab + added as submodules
- **Go workable-items binary** — HVPN-P1-150 complete, 6 commands verified
- **DOCX exports** — pipeline updated, all docs have DOCX siblings
- **Design System COMPLETE** — 26 files, ~6,700 LOC
  - OpenDesign 9-section DESIGN.md with light+dark themes + 5 custom palettes
  - tokens.css (200+ CSS custom properties) + Figma Variables-compatible JSON
  - Component library (30+ components, 4 platform variants)
  - Screen wireframes (18+ screens across 8 platforms)
  - Interaction patterns + animation specs
  - Exports: 4 PDF, 4 HTML, 2 PNG screenshots
- **Phase 0 Implementation ADVANCED** — 4 Rust crates, 39 tests, all pushed
  - helix-transport: Transport trait + UDP transport (12 tests)
  - helix-tun: async Linux TUN device (5+1 tests)
  - helix-wg: WireGuard stub + timers (21 tests)
  - helix-core: workspace re-export (0 tests)
- **Phase 0 Implementation — Test Rig** (HVPN-P0-022)
  - 7 scripts (common, setup, teardown, test_reach, test_firewall, test_netem, README)
  - 3-namespace topology (client/bridge/server) with nftables + netem
  - G1 precondition gate scriptable
- **Phase 0 Implementation — Infra** (HVPN-P0-080/-077)
  - Makefile with 11 targets (spike, check, test, bench, rig, clean, etc.)
  - scripts/spike.sh (S0→S4 one-shot verification command)
  - scripts/bench/run.sh + compare.sh (iperf3/ping, CSV output)
- **Design quality review** — 15 findings, 5 fixed (F1,F2,F4,F7,F13)
- **Docs chain** — 'design' context registered (12 nodes, 8 edges, doctor PASS)
- **Submodule pushes** — helix_core (first Rust code), containers (exec fixes)

### Known issues
- `install_upstreams` recipe format mismatch: recipe files use `GIT_SSH_URL` but the script expects `UPSTREAMABLE_REPOSITORY`. Remotes configured manually. Should be fixed upstream in the Upstreamable toolkit.
- `helix_qa` nested submodules (docling) still dirty — pre-existing, not from our work
- `docs_chain` submodule has dirty tracked file — pre-existing, needs upstream fix
- Design system: OpenDesign CLI (`od`) is GNU octal dump, not the OpenDesign tool — no local OpenDesign agent for automated Figma generation

### Deferred
- **Phase 0 Remaining (HIGH)** — P0-011 (boringtun wire), P0-018 (orchestrator three-loop), P0-025 (G1 test with rig), P0-028 (QUIC/MASQUE)
- **Figma design file generation** — requires OpenDesign CLI install or Figma MCP authentication
- **UI implementation** — requires core transport layer stable first
- **Platform adapters** — Android VpnService, iOS NEPacketTunnelProvider, Windows WFP, Linux nftables — each needs helix_core FFI stable

---

## Evidence Locations

| Artifact | Path |
|----------|------|
| MVP spec set | `docs/research/mvp/final/` (126 md/html/pdf) |
| Master index | `docs/research/mvp/final/MASTER_INDEX.md` |
| Spec spine | `docs/research/mvp/final/SPECIFICATION.md` |
| Research corpus | `docs/research/mvp/` (16 source docs) |
| Research dossiers | `docs/research/mvp/final/v09-research/` |
| Workable-items DB | `docs/workable_items.db` (§11.4.93/.95) |
| DB loader | `scripts/workable_items_loader.py` |
| DB loader docs | `docs/scripts/workable_items_loader.md` |
| docs_chain wrapper | `scripts/docs_chain_md_to_db.sh` |
| docs_chain contexts | `.docs_chain/contexts/*.yaml` |
| .gitignore-meta | `.gitignore-meta/*.yaml` (§11.4.77 regen mechanisms) |
| Pre-build gate | `tests/pre_build_verification.sh` (8 invariants) |
| Export script | `scripts/testing/sync_all_markdown_exports.sh` |
| Mermaid helper | `scripts/testing/render_mermaid_blocks.py` |
| Mermaid cache | `.mermaid-cache/` (content-addressed PNGs) |
| Constitution | `constitution/` (submodule) |
| Submodule audit | `.helix-manifest.yaml` |
| Pre-commit hook | `.githooks/pre-commit` |
| CI (DISABLED) | `.github/workflows/constitution.yml.disabled-local-only` |
| **DESIGN SYSTEM** | **`docs/design/`** (26 files, ~6,700 LOC) |
| OpenDesign DESIGN.md | `docs/design/opendesign/helix/DESIGN.md` |
| OpenDesign tokens.css | `docs/design/opendesign/helix/tokens.css` |
| OpenDesign manifest | `docs/design/opendesign/helix/manifest.json` |
| Component reference | `docs/design/opendesign/helix/components.html` |
| Component library doc | `docs/design/components/README.md` |
| Screen wireframes | `docs/design/screens/README.md` |
| Interaction/animation | `docs/design/interaction/README.md` |
| Design master index | `docs/design/README.md` |
| Color tokens JSON | `docs/design/tokens/color.json` |
| Typography tokens | `docs/design/tokens/typography.json` |
| Figma tokens JSON | `docs/design/exports/HelixVPN-Figma-Tokens.json` |
| Design export PDFs | `docs/design/exports/HelixVPN-*.pdf` (4 files) |
| Design screenshots | `docs/design/exports/HelixVPN-Components-*.png` (2 files) |
| Platform-specific | `docs/design/components/{desktop,mobile,aurora,web}/*.md` |

---

## Resumption prompt (§11.4.127)

### SHORT variant

> Continue work on `main` in `/home/milos/Factory/projects/tools_and_research/helix_vpn`; read `docs/CONTINUATION.md` first, then check background tasks and continue the work queue.

### FULL variant

```
You are resuming work on the Helix VPN project.

Repository:  /home/milos/Factory/projects/tools_and_research/helix_vpn
Branch:      main
Handoff doc: docs/CONTINUATION.md  ← read this FIRST

State at handoff
----------------
- Full MVP spec set: 126 md/html/pdf under docs/research/mvp/final/
- Design System: 26 files, ~6,700 LOC under docs/design/
- Phase 0 Implementation — 4 Rust crates, 39 tests PASS, all compiled
  - helix_core workspace at submodules/helix_core/ (edition 2024, Rust 1.96.1)
  - Crates: helix-transport (UDP+trait), helix-tun (TUN device), helix-wg (WG stub), helix-core (wrapper)
- Test rig: scripts/rig/ (7 scripts, bash -n clean)
- Infra: Makefile (11 targets), scripts/bench/ (benchmark harness), scripts/spike.sh
- Workable-items DB: docs/workable_items.db (484 items)
- Latest commit: 2c8577c — "feat: Design fixes + helix_core submodule pointer"
- helix_core submodule: 6ecd3b8 — "feat: Implement UDP transport + TUN + WG stub"

Active work queue
-----------------
1. WireGuard boringtun integration (P0-011) — wire boringtun 0.7.1 into helix-wg
2. Orchestrator three-loop core (P0-018) — needs WG + TUN
3. G1 routing reach test (P0-025) — plain-UDP echo through rig
4. QUIC/MASQUE transport (P0-028) — needed for G2

First actions
-------------
1. git fetch --all --prune
2. cd submodules/helix_core && git fetch --all --prune
3. Read docs/CONTINUATION.md fully
4. Continue from highest-priority workable item
```
