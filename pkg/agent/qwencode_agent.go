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

// Round-76 §11.4 forensic anchor — real QwenCode CLI client wiring.
// FIFTH AND FINAL builder in the LLMOrchestrator round-60 sentinel arc
// (rounds 64 + 66 + 69 + 71 + 76 = 5/5 — arc COMPLETE).
//
// This file converts the round-60 ErrQwenCodeClientNotWired "stub
// returns sentinel" surface into a real os/exec-spawned bridge to the
// `qwen` CLI binary (Alibaba's Qwen Code CLI tool — distributed via
// npm as `@qwen-code/qwen-code`, installed binary name is `qwen`).
// The architecture is a direct sibling of round-64's OpenCode wiring
// (opencode_agent.go), round-66's ClaudeCode wiring (claudecode_agent.go),
// round-69's Gemini wiring (gemini_agent.go), and round-71's Junie
// wiring (junie_agent.go) — Path A: reuse the package-private process-
// group helpers (setProcessGroup / killProcessGroup) shared by every
// CLI-bridge agent in this package so context cancellation reaps not
// just the direct `qwen` child but every grandchild it spawned (node
// runtime, MCP server processes, sh subshells, etc.).
//
// Transport pattern (verified via `qwen --help` 2026-05-18 against
// installed `qwen` v0.14.5):
//
//   qwen "<prompt>"                     → run in non-interactive one-
//                                         shot mode with the prompt as
//                                         the positional query argument
//                                         ("qwen [query..]" per CLI
//                                         usage line); prints model
//                                         response to stdout and exits.
//   qwen -p "<prompt>"                  → deprecated short-form flag
//                                         alias for the positional
//                                         delivery (`--prompt` long
//                                         form also accepted; CLI help
//                                         marks both as "deprecated:
//                                         Use the positional prompt
//                                         instead"). PromptFlag knob
//                                         supports operators that still
//                                         want explicit flag delivery.
//   qwen --version                      → version probe (health-check)
//
// The CLI delivers a one-shot prompt per process invocation; running
// `qwen` with no positional argument enters interactive mode (would
// block on stdin), so the agent ALWAYS supplies a prompt argument.
// Qwen Code supports two delivery modes — positional ("qwen <prompt>")
// and -p/--prompt flag ("qwen -p <prompt>"); operators can switch via
// cfg.PromptFlag ("" = positional default; "-p" or "--prompt" = flag
// form, the latter using =-separated long form to keep the prompt in
// one argv slot even when it contains spaces).
//
// Each Send invocation spawns a fresh `qwen <prompt>` subprocess (no
// persistent session), matching how the CLI is documented to be used
// non-interactively and avoiding leaking a long-lived background
// process when the pool is shut down. The CLI also accepts
// `--checkpointing` to follow up an existing session — this is
// available via cfg.ExtraArgs but the default flow is fresh-session
// per Send to preserve idempotency.
//
// API keys (Alibaba DashScope / Qwen-OAuth token via `qwen auth`,
// `OPENAI_API_KEY` for OpenAI-compatible endpoints, etc.), model
// selection (`-m <model>` / `--model <model>`), and any extra flags
// (`--sandbox`, `--approval-mode yolo`, `--system-prompt=…`, etc.) are
// passed through cfg.Env / cfg.ExtraArgs so this file never hardcodes
// credentials (CONST-042) and never assumes which Qwen Code auth
// backend the operator wants (CONST-046).
//
// Constitutional anchors: CONST-035 (anti-bluff covenant — every PASS
// carries runtime evidence), CONST-042 (no-secret-leak — env-key
// handling), CONST-050(A) (no-fakes-beyond-unit-tests — production
// code path never imports test mocks), Article XI §11.9 (end-user
// quality forensic anchor).

