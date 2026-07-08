# Platform VPN Adapter APIs — Deep-Architecture Research

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Authority:** Helix VPN Phase 1 platform-shim research (§11.4.8 deep-web-research)
**Scope:** Covers Android, iOS/macOS, Windows, and Linux — the four platforms targeted by Phase 1 (HVPN-P1-E12) and Phase 2

---

## 1. Executive summary

The Helix VPN project's `helix_core` crate ships a transport abstraction (`Transport`, `Connection`, `Listener` traits + `TransportRegistry`) that every platform-specific adapter must bridge. The Phase 1 WBS (HVPN-P1-E12) defines three platforms for MVP — Android (VpnService+JNI), iOS (NEPacketTunnelProvider), and Linux (native Rust) — with Windows deferred to Phase 2. This document maps the architecture gap between the Rust core and each platform's native VPN API, covering API signatures, FFI integration points, required permissions/entitlements, provisioning requirements, reference implementations (open-source VPN apps), and known limitations.

**Key architectural decision:** The project already uses the Mozilla UniFFI framework (`uniffi` v0.31.1, April 2026) ecosystem for cross-platform FFI. The Rust core exposes `Transport`, `Connection`, and `Listener` through UniFFI-generated bindings; platform shims implement the bridge between UniFFI types and platform-specific packet I/O primitives. The `TransportRegistry` is the single point of runtime transport selection shared across all platforms.

**Platform readiness summary:**

| Platform | Phase | API maturity | FFI path | Key risk |
|---|---|---|---|---|
| **Linux** | 1 (HVPN-P1-120) | Production-grade (native Rust) | Direct Rust (TUN + nftables) | Low — reference platform, no FFI needed |
| **Android** | 1 (HVPN-P1-121) | Production-grade (API 14+) | UniFFI → Kotlin JNI → VpnService | Medium — VpnService thread model + Doze |
| **iOS/macOS** | 1 (HVPN-P1-122) | Production-grade (iOS 9+) | UniFFI → Swift → NEPacketTunnelProvider | High — iOS memory ceiling G3 gate + extension lifecycle |
| **Windows** | 2 | Production-grade (Vista+) | UniFFI → C-ABI → WinDivert or WFP callout | Medium — driver signing + elevation requirements |

---

## 2. Linux (native Rust) — Reference Platform

### 2.1 API overview

The Linux platform shim is the simplest: no cross-language FFI boundary. The Rust core drives a TUN device directly through the `helix-tun` crate (which wraps `tokio::task::spawn_blocking` around raw TUN I/O) and programs nftables rules via `nft` CLI or the `libnftables` C library.

**Current rig architecture** (`scripts/rig/setup.sh`):
- Three network namespaces (`hx-client`, `hx-server`, `hx-bridge`)
- veth pairs connected through a Linux bridge (`br0` in `hx-bridge`)
- Baseline nftables ruleset: `inet filter` with default-accept chains (`input`, `forward`, `output`)
- IPv4 forwarding enabled on host

**Phase 1 requirement** (HVPN-P1-120):
- Linux core drives TUN directly (kernel WireGuard or userspace `boringtun`)
- Reference platform for the netns E2E rig
- Full enroll -> connect -> reach journey on the netns rig

### 2.2 Integration with helix_core

```rust
// Direct Rust integration — no FFI boundary
use helix_tun::TunDevice;
use helix_transport::{Transport, Connection, TransportRegistry};

// TUN I/O via tokio::spawn_blocking
// nftables rules via std::process::Command("nft") or libnftables-rs
```

**nftables integration points:**
- **Kill-switch** (HVPN-P1-101): default-deny output chain + allow-only-VPN-interface rules, atomic rule swap on tunnel state change
- **Verdict-map enforcement** (HVPN-P1-102): port-level allow/deny beyond WireGuard CIDR-only `AllowedIPs`
- **DNS leak prevention**: redirect port 53 to overlay resolver, block all other DNS paths
- **Edge filtering**: `inet filter output` chain with cgroupv2 matching for per-application routing

**Key nftables techniques** from deep research (burpwn, cgtproxy, Xray TProxy):
1. `REDIRECT` / `TPROXY` for transparent proxying
2. `meta mark set` + policy routing (`ip rule add fwmark 1 table 100`) for loop prevention
3. `socket cgroupv2 level N` matching for per-cgroup verdicts
4. Atomic rule transactions: `nft -f ruleset.nft` for kill-switch on/off

### 2.3 Reference implementations

| Project | Key technique |
|---|---|
| `burpwn` (own2pwn-fr, v0.1.1, June 2026) | Per-namespace nftables `REDIRECT` rules, rootless `CAP_NET_ADMIN` in child namespace |
| `cgtproxy` (black-desk) | Go-based cgroupv2 dynamic TPROXY with inotify-driven nftables updates |
| Xray-core TProxy (xtls.github.io) | Canonical nftables TPROXY + policy routing setup for transparent proxying |
| `meow-rs` (madeye) | Rust transparent proxy with `SO_ORIGINAL_DST` + `SO_MARK` loop prevention |

### 2.4 Gotchas

- **Reply-packet loops**: TPROXY scenarios require `type route` output chain + `meta mark set 0` to prevent proxy return traffic from being re-captured (documented in hysteria2-tproxy case study)
- **Mode mismatch**: `REDIRECT` requires transparent proxy mode (`SO_ORIGINAL_DST`); regular-mode proxy + REDIRECT silently fails
- **cgroupv2 availability**: older kernels (pre-4.15) lack cgroupv2; fallback to UID matching or network namespace isolation
- **Teardown race**: flushing entire nftables chain instead of specific rule removal can create a plaintext window during reconnection

