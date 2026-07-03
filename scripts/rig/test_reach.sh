#!/usr/bin/env bash
# Two-way ping reachability test
#
# Purpose:  Verifies L3 connectivity in both directions between the
#           client and server namespaces through the bridge.  Runs
#           setup.sh automatically if the rig is not yet up.
# Usage:    sudo ./test_reach.sh
# Inputs:   (none)
# Outputs:  PASS/FAIL for each direction
# Side-effects: May auto-create namespaces if missing.
# Dependencies: iproute2, bash 4+

set -euo pipefail
source "$(cd "$(dirname "$0")" && pwd)/common.sh"

require_root
check_rig || { warn "Rig not set up — running setup first"; "$(dirname "$0")/setup.sh"; }

NS_CLIENT="hx-client"; NS_SERVER="hx-server"

log "Testing client → server reachability..."
ip netns exec "${NS_CLIENT}" ping -c 3 -W 2 10.0.240.3 && \
  log "Client → server OK" || fail "Client cannot reach server"

log "Testing server → client reachability..."
ip netns exec "${NS_SERVER}" ping -c 3 -W 2 10.0.240.2 && \
  log "Server → client OK" || fail "Server cannot reach client"

log "Both directions reachable — G1 precondition met"
