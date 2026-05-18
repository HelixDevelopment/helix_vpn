// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"fmt"
)

// Round-60 §11.4 forensic anchor — per-provider ClientBuilder stubs.
//
// Each builder below returns a ClientBuilder closure that always
// surfaces a provider-specific "client SDK not wired" sentinel. The
// SimpleAgentPool that wraps the closure is REAL — it correctly
// manages capacity, available/in-use bookkeeping, blocking Acquire,
// and Shutdown. What is NOT yet wired is the bridge from the
// closure to the actual provider transport (CLI binary via os/exec
// for opencode/claude-code/junie, HTTP/SDK for gemini, HTTP/SDK
// for qwen-code). Each provider's wiring is a follow-up round
// (round-61+) so that one provider's SDK integration can land
// without re-touching every pool's plumbing.
//
// This pattern keeps the anti-bluff guarantee intact at every layer:
//   1. NewMultiProviderPool with a valid (non-nil) PoolConfig now
//      returns a real *MultiProviderPool whose .Acquire path goes
//      through the per-provider SimpleAgentPool;
//   2. SimpleAgentPool.Acquire on a fresh pool calls the injected
//      ClientBuilder, which returns the per-provider sentinel below;
//   3. SimpleAgentPool wraps that error with its own pool name and
//      bubbles it back to the caller — so the caller sees both the
//      pool that failed AND the precise provider-wiring gap.
//
// No silent fall-back path exists. No nil Agent is ever returned.
// Every failure surfaces as an errors.Is-checkable typed sentinel.
//
// Constitutional anchors: CONST-035 (anti-bluff covenant),
// CONST-050(A) (no-fakes-beyond-unit-tests; all stub returns are
// loud errors, not silent agents), Article XI §11.9.

// ErrOpenCodeClientNotWired was the round-60 §11.4 sentinel that
// signalled "OpenCode CLI binary integration not implemented in this
// repository". Round-64 §11.4 lands the real os/exec-based bridge to
// `opencode run` (see opencode_agent.go) — so this sentinel's meaning
// has been NARROWED, not removed.
//
// Round-60 semantics: "no implementation exists; the builder is a stub".
// Round-64 semantics: "implementation exists, but this code path took
// the legacy stub branch — caller invoked OpenCodeClientBuilder with
// nil PoolConfig (programmer error: factories should never propagate a
// nil cfg into the builder)".
//
// In the normal end-to-end path (NewOpenCodePool with non-nil
// PoolConfig → OpenCodeClientBuilder → NewOpenCodeAgent), the builder
// returns a real *OpenCodeAgent and this sentinel never fires. Tests
// that explicitly pass a nil PoolConfig into OpenCodeClientBuilder
// continue to receive this sentinel for backward compatibility.
//
// See ErrOpenCodeClientNotConfigured (opencode_agent.go) for the
// distinct "config present but zero-value" case landed in round 64.
var ErrOpenCodeClientNotWired = fmt.Errorf(
	"opencode agent: ClientBuilder received nil PoolConfig — round-64 " +
		"wired the real os/exec bridge to `opencode run`; this sentinel " +
		"now narrows to the nil-cfg backstop path (caller programmer error)")

// ErrClaudeCodeClientNotWired is returned by the Claude-Code ClientBuilder
// until round-61+ wires the Claude Code CLI binary integration
// (anticipated transport: os/exec spawning `claude` CLI with the
// Anthropic Messages API session protocol).
var ErrClaudeCodeClientNotWired = fmt.Errorf(
	"claude-code agent: client SDK integration not wired in this round — " +
		"pool's Acquire will fail loudly until round-61+ wires the " +
		"Claude Code CLI binary integration via os/exec spawning `claude`")

// ErrGeminiClientNotWired is returned by the Gemini ClientBuilder
// until round-61+ wires the Gemini provider integration
// (anticipated transport: HTTP to Google AI Studio / Vertex AI
// generateContent endpoint with cfg.APIKey).
var ErrGeminiClientNotWired = fmt.Errorf(
	"gemini agent: client SDK integration not wired in this round — " +
		"pool's Acquire will fail loudly until round-61+ wires the " +
		"Gemini HTTP transport to generativelanguage.googleapis.com")

// ErrJunieClientNotWired is returned by the Junie ClientBuilder
// until round-61+ wires the Junie (JetBrains AI Assistant CLI) binary
// integration (anticipated transport: os/exec spawning `junie` CLI
// configured via cfg.BinaryPath).
var ErrJunieClientNotWired = fmt.Errorf(
	"junie agent: client SDK integration not wired in this round — " +
		"pool's Acquire will fail loudly until round-61+ wires the " +
		"Junie (JetBrains AI Assistant) CLI integration via os/exec")