---

## 3. Android — VpnService + JNI/UniFFI

### 3.1 API overview — `android.net.VpnService`

**Core classes:**

| Class | Role |
|---|---|
| `VpnService` | Base `Service` subclass; creates & manages the virtual TUN interface |
| `VpnService.Builder` | Fluent config: addresses, routes, DNS, MTU, app filtering |
| `ParcelFileDescriptor` | Returned by `Builder.establish()` — the TUN file descriptor for raw IP I/O |
| `FileInputStream` / `FileOutputStream` | Java I/O wrappers on the `ParcelFileDescriptor` for packet read/write |

**Builder API (Kotlin):**
```kotlin
val builder = VpnService.Builder()
    .setSession("Helix VPN")
    .setMtu(1500)                       // 1420 for WG headroom per helix-transport
    .addAddress("10.0.0.2", 32)         // TUN interface address
    .addRoute("0.0.0.0", 0)             // Route all traffic
    .addDnsServer("8.8.8.8")
    .addDnsServer("8.8.4.4")
    // App filtering (API 21+):
    // .addAllowedApplication("com.example.app")   // VPN only for specific apps
    // .addDisallowedApplication("com.android.chrome") // exclude apps
    .allowBypass()                      // API 28+: allow traffic to bypass VPN
val tunFd: ParcelFileDescriptor? = builder.establish()  // null = app not prepared
```

**Key lifecycle callbacks:**
- `onStartCommand()` — create TUN, start packet I/O thread (return `START_STICKY`)
- `onRevoke()` — system revoked the VPN; must stop
- `VpnService.prepare(context)` — static method; returns `null` (already authorized) or an `Intent` to launch the system authorization dialog

**Always-on VPN:**
```xml
<meta-data android:name="android.net.VpnService.SUPPORTS_ALWAYS_ON"
           android:value="true" />
```
Enabled by user in Settings -> Network & internet -> VPN -> (gear icon) for the app. Service automatically restarts after reboot or crash.

**Lockdown mode** (API 22+, the "kill-switch" analogue):
- Viewable via `VpnService.isLockdownEnabled()`
- Configured by user in system VPN settings
- Blocks all traffic that doesn't go through the VPN

### 3.2 Integration with Rust core (FFI path)

The project's Phase 1 plan specifies JNI as the bridge. However, the broader helix ecosystem already uses **UniFFI** (Mozilla's cross-platform bindgen), which generates both Kotlin and Swift bindings from a single Rust UDL/ proc-macro definition.

**Recommended approach: UniFFI (v0.31.x) as primary, raw JNI as escape hatch.**

```
┌──────────────────────────────────────────────────────┐
│  Kotlin (VpnService subclass)                        │
│  ┌──────────────────────────┐                        │
│  │ Packet I/O thread        │                        │
│  │ FileInputStream.read()   │                        │
│  └──────────┬───────────────┘                        │
│             │ raw IP packet (byte[])                  │
│             ▼                                         │
│  ┌──────────────────────────┐                        │
│  │ UniFFI-generated Kotlin  │  HelixCore.kt           │
│  │ (JNI under the hood)     │  send_packet(buf)       │
│  └──────────┬───────────────┘  recv_packet() -> buf   │
│             │ JNI call                                │
├─────────────┼─────────────────────────────────────────┤
│  Native (.so)                                         │
│  ┌──────────▼───────────────┐                        │
│  │ Rust helix-core          │  WireGuard + transport  │
│  │ (cdylib, aarch64-android)│  encrypt/decrypt/route  │
│  └──────────────────────────┘                        │
└──────────────────────────────────────────────────────┘
```

**UniFFI type mapping:**

| Rust | Kotlin (UniFFI) |
|---|---|
| `Vec<u8>` | `List<Byte>` |
| `String` | `String` |
| `Result<T, E>` | throws HelixException |
| `enum TransportKind` | `sealed class TransportKind` |
| async fn | `suspend fun` (Kotlin coroutines) |

**Build pipeline:**
```bash
# Cross-compile Rust to Android .so
cargo ndk -t aarch64-linux-android -t armv7-linux-androideabi \
  -o app/src/main/jniLibs build --release

# Generate UniFFI bindings
uniffi-bindgen generate src/helix_core.udl --language kotlin \
  --out-dir app/src/main/java/com/helix/vpn/bridge/
```

**Reference:** defguard_boringtun v0.6.5 (Feb 2026) ships both JNI (`src/jni.rs`) and UniFFI Swift bindings from the same Rust crate, proving the dual-FFI approach is viable.

### 3.3 Required permissions

```xml
<uses-permission android:name="android.permission.INTERNET" />
<!-- FOREGROUND_SERVICE required for API 28+ to start foreground service -->
<uses-permission android:name="android.permission.FOREGROUND_SERVICE" />
<!-- FOREGROUND_SERVICE_SPECIAL_USE for API 34+ -->
<uses-permission android:name="android.permission.FOREGROUND_SERVICE_SPECIAL_USE" />

<service
    android:name=".HelixVpnService"
    android:permission="android.permission.BIND_VPN_SERVICE"
    android:exported="false">
    <intent-filter>
        <action android:name="android.net.VpnService" />
    </intent-filter>
    <meta-data android:name="android.net.VpnService.SUPPORTS_ALWAYS_ON"
               android:value="true" />
</service>
```

### 3.4 Device provisioning

- **No special provisioning** required beyond the regular app install
- User must authorize VPN creation via system dialog (one-time per app install)
- Target API: Android 7.0+ (API 24) — covers >95% of active devices
- No root required; VpnService works on unmodified devices
- Foreground service notification mandatory (API 26+): `startForeground(id, notification)`

