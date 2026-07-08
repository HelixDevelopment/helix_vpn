module helix_vpn_containers_check

go 1.26.2

// HVPN-P0-080 — genuine containers-submodule runtime-connectivity check
// invoked from scripts/spike.sh's S4 step. Standalone module, own go.mod
// — NEVER edits submodules/containers, consumes it only via this replace
// directive (mirrors scripts/bench/tools/go_edge_bench's approach).
require digital.vasic.containers v0.0.0

replace digital.vasic.containers => ../../../submodules/containers
