//! HVPN-P0-045 benchmark driver for the Rust `helix-edge` MASQUE
//! termination path (G4 A/B bench, HelixVPN-Phase0-Spike.md §7/§7.2).
//!
//! This binary drives the REAL `helix_edge::edge::spawn_edge` entry point
//! (the exact function `helix-edge`'s own `src/main.rs` calls) and the
//! REAL `helix-masque` client transport (`MasqueTransport::new_client`),
//! over loopback — there is no fake/mocked MASQUE termination here.
//!
//! # Honest scope (read before trusting a number from this tool)
//!
//! - Runs entirely on 127.0.0.1. This sandbox has no passwordless sudo, so
//!   real kernel WireGuard, real network namespaces, and a real
//!   privileged `:443` bind are out of reach — exactly the same
//!   constraint `helix_edge`'s own tests document.
//! - Neither edge (Rust or Go) has a real kernel-WireGuard/boringtun
//!   gateway-socket integration yet (confirmed from `helix_edge`'s own
//!   README "Not yet implemented" list). This tool therefore benchmarks
//!   the MASQUE **termination + gateway-relay hand-off** data path itself
//!   — real QUIC handshake, real (hand-rolled, see helix-masque's own
//!   module docs) CONNECT-UDP-standin flow establishment, real datagram
//!   relay to a real loopback UDP sink — NOT an end-to-end WireGuard
//!   tunnel (that full slice does not exist yet for either edge).
//! - Two roles run as SEPARATE OS processes (`--role server` /
//!   `--role client`), mirroring iperf3's `-s`/`-c` split, so the
//!   orchestrating bench script can attribute CPU/RSS to the edge
//!   process alone via `/proc/<pid>` — this binary does not self-report
//!   resource usage.
//!
//! # CLI
//!
//! ```text
//! rust_edge_bench --role server --bind-ip 127.0.0.1 --sni-host <host> --cert-out <path>
//! rust_edge_bench --role client --edge-addr <ip:port> --target-addr <ip:port> \
//!     --cert <path> --sni-host <host> --mode {throughput|latency|churn} \
//!     --concurrency N --duration-secs S --payload-bytes B
//! ```
//!
//! Client output: CSV lines prefixed `CSV,` on stdout, columns
//! `edge,mode,concurrency,metric,value,unit`.

use std::collections::HashMap;
use std::net::{IpAddr, SocketAddr};
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use helix_edge::edge::{EdgeConfig, spawn_edge};
use helix_masque::{MasqueConfig, MasqueTransport};
use helix_transport::Transport;
use rustls::pki_types::{CertificateDer, PrivateKeyDer};
use tokio::net::UdpSocket;

fn parse_args() -> HashMap<String, String> {
    let mut map = HashMap::new();
    let args: Vec<String> = std::env::args().collect();
    let mut i = 1;
    while i < args.len() {
        if let Some(name) = args[i].strip_prefix("--") {
            let value = args.get(i + 1).cloned().unwrap_or_default();
            map.insert(name.to_string(), value);
            i += 2;
        } else {
            i += 1;
        }
    }
    map
}

fn arg<'a>(m: &'a HashMap<String, String>, name: &str, default: &'a str) -> String {
    m.get(name).cloned().unwrap_or_else(|| default.to_string())
}

fn generate_cert(sni_host: &str) -> (CertificateDer<'static>, PrivateKeyDer<'static>) {
    let certified_key = rcgen::generate_simple_self_signed(vec![sni_host.to_string()])
        .expect("rcgen self-signed certificate generation");
    let cert_der = CertificateDer::from(certified_key.cert);
    let key_der: PrivateKeyDer<'static> =
        rustls::pki_types::PrivatePkcs8KeyDer::from(certified_key.key_pair.serialize_der()).into();
    (cert_der, key_der)
}

fn main() {
    let args = parse_args();
    let role = arg(&args, "role", "");

    let rt = tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()
        .expect("build tokio runtime");

    match role.as_str() {
        "server" => rt.block_on(run_server(&args)),
        "client" => rt.block_on(run_client(&args)),
        _ => {
            eprintln!("usage: rust_edge_bench --role {{server|client}} ...");
            std::process::exit(2);
        }
    }
}

