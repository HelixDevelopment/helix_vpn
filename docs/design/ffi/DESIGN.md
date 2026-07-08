# Helix VPN -- FFI Surface Design (G5 Gate, P1-005)

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Authority:** Helix VPN Phase 1 FFI design, G5 gate deliverable
**Scope:** Defines the UniFFI FFI boundary for `helix_core` -- the UDL contract, per-platform binding sketches, data-plane bridge, build pipeline, and test strategy

---

## 1. FFI Architecture Overview

### 1.1 Layer diagram

```
+------------------------------------------------------------+
|  Platform-specific UI / VPN service                        |
|  +---------------------------+  +------------------------+ |
|  | Android: Kotlin/VpnService|  | iOS: Swift/NEPacketTP  | |
|  | Linux: Rust direct        |  | Windows: C-ABI .dll    | |
|  +------------+--------------+  +-----------+------------+ |
+---------------|----------------------------|--------------+
                | UniFFI-generated bindings  |
+---------------|----------------------------|--------------+
|  helix-core-ffi  (new crate: crates/helix-ffi/)           |
|  +------------------------------------------------------+ |
|  | helix_core.udl  (single UDL contract)                 | |
|  | ffi_tunnel.rs   (Rust impl of UDL interface types)    | |
|  | ffi_events.rs   (callback interface wiring)           | |
|  | ffi_dataplane.rs (hot-path C-ABI escape hatch)        | |
|  +------------------------------------------------------+ |
+------------------------------------------------------------+
|  helix_core (existing workspace)                           |
|  +------------------+ +-------------+ +-----------------+ |
|  | helix-transport  | | helix-wg    | | helix-orch      | |
|  | (Transport trait)| | (boringtun) | | (Orchestrator)  | |
|  +------------------+ +-------------+ +-----------------+ |
|  | helix-tun        | | helix-masque|                   | |
|  +------------------+ +-------------+                   | |
+------------------------------------------------------------+
```

The FFI crate (`helix-ffi`) is a **new workspace member** that sits between the platform bindings and the existing `helix_core` crates. It does NOT expose the internal `Transport`/`Connection`/`Listener` trait hierarchy directly through FFI -- those are Rust-internal abstractions with `async_trait`, `Box<dyn Connection>`, and `Arc<dyn Transport>` that cannot cross the UniFFI boundary. Instead, `helix-ffi` wraps the orchestrator and transport layers behind a higher-level, concrete-type FFI surface that UniFFI can bind.

### 1.2 Design philosophy

1. **One UDL, four platforms** -- a single `helix_core.udl` generates Kotlin (Android), Swift (iOS/macOS), Python (Linux testing), and Ruby bindings with zero per-platform divergence in the definition.
2. **Callback pattern for async** -- UniFFI v0.28+ supports async callback interfaces via a VTable pattern. Events (tunnel state changes, errors, stats) flow from Rust to the platform layer through a callback interface implemented in the foreign language.
3. **Blocking wrapper for lifecycle** -- `start`, `connect`, `disconnect` are inherently async in the orchestrator but exposed as synchronous methods that internally spawn tokio tasks and return immediately; the platform layer observes results through the callback interface.
4. **Raw C-ABI escape hatch for data plane** -- the hot path (packet send/recv through TUN) uses direct `extern "C"` functions to avoid UniFFI's per-call handle-map lookup overhead. UniFFI handles config, lifecycle, and events; raw C-ABI handles the packet loop.
5. **No trait objects across FFI** -- `Arc<dyn Transport>`, `Box<dyn Connection>`, and `Box<dyn Listener>` are Rust-internal; the FFI surface uses only concrete enums (`TransportKind`) and opaque handle types (`TunnelHandle`).

---

## 2. UDL Definition

### 2.1 `helix_core.udl`

