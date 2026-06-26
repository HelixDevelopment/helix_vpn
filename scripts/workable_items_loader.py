#!/usr/bin/env python3
# ============================================================================
# workable_items_loader.py
# ============================================================================
#
# Purpose:
#   Parse the four HelixVPN WBS Markdown docs and populate the §11.4.93
#   SQLite single-source-of-truth at docs/workable_items.db. This is the
#   md-to-db direction of the bidirectional sync contract.
#
# Usage:
#   python3 scripts/workable_items_loader.py [--dry-run]
#
# Inputs:
#   docs/research/mvp/final/06-phase0-spike-wbs.md  (P0)
#   docs/research/mvp/final/07-phase1-mvp-wbs.md    (P1)
#   docs/research/mvp/final/08-phase2-parity-wbs.md (P2)
#   docs/research/mvp/final/09-phase3-reach-wbs.md  (P3)
#
# Outputs:
#   docs/workable_items.db (SQLite, git-tracked per §11.4.95)
#
# Cross-references:
#   - Constitution §11.4.93 (SQLite-backed SSoT)
#   - Constitution §11.4.95 (DB tracked in git)
#   - Constitution §11.4.54 (HVPN-Pn-NNN id convention)
#   - Constitution §11.4.148 (integrity contract)
#   - Constitution §11.4.91 (description clarity)
#   - Spec: v07-execution/workable-items-model.md
# ============================================================================

import json
import os
import re
import sqlite3
import sys
from datetime import datetime, timezone
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
DB_PATH = REPO_ROOT / "docs" / "workable_items.db"
WBS_FILES = {
    "P0": REPO_ROOT / "docs/research/mvp/final/06-phase0-spike-wbs.md",
    "P1": REPO_ROOT / "docs/research/mvp/final/07-phase1-mvp-wbs.md",
    "P2": REPO_ROOT / "docs/research/mvp/final/08-phase2-parity-wbs.md",
    "P3": REPO_ROOT / "docs/research/mvp/final/09-phase3-reach-wbs.md",
}

DEEPENING_FILES = {
    "P1": REPO_ROOT / "docs/research/mvp/final/v07-execution/subtask-deepening-p1.md",
    "P2": REPO_ROOT / "docs/research/mvp/final/v07-execution/subtask-deepening-p2.md",
    "P3": REPO_ROOT / "docs/research/mvp/final/v07-execution/subtask-deepening-p3.md",
}

# Regex for parent task heading in deepening docs
DEEPENING_PARENT_RE = re.compile(
    r'^\*\*(HVPN-P\d-\d{3})\s*[—–-]\s*(.+?)\*\*'
)

# Regex for subtask table rows in deepening docs
SUBTASK_ROW_RE = re.compile(
    r'^\|\s*`\.(\d+)`\s*\|\s*(.+?)\s*\|\s*(.+?)\s*\|\s*(.+?)\s*\|\s*(\w+)\s*\|'
)

CX_MAP = {"XS": 0.5, "S": 1.5, "M": 2.5, "L": 4}

# Regex for task/subtask headings (Phase 0 uses ### TASK headings)
TASK_RE = re.compile(
    r'^###\s+(?:TASK|SUBTASK)\s+(HVPN-P(\d)-(\d{3}(?:\.\d+)?))\s*[—–-]\s*(.+)$'
)

# Regex for inline bold tasks (Phase 1+ uses `- **HVPN-P1-NNN — Title.**`)
INLINE_TASK_RE = re.compile(
    r'^[-*]\s+\*\*(HVPN-P(\d)-(\d{3}(?:\.\d+)?))\s*[—–-]\s*(.+?)\.\*\*'
)

# Regex for gate headings (Phase 0: `## 1. Exit gates`)
GATE_RE = re.compile(
    r'^###?\s+(?:Exit\s+)?[Gg]ate\s+(G\d+)\s*[—–-]\s*(.+)$'
)

# Regex for epic/milestone headings in Phase 0
MILESTONE_RE = re.compile(
    r'^##\s+\d+\.\s+Milestone\s+(S\d+)\s*[—–-]\s*(.+?)(?:\s*\(HVPN-P0-\d+\.\.\d+\))?\s*(?:·.*)?$'
)

# Regex for epic headings in Phase 1+
EPIC_RE = re.compile(
    r'^##\s+\d+\.\s+(E\d+)\s*[—–-]\s*(.+)$'
)

