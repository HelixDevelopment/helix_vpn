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
// Round-64 §11.4 — OpenCodeAgent real-wiring unit tests.
//
// Each test that depends on a binary stages a mock `opencode` shell
// script in t.TempDir() and prepends that directory to PATH for the
// scope of the subtest. This is the standard Go test pattern for
// CLI-bridge tests: real os/exec, real subprocess, real stdout/stderr,
// real exit codes — nothing about the bridge is mocked, only the CLI
// binary is replaced with a script that mimics OpenCode's surface
// (`opencode run <prompt>` echoes the prompt; `opencode --version`
// returns a version string).
//
// This pattern keeps the production code under test (NewOpenCodeAgent
// + Send + Health) honest per CONST-050(A) — production code never
// imports test mocks; the test substitutes the BINARY, not the agent.
// ---------------------------------------------------------------------

// stageMockOpencode writes a shell script at <dir>/opencode that
// reacts to `--version` / `run` arguments. body lets the test inject
// custom behaviour (sleep, exit non-zero, etc.).
//
// Returns the full path to the staged script. The script is chmod 0755.
//
// Linux/macOS only — Windows is intentionally skipped because the
// LLMOrchestrator submodule's primary deployment target is Linux
// servers and macOS dev machines. Windows-host CI would need a .bat
// equivalent that is not yet in scope.
func stageMockOpencode(t *testing.T, dir string, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("SKIP-OK: #LLMORCHESTRATOR-OPENCODE-ROUND64-WIN — mock opencode shell script is POSIX-only")
	}
	path := filepath.Join(dir, "opencode")
	script := "#!/bin/sh\n" + body
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("stage mock opencode: %v", err)
	}
	return path
}