/// Server role: real MASQUE listener (`spawn_edge`) + two real loopback
/// UDP sinks standing in for "the gateway/kernel-WG socket" — an echo
/// sink (round-trip tests: latency, churn) and a counting sink (one-way
/// throughput test, sink-side byte count is the authoritative measured
/// goodput, per this project's "trust the sink, not the sender" ethos).
async fn run_server(args: &HashMap<String, String>) {
    let bind_ip: IpAddr = arg(args, "bind-ip", "127.0.0.1").parse().expect("bind-ip");
    let sni_host = arg(args, "sni-host", "edge-bench.invalid");
    let cert_out = arg(args, "cert-out", "");
    if cert_out.is_empty() {
        eprintln!("--cert-out is required for --role server");
        std::process::exit(2);
    }

    let (cert_der, key_der) = generate_cert(&sni_host);
    std::fs::write(&cert_out, cert_der.as_ref()).expect("write cert-out DER file");

    let config = EdgeConfig {
        bind_ip,
        masque_port: 0,
        sni_host: sni_host.clone(),
        cert_chain: vec![cert_der],
        key: key_der,
    };
    let addrs = spawn_edge(config).await.expect("spawn_edge");

    // Echo sink — round-trips every datagram (latency / churn tests).
    let echo_sock = UdpSocket::bind(SocketAddr::new(bind_ip, 0))
        .await
        .expect("bind echo sink");
    let echo_addr = echo_sock.local_addr().expect("echo sink local_addr");
    tokio::spawn(async move {
        let mut buf = vec![0u8; 65536];
        loop {
            match echo_sock.recv_from(&mut buf).await {
                Ok((n, from)) => {
                    let _ = echo_sock.send_to(&buf[..n], from).await;
                }
                Err(_) => break,
            }
        }
    });

    // Counting sink — one-way (throughput test); never echoes, just
    // counts bytes/packets and periodically prints sink-side stats that
    // the orchestrating bash script scrapes as the authoritative
    // received-goodput number (real sink-side positive evidence, not a
    // client-reported "offered load" guess).
    let count_sock = UdpSocket::bind(SocketAddr::new(bind_ip, 0))
        .await
        .expect("bind count sink");
    let count_addr = count_sock.local_addr().expect("count sink local_addr");
    let bytes_total = Arc::new(AtomicU64::new(0));
    let packets_total = Arc::new(AtomicU64::new(0));
    {
        let bytes_total = bytes_total.clone();
        let packets_total = packets_total.clone();
        tokio::spawn(async move {
            let mut buf = vec![0u8; 65536];
            loop {
                match count_sock.recv_from(&mut buf).await {
                    Ok((n, _from)) => {
                        bytes_total.fetch_add(n as u64, Ordering::Relaxed);
                        packets_total.fetch_add(1, Ordering::Relaxed);
                    }
                    Err(_) => break,
                }
            }
        });
    }
    {
        let bytes_total = bytes_total.clone();
        let packets_total = packets_total.clone();
        tokio::spawn(async move {
            let mut interval = tokio::time::interval(Duration::from_millis(200));
            loop {
                interval.tick().await;
                let ts = std::time::SystemTime::now()
                    .duration_since(std::time::UNIX_EPOCH)
                    .unwrap()
                    .as_millis();
                println!(
                    "COUNT_STATS ts={} bytes={} packets={}",
                    ts,
                    bytes_total.load(Ordering::Relaxed),
                    packets_total.load(Ordering::Relaxed)
                );
                use std::io::Write as _;
                let _ = std::io::stdout().flush();
            }
        });
    }

    println!(
        "READY edge={} echo_sink={} count_sink={} cert={} pid={}",
        addrs.masque_addr,
        echo_addr,
        count_addr,
        cert_out,
        std::process::id()
    );
    {
        use std::io::Write as _;
        let _ = std::io::stdout().flush();
    }

    // Run until killed (SIGTERM/SIGINT from the orchestrating script).
    let _ = tokio::signal::ctrl_c().await;
}

