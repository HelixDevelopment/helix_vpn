// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.llmorchestrator/pkg/i18n"
)

// Round-387 §11.4 sentinel + mutation evidence for the standalone
// orchestrator CLI (cmd/orchestrator/main.go). When a real Translator
// is installed via i18n.SetPkgTranslator, every migrated CLI console
// message MUST route its user-facing text through it instead of the
// fmt.Sprintf fallback. With the default NoopTranslator the call site
// MUST fall back so the captured data stays visible to the operator.
//
// CONST-050(A): mocks are permitted in unit tests only.

type captureTranslator struct {
	lastID string
	out    string
}

func (c *captureTranslator) T(
	_ context.Context,
	id string,
	_ map[string]any,
) (string, error) {
	c.lastID = id
	return c.out, nil
}

func (c *captureTranslator) TPlural(
	_ context.Context,
	id string,
	_ int,
	_ map[string]any,
) (string, error) {
	c.lastID = id
	return c.out, nil
}

// TestCliMsg_RoutesThroughI18n proves cliMsg consults the installed
// translator and returns its output verbatim when it differs from the
// bare message ID.
func TestCliMsg_RoutesThroughI18n(t *testing.T) {
	ids := []string{
		"llmorchestrator_cli_version_banner",
		"llmorchestrator_cli_config_load_error",
		"llmorchestrator_cli_config_invalid",
		"llmorchestrator_cli_agent_skipped",
		"llmorchestrator_cli_agent_unknown",
		"llmorchestrator_cli_agent_register_error",
		"llmorchestrator_cli_agent_registered",
		"llmorchestrator_cli_ready_banner",
		"llmorchestrator_cli_shutting_down",
		"llmorchestrator_cli_shutdown_error",
		"llmorchestrator_cli_shutdown_complete",
	}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			cap := &captureTranslator{out: "TRANSLATED_" + id}
			i18n.SetPkgTranslator(cap)
			t.Cleanup(func() { i18n.SetPkgTranslator(nil) })

			got := cliMsg(id, map[string]any{"x": 1}, "ORIGINAL_FALLBACK")
			if got != "TRANSLATED_"+id {
				t.Fatalf("cliMsg(%q) = %q, want %q (i18n seam not wired)", id, got, "TRANSLATED_"+id)
			}
			if cap.lastID != id {
				t.Fatalf("translator received id %q, want %q", cap.lastID, id)
			}
		})
	}
}

// TestCliMsg_NoopTranslator_FallsBackToFallback is the paired mutation:
// with the default NoopTranslator (returns the message ID verbatim)
// cliMsg MUST return the fallback literal — never the bare message ID.
func TestCliMsg_NoopTranslator_FallsBackToFallback(t *testing.T) {
	i18n.SetPkgTranslator(nil) // reset to default Noop
	const id = "llmorchestrator_cli_ready_banner"
	got := cliMsg(id, map[string]any{"version": "9.9.9", "count": 7}, "FALLBACK_TEXT_ABC")
	if got == id {
		t.Fatalf("cliMsg leaked bare message ID: %q", got)
	}
	if got != "FALLBACK_TEXT_ABC" {
		t.Fatalf("cliMsg = %q, want fallback %q", got, "FALLBACK_TEXT_ABC")
	}
}

// TestCliMsgf_NoopTranslator_FormatsFallback proves the printf-style
// wrapper builds its fallback from the format string + args when no
// real translator is installed, keeping captured data visible.
func TestCliMsgf_NoopTranslator_FormatsFallback(t *testing.T) {
	i18n.SetPkgTranslator(nil) // reset to default Noop
	got := cliMsgf(
		"llmorchestrator_cli_agent_skipped",
		map[string]any{"agent": "gemini", "error": "boom"},
		"Warning: skipping agent %s: %v\n", "gemini", "boom",
	)
	if !strings.Contains(got, "gemini") || !strings.Contains(got, "boom") {
		t.Fatalf("cliMsgf = %q, want substrings %q and %q", got, "gemini", "boom")
	}
	if strings.Contains(got, "llmorchestrator_cli_agent_skipped") {
		t.Fatalf("cliMsgf leaked bare message ID: %q", got)
	}
}

// TestCliMsgf_RoutesThroughI18n proves cliMsgf prefers the installed
// translator over its format-string fallback.
func TestCliMsgf_RoutesThroughI18n(t *testing.T) {
	cap := &captureTranslator{out: "TRANSLATED_VIA_CLIMSGF"}
	i18n.SetPkgTranslator(cap)
	t.Cleanup(func() { i18n.SetPkgTranslator(nil) })

	got := cliMsgf(
		"llmorchestrator_cli_config_invalid",
		map[string]any{"error": "bad"},
		"Invalid config: %v\n", "bad",
	)
	if got != "TRANSLATED_VIA_CLIMSGF" {
		t.Fatalf("cliMsgf = %q, want %q (i18n seam not wired)", got, "TRANSLATED_VIA_CLIMSGF")
	}
	if cap.lastID != "llmorchestrator_cli_config_invalid" {
		t.Fatalf("translator received id %q, want %q", cap.lastID, "llmorchestrator_cli_config_invalid")
	}
}
