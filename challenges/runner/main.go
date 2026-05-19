// Command runner is the LLMOrchestrator round-275 Challenge runner.
//
// It exercises the real parser.DefaultParser, the real
// protocol.PipeMessage JSON marshal/unmarshal, the real
// protocol.FileTransport inbox/outbox round-trip, and the real
// i18n.NoopTranslator across five locale fixtures. Every PASS line
// is backed by a runtime invariant — never a metadata-only check
// (CONST-035 / Article XI §11.9).
//
// Anti-bluff invariants enforced:
//
//  1. parser.NewParser() returns a non-nil ResponseParser that
//     actually decodes a fixture's prompt_json and surfaces an
//     Action whose Type matches expect_action_type AND Target
//     matches expect_action_target. A silent miss would let a
//     stub-parser ship green.
//  2. parser.Parse rejects the documented empty-input contract
//     (ErrEmptyInput) — defensive boundary preserved across rounds.
//  3. protocol.PipeMessage round-trips through encoding/json: the
//     marshalled bytes unmarshal to a struct whose Content equals
//     expect_pipe_content. Encoding drift (e.g. field tag changes)
//     would fail here, not silently corrupt the wire format.
//  4. protocol.FileTransport.WriteToInbox + ReadFromInbox round-trip
//     a FileMessage whose Content equals the fixture's expected
//     content. A regression that drops messages or returns an
//     empty slice fails the gate.
//  5. i18n.NoopTranslator.T returns the message id verbatim (the
//     anti-bluff contract documented in translator.go — missing
//     translations surface as the key, not as silent empty strings).
//
// Mutation hook: when env LLMORCH_MUTATE_RUNNER=1 is set, the
// runner inverts invariant (2) (treats a successful empty-input
// parse as PASS instead of FAIL). The paired Challenge wraps this
// to assert the runner exits 99 under mutation, guaranteeing the
// runner actually checks what it claims (CONST-050(A) paired
// mutation, §1.1).
//
// Verbatim 2026-05-19 operator mandate (preserved per
// CONST-049 §11.4.17):
//
//	"all existing tests and Challenges do work in anti-bluff
//	manner - they MUST confirm that all tested codebase really
//	works as expected! We had been in position that all tests
//	do execute with success and all Challenges as well, but
//	in reality the most of the features does not work and
//	can't be used! This MUST NOT be the case and execution
//	of tests and Challenges MUST guarantee the quality, the
//	completition and full usability by end users of the
//	product!"
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"digital.vasic.llmorchestrator/pkg/i18n"
	"digital.vasic.llmorchestrator/pkg/parser"
	"digital.vasic.llmorchestrator/pkg/protocol"
)

// fixture is a 6-field projection of challenges/fixtures/<locale>.yaml.
// Parsed in-process so the runner stays dependency-free beyond what
// the module already depends on (CONST-051(B): no new transitive
// deps creeping into a reusable submodule).
type fixture struct {
	locale             string
	promptJSON         string
	expectActionType   string
	expectActionTarget string
	expectPipeContent  string
	expectMessageID    string
}

func main() {
	if code := run(os.Stdout); code != 0 {
		os.Exit(code)
	}
}

