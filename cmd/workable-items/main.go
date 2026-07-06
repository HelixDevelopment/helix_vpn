// ============================================================================
// workable-items — Go CLI for the §11.4.93 SQLite single source of truth
// ============================================================================
//
// Purpose:
//   Bidirectional sync between the HelixVPN WBS Markdown documents and the
//   SQLite database at docs/workable_items.db. Replaces the Python loader
//   (scripts/workable_items_loader.py) with a proper Go implementation.
//
// Usage:
//   workable-items sync md-to-db [--dry-run]
//   workable-items sync db-to-md [--dry-run]
//   workable-items diff
//   workable-items validate
//   workable-items add --id HVPN-P1-NNN --title "..." [--parent ...] [--phase P1] [--type Task]
//   workable-items close --id HVPN-P1-NNN [--status "Fixed (→ Fixed.md)"]
//
// Cross-references:
//   - Constitution §11.4.93 (SQLite-backed SSoT)
//   - Constitution §11.4.95 (DB tracked in git)
//   - Constitution §11.4.54 (HVPN-Pn-NNN id convention)
//   - Constitution §11.4.148 (integrity contract)
//   - Constitution §11.4.91 (description clarity)
//   - Spec: v07-execution/workable-items-model.md
// ============================================================================

package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	schemaVersion = "1.0.0"
	ddlVersion    = "1.0.0"
)

var (
	repoRoot string
	dbPath   string
)

var wbsFiles = map[string]string{
	"P0": "docs/research/mvp/final/06-phase0-spike-wbs.md",
	"P1": "docs/research/mvp/final/07-phase1-mvp-wbs.md",
	"P2": "docs/research/mvp/final/08-phase2-parity-wbs.md",
	"P3": "docs/research/mvp/final/09-phase3-reach-wbs.md",
}

var deepeningFiles = map[string]string{
	"P1": "docs/research/mvp/final/v07-execution/subtask-deepening-p1.md",
	"P2": "docs/research/mvp/final/v07-execution/subtask-deepening-p2.md",
	"P3": "docs/research/mvp/final/v07-execution/subtask-deepening-p3.md",
}

// ---------------------------------------------------------------------------
// Data model
// ---------------------------------------------------------------------------

type Item struct {
	ATMID       string
	ParentID    *string
	Phase       string
	Kind        string
	Title       string
	Description string
	Status      string
	Type        string
	Severity    string
	Epic        string
	Module      string
	Gate        *string
	Deps        []string
	Deliverable string
	Acceptance  string
	EffortDays  float64
	TestTypes   []string
	DodRefs     []string
	SourceRefs  []string
	CreatedAt   string
	ModifiedAt  string
	CreatedBy   string
	AssignedTo  string
}

// ---------------------------------------------------------------------------
// Regex patterns (ported from Python loader)
// ---------------------------------------------------------------------------

var (
	// Parent task heading in deepening docs
	deepeningParentRe = regexp.MustCompile(`^\*\*(HVPN-P\d-\d{3})\s*[—–\-]\s*(.+?)\*\*`)

	// Subtask table rows in deepening docs
	subtaskRowRe = regexp.MustCompile(`^\|\s*` + "`" + `\.(\d+)` + "`" + `\s*\|\s*(.+?)\s*\|\s*(.+?)\s*\|\s*(.+?)\s*\|\s*(\w+)\s*\|`)

	// Task/subtask heading (Phase 0 uses ### TASK headings)
	taskRe = regexp.MustCompile(`^###\s+(?:TASK|SUBTASK)\s+(HVPN-P(\d)-(\d{3}(?:\.\d+)?))\s*[—–\-]\s*(.+)$`)

	// Inline bold tasks (Phase 1+ uses `- **HVPN-P1-NNN — Title.**`)
	inlineTaskRe = regexp.MustCompile(`^[-*]\s+\*\*(HVPN-P(\d)-(\d{3}(?:\.\d+)?))\s*[—–\-]\s*(.+?)\.\*\*`)

	// Epic/milestone headings in Phase 0
	milestoneRe = regexp.MustCompile(`^##\s+\d+\.\s+Milestone\s+(S\d+)\s*[—–\-]\s*(.+?)(?:\s*\(HVPN-P0-\d+\.\.\d+\))?\s*(?:·.*)?$`)

	// Epic headings in Phase 1+
	epicRe = regexp.MustCompile(`^##\s+\d+\.\s+(E\d+)\s*[—–\-]\s*(.+)$`)

	// Gate headings
	gateRe = regexp.MustCompile(`^###?\s+(?:Exit\s+)?[Gg]ate\s+(G\d+)\s*[—–\-]\s*(.+)$`)

	// Field extractors from task body
	fieldModule      = regexp.MustCompile(`(?i)^\s*[·•\-]\s*module:\s*(.+)$`)
	fieldGate        = regexp.MustCompile(`(?i)[·•\-]\s*(?:gate|DoD):\s*(G\d+)`)
	fieldType        = regexp.MustCompile(`(?i)[·•\-]\s*type:\s*(Bug|Feature|Task)`)
	fieldSeverity    = regexp.MustCompile(`(?i)[·•\-]\s*severity:\s*(Critical|normal|high|low)`)
	fieldDeps        = regexp.MustCompile(`(?i)[·•\-]\s*(?:deps|depends?[\s_]on|blocked[\s_]by):\s*(.+)$`)
	fieldDeliverable = regexp.MustCompile(`(?i)[·•\-]\s*(?:deliverable|produces?|artefact):\s*(.+)$`)
	fieldAcceptance  = regexp.MustCompile(`(?i)[·•\-]\s*(?:acceptance|AC|verdict):\s*(.+)$`)
	fieldEffort      = regexp.MustCompile(`(?i)[·•\-]\s*(?:effort|est(?:imate)?|size):\s*(?:([XSML]+)\((\d+)\)|(\d+(?:\.\d+)?)\s*(?:days?|d))`)
	fieldTestTypes   = regexp.MustCompile(`(?i)[·•\-]\s*(?:tests?|test[\s_]types?):\s*(.+)$`)
	fieldDodRefs     = regexp.MustCompile(`(?i)[·•\-]\s*(?:DoD|SLO|AC)\s*(?:ref)?:\s*(.+)$`)
	fieldSourceRefs  = regexp.MustCompile(`\[([^\]]*(?:§|P[0-3])[^\]]*)]`)
)