// DefaultQwenCodeBinary is the binary name resolved via $PATH when
// QwenCodeAgentConfig.Binary is empty. Matches the canonical Qwen
// Code CLI install (`qwen` on PATH; typically resolves under
// ~/.npm-global/bin/qwen for user installs of @qwen-code/qwen-code
// or /usr/local/bin/qwen for system-wide installs). NOTE: the CLI
// tool is named "Qwen Code" but the installed binary is `qwen`, not
// `qwen-code` — operators that have aliased to `qwen-code` can override
// via QwenCodeAgentConfig.Binary.
const DefaultQwenCodeBinary = "qwen"

// DefaultQwenCodePromptFlag is the flag prepended to the prompt for
// one-shot non-interactive Send invocations. Empty string means
// "positional argument" — `qwen <prompt>` — which is the canonical
// one-shot delivery per `qwen --help` ("qwen [query..]" usage line;
// `-p/--prompt` flag is marked deprecated in favour of positional).
// Operators that prefer the long-form `--prompt=<text>` can override
// via QwenCodeAgentConfig.PromptFlag.
const DefaultQwenCodePromptFlag = ""

// ErrQwenCodeBinaryNotFound is returned by NewQwenCodeAgent when the
// configured (or default) `qwen` binary cannot be located on $PATH.
//
// Distinct from round-60's ErrQwenCodeClientNotWired (which signalled
// "no implementation exists at all" — now narrowed to the nil-cfg
// backstop in builders.go) — this round-76 sentinel fires AFTER
// round-76 wired the real implementation but the binary is missing
// at runtime.
var ErrQwenCodeBinaryNotFound = errors.New(
	"qwen-code agent: binary not found on PATH — install Alibaba Qwen " +
		"Code CLI (`npm install -g @qwen-code/qwen-code` exposes `qwen` " +
		"on PATH; see https://github.com/QwenLM/qwen-code) or set " +
		"QwenCodeAgentConfig.Binary to an absolute path")

// ErrQwenCodeClientNotConfigured is returned by
// QwenCodeClientBuilderFromConfig when the supplied
// QwenCodeBuilderConfig is zero-value (no binary override, no extra
// args, no env). Round-60's ErrQwenCodeClientNotWired signalled
// "implementation not present"; round-76's
// ErrQwenCodeClientNotConfigured signals "implementation present but
// caller passed an empty config".
//
// Operators can still get a working agent by passing
// QwenCodeBuilderConfig{Binary: "qwen"} (or letting the default fire
// via the legacy QwenCodeClientBuilder PATH-fallback entry).
var ErrQwenCodeClientNotConfigured = errors.New(
	"qwen-code agent: QwenCodeBuilderConfig is zero-value — populate " +
		"Binary (or rely on PATH-resolved default `qwen`) and re-invoke")

// ErrQwenCodeInvocationFailed wraps any non-zero exit from
// `qwen <prompt>`. Callers may errors.Is on this sentinel to
// distinguish CLI failures from binary-not-found / context-cancel
// failures. The underlying *exec.ExitError (including its captured
// stderr) is preserved via Unwrap so callers may errors.As it for
// the exit code.
var ErrQwenCodeInvocationFailed = errors.New(
	"qwen-code agent: `qwen` exited non-zero")

// QwenCodeAgentConfig configures a QwenCodeAgent instance.
//
// Binary may be "" → use DefaultQwenCodeBinary resolved via
// exec.LookPath.
// PromptFlag selects how the prompt is delivered to the CLI:
//   - "" (default) → positional argument: `qwen <ExtraArgs> <prompt>`
//   - "-p" or "--prompt" or any non-empty value → flag form:
//     `qwen <ExtraArgs> --prompt=<prompt>` (using =-separated form so
//     the prompt is one argv slot even when it contains spaces).
//     NOTE: per `qwen --help`, both `-p` and `--prompt` are marked
//     deprecated in favour of positional delivery — operators that
//     supply PromptFlag are opting into the deprecated surface
//     deliberately.
//
// ExtraArgs are placed BEFORE the prompt-delivery slot in
// `qwen <ExtraArgs> [<PromptFlag>=]<message>`. Useful for `-m
// qwen-coder-plus`, `--model qwen2.5-coder-32b-instruct`, `--sandbox`,
// `--approval-mode yolo`, `--system-prompt=<text>`, `--checkpointing`,
// etc.
// WorkingDir defaults to the caller's CWD when empty.
// Env defaults to inheriting the parent process environment via
// exec.Cmd's default behaviour when nil — callers MUST source any
// provider credentials (Alibaba DashScope token, OPENAI_API_KEY for
// OpenAI-compatible endpoints, Qwen-OAuth tokens, etc.) into this
// slice rather than hardcoding them in source (CONST-042).
type QwenCodeAgentConfig struct {
	Binary     string
	PromptFlag string
	ExtraArgs  []string
	WorkingDir string
	Env        []string
	// IDOverride is used by tests to make Agent.ID deterministic; in
	// production it is left empty and a default
	// "qwen-code-<binary>" ID is generated.
	IDOverride string
}