```webidl
// =========================================================================
// helix_core.udl -- UniFFI Interface Definition for Helix VPN Core
//
// Single UDL contract for all platforms: Kotlin, Swift, Python, Ruby.
// Generated via: uniffi-bindgen generate src/helix_core.udl --language <lang>
// =========================================================================

namespace helix_core {
    // Top-level version probe -- callable without creating a tunnel.
    string version();
};

// -------------------------------------------------------------------------
// Transport kind -- closed set matching Transport::kind() labels
// -------------------------------------------------------------------------

enum TransportKind {
    "PlainUdp",
    "MasqueQuicStandin",
};

// -------------------------------------------------------------------------
// Tunnel configuration -- one-way (Rust -> platform) data
// -------------------------------------------------------------------------

dictionary TunnelConfig {
    /// VPN server address (host:port or IP:port).
    string server_addr;

    /// Transport protocol to use. Resolved against the built-in registry.
    /// Valid values: "plain-udp", "masque-quic-standin".
    string transport_name;

    /// Local UDP listen port (0 = OS-assigned random port).
    u16 listen_port;

    /// WireGuard private key (64 hex characters).
    /// MUST be provided via environment variable, never hardcoded.
    /// The FFI layer reads HELIX_WG_PRIVATE_KEY from the environment
    /// before constructing the tunnel -- this value is NEVER stored
    /// or transmitted in plaintext through the FFI boundary.
    /// If unset, an ephemeral keypair is generated.
    string? private_key_hex;

    /// Statistics polling interval, in seconds.
    u32 stats_interval_secs;

    /// Keepalive interval, in seconds (0 = disabled).
    u32 keepalive_interval_secs;
};

// -------------------------------------------------------------------------
// Tunnel status -- one-way (Rust -> platform) snapshot data
// -------------------------------------------------------------------------

dictionary TunnelStatus {
    /// Current lifecycle state label (e.g. "Connected", "Idle", "Error").
    string state_label;

    /// Total bytes sent through the tunnel.
    u64 bytes_sent;

    /// Total bytes received through the tunnel.
    u64 bytes_recv;

    /// Session duration in seconds.
    u64 session_duration_secs;

    /// Last measured RTT to the server, in milliseconds.
    u16 latency_ms;

    /// Download speed (bytes/sec, rolling 5s window).
    f64 download_speed;

    /// Upload speed (bytes/sec, rolling 5s window).
    f64 upload_speed;

    /// Currently active transport name.
    string active_transport;
};

// -------------------------------------------------------------------------
// Tunnel error -- closed-set error type
// -------------------------------------------------------------------------

[Error]
enum TunnelError {
    /// No transport configured (call start/connect before setting transport).
    "NoTransport",

    /// The connection attempt to the server failed.
    "ConnectionFailed",

    /// The tunnel is already connected (connect called twice).
    "AlreadyConnected",

    /// A transport-layer error occurred (I/O, closed, address in use, etc.).
    "TransportError",

    /// The WireGuard handshake timed out.
    "HandshakeTimeout",

    /// The WireGuard Noise IK handshake failed (crypto/protocol error).
    "WgHandshake",

    /// Invalid configuration (bad address, unknown transport, bad key format).
    "ConfigInvalid",

    /// The tunnel handle was already released (use-after-free).
    "HandleReleased",

    /// Catch-all for unexpected internal errors.
    "Internal",
};

// -------------------------------------------------------------------------
// Tunnel lifecycle event -- delivered through callback interface
// -------------------------------------------------------------------------

dictionary TunnelEventData {
    /// Event kind discriminator (matches the event enum variant names).
    string kind;

    /// Human-readable label (for UI display).
    string label;

    /// Optional additional data, JSON-encoded.
    /// Carries variant-specific fields:
    ///   StateChanged -> {"state": "<label>"}
    ///   Connected    -> {"transport": "<name>", "rtt_ms": <u64>}
    ///   Down         -> {"reason": "<string>"}
    ///   Established  -> {"server": "<addr>", "latency_ms": <u16>}
    ///   Error        -> {"source": "<string>", "message": "<string>"}
    ///   StatsUpdate  -> TunnelStatus fields as JSON
    string? data_json;
};

// -------------------------------------------------------------------------
// Callback interface -- platform layer implements this trait
// -------------------------------------------------------------------------

callback interface TunnelEventCallback {
    /// Called by Rust whenever a tunnel lifecycle event occurs.
    /// The foreign implementation MUST be thread-safe (Send + Sync).
    /// Called from a tokio runtime worker thread -- the implementation
    /// should hand off to the main/platform thread as needed.
    [Async]
    void on_event(TunnelEventData event);
};

// -------------------------------------------------------------------------
// Tunnel handle -- the primary FFI object
// -------------------------------------------------------------------------

interface Tunnel {
    /// Create a new tunnel with the given configuration and event callback.
    ///
    /// The tunnel starts in the Idle state. No background loops run yet.
    /// The callback is invoked on tunnel lifecycle events (state changes,
    /// errors, stats updates).
    ///
    /// Errors: ConfigInvalid if the transport name is unknown or the
    /// server address is malformed.
    [Throws=TunnelError]
    constructor(TunnelConfig config, TunnelEventCallback callback);

    /// Start the tunnel orchestrator background loops.
    ///
    /// Spawns three tokio tasks (stats, connection, keepalive) and returns
    /// immediately. The callback receives StateChanged events as the tunnel
    /// transitions through its lifecycle.
    ///
    /// Errors: NoTransport (internal -- should not happen if config was valid).
    [Throws=TunnelError]
    void start();

    /// Initiate a connection to the configured server.
    ///
    /// Transitions through Connecting -> Handshaking -> Connected (on success)
    /// or -> Down (on failure). Each transition fires a callback event.
    ///
    /// Errors: NoTransport, ConnectionFailed.
    [Throws=TunnelError]
    void connect();

    /// Disconnect from the server (graceful teardown).
    ///
    /// Transitions through Disconnecting -> Disconnected. The callback
    /// receives Down with reason "user initiated".
    [Throws=TunnelError]
    void disconnect();

    /// Get the current tunnel status snapshot.
    ///
    /// Returns the most recent stats and state label. Non-blocking.
    TunnelStatus status();

    /// Release all resources associated with this tunnel.
    ///
    /// After release, all background loops stop, the callback is
    /// unregistered, and further calls on this handle return
    /// TunnelError::HandleReleased.
    void release();
};

// =========================================================================
// Hot-path data plane (C-ABI escape hatch -- NOT in UDL)
//
// The following functions are exposed as raw extern "C" to avoid
// UniFFI's per-call handle-map lookup overhead on every packet.
// They are documented here for completeness but defined in
// ffi_dataplane.rs and called directly through C-ABI in the
// platform adapter's native code (JNI, Swift C interop, ctypes).
//
//   uint32_t helix_send_packet(uint64_t tunnel_handle,
//                              const uint8_t *packet, uint32_t len);
//   uint32_t helix_recv_packet(uint64_t tunnel_handle,
//                              uint8_t *buf, uint32_t buf_capacity);
//   uint32_t helix_tunnel_mtu(uint64_t tunnel_handle);
//
// Returns: number of bytes sent/received, or 0 on error/timeout.
// The tunnel_handle is obtained from Tunnel::handle() (a u64
// opaque identifier exposed through the UDL Tunnel interface).
// =========================================================================
```