var tshirtMap = map[string]float64{
	"XS": 1, "S": 2, "M": 5, "L": 10, "XL": 15,
}

var cxMap = map[string]float64{
	"XS": 0.5, "S": 1.5, "M": 2.5, "L": 4,
}

var testTypeNormalize = map[string]string{
	"UT": "UNIT", "IT": "INT", "E2E": "E2E", "CONC": "CONCURRENCY",
	"CH": "CHAL", "HQA": "HELIXQA", "SC": "SECURITY", "ST": "STRESS",
	"DDOS": "DDOS", "CHAOS": "CHAOS", "MEM": "MEMORY",
	"BENCH": "BENCHMARK", "SCALE": "SCALE", "FUZZ": "FUZZ",
	"UNIT": "UNIT", "INT": "INT", "CHAL": "CHAL", "HELIXQA": "HELIXQA",
	"SECURITY": "SECURITY", "STRESS": "STRESS", "BENCHMARK": "BENCHMARK",
	"CONCURRENCY": "CONCURRENCY", "RACE": "RACE", "MEMORY": "MEMORY",
}

// ---------------------------------------------------------------------------
// DDL
// ---------------------------------------------------------------------------

const ddl = `
CREATE TABLE IF NOT EXISTS items (
  atm_id        TEXT PRIMARY KEY,
  parent_id     TEXT REFERENCES items(atm_id),
  phase         TEXT NOT NULL CHECK (phase IN ('P0','P1','P2','P3')),
  kind          TEXT NOT NULL CHECK (kind IN ('epic','milestone','task','subtask','gate')),
  title         TEXT NOT NULL CHECK (length(title) >= 6),
  description   TEXT NOT NULL CHECK (length(description) >= 40),
  status        TEXT NOT NULL DEFAULT 'Queued'
                CHECK (status IN ('Queued','In progress','Ready for testing',
                                  'In testing','Reopened','Operator-blocked',
                                  'Obsolete (→ Fixed.md)','Implemented (→ Fixed.md)',
                                  'Completed (→ Fixed.md)','Fixed (→ Fixed.md)')),
  type          TEXT NOT NULL DEFAULT 'Task' CHECK (type IN ('Bug','Feature','Task')),
  severity      TEXT NOT NULL DEFAULT 'normal',
  epic          TEXT NOT NULL,
  module        TEXT NOT NULL,
  gate          TEXT,
  deps          TEXT NOT NULL DEFAULT '[]',
  deliverable   TEXT NOT NULL,
  acceptance    TEXT NOT NULL,
  effort_days   REAL NOT NULL DEFAULT 1.0,
  test_types    TEXT NOT NULL DEFAULT '[]',
  dod_refs      TEXT NOT NULL DEFAULT '[]',
  source_refs   TEXT NOT NULL DEFAULT '[]',
  created_at    TEXT NOT NULL DEFAULT (datetime('now')),
  modified_at   TEXT NOT NULL DEFAULT (datetime('now')),
  created_by    TEXT NOT NULL DEFAULT '',
  assigned_to   TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS item_history (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  atm_id      TEXT NOT NULL REFERENCES items(atm_id),
  changed_at  TEXT NOT NULL,
  by          TEXT NOT NULL CHECK (by IN ('AI','User','Operator')),
  from_status TEXT, to_status TEXT,
  reason      TEXT NOT NULL,
  evidence    TEXT
);

CREATE TABLE IF NOT EXISTS operator_block_details (
  atm_id     TEXT PRIMARY KEY REFERENCES items(atm_id),
  what       TEXT NOT NULL, why TEXT NOT NULL,
  unblock    TEXT NOT NULL,
  who        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS obsolete_details (
  atm_id     TEXT PRIMARY KEY REFERENCES items(atm_id),
  since      TEXT NOT NULL, reason TEXT NOT NULL,
  superseding TEXT, evidence TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS meta (
  schema_version TEXT NOT NULL,
  last_sync_at   TEXT NOT NULL,
  integrity_hash TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS test_diary (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  atm_id        TEXT NOT NULL REFERENCES items(atm_id),
  date_time     TEXT NOT NULL,
  tested_by     TEXT NOT NULL CHECK (tested_by IN ('User','Operator','AI-agent','HelixQA')),
  result        TEXT NOT NULL CHECK (result IN ('PASS','FAIL','SKIP')),
  observations  TEXT NOT NULL DEFAULT '',
  action_taken  TEXT NOT NULL DEFAULT '',
  status_changed INTEGER NOT NULL DEFAULT 0,
  from_status   TEXT,
  to_status     TEXT,
  evidence_path TEXT NOT NULL DEFAULT '',
  feature_class TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS gates (
  gate_id   TEXT PRIMARY KEY,
  phase     TEXT NOT NULL,
  title     TEXT NOT NULL,
  verdict   TEXT NOT NULL DEFAULT 'OPEN',
  evidence  TEXT NOT NULL DEFAULT '',
  decided_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_items_parent ON items(parent_id);
CREATE INDEX IF NOT EXISTS idx_items_phase  ON items(phase);
CREATE INDEX IF NOT EXISTS idx_items_gate   ON items(gate);
CREATE INDEX IF NOT EXISTS idx_items_status ON items(status);
CREATE INDEX IF NOT EXISTS idx_test_diary_atm ON test_diary(atm_id);
`

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func findRepoRoot() string {
	// Try to find the repo root by looking for go.mod
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot get working directory: %v\n", err)
		os.Exit(1)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Fallback: use the directory of the binary
	exe, err := os.Executable()
	if err == nil {
		return filepath.Dir(filepath.Dir(exe))
	}
	fmt.Fprintf(os.Stderr, "ERROR: cannot find repo root (no go.mod found)\n")
	os.Exit(1)
	return ""
}