// QwenCodeBuilderConfig is the shape supplied to
// QwenCodeClientBuilderFromConfig. It mirrors QwenCodeAgentConfig but
// is exported under a distinct name so future builder-only knobs
// (retries, per-request timeouts) can land without polluting the
// per-agent config.
type QwenCodeBuilderConfig struct {
	Binary     string
	PromptFlag string
	ExtraArgs  []string
	WorkingDir string
	Env        []string
}

// IsZero reports whether the QwenCodeBuilderConfig has no configured
// fields. Empty config triggers ErrQwenCodeClientNotConfigured at
// build time (callers must opt in to the PATH-resolved default via
// the legacy QwenCodeClientBuilder entrypoint).
func (c QwenCodeBuilderConfig) IsZero() bool {
	return c.Binary == "" &&
		c.PromptFlag == "" &&
		len(c.ExtraArgs) == 0 &&
		c.WorkingDir == "" &&
		len(c.Env) == 0
}

// QwenCodeAgent is a real os/exec-backed bridge to the `qwen` CLI.
//
// Each Send invocation spawns a fresh `qwen <prompt>` subprocess,
// captures stdout, and returns it as the Response.Content. The
// subprocess honors the parent context's deadline + cancellation via
// exec.CommandContext, AND a platform-specific process-group SIGKILL
// helper (setProcessGroup / killProcessGroup, shared with
// OpenCodeAgent + ClaudeCodeAgent + GeminiAgent + JunieAgent in
// {opencode_agent_unix,opencode_agent_windows}.go) so that the CLI's
// child processes (node runtime, MCP server subprocesses, sh
// subshells, etc.) are reaped along with the parent and
// SimpleAgentPool's Shutdown path cancels in-flight CLI calls cleanly.
//
// Concurrency: QwenCodeAgent is safe for concurrent Send calls — each
// invocation gets its own subprocess and the only shared mutable
// state (running flag) is guarded by mu.
//
// This type implements the full Agent interface declared in agent.go.
type QwenCodeAgent struct {
	id         string
	binary     string
	promptFlag string
	extraArgs  []string
	workingDir string
	env        []string

	mu      sync.Mutex
	running bool
}

// NewQwenCodeAgent constructs a QwenCodeAgent, validating that the
// configured (or default) binary exists on $PATH at construction
// time.
//
// Returns ErrQwenCodeBinaryNotFound if exec.LookPath fails. This is
// the fail-fast contract — callers see a real failure at pool-build
// time instead of a deferred "command not found" inside Acquire.
func NewQwenCodeAgent(cfg QwenCodeAgentConfig) (*QwenCodeAgent, error) {
	bin := cfg.Binary
	if bin == "" {
		bin = DefaultQwenCodeBinary
	}
	resolved, err := exec.LookPath(bin)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrQwenCodeBinaryNotFound, err)
	}
	id := cfg.IDOverride
	if id == "" {
		id = "qwen-code-" + resolved
	}
	return &QwenCodeAgent{
		id:         id,
		binary:     resolved,
		promptFlag: cfg.PromptFlag, // "" means positional delivery
		extraArgs:  append([]string(nil), cfg.ExtraArgs...),
		workingDir: cfg.WorkingDir,
		env:        append([]string(nil), cfg.Env...),
	}, nil
}

