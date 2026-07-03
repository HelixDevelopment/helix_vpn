#!/usr/bin/env bash
# Tear down the Helix VPN network test topology
#
# Purpose:  Removes the three namespaces created by setup.sh,
#           which implicitly destroys all veth pairs and resets
#           any nftables rulesets installed inside them.
# Usage:    sudo ./teardown.sh
# Inputs:   (none)
# Outputs:  Namespaces removed; host ip_forward left as-is.
# Side-effects: Kills any processes still running inside the
#               namespaces (ping, iperf, etc.).
# Dependencies: iproute2, bash 4+

set -euo pipefail
source "$(cd "$(dirname "$0")" && pwd)/common.sh"

require_root

for ns in hx-client hx-server hx-bridge; do
  ip netns del "${ns}" 2>/dev/null || true
done

log "All Helix VPN test-rig namespaces removed"
