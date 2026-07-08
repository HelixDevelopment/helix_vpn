# Round 1 — Independent QA Review: HelixVPN MVP Test Package

**Reviewer:** Independent adversarial QA reviewer (Kimi Code CLI subagent)  
**Scope:** `docs/research/mvp/final/implementation/09-testing-qa/`, `submodules/challenges/helix_vpn/`, `submodules/helix_qa/banks/helix_vpn/`, `docs/reviews/mvp-final/findings/phase4-qa-coverage-report.md`  
**Started:** 2026-07-05T19:17:34Z  
**Finished:** 2026-07-05T19:25:00Z  
**Verdict:** **GO-with-conditions**

---

## Executive summary

The HelixVPN MVP QA package is structurally coherent and honest about its scaffolding state. All four bank files parse cleanly, the six critical-path Challenge IDs and eight HelixQA test-case IDs are present, and the implementation README correctly describes the test pyramid, evidence model, and CI/CD gate mapping. GAP-6 DDoS ownership is explicitly assigned to NFR-413/NFR-414 in the coverage ledger.

However, the coverage ledger is not traceable to the actual bank contents: it references 133 Challenge IDs and 135 HelixQA IDs that do not exist in the banks, while the 6 Challenge IDs and 8 HelixQA IDs that *do* exist in the banks are not referenced in the ledger. The HelixQA bank also omits a top-level `description` field for every test case, and the NFR-413/NFR-414 HelixQA IDs differ between the ledger and the bank. These are blockers to calling the package release-ready.

---

## Check results

### 1. Parse `helix_vpn_challenges.json` — PASS

```bash
python3 -c "import json; json.load(open('submodules/challenges/helix_vpn/helix_vpn_challenges.json'))"
```

Result: parses as a dict with keys `version`, `name`, `metadata`, `challenges`.

### 2. Parse `helix_vpn_challenges.yaml` — PASS

```bash
python3 -c "import yaml; yaml.safe_load(open('submodules/challenges/helix_vpn/helix_vpn_challenges.yaml'))"
```

Result: parses as a dict; structure mirrors JSON.

### 3. Parse `helix_vpn_bank.json` — PASS

```bash
python3 -c "import json; json.load(open('submodules/helix_qa/banks/helix_vpn/helix_vpn_bank.json'))"
```

Result: parses as a dict with keys `version`, `name`, `description`, `metadata`, `test_cases`.

### 4. Parse `helix_vpn_bank.yaml` — PASS

```bash
python3 -c "import yaml; yaml.safe_load(open('submodules/helix_qa/banks/helix_vpn/helix_vpn_bank.yaml'))"
```

Result: parses as a dict; structure mirrors JSON.

### 5. Coverage ledger maps DDOS ownership to NFR-413/NFR-414 — PARTIAL PASS

The ledger explicitly states:

| Requirement | Subsystem | Test types | Status | Challenge IDs | HelixQA bank IDs | DoD / Gate |
|---|---|---|---|---|---|---|
| **HVPN-NFR-413** control-plane API rate limiting | `svc-api` / Redis | **DDOS**, STR, SEC | **defined** | `HVPN-CHAL-NFR413-API-Rate-Limit` | `HVPN-HQA-NFR413-Token-Bucket` | **GAP-6 owner** |
| **HVPN-NFR-414** data-plane DDoS resilience | edge / `helix-transport` | **DDOS**, SEC | **defined** | `HVPN-CHAL-NFR414-Edge-Flood` | `HVPN-HQA-NFR414-Fail-Static` | **GAP-6 owner** |

DDOS ownership is clearly assigned. **But** the actual HelixQA bank uses different IDs (`HVPN-HQA-NFR413-API-Rate-Limit`, `HVPN-HQA-NFR414-Edge-Flood`), and the Challenge bank contains no NFR-413/NFR-414 entries at all.

### 6. Every Challenge and HelixQA item has ID, description, evidence model, acceptance criterion — PARTIAL PASS

**Challenges:** every challenge has `id`, `description`, `evidence`, and `assertions` (acceptance criteria). PASS.

**HelixQA:** every test case has `id`, `steps`, `expected_result`, and `evidence`. **FAIL on `description`:** the test-case schema has no top-level `description` field. The `name` and `expected_result` partially cover the intent, but the stated check requires an explicit description.

### 7. Six critical-path challenges and eight HelixQA test cases exist — PASS

Challenge IDs found:

