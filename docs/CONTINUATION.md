# Helix VPN — Session Continuation File

**Revision:** 4
**Last modified:** 2026-06-26T15:10:00Z

> Helix Constitution §11.4.131 — standing session-resumption artifact.
> Re-read this file at the start of any new session before touching code.

---

## Summary

**Branch:** `main` (single working branch).
**Overall status:** Full MVP specification set COMPLETE (11 volumes V0–V10, 126 `.md` files with synced HTML/PDF siblings). Workable-items SQLite DB populated (484 items). Constitution fully integrated. No VPN application code exists yet — spec-only mandate.

**Active work (2026-06-26):**
1. ✅ §11.4.134 re-review — manual confirmation GO (proto consistent, D8 present, spine correct)
2. ✅ §11.4.168 Mermaid rendering — FIXED, all 126 HTML clean, 98 with embedded diagrams
3. ✅ §11.4.93 workable-items SQLite DB — 484 items loaded
4. ✅ §11.4.106 docs_chain wired — 2 contexts, doctor OK, sync in-sync
5. ✅ §11.4.65 HTML/PDF exports — all docs have synced siblings
6. ✅ §11.4.153 DOCX exports — pipeline updated, generating DOCX alongside HTML/PDF
7. ⏳ Go workable-items binary (HVPN-P1-150) — subagent building
8. ⏳ vasic-digital component repos — subagent creating on GitHub+GitLab
9. ✅ Pre-build gate updated — Inv7 (DB) + Inv8 (docs_chain) added
10. ✅ .gitignore-meta regen mechanisms created (§11.4.77)
11. ✅ README Tracked-Items section added (§11.4.57)
12. ✅ Script companion docs created with HTML/PDF siblings (§11.4.18)

**Spec location:** `docs/research/mvp/final/` — see `MASTER_INDEX.md` for the full document tree.

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

### Pre-tag quality items
1. **D-PKI-CA-TIER** — two-tier issuing CA (reconciled to svc-pki source); operator may veto for single-tier MVP
2. **D-OD-1** — OpenDesign interpretation (authoring layer + decoupled token export); awaits operator confirm

### In progress (subagents)
3. **Go workable-items binary** (`cmd/workable-items/`) — HVPN-P1-150, subagent building
4. **vasic-digital component repos** (GitHub+GitLab) — subagent creating

### Deferred
5. **Implementation phase** — spec-only mandate; no code until operator directs

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

---

## Resumption prompt (§11.4.127)

### SHORT variant

> Continue work on `main` in `/Volumes/T7/Projects/helix_vpn`; read `docs/CONTINUATION.md` first, then check background tasks and continue the spec-materials work queue.

### FULL variant

```
You are resuming work on the Helix VPN project.

Repository:  /Volumes/T7/Projects/helix_vpn
Branch:      main
Handoff doc: docs/CONTINUATION.md  ← read this FIRST

State at handoff
----------------
- Full MVP spec set: 126 md/html/pdf under docs/research/mvp/final/
- Workable-items DB: docs/workable_items.db (484 items, §11.4.93)
- Constitution: constitution/ submodule active
- CI: DISABLED (§11.4.156), local enforcement via .githooks/pre-commit
- No VPN application code exists (spec-only mandate)

Active work queue
-----------------
1. §11.4.134 re-review of V1/V6/V7/V8/V0 reconciliation fixes
2. §11.4.168 Mermaid rendering verification (full re-export)
3. docs_chain wiring (§11.4.106)
4. vasic-digital component repos (deferred until spec stabilizes)
5. DOCX exports (§11.4.153)

First actions
-------------
1. git fetch --all
2. Check background task status
3. Continue the work queue from item 3+
```