### 2.2 Type mapping table

| Rust (internal) | UDL type | Kotlin (generated) | Swift (generated) | Python (generated) |
|---|---|---|---|---|
| `TransportKind` enum | `enum TransportKind` | `sealed class TransportKind` | `enum TransportKind` | `class TransportKind(enum.Enum)` |
| `OrchConfig` + extras | `dictionary TunnelConfig` | `data class TunnelConfig` | `struct TunnelConfig` | `class TunnelConfig` |
| `TunnelState` + `TunnelStats` | `dictionary TunnelStatus` | `data class TunnelStatus` | `struct TunnelStatus` | `class TunnelStatus` |
| `OrchError` + extras | `[Error] enum TunnelError` | `sealed class TunnelException : Exception` | `enum TunnelError : Error` | `class TunnelError(Exception)` |
| `TunnelEvent` (serialized) | `dictionary TunnelEventData` | `data class TunnelEventData` | `struct TunnelEventData` | `class TunnelEventData` |
| `EventBus::subscribe` loop | `callback interface TunnelEventCallback` | `interface TunnelEventCallback` | `protocol TunnelEventCallback` | `class TunnelEventCallback(abc.ABC)` |
| `Orchestrator` wrapper | `interface Tunnel` | `class Tunnel : AutoCloseable` | `class Tunnel` | `class Tunnel` |
| `Option<PrivateKey>` | `string? private_key_hex` | `String?` | `String?` | `Optional[str]` |

### 2.3 UDL constraints (verified from UniFFI docs)

| Constraint | Implication for helix_core.udl |
|---|---|
| One UDL per crate | All types in `helix_core.udl`; no splitting across files |
| Namespace must match crate name | `namespace helix_core { };` must match `[lib]` name in `Cargo.toml` |
| No `Arc<dyn Trait>` over FFI | Use concrete enum (`TransportKind`) + opaque handle (`Tunnel`) instead |
| No `async_trait` over FFI | Expose synchronous wrapper methods; async handled internally via tokio |
| Callback methods can be `[Async]` | `on_event` is `[Async]` so foreign side can dispatch to main thread |
| Error variants are flat strings | `TunnelError` variants are identifier strings, not rich data |
| Callback impls must be `Send + Sync` | Platform adapter must ensure thread-safety of callback code |
| VTable init before any instances | Foreign code MUST call the generated `init` function before creating a `Tunnel` |

---

## 3. Per-Platform Binding Sketch

### 3.1 Kotlin (Android)

**Generated code shape:**

```kotlin
// Generated by uniffi-bindgen from helix_core.udl
// Package: com.helix.vpn.bridge

sealed class TransportKind {
    object PlainUdp : TransportKind()
    object MasqueQuicStandin : TransportKind()
}

data class TunnelConfig(
    val serverAddr: String,
    val transportName: String,
    val listenPort: Short,
    val privateKeyHex: String?,
    val statsIntervalSecs: Int,
    val keepaliveIntervalSecs: Int,
)

data class TunnelStatus(
    val stateLabel: String,
    val bytesSent: Long,
    val bytesRecv: Long,
    val sessionDurationSecs: Long,
    val latencyMs: Short,
    val downloadSpeed: Double,
    val uploadSpeed: Double,
    val activeTransport: String,
)

data class TunnelEventData(
    val kind: String,
    val label: String,
    val dataJson: String?,
)

sealed class TunnelException(message: String) : Exception(message) {
    class NoTransport : TunnelException("no transport configured")
    class ConnectionFailed(cause: String) : TunnelException(cause)
    // ... etc.
}

// Platform adapter implements this interface:
interface TunnelEventCallback {
    suspend fun onEvent(event: TunnelEventData)
}

class Tunnel internal constructor(handle: Long) : AutoCloseable {
    // Companion object for constructor:
    companion object {
        suspend fun create(config: TunnelConfig, callback: TunnelEventCallback): Tunnel
    }

    suspend fun start()
    suspend fun connect()
    suspend fun disconnect()
    fun status(): TunnelStatus
    override fun close()  // calls release()
    fun handle(): Long    // for C-ABI data-plane escape hatch
}
```

**Platform adapter integration (VpnService):**

