# Helix VPN Network Test Rig

A reproducible network test environment implemented entirely in shell
scripts — no Rust compilation needed.  Uses Linux network namespaces,
virtual ethernet (veth) pairs, nftables firewall rules, and the
`netem` network emulator to simulate and verify VPN-like network
conditions.

## Topology

```
┌──────────────┐     veth-c      ┌──────────────┐     veth-s      ┌──────────────┐
│  ns-client   │◄──────────────►│  hx-bridge   │◄──────────────►│  ns-server   │
│  hx-client   │                │  (br0)       │                │  hx-server   │
│  10.0.240.2  │                │  10.0.240.1  │                │  10.0.240.3  │
└──────────────┘                └──────────────┘                └──────────────┘
```

- **ns-client** — simulated VPN client
- **ns-server** — simulated VPN server / remote hop
- **hx-bridge** — L3 bridge namespace forwarding traffic between client and server

All traffic between client and server passes through the bridge, which
is the natural attachment point for netem impairments and nftables
rules in kill-switch tests.

## Files

| File | Purpose |
|---|---|
| `common.sh` | Shared utilities (coloured logging, root/tool checks, rig-state detection) |
| `setup.sh` | Creates the three namespaces, veth wiring, bridge, IPs, default routes, and baseline nftables tables |
| `teardown.sh` | Destroys all three namespaces (implicitly cleans up veth pairs and nftables state) |
| `test_reach.sh` | Two-way ping test (G1 precondition gate) |
| `test_firewall.sh` | nftables DROP-policy kill-switch test (G1 gate) |
| `test_netem.sh` | Latency + packet-loss injection and verification |

## Quick start

```bash
# 1. Set up the topology (requires root)
sudo ./scripts/rig/setup.sh

# 2. Run the reachability gate
sudo ./scripts/rig/test_reach.sh

# 3. Test the nftables kill-switch
sudo ./scripts/rig/test_firewall.sh

# 4. Test netem impairment injection
sudo ./scripts/rig/test_netem.sh

# 5. Tear down when done
sudo ./scripts/rig/teardown.sh
```

All test scripts auto-detect whether the rig is up and call `setup.sh`
if needed, so running a test directly works as a single command.

## Design principles

- **No Rust dependency** — pure bash, testable immediately.
- **Reproducible** — deterministic IP layout and namespace names.
- **Self-healing** — each test auto-sets up the rig if missing.
- **Clean teardown** — `teardown.sh` removes everything.
- **Kill-switch ready** — the G1 gate (nftables DROP on client output)
  validates that the VPN's emergency kill-switch actually blocks
  traffic, and that connectivity is restored when the switch is
  released.
- **Impairment-ready** — netem delay/loss/jitter can be injected at
  the bridge to simulate real-world WAN conditions without touching
  the host's network stack.
