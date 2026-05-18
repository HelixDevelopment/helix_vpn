// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------
// Round-76 §11.4 — QwenCodeAgent real-wiring unit tests. FINAL builder
// in the LLMOrchestrator round-60 sentinel arc (rounds 64+66+69+71+76 =
// 5/5 — arc COMPLETE).
//
// Each test that depends on a binary stages a mock `qwen` shell script
// in t.TempDir() and prepends that directory to PATH for the scope of
// the subtest. This is the same Go test pattern round-64 / round-66 /
// round-69 / round-71 used for OpenCodeAgent / ClaudeCodeAgent /
// GeminiAgent / JunieAgent: real os/exec, real subprocess, real
// stdout/stderr, real exit codes — nothing about the bridge is mocked,
// only the CLI binary is replaced with a script that mimics Qwen Code
// CLI's surface (`qwen <prompt>` echoes the prompt; `qwen --version`
// returns a version string).
//
// This pattern keeps the production code under test (NewQwenCodeAgent
// + Send + Health) honest per CONST-050(A) — production code never
// imports test mocks; the test substitutes the BINARY, not the agent.
// ---------------------------------------------------------------------

// stageMockQwen writes a shell script at <dir>/qwen that reacts to
// `--version` / positional / -p / --prompt arguments. body lets the
// test inject custom behaviour (sleep, exit non-zero, etc.).
//
// Returns the full path to the staged script. The script is chmod 0755.
//
// Linux/macOS only — Windows is intentionally skipped (mirrors the
// round-64 / round-66 / round-69 / round-71 SKIP-OK marker pattern).
func stageMockQwen(t *testing.T, dir string, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("SKIP-OK: #LLMORCHESTRATOR-QWENCODE-ROUND76-WIN — mock qwen shell script is POSIX-only")
	}
	path := filepath.Join(dir, "qwen")
	script := "#!/bin/sh\n" + body
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("stage mock qwen: %v", err)
	}
	return path
}

// ---------------------------------------------------------------------
// NewQwenCodeAgent — constructor tests
// ---------------------------------------------------------------------

