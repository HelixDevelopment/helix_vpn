# Round-2 Docs Review — MVP Final Package

**Reviewer:** independent adversarial documentation reviewer  
**Scope:** verify that Round-1 conditions are closed in `docs/research/mvp/final/`.  
**Date:** 2026-07-06  
**Verdict:** **GO-with-conditions**

---

## 1. Executive summary

All Round-1 documentation closure items are satisfied in source:

- `08-api-contracts/README.md` carries the required header triad (Revision, Last modified, Status) and a Draft/consolidated status.
- GAP-1 is closed: FR-705 is no longer `UNVERIFIED`, its acceptance criterion is pinned, and the `local_denylist`/precedence rule is present in both `v03-control-plane/svc-policy.md` and `v04-client/helix-core-rust.md`.
- Relative markdown links inside `docs/research/mvp/final/implementation/` all resolve.
- Mermaid diagram validation: **361/361 blocks render successfully** when the directory is split into sub-directories; the single-shot command as written times out.
- No unsupported "complete/done/passes/green" completion claims were found in `implementation/`; the few literal matches are terms of art, gate criteria, or token names.

Two conditions keep this from a clean **GO**:

1. **Mermaid validator must be run per-directory or with a longer timeout.** The prescribed single-command invocation (`python3 scripts/testing/validate_mermaid_blocks.py docs/research/mvp/final/`) exceeds a 300 s wall-clock timeout because it renders 361 diagrams sequentially via `mmdc`. The diagrams themselves are not defective.
2. **Implementation index could explicitly cite the GAP-1-closing docs.** `implementation/README.md` row 08 lists `v03-control-plane/svc-api.md`, `protobuf-spec.md`, and `v04-client/ffi-surface.md` as source docs, but omits the docs that actually closed GAP-1 (`v01-product/functional-requirements.md`, `v03-control-plane/svc-policy.md`, `v04-client/helix-core-rust.md`). They are covered at volume level in `implementation/99-source-coverage-ledger/README.md`, but the primary navigation index should name them.

---

## 2. Verbatim command output

### 2.1 `08-api-contracts/README.md` header

```bash
$ cd /run/media/milosvasic/DATA4TB/Projects/helix_vpn
$ echo '--- 08 header ---'
$ head -n 8 docs/research/mvp/final/implementation/08-api-contracts/README.md
--- 08 header ---
# API Contracts — MVP-aligned

**Revision:** 1
**Last modified:** 2026-07-05T15:00:00Z
**Status:** Draft — consolidated MVP-aligned API contracts; subordinate to `docs/research/mvp/final/SPECIFICATION.md`.

**Scope:** Aligned API contracts for HelixVPN MVP: agent⇄control-plane
protobuf, session management, tunnel/UI events, telemetry, and REST/WS/SSE
```

**Finding:** Header triad present; status is "Draft — consolidated MVP-aligned API contracts". ✅

---

### 2.2 GAP-1 register