# Field extractors from task body
FIELD_PATTERNS = {
    "module": re.compile(r'^\s*[·•-]\s*module:\s*(.+)$', re.IGNORECASE | re.MULTILINE),
    "gate": re.compile(r'[·•-]\s*(?:gate|DoD):\s*(G\d+)', re.IGNORECASE),
    "type": re.compile(r'[·•-]\s*type:\s*(Bug|Feature|Task)', re.IGNORECASE),
    "severity": re.compile(r'[·•-]\s*severity:\s*(Critical|normal|high|low)', re.IGNORECASE),
    "deps": re.compile(r'[·•-]\s*(?:deps|depends?[\s_]on|blocked[\s_]by):\s*(.+)$', re.IGNORECASE),
    "deliverable": re.compile(r'[·•-]\s*(?:deliverable|produces?|artefact):\s*(.+)$', re.IGNORECASE),
    "acceptance": re.compile(r'[·•-]\s*(?:acceptance|AC|verdict):\s*(.+)$', re.IGNORECASE),
    "effort": re.compile(r'[·•-]\s*(?:effort|est(?:imate)?|size):\s*(?:([XSML]+)\((\d+)\)|(\d+(?:\.\d+)?)\s*(?:days?|d))', re.IGNORECASE),
    "test_types": re.compile(r'[·•-]\s*(?:tests?|test[\s_]types?):\s*(.+)$', re.IGNORECASE),
    "dod_refs": re.compile(r'[·•-]\s*(?:DoD|SLO|AC)\s*(?:ref)?:\s*(.+)$', re.IGNORECASE),
    "source_refs": re.compile(r'\[([^\]]*(?:§|P[0-3])[^\]]*)\]'),
}

TSHIRT_MAP = {"XS": 1, "S": 2, "M": 5, "L": 10, "XL": 15}

TEST_TYPE_NORMALIZE = {
    "UT": "UNIT", "IT": "INT", "E2E": "E2E", "CONC": "CONCURRENCY",
    "CH": "CHAL", "HQA": "HELIXQA", "SC": "SECURITY", "ST": "STRESS",
    "DDOS": "DDOS", "CHAOS": "CHAOS", "RACE": "RACE", "MEM": "MEMORY",
    "BENCH": "BENCHMARK", "SCALE": "SCALE", "FUZZ": "FUZZ",
    "UNIT": "UNIT", "INT": "INT", "CHAL": "CHAL", "HELIXQA": "HELIXQA",
    "SECURITY": "SECURITY", "STRESS": "STRESS", "BENCHMARK": "BENCHMARK",
    "CONCURRENCY": "CONCURRENCY", "RACE": "RACE", "MEMORY": "MEMORY",
}


def normalize_test_types(raw: str) -> list:
    """Parse and normalize test type codes from a raw string."""
    if not raw or raw.strip() in ("—", "-", "TBD", "TBD.", "none"):
        return []
    codes = re.split(r'[,;/\s]+', raw.strip().strip('.'))
    result = []
    for c in codes:
        c = c.strip().upper().replace("-", "_")
        if c in TEST_TYPE_NORMALIZE:
            result.append(TEST_TYPE_NORMALIZE[c])
        elif c:
            result.append(c)
    return sorted(set(result))


def parse_deps(raw: str) -> list:
    """Parse dependency references from a raw string."""
    if not raw or raw.strip() in ("—", "-", "none", "TBD"):
        return []
    ids = re.findall(r'HVPN-P\d-\d{3}(?:\.\d+)?', raw)
    return sorted(set(ids))


def parse_json_array(raw: str) -> list:
    """Parse a comma/semicolon-separated list into a JSON array."""
    if not raw or raw.strip() in ("—", "-", "none", "TBD"):
        return []
    items = re.split(r'[,;]+', raw.strip())
    return [i.strip() for i in items if i.strip()]


