// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------
// SimpleAgentPool unit tests (round-60 §11.4 — concrete pool struct).
// Each test exercises a real SimpleAgentPool through real Acquire /
// Release / Shutdown calls with a unit-test ClientBuilder; nothing is
// stubbed beyond the in-test mockAgent + builder.
// ---------------------------------------------------------------------

// makeMockBuilder returns a ClientBuilder that yields a fresh mockAgent
// per invocation with deterministic IDs ("<name>-<n>"). The returned
// counter pointer lets tests assert how many times the builder ran.
func makeMockBuilder(name string) (ClientBuilder, *atomic.Int64) {
	var counter atomic.Int64
	b := func(_ context.Context) (Agent, error) {
		n := counter.Add(1)
		return newMockAgent(fmt.Sprintf("%s-%d", name, n), name), nil
	}
	return b, &counter
}

func TestSimpleAgentPool_Acquire_BuildsAndReturns(t *testing.T) {
	builder, calls := makeMockBuilder("opencode")
	pool := NewSimpleAgentPool("opencode", 2, builder)

	a, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err != nil {
		t.Fatalf("Acquire: unexpected error: %v", err)
	}
	if a == nil {
		t.Fatal("Acquire returned nil agent — round-28/60 §11.4 CONTRACT-bluff regression")
	}
	if got := a.Name(); got != "opencode" {
		t.Errorf("agent.Name() = %q, want %q", got, "opencode")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("builder call count = %d, want 1", got)
	}
	if pool.InUse() != 1 {
		t.Errorf("InUse() = %d, want 1", pool.InUse())
	}
	if pool.Size() != 2 {
		t.Errorf("Size() = %d, want 2", pool.Size())
	}
}

func TestSimpleAgentPool_Release_MakesAvailable(t *testing.T) {
	builder, _ := makeMockBuilder("gemini")
	pool := NewSimpleAgentPool("gemini", 1, builder)

	a, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	pool.Release(a)

	if got := len(pool.Available()); got != 1 {
		t.Errorf("Available() len = %d, want 1", got)
	}
	if got := pool.InUse(); got != 0 {
		t.Errorf("InUse() = %d, want 0", got)
	}

	// Re-acquire should hand back the same agent (no second build).
	a2, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err != nil {
		t.Fatalf("second Acquire: %v", err)
	}
	if a2.ID() != a.ID() {
		t.Errorf("re-acquired agent ID = %q, want %q", a2.ID(), a.ID())
	}
}

func TestSimpleAgentPool_Acquire_BlocksWhenExhausted(t *testing.T) {
	builder, _ := makeMockBuilder("claude-code")
	pool := NewSimpleAgentPool("claude-code", 1, builder)

	first, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}

	// Second Acquire MUST block until Release.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct {
		a   Agent
		err error
	}, 1)
	go func() {
		a, err := pool.Acquire(ctx, AgentRequirements{})
		done <- struct {
			a   Agent
			err error
		}{a, err}
	}()

	// Verify it really is blocked: nothing arrives within 100ms.
	select {
	case r := <-done:
		t.Fatalf("second Acquire returned without blocking — agent=%v err=%v", r.a, r.err)
	case <-time.After(100 * time.Millisecond):
	}

	// Release the first agent; the second Acquire MUST now succeed.
	pool.Release(first)
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("second Acquire after Release: %v", r.err)
		}
		if r.a == nil {
			t.Fatal("second Acquire returned nil agent after Release")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second Acquire did not unblock within 2s of Release")
	}
}