fn build_client_transport(args: &HashMap<String, String>) -> (MasqueTransport, SocketAddr) {
    let edge_addr: SocketAddr = arg(args, "edge-addr", "").parse().expect("--edge-addr");
    let sni_host = arg(args, "sni-host", "edge-bench.invalid");
    let cert_path = arg(args, "cert", "");
    let cert_bytes = std::fs::read(&cert_path).expect("read --cert DER file");
    let cert_der = CertificateDer::from(cert_bytes);

    let config = MasqueConfig {
        proxy_host: edge_addr.ip().to_string(),
        proxy_port: edge_addr.port(),
        verify_tls: true,
        connect_timeout_secs: 10,
    };
    (
        MasqueTransport::new_client(config, sni_host, vec![cert_der]),
        edge_addr,
    )
}

async fn run_client(args: &HashMap<String, String>) {
    let (transport, _edge_addr) = build_client_transport(args);
    let transport = Arc::new(transport);
    let target_addr: SocketAddr = arg(args, "target-addr", "").parse().expect("--target-addr");
    let mode = arg(args, "mode", "throughput");
    let concurrency: usize = arg(args, "concurrency", "1").parse().expect("--concurrency");
    let duration_secs: u64 = arg(args, "duration-secs", "5")
        .parse()
        .expect("--duration-secs");
    let payload_bytes: usize = arg(args, "payload-bytes", "1200")
        .parse()
        .expect("--payload-bytes");

    match mode.as_str() {
        "throughput" => run_throughput(transport, target_addr, concurrency, duration_secs, payload_bytes).await,
        "latency" => run_latency(transport, target_addr, payload_bytes).await,
        "churn" => run_churn(transport, target_addr, concurrency, duration_secs).await,
        other => {
            eprintln!("unknown --mode {other}");
            std::process::exit(2);
        }
    }
}

fn csv(edge: &str, mode: &str, concurrency: usize, metric: &str, value: f64, unit: &str) {
    println!("CSV,{edge},{mode},{concurrency},{metric},{value:.4},{unit}");
}

async fn run_throughput(
    transport: Arc<MasqueTransport>,
    target_addr: SocketAddr,
    concurrency: usize,
    duration_secs: u64,
    payload_bytes: usize,
) {
    let deadline = Instant::now() + Duration::from_secs(duration_secs);
    let handshake_times = Arc::new(std::sync::Mutex::new(Vec::<f64>::new()));
    let bytes_sent = Arc::new(AtomicU64::new(0));

    let mut tasks = Vec::with_capacity(concurrency);
    for _ in 0..concurrency {
        let transport = transport.clone();
        let handshake_times = handshake_times.clone();
        let bytes_sent = bytes_sent.clone();
        tasks.push(tokio::spawn(async move {
            let t0 = Instant::now();
            let conn = match transport.dial(target_addr).await {
                Ok(c) => c,
                Err(e) => {
                    eprintln!("dial failed: {e}");
                    return;
                }
            };
            let handshake_ms = t0.elapsed().as_secs_f64() * 1000.0;
            handshake_times.lock().unwrap().push(handshake_ms);

            let payload = vec![0x5Au8; payload_bytes];
            while Instant::now() < deadline {
                if conn.send(&payload).await.is_ok() {
                    bytes_sent.fetch_add(payload.len() as u64, Ordering::Relaxed);
                } else {
                    break;
                }
                // `MasqueConnection::send` wraps `quinn::Connection::send_datagram`,
                // a synchronous, non-blocking call with no internal `.await`
                // suspension point (RFC 9221 datagrams are fire-and-forget by
                // design). A tight loop with no yield point never gives the
                // tokio runtime a chance to actually drive quinn's endpoint
                // I/O task, so quinn's bounded outgoing-datagram buffer
                // (`DATAGRAM_BUFFER_SIZE` = 64 KiB, see helix-masque's
                // `quic.rs`) fills and silently drops older undelivered
                // datagrams — observed directly: an un-yielded loop reported
                // >100 Gbps "accepted" locally while the sink received zero
                // bytes. Yielding after every send lets the real socket I/O
                // interleave with sending, which is also a more honest model
                // of a real sender than a CPU-bound blast loop.
                tokio::task::yield_now().await;
            }
            let _ = conn.close().await;
        }));
    }
    for t in tasks {
        let _ = t.await;
    }

    let times = handshake_times.lock().unwrap();
    let mean_handshake_ms = if times.is_empty() {
        0.0
    } else {
        times.iter().sum::<f64>() / times.len() as f64
    };
    let total_bytes = bytes_sent.load(Ordering::Relaxed) as f64;
    let offered_mbps = (total_bytes * 8.0) / (duration_secs as f64) / 1_000_000.0;

    csv("client", "throughput", concurrency, "handshake_setup_ms", mean_handshake_ms, "ms");
    csv("client", "throughput", concurrency, "client_offered_mbps", offered_mbps, "Mbps");
}

