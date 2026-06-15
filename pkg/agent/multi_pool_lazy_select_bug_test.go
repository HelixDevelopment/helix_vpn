// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"testing"
)

// TestRoundRobinSelector_LazyPoolWithBuildCapacity_IsSelected is a
// reproduce-first RED test (§11.4.115) for the lazy-build multi-provider
// dead-path bug.
//
// FACT (pre-fix): RoundRobinSelector.Select only returns a provider whose
// pool.Available() is already non-empty (the requirements-match loop AND
// the fallback loop both gate on Available()). A freshly-built lazy
// SimpleAgentPool — capacity > 0, working ClientBuilder, ZERO pre-registered
// agents, exactly what NewMultiProviderPool produces — has an EMPTY
// Available() set, so the selector NEVER picks it and MultiProviderPool.Acquire
// returns ErrNoSuitableAgent. The lazy on-demand build path
// (SimpleAgentPool.Acquire's "len(inUse)+len(available) < capacity → builder"
// branch) is therefore DEAD on first use, even though it could build an agent.
//
// This test constructs a MultiProviderPool directly with ONE lazy
// SimpleAgentPool whose builder returns a real (mock) Agent on demand, then
// asserts Acquire SUCCEEDS and returns that built agent. It MUST FAIL on the
// pre-fix code (Acquire returns ErrNoSuitableAgent, nil agent) and PASS once
// Select also honours spare BUILD capacity.
func TestRoundRobinSelector_LazyPoolWithBuildCapacity_IsSelected(t *testing.T) {
	const wantID = "lazy-built-agent"

	built := 0
	builder := func(ctx context.Context) (Agent, error) {
		built++
		return newMockAgent(wantID, "opencode"), nil
	}

	// A lazy pool: capacity 1, working builder, ZERO pre-registered agents.
	// This is exactly the shape NewMultiProviderPool produces for a
	// configured-but-not-yet-acquired provider.
	lazy := NewSimpleAgentPool("opencode", 1, builder)

	// Sanity: the pool genuinely has spare build capacity and an empty
	// available set — the precondition that triggers the bug.
	if got := len(lazy.Available()); got != 0 {
		t.Fatalf("precondition: lazy pool Available() = %d, want 0", got)
	}
	if lazy.Size() <= lazy.InUse() {
		t.Fatalf("precondition: lazy pool has no spare build capacity (Size=%d InUse=%d)",
			lazy.Size(), lazy.InUse())
	}

	mp := &MultiProviderPool{
		pools:    map[string]AgentPool{"opencode": lazy},
		selector: NewRoundRobinSelector(),
	}

	got, err := mp.Acquire(context.Background(), AgentRequirements{})
	if err != nil {
		t.Fatalf("Acquire on lazy multi-provider pool returned error %v; "+
			"want a built agent (lazy-build path is dead on first use)", err)
	}
	if got == nil {
		t.Fatal("Acquire returned nil agent with nil error")
	}
	if got.ID() != wantID {
		t.Fatalf("Acquire returned agent ID %q, want %q (builder not invoked through Acquire)",
			got.ID(), wantID)
	}
	if built == 0 {
		t.Fatal("ClientBuilder was never invoked — lazy on-demand build path not reached")
	}
}