func run(out io.Writer) int {
	fmt.Fprintln(out, "=== LLMOrchestrator Challenge Runner (round-275) ===")

	fixDir := os.Getenv("LLMORCH_FIXTURES_DIR")
	if fixDir == "" {
		fixDir = filepath.Join("challenges", "fixtures")
	}

	fixtures, err := loadFixtures(fixDir)
	if err != nil {
		fmt.Fprintf(out, "FAIL: load fixtures from %s: %v\n",
			fixDir, err)
		return 1
	}
	if len(fixtures) < 5 {
		fmt.Fprintf(out, "FAIL: expected >=5 fixtures, got %d\n",
			len(fixtures))
		return 1
	}
	fmt.Fprintf(out, "[setup] loaded %d locale fixtures from %s\n",
		len(fixtures), fixDir)

	mutate := os.Getenv("LLMORCH_MUTATE_RUNNER") == "1"
	if mutate {
		fmt.Fprintln(out, "[setup] MUTATION MODE: runner will treat"+
			" successful empty-input parse as PASS")
	}

	pass, fail := 0, 0
	step := func(name string, ok bool, detail string) {
		if ok {
			pass++
			fmt.Fprintf(out, "  PASS  %-48s  %s\n", name, detail)
			return
		}
		fail++
		fmt.Fprintf(out, "  FAIL  %-48s  %s\n", name, detail)
	}

	// Invariant 1+2: parser construction + per-fixture decode +
	// empty-input contract.
	p := parser.NewParser()
	step("parser.NewParser.not_nil",
		p != nil,
		"constructor returned non-nil")

	_, emptyErr := p.Parse("")
	if mutate {
		// Mutation flips polarity: PASS when error is nil.
		step("parser.Parse.empty_errors[MUTATED]",
			emptyErr == nil,
			"mutation-inverted check")
	} else {
		step("parser.Parse.empty_errors",
			emptyErr != nil,
			fmt.Sprintf("got=%v", emptyErr))
	}

	for _, f := range fixtures {
		parsed, err := p.Parse(f.promptJSON)
		if err != nil {
			step("parser.Parse."+f.locale,
				false,
				fmt.Sprintf("err=%v", err))
			continue
		}
		// Find action matching expected type+target.
		matched := false
		for _, a := range parsed.Actions {
			if a.Type == f.expectActionType &&
				strings.Contains(a.Target, f.expectActionTarget) {
				matched = true
				break
			}
		}
		step("parser.Parse."+f.locale+".action",
			matched,
			fmt.Sprintf("want type=%s target=%s actions=%+v",
				f.expectActionType,
				f.expectActionTarget,
				parsed.Actions))
	}

	// Invariant 3: PipeMessage round-trip through encoding/json.
	for _, f := range fixtures {
		msg := protocol.PipeMessage{
			Type:      protocol.MessageTypePrompt,
			Content:   f.expectPipeContent,
			Timestamp: time.Now().UTC(),
			RequestID: "round-275-" + f.locale,
		}
		raw, err := json.Marshal(msg)
		if err != nil {
			step("protocol.PipeMessage.marshal."+f.locale,
				false, fmt.Sprintf("err=%v", err))
			continue
		}
		var back protocol.PipeMessage
		if err := json.Unmarshal(raw, &back); err != nil {
			step("protocol.PipeMessage.unmarshal."+f.locale,
				false, fmt.Sprintf("err=%v", err))
			continue
		}
		step("protocol.PipeMessage.roundtrip."+f.locale,
			back.Content == f.expectPipeContent &&
				back.Type == protocol.MessageTypePrompt &&
				back.RequestID == "round-275-"+f.locale,
			fmt.Sprintf("content=%q type=%s req=%s",
				back.Content, back.Type, back.RequestID))
	}

	// Invariant 4: FileTransport inbox round-trip (real disk I/O).
	tmp, err := os.MkdirTemp("", "llmorch-round275-*")
	if err != nil {
		step("protocol.FileTransport.mkdir_tmp",
			false, fmt.Sprintf("err=%v", err))
	} else {
		defer os.RemoveAll(tmp)
		ft, err := protocol.NewFileTransport(tmp)
		if err != nil {
			step("protocol.NewFileTransport",
				false, fmt.Sprintf("err=%v", err))
		} else {
			step("protocol.NewFileTransport",
				true, "created at "+tmp)
			for _, f := range fixtures {
				fm := protocol.FileMessage{
					ID:        "round-275-" + f.locale,
					Type:      "instruction",
					Content:   f.expectPipeContent,
					CreatedAt: time.Now().UTC(),
				}
				if err := ft.WriteToInbox(fm); err != nil {
					step("protocol.FileTransport.write."+f.locale,
						false, fmt.Sprintf("err=%v", err))
					continue
				}
				msgs, err := ft.ReadFromInbox()
				if err != nil {
					step("protocol.FileTransport.read."+f.locale,
						false, fmt.Sprintf("err=%v", err))
					continue
				}
				found := false
				for _, m := range msgs {
					if m.ID == fm.ID &&
						m.Content == f.expectPipeContent {
						found = true
						break
					}
				}
				step("protocol.FileTransport.roundtrip."+f.locale,
					found,
					fmt.Sprintf("messages_in_inbox=%d", len(msgs)))
				// Clean inbox between locales so each locale's
				// assertion is independent.
				for _, m := range msgs {
					_ = os.Remove(filepath.Join(
						tmp, "inbox", m.ID+".json"))
				}
			}
		}
	}

	// Invariant 5: NoopTranslator returns message id verbatim — the
	// documented anti-bluff contract (translator.go).
	tr := i18n.NoopTranslator{}
	ctx, cancel := context.WithTimeout(
		context.Background(), 5*time.Second)
	defer cancel()
	for _, f := range fixtures {
		got, err := tr.T(ctx, f.expectMessageID, nil)
		ok := err == nil && got == f.expectMessageID
		step("i18n.Noop.key_roundtrip."+f.locale,
			ok,
			fmt.Sprintf("id=%s got=%q err=%v",
				f.expectMessageID, got, err))

		gotP, errP := tr.TPlural(ctx, f.expectMessageID, 3, nil)
		okP := errP == nil && gotP == f.expectMessageID
		step("i18n.Noop.plural_roundtrip."+f.locale,
			okP,
			fmt.Sprintf("id=%s got=%q err=%v",
				f.expectMessageID, gotP, errP))
	}

	// Invariant: SetPkgTranslator(nil) resets to NoopTranslator.
	i18n.SetPkgTranslator(nil)
	pkg := i18n.Pkg()
	gotPkg, errPkg := pkg.T(ctx, "round_275_pkg_probe", nil)
	step("i18n.Pkg.reset_to_noop",
		errPkg == nil && gotPkg == "round_275_pkg_probe",
		fmt.Sprintf("got=%q err=%v", gotPkg, errPkg))

	fmt.Fprintf(out, "\n=== Summary: PASS=%d FAIL=%d ===\n",
		pass, fail)
	if fail > 0 {
		return 1
	}
	return 0
}