```kotlin
class HelixVpnService : VpnService(), TunnelEventCallback {
    private var tunnel: Tunnel? = null
    private var tunFd: ParcelFileDescriptor? = null
    private var packetThread: Thread? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        // 1. Build TunnelConfig from stored preferences
        val config = TunnelConfig(
            serverAddr = "vpn.example.com:51820",
            transportName = "plain-udp",
            listenPort = 0,
            privateKeyHex = null,  // read from env, not passed
            statsIntervalSecs = 5,
            keepaliveIntervalSecs = 25,
        )

        // 2. Create tunnel with this as callback
        tunnel = Tunnel.create(config, this)

        // 3. Create VpnService TUN
        val builder = Builder()
            .setSession("Helix VPN")
            .setMtu(tunnel!!.handle().let { helixTunnelMtu(it).toInt() }) // C-ABI
            .addAddress("10.0.0.2", 32)
            .addRoute("0.0.0.0", 0)
            .addDnsServer("1.1.1.1")
        tunFd = builder.establish()!!

        // 4. Start orchestrator, then connect
        CoroutineScope(Dispatchers.IO).launch {
            tunnel!!.start()
            tunnel!!.connect()
        }

        // 5. Start packet I/O thread (C-ABI hot path)
        packetThread = Thread { runPacketLoop(tunFd!!, tunnel!!.handle()) }
        packetThread!!.start()

        startForeground(NOTIFICATION_ID, buildNotification())
        return START_STICKY
    }

    // C-ABI hot path (not through UniFFI):
    private external fun helixSendPacket(handle: Long, packet: ByteArray, len: Int): Int
    private external fun helixRecvPacket(handle: Long, buf: ByteArray, capacity: Int): Int
    private external fun helixTunnelMtu(handle: Long): Int

    private fun runPacketLoop(fd: ParcelFileDescriptor, handle: Long) {
        val input = FileInputStream(fd.fileDescriptor)
        val output = FileOutputStream(fd.fileDescriptor)
        val buf = ByteArray(2048)
        val outBuf = ByteArray(2048)
        while (!Thread.interrupted()) {
            val n = input.read(buf)
            if (n > 0) {
                val sent = helixSendPacket(handle, buf, n)
                if (sent > 0) {
                    val recv = helixRecvPacket(handle, outBuf, outBuf.size)
                    if (recv > 0) output.write(outBuf, 0, recv)
                }
            }
        }
    }

    // TunnelEventCallback implementation:
    override suspend fun onEvent(event: TunnelEventData) {
        withContext(Dispatchers.Main) {
            when (event.kind) {
                "connected" -> updateNotification("Connected")
                "down" -> updateNotification("Disconnected")
                "stats_update" -> { /* update UI */ }
                "error" -> handleError(event)
            }
        }
    }
}
```

### 3.2 Swift (iOS/macOS)

**Generated code shape:**

```swift
// Generated by uniffi-bindgen from helix_core.udl

public enum TransportKind {
    case plainUdp
    case masqueQuicStandin
}

public struct TunnelConfig {
    public var serverAddr: String
    public var transportName: String
    public var listenPort: UInt16
    public var privateKeyHex: String?
    public var statsIntervalSecs: UInt32
    public var keepaliveIntervalSecs: UInt32
}

public struct TunnelStatus {
    public var stateLabel: String
    public var bytesSent: UInt64
    public var bytesRecv: UInt64
    public var sessionDurationSecs: UInt64
    public var latencyMs: UInt16
    public var downloadSpeed: Double
    public var uploadSpeed: Double
    public var activeTransport: String
}

public struct TunnelEventData {
    public var kind: String
    public var label: String
    public var dataJson: String?
}

public enum TunnelError: Error {
    case noTransport
    case connectionFailed(String)
    case alreadyConnected
    case transportError(String)
    case handshakeTimeout
    case wgHandshake(String)
    case configInvalid(String)
    case handleReleased
    case `internal`(String)
}

// Platform adapter conforms to this protocol:
public protocol TunnelEventCallback: AnyObject {
    func onEvent(event: TunnelEventData) async
}

public class Tunnel {
    public static func create(config: TunnelConfig,
                              callback: TunnelEventCallback) async throws -> Tunnel
    public func start() async throws
    public func connect() async throws
    public func disconnect() async throws
    public func status() -> TunnelStatus
    public func release() throws
    public func handle() -> UInt64  // for C-ABI escape hatch
}
```

**Platform adapter integration (NEPacketTunnelProvider):**

