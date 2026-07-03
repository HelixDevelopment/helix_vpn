# Helix VPN — Session Continuation File

**Revision:** 6
**Last modified:** 2026-07-04T03:50:00Z

> Helix Constitution §11.4.131 — standing session-resumption artifact.
> Re-read this file at the start of any new session before touching code.

---

## Summary

**Branch:** `main` (single working branch).
**Overall status:** Full MVP specification set COMPLETE (11 volumes V0–V10, 126 `.md` files with synced HTML/PDF siblings). **Design System COMPLETE** (26 files, ~6,700 LOC, 8 platforms, OpenDesign). **Phase 0 Implementation BEGUN** — helix_core workspace bootstrapped + Transport trait defined + test rig built + Makefile + bench/spike harness. All committed and pushed.

**Active work (2026-07-04):**
1. ✅ **MVP spec set** — 11 vols, 126 md/html/pdf, all synced, Mermaid renders clean
2. ✅ **Constitution submodule** — fully integrated, pre-commit hooks active, CI disabled
3. ✅ **Workable-items DB** — 484 items, all Queued/Task
4. ✅ **Design System** — OpenDesign (DESIGN.md, tokens.css, components.html, manifest), 30+ components, 18+ screens, interaction/animation specs, Figma tokens, HTML+PDF+PNG exports
5. ✅ **docs_chain** — 3 contexts (workable-items, spec-exports, design), all doctor PASS
6. ✅ **Phase 0: Core Transport** (HVPN-P0-001) — Cargo workspace, Transport trait, cargo check+test PASS
7. ✅ **Phase 0: Test Rig** (HVPN-P0-022) — 7 scripts, netns+nftables+netem topology
8. ✅ **Phase 0: Make + Bench + Spike** (HVPN-P0-080/-077) — Makefile (11 targets), bench runner, spike.sh
9. ✅ **Design review fixes** — connected color mismatch, missing buttons.md, spacing.json, dark border-error
10. ✅ **Submodules** — helix_core pushed (first Rust code!), containers exec-bit fixes pushed

**Next work queue (priority order):**
1. **HVPN-P0-004** — Transport trait refinement + unit tests
2. **HVPN-P0-008** — Plain-UDP transport implementation
3. **HVPN-P0-011** — WireGuard boringtun wrapper
4. **HVPN-P0-015** — Linux TUN device abstraction
5. **HVPN-P0-025** — G1 two-way routing reach test
6. **Design PDF re-export** — with fixes applied

**Spec location:** `docs/research/mvp/final/` — see `MASTER_INDEX.md`
**Design location:** `docs/design/` — see `docs/design/README.md`
**Rust workspace:** `submodules/helix_core/`
**Test rig:** `scripts/rig/`

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
- **Phase 0 Implementation — Core Transport** (HVPN-P0-001)
  - Cargo workspace with 2 member crates, compiled (0 warnings)
  - Transport trait — the foundational interface (dial/listen/accept)
  - TransportRegistry with register/resolve
  - All 6 scripts executable, bash -n clean
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
- **Implementation Phase 0 Remaining Items** — P0-004 (trait refinement), P0-008 (plain-UDP), P0-011 (boringtun), P0-015 (TUN), P0-025 (G1 test), P0-018 (orchestrator)
- **Figma design file generation** — requires OpenDesign CLI install or Figma MCP authentication
- **UI implementation** — requires core transport layer stable first

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
  - OpenDesign 9-section DESIGN.md + tokens.css + components.html
  - Component library (30+ components, 4 platform variants)
  - Screen wireframes (18+ screens across 8 platforms)
  - Interaction/animation specs with full state machine
  - Exports: PDF (4), HTML (4), PNG (2), Figma tokens JSON
- Phase 0 Implementation BEGUN:
  - HELIX_CORE workspace: submodules/helix_core/ — Transport trait, cargo check PASS
  - Test rig: scripts/rig/ — 7 scripts, netns+nftables+netem topology
  - Infra: Makefile (11 targets), scripts/spike.sh, scripts/bench/
- Workable-items DB: docs/workable_items.db (484 items)
- Constitution: constitution/ submodule active, pre-commit hooks active
- CI: DISABLED (§11.4.156)
- Latest commit: f33c282 — "feat: Implementation Phase 0"
  (28 files, 1296 insertions — pushed to all upstreams)

Active work queue
-----------------
1. Transport trait refinement + unit tests (HVPN-P0-004)
2. Plain-UDP transport implementation (HVPN-P0-008)
3. WireGuard boringtun wrapper (HVPN-P0-011)
4. Linux TUN device abstraction (HVPN-P0-015)
5. G1 two-way routing reach test (HVPN-P0-025)
6. Design PDF re-export with fixes applied

First actions
-------------
1. git fetch --all --prune
2. Read docs/CONTINUATION.md fully
3. Read any pending /tmp/*.md subagent reports
4. Continue from highest-priority workable item
```