// ID returns the agent's unique identifier.
func (a *QwenCodeAgent) ID() string { return a.id }

// Name returns the provider name "qwen-code" — used by the pool's
// preferred-agent matcher and by callers that need to identify which
// CLI backend serviced a given request. The string matches the
// builder/pool naming used in multi_pool.go ("qwen-code").
func (a *QwenCodeAgent) Name() string { return "qwen-code" }

// Start marks the agent as running. The CLI itself is one-shot per
// Send call, so there is no persistent process to spawn here; Start
// only updates state so Health() reflects "ready". Idempotent.
func (a *QwenCodeAgent) Start(_ context.Context) error {
	a.mu.Lock()
	a.running = true
	a.mu.Unlock()
	return nil
}

// Stop marks the agent as not-running. There is no long-lived process
// to terminate; in-flight Send calls cancel through their own
// contexts. Idempotent.
func (a *QwenCodeAgent) Stop(_ context.Context) error {
	a.mu.Lock()
	a.running = false
	a.mu.Unlock()
	return nil
}

// IsRunning reports whether Start has been called since the last Stop.
func (a *QwenCodeAgent) IsRunning() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

// Health runs `qwen --version` as a lightweight probe. PASS = exit 0
// + non-empty stdout. Any failure populates HealthStatus.Error with
// the captured stderr so operators can see what broke.
func (a *QwenCodeAgent) Health(ctx context.Context) HealthStatus {
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
		status.Error = wrapQwenCodeExitError(err, "qwen --version")
		return status
	}
	if len(out) == 0 {
		status.Healthy = false
		status.Error = errors.New("qwen --version: empty stdout (unexpected for a healthy CLI)")
		return status
	}
	status.Healthy = true
	return status
}

// buildPromptArgs returns the argv slice for delivering a prompt to
// the Qwen Code CLI. When promptFlag is empty, the prompt is appended
// as a positional argument (`qwen <ExtraArgs> <prompt>`). Otherwise
// the =-separated long-flag form is used (`qwen <ExtraArgs>
// --prompt=<prompt>`) so the prompt is a single argv slot even when
// it contains spaces.
func (a *QwenCodeAgent) buildPromptArgs(prompt string) []string {
	args := make([]string, 0, len(a.extraArgs)+2)
	args = append(args, a.extraArgs...)
	if a.promptFlag == "" {
		args = append(args, prompt)
	} else {
		args = append(args, a.promptFlag+"="+prompt)
	}
	return args
}

// Send runs `qwen <ExtraArgs> <prompt>` (or
// `qwen <ExtraArgs> --prompt=<prompt>` when PromptFlag is non-empty)
// as a one-shot subprocess and returns the captured stdout as
// Response.Content. Honors ctx cancellation via exec.CommandContext
// AND the package-shared process-group SIGKILL helper
// (setProcessGroup / killProcessGroup) so that the CLI's child
// processes (node runtime, MCP server subprocesses, sh subshells,
// etc.) are reaped along with the parent.
//
// CONST-042: API keys MUST be supplied via a.env (sourced by the
// caller from os.Environ() or a secrets manager); this method never
// reads credentials from anywhere else.
func (a *QwenCodeAgent) Send(ctx context.Context, prompt string) (Response, error) {
	start := time.Now()
	args := a.buildPromptArgs(prompt)

	out, err := a.runCapture(ctx, args)
	latency := time.Since(start)

	if err != nil {
		// Propagate ctx.Err verbatim when the failure is a
		// cancellation — exec wraps it inside *exec.ExitError, but
		// callers expect errors.Is(err, context.Canceled /
		// context.DeadlineExceeded).
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Response{Latency: latency, Error: ctxErr}, ctxErr
		}
		op := "qwen"
		if a.promptFlag != "" {
			op = "qwen " + a.promptFlag
		}
		wrapped := wrapQwenCodeExitError(err, op)
		return Response{Latency: latency, Error: wrapped}, wrapped
	}

	return Response{
		Content: string(out),
		Latency: latency,
	}, nil
}

