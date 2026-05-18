// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Round-64 §11.4 forensic anchor — real OpenCode CLI client wiring.
//
// This file converts the round-60 ErrOpenCodeClientNotWired "stub returns
// sentinel" surface into a real os/exec-spawned bridge to the `opencode`
// CLI binary. The architecture preserves the round-60 SimpleAgentPool
// composition: SimpleAgentPool still owns capacity, available/in-use
// bookkeeping, blocking Acquire, and Shutdown. The OpenCodeAgent built
// here is what the pool hands back when its ClientBuilder is wired with
// a non-empty config.
//
// Transport pattern (verified via `opencode --help` 2026-05-18):
//
//   opencode run <message>            → run with a positional message
//   opencode --version                → version probe (health-check)
//
// The Agent runs each `Send` call as a fresh `opencode run` subprocess
// (one-shot invocation, no persistent session). This matches how the
// CLI is documented to be used non-interactively and avoids leaking a
// long-lived background process when the pool is shut down.
//
// API keys, model selection, and any extra flags (--model, --agent,
// --format json) are passed through cfg.Env / cfg.ExtraArgs so this
// file never hardcodes credentials (CONST-042) and never assumes which
// provider/model the operator wants (CONST-046).
//
// Constitutional anchors: CONST-035 (anti-bluff covenant — every PASS
// carries runtime evidence), CONST-042 (no-secret-leak — env-key
// handling), CONST-050(A) (no-fakes-beyond-unit-tests — production
// code path never imports test mocks), Article XI §11.9 (end-user
// quality forensic anchor).

// DefaultOpenCodeBinary is the binary name resolved via $PATH when
// OpenCodeAgentConfig.Binary is empty.
const DefaultOpenCodeBinary = "opencode"

// ErrOpenCodeBinaryNotFound is returned by NewOpenCodeAgent when the
// configured (or default) `opencode` binary cannot be located on $PATH.
//
// Distinct from round-60's ErrOpenCodeClientNotWired (which signalled
// "no implementation exists at all") — this sentinel fires AFTER round-64
// wired the real implementation but the binary is missing at runtime.
var ErrOpenCodeBinaryNotFound = errors.New(
	"opencode agent: binary not found on PATH — install `opencode` " +
		"(https://opencode.ai) or set OpenCodeAgentConfig.Binary to an absolute path")

// ErrOpenCodeClientNotConfigured is returned by OpenCodeClientBuilder
// when the supplied OpenCodeBuilderConfig is zero-value (no binary
// override, no extra args, no env). Round-60's
// ErrOpenCodeClientNotWired signalled "implementation not present";
// round-64's ErrOpenCodeClientNotConfigured signals "implementation
// present but caller passed an empty config".
//
// Operators can still get a working agent by passing
// OpenCodeBuilderConfig{Binary: "opencode"} (or letting the default
// fire) — see UseOpenCodeBuilderConfigDefaults helper.
var ErrOpenCodeClientNotConfigured = errors.New(
	"opencode agent: OpenCodeBuilderConfig is zero-value — populate " +
		"Binary (or rely on PATH-resolved default `opencode`) and re-invoke")

// ErrOpenCodeInvocationFailed wraps any non-zero exit from `opencode run`.
// Callers may errors.Is on this sentinel to distinguish CLI failures
// from binary-not-found / context-cancel failures. The underlying
// *exec.ExitError (including its captured stderr) is preserved via
// %w so callers may errors.As it for the exit code.
var ErrOpenCodeInvocationFailed = errors.New(
	"opencode agent: `opencode run` exited non-zero")

// OpenCodeAgentConfig configures an OpenCodeAgent instance.
//
// Binary may be "" → use DefaultOpenCodeBinary resolved via exec.LookPath.
// ExtraArgs are prepended before the message in `opencode run <args> <message>`.
// WorkingDir defaults to the caller's CWD when empty.
// Env defaults to os.Environ() at agent-construction time when nil —
// callers MUST source any provider API keys (OPENAI_API_KEY,
// ANTHROPIC_API_KEY, etc.) into this slice rather than hardcoding them
// in source (CONST-042).
type OpenCodeAgentConfig struct {
	Binary     string
	ExtraArgs  []string
	WorkingDir string
	Env        []string
	// IDOverride is used by tests to make Agent.ID deterministic; in
	// production it is left empty and a default "opencode-<binary>" ID
	// is generated.
	IDOverride string
}