### 3.5 Reference implementations

| Project | Key technique | Link |
|---|---|---|
| **WireGuard Android** | Official Android VPN client; uses `tunnel` library (wg-go) + VpnService | `com.wireguard.android:tunnel:1.0.20260102` |
| **Mullvad VPN** | Production VPN; `talpid-tunnel` crate calls VpnService via JNI bridge from Rust | `mullvad/mullvadvpn-app` (GitHub) |
| **vpn-frontend (ian52n)** | P2P VPN; BoringTun Rust via JNI for WG crypto on Android 7.0+ | `ian52n/vpn-frontend` (GitHub, Jan 2026) |
| **NetGuard (M66B)** | No-root firewall; canonical example of VpnService packet filtering | `M66B/NetGuard` (GitHub) |
| **shadowsocks-android** | Proxy client; VpnService + encryption/forwarding | `shadowsocks/shadowsocks-android` (GitHub) |

### 3.6 Gotchas

- **Single VPN per device**: only one VpnService can be active at a time; creating a new one deactivates the old
- **`establish()` returns null**: app must call `VpnService.prepare()` first and receive user authorization
- **Doze mode**: Android's power-saving may pause the VPN thread; use `WakeLock` (PARTIAL_WAKE_LOCK) for sustained transfers
- **Packet I/O is blocking**: must run on a dedicated thread — never on the service main thread
- **API 29+ restriction**: cannot add own package to `addDisallowedApplication()` (system blocks self-exclusion)
- **Android 12+ foreground service restrictions**: stricter notification requirements, may need `foregroundServiceType="specialUse"`
- **IPv6 on TUN**: Android's VpnService TUN does not support IPv6 by default on all devices; test on target hardware
- **MTU clamping**: WireGuard overhead (~80 bytes) must be subtracted from the VpnService MTU; `helix-transport` defines `effective_mtu()` returning 1420 for plain-udp, 1280 for MASQUE
- **The §11.4.111 trap**: don't bind to a hardcoded TUN address guessing stability — configure via the spec

---

## 4. iOS / macOS — NEPacketTunnelProvider

### 4.1 API overview — NetworkExtension Framework

**Core classes:**

| Class | Role |
|---|---|
| `NEPacketTunnelProvider` | Subclass lives in a Network Extension target; entry point for custom VPN protocol |
| `NEPacketTunnelNetworkSettings` | Declares tunnel interface configuration: addresses, routes, DNS, MTU |
| `NEIPv4Settings` / `NEIPv6Settings` | IPv4/IPv6 address, subnet, and route configuration |
| `NEDNSSettings` / `NEDNSOverHTTPSSettings` | DNS configuration for the tunnel |
| `NEPacketTunnelFlow` | The packet I/O pipe: `readPacketObjects(completionHandler:)` and `writePacketObjects(_:completionHandler:)` |
| `NETunnelProviderManager` | Main-app API to create, save, and start VPN configurations |
| `NETunnelProviderProtocol` | Protocol configuration: `providerBundleIdentifier`, `serverAddress`, `providerConfiguration` (custom dict) |

**Lifecycle:**
```
User taps "Connect" (main app)
  └─> NETunnelProviderManager.startVPNTunnel()
       └─> System launches NEPacketTunnelProvider extension process
            └─> startTunnel(options:completionHandler:)  ← Your code
                 ├── Configure NEPacketTunnelNetworkSettings
                 ├── setTunnelNetworkSettings(_:completionHandler:)
                 ├── Begin readPacketObjects / writePacketObjects loop
                 └── completionHandler(nil)  ← Signal "connected"
       
User taps "Disconnect" / system revokes
  └─> stopTunnel(with:completionHandler:)
       └─> Cleanup, close connections
```

**Network settings example (Swift):**
```swift
let settings = NEPacketTunnelNetworkSettings(tunnelRemoteAddress: "10.0.0.1")

let ipv4 = NEIPv4Settings(addresses: ["10.0.0.2"],
                          subnetMasks: ["255.255.255.0"])
ipv4.includedRoutes = [NEIPv4Route.default()]  // Route all traffic
settings.ipv4Settings = ipv4

let ipv6 = NEIPv6Settings(addresses: ["fd00::2"],
                          networkPrefixLengths: [120])  // /120 clamp (iOS gotcha)
ipv6.includedRoutes = [NEIPv6Route.default()]
settings.ipv6Settings = ipv6

let dns = NEDNSSettings(servers: ["1.1.1.1", "1.0.0.1"])
settings.dnsSettings = dns

settings.mtu = 1440  // Conservative for WG overhead + iOS stack
```

**On-demand rules (auto-connect):**
```swift
// Connect automatically on any network change
let rule = NEOnDemandRuleConnect()
rule.interfaceTypeMatch = .any
manager.onDemandRules = [rule]
manager.isOnDemandEnabled = true
```

### 4.2 Integration with Rust core (FFI path)

**Architecture: UniFFI → Swift staticlib**

```
┌──────────────────────────────────────────────────────┐
│  Swift (NEPacketTunnelProvider subclass)              │
│  ┌──────────────────────────┐                        │
│  │ Packet I/O loop           │                        │
│  │ packetFlow.readPackets()  │  NEPacketTunnelFlow    │
│  └──────────┬───────────────┘                        │
│             │ Data / [Data] (NEPacket)                │
│             ▼                                         │
│  ┌──────────────────────────┐                        │
│  │ UniFFI-generated Swift    │  HelixCore.swift       │
│  │ (C-ABI under the hood)   │  sendPacket(data:)     │
│  └──────────┬───────────────┘  recvPacket() -> Data  │
│             │ C-ABI call (extern "C")                 │
├─────────────┼─────────────────────────────────────────┤
│  Native (.a, staticlib)                                │
│  ┌──────────▼───────────────┐                        │
│  │ Rust helix-core          │  WireGuard + transport  │
│  │ (staticlib, aarch64-ios) │  encrypt/decrypt/route  │
│  └──────────────────────────┘                        │
└──────────────────────────────────────────────────────┘
```