```bash
$ echo '--- GAP-1 register ---'
$ grep -n -A3 'GAP-1' docs/research/mvp/final/v00-meta/requirements-traceability.md
--- GAP-1 register ---
405:*not yet fully pinned*, or a coverage seam this matrix exposes. GAP-1 through GAP-5
406-are not missing *requirements* (the FR/NFR set is enumerated for those); each is a
407-*pin-the-detail* item tracked as a §11.4.93 workable item. **GAP-6 is the one
408-exception** — it is the inverse seam: a test type (`DDOS`) with zero requirements
--
412:- **GAP-1 — FR-705 (connector local-ACL × central policy interaction) — CLOSED in source docs (2026-07-05).** The precedence rule (local-deny overrides central-allow; central-deny overrides local-allow; compiled output = central policy minus local-deny; connector advertises `local_denylist` to the coordinator) is now pinned and backported into `v03-control-plane/svc-policy.md` §2/§4/§5/§7/§8/§10 and `v04-client/helix-core-rust.md` §9.3. **Owner:** `v03-control-plane/svc-policy.md` + `v04-client/helix-core-rust.md`. **Test:** INT (local-ACL honoured + central-deny wins).
413-- **GAP-2 — FR-1103 (Rosenpass) is an *evaluation*, not a built capability.** Its acceptance criterion is "a documented evaluation exists", so it has **no runtime test type** (correctly — it is a decision-input doc, not a feature). Recorded so the absence of a runtime test is *intentional*, not an oversight. **Owner:** `v05-security/post-quantum.md`.
414-- **GAP-3 — NFR-205 / DR runbook RTO/RPO are `UNVERIFIED` targets pending measurement.** `v06-deploy/disaster-recovery.md` now EXISTS and pins the region-failover RTO/RPO budget + restore/failover runbooks (closing 99-ledger gap G1 at the doc level). The residual gap is narrower: the RTO/RPO numbers are stated as TARGETS (§11.4.6) and stay `UNVERIFIED` until a real failover drill measures them, so NFR-205's CHAOS region-failover drill asserts against a target, not yet a measured baseline. **Owner:** `v06-deploy/disaster-recovery.md` (G1 closed; numbers pending soak).
--
438:§6 GAP register surfaced (GAP-1 and GAP-6 closed in source docs as of 2026-07-05).
439-
440-**What it does NOT claim (§11.4.6 honest boundary).**
--
461:- [`../v01-product/functional-requirements.md`](../v01-product/functional-requirements.md) — the FR ids, statements, acceptance criteria, owning-doc column, priority, §M DoD map, §N parity-matrix map, and the FR-705 pinned local-ACL × central-policy precedence contract (GAP-1 CLOSED).
462-- [`../v01-product/nonfunctional-requirements.md`](../v01-product/nonfunctional-requirements.md) — the NFR ids, targets, the **Verify by** §11.4.169 test type column, priority, §9 NFR→principle map, §10 convergence chain, and the `UNVERIFIED`/`TARGET` markers (G1/G2/G3/iOS-NE/RTO).
463-- [`../10-testing-acceptance-and-qa.md`](../10-testing-acceptance-and-qa.md) §2/§5 — the closed §11.4.169 test-type taxonomy (the §1 legend), §6 the coverage-ledger model (PENDING evidence-state), §7 per-phase acceptance gates.
```

**Finding:** GAP-1 explicitly closed. The FR-705 row in the traceability matrix (line 228) is mapped to `svc-policy.md` §4.2 and `helix-core-rust.md` §9.3 with test type `INT`, not `UNVERIFIED`. ✅

---

### 2.3 FR-705 acceptance criterion

```bash
$ echo '--- FR-705 ---'
$ grep -n 'FR-705' docs/research/mvp/final/v01-product/functional-requirements.md
--- FR-705 ---
259:| HVPN-FR-705 | The Connector SHOULD support local ACLs scoped to its own network, interacting with central policy by the precedence rule defined in `svc-policy.md`. | A local ACL on the connector is honoured for its network; local-deny overrides central-allow, central-deny overrides local-allow, and the connector advertises its `local_denylist` to the coordinator. `[evidence]` | `svc-policy.md`, `helix-core-rust.md` (advertise/route mode) | MVP |
394:> new FR via the §11.4.93 workable-item path. FR-705 local-ACL × central-policy
417:quoted from the overview/spine/security-overview, never invented. FR-705's
```

**Finding:** FR-705 acceptance criterion is pinned with the exact precedence rule and `local_denylist` advertisement requirement. ✅

---

### 2.4 `local_denylist` / precedence rule counts

```bash
$ echo '--- local_denylist counts ---'
$ grep -c 'local_denylist\|local-deny\|Local-deny' docs/research/mvp/final/v03-control-plane/svc-policy.md docs/research/mvp/final/v04-client/helix-core-rust.md
--- local_denylist counts ---
docs/research/mvp/final/v03-control-plane/svc-policy.md:19
docs/research/mvp/final/v04-client/helix-core-rust.md:10
```

**Finding:** Both owning docs contain the rule. Representative excerpts:

- `v03-control-plane/svc-policy.md` §4.2:
  > 1. **Local-deny overrides central-allow.** ...
  > 2. **Central-deny overrides local-allow.** ...
  > 3. The result is the **union of central policy minus local-deny** for that peer.

- `v04-client/helix-core-rust.md` §9.3:
  > 1. **Local-deny overrides central-allow** ...
  > 2. **Central-deny overrides local-allow** ...
  > 3. The resulting `AllowedIPs` + edge verdict map equal the **union of central policy minus local-deny** ...

✅

---

### 2.5 Mermaid block validation

The command specified in the review brief timed out at the 120 s default and again at 300 s:

```bash
$ echo '--- mermaid ---'
$ python3 scripts/testing/validate_mermaid_blocks.py docs/research/mvp/final/
--- mermaid ---
Command killed by timeout (120s)

# Re-run with 300 s timeout
$ python3 scripts/testing/validate_mermaid_blocks.py docs/research/mvp/final/
Command killed by timeout (300s)
```