- `HVPN-CHAL-AUTH-TUNNEL`
- `HVPN-CHAL-RECONNECT-ROAMING`
- `HVPN-CHAL-KILL-SWITCH`
- `HVPN-CHAL-DNS-LEAK`
- `HVPN-CHAL-CONTROL-PLANE-HA`
- `HVPN-CHAL-CLIENT-UI-VISUAL`

HelixQA IDs found:

- `HVPN-HQA-AUTH-TUNNEL`
- `HVPN-HQA-RECONNECT-ROAMING`
- `HVPN-HQA-KILL-SWITCH`
- `HVPN-HQA-DNS-LEAK`
- `HVPN-HQA-CONTROL-PLANE-HA`
- `HVPN-HQA-CLIENT-UI-VISUAL`
- `HVPN-HQA-NFR413-API-Rate-Limit`
- `HVPN-HQA-NFR414-Edge-Flood`

Counts match the requirement.

### 8. Testing README describes test pyramid, evidence model, and CI/CD gate mapping — PASS

`docs/research/mvp/final/implementation/09-testing-qa/README.md` contains:

- A visual test pyramid (UNIT → INT → E2E → SEC/CHAOS/STRESS → CHAL/HQA → REC/UI) in §2.
- An evidence-model table mapping feature classes to artifact classes in §5.
- CI/CD integration plan with local gate layers (`pre-commit`, `pre-push`, `make test`, `make qa`) in §4.

---

## Detailed findings

### Finding 1 — Coverage ledger is not traceable to the actual banks (HIGH)

**Evidence:**

```bash
$ python3 - <<'PY'
import json, re
from pathlib import Path
chal = json.loads(Path("submodules/challenges/helix_vpn/helix_vpn_challenges.json").read_text())
hqa  = json.loads(Path("submodules/helix_qa/banks/helix_vpn/helix_vpn_bank.json").read_text())
ledger = Path("docs/research/mvp/final/implementation/09-testing-qa/coverage-ledger.md").read_text()
chal_ids = {c['id'] for c in chal['challenges']}
hqa_ids  = {t['id'] for t in hqa['test_cases']}
ledger_chal = set(re.findall(r'`(HVPN-CHAL-[\w-]+)`', ledger))
ledger_hqa  = set(re.findall(r'`(HVPN-HQA-[\w-]+)`', ledger))
print("Challenge IDs in ledger but not bank:", len(ledger_chal - chal_ids))
print("Challenge IDs in bank but not ledger:", len(chal_ids - ledger_chal))
print("HelixQA IDs in ledger but not bank: ", len(ledger_hqa - hqa_ids))
print("HelixQA IDs in bank but not ledger: ", len(hqa_ids - ledger_hqa))
PY
Challenge IDs in ledger but not bank: 133
Challenge IDs in bank but not ledger: 6
HelixQA IDs in ledger but not bank:  135
HelixQA IDs in bank but not ledger:  8
```

The ledger claims 91 Challenge IDs and 91 HelixQA IDs minted, but the actual banks contain only the 6 critical-path Challenge IDs and 8 HelixQA IDs. Conversely, the IDs that *do* exist in the banks (`HVPN-CHAL-AUTH-TUNNEL`, `HVPN-HQA-AUTH-TUNNEL`, etc.) are absent from the ledger. The Phase 4 report acknowledges that the banks are "spec-level skeletons," but the ledger presents a fully enumerated coverage claim that cannot be validated against the committed artifacts.

**Recommendation:** Align the ledger with the actual bank contents. Either (a) add the 6 Challenge and 8 HelixQA critical-path IDs to the ledger and remove or mark the 130+ un-minted IDs as `not-defined`, or (b) expand the banks to include the per-FR/NFR IDs referenced by the ledger. Option (a) is preferred for MVP honesty.

### Finding 2 — HelixQA test cases lack a top-level `description` field (MEDIUM)

**Evidence:**

```bash
$ python3 - <<'PY'
import json
hqa = json.load(open("submodules/helix_qa/banks/helix_vpn/helix_vpn_bank.json"))
for t in hqa['test_cases']:
    print(t['id'], 'description' in t)
PY
HVPN-HQA-AUTH-TUNNEL False
HVPN-HQA-RECONNECT-ROAMING False
HVPN-HQA-KILL-SWITCH False
HVPN-HQA-DNS-LEAK False
HVPN-HQA-CONTROL-PLANE-HA False
HVPN-HQA-CLIENT-UI-VISUAL False
HVPN-HQA-NFR413-API-Rate-Limit False
HVPN-HQA-NFR414-Edge-Flood False
```

