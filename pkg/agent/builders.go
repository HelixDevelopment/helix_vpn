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

// ErrClaudeCodeClientNotWired was the round-60 §11.4 sentinel that
// signalled "Claude Code CLI binary integration not implemented in this
// repository". Round-66 §11.4 lands the real os/exec-based bridge to
// `claude --print` (see claudecode_agent.go) — so this sentinel's
// meaning has been NARROWED, not removed.
//
// Round-60 semantics: "no implementation exists; the builder is a stub".
// Round-66 semantics: "implementation exists, but this code path took
// the legacy stub branch — caller invoked ClaudeCodeClientBuilder with
// nil PoolConfig (programmer error: factories should never propagate a
// nil cfg into the builder)".
//
// In the normal end-to-end path (NewClaudeCodePool with non-nil
// PoolConfig → ClaudeCodeClientBuilder → NewClaudeCodeAgent), the
// builder returns a real *ClaudeCodeAgent and this sentinel never
// fires. Tests that explicitly pass a nil PoolConfig into
// ClaudeCodeClientBuilder continue to receive this sentinel for
// backward compatibility.
//
// See ErrClaudeCodeClientNotConfigured (claudecode_agent.go) for the
// distinct "config present but zero-value" case landed in round 66.
var ErrClaudeCodeClientNotWired = fmt.Errorf(
	"claude-code agent: ClientBuilder received nil PoolConfig — round-66 " +
		"wired the real os/exec bridge to `claude --print`; this sentinel " +
		"now narrows to the nil-cfg backstop path (caller programmer error)")

// ErrGeminiClientNotWired was the round-60 §11.4 sentinel that
// signalled "Gemini CLI binary integration not implemented in this
// repository". Round-69 §11.4 lands the real os/exec-based bridge to
// `gemini -p` (see gemini_agent.go) — so this sentinel's meaning has
// been NARROWED, not removed.
//
// Round-60 semantics: "no implementation exists; the builder is a stub".
// Round-69 semantics: "implementation exists, but this code path took
// the legacy stub branch — caller invoked GeminiClientBuilder with
// nil PoolConfig (programmer error: factories should never propagate a
// nil cfg into the builder)".
//
// In the normal end-to-end path (NewGeminiPool with non-nil PoolConfig
// → GeminiClientBuilder → NewGeminiAgent), the builder returns a
// real *GeminiAgent and this sentinel never fires. Tests that
// explicitly pass a nil PoolConfig into GeminiClientBuilder continue
// to receive this sentinel for backward compatibility.
//
// See ErrGeminiClientNotConfigured (gemini_agent.go) for the
// distinct "config present but zero-value" case landed in round 69.
var ErrGeminiClientNotWired = fmt.Errorf(
	"gemini agent: ClientBuilder received nil PoolConfig — round-69 " +
		"wired the real os/exec bridge to `gemini -p`; this sentinel " +
		"now narrows to the nil-cfg backstop path (caller programmer error)")

// ErrJunieClientNotWired was the round-60 §11.4 sentinel that
// signalled "Junie CLI binary integration not implemented in this
// repository". Round-71 §11.4 lands the real os/exec-based bridge to
// `junie <prompt>` (see junie_agent.go) — so this sentinel's meaning
// has been NARROWED, not removed.
//
// Round-60 semantics: "no implementation exists; the builder is a stub".
// Round-71 semantics: "implementation exists, but this code path took
// the legacy stub branch — caller invoked JunieClientBuilder with
// nil PoolConfig (programmer error: factories should never propagate
// a nil cfg into the builder)".
//
// In the normal end-to-end path (NewJuniePool with non-nil PoolConfig
// → JunieClientBuilder → NewJunieAgent), the builder returns a real
// *JunieAgent and this sentinel never fires. Tests that explicitly
// pass a nil PoolConfig into JunieClientBuilder continue to receive
// this sentinel for backward compatibility.
//
// See ErrJunieClientNotConfigured (junie_agent.go) for the distinct
// "config present but zero-value" case landed in round 71.
var ErrJunieClientNotWired = fmt.Errorf(
	"junie agent: ClientBuilder received nil PoolConfig — round-71 " +
		"wired the real os/exec bridge to `junie <prompt>`; this sentinel " +
		"now narrows to the nil-cfg backstop path (caller programmer error)")