func TestSimpleAgentPool_Acquire_RespectsContextCancel(t *testing.T) {
	builder, _ := makeMockBuilder("junie")
	pool := NewSimpleAgentPool("junie", 1, builder)

	// Saturate capacity.
	if _, err := pool.Acquire(context.Background(), AgentRequirements{}); err != nil {
		t.Fatalf("first Acquire: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := pool.Acquire(ctx, AgentRequirements{})
		done <- err
	}()

	// Confirm blocking.
	select {
	case err := <-done:
		t.Fatalf("Acquire returned before cancellation: err=%v", err)
	case <-time.After(50 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("Acquire after cancel returned err = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Acquire did not unblock after context cancellation within 2s")
	}
}

func TestSimpleAgentPool_BuilderError_PropagatesFromAcquire(t *testing.T) {
	wantErr := errors.New("synthetic builder failure")
	builder := func(_ context.Context) (Agent, error) {
		return nil, wantErr
	}
	pool := NewSimpleAgentPool("qwen-code", 2, builder)

	a, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err == nil {
		t.Fatal("Acquire returned nil error despite builder failure — CONTRACT-bluff regression")
	}
	if a != nil {
		t.Errorf("Acquire returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("Acquire: errors.Is(err, wantErr) = false; err = %v", err)
	}
}

func TestSimpleAgentPool_BuilderNilAgent_NoError_StillFails(t *testing.T) {
	// A misbehaving builder that returns (nil, nil) MUST be caught —
	// silently handing out nil would resurrect the round-28 bluff.
	builder := func(_ context.Context) (Agent, error) { return nil, nil }
	pool := NewSimpleAgentPool("gemini", 1, builder)

	a, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err == nil {
		t.Fatal("Acquire returned nil error for nil agent — round-28 CONTRACT-bluff regression")
	}
	if a != nil {
		t.Errorf("Acquire returned non-nil agent alongside error: %v", a)
	}
}

func TestSimpleAgentPool_Close_ReleasesAll(t *testing.T) {
	builder, _ := makeMockBuilder("opencode")
	pool := NewSimpleAgentPool("opencode", 2, builder)

	a1, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	// Start the agent so Shutdown actually exercises the Stop path.
	if err := a1.Start(context.Background()); err != nil {
		t.Fatalf("agent.Start: %v", err)
	}

	if err := pool.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}

	if _, err := pool.Acquire(context.Background(), AgentRequirements{}); !errors.Is(err, ErrSimpleAgentPoolClosed) {
		t.Errorf("Acquire after Shutdown: err = %v, want ErrSimpleAgentPoolClosed", err)
	}
	if got := pool.InUse(); got != 0 {
		t.Errorf("InUse after Shutdown = %d, want 0", got)
	}
	if a1.IsRunning() {
		t.Error("agent still running after Shutdown — Stop was not called")
	}
	// Shutdown is idempotent.
	if err := pool.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown: unexpected err = %v", err)
	}
}

func TestSimpleAgentPool_Register_AvoidsBuilder(t *testing.T) {
	builder, calls := makeMockBuilder("opencode")
	pool := NewSimpleAgentPool("opencode", 2, builder)

	a := newMockAgent("pre-registered-1", "opencode")
	if err := pool.Register(a); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, err := pool.Acquire(context.Background(), AgentRequirements{})
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if got.ID() != a.ID() {
		t.Errorf("Acquire returned ID = %q, want pre-registered %q", got.ID(), a.ID())
	}
	if n := calls.Load(); n != 0 {
		t.Errorf("builder call count = %d, want 0 (pre-registered agent should win)", n)
	}
}

func TestSimpleAgentPool_ConcurrentAcquireRelease_NoRace(t *testing.T) {
	builder, _ := makeMockBuilder("opencode")
	pool := NewSimpleAgentPool("opencode", 4, builder)

	var wg sync.WaitGroup
	const goroutines = 16
	const cycles = 25

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < cycles; j++ {
				a, err := pool.Acquire(context.Background(), AgentRequirements{})
				if err != nil {
					t.Errorf("Acquire: %v", err)
					return
				}
				if a == nil {
					t.Error("Acquire returned nil agent under concurrent load")
					return
				}
				pool.Release(a)
			}
		}()
	}
	wg.Wait()
	if pool.InUse() != 0 {
		t.Errorf("InUse after cycle = %d, want 0", pool.InUse())
	}
}

// ---------------------------------------------------------------------
// Per-provider ClientBuilder sentinel tests (round-60 §11.4).
// Each builder MUST return a closure that yields its provider-specific
// "client SDK not wired" sentinel. The five sentinels MUST be distinct
// so errors.Is can disambiguate.
// ---------------------------------------------------------------------