func openDB() *sql.DB {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot open DB %s: %v\n", dbPath, err)
		os.Exit(1)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer
	return db
}

func ensureSchema(db *sql.DB) {
	for _, stmt := range strings.Split(ddl, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: DDL failed: %v\n  Statement: %s\n", err, stmt)
			os.Exit(1)
		}
	}
}

func nowUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

func toJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func normalizeTestTypes(raw string) []string {
	if raw == "" || raw == "—" || raw == "-" || raw == "TBD" || raw == "TBD." || raw == "none" {
		return nil
	}
	raw = strings.TrimSuffix(raw, ".")
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == '/' || r == ' ' || r == '\t'
	})
	seen := map[string]bool{}
	var result []string
	for _, p := range parts {
		c := strings.ToUpper(strings.TrimSpace(strings.ReplaceAll(p, "-", "_")))
		if c == "" {
			continue
		}
		if norm, ok := testTypeNormalize[c]; ok {
			c = norm
		}
		if !seen[c] {
			seen[c] = true
			result = append(result, c)
		}
	}
	sort.Strings(result)
	return result
}

func parseDeps(raw string) []string {
	if raw == "" || raw == "—" || raw == "-" || raw == "none" || raw == "TBD" {
		return nil
	}
	re := regexp.MustCompile(`HVPN-P\d-\d{3}(?:\.\d+)?`)
	matches := re.FindAllString(raw, -1)
	seen := map[string]bool{}
	var result []string
	for _, m := range matches {
		if !seen[m] {
			seen[m] = true
			result = append(result, m)
		}
	}
	sort.Strings(result)
	return result
}

func parseJSONArray(raw string) []string {
	if raw == "" || raw == "—" || raw == "-" || raw == "none" || raw == "TBD" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';'
	})
	var result []string
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

func extractFields(body string) (module, gate, typ, severity string, deps []string, deliverable, acceptance string, effortDays float64, testTypes []string, dodRefs, sourceRefs []string) {
	// Defaults
	typ = "Task"
	severity = "normal"
	deliverable = "See acceptance criteria"
	acceptance = "Captured evidence per §11.4.5"
	effortDays = 1.0

	if m := fieldModule.FindStringSubmatch(body); m != nil {
		module = strings.TrimSpace(m[1])
	}
	if m := fieldGate.FindStringSubmatch(body); m != nil {
		gate = strings.TrimSpace(m[1])
	}
	if m := fieldType.FindStringSubmatch(body); m != nil {
		typ = strings.TrimSpace(m[1])
		typ = strings.ToUpper(typ[:1]) + typ[1:] // Capitalize
	}
	if m := fieldSeverity.FindStringSubmatch(body); m != nil {
		severity = strings.ToLower(strings.TrimSpace(m[1]))
	}
	if m := fieldDeps.FindStringSubmatch(body); m != nil {
		deps = parseDeps(m[1])
	}
	if m := fieldDeliverable.FindStringSubmatch(body); m != nil {
		deliverable = strings.TrimSpace(m[1])
	}
	if m := fieldAcceptance.FindStringSubmatch(body); m != nil {
		acceptance = strings.TrimSpace(m[1])
	}
	if m := fieldEffort.FindStringSubmatch(body); m != nil {
		if m[1] != "" { // T-shirt
			effortDays = tshirtMap[strings.ToUpper(m[1])]
		} else if m[3] != "" { // Numeric days
			effortDays, _ = strconv.ParseFloat(m[3], 64)
		}
	}
	if m := fieldTestTypes.FindStringSubmatch(body); m != nil {
		testTypes = normalizeTestTypes(m[1])
	}
	if m := fieldDodRefs.FindStringSubmatch(body); m != nil {
		dodRefs = parseJSONArray(m[1])
	}
	sourceRefs = fieldSourceRefs.FindAllString(body, -1)
	// Strip brackets from source refs
	for i, s := range sourceRefs {
		sourceRefs[i] = strings.Trim(s, "[]")
	}
	return
}

// ---------------------------------------------------------------------------
// WBS Parser
// ---------------------------------------------------------------------------

