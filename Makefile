# Helix VPN — Makefile
#
# Purpose:  Central build, test, benchmark, and spike entry points for the
#           Helix VPN project. Delegates to submodule tooling where
#           appropriate.
# Usage:    make <target>    (see `make help`)
# Depends:  scripts/spike.sh, scripts/bench/run.sh, scripts/rig/*.sh

.PHONY: spike spike-fast check test bench bench-compare bench-edge-ab decision-matrix rig rig-teardown rig-test clean help

spike:           ## Run full Phase 0 spike (prerequisites, build, rig, bench)
	sudo ./scripts/spike.sh

spike-fast:      ## Run quick spike (skip benchmarks)
	sudo ./scripts/spike.sh --fast

check:           ## Check all Rust code compiles
	cd submodules/helix_core && cargo check --all-targets

test:            ## Run all Rust tests
	cd submodules/helix_core && cargo test --all-targets

bench:           ## Run full benchmark suite (netns rig + G4 edge A/B), one CSV
	./scripts/bench/run.sh

bench-compare:   ## Compare the two most recent benchmark CSVs
	./scripts/bench/compare.sh --last

bench-edge-ab:   ## Run ONLY the G4 Rust-vs-Go edge A/B bench (no root needed, HVPN-P0-045)
	./scripts/bench/edge_ab.sh

decision-matrix: ## Render the §7.3 G4 decision matrix from the latest edge_ab/bench CSV
	@latest=$$(ls -t bench-results/*.csv 2>/dev/null | head -1); \
	if [ -z "$$latest" ]; then echo "No bench-results/*.csv found — run 'make bench-edge-ab' first"; exit 1; fi; \
	./scripts/bench/decision_matrix.sh "$$latest"

rig:             ## Setup test network namespace topology
	sudo ./scripts/rig/setup.sh

rig-teardown:    ## Remove test namespaces
	sudo ./scripts/rig/teardown.sh

rig-test:        ## Run reachability test
	sudo ./scripts/rig/test_reach.sh

clean:           ## Clean build artifacts
	cd submodules/helix_core && cargo clean 2>/dev/null || true
	cd scripts/bench/tools/rust_edge_bench && cargo clean 2>/dev/null || true
	rm -f scripts/bench/tools/go_edge_bench/go_edge_bench
	rm -rf bench-results/

help:            ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