// ErrQwenCodeClientNotWired is returned by the Qwen-Code ClientBuilder
// until round-61+ wires the Qwen-Code SDK integration
// (anticipated transport: HTTP to Alibaba DashScope generation endpoint
// with cfg.APIKey, or local Qwen-Code CLI via os/exec).
var ErrQwenCodeClientNotWired = fmt.Errorf(
	"qwen-code agent: client SDK integration not wired in this round — " +
		"pool's Acquire will fail loudly until round-61+ wires the " +
		"Qwen-Code SDK HTTP transport to dashscope.aliyuncs.com")

// OpenCodeClientBuilder returns a ClientBuilder that constructs a real
// *OpenCodeAgent on each invocation (round-64 §11.4 wiring).
//
// The supplied PoolConfig contributes the BinaryPath (→ cfg.Binary)
// and the runtime env (sourced from the caller via cfg's lifecycle —
// the legacy PoolConfig struct does not yet carry an explicit Env
// field, so OpenCodeAgent inherits os.Environ() through Go's default
// exec behaviour when cfg.Env stays empty here).
//
// Backstop paths:
//   - nil PoolConfig → ErrOpenCodeClientNotWired (round-60 sentinel,
//     narrowed to "programmer error: factory propagated nil cfg").
//   - cfg present but the would-be OpenCodeBuilderConfig is zero-value
//     (no BinaryPath, no extra args) → caller still gets a working
//     agent IF `opencode` is on $PATH (DefaultOpenCodeBinary fallback).
//     The distinct ErrOpenCodeClientNotConfigured sentinel fires only
//     for the explicit OpenCodeClientBuilderFromConfig() entrypoint.
//   - `opencode` binary missing from $PATH → ErrOpenCodeBinaryNotFound
//     surfaces from NewOpenCodeAgent, gets wrapped by
//     SimpleAgentPool.Acquire, and reaches the caller errors.Is-checkable.
//
// Constitutional anchors: CONST-035 (real CLI invocation, no
// simulation), CONST-042 (env-sourced credentials), CONST-050(A)
// (production-side wiring uses no test mocks).
func OpenCodeClientBuilder(cfg *PoolConfig) ClientBuilder {
	if cfg == nil {
		return func(_ context.Context) (Agent, error) {
			return nil, ErrOpenCodeClientNotWired
		}
	}
	agentCfg := OpenCodeAgentConfig{
		Binary: cfg.BinaryPath,
	}
	return func(_ context.Context) (Agent, error) {
		a, err := NewOpenCodeAgent(agentCfg)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
}

// OpenCodeClientBuilderFromConfig is the round-64 strict-config
// entrypoint: it requires a non-zero OpenCodeBuilderConfig and
// surfaces ErrOpenCodeClientNotConfigured otherwise. Use this when
// the caller wants the "no implicit PATH fallback" contract.
func OpenCodeClientBuilderFromConfig(cfg OpenCodeBuilderConfig) ClientBuilder {
	if cfg.IsZero() {
		return func(_ context.Context) (Agent, error) {
			return nil, ErrOpenCodeClientNotConfigured
		}
	}
	agentCfg := OpenCodeAgentConfig{
		Binary:     cfg.Binary,
		ExtraArgs:  cfg.ExtraArgs,
		WorkingDir: cfg.WorkingDir,
		Env:        cfg.Env,
	}
	return func(_ context.Context) (Agent, error) {
		a, err := NewOpenCodeAgent(agentCfg)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
}

// ClaudeCodeClientBuilder returns a ClientBuilder that surfaces
// ErrClaudeCodeClientNotWired.
func ClaudeCodeClientBuilder(_ *PoolConfig) ClientBuilder {
	return func(_ context.Context) (Agent, error) {
		return nil, ErrClaudeCodeClientNotWired
	}
}

// GeminiClientBuilder returns a ClientBuilder that surfaces
// ErrGeminiClientNotWired.
func GeminiClientBuilder(_ *PoolConfig) ClientBuilder {
	return func(_ context.Context) (Agent, error) {
		return nil, ErrGeminiClientNotWired
	}
}

// JunieClientBuilder returns a ClientBuilder that surfaces
// ErrJunieClientNotWired.
func JunieClientBuilder(_ *PoolConfig) ClientBuilder {
	return func(_ context.Context) (Agent, error) {
		return nil, ErrJunieClientNotWired
	}
}

// QwenCodeClientBuilder returns a ClientBuilder that surfaces
// ErrQwenCodeClientNotWired.
func QwenCodeClientBuilder(_ *PoolConfig) ClientBuilder {
	return func(_ context.Context) (Agent, error) {
		return nil, ErrQwenCodeClientNotWired
	}
}