**Build pipeline:**
```bash
# Compile Rust to iOS static library
cargo build --release --target aarch64-apple-ios

# Generate UniFFI Swift bindings
uniffi-bindgen generate src/helix_core.udl --language swift \
  --out-dir apple/Sources/HelixCore/

# Package as XCFramework for multi-arch
xcodebuild -create-xcframework \
  -library target/aarch64-apple-ios/release/libhelix_core.a \
  -headers apple/Headers/ \
  -output HelixCore.xcframework
```

**Key decision: UniFFI vs raw C-ABI.** The project's Phase 1 WBS references a `helix-core-ffi` crate. UniFFI is the recommended path because: (a) it generates Swift error types from Rust `Result<T,E>`, (b) it maps async Rust to Swift `async throws` natively, (c) it handles reference counting automatically for complex types, and (d) the same UDL serves both Kotlin and Swift — one definition, two platforms.

**Memory ceiling constraint (G3 gate, HVPN-P1-003):** The iOS Network Extension process has a ~50 MB soft memory limit (iOS 16+) / ~15 MB (older). The Rust staticlib + WireGuard state machine MUST stay under this with >=30% headroom over a 30-minute / 1 GB transfer. This is Phase 1's make-or-break gate. Strategies if breached:
1. Use `jemalloc` instead of the system allocator (smaller working set)
2. Minimize buffer sizes (`effective_mtu()` is already conservative)
3. Stream packets one-at-a-time rather than batching
4. Release the NEPacketTunnelFlow read semaphore aggressively between reads
5. Apply the §6.4 fallback ladder (operator decision per §11.4.66) only if all four strategies exhausted

### 4.3 Required entitlements

**Capability: Network Extensions -> Packet Tunnel Provider**

```xml
<!-- Both Main App AND Extension targets: -->
<key>com.apple.developer.networking.networkextension</key>
<array>
    <string>packet-tunnel-provider</string>
</array>

<!-- Both targets (shared data): -->
<key>com.apple.security.application-groups</key>
<array>
    <string>group.com.helix.vpn</string>
</array>
```

