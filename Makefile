# Helix VPN — Makefile
#
# Purpose:  Central build, test, benchmark, and spike entry points for the
#           Helix VPN project. Delegates to submodule tooling where
#           appropriate.
# Usage:    make <target>    (see `make help`)
# Depends:  scripts/spike.sh, scripts/bench/run.sh, scripts/rig/*.sh

.PHONY: spike spike-fast check test bench bench-compare rig rig-teardown rig-test clean help

spike:           ## Run full Phase 0 spike (prerequisites, build, rig, bench)
	sudo ./scripts/spike.sh

spike-fast:      ## Run quick spike (skip benchmarks)
	sudo ./scripts/spike.sh --fast

check:           ## Check all Rust code compiles
	cd submodules/helix_core && cargo check --all-targets

test:            ## Run all Rust tests
	cd submodules/helix_core && cargo test --all-targets

bench:           ## Run benchmark suite
	./scripts/bench/run.sh

bench-compare:   ## Compare the two most recent benchmark CSVs
	./scripts/bench/compare.sh --last

rig:             ## Setup test network namespace topology
	sudo ./scripts/rig/setup.sh

rig-teardown:    ## Remove test namespaces
	sudo ./scripts/rig/teardown.sh

rig-test:        ## Run reachability test
	sudo ./scripts/rig/test_reach.sh

clean:           ## Clean build artifacts
	cd submodules/helix_core && cargo clean 2>/dev/null || true
	rm -rf bench-results/

help:            ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
