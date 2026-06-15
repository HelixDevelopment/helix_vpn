// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package agent

import (
	"context"
	"testing"
	"time"
)

// ---------------------------------------------------------------------
// Reproduce-first RED tests (anti-bluff §11.4.6 / §11.4.115) for two
// genuine orchestration defects in SimpleAgentPool.Release and
// MultiProviderPool.Release. Each test FAILs on the current code and
// PASSes only after the fix. No mutation markers, no stubs beyond the
// existing in-test mockAgent + ClientBuilder.
// ---------------------------------------------------------------------

// BUG A: SimpleAgentPool.Release unconditionally appends the agent to
// p.available even when it is NOT currently tracked as in-use. A double
// Release (or a Release of an agent the pool never handed out) therefore
// inserts a DUPLICATE into p.available. A capacity-1 pool then hands the
// SAME agent instance to two concurrent callers, each believing it has
// exclusive ownership — a capacity-breach / shared-resource defect.
func TestSimpleAgentPool_DoubleRelease_DoesNotDuplicateAvailable(t *testing.T) {
	builder, _ := makeMockBuilder("dup")
	p := NewSimpleAgentPool("dup", 1, builder)
	defer func() { _ = p.Shutdown(context.Background()) }()

	ctx := context.Background()

	a, err := p.Acquire(ctx, AgentRequirements{})
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}

	// Caller (incorrectly, but defensively the pool must tolerate it)
	// releases the same agent twice. The pool MUST NOT end up with two
	// copies of one agent in its available set.
	p.Release(a)
	p.Release(a)

	// Pool capacity is 1, so after correct accounting exactly ONE agent
	// is available. The first Acquire must succeed; the second must be
	// unable to hand out a second distinct agent without exceeding
	// capacity, so it must block. We give it a short deadline.
	a1, err := p.Acquire(ctx, AgentRequirements{})
	if err != nil {
		t.Fatalf("re-acquire #1 failed: %v", err)
	}

	ctx2, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	a2, err := p.Acquire(ctx2, AgentRequirements{})
	if err == nil {
		// BUG REPRODUCED: a second agent was handed out from a
		// capacity-1 pool because the duplicate Release inflated
		// p.available. a2 is the SAME instance as a1.
		if a2 == a1 {
			t.Fatalf("capacity-1 pool handed out the SAME agent instance to two concurrent holders "+
				"(double-Release duplicated it in available): a1=%p a2=%p", a1, a2)
		}
		t.Fatalf("capacity-1 pool handed out two agents after double-Release (capacity breach)")
	}
	// Correct behaviour: second acquire blocks until the deadline.
}

// BUG B: MultiProviderPool.Release calls Release on EVERY underlying
// pool, not just the one that owns the agent. Combined with
// SimpleAgentPool.Release's unconditional append, an agent acquired
// from pool "alpha" and released through the facade leaks into pool
// "beta"'s available set — a foreign pool will then hand out an agent
// it never built, corrupting capacity accounting across providers.
func TestMultiProviderPool_Release_DoesNotLeakAgentIntoForeignPool(t *testing.T) {
	alphaBuilder, _ := makeMockBuilder("alpha")
	betaBuilder, _ := makeMockBuilder("beta")

	alpha := NewSimpleAgentPool("alpha", 1, alphaBuilder)
	beta := NewSimpleAgentPool("beta", 1, betaBuilder)

	// Seed each pool with one pre-registered agent so the RoundRobin
	// selector (which only selects pools that already report an
	// available agent) can serve an Acquire. This isolates the test on
	// the Release-leak defect, not on selection.
	if err := alpha.Register(newMockAgent("alpha-seed", "alpha")); err != nil {
		t.Fatalf("seed alpha: %v", err)
	}
	if err := beta.Register(newMockAgent("beta-seed", "beta")); err != nil {
		t.Fatalf("seed beta: %v", err)
	}

	m := &MultiProviderPool{
		pools:    map[string]AgentPool{"alpha": alpha, "beta": beta},
		selector: NewRoundRobinSelector(),
	}
	defer func() { _ = m.Shutdown(context.Background()) }()

	ctx := context.Background()

	// Acquire one agent. Whichever pool served it, we release through the
	// facade and then check the OTHER pool did not gain a foreign agent.
	a, err := m.Acquire(ctx, AgentRequirements{})
	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}

	owner := a.Name() // "alpha" or "beta"
	var foreign *SimpleAgentPool
	if owner == "alpha" {
		foreign = beta
	} else {
		foreign = alpha
	}

	foreignBefore := len(foreign.Available())

	m.Release(a)

	// BUG REPRODUCED: the foreign pool's available set grows because the
	// facade released the agent into every pool, and SimpleAgentPool.Release
	// appended it unconditionally. The foreign pool must be unchanged.
	if foreignAfter := len(foreign.Available()); foreignAfter != foreignBefore {
		t.Fatalf("MultiProviderPool.Release leaked agent %q (from %q) into foreign pool %q: "+
			"foreign.Available() went %d -> %d", a.Name(), owner, foreign.Name(), foreignBefore, foreignAfter)
	}
}