// OpenCodeBuilderConfig is the shape supplied to OpenCodeClientBuilder.
// It mirrors OpenCodeAgentConfig but is exported under a distinct name
// so future builder-only knobs (retries, per-request timeouts) can land
// without polluting the per-agent config.
type OpenCodeBuilderConfig struct {
	Binary     string
	ExtraArgs  []string
	WorkingDir string
	Env        []string
}

// IsZero reports whether the OpenCodeBuilderConfig has no configured
// fields. Empty config triggers ErrOpenCodeClientNotConfigured at
// build time (callers must opt in to the PATH-resolved default).
func (c OpenCodeBuilderConfig) IsZero() bool {
	return c.Binary == "" &&
		len(c.ExtraArgs) == 0 &&
		c.WorkingDir == "" &&
		len(c.Env) == 0
}

// OpenCodeAgent is a real os/exec-backed bridge to the `opencode` CLI.
//
// Each Send invocation spawns a fresh `opencode run <prompt>` subprocess,
// captures stdout, and returns it as the Response. The subprocess
// honors the parent context's deadline + cancellation via
// exec.CommandContext, so SimpleAgentPool's Shutdown path cancels
// in-flight CLI calls cleanly.
//
// Concurrency: OpenCodeAgent is safe for concurrent Send calls — each
// invocation gets its own subprocess and the only shared mutable state
// (running flag) is guarded by mu.
//
// This type implements the full Agent interface declared in agent.go.
type OpenCodeAgent struct {
	id         string
	binary     string
	extraArgs  []string
	workingDir string
	env        []string

	mu      sync.Mutex
	running bool
}

// NewOpenCodeAgent constructs an OpenCodeAgent, validating that the
// configured (or default) binary exists on $PATH at construction time.
//
// Returns ErrOpenCodeBinaryNotFound if exec.LookPath fails. This is
// the fail-fast contract — callers see a real failure at pool-build
// time instead of a deferred "command not found" inside Acquire.
func NewOpenCodeAgent(cfg OpenCodeAgentConfig) (*OpenCodeAgent, error) {
	bin := cfg.Binary
	if bin == "" {
		bin = DefaultOpenCodeBinary
	}
	resolved, err := exec.LookPath(bin)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOpenCodeBinaryNotFound, err)
	}
	id := cfg.IDOverride
	if id == "" {
		id = "opencode-" + resolved
	}
	return &OpenCodeAgent{
		id:         id,
		binary:     resolved,
		extraArgs:  append([]string(nil), cfg.ExtraArgs...),
		workingDir: cfg.WorkingDir,
		env:        append([]string(nil), cfg.Env...),
	}, nil
}

// ID returns the agent's unique identifier.
func (a *OpenCodeAgent) ID() string { return a.id }

// Name returns the provider name "opencode" — used by the pool's
// preferred-agent matcher and by callers that need to identify which
// CLI backend serviced a given request.
func (a *OpenCodeAgent) Name() string { return "opencode" }

// Start marks the agent as running. The CLI itself is one-shot per
// Send call, so there is no persistent process to spawn here; Start
// only updates state so Health() reflects "ready". Idempotent.
func (a *OpenCodeAgent) Start(_ context.Context) error {
	a.mu.Lock()
	a.running = true
	a.mu.Unlock()
	return nil
}

// Stop marks the agent as not-running. There is no long-lived process
// to terminate; in-flight Send calls cancel through their own contexts.
// Idempotent.
func (a *OpenCodeAgent) Stop(_ context.Context) error {
	a.mu.Lock()
	a.running = false
	a.mu.Unlock()
	return nil
}