// ErrQwenCodeClientNotWired was the round-60 §11.4 sentinel that
// signalled "Qwen Code CLI binary integration not implemented in this
// repository". Round-76 §11.4 lands the real os/exec-based bridge to
// `qwen <prompt>` (see qwencode_agent.go) — so this sentinel's
// meaning has been NARROWED, not removed.
//
// Round-60 semantics: "no implementation exists; the builder is a stub".
// Round-76 semantics: "implementation exists, but this code path took
// the legacy stub branch — caller invoked QwenCodeClientBuilder with
// nil PoolConfig (programmer error: factories should never propagate
// a nil cfg into the builder)".
//
// In the normal end-to-end path (NewQwenCodePool with non-nil
// PoolConfig → QwenCodeClientBuilder → NewQwenCodeAgent), the
// builder returns a real *QwenCodeAgent and this sentinel never
// fires. Tests that explicitly pass a nil PoolConfig into
// QwenCodeClientBuilder continue to receive this sentinel for
// backward compatibility.
//
// See ErrQwenCodeClientNotConfigured (qwencode_agent.go) for the
// distinct "config present but zero-value" case landed in round 76.
//
// Round-76 also marks the COMPLETION of the LLMOrchestrator builder
// arc: rounds 64 (OpenCode) + 66 (ClaudeCode) + 69 (Gemini) +
// 71 (Junie) + 76 (QwenCode) = 5/5 builders wired.
var ErrQwenCodeClientNotWired = fmt.Errorf(
	"qwen-code agent: ClientBuilder received nil PoolConfig — round-76 " +
		"wired the real os/exec bridge to `qwen <prompt>`; this sentinel " +
		"now narrows to the nil-cfg backstop path (caller programmer error)")

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