**Critical provisioning detail** (Apple DTS Engineer Quinn "The Eskimo!"): The `packet-tunnel-provider` entitlement value is **restricted but not managed** — any paid Apple Developer Program member can enable it on their App ID. Do NOT add the "Personal VPN" capability (that's for built-in IKEv2/IPsec only).

### 4.4 Device provisioning — full checklist

| # | Step | Where |
|---|---|---|
| 1 | Create 2 App IDs: `com.helix.vpn` (main) + `com.helix.vpn.tunnel` (extension) | Apple Developer Portal -> Certificates, Identifiers & Profiles |
| 2 | Enable **Network Extensions** capability on both App IDs | Developer Portal -> App ID -> Edit |
| 3 | Create App Group `group.com.helix.vpn` | Developer Portal -> App Groups |
| 4 | Assign App Group to both App IDs | Developer Portal -> App ID -> Edit |
| 5 | Generate 2 provisioning profiles (dev + dist for each App ID) | Developer Portal -> Profiles |
| 6 | In Xcode: add Network Extensions + App Groups to both targets | Xcode -> Signing & Capabilities |
| 7 | Set `NSExtensionPrincipalClass` in extension's `Info.plist` to your `NEPacketTunnelProvider` subclass | Xcode -> Extension target -> Info.plist |
| 8 | Run on **physical device only** (simulator does not support Network Extensions) | Xcode -> Device |

**Bundle ID rule:** Extension bundle ID must be prefixed with the main app's bundle ID.
- Main: `com.helix.vpn`
- Extension: `com.helix.vpn.tunnel`

### 4.5 Reference implementations

| Project | Key technique | Link |
|---|---|---|
| **WireGuard for Apple** | Production-grade NE integration; `PacketTunnelSettingsGenerator.swift` — the canonical reference for route/DNS/MTU/NAT64 configuration | `wireguard-apple` (git.zx2c4.com) |
| **SwiftyXrayKit** | Swift 6.0+ Xray proxy via NEPacketTunnelProvider; modern async/await tunnel lifecycle | Swift Package Registry |
| **URnetwork** | DeepWiki-documented production implementation; detailed memory-limit + DNS-over-HTTPS config | `urnetwork/apple` (DeepWiki) |
| **Mullvad VPN (iOS)** | Production VPN; Rust core ↔ Swift NE bridge via C-ABI | `mullvad/mullvadvpn-app` |
| **iCepa** | iOS Tor integration via NetworkExtension | `iCepa/iCepa` (GitHub) |

### 4.6 Gotchas

- **iOS memory limit (G3 gate):** ~50 MB soft cap (iOS 16+), ~15 MB (older). An OOM kill of the extension appears as the VPN silently disconnecting with no crash log in the main app. Track via Xcode Memory Debugger + Instruments Allocations template.
- **IPv6 `/120` clamp:** iOS networking stack internally clamps IPv6 prefixes to `/120`; use `/120` explicitly in `NEIPv6Settings.networkPrefixLengths`
- **`readPacketObjects` stalled:** common pitfall — using `127.0.0.1/32` as tunnel address with `0.0.0.0/0` subnet; use a valid private address (`10.0.0.2/24` or `169.254.2.1/16`) on the TUN
- **Extension restart race:** iOS may kill and restart the NE process at any time; `startTunnel` must be idempotent and restore state from shared UserDefaults (via App Group)
- **VPN Kill Switch (iOS 14+):** `includeAllNetworks` flag in `NEIPv4Settings` / `NEIPv6Settings` forces all traffic through the tunnel; setting this to `true` without a working tunnel configuration = total network loss (no fallback)
- **On-demand VPN:** `NEOnDemandRule` must be configured with `interfaceTypeMatch` and `SSIDMatch` filters or "any" to auto-connect; on-demand DNS rules are available iOS 14+
- **macOS specifics:** macOS `NEPacketTunnelProvider` requires the app to be in `/Applications/` (notarized); system extension approval via System Preferences -> Network
- **NAT64/DNS64:** IPv6-only networks (mobile carriers) require NAT64 synthesis; WireGuard's iOS implementation addresses this in `PacketTunnelSettingsGenerator.swift`

---

## 5. Windows — WFP / WinDivert

### 5.1 API overview

Windows Filtering Platform (WFP) is the kernel-level packet filtering framework on Windows (Vista+). For VPN packet interception, there are three architectural tiers:

| Tier | Mechanism | Privilege | Use case |
|---|---|---|---|
| **User-mode API** | `FwpmEngineOpen`, `FwpmFilterAdd` via BFE (Base Filtering Engine) | Administrator (UAC) | Basic firewall rules, can't intercept/inject |
| **User-mode packet interception** | **WinDivert** — third-party kernel driver (`WinDivert.sys`) + user-mode library (`WinDivert.dll`) | Administrator + signed driver | Packet capture, modification, reinjection — the most common OSS VPN path |
| **Kernel-mode callout driver** | Custom WFP callout driver (`FwpsCalloutRegister1`) + injection (`FwpsInjectNetworkSendAsync0`) | SYSTEM + WHQL/EV-signed driver | Full control — per-process filtering, ALE layers, stream inspection |

**WinDivert architecture (recommended for Phase 2):**
```
┌──────────────────────────────────────────┐
│  User-mode (Rust via UniFFI → C-ABI .dll)│
│  ┌────────────────────────────────────┐  │
│  │ WinDivert.dll bindings             │  │
│  │ WinDivertOpen(filter, layer, ...)  │  │
│  │ WinDivertRecv() / WinDivertSend()  │  │
│  └──────────┬─────────────────────────┘  │
│             │ IOCTL                       │
├─────────────┼─────────────────────────────┤
│  Kernel-mode                               │
│  ┌──────────▼─────────────────────────┐  │
│  │ WinDivert.sys (WFP callout driver) │  │
│  │ - Registers at FWPM_LAYER_*        │  │
│  │ - Intercepts packets via WFP        │  │
│  │ - Forwards to user-mode handle      │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
```

**Key API (WinDivert):**
```c
HANDLE WinDivertOpen(
    const char *filter,        // e.g. "outbound && !loopback && ip"
    WINDIVERT_LAYER layer,     // NETWORK, NETWORK_FORWARD, FLOW, SOCKET, REFLECT
    INT16 priority,            // lower = earlier intercept
    UINT64 flags               // SNIFF, DROP, RECALC_CHECKSUM, etc.
);

BOOL WinDivertRecv(HANDLE handle, PVOID pPacket, UINT packetLen,
                   PWINDIVERT_ADDRESS pAddr, UINT *pReadLen);
BOOL WinDivertSend(HANDLE handle, PVOID pPacket, UINT packetLen,
                   PWINDIVERT_ADDRESS pAddr, UINT *pWriteLen);
```

### 5.2 Integration with Rust core

**Architecture: UniFFI → C-ABI → WinDivert DLL**

The Platform shim links against `WinDivert.dll` via Rust FFI (`extern "C"` or a `windivert-sys` crate), intercepts IP packets, and feeds them through the `helix-core` transport layer. The Rust core produces encrypted WireGuard packets that are written back to the physical NIC via `WinDivertSend`.

**WinDivert rust bindings** — crates like `windivert` (or a custom `helix-windivert-sys` wrapper) provide safe Rust wrappers around the raw C API.

**Filter example:**
```
filter = "outbound && !loopback && ip && !udp.DstPort == 51820"
```
This catches all outbound non-WireGuard traffic for the kill-switch while allowing WireGuard traffic through unmolested.

### 5.3 Required permissions / elevation

- **WinDivert path:** Administrator privileges required to open the WinDivert handle; the driver must be installed (one-time) via `sc create WinDivert type= kernel start= demand binPath= <path>\WinDivert.sys`
- **Custom WFP callout driver path:** WHQL or EV certificate signing mandatory; driver installation requires Administrator + reboot or `sc start`
- **No Microsoft Store restrictions** for WinDivert-based VPNs — WinDivert is widely deployed in production tools (GoodbyeDPI, Zapret, Portmaster)
- **CVE-2026-32209 (May 2026):** A critical WFP security feature bypass was patched — ensure all target machines have KB5040526 (Win11) / KB5040525 (Win10) applied

### 5.4 Reference implementations

| Project | Key technique | Link |
|---|---|---|
| **WinDivert (basil00)** | Canonical user-mode packet interception driver; `WinDivert.sys` + `WinDivert.dll` | `basil00/WinDivert` (GitHub) |
| **Zapret (bol-van)** | DPI bypass using WinDivert on Windows; detailed WinDivert integration | `bol-van/zapret` (GitHub) |
| **Portmaster (Safing)** | Full WFP kernel extension (`PortmasterKext64.sys`); ALE Auth Connect + Stream callouts + connection cache | `safing/portmaster` (GitHub) |
| **GoodbyeDPI** | DPI circumvention via WinDivert packet modification | `ValdikSS/GoodbyeDPI` (GitHub) |
| **SplitWire-Turkey** | Split tunneling via WinDivert driver system | `cagritaskn/SplitWire-Turkey` (GitHub) |

### 5.5 Gotchas

- **WinDivert PID + reinjection limitation:** WinDivert cannot simultaneously provide the process ID AND allow packet reinjection at the `NETWORK` layer; ALE `BIND_REDIRECT` layers are needed for per-process routing, requiring a custom kernel driver (or multiple WinDivert handles at different layers)
- **Driver signing:** WinDivert ships a signed driver (`WinDivert.sys`); custom WFP callout drivers require WHQL/EV signing — a significant operational burden for Phase 2
- **Loop prevention:** VPN packets must be explicitly excluded from interception via the WinDivert filter string; missing exclusion = infinite packet loop
- **VPN conflict:** Some commercial VPNs (Check Point, NordVPN, etc.) install their own WFP filters that conflict with user-installed interceptors; detection and coexistence logic required
- **Administrator requirement:** Every WinDivert session requires administrator privileges; no unprivileged user-mode interception path exists on Windows
- **WFP filter insertion order:** Multiple WFP filters at the same layer compete by priority (weight); a higher-priority commercial VPN filter can shadow the helix one
- **IPv6 on WFP:** IPv6 WFP layers use separate GUIDs; filter strings must handle both `ip` (v4) and `ipv6`

---

## 6. Linux nftables (existing) — Current State

### 6.1 Current rig architecture

**Location:** `scripts/rig/setup.sh` + `scripts/rig/common.sh`

**Top-level topology:**
```
hx-client (10.0.240.2/24)          hx-server (10.0.240.3/24)
    │ veth-c                            │ veth-s
    │                                   │
    └──────────────┬────────────────────┘
                   │
              hx-bridge (10.0.240.1/24)
              bridge: br0
              veth-c-br, veth-s-br
```

**Current nftables ruleset** (per namespace, `inet filter`):
- `input` chain: `type filter hook input priority 0; policy accept;`
- `forward` chain: `type filter hook forward priority 0; policy accept;`
- `output` chain: `type filter hook output priority 0; policy accept;`

This is a **baseline default-accept** configuration — no kill-switch, no DNS-leak prevention, no port-level verdict maps. Tests install their own policies on top.

### 6.2 What Phase 1 adds (from the WBS)

- **HVPN-P1-101 (Kill-switch module):** Atomic nftables rule swap between "VPN-up" (allow-only-tunnel) and "VPN-down" (default-deny output) states; no plaintext egress, no DNS leak
- **HVPN-P1-102 (Verdict-map enforcement):** Port-level allow/deny beyond CIDR-only WireGuard `AllowedIPs`; drop revoked device packets within <1 second
- **HVPN-P1-120 (Linux tun shim):** TUN device => transport traits => WireGuard; reference platform for netns E2E testing

### 6.3 Integration with helix_core

The Linux shim is the simplest integration: `helix-tun` crate provides `TunDevice` (raw IP packet read/write via tokio blocking I/O), `helix-wg` provides the WireGuard protocol wrapper via `boringtun`, and nftables is driven programmatically from the orchestrator (`helix-orch`). No FFI boundary — pure Rust.

### 6.4 Gotchas (from the deep research)

- **Reply-packet loops:** TPROXY scenarios need `type route` output chain + `meta mark set 0` to prevent proxy return packets from re-entering the capture chain (hysteria2 case study)
- **Mode mismatch:** nftables `REDIRECT` requires the proxy to use transparent mode (`SO_ORIGINAL_DST`)
- **Teardown atomicity:** Flushing the entire chain instead of removing specific rules creates a plaintext window
- **cgroupv2 compatibility:** pre-4.15 kernels lack cgroupv2 matching; fall back to UID or namespace isolation
- **Conntrack interaction:** nftables `redirect` interacts with connection tracking; stale conntrack entries can route packets around kill-switch rules after a reconnect

---

## 7. Cross-platform FFI — Unified approach

### 7.1 Recommended FFI stack: UniFFI

The project should standardize on **Mozilla UniFFI** (v0.31.x, actively maintained April 2026) as the single FFI definition layer:

```
helix-core (Rust) ── UniFFI UDL ──┬── Kotlin bindings (Android)
                                  ├── Swift bindings (iOS/macOS)
                                  ├── Python bindings (Linux testing)
                                  └── C-ABI fallback (Windows, Go bridge)
```

**Why UniFFI over raw JNI + raw C-ABI:**

| Concern | Raw JNI/C-ABI | UniFFI |
|---|---|---|
| Kotlin type mapping | Manual `env->GetByteArrayElements` | Automatic: `Vec<u8>` -> `List<Byte>` |
| Swift type mapping | Manual `UnsafePointer<UInt8>` | Automatic: `Vec<u8>` -> `Data` |
| Error propagation | Manual `jni::errors::*`, `NSError` | Automatic: `Result<T,E>` -> Kotlin exceptions / Swift `throws` |
| Async bridge | Manual channel + callback | Automatic: `async fn` -> Kotlin `suspend` / Swift `async throws` |
| Reference counting | Manual `Arc` + `JNIEnv::new_global_ref` | Automatic via UniFFI's handle map |
| Cross-platform drift | One platform can silently diverge | Single UDL is the contract — divergence = compile error |

**Escape hatch:** Raw JNI (`jni` crate) and raw C-ABI (`extern "C"`) remain available for performance-critical hot paths (packet I/O loops) where UniFFI's per-call overhead (handle-map lookup) is measurable. The `defguard_boringtun` crate (Feb 2026) demonstrates the dual approach: UniFFI for the primary API surface, raw JNI for the packet path.

### 7.2 UniFFI definition sketch for helix-core

```
// Proposed helix_core.udl (UniFFI Interface Definition)
namespace helix_core {
    // Core types
    TransportKind enum { "PlainUdp", "MasqueH3", "WireGuard" };
};

dictionary TunnelConfig {
    TransportKind transport;
    string listen_addr;
    string peer_endpoint;
    // ...WG keys, DNS, routes...
};

interface Tunnel {
    [Throws=HelixError]
    constructor(TunnelConfig config);

    [Throws=HelixError]
    void connect();

    [Throws=HelixError]
    sequence<u8> send_packet(sequence<u8> packet);

    [Throws=HelixError]
    sequence<u8> recv_packet();

    void disconnect();

    TunnelStatus status();
};

[Error]
enum HelixError {
    "ConfigInvalid",
    "ConnectionFailed",
    "TransportError",
    "KillSwitchEngaged",
};
```

---

## 8. Platform adapter implementation plan — architecture sketch

### 8.1 Common pattern across all platforms

```
┌──────────────────────────────────────────┐
│  Platform-independent Rust core          │
│  ┌────────────────────────────────────┐  │
│  │ helix_core::Tunnel                  │  │
│  │ helix_transport::TransportRegistry  │  │
│  │ helix_wg::WireGuardTunnel           │  │
│  └────────────────────────────────────┘  │
│                  │ UniFFI                 │
├──────────────────┼────────────────────────┤
│  Platform shim   │ (per-platform layer)   │
│                  ▼                         │
│  ┌────────────────────────────────────┐  │
│  │ Packet I/O bridge                  │  │
│  │ (TUN fd → send_packet/recv_packet) │  │
│  │ Kill-switch adapter                │  │
│  │ DNS-leak adapter                   │  │
│  ├────────────────────────────────────┤  │
│  │ Platform VPN API                   │  │
│  │ Linux: TUN + nftables              │  │
│  │ Android: VpnService + JNI          │  │
│  │ iOS: NEPacketTunnelProvider        │  │
│  │ Windows: WinDivert                 │  │
│  └────────────────────────────────────┘  │
└──────────────────────────────────────────┘
```

### 8.2 Per-platform packet I/O bridge

| Platform | TUN primitive | Read direction | Write direction |
|---|---|---|---|
| **Linux** | `TunDevice` (helix-tun) | `tokio::spawn_blocking(|| tun.read(buf))` | `tokio::spawn_blocking(|| tun.write(buf))` |
| **Android** | `ParcelFileDescriptor` | `FileInputStream.read(buf)` on dedicated thread | `FileOutputStream.write(buf)` on same thread |
| **iOS** | `NEPacketTunnelFlow` | `readPacketObjects(completionHandler:)` array of `NEPacket` | `writePacketObjects(_:completionHandler:)` array of `Data` |
| **Windows** | WinDivert handle | `WinDivertRecv(handle, buf, ...)` | `WinDivertSend(handle, buf, ...)` |

### 8.3 Kill-switch mapping

| Platform | Mechanism | Atomicity | Leak risk |
|---|---|---|---|
| **Linux** | nftables default-deny output chain + tunnel-allow exceptions | Atomic rule swap (`nft -f ruleset.nft`) | Low — kernel-level enforcement |
| **Android** | `VpnService` always-on + lock-down mode | System-enforced | Low — system-enforced, but user can disable in Settings |
| **iOS** | `includeAllNetworks = true` on `NEIPv4Settings`/`NEIPv6Settings` | System-enforced | Low — but misconfigured tunnel = total network loss |
| **Windows** | WinDivert filter + DROP verdict | Single filter string | Medium — filter order can be subverted by higher-priority WFP filters |

---

## 9. Summary table

| Platform | Phase | API | FFI approach | Permissions | Provisioning | Reference impl | Key risk |
|---|---|---|---|---|---|---|---|
| **Linux** | 1 | TUN + nftables | Direct Rust | root (namespaces + nftables) | iproute2, nftables | burpwn, cgtproxy, Xray TProxy | None — reference platform |
| **Android** | 1 | VpnService (API 14+) | UniFFI → Kotlin | INTERNET, BIND_VPN_SERVICE, FOREGROUND_SERVICE | None beyond app install | WireGuard, Mullvad, vpn-frontend, NetGuard | VpnService thread model, Doze mode, single-VPN-at-a-time |
| **iOS** | 1 | NEPacketTunnelProvider (iOS 9+) | UniFFI → Swift staticlib | `packet-tunnel-provider` NetworkExtension entitlement + App Groups | 2 App IDs + 2 provisioning profiles + physical device only | WireGuard-Apple, SwiftyXrayKit, URnetwork, Mullvad | Memory ceiling G3 gate (~50 MB iOS 16+), extension lifecycle |
| **macOS** | 2 | NEPacketTunnelProvider (macOS 10.15+) | UniFFI → Swift staticlib | Same as iOS + `com.apple.developer.networking.vpn.api` | Notarized app in /Applications | WireGuard-Apple, Mullvad macOS | System Extension approval UX |
| **Windows** | 2 | WFP / WinDivert | UniFFI → C-ABI .dll | Administrator (UAC) + signed driver | WinDivert.sys installation | WinDivert, Zapret, Portmaster, GoodbyeDPI | Driver signing, filter priority conflicts, no unprivileged path |

---

## 10. Risks and mitigation

| Risk | Severity | Platform | Mitigation |
|---|---|---|---|
| **iOS memory G3 gate fails** | CRITICAL | iOS | Apply §6.4 fallback ladder: jemalloc, minimize buffers, stream packets, release read semaphore; operator decides only if all four exhausted |
| **WinDivert PID + reinjection limitation** | MEDIUM | Windows | Accept the limitation for Phase 2; document that per-app routing requires custom WFP callout driver |
| **Android Doze kills VPN thread** | MEDIUM | Android | Use PARTIAL_WAKE_LOCK for sustained transfers; foreground service notification keeps process priority elevated |
| **iOS extension OOM kill is silent** | MEDIUM | iOS | Implement persistent status indicator via App Group UserDefaults; main app polls extension health |
| **WFP filter priority conflict** | LOW | Windows | Detect coexistence via `FwpmFilterGetByKey0`; use lowest-possible priority; document conflict with commercial VPNs |
| **nftables conntrack stale entries** | LOW | Linux | Flush conntrack on tunnel state change (`conntrack -D`); use `notrack` for VPN traffic to avoid conntrack entirely |
| **UniFFI per-call overhead in hot path** | LOW | All mobile | Escape hatch: raw JNI/C-ABI for the packet send/recv loop; UniFFI for everything else (config, status, lifecycle) |

---

## Sources verified

| Source | URL | Verified |
|---|---|---|
| Android VpnService API Reference | `https://developer.android.com/reference/android/net/VpnService` | 2026-07-08 |
| Android VpnService.Builder API Reference | `https://developer.android.com/reference/android/net/VpnService.Builder` | 2026-07-08 |
| Apple Packet Tunnel Provider Documentation | `https://developer.apple.com/documentation/networkextension/packet-tunnel-provider` | 2026-07-08 |
| Apple Developer Forums — NEPacketTunnelProvider Entitlement (Quinn "The Eskimo!") | `https://developer.apple.com/forums/thread/807080` | 2026-07-08 |
| Microsoft WFP Sample (WFPSampler) | `https://learn.microsoft.com/samples/microsoft/windows-driver-samples/windows-filtering-platform-sample/` | 2026-07-08 |
| WinDivert — basil00/WinDivert | `https://github.com/basil00/WinDivert` | 2026-07-08 |
| Portmaster Windows Implementation (DeepWiki) | `https://deepwiki.com/safing/portmaster/3.2-windows-implementation` | 2026-07-08 |
| vpn-frontend — Rust/JNI WireGuard on Android | `https://github.com/ian52n/vpn-frontend` (Jan 2026) | 2026-07-08 |
| Mullvad VPN Tunnel Implementation (DeepWiki) | `https://deepwiki.com/mullvad/mullvadvpn-app/3.1-tunnel-implementation` | 2026-07-08 |
| defguard_boringtun — UniFFI + JNI dual FFI | `https://lib.rs/crates/defguard_boringtun` (v0.6.5, Feb 2026) | 2026-07-08 |
| UniFFI — Mozilla's cross-platform FFI toolkit | `https://lib.rs/crates/uniffi` (v0.31.1, Apr 2026) | 2026-07-08 |
| UniFFI Build System Integration (DeepWiki) | `https://deepwiki.com/mozilla/uniffi-rs/7.3-build-system-integration` | 2026-07-08 |
| SwiftyXrayKit — Modern NEPacketTunnelProvider Swift 6.0 | `https://swiftpackageregistry.com/dima-u/SwiftyXrayKit` | 2026-07-08 |
| URnetwork Packet Tunnel Provider (DeepWiki) | `https://deepwiki.com/urnetwork/apple/4.1-packet-tunnel-provider` | 2026-07-08 |
| WireGuard Apple — PacketTunnelSettingsGenerator.swift | `https://git.zx2c4.com/wireguard-apple/` | 2026-07-08 |
| burpwn — Transparent proxy + nftables + network namespaces | `https://github.com/own2pwn-fr/burpwn` (v0.1.1, Jun 2026) | 2026-07-08 |
| cgtproxy — cgroupv2 dynamic TPROXY with nftables (DeepWiki) | `https://deepwiki.com/black-desk/cgtproxy/5.6-nftables-rule-structure` | 2026-07-08 |
| Xray TProxy IPv4+IPv6 Configuration (XTLS Official) | `https://xtls.github.io/en/document/level-2/tproxy_ipv4_and_ipv6.html` | 2026-07-08 |
| Transparent Proxy + Hysteria2 on Same Machine (asharca) | `https://asharca.github.io/hysteria2-tproxy-reply-hijack/` | 2026-07-08 |
| CVE-2026-32209 — WFP Security Feature Bypass | `https://windowsnews.ai/article/cve-2026-32209-microsoft-patches-critical-windows-filtering-platform-bypass-in-may-2026-patch-tuesda.417969` | 2026-07-08 |
| WHQL-Signed WFP Kernel Backdoor (wskmon.sys) — Nextron Systems | `https://www.nextron-systems.com/2026/06/26/anatomy-of-a-whql-signed-windows-filtering-platform-wfp-kernel-resident-network-backdoor/` | 2026-07-08 |

*Gaps/silences: UniFFI examples for the exact `helix_core` API surface are the project's own implementation work — no external template exists for this specific transport abstraction. Windows WFP custom callout driver implementation details are deferred to Phase 2 deep-research (§11.4.8). macOS NEPacketTunnelProvider entitlements for App Store distribution require additional Apple Developer Program verification at provisioning time — documented in the checklist but not pre-verified for this project's specific bundle IDs.*