def extract_fields(body: str) -> dict:
    """Extract structured fields from a task body block."""
    fields = {}

    # Module
    m = FIELD_PATTERNS["module"].search(body)
    fields["module"] = m.group(1).strip() if m else ""

    # Gate
    m = FIELD_PATTERNS["gate"].search(body)
    fields["gate"] = m.group(1).strip() if m else None

    # Type
    m = FIELD_PATTERNS["type"].search(body)
    fields["type"] = m.group(1).strip().capitalize() if m else "Task"

    # Severity
    m = FIELD_PATTERNS["severity"].search(body)
    fields["severity"] = m.group(1).strip().lower() if m else "normal"

    # Deps
    m = FIELD_PATTERNS["deps"].search(body)
    fields["deps"] = parse_deps(m.group(1)) if m else []

    # Deliverable
    m = FIELD_PATTERNS["deliverable"].search(body)
    fields["deliverable"] = m.group(1).strip() if m else "See acceptance criteria"

    # Acceptance
    m = FIELD_PATTERNS["acceptance"].search(body)
    fields["acceptance"] = m.group(1).strip() if m else "Captured evidence per §11.4.5"

    # Effort
    m = FIELD_PATTERNS["effort"].search(body)
    if m:
        if m.group(1):  # T-shirt
            fields["effort_days"] = TSHIRT_MAP.get(m.group(1).upper(), 5)
        elif m.group(3):  # Numeric days
            fields["effort_days"] = float(m.group(3))
    else:
        fields["effort_days"] = 1.0

    # Test types
    m = FIELD_PATTERNS["test_types"].search(body)
    fields["test_types"] = normalize_test_types(m.group(1)) if m else []

    # DoD refs
    m = FIELD_PATTERNS["dod_refs"].search(body)
    fields["dod_refs"] = parse_json_array(m.group(1)) if m else []

    # Source refs
    fields["source_refs"] = FIELD_PATTERNS["source_refs"].findall(body)

    return fields