Each test case has a `name`, `steps`, and `expected_result`, but no `description` key. The review checklist explicitly requires "ID, description, evidence model, acceptance criterion" for every item.

**Recommendation:** Add a concise `description` field to each HelixQA test case summarizing the anti-bluff claim it proves.

### Finding 3 — NFR-413/NFR-414 bank IDs are inconsistent and Challenge bank omits DDoS entries (MEDIUM)

**Evidence:**

| Artifact | NFR-413 ID | NFR-414 ID |
|---|---|---|
| Coverage ledger Challenge cell | `HVPN-CHAL-NFR413-API-Rate-Limit` | `HVPN-CHAL-NFR414-Edge-Flood` |
| Coverage ledger HelixQA cell | `HVPN-HQA-NFR413-Token-Bucket` | `HVPN-HQA-NFR414-Fail-Static` |
| Actual HelixQA bank | `HVPN-HQA-NFR413-API-Rate-Limit` | `HVPN-HQA-NFR414-Edge-Flood` |
| Actual Challenge bank | *(none)* | *(none)* |

The ledger maps GAP-6 DDoS ownership correctly at the requirement level, but the concrete IDs drift between documents, and the Challenge bank has no DDoS challenge entries at all despite the ledger assigning `HVPN-CHAL-NFR413-API-Rate-Limit` and `HVPN-CHAL-NFR414-Edge-Flood`.

**Recommendation:** Choose one canonical ID pair and apply it to the ledger, the HelixQA bank, and the Challenge bank. Add the two DDoS Challenge entries to `helix_vpn_challenges.json/yaml` so the ledger's Challenge references resolve.

### Finding 4 — Driver skeletons are not executable (LOW)

**Evidence:**

```bash
$ ls -l submodules/challenges/helix_vpn/drivers/*.sh | awk '{print $1, $NF}'
-rw-r--r-- auth_tunnel_establish.sh
-rw-r--r-- client_ui_visual.sh
-rw-r--r-- control_plane_ha.sh
-rw-r--r-- dns_leak.sh
-rw-r--r-- kill_switch.sh
-rw-r--r-- reconnect_roaming.sh
```

None of the six driver skeletons have the executable bit set. They are acknowledged placeholders, but a runnable harness cannot invoke them without `chmod +x` or an explicit shell interpreter.

**Recommendation:** Either set executable permissions on the skeletons or document that they are intentionally non-executable until implemented.

---

## Verdict rationale

**GO-with-conditions** is chosen because:

- The bank files are syntactically valid and loadable.
- The six critical-path challenges and eight HelixQA test cases exist with clear evidence models and acceptance criteria (Challenges) or steps + expected results (HelixQA).
- The implementation README covers the test pyramid, evidence model, and CI/CD gate mapping.
- GAP-6 DDoS ownership is explicitly assigned to NFR-413/NFR-414.
- The package honestly labels itself as `defined`/`not-automated` rather than claiming false PASSes.

The package is **not** a clean GO because the coverage ledger is not traceable to the committed bank contents, the HelixQA schema omits the required `description` field, and the NFR-413/NFR-414 IDs drift between documents.

---

## Conditions for promotion to GO

1. **Fix traceability between ledger and banks:** Ensure every ID referenced in the coverage ledger exists in the corresponding bank, and every ID in the banks is referenced in the ledger. Remove or honestly mark the 130+ ledger-only IDs.
2. **Add HelixQA descriptions:** Add a top-level `description` field to each of the eight HelixQA test cases.
3. **Align NFR-413/NFR-414 IDs and add Challenge entries:** Choose canonical IDs for the two DDoS rows, update both JSON/YAML banks and the ledger consistently, and add the missing Challenge entries to `helix_vpn_challenges.json/yaml`.
4. **(Optional) Executable bits:** Set `+x` on driver skeletons or document their non-executable status.

---

## Top 3 findings

1. **Coverage ledger is not traceable to actual bank contents.** The ledger references 133 Challenge IDs and 135 HelixQA IDs that do not exist in the banks, while the 6 Challenge IDs and 8 HelixQA IDs that do exist are absent from the ledger.
2. **HelixQA test cases lack a top-level `description` field.** All eight test cases have `name`, `steps`, `expected_result`, and `evidence`, but no `description`, failing the stated completeness check.
3. **NFR-413/NFR-414 IDs drift and Challenge bank omits DDoS entries.** The ledger and bank use different HelixQA IDs for the GAP-6 DDoS rows, and the Challenge bank contains no DDoS challenge entries despite the ledger assigning them.
