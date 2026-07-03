#!/usr/bin/env bash
# Setup the Helix VPN network test topology
#
# Purpose:  Creates three network namespaces (client, server, bridge)
#           connected by veth pairs through a Linux bridge, configures
#           IP addressing and default routes, and installs a baseline
#           nftables ruleset (default accept) on each namespace.
# Usage:    sudo ./setup.sh
# Inputs:   (none; configuration is hardcoded below)
# Outputs:  Three namespaces with working L3 connectivity.
# Side-effects: Removes any pre-existing namespaces with the same
#               names; enables ip_forward on the host.
# Dependencies: iproute2, nftables, bash 4+

set -euo pipefail

RIG_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${RIG_DIR}/common.sh"

# ── Configuration ───────────────────────────────────────────────────
NS_CLIENT="hx-client"
NS_SERVER="hx-server"
NS_BRIDGE="hx-bridge"
VETH_CLIENT="veth-c"
VETH_SERVER="veth-s"
BRIDGE_IP="10.0.240.1/24"
CLIENT_IP="10.0.240.2/24"
SERVER_IP="10.0.240.3/24"

# ── Pre-flight ──────────────────────────────────────────────────────
require_root
require_tools ip nft

# Tear down any leftovers from a previous run
for ns in "${NS_CLIENT}" "${NS_SERVER}" "${NS_BRIDGE}"; do
  ip netns del "${ns}" 2>/dev/null || true
done

# ── Phase 1: Create namespaces ──────────────────────────────────────
log "Creating network namespaces..."
ip netns add "${NS_CLIENT}"
ip netns add "${NS_SERVER}"
ip netns add "${NS_BRIDGE}"

# ── Phase 2: Create veth pairs ──────────────────────────────────────
ip link add "${VETH_CLIENT}" type veth peer name "${VETH_CLIENT}-br"
ip link add "${VETH_SERVER}" type veth peer name "${VETH_SERVER}-br"

# ── Phase 3: Move ends into namespaces ──────────────────────────────
ip link set "${VETH_CLIENT}" netns "${NS_CLIENT}"
ip link set "${VETH_SERVER}" netns "${NS_SERVER}"
ip link set "${VETH_CLIENT}-br" netns "${NS_BRIDGE}"
ip link set "${VETH_SERVER}-br" netns "${NS_BRIDGE}"

# ── Phase 4: Configure bridge namespace ─────────────────────────────
ip netns exec "${NS_BRIDGE}" ip link add br0 type bridge
ip netns exec "${NS_BRIDGE}" ip link set "${VETH_CLIENT}-br" master br0
ip netns exec "${NS_BRIDGE}" ip link set "${VETH_SERVER}-br" master br0
for iface in br0 "${VETH_CLIENT}-br" "${VETH_SERVER}-br"; do
  ip netns exec "${NS_BRIDGE}" ip link set "${iface}" up
done
ip netns exec "${NS_BRIDGE}" ip addr add "${BRIDGE_IP}" dev br0

# ── Phase 5: Configure client namespace ──────────────────────────────
ip netns exec "${NS_CLIENT}" ip addr add "${CLIENT_IP}" dev "${VETH_CLIENT}"
ip netns exec "${NS_CLIENT}" ip link set lo up
ip netns exec "${NS_CLIENT}" ip link set "${VETH_CLIENT}" up
ip netns exec "${NS_CLIENT}" ip route add default via 10.0.240.1

# ── Phase 6: Configure server namespace ──────────────────────────────
ip netns exec "${NS_SERVER}" ip addr add "${SERVER_IP}" dev "${VETH_SERVER}"
ip netns exec "${NS_SERVER}" ip link set lo up
ip netns exec "${NS_SERVER}" ip link set "${VETH_SERVER}" up
ip netns exec "${NS_SERVER}" ip route add default via 10.0.240.1

# ── Phase 7: Enable IPv4 forwarding on host (needed for bridge) ─────
echo 1 > /proc/sys/net/ipv4/ip_forward 2>/dev/null || true

# ── Phase 8: Baseline nftables rulesets (default accept) ────────────
# Each namespace gets a clean inet filter table with default accept on
# input/forward/output so tests can install their own policies.
for ns in "${NS_CLIENT}" "${NS_SERVER}" "${NS_BRIDGE}"; do
  ip netns exec "${ns}" nft add table inet filter 2>/dev/null || true
  ip netns exec "${ns}" nft add chain inet filter input   '{ type filter hook input   priority 0; policy accept; }' 2>/dev/null || true
  ip netns exec "${ns}" nft add chain inet filter forward '{ type filter hook forward priority 0; policy accept; }' 2>/dev/null || true
  ip netns exec "${ns}" nft add chain inet filter output  '{ type filter hook output  priority 0; policy accept; }' 2>/dev/null || true
done

log "Topology created: client(${CLIENT_IP}) <-> bridge(${BRIDGE_IP}) <-> server(${SERVER_IP})"
log "Run: ip netns exec ${NS_CLIENT} ping ${SERVER_IP}"