def parse_wbs(filepath: Path, phase: str) -> list:
    """Parse a WBS markdown file and return a list of item dicts."""
    text = filepath.read_text(encoding="utf-8")
    lines = text.splitlines()
    items = []
    current_epic = ""
    current_module = ""

    i = 0
    while i < len(lines):
        line = lines[i]

        # Check for epic/milestone heading
        m = MILESTONE_RE.match(line)
        if m:
            current_epic = m.group(1)
            # Extract module from milestone title if present
            title = m.group(2).strip()
            items.append({
                "atm_id": f"HVPN-P0-{current_epic}",
                "parent_id": None,
                "phase": "P0",
                "kind": "milestone",
                "title": title[:200],
                "description": f"Milestone {current_epic}: {title}. " +
                               "Contains tasks that must be completed to pass the milestone gate.",
                "status": "Queued",
                "type": "Task",
                "severity": "normal",
                "epic": current_epic,
                "module": "",
                "gate": None,
                "deps": [],
                "deliverable": "See constituent tasks",
                "acceptance": "All constituent tasks completed with captured evidence",
                "effort_days": 0,
                "test_types": [],
                "dod_refs": [],
                "source_refs": [],
            })
            i += 1
            continue

        m = EPIC_RE.match(line)
        if m:
            current_epic = m.group(1)
            title = m.group(2).strip()
            # Try to extract module from the epic body
            body_lines = []
            j = i + 1
            while j < len(lines) and not lines[j].startswith("## "):
                body_lines.append(lines[j])
                j += 1
            body = "\n".join(body_lines)
            fm = FIELD_PATTERNS["module"].search(body)
            current_module = fm.group(1).strip() if fm else current_epic.lower()

            items.append({
                "atm_id": f"HVPN-{phase}-{current_epic}",
                "parent_id": None,
                "phase": phase,
                "kind": "epic",
                "title": title[:200],
                "description": f"Epic {current_epic}: {title}. " +
                               "Groups related work items for the control plane, data plane, or client.",
                "status": "Queued",
                "type": "Task",
                "severity": "normal",
                "epic": current_epic,
                "module": current_module,
                "gate": None,
                "deps": [],
                "deliverable": "See constituent tasks",
                "acceptance": "All constituent tasks completed with captured evidence",
                "effort_days": 0,
                "test_types": [],
                "dod_refs": [],
                "source_refs": [],
            })
            i += 1
            continue

        # Check for task/subtask heading (Phase 0 ### TASK format)
        m = TASK_RE.match(line)
        if m:
            atm_id = m.group(1)
            phase_tag = f"P{m.group(2)}"
            title = m.group(4).strip()

            # Gather body until next ### or ## heading
            body_lines = []
            j = i + 1
            while j < len(lines):
                if lines[j].startswith("### ") or lines[j].startswith("## "):
                    break
                body_lines.append(lines[j])
                j += 1
            body = "\n".join(body_lines)

            fields = extract_fields(body)

            # Determine parent
            parent_id = None
            if "." in atm_id:
                parent_id = atm_id.rsplit(".", 1)[0]
            elif current_epic:
                parent_id = f"HVPN-P0-{current_epic}" if phase == "P0" else f"HVPN-{phase_tag}-{current_epic}"

            # Build description from body
            desc_lines = [l.strip() for l in body.split("\n") if l.strip()
                          and not l.strip().startswith("·") and not l.strip().startswith("•")
                          and not l.strip().startswith("- ") and not l.strip().startswith("|")]
            desc_text = " ".join(desc_lines[:5]).strip()
            if len(desc_text) < 40:
                desc_text = f"Task {atm_id}: {title}. " + desc_text
            if len(desc_text) < 40:
                desc_text = desc_text + " Implementation details in the WBS document."
            desc_text = desc_text[:2000]

            kind = "subtask" if "." in atm_id else "task"

            items.append({
                "atm_id": atm_id,
                "parent_id": parent_id,
                "phase": phase_tag,
                "kind": kind,
                "title": title[:500],
                "description": desc_text,
                "status": "Queued",
                "type": fields.get("type", "Task"),
                "severity": fields.get("severity", "normal"),
                "epic": current_epic,
                "module": fields.get("module", "") or current_module,
                "gate": fields.get("gate"),
                "deps": fields.get("deps", []),
                "deliverable": fields.get("deliverable", "See acceptance criteria"),
                "acceptance": fields.get("acceptance", "Captured evidence per §11.4.5"),
                "effort_days": fields.get("effort_days", 1.0),
                "test_types": fields.get("test_types", []),
                "dod_refs": fields.get("dod_refs", []),
                "source_refs": fields.get("source_refs", []),
            })

            i = j
            continue

        # Check for inline bold tasks (Phase 1+ format: `- **HVPN-P1-NNN — Title.**`)
        m = INLINE_TASK_RE.match(line)
        if m:
            atm_id = m.group(1)
            phase_tag = f"P{m.group(2)}"
            title = m.group(4).strip()

            # Gather body: the rest of the bold line + indented continuation lines
            full_line = line
            body_lines = [full_line]
            j = i + 1
            while j < len(lines):
                next_line = lines[j]
                # Stop at next task, heading, or non-indented non-empty line
                if next_line.startswith("- **HVPN-") or next_line.startswith("## "):
                    break
                if next_line.startswith("### "):
                    break
                body_lines.append(next_line)
                j += 1
            body = "\n".join(body_lines)

            fields = extract_fields(body)

            # Determine parent
            parent_id = None
            if "." in atm_id:
                parent_id = atm_id.rsplit(".", 1)[0]
            elif current_epic:
                parent_id = f"HVPN-{phase_tag}-{current_epic}"

            # Build description from the inline text
            desc_text = re.sub(r'\*\*', '', full_line).strip().lstrip("- ").strip()
            if len(desc_text) < 40:
                desc_text = f"Task {atm_id}: {title}. " + desc_text
            if len(desc_text) < 40:
                desc_text = desc_text + " Implementation details in the WBS document."
            desc_text = desc_text[:2000]

            kind = "subtask" if "." in atm_id else "task"

            items.append({
                "atm_id": atm_id,
                "parent_id": parent_id,
                "phase": phase_tag,
                "kind": kind,
                "title": title[:500],
                "description": desc_text,
                "status": "Queued",
                "type": fields.get("type", "Task"),
                "severity": fields.get("severity", "normal"),
                "epic": current_epic,
                "module": fields.get("module", "") or current_module,
                "gate": fields.get("gate"),
                "deps": fields.get("deps", []),
                "deliverable": fields.get("deliverable", "See acceptance criteria"),
                "acceptance": fields.get("acceptance", "Captured evidence per §11.4.5"),
                "effort_days": fields.get("effort_days", 1.0),
                "test_types": fields.get("test_types", []),
                "dod_refs": fields.get("dod_refs", []),
                "source_refs": fields.get("source_refs", []),
            })

            i = j
            continue

        i += 1

    return items