// runCapture configures and runs a command with process-group
// SIGKILL on ctx cancellation, returning stdout (matching
// exec.Cmd.Output() semantics). Uses the package-shared
// setProcessGroup / killProcessGroup helpers introduced in round 64
// (opencode_agent_{unix,windows}.go) — no new platform helper file
// is needed because those helpers already use generic names and the
// process-group reaping pattern is identical for every CLI bridge.
func (a *QwenCodeAgent) runCapture(ctx context.Context, args []string) ([]byte, error) {
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

// SendStream is not yet supported by the Qwen Code CLI client
// integration. Streaming requires Qwen Code CLI event-stream output
// (no documented stable flag on v0.14.5 — the CLI's ACP mode is an
// agent-to-agent protocol, not a model-response stream consumable by
// this bridge) — follow-up work for a later round. Until then this
// surfaces a typed error rather than silently buffering Send's
// response into a fake stream.
func (a *QwenCodeAgent) SendStream(_ context.Context, _ string) (<-chan StreamChunk, error) {
	return nil, errors.New("qwen-code agent: SendStream not yet wired (Qwen Code CLI v0.14.5 lacks a stable model-response event-stream flag — follow-up round)")
}

// SendWithAttachments runs `qwen --include-directories=<dir> <prompt>`
// when the CLI supports the `--include-directories` flag (Qwen Code
// inherits Gemini CLI's directory-include surface for its
// "@directory" context-priming convention). Each attachment.Path
// becomes one --include-directories= argv slot. Empty-Path
// attachments are skipped. Attachment.Name is currently ignored at
// the CLI surface (the CLI's directory-include flag takes path-only;
// per-file naming is reserved for future Qwen Code CLI revisions).
//
// NOTE: Qwen Code's documented surface for adding directories to the
// workspace is `--include-directories=<text>` (`qwen --help`:
// "Additional directories to include in the workspace.") — supplying
// multiple `--include-directories=` values is the canonical pattern;
// if a future Qwen Code CLI revision changes the flag name, the
// override path is cfg.ExtraArgs ["--include-directories=…"] + Send
// (skip SendWithAttachments).
func (a *QwenCodeAgent) SendWithAttachments(ctx context.Context, prompt string, attachments []Attachment) (Response, error) {
	start := time.Now()
	args := make([]string, 0, len(a.extraArgs)+2*len(attachments)+2)
	args = append(args, a.extraArgs...)
	for _, att := range attachments {
		if att.Path == "" {
			continue
		}
		args = append(args, "--include-directories="+att.Path)
	}
	if a.promptFlag == "" {
		args = append(args, prompt)
	} else {
		args = append(args, a.promptFlag+"="+prompt)
	}

	out, err := a.runCapture(ctx, args)
	latency := time.Since(start)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Response{Latency: latency, Error: ctxErr}, ctxErr
		}
		op := "qwen --include-directories"
		if a.promptFlag != "" {
			op = "qwen " + a.promptFlag + " --include-directories"
		}
		wrapped := wrapQwenCodeExitError(err, op)
		return Response{Latency: latency, Error: wrapped}, wrapped
	}
	return Response{Content: string(out), Latency: latency}, nil
}

// OutputDir returns the working directory the CLI subprocess runs in.
// When empty, the subprocess inherits the parent's CWD.
func (a *QwenCodeAgent) OutputDir() string { return a.workingDir }