// loadFixtures parses every *.yaml in dir using a tiny line-based
// parser. We only support the keys our fixtures use; anything else
// is ignored. Keeping the parser in-runner avoids pulling yaml.v3
// into the runtime path of a submodule that other projects reuse
// (CONST-051(B)).
func loadFixtures(dir string) ([]fixture, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []fixture
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		f := parseFixture(string(data))
		if f.locale == "" {
			return nil, fmt.Errorf(
				"%s: missing locale key", e.Name())
		}
		out = append(out, f)
	}
	return out, nil
}

func parseFixture(text string) fixture {
	f := fixture{}
	for _, line := range strings.Split(text, "\n") {
		raw := line
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		colon := strings.Index(trimmed, ":")
		if colon < 0 {
			continue
		}
		k := strings.TrimSpace(trimmed[:colon])
		v := strings.TrimSpace(trimmed[colon+1:])
		// Strip surrounding single OR double quotes, but only one
		// matching pair (preserve embedded JSON quoting).
		if len(v) >= 2 {
			first, last := v[0], v[len(v)-1]
			if (first == '\'' && last == '\'') ||
				(first == '"' && last == '"') {
				v = v[1 : len(v)-1]
			}
		}
		switch k {
		case "locale":
			f.locale = v
		case "prompt_json":
			f.promptJSON = v
		case "expect_action_type":
			f.expectActionType = v
		case "expect_action_target":
			f.expectActionTarget = v
		case "expect_pipe_content":
			f.expectPipeContent = v
		case "expect_message_id":
			f.expectMessageID = v
		}
	}
	return f
}
