// Command containers_check is HVPN-P0-080's genuine (not merely
// executable-bit-checking) exercise of the submodules/containers
// runtime-abstraction layer, invoked from scripts/spike.sh's S4 step.
//
// # Why this exists (and why it stops here)
//
// HVPN-P0-080 asks: does scripts/spike.sh genuinely USE the containers
// submodule to boot the Phase-0 test rig topology, or is the existing
// scripts/rig/*.sh netns rig (ip netns + veth + bridge + nftables + tc)
// the correct tool for that job? Investigation (documented in full in
// this task's final report) found: the containers submodule's
// abstraction (runtime.ContainerRuntime / boot.BootManager / compose)
// orchestrates OCI containers (Docker/Podman/K8s) — image pull, compose
// up, health-check, lifecycle. It has NO concept of Linux network
// namespaces, veth pairs, bridges, nftables rulesets, or tc netem — the
// exact kernel primitives HelixVPN-Phase0-Spike.md §3's topology needs.
// Forcing container-boot into netns/veth/bridge/nftables/tc creation
// would misuse the submodule outside its design for zero real benefit
// (no OCI image to pull, no compose service, no HTTP/TCP health check
// applies to a raw kernel network namespace). scripts/rig/*.sh therefore
// remains the correct, sufficient approach for the rig's own topology —
// this program does NOT replace or wrap it.
//
// What IS genuinely exercised here: the one piece of §3's topology that
// naturally fits "a service to boot and health-check" — a future
// containerized stand-in for the connector-site LAN service (today an ad
// hoc `python3 -m http.server` per §3) — needs the runtime layer to
// actually detect and talk to a real container engine first. This
// program proves that foundation is real and working in THIS project's
// environment: `runtime.AutoDetect` genuinely finds and queries the
// live, already-installed rootless Podman (confirmed separately via
// `podman info`), rather than merely checking that a prebuilt `bin/boot`
// file has its executable bit set (spike.sh's previous S4 behaviour).
//
// Wiring an actual compose-based containerized LAN-service replacement
// for the rig's current ad-hoc HTTP stand-in is tracked as separate
// future work (S2 rig completion), not implemented here — that is a
// materially bigger, image-pull-dependent lift than this task's scope,
// and the current rig already satisfies the G1/G2 gates without it.
//
// This module is standalone (own go.mod, replace-directive path
// dependency on submodules/containers) — it never edits that submodule.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"digital.vasic.containers/pkg/runtime"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rt, err := runtime.AutoDetect(ctx)
	if err != nil {
		fmt.Printf("FAIL: no container runtime detected: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("runtime detected: %s\n", rt.Name())

	version, err := rt.Version(ctx)
	if err != nil {
		fmt.Printf("FAIL: runtime %q detected but Version() query failed: %v\n", rt.Name(), err)
		os.Exit(1)
	}
	fmt.Printf("runtime version: %s\n", version)

	available := rt.IsAvailable(ctx)
	fmt.Printf("runtime available: %v\n", available)
	if !available {
		fmt.Println("FAIL: runtime reported unavailable on second check")
		os.Exit(1)
	}

	containers, err := rt.List(ctx, runtime.ListFilter{All: true})
	if err != nil {
		fmt.Printf("FAIL: runtime %q List() query failed: %v\n", rt.Name(), err)
		os.Exit(1)
	}
	fmt.Printf("containers currently visible to this runtime: %d\n", len(containers))

	fmt.Println("PASS: containers submodule genuinely detected + queried a live container runtime")
}