func TestOpenCodeClientBuilder_NotWired_ReturnsSentinel(t *testing.T) {
	b := OpenCodeClientBuilder(&PoolConfig{Size: 1, BinaryPath: "/nonexistent/opencode"})
	a, err := b(context.Background())
	if err == nil {
		t.Fatal("OpenCodeClientBuilder closure returned nil error — round-60 §11.4 sentinel regression")
	}
	if a != nil {
		t.Errorf("OpenCodeClientBuilder returned non-nil agent alongside error: %v", a)
	}
	if !errors.Is(err, ErrOpenCodeClientNotWired) {
		t.Errorf("errors.Is(err, ErrOpenCodeClientNotWired) = false; err = %v", err)
	}
}

func TestClaudeCodeClientBuilder_NotWired_ReturnsSentinel(t *testing.T) {
	b := ClaudeCodeClientBuilder(&PoolConfig{Size: 1})
	a, err := b(context.Background())
	if err == nil || a != nil || !errors.Is(err, ErrClaudeCodeClientNotWired) {
		t.Errorf("got (agent=%v err=%v), want (nil, errors.Is=ErrClaudeCodeClientNotWired)", a, err)
	}
}

func TestGeminiClientBuilder_NotWired_ReturnsSentinel(t *testing.T) {
	b := GeminiClientBuilder(&PoolConfig{Size: 1, APIKey: "fake"})
	a, err := b(context.Background())
	if err == nil || a != nil || !errors.Is(err, ErrGeminiClientNotWired) {
		t.Errorf("got (agent=%v err=%v), want (nil, errors.Is=ErrGeminiClientNotWired)", a, err)
	}
}

func TestJunieClientBuilder_NotWired_ReturnsSentinel(t *testing.T) {
	b := JunieClientBuilder(&PoolConfig{Size: 1})
	a, err := b(context.Background())
	if err == nil || a != nil || !errors.Is(err, ErrJunieClientNotWired) {
		t.Errorf("got (agent=%v err=%v), want (nil, errors.Is=ErrJunieClientNotWired)", a, err)
	}
}

func TestQwenCodeClientBuilder_NotWired_ReturnsSentinel(t *testing.T) {
	b := QwenCodeClientBuilder(&PoolConfig{Size: 1, APIKey: "fake"})
	a, err := b(context.Background())
	if err == nil || a != nil || !errors.Is(err, ErrQwenCodeClientNotWired) {
		t.Errorf("got (agent=%v err=%v), want (nil, errors.Is=ErrQwenCodeClientNotWired)", a, err)
	}
}

// TestClientBuilderSentinels_AreDistinct — disambiguation invariant.
// errors.Is(opencode, claude) MUST be false; without this guarantee,
// callers cannot tell which provider's wiring is still missing.
func TestClientBuilderSentinels_AreDistinct(t *testing.T) {
	sentinels := []error{
		ErrOpenCodeClientNotWired,
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
				t.Errorf("sentinel %d incorrectly matches sentinel %d: %q errors.Is %q", i, j, a, b)
			}
		}
	}
}

// ---------------------------------------------------------------------
// Round-60 factory tests — the *new* contract.
//   - nil cfg → ErrProviderPoolNotImplemented (preserved round-28 sentinel)
//   - non-nil cfg → real *SimpleAgentPool whose first Acquire surfaces
//                   the per-provider Err{Provider}ClientNotWired sentinel.
// ---------------------------------------------------------------------

func TestNewOpenCodePool_NilConfig_ReturnsRoundTwentyEightSentinel(t *testing.T) {
	p, err := NewOpenCodePool(nil)
	if err == nil {
		t.Fatal("NewOpenCodePool(nil) returned nil error — round-28 nil-config sentinel regression")
	}
	if p != nil {
		t.Errorf("NewOpenCodePool(nil) returned non-nil pool alongside error: %v", p)
	}
	if !errors.Is(err, ErrProviderPoolNotImplemented) {
		t.Errorf("errors.Is(err, ErrProviderPoolNotImplemented) = false; err = %v", err)
	}
}