// prependPath prepends dir to PATH for the lifetime of the test
// via t.Setenv. The original PATH is restored automatically.
func prependPath(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// isolatePath forces PATH to a dir with no opencode binary so
// exec.LookPath fails predictably.
func isolatePath(t *testing.T, emptyDir string) {
	t.Helper()
	t.Setenv("PATH", emptyDir)
}

// ---------------------------------------------------------------------
// NewOpenCodeAgent — constructor tests
// ---------------------------------------------------------------------

func TestNewOpenCodeAgent_NoBinaryOnPath_ReturnsBinaryNotFound(t *testing.T) {
	emptyDir := t.TempDir()
	isolatePath(t, emptyDir)

	a, err := NewOpenCodeAgent(OpenCodeAgentConfig{})
	if err == nil {
		t.Fatal("NewOpenCodeAgent with empty PATH returned nil error — ErrOpenCodeBinaryNotFound regression")
	}
	if a != nil {
		t.Errorf("NewOpenCodeAgent returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, ErrOpenCodeBinaryNotFound) {
		t.Errorf("errors.Is(err, ErrOpenCodeBinaryNotFound) = false; err = %v", err)
	}
}

func TestNewOpenCodeAgent_ExplicitBinaryMissing_ReturnsBinaryNotFound(t *testing.T) {
	a, err := NewOpenCodeAgent(OpenCodeAgentConfig{Binary: "/nonexistent/path/opencode"})
	if err == nil {
		t.Fatal("NewOpenCodeAgent with bogus Binary returned nil error")
	}
	if a != nil {
		t.Errorf("NewOpenCodeAgent returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, ErrOpenCodeBinaryNotFound) {
		t.Errorf("errors.Is(err, ErrOpenCodeBinaryNotFound) = false; err = %v", err)
	}
}

func TestNewOpenCodeAgent_DefaultBinary_FoundOnPath_OK(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockOpencode(t, dir, `exit 0`)
	prependPath(t, dir)

	a, err := NewOpenCodeAgent(OpenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewOpenCodeAgent (default Binary, staged PATH): unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("NewOpenCodeAgent returned nil agent + nil error — CONTRACT-bluff")
	}
	if a.Name() != "opencode" {
		t.Errorf("Name() = %q, want %q", a.Name(), "opencode")
	}
	if a.ID() == "" {
		t.Error("ID() returned empty string")
	}
	// Interface compliance — the *OpenCodeAgent MUST satisfy Agent so
	// SimpleAgentPool can hand it back from Acquire.
	var _ Agent = a
}

// ---------------------------------------------------------------------
// Send — real subprocess invocation
// ---------------------------------------------------------------------

func TestOpenCodeAgent_Send_SuccessfulInvocation(t *testing.T) {
	dir := t.TempDir()
	// Mock script: `opencode run <message>` echoes a canned response
	// that references the message so the test can verify wire-through.
	_ = stageMockOpencode(t, dir, `
if [ "$1" = "run" ]; then
  shift
  echo "OPENCODE-CANNED-RESPONSE: $*"
  exit 0
fi
echo "unknown args: $*" >&2
exit 2
`)
	prependPath(t, dir)

	a, err := NewOpenCodeAgent(OpenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewOpenCodeAgent: %v", err)
	}

	resp, err := a.Send(context.Background(), "what is 2+2?")
	if err != nil {
		t.Fatalf("Send: unexpected error: %v", err)
	}
	if !strings.Contains(resp.Content, "OPENCODE-CANNED-RESPONSE") {
		t.Errorf("Send response missing canned marker; got: %q", resp.Content)
	}
	if !strings.Contains(resp.Content, "what is 2+2?") {
		t.Errorf("Send response missing wired-through prompt; got: %q", resp.Content)
	}
	if resp.Latency <= 0 {
		t.Errorf("Send response Latency = %v, want > 0", resp.Latency)
	}
}

func TestOpenCodeAgent_Send_NonZeroExit_ReturnsInvocationFailed(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockOpencode(t, dir, `
echo "synthetic stderr" >&2
exit 1
`)
	prependPath(t, dir)

	a, err := NewOpenCodeAgent(OpenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewOpenCodeAgent: %v", err)
	}

	resp, err := a.Send(context.Background(), "anything")
	if err == nil {
		t.Fatal("Send returned nil error despite mock exit 1 — CONTRACT-bluff regression")
	}
	if !errors.Is(err, ErrOpenCodeInvocationFailed) {
		t.Errorf("errors.Is(err, ErrOpenCodeInvocationFailed) = false; err = %v", err)
	}
	if !strings.Contains(err.Error(), "synthetic stderr") {
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

func TestOpenCodeAgent_Send_ContextCancel_ReturnsCtxErr(t *testing.T) {
	dir := t.TempDir()
	// Mock script: sleeps long enough that the test's ctx will cancel
	// before it exits. This proves the production code path honors
	// ctx via exec.CommandContext (process gets SIGKILL on ctx.Done()).
	_ = stageMockOpencode(t, dir, `sleep 30`)
	prependPath(t, dir)

	a, err := NewOpenCodeAgent(OpenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewOpenCodeAgent: %v", err)
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
	// not the full 30s the script would sleep otherwise.
	if elapsed > 5*time.Second {
		t.Errorf("Send took %v after ctx deadline 100ms — exec.CommandContext did not SIGKILL the child", elapsed)
	}
}

// ---------------------------------------------------------------------
// Health — `opencode --version` probe
// ---------------------------------------------------------------------

func TestOpenCodeAgent_Health_VersionPasses(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockOpencode(t, dir, `
if [ "$1" = "--version" ]; then
  echo "opencode v0.0.0-mock"
  exit 0
fi
exit 2
`)
	prependPath(t, dir)

	a, err := NewOpenCodeAgent(OpenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewOpenCodeAgent: %v", err)
	}

	status := a.Health(context.Background())
	if !status.Healthy {
		t.Errorf("Health = unhealthy; err = %v", status.Error)
	}
	if status.AgentName != "opencode" {
		t.Errorf("Health.AgentName = %q, want %q", status.AgentName, "opencode")
	}
	if status.Error != nil {
		t.Errorf("Health.Error = %v, want nil", status.Error)
	}
}

func TestOpenCodeAgent_Health_VersionFails_Unhealthy(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockOpencode(t, dir, `exit 7`)
	prependPath(t, dir)

	a, err := NewOpenCodeAgent(OpenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewOpenCodeAgent: %v", err)
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

func TestOpenCodeAgent_StartStop_IsRunning(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockOpencode(t, dir, `exit 0`)
	prependPath(t, dir)

	a, err := NewOpenCodeAgent(OpenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewOpenCodeAgent: %v", err)
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
// Builder integration tests — round-64 narrowed sentinels
// ---------------------------------------------------------------------

func TestOpenCodeClientBuilderFromConfig_ZeroConfig_ReturnsNotConfigured(t *testing.T) {
	b := OpenCodeClientBuilderFromConfig(OpenCodeBuilderConfig{})
	a, err := b(context.Background())
	if err == nil {
		t.Fatal("OpenCodeClientBuilderFromConfig(zero) returned nil error — ErrOpenCodeClientNotConfigured regression")
	}
	if a != nil {
		t.Errorf("OpenCodeClientBuilderFromConfig returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, ErrOpenCodeClientNotConfigured) {
		t.Errorf("errors.Is(err, ErrOpenCodeClientNotConfigured) = false; err = %v", err)
	}
}

func TestOpenCodeClientBuilderFromConfig_ValidConfig_BuildsAgent(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockOpencode(t, dir, `exit 0`)
	prependPath(t, dir)

	b := OpenCodeClientBuilderFromConfig(OpenCodeBuilderConfig{Binary: "opencode"})
	a, err := b(context.Background())
	if err != nil {
		t.Fatalf("OpenCodeClientBuilderFromConfig(valid): unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("OpenCodeClientBuilderFromConfig returned nil agent + nil error")
	}
	if a.Name() != "opencode" {
		t.Errorf("agent.Name() = %q, want %q", a.Name(), "opencode")
	}
}

func TestOpenCodeClientBuilder_BinaryOnPath_BuildsRealAgent(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockOpencode(t, dir, `exit 0`)
	prependPath(t, dir)

	b := OpenCodeClientBuilder(&PoolConfig{Size: 1})
	a, err := b(context.Background())
	if err != nil {
		t.Fatalf("OpenCodeClientBuilder({Size:1}, opencode on PATH): unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("OpenCodeClientBuilder returned nil agent + nil error — round-64 wiring regression")
	}
	if _, ok := a.(*OpenCodeAgent); !ok {
		t.Errorf("OpenCodeClientBuilder returned %T, want *OpenCodeAgent", a)
	}
}

// ---------------------------------------------------------------------
// SimpleAgentPool end-to-end — round-64 real Agent through real pool
// ---------------------------------------------------------------------

func TestNewOpenCodePool_WithRealBuilder_AcquireReturnsRealAgent(t *testing.T) {
	dir := t.TempDir()
	_ = stageMockOpencode(t, dir, `exit 0`)
	prependPath(t, dir)

	pool, err := NewOpenCodePool(&PoolConfig{Size: 2})
	if err != nil {
		t.Fatalf("NewOpenCodePool: %v", err)
	}

	a, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err != nil {
		t.Fatalf("Acquire: unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("Acquire returned nil agent + nil error — CONTRACT-bluff")
	}
	if _, ok := a.(*OpenCodeAgent); !ok {
		t.Errorf("pool.Acquire returned %T, want *OpenCodeAgent", a)
	}
	if a.Name() != "opencode" {
		t.Errorf("agent.Name() = %q, want %q", a.Name(), "opencode")
	}
}

func TestNewOpenCodePool_NoBinaryOnPath_AcquireSurfacesBinaryNotFound(t *testing.T) {
	emptyDir := t.TempDir()
	isolatePath(t, emptyDir)

	pool, err := NewOpenCodePool(&PoolConfig{Size: 1})
	if err != nil {
		t.Fatalf("NewOpenCodePool: %v", err)
	}

	a, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err == nil {
		t.Fatal("Acquire returned nil error despite missing binary — round-64 wiring regression")
	}
	if a != nil {
		t.Errorf("Acquire returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, ErrOpenCodeBinaryNotFound) {
		t.Errorf("errors.Is(err, ErrOpenCodeBinaryNotFound) = false; err = %v", err)
	}
}

// ---------------------------------------------------------------------
// Round-64 sentinel distinguishability (paired with round-60 sentinels)
// ---------------------------------------------------------------------

func TestRound64Sentinels_AreDistinct(t *testing.T) {
	// Round-64 introduces 3 new sentinels in addition to the round-60
	// five. All eight MUST be distinct under errors.Is so callers can
	// disambiguate (binary missing vs config missing vs invocation
	// failed vs the four non-opencode providers still pending wiring).
	sentinels := []error{
		ErrOpenCodeBinaryNotFound,
		ErrOpenCodeClientNotConfigured,
		ErrOpenCodeInvocationFailed,
		ErrOpenCodeClientNotWired, // round-60, narrowed
		ErrClaudeCodeClientNotWired,
		ErrGeminiClientNotWired,
		ErrJunieClientNotWired,
		ErrQwenCodeClientNotWired,
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
// Real-binary integration test — runs against the actual installed
// `opencode` CLI if present, otherwise SKIP per §11.4.1 SKIP-OK rules.
// ---------------------------------------------------------------------

func TestOpenCodeAgent_RealBinary_VersionRoundtripsOK(t *testing.T) {
	if _, err := exec.LookPath(DefaultOpenCodeBinary); err != nil {
		t.Skip("SKIP-OK: #LLMORCHESTRATOR-OPENCODE-REAL-ROUND64 — `opencode` not installed on PATH; install from https://opencode.ai to exercise the real-binary path")
	}

	a, err := NewOpenCodeAgent(OpenCodeAgentConfig{})
	if err != nil {
		t.Fatalf("NewOpenCodeAgent (real binary): %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := a.Health(ctx)
	if !status.Healthy {
		t.Errorf("Health (real binary): unhealthy; err = %v", status.Error)
	}
	if status.Latency <= 0 {
		t.Errorf("Health.Latency = %v, want > 0", status.Latency)
	}
}