// IsRunning reports whether Start has been called since the last Stop.
func (a *OpenCodeAgent) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

// Health runs `opencode --version` as a lightweight probe. PASS = exit 0
// + non-empty stdout. Any failure populates HealthStatus.Error with the
// captured stderr so operators can see what broke.
func (a *OpenCodeAgent) Health(ctx context.Context) HealthStatus {
	start := time.Now()
	out, err := a.runCapture(ctx, []string{"--version"})
	status := HealthStatus{
		AgentID:   a.id,
		AgentName: a.Name(),
		Latency:   time.Since(start),
		CheckedAt: time.Now(),
	}
	if err != nil {
		status.Healthy = false
		status.Error = wrapExitError(err, "opencode --version")
		return status
	}
	if len(out) == 0 {
		status.Healthy = false
		status.Error = errors.New("opencode --version: empty stdout (unexpected for a healthy CLI)")
		return status
	}
	status.Healthy = true
	return status
}

// Send runs `opencode run <prompt>` as a one-shot subprocess and
// returns the captured stdout as the Response.Content. Honors
// ctx cancellation via exec.CommandContext AND a platform-specific
// process-group SIGKILL helper (see runWithGroupKill) so that the
// CLI's child processes (sh subshells, sleep, etc.) are reaped along
// with the parent.
//
// CONST-042: API keys MUST be supplied via a.env (sourced by the caller
// from os.Environ() or a secrets manager); this method never reads
// credentials from anywhere else.
func (a *OpenCodeAgent) Send(ctx context.Context, prompt string) (Response, error) {
	start := time.Now()
	args := make([]string, 0, len(a.extraArgs)+2)
	args = append(args, "run")
	args = append(args, a.extraArgs...)
	args = append(args, prompt)

	out, err := a.runCapture(ctx, args)
	latency := time.Since(start)

	if err != nil {
		// Propagate ctx.Err verbatim when the failure is a cancellation
		// — exec wraps it inside *exec.ExitError, but callers expect
		// errors.Is(err, context.Canceled / context.DeadlineExceeded).
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Response{Latency: latency, Error: ctxErr}, ctxErr
		}
		wrapped := wrapExitError(err, "opencode run")
		return Response{Latency: latency, Error: wrapped}, wrapped
	}

	return Response{
		Content: string(out),
		Latency: latency,
	}, nil
}

// runCapture configures and runs a command with process-group
// SIGKILL on ctx cancellation, returning stdout (matching
// exec.Cmd.Output() semantics).
func (a *OpenCodeAgent) runCapture(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, a.binary, args...)
	if a.workingDir != "" {
		cmd.Dir = a.workingDir
	}
	if len(a.env) > 0 {
		cmd.Env = a.env
	}
	setProcessGroup(cmd)
	cmd.Cancel = func() error {
		return killProcessGroup(cmd)
	}
	return cmd.Output()
}

// SendStream is not yet supported by the OpenCode CLI client integration.
// Streaming requires either `opencode serve` + WebSocket attach OR
// `opencode run --format json` event-stream parsing — both follow-up
// work for a later round. Until then this surfaces a typed error rather
// than silently buffering Send's response into a fake stream.
func (a *OpenCodeAgent) SendStream(_ context.Context, _ string) (<-chan StreamChunk, error) {
	return nil, errors.New("opencode agent: SendStream not yet wired (requires `opencode serve` attach OR --format json parsing — follow-up round)")
}

// SendWithAttachments runs `opencode run --file <path> ... <prompt>`.
// Each attachment.Path is passed as a separate -f flag per the OpenCode
// CLI surface (`opencode run --help`: "--file file(s) to attach to message").
func (a *OpenCodeAgent) SendWithAttachments(ctx context.Context, prompt string, attachments []Attachment) (Response, error) {
	start := time.Now()
	args := make([]string, 0, len(a.extraArgs)+2*len(attachments)+2)
	args = append(args, "run")
	args = append(args, a.extraArgs...)
	for _, att := range attachments {
		if att.Path == "" {
			continue
		}
		args = append(args, "--file", att.Path)
	}
	args = append(args, prompt)

	out, err := a.runCapture(ctx, args)
	latency := time.Since(start)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Response{Latency: latency, Error: ctxErr}, ctxErr
		}
		wrapped := wrapExitError(err, "opencode run --file")
		return Response{Latency: latency, Error: wrapped}, wrapped
	}
	return Response{Content: string(out), Latency: latency}, nil
}