async fn run_latency(transport: Arc<MasqueTransport>, target_addr: SocketAddr, payload_bytes: usize) {
    const ITERATIONS: usize = 200;
    let conn = transport.dial(target_addr).await.expect("dial (latency)");
    let payload = vec![0x5Au8; payload_bytes];
    let mut recv_buf = vec![0u8; 65536];
    let mut rtts_ms = Vec::with_capacity(ITERATIONS);

    for _ in 0..ITERATIONS {
        let t0 = Instant::now();
        if conn.send(&payload).await.is_err() {
            break;
        }
        match tokio::time::timeout(Duration::from_secs(2), conn.recv(&mut recv_buf)).await {
            Ok(Ok(_n)) => rtts_ms.push(t0.elapsed().as_secs_f64() * 1000.0),
            _ => break,
        }
    }
    let _ = conn.close().await;

    rtts_ms.sort_by(|a, b| a.partial_cmp(b).unwrap());
    let p50 = percentile(&rtts_ms, 50.0);
    let p99 = percentile(&rtts_ms, 99.0);
    csv("client", "latency", 1, "p50_ms", p50, "ms");
    csv("client", "latency", 1, "p99_ms", p99, "ms");
    csv("client", "latency", 1, "samples", rtts_ms.len() as f64, "count");
}

fn percentile(sorted: &[f64], pct: f64) -> f64 {
    if sorted.is_empty() {
        return 0.0;
    }
    let idx = ((pct / 100.0) * (sorted.len() as f64 - 1.0)).round() as usize;
    sorted[idx.min(sorted.len() - 1)]
}

async fn run_churn(
    transport: Arc<MasqueTransport>,
    target_addr: SocketAddr,
    concurrency: usize,
    duration_secs: u64,
) {
    let deadline = Instant::now() + Duration::from_secs(duration_secs);
    let completed = Arc::new(AtomicU64::new(0));
    let failed = Arc::new(AtomicU64::new(0));

    let mut tasks = Vec::with_capacity(concurrency);
    for _ in 0..concurrency {
        let transport = transport.clone();
        let completed = completed.clone();
        let failed = failed.clone();
        tasks.push(tokio::spawn(async move {
            let payload = [0x5Au8; 64];
            let mut recv_buf = vec![0u8; 1500];
            while Instant::now() < deadline {
                match transport.dial(target_addr).await {
                    Ok(conn) => {
                        let ok = conn.send(&payload).await.is_ok()
                            && tokio::time::timeout(Duration::from_secs(1), conn.recv(&mut recv_buf))
                                .await
                                .map(|r| r.is_ok())
                                .unwrap_or(false);
                        let _ = conn.close().await;
                        if ok {
                            completed.fetch_add(1, Ordering::Relaxed);
                        } else {
                            failed.fetch_add(1, Ordering::Relaxed);
                        }
                    }
                    Err(_) => {
                        failed.fetch_add(1, Ordering::Relaxed);
                    }
                }
            }
        }));
    }
    for t in tasks {
        let _ = t.await;
    }

    let handshakes_per_sec = completed.load(Ordering::Relaxed) as f64 / duration_secs as f64;
    csv("client", "churn", concurrency, "handshakes_per_sec", handshakes_per_sec, "per_sec");
    csv("client", "churn", concurrency, "failed_handshakes", failed.load(Ordering::Relaxed) as f64, "count");
}