func TestNewQwenCodeAgent_NoBinaryOnPath_ReturnsBinaryNotFound(t *testing.T) {
	emptyDir := t.TempDir()
	// Reuse round-64's isolatePath helper from opencode_agent_test.go.
	isolatePath(t, emptyDir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err == nil {
		t.Fatal("NewQwenCodeAgent with empty PATH returned nil error — ErrQwenCodeBinaryNotFound regression")
	}
	if a != nil {
		t.Errorf("NewQwenCodeAgent returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, ErrQwenCodeBinaryNotFound) {
		t.Errorf("errors.Is(err, ErrQwenCodeBinaryNotFound) = false; err = %v", err)
	}
}

func TestNewQwenCodeAgent_ExplicitBinaryMissing_ReturnsBinaryNotFound(t *testing.T) {
	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{Binary: "/nonexistent/path/qwen"})
	if err == nil {
		t.Fatal("NewQwenCodeAgent with bogus Binary returned nil error")
	}
	if a != nil {
		t.Errorf("NewQwenCodeAgent returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, ErrQwenCodeBinaryNotFound) {
		t.Errorf("errors.Is(err, ErrQwenCodeBinaryNotFound) = false; err = %v", err)
	}
}

func TestNewQwenCodeAgent_DefaultBinary_FoundOnPath_OK(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockQwen(t, dir, `exit 0`)
	prependPath(t, dir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent (default Binary, staged PATH): unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("NewQwenCodeAgent returned nil agent + nil error — CONTRACT-bluff")
	}
	if a.Name() != "qwen-code" {
		t.Errorf("Name() = %q, want %q", a.Name(), "qwen-code")
	}
	if a.ID() == "" {
		t.Error("ID() returned empty string")
	}
	// Interface compliance — the *QwenCodeAgent MUST satisfy Agent so
	// SimpleAgentPool can hand it back from Acquire.
	var _ Agent = a
}

// ---------------------------------------------------------------------
// Send — real subprocess invocation
// ---------------------------------------------------------------------

func TestQwenCodeAgent_Send_SuccessfulInvocation(t *testing.T) {
	dir := t.TempDir()
	// Mock script: `qwen <message>` (positional default) echoes a
	// canned response that references the message so the test can
	// verify wire-through.
	_ = stageMockQwen(t, dir, `
echo "QWEN-CANNED-RESPONSE: $*"
exit 0
`)
	prependPath(t, dir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent: %v", err)
	}

	resp, err := a.Send(context.Background(), "what is 2+2?")
	if err != nil {
		t.Fatalf("Send: unexpected error: %v", err)
	}
	if !strings.Contains(resp.Content, "QWEN-CANNED-RESPONSE") {
		t.Errorf("Send response missing canned marker; got: %q", resp.Content)
	}
	if !strings.Contains(resp.Content, "what is 2+2?") {
		t.Errorf("Send response missing wired-through prompt; got: %q", resp.Content)
	}
	if resp.Latency <= 0 {
		t.Errorf("Send response Latency = %v, want > 0", resp.Latency)
	}
}

func TestQwenCodeAgent_Send_PromptFlagOverride_OK(t *testing.T) {
	dir := t.TempDir()
	// Mock script that ONLY recognises a single argv slot starting
	// with `--prompt=` — proves the PromptFlag override actually wires
	// through to argv (default is positional).
	_ = stageMockQwen(t, dir, `
case "$1" in
  --prompt=*)
    echo "QWEN-PROMPT-FLAG: ${1#--prompt=}"
    exit 0
    ;;
esac
echo "expected --prompt=…, got: $*" >&2
exit 3
`)
	prependPath(t, dir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{PromptFlag: "--prompt"})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent: %v", err)
	}
	resp, err := a.Send(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("Send: unexpected error: %v", err)
	}
	if !strings.Contains(resp.Content, "QWEN-PROMPT-FLAG") {
		t.Errorf("Send response missing prompt-flag marker; got: %q", resp.Content)
	}
	if !strings.Contains(resp.Content, "hello world") {
		t.Errorf("Send response missing wired-through prompt; got: %q", resp.Content)
	}
}

func TestQwenCodeAgent_Send_NonZeroExit_ReturnsInvocationFailed(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockQwen(t, dir, `
echo "synthetic qwen stderr" >&2
exit 1
`)
	prependPath(t, dir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent: %v", err)
	}

	resp, err := a.Send(context.Background(), "anything")
	if err == nil {
		t.Fatal("Send returned nil error despite mock exit 1 — CONTRACT-bluff regression")
	}
	if !errors.Is(err, ErrQwenCodeInvocationFailed) {
		t.Errorf("errors.Is(err, ErrQwenCodeInvocationFailed) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "synthetic qwen stderr") {
		t.Errorf("Send error should surface captured stderr; got: %v", err)
	}
	// The exec.ExitError MUST still be reachable via errors.As so
	// callers can extract the exit code.
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Errorf("errors.As(err, *exec.ExitError) = false; err = %v", err)
	} else if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", exitErr.ExitCode())
	}
	if resp.Error == nil {
		t.Error("Response.Error not populated on failure")
	}
}

func TestQwenCodeAgent_Send_ContextCancel_ReturnsCtxErr(t *testing.T) {
	dir := t.TempDir()
	// Mock script: sleeps long enough that the test's ctx will cancel
	// before it exits. This proves the production code path honors ctx
	// via exec.CommandContext + process-group SIGKILL (setProcessGroup
	// / killProcessGroup, shared with OpenCodeAgent + ClaudeCodeAgent
	// + GeminiAgent + JunieAgent).
	_ = stageMockQwen(t, dir, `exec tail -f /dev/null`)
	prependPath(t, dir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = a.Send(ctx, "ignored")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Send returned nil error despite ctx cancellation — CONTRACT-bluff regression")
	}
	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("err did not match context.DeadlineExceeded or context.Canceled; got: %v", err)
	}
	// SIGKILL via exec.CommandContext should reap within a few hundred ms,
	// not block indefinitely on tail -f /dev/null.
	if elapsed > 5*time.Second {
		t.Errorf("Send took %v after ctx deadline 100ms — exec.CommandContext did not SIGKILL the child", elapsed)
	}
}