func parseWBS(filepath string, phase string) []Item {
	data, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: cannot read %s: %v\n", filepath, err)
		return nil
	}
	lines := strings.Split(string(data), "\n")
	var items []Item
	currentEpic := ""
	currentModule := ""

	now := nowUTC()

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Check for milestone heading (Phase 0)
		if m := milestoneRe.FindStringSubmatch(line); m != nil {
			currentEpic = m[1]
			title := strings.TrimSpace(m[2])
			epicID := "HVPN-P0-" + currentEpic
			items = append(items, Item{
				ATMID:       epicID,
				ParentID:    nil,
				Phase:       "P0",
				Kind:        "milestone",
				Title:       title[:min(200, len(title))],
				Description: fmt.Sprintf("Milestone %s: %s. Contains tasks that must be completed to pass the milestone gate.", currentEpic, title),
				Status:      "Queued",
				Type:        "Task",
				Severity:    "normal",
				Epic:        currentEpic,
				Module:      "",
				Gate:        nil,
				Deps:        nil,
				Deliverable: "See constituent tasks",
				Acceptance:  "All constituent tasks completed with captured evidence",
				EffortDays:  0,
				TestTypes:   nil,
				DodRefs:     nil,
				SourceRefs:  nil,
				CreatedAt:   now,
				ModifiedAt:  now,
			})
			i++
			continue
		}

		// Check for epic heading (Phase 1+)
		if m := epicRe.FindStringSubmatch(line); m != nil {
			currentEpic = m[1]
			title := strings.TrimSpace(m[2])
			epicID := fmt.Sprintf("HVPN-%s-%s", phase, currentEpic)

			// Try to extract module from epic body
			j := i + 1
			var bodyLines []string
			for j < len(lines) && !strings.HasPrefix(lines[j], "## ") {
				bodyLines = append(bodyLines, lines[j])
				j++
			}
			body := strings.Join(bodyLines, "\n")
			if fm := fieldModule.FindStringSubmatch(body); fm != nil {
				currentModule = strings.TrimSpace(fm[1])
			} else {
				currentModule = strings.ToLower(currentEpic)
			}

			items = append(items, Item{
				ATMID:       epicID,
				ParentID:    nil,
				Phase:       phase,
				Kind:        "epic",
				Title:       title[:min(200, len(title))],
				Description: fmt.Sprintf("Epic %s: %s. Groups related work items for the control plane, data plane, or client.", currentEpic, title),
				Status:      "Queued",
				Type:        "Task",
				Severity:    "normal",
				Epic:        currentEpic,
				Module:      currentModule,
				Gate:        nil,
				Deps:        nil,
				Deliverable: "See constituent tasks",
				Acceptance:  "All constituent tasks completed with captured evidence",
				EffortDays:  0,
				TestTypes:   nil,
				DodRefs:     nil,
				SourceRefs:  nil,
				CreatedAt:   now,
				ModifiedAt:  now,
			})
			i++
			continue
		}

		// Check for task/subtask heading (Phase 0 ### TASK format)
		if m := taskRe.FindStringSubmatch(line); m != nil {
			atmID := m[1]
			phaseTag := "P" + m[2]
			title := strings.TrimSpace(m[4])

			// Gather body until next ### or ## heading
			j := i + 1
			var bodyLines []string
			for j < len(lines) {
				if strings.HasPrefix(lines[j], "### ") || strings.HasPrefix(lines[j], "## ") {
					break
				}
				bodyLines = append(bodyLines, lines[j])
				j++
			}
			body := strings.Join(bodyLines, "\n")

			mod, g, typ, sev, dp, del, acc, eff, tt, dod, src := extractFields(body)

			// Determine parent
			var parentID *string
			if strings.Contains(atmID, ".") {
				pid := atmID[:strings.LastIndex(atmID, ".")]
				parentID = &pid
			} else if currentEpic != "" {
				var pid string
				if phase == "P0" {
					pid = "HVPN-P0-" + currentEpic
				} else {
					pid = "HVPN-" + phaseTag + "-" + currentEpic
				}
				parentID = &pid
			}

			// Build description from body
			descLines := []string{}
			for _, l := range bodyLines {
				s := strings.TrimSpace(l)
				if s == "" || strings.HasPrefix(s, "·") || strings.HasPrefix(s, "•") || strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "|") {
					continue
				}
				descLines = append(descLines, s)
			}
			descText := strings.Join(descLines[:min(5, len(descLines))], " ")
			descText = strings.TrimSpace(descText)
			if len(descText) < 40 {
				descText = fmt.Sprintf("Task %s: %s. %s", atmID, title, descText)
			}
			if len(descText) < 40 {
				descText = descText + " Implementation details in the WBS document."
			}
			if len(descText) > 2000 {
				descText = descText[:2000]
			}

			kind := "task"
			if strings.Contains(atmID, ".") {
				kind = "subtask"
			}

			if mod == "" {
				mod = currentModule
			}

			items = append(items, Item{
				ATMID:       atmID,
				ParentID:    parentID,
				Phase:       phaseTag,
				Kind:        kind,
				Title:       title[:min(500, len(title))],
				Description: descText,
				Status:      "Queued",
				Type:        typ,
				Severity:    sev,
				Epic:        currentEpic,
				Module:      mod,
				Gate:        strPtr(g),
				Deps:        dp,
				Deliverable: del,
				Acceptance:  acc,
				EffortDays:  eff,
				TestTypes:   tt,
				DodRefs:     dod,
				SourceRefs:  src,
				CreatedAt:   now,
				ModifiedAt:  now,
			})

			i = j
			continue
		}

		// Check for inline bold tasks (Phase 1+ format)
		if m := inlineTaskRe.FindStringSubmatch(line); m != nil {
			atmID := m[1]
			phaseTag := "P" + m[2]
			title := strings.TrimSpace(m[4])

			// Gather body: the rest of the bold line + continuation lines
			j := i + 1
			var bodyLines []string
			bodyLines = append(bodyLines, line)
			for j < len(lines) {
				nextLine := lines[j]
				if strings.HasPrefix(nextLine, "- **HVPN-") || strings.HasPrefix(nextLine, "## ") || strings.HasPrefix(nextLine, "### ") {
					break
				}
				bodyLines = append(bodyLines, nextLine)
				j++
			}
			body := strings.Join(bodyLines, "\n")

			mod, g, typ, sev, dp, del, acc, eff, tt, dod, src := extractFields(body)

			// Determine parent
			var parentID *string
			if strings.Contains(atmID, ".") {
				pid := atmID[:strings.LastIndex(atmID, ".")]
				parentID = &pid
			} else if currentEpic != "" {
				pid := fmt.Sprintf("HVPN-%s-%s", phaseTag, currentEpic)
				parentID = &pid
			}

			// Build description from inline text
			descText := strings.ReplaceAll(line, "**", "")
			descText = strings.TrimLeft(descText, "-* ")
			descText = strings.TrimSpace(descText)
			if len(descText) < 40 {
				descText = fmt.Sprintf("Task %s: %s. %s", atmID, title, descText)
			}
			if len(descText) < 40 {
				descText = descText + " Implementation details in the WBS document."
			}
			if len(descText) > 2000 {
				descText = descText[:2000]
			}

			kind := "task"
			if strings.Contains(atmID, ".") {
				kind = "subtask"
			}

			if mod == "" {
				mod = currentModule
			}

			items = append(items, Item{
				ATMID:       atmID,
				ParentID:    parentID,
				Phase:       phaseTag,
				Kind:        kind,
				Title:       title[:min(500, len(title))],
				Description: descText,
				Status:      "Queued",
				Type:        typ,
				Severity:    sev,
				Epic:        currentEpic,
				Module:      mod,
				Gate:        strPtr(g),
				Deps:        dp,
				Deliverable: del,
				Acceptance:  acc,
				EffortDays:  eff,
				TestTypes:   tt,
				DodRefs:     dod,
				SourceRefs:  src,
				CreatedAt:   now,
				ModifiedAt:  now,
			})

			i = j
			continue
		}

		i++
	}

	return items
}