// Capabilities reports the Qwen Code CLI's known capability surface.
// The CLI itself supports tool-use (file edit, shell, sandboxed
// execution via `--sandbox`, MCP server orchestration, ACP agent-to-
// agent protocol, checkpointing of file edits), and pluggable model
// backends (Alibaba Qwen-Coder series, OpenAI-compatible endpoints).
// Per-model context windows vary by backend; the conservative default
// reported here is the Qwen2.5-Coder-class 128k upper bound —
// operators that select Qwen3-Coder or larger models can extend at
// the model layer.
func (a *QwenCodeAgent) Capabilities() AgentCapabilities {
	return AgentCapabilities{
		Vision:    false, // depends on selected model; conservative default
		Streaming: false, // wired in a later round per SendStream comment
		ToolUse:   true,
		MaxTokens: 128000,
	}
}

// SupportsVision returns whether the agent (and its current model
// selection) supports vision. Conservative default: false. Operators
// can override after construction if they have selected a vision-
// capable model via cfg.ExtraArgs `--model qwen-vl-max`.
func (a *QwenCodeAgent) SupportsVision() bool { return false }

// ModelInfo returns the minimal model identification this agent
// surfaces. The Qwen Code CLI delegates model selection to the
// operator's `-m` / `--model` flag; the agent itself only reports the
// binary path so callers can correlate logs back to which CLI install
// serviced them.
func (a *QwenCodeAgent) ModelInfo() ModelInfo {
	return ModelInfo{
		ID:       a.id,
		Provider: "qwen-code",
		Name:     "qwen-code-cli",
	}
}

// qwenCodeInvocationError chains BOTH the QwenCode sentinel AND the
// original *exec.ExitError so callers may errors.Is the sentinel
// AND errors.As the ExitError from the same error value. Standard
// fmt.Errorf("%w") can wrap exactly one — chaining via this struct
// unlocks both. Mirrors OpenCode's invocationError, ClaudeCode's
// claudeCodeInvocationError, Gemini's geminiInvocationError, and
// Junie's junieInvocationError patterns with a distinct sentinel
// target so errors.Is(err, ErrOpenCodeInvocationFailed) /
// errors.Is(err, ErrClaudeCodeInvocationFailed) /
// errors.Is(err, ErrGeminiInvocationFailed) /
// errors.Is(err, ErrJunieInvocationFailed) return false for QwenCode
// failures and vice versa.
type qwenCodeInvocationError struct {
	op       string
	exitCode int
	stderr   string
	wrapped  error // the underlying *exec.ExitError (or other non-exit failure)
}

func (e *qwenCodeInvocationError) Error() string {
	if e.stderr != "" {
		return fmt.Sprintf("%s: %s exit %d: %s", ErrQwenCodeInvocationFailed.Error(), e.op, e.exitCode, e.stderr)
	}
	if e.exitCode != 0 {
		return fmt.Sprintf("%s: %s exit %d", ErrQwenCodeInvocationFailed.Error(), e.op, e.exitCode)
	}
	return fmt.Sprintf("%s: %s: %v", ErrQwenCodeInvocationFailed.Error(), e.op, e.wrapped)
}

// Unwrap surfaces the underlying error (typically *exec.ExitError) so
// errors.As(err, **exec.ExitError) works on the wrapper.
func (e *qwenCodeInvocationError) Unwrap() error { return e.wrapped }

// Is matches the public QwenCode sentinel so
// errors.Is(err, ErrQwenCodeInvocationFailed) returns true even though
// the sentinel is not in the Unwrap chain.
func (e *qwenCodeInvocationError) Is(target error) bool {
	return target == ErrQwenCodeInvocationFailed
}

// wrapQwenCodeExitError produces a sentinel-wrapped error that
// preserves the underlying *exec.ExitError (for callers that want to
// extract exit code / stderr) and adds the captured Stderr bytes to
// the message so operators see WHY the CLI failed without an extra
// Output() call.
func wrapQwenCodeExitError(err error, op string) error {
	wrap := &qwenCodeInvocationError{op: op, wrapped: err}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		wrap.exitCode = exitErr.ExitCode()
		wrap.stderr = strings.TrimSpace(string(exitErr.Stderr))
	}
	return wrap
}

// Compile-time assertion: QwenCodeAgent satisfies the Agent contract.
var _ Agent = (*QwenCodeAgent)(nil)