// ---------------------------------------------------------------------
// SendWithAttachments — file-flag handling
// ---------------------------------------------------------------------

func TestQwenCodeAgent_SendWithAttachments_WiresIncludeDirectoriesFlags(t *testing.T) {
	dir := t.TempDir()
	// Mock script: dumps full argv to stdout so the test can verify
	// every --include-directories= flag landed in the correct slot.
	_ = stageMockQwen(t, dir, `
printf 'ARGV:'
for arg in "$@"; do
  printf ' %s' "$arg"
done
printf '\n'
exit 0
`)
	prependPath(t, dir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent: %v", err)
	}
	attachments := []Attachment{
		{Path: "/tmp/a"},
		{Path: "/tmp/b", Name: "dir_b"},
		{Path: "" /* skipped */},
	}
	resp, err := a.SendWithAttachments(context.Background(), "summarise", attachments)
	if err != nil {
		t.Fatalf("SendWithAttachments: %v", err)
	}
	if !strings.Contains(resp.Content, "--include-directories=/tmp/a") {
		t.Errorf("argv missing --include-directories=/tmp/a; got: %q", resp.Content)
	}
	if !strings.Contains(resp.Content, "--include-directories=/tmp/b") {
		t.Errorf("argv missing --include-directories=/tmp/b; got: %q", resp.Content)
	}
	if strings.Contains(resp.Content, "--include-directories= ") || strings.Contains(resp.Content, "--include-directories=\n") {
		t.Errorf("argv contains empty --include-directories= from skipped attachment; got: %q", resp.Content)
	}
	if !strings.HasSuffix(strings.TrimRight(resp.Content, "\n"), "summarise") {
		t.Errorf("argv should end with prompt; got: %q", resp.Content)
	}
}

// ---------------------------------------------------------------------
// SendStream — typed not-yet-wired error
// ---------------------------------------------------------------------

func TestQwenCodeAgent_SendStream_ReturnsTypedNotWired(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockQwen(t, dir, `exit 0`)
	prependPath(t, dir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent: %v", err)
	}
	ch, err := a.SendStream(context.Background(), "ignored")
	if err == nil {
		t.Fatal("SendStream returned nil error — anti-bluff regression (must return typed not-wired error per CONST-050(A))")
	}
	if ch != nil {
		t.Errorf("SendStream returned non-nil channel alongside not-wired error: %v", ch)
	}
	if !strings.Contains(err.Error(), "SendStream not yet wired") {
		t.Errorf("SendStream error message missing not-wired narrative; got: %v", err)
	}
}

// ---------------------------------------------------------------------
// Health — `qwen --version` probe
// ---------------------------------------------------------------------

func TestQwenCodeAgent_Health_VersionPasses(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockQwen(t, dir, `
if [ "$1" = "--version" ]; then
  echo "qwen version 0.0.0-mock"
  exit 0
fi
exit 2
`)
	prependPath(t, dir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent: %v", err)
	}

	status := a.Health(context.Background())
	if !status.Healthy {
		t.Errorf("Health = unhealthy; err = %v", status.Error)
	}
	if status.AgentName != "qwen-code" {
		t.Errorf("Health.AgentName = %q, want %q", status.AgentName, "qwen-code")
	}
	if status.Error != nil {
		t.Errorf("Health.Error = %v, want nil", status.Error)
	}
}

func TestQwenCodeAgent_Health_VersionFails_Unhealthy(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockQwen(t, dir, `exit 7`)
	prependPath(t, dir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent: %v", err)
	}

	status := a.Health(context.Background())
	if status.Healthy {
		t.Error("Health.Healthy = true despite mock exit 7 — CONTRACT-bluff regression")
	}
	if status.Error == nil {
		t.Error("Health.Error nil despite mock failure")
	}
}

// ---------------------------------------------------------------------
// Start / Stop / IsRunning
// ---------------------------------------------------------------------

