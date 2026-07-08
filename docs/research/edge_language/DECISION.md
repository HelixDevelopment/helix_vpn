# Edge Language Decision — Rust (quinn) vs Go (quic-go + masque-go) for MASQUE Relay

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Scope:** Helix VPN P1-004 / G4 Gate — Edge-service language for MASQUE connection termination at relay/concentrator
**Authority:** §11.4.8 (deep web research), §11.4.99 (latest-source cross-reference), §11.4.6 (no guessing)

---

## 1. Executive Summary

The project needs an "edge" service that terminates MASQUE (RFC 9298 CONNECT-UDP)
connections at a relay/concentrator. Two paths exist: extend our existing Rust `helix_masque`
crate (quinn-based) to real RFC 9298 compliance, or start a Go service using quic-go +
masque-go.

**Recommendation: Stay Rust, but switch to Cloudflare `tokio-quiche` for the MASQUE data
plane.** Our existing quinn `QuicServerEndpoint` + `QuicClientEndpoint` are solid QUIC
transport primitives; the gap is the HTTP/3 Extended CONNECT (RFC 9220) layer that real
RFC 9298 CONNECT-UDP requires. The Rust ecosystem now has a production-grade path through
Cloudflare's tokio-quiche, which already powers the WARP MASQUE client handling millions of
RPS. Go's masque-go implements real RFC 9298 out of the box but is pre-1.0 (v0.4.0, 12
open issues), and adopting Go would introduce a second language into the project.

---

## 2. What We Have Today (Rust / quinn / helix_masque)

### 2.1 What's real and working

Our `helix_masque` crate (in `submodules/helix_core/crates/helix-masque/`) is a genuine
working MASQUE transport with these production-quality pieces:

| Layer | Status | File |
|-------|--------|------|
| QUIC transport (quinn client/server endpoints) | Real, tested | `quic.rs` |
| TLS 1.3 with SNI enforcement | Real, tested (matching + mismatched SNI) | `quic.rs` |
| RFC 9221 DATAGRAM frames (unreliable, no streams) | Real, tested (1200-byte round-trip, frame stats verified) | `quic.rs` |
| RFC 9297 HTTP-Datagram framing (context-id + payload) | Real, tested | `datagram.rs` |
| CONNECT-UDP-flow establishment (stand-in) | Working simplified protocol, tested E2E | `connect.rs` |
| Connection trait impl (send/recv/close/kind/effective_mtu) | Real, tested | `lib.rs` |
| DPI-block survival (real nft rules, real boringtun + quinn) | Proven (G2 gate PASS) | `G2-RESULTS.md` |
| Wire fingerprint (zero WG on port 443, all QUIC) | Proven (G2 gate PASS) | `G2-RESULTS.md` |
| Dial timeout enforcement | Real, tested | `lib.rs` |
| Context-id filtering + malformed datagram tolerance | Real, tested | `lib.rs` |

### 2.2 What's explicitly NOT real (honest labels)

The crate is honest about its limitations (quoted from `lib.rs`):

> "The connect flow-establishment handshake is an explicitly-labeled simplified stand-in,
> NOT real HTTP/3 Extended CONNECT (RFC 9220) / real RFC 9298 CONNECT-UDP."

Deep research documented in the crate found the `h3` crate's Extended CONNECT support was
"recent-and-narrow (only the `:protocol` pseudo-header primitive, no CONNECT-UDP/MASQUE
builder)" and its datagram-to-stream association was "actively broken upstream as of ~2
months before this task."

The `kind()` method returns `"masque-quic-standin"` — deliberately NOT `"masque-h3"`,
exactly as this project's anti-bluff discipline requires.

### 2.3 The path to production-quality RFC 9298 on Rust

There are now **two viable upstream paths** that did not exist when the Phase-0 spike was
written:

**Path A: Cloudflare tokio-quiche (recommended)**