def parse_deepening(filepath: Path, phase: str) -> list:
    """Parse a subtask deepening markdown file (table format) and return subtask dicts."""
    if not filepath.exists():
        return []
    text = filepath.read_text(encoding="utf-8")
    lines = text.splitlines()
    items = []
    current_parent = None
    current_parent_title = ""

    i = 0
    while i < len(lines):
        line = lines[i]

        # Check for parent task heading
        m = DEEPENING_PARENT_RE.match(line)
        if m:
            current_parent = m.group(1)
            current_parent_title = m.group(2).strip().rstrip("(").strip()
            i += 1
            continue

        # Check for subtask table row
        m = SUBTASK_ROW_RE.match(line)
        if m and current_parent:
            subtask_num = m.group(1)
            title = m.group(2).strip()
            acceptance = m.group(3).strip()
            test_types_raw = m.group(4).strip()
            cx = m.group(5).strip()

            atm_id = f"{current_parent}.{subtask_num}"
            test_types = normalize_test_types(test_types_raw)
            effort_days = CX_MAP.get(cx, 1.0)

            # Build description
            desc = f"Subtask of {current_parent}: {title}. Acceptance: {acceptance}"
            if len(desc) < 40:
                desc = desc + " See parent task for full context."
            desc = desc[:2000]

            items.append({
                "atm_id": atm_id,
                "parent_id": current_parent,
                "phase": phase,
                "kind": "subtask",
                "title": title[:500],
                "description": desc,
                "status": "Queued",
                "type": "Task",
                "severity": "normal",
                "epic": "",
                "module": "",
                "gate": None,
                "deps": [],
                "deliverable": "See acceptance criteria",
                "acceptance": acceptance,
                "effort_days": effort_days,
                "test_types": test_types,
                "dod_refs": [],
                "source_refs": [],
            })

            i += 1
            continue

        i += 1

    return items


def load_into_db(all_items: list, dry_run: bool = False):
    """Load parsed items into the SQLite DB."""
    if dry_run:
        print(f"DRY RUN: would insert {len(all_items)} items")
        for item in all_items[:5]:
            print(f"  {item['atm_id']}: {item['title'][:60]}")
        if len(all_items) > 5:
            print(f"  ... and {len(all_items) - 5} more")
        return

    conn = sqlite3.connect(str(DB_PATH))
    conn.execute("PRAGMA journal_mode=WAL")
    conn.execute("PRAGMA foreign_keys=ON")

    # Clear existing items (re-load is idempotent)
    conn.execute("DELETE FROM item_history")
    conn.execute("DELETE FROM operator_block_details")
    conn.execute("DELETE FROM obsolete_details")
    conn.execute("DELETE FROM items")

    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    inserted = 0
    skipped = 0

    for item in all_items:
        try:
            conn.execute("""
                INSERT INTO items (
                    atm_id, parent_id, phase, kind, title, description,
                    status, type, severity, epic, module, gate, deps,
                    deliverable, acceptance, effort_days, test_types,
                    dod_refs, source_refs, created_at, modified_at,
                    created_by, assigned_to
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '')
            """, (
                item["atm_id"],
                item["parent_id"],
                item["phase"],
                item["kind"],
                item["title"],
                item["description"],
                item["status"],
                item["type"],
                item["severity"],
                item["epic"],
                item["module"],
                item["gate"],
                json.dumps(item["deps"]),
                item["deliverable"],
                item["acceptance"],
                item["effort_days"],
                json.dumps(item["test_types"]),
                json.dumps(item["dod_refs"]),
                json.dumps(item["source_refs"]),
                now,
                now,
            ))
            inserted += 1
        except sqlite3.IntegrityError as e:
            print(f"SKIP {item['atm_id']}: {e}", file=sys.stderr)
            skipped += 1

    # Update meta
    import hashlib
    keyset = sorted(i["atm_id"] for i in all_items)
    integrity = hashlib.sha256("\n".join(keyset).encode()).hexdigest()
    conn.execute("DELETE FROM meta")
    conn.execute(
        "INSERT INTO meta (schema_version, last_sync_at, integrity_hash) VALUES (?, ?, ?)",
        ("1.0.0", now, integrity),
    )

    conn.commit()
    conn.close()
    print(f"Loaded {inserted} items, skipped {skipped} (integrity: {integrity[:16]}…)")


def main():
    dry_run = "--dry-run" in sys.argv

    all_items = []
    for phase, filepath in WBS_FILES.items():
        if not filepath.exists():
            print(f"WARN: {filepath} not found, skipping {phase}", file=sys.stderr)
            continue
        items = parse_wbs(filepath, phase)
        print(f"Parsed {phase} WBS: {len(items)} items from {filepath.name}")
        all_items.extend(items)

    # Parse subtask deepening docs
    for phase, filepath in DEEPENING_FILES.items():
        if not filepath.exists():
            continue
        items = parse_deepening(filepath, phase)
        print(f"Parsed {phase} deepening: {len(items)} subtasks from {filepath.name}")
        all_items.extend(items)

    print(f"Total: {len(all_items)} items across all phases")

    load_into_db(all_items, dry_run)


if __name__ == "__main__":
    main()