func TestQwenCodeAgent_StartStop_IsRunning(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockQwen(t, dir, `exit 0`)
	prependPath(t, dir)

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent: %v", err)
	}
	if a.IsRunning() {
		t.Error("IsRunning() = true on fresh agent, want false")
	}
	if err := a.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !a.IsRunning() {
		t.Error("IsRunning() = false after Start, want true")
	}
	if err := a.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if a.IsRunning() {
		t.Error("IsRunning() = true after Stop, want false")
	}
}

// ---------------------------------------------------------------------
// Builder integration tests — round-76 narrowed sentinels
// ---------------------------------------------------------------------

func TestQwenCodeClientBuilderFromConfig_ZeroConfig_ReturnsNotConfigured(t *testing.T) {
	b := QwenCodeClientBuilderFromConfig(QwenCodeBuilderConfig{})
	a, err := b(context.Background())
	if err == nil {
		t.Fatal("QwenCodeClientBuilderFromConfig(zero) returned nil error — ErrQwenCodeClientNotConfigured regression")
	}
	if a != nil {
		t.Errorf("QwenCodeClientBuilderFromConfig returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, ErrQwenCodeClientNotConfigured) {
		t.Errorf("errors.Is(err, ErrQwenCodeClientNotConfigured) = false; err = %v", err)
	}
}

func TestQwenCodeClientBuilderFromConfig_ValidConfig_BuildsAgent(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockQwen(t, dir, `exit 0`)
	prependPath(t, dir)

	b := QwenCodeClientBuilderFromConfig(QwenCodeBuilderConfig{Binary: "qwen"})
	a, err := b(context.Background())
	if err != nil {
		t.Fatalf("QwenCodeClientBuilderFromConfig(valid): unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("QwenCodeClientBuilderFromConfig returned nil agent + nil error")
	}
	if a.Name() != "qwen-code" {
		t.Errorf("agent.Name() = %q, want %q", a.Name(), "qwen-code")
	}
}

func TestQwenCodeClientBuilder_BinaryOnPath_BuildsRealAgent(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockQwen(t, dir, `exit 0`)
	prependPath(t, dir)

	b := QwenCodeClientBuilder(&PoolConfig{Size: 1})
	a, err := b(context.Background())
	if err != nil {
		t.Fatalf("QwenCodeClientBuilder({Size:1}, qwen on PATH): unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("QwenCodeClientBuilder returned nil agent + nil error — round-76 wiring regression")
	}
	if _, ok := a.(*QwenCodeAgent); !ok {
		t.Errorf("QwenCodeClientBuilder returned %T, want *QwenCodeAgent", a)
	}
}

func TestQwenCodeClientBuilder_NilConfig_ReturnsNotWired(t *testing.T) {
	b := QwenCodeClientBuilder(nil)
	a, err := b(context.Background())
	if err == nil {
		t.Fatal("QwenCodeClientBuilder(nil) returned nil error — round-60 narrowed sentinel regression")
	}
	if a != nil {
		t.Errorf("QwenCodeClientBuilder(nil) returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, ErrQwenCodeClientNotWired) {
		t.Errorf("errors.Is(err, ErrQwenCodeClientNotWired) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------
// SimpleAgentPool end-to-end — round-76 real Agent through real pool
// ---------------------------------------------------------------------

func TestNewQwenCodePool_WithRealBuilder_AcquireReturnsRealAgent(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockQwen(t, dir, `exit 0`)
	prependPath(t, dir)

	pool, err := NewQwenCodePool(&PoolConfig{Size: 2})
	if err != nil {
		t.Fatalf("NewQwenCodePool: %v", err)
	}

	a, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err != nil {
		t.Fatalf("Acquire: unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("Acquire returned nil agent + nil error — CONTRACT-bluff")
	}
	if _, ok := a.(*QwenCodeAgent); !ok {
		t.Errorf("pool.Acquire returned %T, want *QwenCodeAgent", a)
	}
	if a.Name() != "qwen-code" {
		t.Errorf("agent.Name() = %q, want %q", a.Name(), "qwen-code")
	}
}

func TestNewQwenCodePool_NoBinaryOnPath_AcquireSurfacesBinaryNotFound(t *testing.T) {
	emptyDir := t.TempDir()
	isolatePath(t, emptyDir)

	pool, err := NewQwenCodePool(&PoolConfig{Size: 1})
	if err != nil {
		t.Fatalf("NewQwenCodePool: %v", err)
	}

	a, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err == nil {
		t.Fatal("Acquire returned nil error despite missing binary — round-76 wiring regression")
	}
	if a != nil {
		t.Errorf("Acquire returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, ErrQwenCodeBinaryNotFound) {
		t.Errorf("errors.Is(err, ErrQwenCodeBinaryNotFound) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------
// Round-76 sentinel distinguishability — paired with round-60 + 64 +
// 66 + 69 + 71
// ---------------------------------------------------------------------

// TestRound76Sentinels_AreDistinct — disambiguation invariant.
// Round-76 introduces 3 new sentinels (ErrQwenCodeBinaryNotFound,
// ErrQwenCodeClientNotConfigured, ErrQwenCodeInvocationFailed) on top
// of round-60's 5 builder-wired sentinels, round-64's 3 opencode-
// specific sentinels, round-66's 3 claudecode-specific sentinels,
// round-69's 3 gemini-specific sentinels, and round-71's 3 junie-
// specific sentinels. All 20 MUST be distinct under errors.Is so
// callers can disambiguate which provider's wiring is in which state
// (binary missing vs config missing vs invocation failed vs the
// nil-cfg backstop) — and after round-76 every provider in the arc
// is wired (5/5 builders complete: OpenCode + ClaudeCode + Gemini +
// Junie + QwenCode).
func TestRound76Sentinels_AreDistinct(t *testing.T) {
	sentinels := []error{
		// Round-76 (this round) — FINAL builder of the arc
		ErrQwenCodeBinaryNotFound,
		ErrQwenCodeClientNotConfigured,
		ErrQwenCodeInvocationFailed,
		// Round-71 (junie wiring)
		ErrJunieBinaryNotFound,
		ErrJunieClientNotConfigured,
		ErrJunieInvocationFailed,
		// Round-69 (gemini wiring)
		ErrGeminiBinaryNotFound,
		ErrGeminiClientNotConfigured,
		ErrGeminiInvocationFailed,
		// Round-66 (claudecode wiring)
		ErrClaudeCodeBinaryNotFound,
		ErrClaudeCodeClientNotConfigured,
		ErrClaudeCodeInvocationFailed,
		// Round-64 (opencode wiring)
		ErrOpenCodeBinaryNotFound,
		ErrOpenCodeClientNotConfigured,
		ErrOpenCodeInvocationFailed,
		// Round-60 (narrowed builder sentinels)
		ErrOpenCodeClientNotWired,
		ErrClaudeCodeClientNotWired,
		ErrGeminiClientNotWired,
		ErrJunieClientNotWired,
		ErrQwenCodeClientNotWired,
	}
	if got, want := len(sentinels), 20; got != want {
		t.Fatalf("sentinel grid size = %d, want %d (round-76 20-sentinel matrix — arc complete)", got, want)
	}
	for i, a := range sentinels {
		for j, b := range sentinels {
			if i == j {
				continue
			}
			if errors.Is(a, b) {
				t.Errorf("sentinel %d (%v) incorrectly matches sentinel %d (%v)", i, a, j, b)
			}
		}
	}
}

// ---------------------------------------------------------------------
// Round-76 milestone — all 5 builders in the round-60 sentinel arc
// are wired. This test documents arc completion: every vendor's
// builder, when given a non-nil PoolConfig AND a mock binary on PATH,
// returns a real, non-nil, non-sentinel Agent.
// ---------------------------------------------------------------------

// TestRound76_AllFiveBuildersAreWired asserts the LLMOrchestrator
// builder arc is COMPLETE after round-76: every one of the 5 builders
// originally stubbed in round-60 (OpenCode, ClaudeCode, Gemini, Junie,
// QwenCode) now returns a real, non-nil Agent when invoked with a
// non-nil PoolConfig AND the canonical CLI binary on $PATH. The test
// stages a mock script for each binary, runs the corresponding builder,
// and asserts (a) err == nil, (b) returned Agent is non-nil, (c) the
// returned Agent is the expected concrete type per vendor.
//
// This is the documenting test for the arc closure (rounds
// 64+66+69+71+76 = 5/5).
func TestRound76_AllFiveBuildersAreWired(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SKIP-OK: #LLMORCHESTRATOR-QWENCODE-ROUND76-WIN — mock shell scripts are POSIX-only")
	}

	// Stage a single tempdir with one mock script per vendor binary
	// so all 5 builders can resolve their canonical binary via PATH.
	dir := t.TempDir()
	for _, bin := range []string{"opencode", "claude", "gemini", "junie", "qwen"} {
		path := filepath.Join(dir, bin)
		if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("stage mock %s: %v", bin, err)
		}
	}
	prependPath(t, dir)

	// NOTE: agent.Name() values are the canonical vendor identifiers used
	// internally — ClaudeCodeAgent.Name() returns "claudecode" (no hyphen)
	// for historical reasons; the other four match their pool names
	// directly. The wantName field captures the exact Name() each
	// concrete type returns so this milestone test catches future
	// regressions in either direction.
	cases := []struct {
		name     string
		wantName string
		builder  func(*PoolConfig) ClientBuilder
		// matcher returns true if a is the concrete vendor agent type.
		matcher func(a Agent) bool
	}{
		{
			name:     "opencode",
			wantName: "opencode",
			builder:  OpenCodeClientBuilder,
			matcher:  func(a Agent) bool { _, ok := a.(*OpenCodeAgent); return ok },
		},
		{
			name:     "claude-code",
			wantName: "claudecode",
			builder:  ClaudeCodeClientBuilder,
			matcher:  func(a Agent) bool { _, ok := a.(*ClaudeCodeAgent); return ok },
		},
		{
			name:     "gemini",
			wantName: "gemini",
			builder:  GeminiClientBuilder,
			matcher:  func(a Agent) bool { _, ok := a.(*GeminiAgent); return ok },
		},
		{
			name:     "junie",
			wantName: "junie",
			builder:  JunieClientBuilder,
			matcher:  func(a Agent) bool { _, ok := a.(*JunieAgent); return ok },
		},
		{
			name:     "qwen-code",
			wantName: "qwen-code",
			builder:  QwenCodeClientBuilder,
			matcher:  func(a Agent) bool { _, ok := a.(*QwenCodeAgent); return ok },
		},
	}

	if got, want := len(cases), 5; got != want {
		t.Fatalf("milestone test: builder count = %d, want %d (round-76 closes 5/5 arc)", got, want)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := tc.builder(&PoolConfig{Size: 1})
			a, err := b(context.Background())
			if err != nil {
				t.Fatalf("%s builder returned error after round-76 wiring: %v", tc.name, err)
			}
			if a == nil {
				t.Fatalf("%s builder returned nil agent + nil error — CONTRACT-bluff regression", tc.name)
			}
			if !tc.matcher(a) {
				t.Errorf("%s builder returned wrong concrete type: %T", tc.name, a)
			}
			if a.Name() != tc.wantName {
				t.Errorf("%s builder agent.Name() = %q, want %q", tc.name, a.Name(), tc.wantName)
			}
		})
	}
}

// ---------------------------------------------------------------------
// Real-binary integration test — runs against the actual installed
// `qwen` CLI if present, otherwise SKIP per §11.4.1 SKIP-OK rules.
// ---------------------------------------------------------------------

func TestQwenCodeAgent_RealBinary_VersionRoundtripsOK(t *testing.T) {
	if _, err := exec.LookPath(DefaultQwenCodeBinary); err != nil {
		t.Skip("SKIP-OK: #LLMORCHESTRATOR-QWENCODE-REAL-ROUND76 — `qwen` not installed on PATH; install Alibaba Qwen Code CLI via `npm install -g @qwen-code/qwen-code` to exercise the real-binary path")
	}

	a, err := NewQwenCodeAgent(QwenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewQwenCodeAgent (real binary): %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status := a.Health(ctx)
	if !status.Healthy {
		t.Errorf("Health (real binary): unhealthy; err = %v", status.Error)
	}
	if status.Latency <= 0 {
		t.Errorf("Health.Latency = %v, want > 0", status.Latency)
	}
}