```swift
class HelixTunnelProvider: NEPacketTunnelProvider, TunnelEventCallback {
    private var tunnel: Tunnel?
    private var packetTask: Task<Void, Never>?

    override func startTunnel(options: [String: NSObject]?,
                              completionHandler: @escaping (Error?) -> Void) {
        Task {
            do {
                // 1. Build config from protocolConfiguration
                let proto = self.protocolConfiguration as! NETunnelProviderProtocol
                let config = TunnelConfig(
                    serverAddr: proto.serverAddress!,
                    transportName: "plain-udp",
                    listenPort: 0,
                    privateKeyHex: nil,  // from env
                    statsIntervalSecs: 5,
                    keepaliveIntervalSecs: 25
                )

                // 2. Create tunnel
                tunnel = try await Tunnel.create(config: config, callback: self)

                // 3. Configure TUN
                let settings = NEPacketTunnelNetworkSettings(
                    tunnelRemoteAddress: "10.0.0.1"
                )
                let ipv4 = NEIPv4Settings(
                    addresses: ["10.0.0.2"],
                    subnetMasks: ["255.255.255.0"]
                )
                ipv4.includedRoutes = [NEIPv4Route.default()]
                settings.ipv4Settings = ipv4
                settings.mtu = NSNumber(value: tunnel!.handle())
                    .let { helixTunnelMtu($0.uint64Value) } // C-ABI

                try await self.setTunnelNetworkSettings(settings)

                // 4. Start + connect
                try await tunnel!.start()
                try await tunnel!.connect()

                // 5. Start packet loop (C-ABI hot path)
                packetTask = Task { [weak self] in
                    await self?.runPacketLoop()
                }

                completionHandler(nil)
            } catch {
                completionHandler(error)
            }
        }
    }

    private func runPacketLoop() async {
        guard let tunnel = tunnel else { return }
        let handle = tunnel.handle()
        var buf = Data(count: 2048)
        var outBuf = Data(count: 2048)

        while !Task.isCancelled {
            let packets = await packetFlow.readPacketObjects()
            for packet in packets {
                let n = packet.data.withUnsafeBytes { ptr in
                    helixSendPacket(handle, ptr.baseAddress!, UInt32(packet.data.count))
                }
                if n > 0 {
                    let recv = outBuf.withUnsafeMutableBytes { ptr in
                        helixRecvPacket(handle, ptr.baseAddress!, UInt32(outBuf.count))
                    }
                    if recv > 0 {
                        let outPacket = NEPacket(data: outBuf.prefix(Int(recv)),
                                                  protocolFamily: .ipv4)
                        await packetFlow.writePacketObjects([outPacket])
                    }
                }
            }
        }
    }

    // TunnelEventCallback -- async, dispatched to NE process main queue
    func onEvent(event: TunnelEventData) async {
        switch event.kind {
        case "connected":
            NSLog("[Helix] Connected: \(event.dataJson ?? "")")
        case "down":
            self.cancelTunnelWithError(nil)
        case "stats_update":
            // Update shared UserDefaults for main app to poll
            break
        default:
            break
        }
    }
}
```

### 3.3 Python (Linux testing/CLI)

**Generated code shape:**

```python
# Generated by uniffi-bindgen from helix_core.udl

from enum import Enum
from dataclasses import dataclass
from typing import Optional
import asyncio

class TransportKind(Enum):
    PLAIN_UDP = 1
    MASQUE_QUIC_STANDIN = 2

@dataclass
class TunnelConfig:
    server_addr: str
    transport_name: str
    listen_port: int
    private_key_hex: Optional[str]
    stats_interval_secs: int
    keepalive_interval_secs: int

@dataclass
class TunnelStatus:
    state_label: str
    bytes_sent: int
    bytes_recv: int
    session_duration_secs: int
    latency_ms: int
    download_speed: float
    upload_speed: float
    active_transport: str

@dataclass
class TunnelEventData:
    kind: str
    label: str
    data_json: Optional[str]

class TunnelError(Exception):
    pass
# Sub-exceptions: TunnelError.NoTransport, .ConnectionFailed, etc.

class TunnelEventCallback(abc.ABC):
    @abc.abstractmethod
    async def on_event(self, event: TunnelEventData):
        ...

class Tunnel:
    @staticmethod
    async def create(config: TunnelConfig,
                     callback: TunnelEventCallback) -> "Tunnel":
        ...

    async def start(self): ...
    async def connect(self): ...
    async def disconnect(self): ...
    def status(self) -> TunnelStatus: ...
    def release(self): ...
    def handle(self) -> int: ...  # for C-ABI escape hatch
```

**Platform adapter (Linux -- direct Rust, Python for testing):**

```python
import asyncio
import os
from helix_core import Tunnel, TunnelConfig, TunnelEventCallback, TunnelEventData

class PrintCallback(TunnelEventCallback):
    async def on_event(self, event: TunnelEventData):
        print(f"[{event.kind}] {event.label} {event.data_json or ''}")

async def main():
    callback = PrintCallback()
    config = TunnelConfig(
        server_addr="127.0.0.1:51820",
        transport_name="plain-udp",
        listen_port=0,
        private_key_hex=os.environ.get("HELIX_WG_PRIVATE_KEY"),
        stats_interval_secs=5,
        keepalive_interval_secs=25,
    )

    tunnel = await Tunnel.create(config, callback)
    await tunnel.start()
    await tunnel.connect()

    # Read stats periodically
    for _ in range(60):
        await asyncio.sleep(5)
        status = tunnel.status()
        print(f"  tx={status.bytes_sent} rx={status.bytes_recv} "
              f"rtt={status.latency_ms}ms")

    await tunnel.disconnect()
    tunnel.release()

if __name__ == "__main__":
    asyncio.run(main())
```

---

## 4. Data-Plane Bridge (C-ABI Escape Hatch)

### 4.1 Problem statement