// ClaudeCodeClientBuilder returns a ClientBuilder that constructs a real
// *ClaudeCodeAgent on each invocation (round-66 §11.4 wiring).
//
// The supplied PoolConfig contributes the BinaryPath (→ cfg.Binary)
// and the runtime env (sourced from the caller via cfg's lifecycle —
// the legacy PoolConfig struct does not yet carry an explicit Env
// field, so ClaudeCodeAgent inherits the parent process environment
// through Go's default exec behaviour when cfg.Env stays empty here).
//
// Backstop paths:
//   - nil PoolConfig → ErrClaudeCodeClientNotWired (round-60 sentinel,
//     narrowed to "programmer error: factory propagated nil cfg").
//   - cfg present but the would-be ClaudeCodeBuilderConfig is
//     zero-value (no BinaryPath, no extra args) → caller still gets
//     a working agent IF `claude` is on $PATH
//     (DefaultClaudeCodeBinary fallback). The distinct
//     ErrClaudeCodeClientNotConfigured sentinel fires only for the
//     explicit ClaudeCodeClientBuilderFromConfig() entrypoint.
//   - `claude` binary missing from $PATH → ErrClaudeCodeBinaryNotFound
//     surfaces from NewClaudeCodeAgent, gets wrapped by
//     SimpleAgentPool.Acquire, and reaches the caller errors.Is-
//     checkable.
//
// Constitutional anchors: CONST-035 (real CLI invocation, no
// simulation), CONST-042 (env-sourced credentials), CONST-050(A)
// (production-side wiring uses no test mocks).
func ClaudeCodeClientBuilder(cfg *PoolConfig) ClientBuilder {
	if cfg == nil {
		return func(_ context.Context) (Agent, error) {
			return nil, ErrClaudeCodeClientNotWired
		}
	}
	agentCfg := ClaudeCodeAgentConfig{
		Binary: cfg.BinaryPath,
	}
	return func(_ context.Context) (Agent, error) {
		a, err := NewClaudeCodeAgent(agentCfg)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
}

// ClaudeCodeClientBuilderFromConfig is the round-66 strict-config
// entrypoint: it requires a non-zero ClaudeCodeBuilderConfig and
// surfaces ErrClaudeCodeClientNotConfigured otherwise. Use this when
// the caller wants the "no implicit PATH fallback" contract.
func ClaudeCodeClientBuilderFromConfig(cfg ClaudeCodeBuilderConfig) ClientBuilder {
	if cfg.IsZero() {
		return func(_ context.Context) (Agent, error) {
			return nil, ErrClaudeCodeClientNotConfigured
		}
	}
	agentCfg := ClaudeCodeAgentConfig{
		Binary:     cfg.Binary,
		PromptFlag: cfg.PromptFlag,
		ExtraArgs:  cfg.ExtraArgs,
		WorkingDir: cfg.WorkingDir,
		Env:        cfg.Env,
	}
	return func(_ context.Context) (Agent, error) {
		a, err := NewClaudeCodeAgent(agentCfg)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
}

// GeminiClientBuilder returns a ClientBuilder that constructs a real
// *GeminiAgent on each invocation (round-69 §11.4 wiring).
//
// The supplied PoolConfig contributes the BinaryPath (→ cfg.Binary)
// and the runtime env (sourced from the caller via cfg's lifecycle —
// the legacy PoolConfig struct does not yet carry an explicit Env
// field, so GeminiAgent inherits the parent process environment
// through Go's default exec behaviour when cfg.Env stays empty here).
//
// Backstop paths:
//   - nil PoolConfig → ErrGeminiClientNotWired (round-60 sentinel,
//     narrowed to "programmer error: factory propagated nil cfg").
//   - cfg present but the would-be GeminiBuilderConfig is zero-value
//     (no BinaryPath, no extra args) → caller still gets a working
//     agent IF `gemini` is on $PATH (DefaultGeminiBinary fallback).
//     The distinct ErrGeminiClientNotConfigured sentinel fires only
//     for the explicit GeminiClientBuilderFromConfig() entrypoint.
//   - `gemini` binary missing from $PATH → ErrGeminiBinaryNotFound
//     surfaces from NewGeminiAgent, gets wrapped by
//     SimpleAgentPool.Acquire, and reaches the caller errors.Is-
//     checkable.
//
// Constitutional anchors: CONST-035 (real CLI invocation, no
// simulation), CONST-042 (env-sourced credentials), CONST-050(A)
// (production-side wiring uses no test mocks).
func GeminiClientBuilder(cfg *PoolConfig) ClientBuilder {
	if cfg == nil {
		return func(_ context.Context) (Agent, error) {
			return nil, ErrGeminiClientNotWired
		}
	}
	agentCfg := GeminiAgentConfig{
		Binary: cfg.BinaryPath,
	}
	return func(_ context.Context) (Agent, error) {
		a, err := NewGeminiAgent(agentCfg)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
}

// GeminiClientBuilderFromConfig is the round-69 strict-config
// entrypoint: it requires a non-zero GeminiBuilderConfig and surfaces
// ErrGeminiClientNotConfigured otherwise. Use this when the caller
// wants the "no implicit PATH fallback" contract.
func GeminiClientBuilderFromConfig(cfg GeminiBuilderConfig) ClientBuilder {
	if cfg.IsZero() {
		return func(_ context.Context) (Agent, error) {
			return nil, ErrGeminiClientNotConfigured
		}
	}
	agentCfg := GeminiAgentConfig{
		Binary:     cfg.Binary,
		PromptFlag: cfg.PromptFlag,
		ExtraArgs:  cfg.ExtraArgs,
		WorkingDir: cfg.WorkingDir,
		Env:        cfg.Env,
	}
	return func(_ context.Context) (Agent, error) {
		a, err := NewGeminiAgent(agentCfg)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
}

// JunieClientBuilder returns a ClientBuilder that constructs a real
// *JunieAgent on each invocation (round-71 §11.4 wiring).
//
// The supplied PoolConfig contributes the BinaryPath (→ cfg.Binary)
// and the runtime env (sourced from the caller via cfg's lifecycle —
// the legacy PoolConfig struct does not yet carry an explicit Env
// field, so JunieAgent inherits the parent process environment
// through Go's default exec behaviour when cfg.Env stays empty here).
//
// Backstop paths:
//   - nil PoolConfig → ErrJunieClientNotWired (round-60 sentinel,
//     narrowed to "programmer error: factory propagated nil cfg").
//   - cfg present but the would-be JunieBuilderConfig is zero-value
//     (no BinaryPath, no extra args) → caller still gets a working
//     agent IF `junie` is on $PATH (DefaultJunieBinary fallback).
//     The distinct ErrJunieClientNotConfigured sentinel fires only
//     for the explicit JunieClientBuilderFromConfig() entrypoint.
//   - `junie` binary missing from $PATH → ErrJunieBinaryNotFound
//     surfaces from NewJunieAgent, gets wrapped by
//     SimpleAgentPool.Acquire, and reaches the caller errors.Is-
//     checkable.
//
// Constitutional anchors: CONST-035 (real CLI invocation, no
// simulation), CONST-042 (env-sourced credentials), CONST-050(A)
// (production-side wiring uses no test mocks).
func JunieClientBuilder(cfg *PoolConfig) ClientBuilder {
	if cfg == nil {
		return func(_ context.Context) (Agent, error) {
			return nil, ErrJunieClientNotWired
		}
	}
	agentCfg := JunieAgentConfig{
		Binary: cfg.BinaryPath,
	}
	return func(_ context.Context) (Agent, error) {
		a, err := NewJunieAgent(agentCfg)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
}

// JunieClientBuilderFromConfig is the round-71 strict-config
// entrypoint: it requires a non-zero JunieBuilderConfig and surfaces
// ErrJunieClientNotConfigured otherwise. Use this when the caller
// wants the "no implicit PATH fallback" contract.
func JunieClientBuilderFromConfig(cfg JunieBuilderConfig) ClientBuilder {
	if cfg.IsZero() {
		return func(_ context.Context) (Agent, error) {
			return nil, ErrJunieClientNotConfigured
		}
	}
	agentCfg := JunieAgentConfig{
		Binary:     cfg.Binary,
		PromptFlag: cfg.PromptFlag,
		ExtraArgs:  cfg.ExtraArgs,
		WorkingDir: cfg.WorkingDir,
		Env:        cfg.Env,
	}
	return func(_ context.Context) (Agent, error) {
		a, err := NewJunieAgent(agentCfg)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
}

// QwenCodeClientBuilder returns a ClientBuilder that constructs a real
// *QwenCodeAgent on each invocation (round-76 §11.4 wiring — FINAL
// builder in the LLMOrchestrator round-60 sentinel arc; rounds
// 64+66+69+71+76 = 5/5 builders COMPLETE).
//
// The supplied PoolConfig contributes the BinaryPath (→ cfg.Binary)
// and the runtime env (sourced from the caller via cfg's lifecycle —
// the legacy PoolConfig struct does not yet carry an explicit Env
// field, so QwenCodeAgent inherits the parent process environment
// through Go's default exec behaviour when cfg.Env stays empty here).
//
// Backstop paths:
//   - nil PoolConfig → ErrQwenCodeClientNotWired (round-60 sentinel,
//     narrowed to "programmer error: factory propagated nil cfg").
//   - cfg present but the would-be QwenCodeBuilderConfig is zero-value
//     (no BinaryPath, no extra args) → caller still gets a working
//     agent IF `qwen` is on $PATH (DefaultQwenCodeBinary fallback).
//     The distinct ErrQwenCodeClientNotConfigured sentinel fires only
//     for the explicit QwenCodeClientBuilderFromConfig() entrypoint.
//   - `qwen` binary missing from $PATH → ErrQwenCodeBinaryNotFound
//     surfaces from NewQwenCodeAgent, gets wrapped by
//     SimpleAgentPool.Acquire, and reaches the caller errors.Is-
//     checkable.
//
// Constitutional anchors: CONST-035 (real CLI invocation, no
// simulation), CONST-042 (env-sourced credentials), CONST-050(A)
// (production-side wiring uses no test mocks).
func QwenCodeClientBuilder(cfg *PoolConfig) ClientBuilder {
	if cfg == nil {
		return func(_ context.Context) (Agent, error) {
			return nil, ErrQwenCodeClientNotWired
		}
	}
	agentCfg := QwenCodeAgentConfig{
		Binary: cfg.BinaryPath,
	}
	return func(_ context.Context) (Agent, error) {
		a, err := NewQwenCodeAgent(agentCfg)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
}

// QwenCodeClientBuilderFromConfig is the round-76 strict-config
// entrypoint: it requires a non-zero QwenCodeBuilderConfig and
// surfaces ErrQwenCodeClientNotConfigured otherwise. Use this when
// the caller wants the "no implicit PATH fallback" contract.
func QwenCodeClientBuilderFromConfig(cfg QwenCodeBuilderConfig) ClientBuilder {
	if cfg.IsZero() {
		return func(_ context.Context) (Agent, error) {
			return nil, ErrQwenCodeClientNotConfigured
		}
	}
	agentCfg := QwenCodeAgentConfig{
		Binary:     cfg.Binary,
		PromptFlag: cfg.PromptFlag,
		ExtraArgs:  cfg.ExtraArgs,
		WorkingDir: cfg.WorkingDir,
		Env:        cfg.Env,
	}
	return func(_ context.Context) (Agent, error) {
		a, err := NewQwenCodeAgent(agentCfg)
		if err != nil {
			return nil, err
		}
		return a, nil
	}
}