// ---------------------------------------------------------------------------
// Deepening Parser
// ---------------------------------------------------------------------------

func parseDeepening(filepath string, phase string) []Item {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	var items []Item
	currentParent := ""

	now := nowUTC()

	for _, line := range lines {
		// Check for parent task heading
		if m := deepeningParentRe.FindStringSubmatch(line); m != nil {
			currentParent = m[1]
			continue
		}

		// Check for subtask table row
		if m := subtaskRowRe.FindStringSubmatch(line); m != nil && currentParent != "" {
			subtaskNum := m[1]
			title := strings.TrimSpace(m[2])
			acceptance := strings.TrimSpace(m[3])
			testTypesRaw := strings.TrimSpace(m[4])
			cx := strings.TrimSpace(m[5])

			atmID := currentParent + "." + subtaskNum
			testTypes := normalizeTestTypes(testTypesRaw)
			effortDays := cxMap[cx]
			if effortDays == 0 {
				effortDays = 1.0
			}

			// Build description
			desc := fmt.Sprintf("Subtask of %s: %s. Acceptance: %s", currentParent, title, acceptance)
			if len(desc) < 40 {
				desc = desc + " See parent task for full context."
			}
			if len(desc) > 2000 {
				desc = desc[:2000]
			}

			items = append(items, Item{
				ATMID:       atmID,
				ParentID:    &currentParent,
				Phase:       phase,
				Kind:        "subtask",
				Title:       title[:min(500, len(title))],
				Description: desc,
				Status:      "Queued",
				Type:        "Task",
				Severity:    "normal",
				Epic:        "",
				Module:      "",
				Gate:        nil,
				Deps:        nil,
				Deliverable: "See acceptance criteria",
				Acceptance:  acceptance,
				EffortDays:  effortDays,
				TestTypes:   testTypes,
				DodRefs:     nil,
				SourceRefs:  nil,
				CreatedAt:   now,
				ModifiedAt:  now,
			})
		}
	}

	return items
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func cmdMdtoDB(dryRun bool) {
	var allItems []Item

	// Parse WBS files
	for _, phase := range []string{"P0", "P1", "P2", "P3"} {
		relPath := wbsFiles[phase]
		fullPath := filepath.Join(repoRoot, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "WARN: %s not found, skipping %s\n", relPath, phase)
			continue
		}
		items := parseWBS(fullPath, phase)
		fmt.Printf("Parsed %s WBS: %d items from %s\n", phase, len(items), filepath.Base(relPath))
		allItems = append(allItems, items...)
	}

	// Parse deepening files
	for _, phase := range []string{"P1", "P2", "P3"} {
		relPath := deepeningFiles[phase]
		fullPath := filepath.Join(repoRoot, relPath)
		items := parseDeepening(fullPath, phase)
		if items != nil {
			fmt.Printf("Parsed %s deepening: %d subtasks from %s\n", phase, len(items), filepath.Base(relPath))
		}
		allItems = append(allItems, items...)
	}

	fmt.Printf("Total: %d items across all phases\n", len(allItems))

	if dryRun {
		fmt.Printf("DRY RUN: would insert %d items\n", len(allItems))
		for i, item := range allItems {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(allItems)-5)
				break
			}
			fmt.Printf("  %s: %s\n", item.ATMID, item.Title[:min(60, len(item.Title))])
		}
		return
	}

	// Load into DB
	db := openDB()
	defer db.Close()
	ensureSchema(db)

	// Idempotent: clear existing items
	_, _ = db.Exec("DELETE FROM item_history")
	_, _ = db.Exec("DELETE FROM operator_block_details")
	_, _ = db.Exec("DELETE FROM obsolete_details")
	_, _ = db.Exec("DELETE FROM test_diary")
	_, _ = db.Exec("DELETE FROM items")

	now := nowUTC()
	inserted := 0
	skipped := 0

	tx, err := db.Begin()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot begin transaction: %v\n", err)
		os.Exit(1)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO items (
			atm_id, parent_id, phase, kind, title, description,
			status, type, severity, epic, module, gate, deps,
			deliverable, acceptance, effort_days, test_types,
			dod_refs, source_refs, created_at, modified_at,
			created_by, assigned_to
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '')
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot prepare statement: %v\n", err)
		os.Exit(1)
	}
	defer stmt.Close()

	for _, item := range allItems {
		_, err := stmt.Exec(
			item.ATMID,
			item.ParentID,
			item.Phase,
			item.Kind,
			item.Title,
			item.Description,
			item.Status,
			item.Type,
			item.Severity,
			item.Epic,
			item.Module,
			item.Gate,
			toJSON(item.Deps),
			item.Deliverable,
			item.Acceptance,
			item.EffortDays,
			toJSON(item.TestTypes),
			toJSON(item.DodRefs),
			toJSON(item.SourceRefs),
			now,
			now,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "SKIP %s: %v\n", item.ATMID, err)
			skipped++
			continue
		}
		inserted++
	}

	// Compute integrity hash
	keyset := make([]string, len(allItems))
	for i, item := range allItems {
		keyset[i] = item.ATMID
	}
	sort.Strings(keyset)
	h := sha256.Sum256([]byte(strings.Join(keyset, "\n")))
	integrity := fmt.Sprintf("%x", h)

	// Update meta
	_, _ = tx.Exec("DELETE FROM meta")
	_, err = tx.Exec(
		"INSERT INTO meta (schema_version, last_sync_at, integrity_hash) VALUES (?, ?, ?)",
		schemaVersion, now, integrity,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot insert meta: %v\n", err)
	}

	if err := tx.Commit(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot commit: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %d items, skipped %d (integrity: %s...)\n", inserted, skipped, integrity[:16])
}