The TUN packet loop sends and receives a raw IP packet every few microseconds under load. UniFFI's per-call overhead includes:
- Handle-map lookup (hashtable lookup for every call entering Rust)
- Serialization of `Vec<u8>` / `sequence<u8>` through the FFI buffer
- Kotlin: `List<Byte>` allocation per call (boxed `Byte`, heap pressure)
- Swift: `Data` bridging through `RustBuffer`

For the lifecycle API (start/connect/disconnect, called once per user action) this overhead is negligible. For the data plane (called ~10,000-100,000 times per second), it dominates.

### 4.2 Solution: Layered FFI

| Layer | Mechanism | Call frequency | Example |
|---|---|---|---|
| Lifecycle (start, stop, status) | UniFFI | ~0.01 Hz | `tunnel.connect()` |
| Events (state changes, errors) | UniFFI callback interface | ~0.1-1 Hz | `callback.onEvent(data)` |
| **Data plane (packet I/O)** | **Raw C-ABI** | **~10 kHz** | `helix_send_packet(handle, buf, len)` |

### 4.3 C-ABI functions (`ffi_dataplane.rs`)

```rust
// crates/helix-ffi/src/ffi_dataplane.rs
// These are NOT in the UDL -- they are raw extern "C" functions
// called directly from the platform adapter's native code.

/// Opaque tunnel handle. Obtained from `Tunnel::handle()` via UniFFI.
pub type TunnelHandle = u64;

/// Send a raw IP packet through the tunnel.
///
/// Returns the number of bytes consumed from `packet` (should equal `len`
/// on success), or 0 if the tunnel is not connected / the buffer is full.
///
/// # Safety
/// `packet` must point to `len` valid bytes. `handle` must be a valid
/// tunnel handle obtained from `Tunnel::handle()`.
#[no_mangle]
pub extern "C" fn helix_send_packet(
    handle: TunnelHandle,
    packet: *const u8,
    len: u32,
) -> u32 {
    // Look up tunnel by handle, call WireGuard encrypt, send via transport
    // ...
}

/// Receive a raw IP packet from the tunnel.
///
/// Returns the number of bytes written to `buf`, or 0 if no packet is
/// available (non-blocking -- the caller should poll or wait on a fd).
///
/// # Safety
/// `buf` must point to at least `buf_capacity` writable bytes.
/// `handle` must be a valid tunnel handle.
#[no_mangle]
pub extern "C" fn helix_recv_packet(
    handle: TunnelHandle,
    buf: *mut u8,
    buf_capacity: u32,
) -> u32 {
    // Look up tunnel by handle, recv from transport, call WireGuard decrypt
    // ...
}

/// Return the effective MTU for this tunnel's transport.
///
/// The platform adapter MUST use this value when configuring its TUN
/// interface MTU (VpnService.Builder.setMtu, NEIPv4Settings, etc.).
/// For plain-udp this is 1420; for masque-quic-standin this is 1280.
#[no_mangle]
pub extern "C" fn helix_tunnel_mtu(handle: TunnelHandle) -> u32 {
    // ...
}
```

### 4.4 Handle-map internals

The `TunnelHandle` (u64) is obtained from the UniFFI `Tunnel` object via a dedicated method:

```webidl
interface Tunnel {
    // ... other methods ...
    u64 handle();
};
```

Internally, `helix-ffi` maintains a global handle map:

```rust
// crates/helix-ffi/src/handle_map.rs

use std::collections::HashMap;
use std::sync::Mutex;

static HANDLE_MAP: once_cell::sync::Lazy<Mutex<HashMap<u64, TunnelState>>> =
    once_cell::sync::Lazy::new(|| Mutex::new(HashMap::new()));

/// Tunnel state for the data-plane hot path.
pub struct TunnelState {
    pub transport_sender: tokio::sync::mpsc::Sender<Vec<u8>>,
    pub transport_receiver: tokio::sync::mpsc::Receiver<Vec<u8>>,
    pub effective_mtu: u16,
}
```

The C-ABI functions look up the handle in this map, push/pull from the mpsc channels, and return immediately. The UniFFI `Tunnel` object's internal loops bridge the mpsc channels to the actual transport `Connection` trait (async send/recv).

---

## 5. Build Pipeline

### 5.1 Crate structure

```
crates/helix-ffi/
  Cargo.toml          # [lib] crate-type = ["cdylib", "staticlib"]
  src/
    lib.rs             # UniFFI scaffolding include + re-exports
    ffi_tunnel.rs      # Tunnel, TunnelConfig, TunnelStatus impls
    ffi_events.rs      # TunnelEventCallback wiring
    ffi_dataplane.rs   # extern "C" hot-path functions
    handle_map.rs      # Global Mutex<HashMap<u64, TunnelState>>
  src/
    helix_core.udl     # UniFFI definition (shown above)
  build.rs             # uniffi_build::generate_scaffolding
```

### 5.2 Android build (cargo-ndk)