// OutputDir returns the working directory the CLI subprocess runs in.
// When empty, the subprocess inherits the parent's CWD.
func (a *OpenCodeAgent) OutputDir() string { return a.workingDir }

// Capabilities reports the OpenCode CLI's known capability surface. The
// CLI itself supports tool-use (file edits, shell), streaming via the
// `--format json` event stream, and per-model context windows that
// exceed 100k tokens for most modern providers.
func (a *OpenCodeAgent) Capabilities() AgentCapabilities {
	return AgentCapabilities{
		Vision:    false, // depends on selected model; conservative default
		Streaming: false, // wired in a later round per SendStream comment
		ToolUse:   true,
		MaxTokens: 100000,
	}
}

// SupportsVision returns whether the agent (and its current model
// selection) supports vision. Conservative default: false. Operators
// can override after construction if they have selected a vision-capable
// model via cfg.ExtraArgs `--model anthropic/claude-3-5-sonnet`.
func (a *OpenCodeAgent) SupportsVision() bool { return false }

// ModelInfo returns the minimal model identification this agent surfaces.
// The OpenCode CLI delegates model selection to the operator's
// `--model` flag; the agent itself only reports the binary path so
// callers can correlate logs back to which CLI install serviced them.
func (a *OpenCodeAgent) ModelInfo() ModelInfo {
	return ModelInfo{
		ID:       a.id,
		Provider: "opencode",
		Name:     "opencode-cli",
	}
}

// invocationError chains BOTH the OpenCode sentinel AND the original
// *exec.ExitError so callers may errors.Is the sentinel AND errors.As
// the ExitError from the same error value. Standard fmt.Errorf("%w")
// can wrap exactly one — chaining via this struct unlocks both.
type invocationError struct {
	op       string
	exitCode int
	stderr   string
	wrapped  error // the underlying *exec.ExitError (or other non-exit failure)
}

func (e *invocationError) Error() string {
	if e.stderr != "" {
		return fmt.Sprintf("%s: %s exit %d: %s", ErrOpenCodeInvocationFailed.Error(), e.op, e.exitCode, e.stderr)
	}
	if e.exitCode != 0 {
		return fmt.Sprintf("%s: %s exit %d", ErrOpenCodeInvocationFailed.Error(), e.op, e.exitCode)
	}
	return fmt.Sprintf("%s: %s: %v", ErrOpenCodeInvocationFailed.Error(), e.op, e.wrapped)
}

// Unwrap surfaces the underlying error (typically *exec.ExitError) so
// errors.As(err, **exec.ExitError) works on the wrapper.
func (e *invocationError) Unwrap() error { return e.wrapped }

// Is matches the public sentinel so errors.Is(err, ErrOpenCodeInvocationFailed)
// returns true even though the sentinel is not in the Unwrap chain.
func (e *invocationError) Is(target error) bool {
	return target == ErrOpenCodeInvocationFailed
}

// wrapExitError produces a sentinel-wrapped error that preserves the
// underlying *exec.ExitError (for callers that want to extract exit
// code / stderr) and adds the captured Stderr bytes to the message so
// operators see WHY the CLI failed without an extra Output() call.
func wrapExitError(err error, op string) error {
	wrap := &invocationError{op: op, wrapped: err}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		wrap.exitCode = exitErr.ExitCode()
		wrap.stderr = strings.TrimSpace(string(exitErr.Stderr))
	}
	return wrap
}

// Compile-time assertion: OpenCodeAgent satisfies the Agent contract.
var _ Agent = (*OpenCodeAgent)(nil)
