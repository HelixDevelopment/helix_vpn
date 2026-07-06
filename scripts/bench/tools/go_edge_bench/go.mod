module helix_vpn_go_edge_bench

go 1.26.2

// HVPN-P0-045 benchmark driver for the Go go-edge MASQUE termination path
// (G4 A/B bench). Standalone module under scripts/bench/ — NOT a member
// of submodules/helix_go's own module and never edits it; consumes it
// only via this replace directive, exactly mirroring the sibling
// rust_edge_bench tool's Cargo path-dependency approach.
require github.com/vasic-digital/helix_go v0.0.0

require (
	github.com/quic-go/masque-go v0.4.0
	github.com/quic-go/quic-go v0.60.0
	github.com/yosida95/uritemplate/v3 v3.0.2
)

require (
	github.com/dunglas/httpsfv v1.1.0 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/text v0.38.0 // indirect
)

replace github.com/vasic-digital/helix_go => ../../../../submodules/helix_go