Cloudflare open-sourced [tokio-quiche](https://blog.cloudflare.com/async-quic-and-http-3-made-easy-tokio-quiche-is-now-open-source/)
in November 2025. It provides:

- Full RFC 9298 CONNECT-UDP support via its `H3Driver` abstraction
- Production-proven at Cloudflare scale: powers WARP MASQUE client, Apple iCloud Private
  Relay (Proxy B), and Oxy-based HTTP/3 proxies, handling millions of RPS ([source](https://blog.cloudflare.com/async-quic-and-http-3-made-easy-tokio-quiche-is-now-open-source/))
- Tokio-native async API (unlike raw quiche which is sans-IO)
- Built-in CONNECT-UDP and MASQUE support
- Available on crates.io as `tokio-quiche`

Trade-off: quiche uses BoringSSL (C dependency), not rustls. This means CGO-like build
complexity for the Rust binary (cross-compilation needs the BoringSSL toolchain per target).

**Path B: Finish real RFC 9298 on quinn + a newer h3 stack**

The `h3x` crate ([crates.io](https://crates.io/crates/h3x), [GitHub](https://github.com/genmeta/h3x))
now supports Extended CONNECT (RFC 9220) explicitly, with releases through v0.5.0
(June 2026). Combined with our existing quinn endpoints, this could provide a pure-Rust
RFC 9298 path. However, h3x is even newer than masque-go (v0.5.0, single-digit versions)
and has no known production deployment.

**Path C: Mozilla neqo**

Mozilla's [neqo](https://github.com/mozilla/neqo) merged CONNECT-UDP support in PR
[#2796](https://github.com/mozilla/neqo/pull/2796). neqo is used in Firefox and has
Mozilla's security backing, but is less idiomatic for non-browser use cases.

### 2.4 What we keep regardless of path

Our existing `quic.rs` QUIC endpoints, `datagram.rs` RFC 9297 framing, and the entire
`MasqueTransport` / `MasqueConnection` / `MasqueListener` trait-based architecture are
valuable regardless of which HTTP/3 layer sits underneath. The transport trait abstraction
(`helix_transport`) means the edge service can swap the underlying MASQUE implementation
without changing the orchestrator or WireGuard layers.

---

## 3. The Go Option (quic-go + masque-go)

### 3.1 quic-go maturity

[quic-go](https://github.com/quic-go/quic-go) is the dominant Go QUIC implementation:

- **11,653+ GitHub stars**
- Production users include: Caddy (via quic-go HTTP/3), Cloudflare's go-quic (interop
  tested at IETF 123), [cloudbridge-research/masque-vpn](https://github.com/cloudbridge-research/masque-vpn)
  (full MASQUE VPN with WireGuard integration), and go-gost (MASQUE support merged mid-2026)
- Active maintenance: frequent releases, 1,000+ closed issues, responsive maintainer
  ([@marten-seemann](https://github.com/marten-seemann))
- Performance: ~1.3 Gbit/s single-connection throughput, ~132 KB/connection memory,
  63ms connection setup at 1M connections ([source](https://datasea.cn/go0202451432.html))
- WAN caveat: default NewReno congestion control collapses to ~15% utilization on
  high-BDP links; BBR required ([GitHub issue #5325](https://github.com/quic-go/quic-go/issues/5325))

### 3.2 masque-go: what it gives us out of the box

[masque-go](https://github.com/quic-go/masque-go) (v0.4.0, June 2026) provides:

- **Real RFC 9298 CONNECT-UDP**: native Extended CONNECT (`:protocol = connect-udp`),
  HTTP Datagrams (RFC 9297/9279), capsule protocol — not a stand-in
- **Client API**: `Transport.Dial()`, `ClientConn`, `Request` — one-liner to establish
  a proxied UDP flow
- **Proxy API**: `Proxy.Proxy()`, `ParseProxyRequest()` — HTTP handler that accepts
  CONNECT-UDP requests
- **URI template-based routing**: `?h={target_host}&p={target_port}` — flexible
  target resolution
- **`net.PacketConn`-compatible `Conn`**: standard Go `ReadFrom`/`WriteTo` interface,
  drops into existing Go UDP code

Example proxy setup (from the official docs):

```go
t := uritemplate.MustNew("https://example.org:4443/masque?h={target_host}&p={target_port}")
var proxy masque.Proxy
http.Handle("/masque", func(w http.ResponseWriter, r *http.Request) {
    mreq, err := masque.ParseProxyRequest(r, t)
    proxy.Proxy(w, mreq)
})
http3.ListenAndServeQUIC(":4443", certFile, keyFile, nil)
```

### 3.3 masque-go maturity assessment

| Dimension | Rating | Evidence |
|-----------|--------|----------|
| RFC 9298 compliance | **Real** | Implements Extended CONNECT + HTTP Datagrams natively |
| API stability | **Pre-1.0** | v0.4.0 — semver permits breaking changes |
| Production users | **None known** | No documented production deployments found in research |
| Issue tracker health | **Early** | 12 open issues, 5 open PRs on a small codebase |
| Maintainer | **Strong** | Same quic-go team (marten-seemann), responsive |
| Dependencies | **Minimal** | Only quic-go + uritemplate |

masque-go is the honest early-stage counterpart to our own `helix_masque`: it implements
the real RFC 9298 handshake (which we don't), but it hasn't been battle-tested at scale
(which our QUIC + datagram layers have been through the G2 gate).

### 3.4 Go-specific considerations

**Memory model for long-lived tunnels:**

Go's goroutine-per-connection model costs ~5 KB baseline per connection (goroutine stack +
buffers + connection struct). At 10,000 concurrent tunnels this is ~50 MB — manageable. At
100,000 it is ~500 MB in goroutine overhead alone, plus application state. GC pauses of
1-5 ms at P99 become relevant above ~50K connections. Sources:
[1m-go-tcp-server benchmarks](https://github.com/Zt105/1m-go-tcp-server),
[nbio million-connection test](https://github.com/lesismal/nbio).

For the Helix VPN edge relay, typical scale is hundreds to low thousands of concurrent
tunnels at a concentrator, not hundreds of thousands — Go's per-connection overhead is
well within budget.

**GC characteristics:**

- Go 1.22+ concurrent mark-sweep GC: typical pauses <1ms, but P99 spikes of 3-5ms under
  high allocation rates
- Recommendation: `GOGC=200` (less frequent GC) for a latency-tolerant tunnel relay
- For comparison, Rust (no GC) has flat latency curves regardless of connection count

**Compilation and deployment:**

Go cross-compilation is trivial:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o edge-relay
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o edge-relay-arm64
```

Produces a single static binary (~8-12 MB after strip + UPX) with zero runtime
dependencies — no libc, no OpenSSL, no system packages. Rust with quinn+rustls produces a
similarly standalone static binary (~5-8 MB). Rust with quiche (BoringSSL) requires the
BoringSSL build toolchain and produces a larger binary with a C dependency.

---

## 4. Comparison Matrix

### 4.1 RFC 9298 Compliance

| | Rust (tokio-quiche) | Rust (quinn + h3x) | Go (masque-go) |
|---|---|---|---|
| Real Extended CONNECT (RFC 9220) | Yes, production | Yes (h3x v0.5.0) | Yes |
| Real CONNECT-UDP handshake | Yes | Yes (new, unproven) | Yes |
| HTTP Datagrams (RFC 9297) | Yes | Yes | Yes |
| Capsule protocol (RFC 9297) | Yes | Unknown | Yes |
| CONNECT-UDP-Bind (draft) | No (IETF draft) | No | No |
| Production proven at scale | **Yes** (Cloudflare WARP, millions RPS) | No | No |
| Out-of-box vs build-from-scratch | Out-of-box | Build (wire h3x onto quinn) | Out-of-box |

### 4.2 Performance

| Dimension | Rust (any QUIC stack) | Go (quic-go) | Source |
|---|---|---|---|
| Memory per connection | ~25-50 KB (no GC overhead) | ~132 KB (quic-go measured at 1M conn) | [datasea.cn](https://datasea.cn/go0202451432.html) |
| Memory at 10K tunnels | ~250-500 MB | ~300-600 MB (with GC headroom) | Multiple benchmarks |
| Single-connection throughput (LAN) | 2-6 Gbit/s (quinn), 2.5+ Gbit/s (quiche) | 1.3 Gbit/s | [IFIP Networking 2025 study](https://datatracker.ietf.org/meeting/123/materials/slides-123-maprg-sessb-quic-http-implementation-performance-00), quic-go docs |
| P99 latency under load | Sub-100us (no GC) | 3-15ms (GC pauses at scale) | CloudBridge benchmarks |
| WAN throughput (with BBR) | Good (quiche production-tuned) | ~50% improvement with BBR vs NewReno | [quic-go issue #5325](https://github.com/quic-go/quic-go/issues/5325) |
| CPU efficiency | Higher (no GC cycles) | ~10% CPU spent in GC under load | Multiple Go profiling reports |

### 4.3 Ecosystem Maturity

| Dimension | Rust | Go |
|---|---|---|
| **QUIC library maturity** | quinn: stable, most-used Rust QUIC, 4K+ stars. quiche: Cloudflare battle-tested. s2n-quic: AWS production | quic-go: 11.6K+ stars, dominant Go QUIC, many production users |
| **MASQUE library maturity** | tokio-quiche: production (Cloudflare WARP). h3x: new (v0.5.0). saorsa-transport: early | masque-go: early (v0.4.0, 239 stars, 12 open issues) |
| **Production MASQUE deployments** | Cloudflare WARP (millions of users), Apple iCloud Private Relay (Proxy B) | None documented for masque-go specifically |
| **Issue tracker / maintenance** | quinn: active, responsive. quiche: Cloudflare-backed, actively maintained | quic-go: very active (1K+ closed issues). masque-go: small, maintained by same team |
| **HTTP/3 stack** | h3 crate (limited Extended CONNECT), h3x (new, fuller), quiche H3 (production) | Built into quic-go (mature, production) |

### 4.4 Team Integration

| Dimension | Rust | Go |
|---|---|---|
| **Existing codebase** | Full workspace: helix-core, helix-transport, helix-masque, helix-wg, helix-tun, helix-orch | None — would be a new language in the project |
| **Learning curve** | Team already knows Rust (existing crates) | Team would need to learn Go idioms, toolchain |
| **Code sharing** | Can reuse helix-transport traits, tunnel orchestrator, WireGuard integration | Would need to rebuild or bridge the orchestrator |
| **Single-repo vs multi-repo** | Same workspace, shared deps, one build system | Separate module, separate build, bridge layer needed |
| **Future crates compatibility** | helix-wg, helix-tun, helix-orch all Rust — edge relay in Rust means direct integration | Go edge relay needs an IPC bridge (Unix socket/gRPC) to the Rust orchestrator |

Introducing Go means the edge service cannot directly import `helix_transport::Transport`,
`helix_masque::MasqueConfig`, or the WireGuard orchestrator. A bridge layer (likely a Unix
domain socket or gRPC boundary) would be needed between the Go MASQUE relay and the Rust
tunnel orchestrator. This is not a blocker — it's a standard microservice pattern — but
it adds complexity versus a single-process Rust solution.

### 4.5 Deployment

| Dimension | Rust (quinn + rustls) | Rust (tokio-quiche / BoringSSL) | Go (quic-go) |
|---|---|---|---|
| **Static binary size** | ~5-8 MB (strip + LTO) | ~10-15 MB (C dep linked) | ~8-12 MB (strip + UPX) |
| **Cross-compile amd64** | `cargo build --target x86_64-unknown-linux-musl` | Needs BoringSSL cross-compile toolchain | `CGO_ENABLED=0 GOARCH=amd64 go build` |
| **Cross-compile arm64** | `cargo build --target aarch64-unknown-linux-musl` | Needs aarch64 BoringSSL toolchain | `CGO_ENABLED=0 GOARCH=arm64 go build` |
| **Runtime deps (static)** | None (pure Rust + rustls) | libc only (BoringSSL statically linked) | None (pure Go) |
| **Container size (distroless)** | ~15 MB | ~25 MB | ~15 MB |
| **Build time** | Minutes (full workspace rebuild) | Minutes (plus BoringSSL build) | Seconds |

**Winner for pure-static deployment**: Go (`CGO_ENABLED=0`) and Rust-with-rustls are
equivalent — both produce truly static binaries with zero runtime dependencies. Rust-with-
quiche/BoringSSL adds a C toolchain dependency that complicates cross-compilation.

### 4.6 Security

| Dimension | Rust | Go |
|---|---|---|
| **Memory safety** | Compile-time ownership/borrowing — no use-after-free, no double-free, no data races at compile time | Runtime bounds checking + GC — memory-safe but data races possible with improper goroutine synchronization |
| **CVE history (QUIC libs)** | quinn: minimal (rustls-backed). quiche: Cloudflare security team, BoringSSL | quic-go: well-maintained, prompt CVE response. Go stdlib TLS: strong track record |
| **Fuzzing infrastructure** | quinn: cargo-fuzz + proptest. quiche: Cloudflare internal fuzzing | quic-go: integrated Go fuzzing (since Go 1.18) |
| **Supply chain** | Cargo: crates.io, lockfile. quiche adds BoringSSL C dep | Go modules: proxy.golang.org, go.sum, minimal dependency tree |
| **Sandboxing potential** | seccomp + Landlock compatible (pure Rust) | seccomp compatible (pure Go, CGO_ENABLED=0) |

Both languages are strong on security with different strengths. Rust's compile-time
guarantees eliminate entire classes of memory bugs. Go's simplicity and minimal dependency
tree reduce supply-chain risk. For a network edge service processing untrusted input
(MASQUE datagrams from the public internet), Rust's memory safety at the type level is
a meaningful advantage — a buffer handling bug in Rust is a compile error, not a CVE.

---

## 5. Recommendation

### Stay Rust. Switch to tokio-quiche for real RFC 9298.

**Primary recommendation**: Adopt Cloudflare's `tokio-quiche` as the HTTP/3 + MASQUE
layer underneath our existing `MasqueTransport` / `MasqueConnection` trait
implementations. This gives us:

1. **Real RFC 9298 CONNECT-UDP** — production-grade, not a stand-in
2. **Cloudflare-scale proven** — WARP MASQUE client handles millions of RPS today
3. **Same language** — no second language, no IPC bridge, no duplicated orchestrator
4. **Keep our investment** — `quic.rs` endpoints become the fallback/alternative QUIC
   transport, `datagram.rs` RFC 9297 framing stays relevant, the `helix_transport` trait
   abstraction proves its value by swapping the underlying implementation
5. **Production credibility** — "our MASQUE edge is built on the same stack Cloudflare
   uses for WARP" is a strong answer to any DPI-resistance audit

**Secondary (fallback) recommendation**: If the BoringSSL dependency of quiche is
unacceptable (cross-compilation complexity, C supply chain), pursue Path B: wire `h3x`
(Extended CONNECT) onto our existing quinn endpoints. This stays pure Rust + rustls but
uses a newer, less-proven h3 stack.

**Go is NOT recommended** for the following reasons:

1. masque-go (v0.4.0) is itself early-stage — we would be trading one immature MASQUE
   stack for another, while also adding a second language
2. The Go + Rust bridge adds architectural complexity (IPC boundary, duplicated
   configuration, separate build/deploy pipeline)
3. No production MASQUE deployment of masque-go was found in research — both our stack
   and masque-go are pre-production for real RFC 9298
4. Go's GC and goroutine overhead are manageable at our scale, but Rust's predictable
   latency is objectively better for a network data plane

---

## 6. Honest Assessment — What We Lose With Either Choice

### With Rust + tokio-quiche (the recommended path)

- **BoringSSL C dependency**: Lose the pure-Rust TLS stack. Cross-compilation for arm64
  requires the BoringSSL build toolchain. This is the largest concrete downside.
- **Less community examples**: quic-go has more blog posts, tutorials, and community
  examples for MASQUE than Rust does.
- **Build complexity**: BoringSSL build adds minutes to CI and requires the `cmake` +
  `ninja` + `golang` (BoringSSL's build system uses Go) toolchain.

### With Go (the alternative)

- **Second language in the project**: Every future engineer needs to know both Rust and Go.
  CI needs both toolchains. Dependency management doubles.
- **IPC bridge complexity**: Go edge relay cannot directly use `helix_transport` traits or
  the WireGuard orchestrator — needs a Unix socket/gRPC bridge.
- **GC unpredictability at scale**: Acceptable at hundreds of tunnels, becomes a concern
  at tens of thousands.
- **masque-go is also early-stage**: We would not be gaining production maturity over
  the Rust path — both are pre-production for MASQUE specifically.

### Common to both paths

- **Neither masque-go nor h3x/tokio-quiche MASQUE has an independent security audit** (that
  we could find). Both are trusted by their respective ecosystems but neither has a
  published third-party audit for the MASQUE layer specifically.
- **CONNECT-UDP-Bind (draft-ietf-masque-connect-udp-listen)** is not implemented in
  either stack yet. This extension (for server-initiated UDP flows) will need to be added
  on either path.

---

## 7. Next Steps

1. **G4 Gate Decision**: Operator reviews this document and selects a path.
2. **Spike task (P1-004a)**: If Rust + tokio-quiche is chosen — produce a minimal
   working edge relay binary using tokio-quiche's MASQUE APIs, integrating with our
   existing `helix_transport` traits.
3. **Benchmark task (P1-004b)**: Measure tokio-quiche MASQUE throughput, memory per
   connection, and loss resilience on the same G2 gate methodology (nft DPI rules, tc
   netem, our existing benchmark harness).
4. **Cross-compilation proof (P1-004c)**: Produce static amd64 + arm64 Linux binaries
   from the tokio-quiche + BoringSSL build pipeline, confirming the toolchain works
   in CI.

---

## 8. Sources Verified

All sources fetched, verified, and cross-referenced on 2026-07-08 per §11.4.99.

### Rust ecosystem

- quinn: <https://github.com/quinn-rs/quinn> — Rust QUIC implementation, rustls-native
- Cloudflare tokio-quiche announcement (Nov 2025): <https://blog.cloudflare.com/async-quic-and-http-3-made-easy-tokio-quiche-is-now-open-source/>
- Cloudflare quiche: <https://github.com/cloudflare/quiche>
- h3x crate (Extended CONNECT): <https://crates.io/crates/h3x>, <https://github.com/genmeta/h3x>
- Mozilla neqo CONNECT-UDP PR #2796: <https://github.com/mozilla/neqo/pull/2796>
- saorsa-transport MASQUE relay: <https://docs.rs/saorsa-transport/latest/src/saorsa_transport/masque/relay_client.rs.html>
- ruvnet/midstream ADR-0021 (quinn vs quiche vs s2n-quic): <https://github.com/ruvnet/midstream/blob/main/docs/adr/0021-quic-implementation-quinn.md>
- quinn transport config docs: <https://docs.rs/quinn/latest/quinn/struct.TransportConfig.html>
- rustls CryptoProvider docs: <https://docs.rs/rustls/latest/rustls/crypto/struct.CryptoProvider.html>
- RFC 8200 (IPv6 minimum MTU 1280): <https://www.rfc-editor.org/rfc/rfc8200.html>
- RFC 9221 (QUIC DATAGRAM frames): <https://www.rfc-editor.org/rfc/rfc9221.html>
- WireGuard MTU community guidance: <https://defguard.net/blog/mtu-mss-decision-tree/>, <https://keremerkan.dev/posts/wireguard-mtu-fixes/>

### Go ecosystem

- masque-go GitHub: <https://github.com/quic-go/masque-go> (v0.4.0, 239 stars, 12 open issues, MIT)
- masque-go pkg.go.dev: <https://pkg.go.dev/github.com/quic-go/masque-go>
- quic-go CONNECT-UDP docs: <https://quic-go.net/docs/connect-udp/>
- quic-go GitHub: <https://github.com/quic-go/quic-go> (11,653+ stars)
- quic-go CONNECT-UDP tracking issue #4393: <https://github.com/quic-go/quic-go/issues/4393>
- quic-go WAN throughput issue #5325: <https://github.com/quic-go/quic-go/issues/5325>
- cloudbridge-research/masque-vpn (Go MASQUE VPN with WireGuard): <https://github.com/cloudbridge-research/masque-vpn>
- go-gost MASQUE issue #793: <https://github.com/go-gost/gost/issues/793>

### Performance data

- IFIP Networking 2025 — QUIC/HTTP implementation performance study: <https://datatracker.ietf.org/meeting/123/materials/slides-123-maprg-sessb-quic-http-implementation-performance-00>
- Go networking benchmark (1M connections): <https://datasea.cn/go0202451432.html>
- Go goroutine-per-connection vs epoll benchmarks: <https://github.com/Zt105/1m-go-tcp-server>
- nbio million-connection test: <https://github.com/lesismal/nbio>
- Rust vs Go for networking (CloudBridge): <https://cloudbridge-research.ru/en/blog/go-rust-networking/>

### IETF standards

- RFC 9298 (Proxying UDP in HTTP): <https://datatracker.ietf.org/doc/html/rfc9298>
- RFC 9297 (HTTP Datagrams and Capsule Protocol): <https://datatracker.ietf.org/doc/html/rfc9297>
- RFC 9220 (Bootstrapping WebSockets with HTTP/3 — Extended CONNECT): <https://datatracker.ietf.org/doc/html/rfc9220>
- RFC 9221 (Unreliable Datagram Extension to QUIC): <https://datatracker.ietf.org/doc/html/rfc9221>
- CONNECT-UDP-Bind IETF draft: <https://datatracker.ietf.org/meeting/123/materials/slides-123-masque-connect-udp-bind-00>