func TestNewClaudeCodePool_NilConfig_ReturnsRoundTwentyEightSentinel(t *testing.T) {
	p, err := NewClaudeCodePool(nil)
	if err == nil || p != nil || !errors.Is(err, ErrProviderPoolNotImplemented) {
		t.Errorf("got (pool=%v err=%v), want (nil, errors.Is=ErrProviderPoolNotImplemented)", p, err)
	}
}

func TestNewGeminiPool_NilConfig_ReturnsRoundTwentyEightSentinel(t *testing.T) {
	p, err := NewGeminiPool(nil)
	if err == nil || p != nil || !errors.Is(err, ErrProviderPoolNotImplemented) {
		t.Errorf("got (pool=%v err=%v), want (nil, errors.Is=ErrProviderPoolNotImplemented)", p, err)
	}
}

func TestNewJuniePool_NilConfig_ReturnsRoundTwentyEightSentinel(t *testing.T) {
	p, err := NewJuniePool(nil)
	if err == nil || p != nil || !errors.Is(err, ErrProviderPoolNotImplemented) {
		t.Errorf("got (pool=%v err=%v), want (nil, errors.Is=ErrProviderPoolNotImplemented)", p, err)
	}
}

func TestNewQwenCodePool_NilConfig_ReturnsRoundTwentyEightSentinel(t *testing.T) {
	p, err := NewQwenCodePool(nil)
	if err == nil || p != nil || !errors.Is(err, ErrProviderPoolNotImplemented) {
		t.Errorf("got (pool=%v err=%v), want (nil, errors.Is=ErrProviderPoolNotImplemented)", p, err)
	}
}

// Table-driven non-nil-config cases — assert each factory returns a real
// pool whose first Acquire surfaces the per-provider builder sentinel.
func TestProviderPools_ValidConfig_ReturnRealPool_AcquireFailsWithBuilderSentinel(t *testing.T) {
	cases := []struct {
		name        string
		factory     func(*PoolConfig) (AgentPool, error)
		wantBuilder error
		wantName    string
	}{
		{"opencode", NewOpenCodePool, ErrOpenCodeClientNotWired, "opencode"},
		{"claude-code", NewClaudeCodePool, ErrClaudeCodeClientNotWired, "claude-code"},
		{"gemini", NewGeminiPool, ErrGeminiClientNotWired, "gemini"},
		{"junie", NewJuniePool, ErrJunieClientNotWired, "junie"},
		{"qwen-code", NewQwenCodePool, ErrQwenCodeClientNotWired, "qwen-code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := tc.factory(&PoolConfig{Size: 2})
			if err != nil {
				t.Fatalf("%s factory({Size:2}) unexpected error: %v", tc.name, err)
			}
			if pool == nil {
				t.Fatalf("%s factory({Size:2}) returned nil pool with nil error — CONTRACT-bluff regression", tc.name)
			}
			// Pool MUST be a *SimpleAgentPool with the right name + size.
			sp, ok := pool.(*SimpleAgentPool)
			if !ok {
				t.Fatalf("%s factory returned pool of type %T, want *SimpleAgentPool", tc.name, pool)
			}
			if sp.Name() != tc.wantName {
				t.Errorf("%s factory pool name = %q, want %q", tc.name, sp.Name(), tc.wantName)
			}
			if sp.Size() != 2 {
				t.Errorf("%s factory pool size = %d, want 2", tc.name, sp.Size())
			}
			// First Acquire MUST fail loudly with the provider sentinel.
			a, acqErr := pool.Acquire(context.Background(), AgentRequirements{})
			if acqErr == nil {
				t.Fatalf("%s pool Acquire returned nil error — wiring-sentinel regression", tc.name)
			}
			if a != nil {
				t.Errorf("%s pool Acquire returned non-nil agent alongside error: %v", tc.name, a)
			}
			if !errors.Is(acqErr, tc.wantBuilder) {
				t.Errorf("%s pool Acquire: errors.Is(err, %v) = false; err = %v", tc.name, tc.wantBuilder, acqErr)
			}
		})
	}
}