```bash
# Cross-compile to Android .so libraries
# Targets: arm64-v8a (aarch64-linux-android), armeabi-v7a, x86_64
cargo ndk \
  -t aarch64-linux-android \
  -t armv7-linux-androideabi \
  -t x86_64-linux-android \
  -o app/src/main/jniLibs \
  build --release -p helix-ffi

# Generate Kotlin bindings from UDL
uniffi-bindgen generate crates/helix-ffi/src/helix_core.udl \
  --language kotlin \
  --out-dir app/src/main/java/com/helix/vpn/bridge/

# The generated Kotlin file includes JNI native method declarations
# that are satisfied by the cdylib .so built above.
```

### 5.3 iOS/macOS build (lipo + XCFramework)

```bash
# Build for iOS device (arm64)
cargo build --release -p helix-ffi --target aarch64-apple-ios

# Build for iOS simulator (arm64, M-series Mac)
cargo build --release -p helix-ffi --target aarch64-apple-ios-sim

# Build for macOS (arm64 + x86_64)
cargo build --release -p helix-ffi --target aarch64-apple-darwin
cargo build --release -p helix-ffi --target x86_64-apple-darwin

# Create universal library (iOS device + simulator -> single .a)
lipo -create \
  target/aarch64-apple-ios/release/libhelix_ffi.a \
  target/aarch64-apple-ios-sim/release/libhelix_ffi.a \
  -output target/universal-ios/libhelix_ffi.a

# Generate Swift bindings
uniffi-bindgen generate crates/helix-ffi/src/helix_core.udl \
  --language swift \
  --out-dir apple/Sources/HelixCore/

# Package as XCFramework
xcodebuild -create-xcframework \
  -library target/universal-ios/libhelix_ffi.a \
  -headers apple/Headers/ \
  -library target/aarch64-apple-darwin/release/libhelix_ffi.a \
  -headers apple/Headers/ \
  -output HelixCore.xcframework
```

### 5.4 build.rs for UniFFI scaffolding

```rust
// crates/helix-ffi/build.rs
fn main() {
    uniffi_build::generate_scaffolding("src/helix_core.udl").unwrap();
}
```

### 5.5 lib.rs entry point

```rust
// crates/helix-ffi/src/lib.rs

// Include the UniFFI-generated scaffolding
uniffi::include_scaffolding!("helix_core");

mod ffi_tunnel;
mod ffi_events;
mod ffi_dataplane;
mod handle_map;
```

---

## 6. Test Strategy

### 6.1 Unit tests (Rust, in-workspace)

| Test | What it validates |
|---|---|
| `test_tunnel_create_with_valid_config` | `Tunnel::new()` with `TunnelConfig` succeeds |
| `test_tunnel_create_rejects_unknown_transport` | `Tunnel::new()` returns `ConfigInvalid` |
| `test_tunnel_create_rejects_malformed_addr` | `Tunnel::new()` returns `ConfigInvalid` |
| `test_tunnel_start_connect_disconnect_cycle` | Full lifecycle: start -> connect -> disconnect -> release |
| `test_tunnel_status_returns_zeroes_before_connect` | `status()` on an idle tunnel returns default stats |
| `test_callback_receives_connecting_event` | `on_event` is called with `kind="connecting"` after `connect()` |
| `test_callback_receives_connected_event` | `on_event` is called with `kind="connected"` on success |
| `test_callback_receives_down_event` | `on_event` is called with `kind="down"` after `disconnect()` |
| `test_callback_receives_error_on_bad_server` | `on_event` is called with `kind="error"` when dial fails |
| `test_double_connect_returns_already_connected` | Second `connect()` returns `AlreadyConnected` |
| `test_use_after_release_returns_handle_released` | All methods return `HandleReleased` after `release()` |
| `test_handle_map_lookup_for_valid_handle` | `helix_send_packet` with valid handle returns > 0 |
| `test_handle_map_rejects_invalid_handle` | `helix_send_packet` with bogus handle returns 0 |

### 6.2 Mock callback harness

```rust
// Test helper: a callback that records events into a Vec

struct RecordingCallback {
    events: Mutex<Vec<TunnelEventData>>,
}

#[uniffi::export(callback_interface)]
impl TunnelEventCallback for RecordingCallback {
    async fn on_event(&self, event: TunnelEventData) {
        self.events.lock().unwrap().push(event);
    }
}

// Test:
// 1. Create tunnel with RecordingCallback
// 2. Call start() then connect()
// 3. Wait 50ms for tokio tasks to fire
// 4. Assert events contain: ["state_changed: Disconnected",
//    "connecting", "handshaking", "connected", "state_changed: Connected"]
```

### 6.3 Kotlin binding tests (planned, not Phase 1)

- `TunnelCreateTest` -- create, connect to localhost mock server, disconnect
- `CallbackEventTest` -- assert all 6 event kinds fire in correct order
- `ErrorHandlingTest` -- connect to unreachable server, assert error event
- `StatusPollingTest` -- poll `status()` in a loop, assert counters increment
- `ReleaseTest` -- release tunnel, assert subsequent calls throw `HandleReleased`

### 6.4 Swift binding tests (planned, not Phase 1)

- Same test matrix as Kotlin, plus:
- `MemoryCeilingTest` -- 30-minute transfer, assert RSS stays under 50 MB (G3 gate)
- `ExtensionRestartTest` -- simulate NE process kill + restart, assert tunnel idempotent

### 6.5 Python binding tests (Phase 1, Linux)