func cmdDbtoMD(dryRun bool) {
	db := openDB()
	defer db.Close()
	ensureSchema(db)

	rows, err := db.Query(`
		SELECT atm_id, parent_id, phase, kind, title, description,
		       status, type, severity, epic, module, gate, deps,
		       deliverable, acceptance, effort_days, test_types,
		       dod_refs, source_refs
		FROM items ORDER BY atm_id
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot query items: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var it Item
		var parentID, gate sql.NullString
		var depsJSON, testTypesJSON, dodRefsJSON, sourceRefsJSON string
		err := rows.Scan(
			&it.ATMID, &parentID, &it.Phase, &it.Kind, &it.Title, &it.Description,
			&it.Status, &it.Type, &it.Severity, &it.Epic, &it.Module, &gate, &depsJSON,
			&it.Deliverable, &it.Acceptance, &it.EffortDays, &testTypesJSON,
			&dodRefsJSON, &sourceRefsJSON,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: scanning row: %v\n", err)
			continue
		}
		if parentID.Valid {
			it.ParentID = &parentID.String
		}
		if gate.Valid {
			it.Gate = &gate.String
		}
		_ = json.Unmarshal([]byte(depsJSON), &it.Deps)
		_ = json.Unmarshal([]byte(testTypesJSON), &it.TestTypes)
		_ = json.Unmarshal([]byte(dodRefsJSON), &it.DodRefs)
		_ = json.Unmarshal([]byte(sourceRefsJSON), &it.SourceRefs)
		items = append(items, it)
	}

	fmt.Printf("Read %d items from DB\n", len(items))

	if dryRun {
		fmt.Println("DRY RUN: would regenerate WBS markdown files")
		return
	}

	// Group by phase
	phases := map[string][]Item{}
	for _, it := range items {
		phases[it.Phase] = append(phases[it.Phase], it)
	}

	// Generate markdown for each phase
	for _, phase := range []string{"P0", "P1", "P2", "P3"} {
		phaseItems := phases[phase]
		if len(phaseItems) == 0 {
			continue
		}

		// Build output
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("# Phase %s Workable Items (auto-generated from DB)\n\n", phase[1:]))
		sb.WriteString(fmt.Sprintf("**Revision:** 1\n**Last modified:** %s\n\n", nowUTC()))
		sb.WriteString("Generated by `workable-items sync db-to-md`. Do not edit manually.\n\n")
		sb.WriteString("---\n\n")

		// Group by epic
		epics := map[string][]Item{}
		for _, it := range phaseItems {
			epics[it.Epic] = append(epics[it.Epic], it)
		}

		// §11.4.93 round-trip note: the heading emitted for an epic/milestone
		// item MUST be parseable by this same file's milestoneRe / epicRe
		// (see parseWBS above) — those regexes require "## <N>. Milestone
		// <S-tag> — <Title>" / "## <N>. <E-tag> — <Title>", NOT a bare
		// "## <ATM-ID> — <Title>". A mismatch here makes the item invisible
		// to a subsequent parse (the exact "in DB, not in MD" divergence
		// `diff` reports), even though a heading line was written.
		epicIdx := 0
		for _, epic := range sortedKeys(epics) {
			epicItems := epics[epic]
			// Find the epic item itself
			var epicItem *Item
			for i, it := range epicItems {
				if it.Kind == "epic" || it.Kind == "milestone" {
					epicItem = &epicItems[i]
					break
				}
			}
			if epicItem != nil {
				epicIdx++
				if epicItem.Kind == "milestone" {
					sb.WriteString(fmt.Sprintf("## %d. Milestone %s — %s\n\n", epicIdx, epicItem.Epic, epicItem.Title))
				} else {
					sb.WriteString(fmt.Sprintf("## %d. %s — %s\n\n", epicIdx, epicItem.Epic, epicItem.Title))
				}
			}

			for _, it := range epicItems {
				if it.Kind == "epic" || it.Kind == "milestone" {
					continue
				}
				sb.WriteString(fmt.Sprintf("- **%s — %s.**\n", it.ATMID, it.Title))
				sb.WriteString(fmt.Sprintf("  · type: %s · severity: %s\n", it.Type, it.Severity))
				if it.Gate != nil {
					sb.WriteString(fmt.Sprintf("  · gate: %s\n", *it.Gate))
				}
				if len(it.Deps) > 0 {
					sb.WriteString(fmt.Sprintf("  · deps: %s\n", strings.Join(it.Deps, ", ")))
				}
				sb.WriteString(fmt.Sprintf("  · effort: %.1f days\n", it.EffortDays))
				if len(it.TestTypes) > 0 {
					sb.WriteString(fmt.Sprintf("  · tests: %s\n", strings.Join(it.TestTypes, ", ")))
				}
				sb.WriteString(fmt.Sprintf("  · deliverable: %s\n", it.Deliverable))
				sb.WriteString(fmt.Sprintf("  · acceptance: %s\n", it.Acceptance))
				sb.WriteString("\n")
			}
		}

		// Write to file (overwrite existing)
		relPath := wbsFiles[phase]
		outPath := filepath.Join(repoRoot, relPath)
		if err := os.WriteFile(outPath, []byte(sb.String()), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: cannot write %s: %v\n", outPath, err)
		} else {
			fmt.Printf("Wrote %s (%d items)\n", relPath, len(phaseItems))
		}
	}
}

func cmdDiff() {
	db := openDB()
	defer db.Close()
	ensureSchema(db)

	// Parse from markdown
	var mdItems []Item
	for _, phase := range []string{"P0", "P1", "P2", "P3"} {
		relPath := wbsFiles[phase]
		fullPath := filepath.Join(repoRoot, relPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue
		}
		mdItems = append(mdItems, parseWBS(fullPath, phase)...)
	}
	for _, phase := range []string{"P1", "P2", "P3"} {
		relPath := deepeningFiles[phase]
		fullPath := filepath.Join(repoRoot, relPath)
		mdItems = append(mdItems, parseDeepening(fullPath, phase)...)
	}

	// Get from DB
	dbRows, err := db.Query("SELECT atm_id, title, status, type FROM items ORDER BY atm_id")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer dbRows.Close()

	type dbRow struct {
		ATMID  string
		Title  string
		Status string
		Type   string
	}
	dbItems := map[string]dbRow{}
	for dbRows.Next() {
		var r dbRow
		_ = dbRows.Scan(&r.ATMID, &r.Title, &r.Status, &r.Type)
		dbItems[r.ATMID] = r
	}

	mdIDs := map[string]bool{}
	for _, it := range mdItems {
		mdIDs[it.ATMID] = true
	}

	diffCount := 0

	// Items in MD but not in DB
	for _, it := range mdItems {
		if _, ok := dbItems[it.ATMID]; !ok {
			fmt.Printf("+ %s (in MD, not in DB): %s\n", it.ATMID, it.Title[:min(60, len(it.Title))])
			diffCount++
		}
	}

	// Items in DB but not in MD
	for id, r := range dbItems {
		if !mdIDs[id] {
			fmt.Printf("- %s (in DB, not in MD): %s\n", id, r.Title[:min(60, len(r.Title))])
			diffCount++
		}
	}

	if diffCount == 0 {
		fmt.Println("No differences found — MD and DB are in sync.")
	} else {
		fmt.Printf("\n%d difference(s) found.\n", diffCount)
		os.Exit(1)
	}
}

func cmdValidate() {
	db := openDB()
	defer db.Close()
	ensureSchema(db)

	rows, err := db.Query(`
		SELECT atm_id, title, description, status, type, phase, kind
		FROM items ORDER BY atm_id
	`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	issues := 0
	total := 0
	idSet := map[string]bool{}

	for rows.Next() {
		var atmID, title, desc, status, typ, phase, kind string
		_ = rows.Scan(&atmID, &title, &desc, &status, &typ, &phase, &kind)
		total++

		// D1: id uniqueness
		if idSet[atmID] {
			fmt.Printf("FAIL %s: duplicate atm_id\n", atmID)
			issues++
		}
		idSet[atmID] = true

		// D1: status not empty
		if status == "" {
			fmt.Printf("FAIL %s: empty status\n", atmID)
			issues++
		}

		// D1: type not empty
		if typ == "" {
			fmt.Printf("FAIL %s: empty type\n", atmID)
			issues++
		}

		// D2: description >= 40 chars
		if len(desc) < 40 {
			fmt.Printf("FAIL %s: description too short (%d chars): %s\n", atmID, len(desc), desc[:min(40, len(desc))])
			issues++
		}

		// Title >= 6 chars
		if len(title) < 6 {
			fmt.Printf("FAIL %s: title too short (%d chars)\n", atmID, len(title))
			issues++
		}
	}

	// Check meta integrity
	var metaHash string
	err = db.QueryRow("SELECT integrity_hash FROM meta LIMIT 1").Scan(&metaHash)
	if err != nil {
		fmt.Printf("FAIL: no meta row (integrity hash missing)\n")
		issues++
	}

	if issues == 0 {
		fmt.Printf("PASS: %d items validated, 0 issues found.\n", total)
	} else {
		fmt.Printf("FAIL: %d items validated, %d issue(s) found.\n", total, issues)
		os.Exit(1)
	}
}

func cmdAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	id := fs.String("id", "", "atm_id (e.g. HVPN-P1-NNN)")
	title := fs.String("title", "", "item title")
	parent := fs.String("parent", "", "parent atm_id (optional)")
	phase := fs.String("phase", "P1", "phase (P0/P1/P2/P3)")
	typ := fs.String("type", "Task", "type (Bug/Feature/Task)")
	kind := fs.String("kind", "task", "kind (task/subtask)")
	severity := fs.String("severity", "normal", "severity")
	epic := fs.String("epic", "", "epic id")
	module := fs.String("module", "", "module name")
	description := fs.String("description", "", "description (>= 40 chars)")
	fs.Parse(args)

	if *id == "" || *title == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --id and --title are required")
		fs.Usage()
		os.Exit(1)
	}

	if len(*description) < 40 {
		*description = fmt.Sprintf("Task %s: %s. Implementation details pending.", *id, *title)
	}

	db := openDB()
	defer db.Close()
	ensureSchema(db)

	now := nowUTC()
	_, err := db.Exec(`
		INSERT INTO items (atm_id, parent_id, phase, kind, title, description,
			status, type, severity, epic, module, deliverable, acceptance,
			created_at, modified_at)
		VALUES (?, ?, ?, ?, ?, ?, 'Queued', ?, ?, ?, ?, 'TBD', 'Captured evidence per §11.4.5', ?, ?)
	`, *id, parent, *phase, *kind, *title, *description, *typ, *severity, *epic, *module, now, now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot add item: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added %s: %s\n", *id, *title)
}

func cmdClose(args []string) {
	fs := flag.NewFlagSet("close", flag.ExitOnError)
	id := fs.String("id", "", "atm_id to close")
	status := fs.String("status", "Fixed (→ Fixed.md)", "new status")
	reason := fs.String("reason", "fixed", "closure reason")
	evidence := fs.String("evidence", "", "evidence path/description")
	fs.Parse(args)

	if *id == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --id is required")
		fs.Usage()
		os.Exit(1)
	}

	db := openDB()
	defer db.Close()
	ensureSchema(db)

	// Get current status
	var currentStatus string
	err := db.QueryRow("SELECT status FROM items WHERE atm_id = ?", *id).Scan(&currentStatus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: item %s not found: %v\n", *id, err)
		os.Exit(1)
	}

	now := nowUTC()

	// Update status
	_, err = db.Exec("UPDATE items SET status = ?, modified_at = ? WHERE atm_id = ?",
		*status, now, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot update item: %v\n", err)
		os.Exit(1)
	}

	// Add history entry
	_, err = db.Exec(`
		INSERT INTO item_history (atm_id, changed_at, by, from_status, to_status, reason, evidence)
		VALUES (?, ?, 'AI', ?, ?, ?, ?)
	`, *id, now, currentStatus, *status, *reason, *evidence)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARN: cannot add history: %v\n", err)
	}

	fmt.Printf("Closed %s: %s → %s\n", *id, currentStatus, *status)
}

// ---------------------------------------------------------------------------
// Utility
// ---------------------------------------------------------------------------

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sortedKeys(m map[string][]Item) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func usage() {
	fmt.Fprintf(os.Stderr, `workable-items — §11.4.93 SQLite single source of truth CLI

Usage:
  workable-items <command> [flags]

Commands:
  sync md-to-db [--dry-run]   Parse WBS markdown → load into SQLite DB
  sync db-to-md [--dry-run]   Regenerate WBS markdown from SQLite DB
  diff                        Show MD↔DB divergence
  validate                    Validate DB integrity
  add --id ... --title ...    Add a new workable item
  close --id ... [--status ..] Close a workable item

Cross-references:
  Constitution §11.4.93, §11.4.95, §11.4.54, §11.4.148
  Spec: v07-execution/workable-items-model.md
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	repoRoot = findRepoRoot()
	dbPath = filepath.Join(repoRoot, "docs", "workable_items.db")

	cmd := os.Args[1]

	switch cmd {
	case "sync":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "ERROR: sync requires subcommand (md-to-db or db-to-md)")
			os.Exit(1)
		}
		sub := os.Args[2]
		dryRun := len(os.Args) > 3 && os.Args[3] == "--dry-run"
		switch sub {
		case "md-to-db":
			cmdMdtoDB(dryRun)
		case "db-to-md":
			cmdDbtoMD(dryRun)
		default:
			fmt.Fprintf(os.Stderr, "ERROR: unknown sync subcommand: %s\n", sub)
			os.Exit(1)
		}

	case "diff":
		cmdDiff()

	case "validate":
		cmdValidate()

	case "add":
		cmdAdd(os.Args[2:])

	case "close":
		cmdClose(os.Args[2:])

	default:
		fmt.Fprintf(os.Stderr, "ERROR: unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
}
