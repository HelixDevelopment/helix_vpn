// ============================================================================
// TestDbToMdRoundTripEpicMilestoneHeadings — §11.4.93 MD<->DB round-trip gate
// ============================================================================
//
// Root cause under test (confirmed via reproduction against the live project
// DB, see docs/workable_items.db + `workable-items diff` — 43 divergences,
// all epic (34) + milestone (9), all "in DB, not in MD"):
//
//   cmdDbtoMD DOES emit a heading line for every epic/milestone item:
//       "## <ATM-ID> — <Title>\n\n"
//   but the WBS parser's own milestoneRe / epicRe regexes require:
//       "## <N>. Milestone <S-tag> — <Title>"   (milestone, Phase 0)
//       "## <N>. <E-tag> — <Title>"             (epic, Phase 1+)
//   Neither regex matches the generator's actual output, so the very
//   epic/milestone items db-to-md just wrote become invisible to a
//   subsequent parseWBS() re-parse (used by both `diff` and `sync
//   md-to-db`) — this is NOT an omission (the generator does write a line
//   for the item), it is a writer/reader heading-format mismatch.
//
// This test reproduces the gap directly: insert a milestone item (with a
// child task, mirroring real DB shape) into a throwaway DB, regenerate the
// Phase-0 markdown via the real cmdDbtoMD code path, then re-parse that
// regenerated file with the real parseWBS() and assert the milestone
// round-trips back with its kind + ATM ID intact.
// ============================================================================

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDbToMdRoundTripEpicMilestoneHeadings(t *testing.T) {
	origRepoRoot, origDBPath := repoRoot, dbPath
	defer func() { repoRoot, dbPath = origRepoRoot, origDBPath }()

	tmp := t.TempDir()
	repoRoot = tmp
	dbPath = filepath.Join(tmp, "test_workable_items.db")

	if err := os.MkdirAll(filepath.Join(tmp, "docs", "research", "mvp", "final"), 0o755); err != nil {
		t.Fatalf("mkdir fixture tree: %v", err)
	}

	db := openDB()
	ensureSchema(db)

	now := nowUTC()
	_, err := db.Exec(`INSERT INTO items (
		atm_id, parent_id, phase, kind, title, description,
		status, type, severity, epic, module, gate, deps,
		deliverable, acceptance, effort_days, test_types,
		dod_refs, source_refs, created_at, modified_at, created_by, assigned_to
	) VALUES
	('HVPN-P0-S0', NULL, 'P0', 'milestone', 'Cargo workspace + Transport trait',
	 'Milestone S0: Cargo workspace + Transport trait. Contains tasks that must be completed to pass the milestone gate.',
	 'Queued', 'Task', 'normal', 'S0', '', NULL, '[]', 'See constituent tasks',
	 'All constituent tasks completed with captured evidence', 0, '[]', '[]', '[]', ?, ?, '', ''),
	('HVPN-P0-001', 'HVPN-P0-S0', 'P0', 'task', 'Bootstrap the helix-core workspace skeleton',
	 'Task HVPN-P0-001: Bootstrap the helix-core Cargo workspace skeleton. Implementation details in the WBS document.',
	 'Queued', 'Task', 'normal', 'S0', 'helix-core', NULL, '[]', 'See acceptance criteria',
	 'Captured evidence per section 11.4.5', 1.0, '[]', '[]', '[]', ?, ?, '', '')
	`, now, now, now, now)
	if err != nil {
		t.Fatalf("insert fixture rows: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close fixture db: %v", err)
	}

	// Code under test: regenerate WBS markdown straight from the DB.
	cmdDbtoMD(false)

	genPath := filepath.Join(tmp, wbsFiles["P0"])
	data, err := os.ReadFile(genPath)
	if err != nil {
		t.Fatalf("read regenerated markdown: %v", err)
	}

	// Re-parse the regenerated markdown exactly like `diff` / `sync md-to-db` do.
	parsed := parseWBS(genPath, "P0")

	var found *Item
	for i, it := range parsed {
		if it.ATMID == "HVPN-P0-S0" {
			found = &parsed[i]
			break
		}
	}
	if found == nil {
		t.Fatalf(
			"regenerated markdown did not round-trip milestone item HVPN-P0-S0 back through "+
				"parseWBS (this is the exact divergence `workable-items diff` reports for all "+
				"43 epic/milestone items in the live DB) — first heading line emitted by "+
				"cmdDbtoMD was: %q",
			firstHeadingLine(string(data)),
		)
	}
	if found.Kind != "milestone" {
		t.Errorf("HVPN-P0-S0 round-tripped with kind=%q, want %q", found.Kind, "milestone")
	}
}

func firstHeadingLine(md string) string {
	for _, l := range strings.Split(md, "\n") {
		if strings.HasPrefix(l, "## ") {
			return l
		}
	}
	return "(no ## heading found)"
}