- All Kotlin test cases, runnable as part of `cargo test` via `uniffi::bindings::python`
- Used as the reference binding for CI: if Python tests pass, the UDL is correct for all platforms

---

## 7. Honest Gaps (§11.4.6)

This design explicitly does NOT cover the following. Each gap is documented with its reason and the follow-up work item:

| Gap | Reason | Follow-up |
|---|---|---|
| **Real async UniFFI support (native `async fn` export)** | UniFFI v0.28-v0.31 exposes async Rust functions as `suspend fun` in Kotlin and `async throws` in Swift -- but this works for top-level functions and interface methods, NOT async trait methods. The internal `Transport`/`Connection`/`Listener` traits use `async_trait` and cannot cross FFI. The design sidesteps this by wrapping the orchestrator behind a sync API surface that spawns tokio tasks internally. | Track UniFFI issue #1852 (native async trait export); when it lands, the blocking wrappers in `ffi_tunnel.rs` can be simplified to direct async exports. |
| **Performance benchmarking of FFI overhead** | The claim that "UniFFI per-call overhead demands C-ABI for the data plane" is based on the defguard_boringtun precedent (Feb 2026), not on our own measurements with our exact packet sizes and platforms. The design provides the C-ABI escape hatch upfront, but the actual threshold (packets/sec at which UniFFI becomes the bottleneck) is unmeasured. | P1-005.1: Benchmark UniFFI `sequence<u8>` round-trip latency at 1400-byte payload on aarch64 Android and iOS. If overhead is < 5 us, the C-ABI escape hatch is unnecessary and can be removed (simpler). |
| **Actual per-platform test suite** | This document defines the test strategy (what to test and how), but the actual test implementations for Kotlin and Swift are Phase 1.5 work. The Python binding tests are the Phase 1 reference -- they prove the UDL is correct for all platforms. | Phase 1.5: implement Kotlin test suite (JUnit + mock VpnService), Swift test suite (XCTest + mock NEPacketTunnelFlow). |
| **Windows C-ABI .dll** | Windows is deferred to Phase 2 (HVPN-P2-001). The UDL is designed to generate language-agnostic bindings, but the `extern "C"` data-plane escape hatch currently targets Linux/Android calling conventions. Windows .dll export requires `__declspec(dllexport)` / `__stdcall` annotations. | Phase 2: add Windows calling-convention annotations behind `#[cfg(target_os = "windows")]`, test with `windivert-sys` crate. |
| **Callback thread-safety on iOS** | The UDL callback is `[Async]`, which means the foreign side starts an async operation and returns immediately. On iOS, the NE process has a constrained concurrency model; the callback's async dispatch to `NEPacketTunnelProvider`'s internal queue is correct per the Swift structured-concurrency spec but has not been tested under iOS's memory-ceiling conditions (G3 gate). | Phase 1.5: stress-test callback delivery under 30-min transfer + 50 MB memory ceiling (G3 gate verification). |
| **UniFFI v0.31 vs v0.28** | The platform adapter research doc references UniFFI v0.31.1 (April 2026). The DeepWiki timeline documents features through v0.30.0. There is no confirmed documentation of what changed in v0.31 specifically. The UDL in this design uses syntax features confirmed present in v0.28+ (`[Async]` on callbacks, `[Error]` enums, `dictionary` with optional fields). If v0.31 adds native async trait export or removes limitations, this design can be simplified. | Before implementing the FFI crate, run `cargo doc --open -p uniffi` on the exact version pinned in `Cargo.toml` and verify that every UDL feature used here (async callbacks, `[Error]` enums, `interface` constructors with `[Throws]`, `dictionary` with `string?`) is supported. Update this design if v0.31 adds capabilities that simplify the architecture. |

---

## 8. Sources Verified

| Source | URL | Verified |
|---|---|---|
| UniFFI UDL Files (DeepWiki) | `https://deepwiki.com/mozilla/uniffi-rs/2.1-udl-files` | 2026-07-08 |
| UniFFI Callback Interfaces (DeepWiki) | `https://deepwiki.com/mozilla/uniffi-rs/6.4-callback-interfaces` | 2026-07-08 |
| UniFFI Async Functions (DeepWiki) | `https://deepwiki.com/mozilla/uniffi-rs/6.3-async-functions` | 2026-07-08 |
| defguard_boringtun (UniFFI + JNI dual FFI precedent) | `https://lib.rs/crates/defguard_boringtun` (v0.6.5, Feb 2026) | 2026-07-08 |
| Platform Adapters Research (helix_vpn) | `docs/research/platform_adapters/RESEARCH.md` (this project) | 2026-07-08 |
| UniFFI Build System Integration (DeepWiki) | `https://deepwiki.com/mozilla/uniffi-rs/7.3-build-system-integration` | 2026-07-08 |

*Gaps/silences: UniFFI v0.31 changelog not yet documented in public Wikis -- this design uses v0.28+ confirmed features. Actual UniFFI per-call overhead for 1400-byte payloads on mobile ARM has not been measured (see gap table above). The `[Async]` callback support across all four binding languages is documented for Kotlin, Swift, and Python; Ruby async callback support is unconfirmed.*