// TestNewMultiProviderPool_ValidConfig_BuildsRealPools — the multi-
// provider constructor MUST now succeed when handed non-nil configs.
// Three sub-cases prove the end-to-end Acquire path:
//
//  1. With ZERO agents pre-registered the RoundRobinSelector sees
//     no Available agents and returns ErrNoSuitableAgent — selector
//     semantics, NOT a pool-implementation bluff. This documents the
//     existing selector contract round-60 inherits unchanged.
//
//  2. Registering a real mockAgent into one of the underlying
//     SimpleAgentPools makes round-robin pick that provider and
//     Acquire returns the real agent end-to-end.
//
//  3. Calling SimpleAgentPool.Acquire directly on a fresh (unregistered)
//     pool surfaces the per-provider builder-sentinel — the round-60
//     wiring-gap signal.
func TestNewMultiProviderPool_ValidConfig_BuildsRealPools(t *testing.T) {
	configs := map[string]*PoolConfig{
		"opencode":    {Size: 1},
		"claude-code": {Size: 1},
		"gemini":      {Size: 1, APIKey: "fake"},
	}
	mp, err := NewMultiProviderPool(configs)
	if err != nil {
		t.Fatalf("NewMultiProviderPool: unexpected error %v", err)
	}
	if mp == nil {
		t.Fatal("NewMultiProviderPool returned nil pool with nil error")
	}

	t.Run("no_registered_agents_returns_ErrNoSuitableAgent", func(t *testing.T) {
		a, acqErr := mp.Acquire(context.Background(), AgentRequirements{})
		if a != nil {
			t.Errorf("Acquire returned non-nil agent under empty-Available conditions: %v", a)
		}
		if !errors.Is(acqErr, ErrNoSuitableAgent) {
			t.Errorf("Acquire err = %v, want ErrNoSuitableAgent (selector-semantics signal)", acqErr)
		}
	})

	t.Run("pre_registered_agent_yields_end_to_end_real_agent", func(t *testing.T) {
		// Grab the SimpleAgentPool behind one provider and seed it
		// with a deterministic test agent so the selector picks it.
		sp, ok := mp.pools["opencode"].(*SimpleAgentPool)
		if !ok {
			t.Fatalf("opencode pool type = %T, want *SimpleAgentPool", mp.pools["opencode"])
		}
		want := newMockAgent("opencode-preregistered", "opencode")
		if err := sp.Register(want); err != nil {
			t.Fatalf("Register: %v", err)
		}
		got, err := mp.Acquire(context.Background(), AgentRequirements{PreferredAgent: "opencode"})
		if err != nil {
			t.Fatalf("Acquire: %v", err)
		}
		if got.ID() != want.ID() {
			t.Errorf("Acquire returned agent ID %q, want %q", got.ID(), want.ID())
		}
		// Deliberately NOT calling mp.Release(got) — MultiProviderPool.Release
		// fans out to every pool's Release (see multi_pool.go), which would
		// pollute the claude-code pool's available set for sub-case #3.
		// Releasing directly on the source SimpleAgentPool keeps the
		// other pools fresh for the sentinel sub-case.
		sp.Release(got)
	})

	t.Run("direct_simple_pool_acquire_surfaces_builder_sentinel", func(t *testing.T) {
		sp, ok := mp.pools["claude-code"].(*SimpleAgentPool)
		if !ok {
			t.Fatalf("claude-code pool type = %T, want *SimpleAgentPool", mp.pools["claude-code"])
		}
		a, err := sp.Acquire(context.Background(), AgentRequirements{})
		if err == nil {
			t.Fatal("direct SimpleAgentPool.Acquire returned nil error — wiring-sentinel regression")
		}
		if a != nil {
			t.Errorf("direct Acquire returned non-nil agent alongside error: %v", a)
		}
		if !errors.Is(err, ErrClaudeCodeClientNotWired) {
			t.Errorf("direct Acquire err did not match ErrClaudeCodeClientNotWired: %v", err)
		}
	})
}
