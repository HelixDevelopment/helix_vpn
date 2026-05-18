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

// ErrOpenCodeClientNotWired is returned by the OpenCode ClientBuilder
// until round-61+ wires the OpenCode CLI binary integration
// (anticipated transport: os/exec spawning `opencode` with stdin/stdout
// JSON-RPC, configured via cfg.BinaryPath).
var ErrOpenCodeClientNotWired = fmt.Errorf(
	"opencode agent: client SDK integration not wired in this round — " +
		"pool's Acquire will fail loudly until round-61+ wires the " +
		"OpenCode CLI binary integration via os/exec or HTTP-RPC")

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

// OpenCodeClientBuilder returns a ClientBuilder that surfaces
// ErrOpenCodeClientNotWired. The pool around it is fully real; only
// the per-call client materialisation is sentinel-stubbed.
func OpenCodeClientBuilder(_ *PoolConfig) ClientBuilder {
	return func(_ context.Context) (Agent, error) {
		return nil, ErrOpenCodeClientNotWired
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
