// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Vasic Digital. All rights reserved.

package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// mockPool is a unit-test-only stub installed by tests via dependency
// injection. It must NOT be used in production code (per CONST-050(A)
// — mocks live only in *_test.go). The earlier production-tree
// placement of this type (formerly exported as `MockPool` in
// pkg/agent/multi_pool.go) allowed CONTRACT-bluffs (NewXxxPool returning
// MockPool seeded with nil agents) that round-28 §11.4 audit closed.
//
// Tests that need an AgentPool stand-in MUST construct mockPool here
// in the test package OR — preferably — Register real (test-instance)
// mockAgent values into a NewPool() returned by pool.go.
type mockPool struct {
	name      string
	agents    []Agent
	available []Agent
	mu        sync.Mutex
}

// newMockPool constructs a unit-test-only AgentPool with `size` slots
// pre-allocated. Each slot is initialised with a non-nil mockAgent
// (deterministic IDs `<name>-<index>`) so Acquire() never returns a nil
// agent — the regression that round-28 §11.4 audit caught in the
// previous production-tree MockPool.
func newMockPool(name string, size int) *mockPool {
	agents := make([]Agent, 0, size)
	available := make([]Agent, 0, size)
	for i := 0; i < size; i++ {
		a := newMockAgent(name+"-"+itoa(i), name)
		agents = append(agents, a)
		available = append(available, a)
	}
	return &mockPool{name: name, agents: agents, available: available}
}

// itoa is a tiny strconv.Itoa stand-in to keep the import surface small.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

func (p *mockPool) Register(agent Agent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.agents = append(p.agents, agent)
	p.available = append(p.available, agent)
	return nil
}

func (p *mockPool) Acquire(_ context.Context, _ AgentRequirements) (Agent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.available) == 0 {
		return nil, ErrNoAgentsAvailable
	}
	agent := p.available[0]
	p.available = p.available[1:]
	return agent, nil
}

func (p *mockPool) Release(agent Agent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.available = append(p.available, agent)
}

func (p *mockPool) Available() []Agent {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Agent, len(p.available))
	copy(out, p.available)
	return out
}

func (p *mockPool) HealthCheck(_ context.Context) []HealthStatus {
	return []HealthStatus{}
}

func (p *mockPool) Shutdown(_ context.Context) error {
	return nil
}

// Compile-time guarantee: mockPool satisfies the AgentPool contract.
var _ AgentPool = (*mockPool)(nil)

// TestMockPoolAcquireNeverNil — regression test for round-28 §11.4
// audit. The previous production-tree MockPool seeded `agents[i] = nil`,
// so Acquire() would happily return a nil agent that panicked on first
// use. The replacement test-only mockPool MUST seed every slot with a
// real (test-instance) agent so Acquire() can never return nil.
func TestMockPoolAcquireNeverNil(t *testing.T) {
	p := newMockPool("regression", 3)
	for i := 0; i < 3; i++ {
		a, err := p.Acquire(context.Background(), AgentRequirements{})
		if err != nil {
			t.Fatalf("Acquire %d: unexpected error: %v", i, err)
		}
		if a == nil {
			t.Fatalf("Acquire %d returned a nil agent — round-28 §11.4 CONTRACT-bluff regression", i)
		}
		// Exercise an arbitrary method to confirm the agent is usable
		// (would panic on the previous nil-slot implementation).
		if got := a.Name(); got != "regression" {
			t.Errorf("Acquire %d: agent.Name() = %q, want %q", i, got, "regression")
		}
	}
}

// TestProviderPoolFactoriesNilConfigReturnSentinelError — round-60
// §11.4 evolution of the round-28 regression. Round 28 made every
// factory return ErrProviderPoolNotImplemented unconditionally to
// remove the MockPool-with-nil-agents CONTRACT-bluff. Round 60
// upgraded the factories to return real *SimpleAgentPool instances
// for non-nil PoolConfig; the round-28 sentinel is now reserved for
// the "operator never even configured this provider" case (nil cfg).
//
// This regression test pins that contract: nil cfg in → sentinel out,
// for ALL five provider factories, with provider name in wrapped
// message so operators see which factory rejected nil config.
func TestProviderPoolFactoriesNilConfigReturnSentinelError(t *testing.T) {
	cases := []struct {
		name    string
		factory func(*PoolConfig) (AgentPool, error)
		want    string
	}{
		{"opencode", NewOpenCodePool, "opencode"},
		{"claude-code", NewClaudeCodePool, "claude-code"},
		{"gemini", NewGeminiPool, "gemini"},
		{"junie", NewJuniePool, "junie"},
		{"qwen-code", NewQwenCodePool, "qwen-code"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pool, err := tc.factory(nil)
			if err == nil {
				t.Fatalf("%s factory(nil) returned nil error — round-28 §11.4 nil-config sentinel would regress", tc.name)
			}
			if pool != nil {
				t.Errorf("%s factory(nil) returned non-nil pool alongside error; want (nil, err)", tc.name)
			}
			if !errors.Is(err, ErrProviderPoolNotImplemented) {
				t.Errorf("%s factory(nil): errors.Is(err, ErrProviderPoolNotImplemented) = false; err = %v", tc.name, err)
			}
			if got := err.Error(); !contains(got, tc.want) {
				t.Errorf("%s factory(nil) error %q does not contain provider name %q", tc.name, got, tc.want)
			}
		})
	}
}

// TestNewMultiProviderPoolPropagatesNilConfigSentinel — round-60
// regression. Round-28 contract: NewMultiProviderPool with a configured
// provider whose factory rejected → sentinel propagated. Round-60
// contract narrows the bluff case to nil PoolConfig (truly unconfigured);
// non-nil configs now build real pools and surface per-provider
// builder-sentinel on first Acquire instead. This test pins the
// nil-config-rejected case at the MultiProviderPool layer.
func TestNewMultiProviderPoolPropagatesNilConfigSentinel(t *testing.T) {
	configs := map[string]*PoolConfig{
		"opencode": nil,
	}
	pool, err := NewMultiProviderPool(configs)
	if err == nil {
		t.Fatal("NewMultiProviderPool returned nil error for a nil-config provider — round-28 nil-config sentinel would regress")
	}
	if pool != nil {
		t.Errorf("NewMultiProviderPool returned non-nil pool alongside error; want (nil, err)")
	}
	if !errors.Is(err, ErrProviderPoolNotImplemented) {
		t.Errorf("NewMultiProviderPool: errors.Is(err, ErrProviderPoolNotImplemented) = false; err = %v", err)
	}
}

// contains is a tiny strings.Contains stand-in to keep the import surface
// minimal in this regression-test file.
func contains(haystack, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	if len(haystack) < len(needle) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