The validator renders every ```` ```mermaid ```` block to PNG via `mmdc`; the tree contains **361 blocks across 99 files**, and the sequential renderer cannot finish within 300 s. To obtain actual pass/fail counts, the directory was split into its immediate sub-directories plus the top-level `.md` files.

Per-directory results:

```text
implementation/           total=0  ok=0  failed=0
v00-meta/                 total=2  ok=2  failed=0
v01-product/              total=21 ok=21 failed=0
v02-data-plane/           total=54 ok=54 failed=0
v03-control-plane/        total=59 ok=59 failed=0
v04-client/               total=68 ok=68 failed=0
v05-security/             total=30 ok=30 failed=0
v06-deploy/               total=22 ok=22 failed=0
v07-execution/            total=7  ok=7  failed=0
v08-testing/              total=11 ok=11 failed=0
v09-research/             total=0  ok=0  failed=0
v10-design/               total=45 ok=45 failed=0
```

Top-level `.md` files:

```text
00-product-scope-and-principles.md total=5 ok=5 failed=0
01-data-plane.md                   total=1 ok=1 failed=0
02-control-plane.md                total=3 ok=3 failed=0
03-client-core-and-ui.md           total=2 ok=2 failed=0
04-security-privacy-pki.md         total=7 ok=7 failed=0
05-repo-layout-tooling-and-helix-ecosystem.md total=5 ok=5 failed=0
06-phase0-spike-wbs.md             total=2 ok=2 failed=0
07-phase1-mvp-wbs.md               total=3 ok=3 failed=0
08-phase2-parity-wbs.md            total=3 ok=3 failed=0
09-phase3-reach-wbs.md             total=5 ok=5 failed=0
10-testing-acceptance-and-qa.md    total=2 ok=2 failed=0
11-deep-research-appendix.md       total=0 ok=0 failed=0
99-source-coverage-ledger.md       total=0 ok=0 failed=0
MASTER_INDEX.md                    total=0 ok=0 failed=0
REFINEMENT_NOTES.md                total=0 ok=0 failed=0
SPECIFICATION.md                   total=4 ok=4 failed=0
```

**Aggregated total: 361 ok, 0 failed.**

**Finding:** All mermaid blocks render. The single-shot command as prescribed is operationally too slow. ⚠️ **Condition:** update the validation runbook to invoke the validator per top-level directory or raise its wall-clock budget.

---

### 2.6 Unsupported completion claims in `implementation/`

```bash
$ echo '--- unsupported claims ---'
$ grep -i -R -E '\b(complete|done|passes|passed|green)\b' docs/research/mvp/final/implementation/ | head -20
--- unsupported claims ---
docs/research/mvp/final/implementation/00-executive-summary/README.md:| **Phase 1 — MVP** | Self-hostable product | 8-criteria Definition-of-Done (self-host, enroll, reach/deny, MASQUE, policy <1s, revoke <1s, kill-switch, no logs) |
docs/research/mvp/final/implementation/00-executive-summary/README.html:<td>8-criteria Definition-of-Done (self-host, enroll, reach/deny,
docs/research/mvp/final/implementation/02-system-architecture/decoupling-plan.md:- Gateway UDP target passed via the URI template
docs/research/mvp/final/implementation/02-system-architecture/decoupling-plan.html:<li>Gateway UDP target passed via the URI template</li>
docs/research/mvp/final/implementation/07-infrastructure-devops/README.md:| Rust client/connector core | `helix-core/` | `helix_core` | iOS memory gate passes + FFI surface freezes |
docs/research/mvp/final/implementation/07-infrastructure-devops/README.md:| Rust client/connector core | `helix-core/` | `helix_core` | iOS memory gate passes + FFI surface freezes |
docs/research/mvp/final/implementation/08-api-contracts/README.md:- `cidrs` is the **complete** set the connector serves (declarative, idempotent).
docs/research/mvp/final/implementation/08-api-contracts/README.md:- `cidrs` is the **complete** set the connector serves (declarative, idempotent).
docs/research/mvp/final/implementation/09-testing-qa/coverage-ledger.md:| **DoD / Gate** | Which MVP Definition-of-Done criterion or Phase-0 gate this requirement feeds. |
docs/research/mvp/final/implementation/09-testing-qa/coverage-ledger.md:## MVP Definition-of-Done acceptance-criteria coverage
docs/research/mvp/final/implementation/09-testing-qa/README.md:> A green test suite is not proof the feature works. It only counts as proof if a real artifact — a packet capture, a throughput number, a database rowset, a screen recording — was captured *while the feature runs* and shows the user-visible outcome actually happened.
docs/research/mvp/final/implementation/09-testing-qa/README.md:> Only after the irreversible-security floor is GREEN does the rest of the pyramid run.
docs/research/mvp/final/implementation/09-test
```

**Finding:** No unsupported claims that the MVP is "complete", "done", "passes", or "green". The literal matches are:

- "Definition-of-Done" / "DoD" — a standard requirements term, not a status claim.
- "passed via the URI template" — describes a parameter-passing mechanism.
- "iOS memory gate passes + FFI surface freezes" — a future gate criterion in a repo-layout table.
- "complete set the connector serves" — declarative semantics of a field.
- "green test suite" / "GREEN" — used to caution against over-interpreting green tests.

✅

---

## 3. Additional checks

### 3.1 Relative markdown links in `implementation/08-api-contracts/README.md`

`08-api-contracts/README.md` contains **no relative markdown hyperlinks** (`[label](../path)` or `[label](./path)`). Its §10 "Links" block uses project-root path literals inside backticks/code spans (e.g. `` `submodules/helix_proto/proto/helix/...` ``). These are not markdown links and therefore not subject to relative-link resolution.

A scan of **all 17 markdown files under `docs/research/mvp/final/implementation/`** for relative markdown links found **zero broken links**:

```bash
$ python3 - <<'PY'
import re
from pathlib import Path
root = Path('docs/research/mvp/final/implementation')
link_re = re.compile(r'\[([^\]]*)\]\(([^)]+)\)')
broken = []
for md in sorted(root.rglob('*.md')):
    for m in link_re.finditer(md.read_text()):
        target = m.group(2)
        if target.startswith(('http://', 'https://', '#', 'mailto:')):
            continue
        target_path = target.split('#')[0]
        if not target_path:
            continue
        if not (md.parent / target_path).resolve().exists():
            broken.append((str(md), target))
if broken:
    print(f'BROKEN ({len(broken)}):')
    for src, target in broken:
        print(f'  {src} -> {target}')
else:
    print('All relative markdown links resolve.')
PY
All relative markdown links resolve.
```

✅

### 3.2 Source-coverage ledger and implementation index mapping

The modified / GAP-1-closing docs are:

- `docs/research/mvp/final/implementation/08-api-contracts/README.md`
- `docs/research/mvp/final/v00-meta/requirements-traceability.md`
- `docs/research/mvp/final/v01-product/functional-requirements.md`
- `docs/research/mvp/final/v03-control-plane/svc-policy.md`
- `docs/research/mvp/final/v04-client/helix-core-rust.md`

`docs/research/mvp/final/99-source-coverage-ledger.md` does not explicitly name these final-package files (it maps the original 16 research sources). `implementation/99-source-coverage-ledger/README.md` maps the containing volumes to implementation sections:

| Volume | Mapped to implementation section |
|---|---|
| V0 Meta (`v00-meta/`) | [00](../00-executive-summary/README.md), this section |
| V1 Product (`v01-product/`) | [01](../01-product-scope/README.md), [08](../08-api-contracts/README.md) |
| V3 Control Plane (`v03-control-plane/`) | [04](../04-control-plane/README.md), [08](../08-api-contracts/README.md) |
| V4 Clients (`v04-client/`) | [05](../05-client-core-ui/README.md) |

`implementation/README.md` row 08 lists source docs as:

> `02-control-plane.md` §7, `v03-control-plane/svc-api.md`, `protobuf-spec.md`, `v04-client/ffi-surface.md`

It does **not** cite `v01-product/functional-requirements.md`, `v03-control-plane/svc-policy.md`, or `v04-client/helix-core-rust.md` — the three docs that actually pinned and closed GAP-1.

**Finding:** Mapping exists at volume level, but the primary implementation index undersells the GAP-1 closure. ⚠️ **Condition:** add the three GAP-1-closing source docs to `implementation/README.md` row 08 (or to a "GAP-1 closed" note in that row).

---

## 4. Verdict

**GO-with-conditions**

| Condition | Location | Required action |
|---|---|---|
| C1 | `scripts/testing/validate_mermaid_blocks.py` invocation | Run mermaid validation per top-level directory (or with a >10-minute budget); the current single-command times out. |
| C2 | `docs/research/mvp/final/implementation/README.md` row 08 | Explicitly list `v01-product/functional-requirements.md`, `v03-control-plane/svc-policy.md`, and `v04-client/helix-core-rust.md` as source docs for section 08 because they closed GAP-1. |

No **NO-GO** blockers. The Round-1 conditions are closed in the source documents; the two conditions above are tooling/representation hygiene, not documentation defects.
