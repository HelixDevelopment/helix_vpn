// Command go_edge_bench is the HVPN-P0-045 benchmark driver for the Go
// go-edge MASQUE termination path (G4 A/B bench,
// HelixVPN-Phase0-Spike.md §7/§7.2). It mirrors the sibling
// rust_edge_bench tool's CLI and CSV output so the orchestrating bash
// script can drive both edges identically.
//
// This binary drives the REAL masqueedge.NewServer + masque-go/quic-go
// stack (the exact production code cmd/go-edge's main.go also calls) and
// the REAL masqueedge.DialGateway client helper — no fake/mocked MASQUE
// termination.
//
// # Honest scope
//
//   - Loopback only (127.0.0.1); this sandbox has no passwordless sudo,
//     matching masqueedge's own documented honest-scope constraint.
//   - Neither edge (Go or Rust) has a real kernel-WireGuard/boringtun
//     gateway-socket integration yet — this benchmarks the MASQUE
//     termination + gateway-relay hand-off data path itself, not an
//     end-to-end WireGuard tunnel.
//   - Two roles run as separate OS processes ("server"/"client"),
//     mirroring iperf3's -s/-c split, so the orchestrating script
//     attributes CPU/RSS to the edge process alone via /proc/<pid>.
//
// This module is standalone (own go.mod, replace-directive path
// dependency on submodules/helix_go) — it never edits helix_go.
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	masque "github.com/quic-go/masque-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/vasic-digital/helix_go/pkg/masqueedge"
	"github.com/yosida95/uritemplate/v3"
)

// connectUDPPath mirrors the unexported constant in
// github.com/vasic-digital/helix_go/pkg/masqueedge/server.go verbatim
// (read directly from source, not guessed) — needed here because a
// separate client process cannot call Server.Template() in-process; it
// reconstructs the identical, documented URI template shape instead.
const connectUDPPath = "/connect-udp"

func main() {
	role := flag.String("role", "", "server|client")
	bindIP := flag.String("bind-ip", "127.0.0.1", "server: bind address")
	certOut := flag.String("cert-out", "", "server: path to write the self-signed leaf cert DER")
	edgeAddr := flag.String("edge-addr", "", "client: edge MASQUE address host:port")
	targetAddr := flag.String("target-addr", "", "client: gateway/sink address to relay to")
	certPath := flag.String("cert", "", "client: path to the server's cert DER")
	mode := flag.String("mode", "throughput", "client: throughput|latency|churn")
	concurrency := flag.Int("concurrency", 1, "client: concurrent flows")
	durationSecs := flag.Int("duration-secs", 5, "client: test duration in seconds")
	payloadBytes := flag.Int("payload-bytes", 1200, "client: payload size in bytes")
	flag.Parse()

	switch *role {
	case "server":
		runServer(*bindIP, *certOut)
	case "client":
		runClient(*edgeAddr, *targetAddr, *certPath, *mode, *concurrency, *durationSecs, *payloadBytes)
	default:
		fmt.Fprintln(os.Stderr, "usage: go_edge_bench --role {server|client} ...")
		os.Exit(2)
	}
}

// genSelfSigned creates a fresh, ephemeral, loopback-scoped self-signed
// leaf certificate. Deliberately NOT reusing
// masqueedge.GenerateSelfSignedTLS: that helper does not expose the raw
// CA DER bytes needed to hand the certificate to a client running in a
// SEPARATE OS process, so this tool generates its own (same pattern as
// the sibling rust_edge_bench tool generating its own cert via rcgen
// rather than reaching into helix_edge's test-only helpers). This does
// not modify masqueedge/go-edge in any way.
func genSelfSigned() (tls.Certificate, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: priv}, der, nil
}

// countingSink is a real loopback UDP listener that counts bytes/packets
// received (never echoes) — the one-way throughput-test sink. Its stats
// are periodically printed to stdout as authoritative sink-side captured
// evidence, mirroring rust_edge_bench's COUNT_STATS lines.
func startCountingSink(bindIP string) (addr string) {
	udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(bindIP, "0"))
	if err != nil {
		log.Fatalf("resolve count sink addr: %v", err)
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("bind count sink: %v", err)
	}
	var bytesTotal, packetsTotal uint64
	go func() {
		buf := make([]byte, 65536)
		for {
			n, _, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}
			atomic.AddUint64(&bytesTotal, uint64(n))
			atomic.AddUint64(&packetsTotal, 1)
		}
	}()
	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			fmt.Printf("COUNT_STATS ts=%d bytes=%d packets=%d\n",
				time.Now().UnixMilli(), atomic.LoadUint64(&bytesTotal), atomic.LoadUint64(&packetsTotal))
		}
	}()
	return conn.LocalAddr().String()
}

func runServer(bindIP, certOut string) {
	if certOut == "" {
		log.Fatal("--cert-out is required for --role server")
	}
	cert, certDER, err := genSelfSigned()
	if err != nil {
		log.Fatalf("genSelfSigned: %v", err)
	}
	if err := os.WriteFile(certOut, certDER, 0o644); err != nil {
		log.Fatalf("write --cert-out: %v", err)
	}

	serverTLS := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{http3.NextProtoH3},
	}

	srv, err := masqueedge.NewServer(net.JoinHostPort(bindIP, "0"), serverTLS)
	if err != nil {
		log.Fatalf("NewServer: %v", err)
	}
	go func() {
		if err := srv.Serve(); err != nil {
			log.Printf("edge serve loop ended: %v", err)
		}
	}()

	echoGateway, err := masqueedge.NewLoopbackGateway(net.JoinHostPort(bindIP, "0"))
	if err != nil {
		log.Fatalf("NewLoopbackGateway: %v", err)
	}

	countAddr := startCountingSink(bindIP)

	fmt.Printf("READY edge=%s echo_sink=%s count_sink=%s cert=%s pid=%d\n",
		srv.Addr(), echoGateway.Addr(), countAddr, certOut, os.Getpid())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
}

func loadClientCAs(certPath string) *x509.CertPool {
	der, err := os.ReadFile(certPath)
	if err != nil {
		log.Fatalf("read --cert: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		log.Fatalf("parse --cert DER: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return pool
}

func edgeTemplate(edgeAddr string) *uritemplate.Template {
	host, port, err := net.SplitHostPort(edgeAddr)
	if err != nil {
		log.Fatalf("--edge-addr: %v", err)
	}
	return uritemplate.MustNew(fmt.Sprintf("https://%s:%s%s?h={target_host}&p={target_port}", host, port, connectUDPPath))
}

func dial(ctx context.Context, tmpl *uritemplate.Template, clientCAs *x509.CertPool, targetAddr string) (*masque.Conn, error) {
	return masqueedge.DialGateway(ctx, tmpl, clientCAs, targetAddr)
}

func runClient(edgeAddr, targetAddr, certPath, mode string, concurrency, durationSecs, payloadBytes int) {
	clientCAs := loadClientCAs(certPath)
	tmpl := edgeTemplate(edgeAddr)

	switch mode {
	case "throughput":
		runThroughput(tmpl, clientCAs, targetAddr, concurrency, durationSecs, payloadBytes)
	case "latency":
		runLatency(tmpl, clientCAs, targetAddr, payloadBytes)
	case "churn":
		runChurn(tmpl, clientCAs, targetAddr, concurrency, durationSecs)
	default:
		fmt.Fprintf(os.Stderr, "unknown --mode %s\n", mode)
		os.Exit(2)
	}
}

func csvLine(edge, mode string, concurrency int, metric string, value float64, unit string) {
	fmt.Printf("CSV,%s,%s,%d,%s,%.4f,%s\n", edge, mode, concurrency, metric, value, unit)
}

func runThroughput(tmpl *uritemplate.Template, clientCAs *x509.CertPool, targetAddr string, concurrency, durationSecs, payloadBytes int) {
	deadline := time.Now().Add(time.Duration(durationSecs) * time.Second)
	var wg sync.WaitGroup
	var bytesSent uint64
	handshakeMs := make([]float64, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			t0 := time.Now()
			conn, err := dial(ctx, tmpl, clientCAs, targetAddr)
			if err != nil {
				log.Printf("dial failed: %v", err)
				return
			}
			handshakeMs[i] = float64(time.Since(t0).Microseconds()) / 1000.0
			defer conn.Close()

			payload := make([]byte, payloadBytes)
			for time.Now().Before(deadline) {
				if _, err := conn.WriteTo(payload, nil); err != nil {
					break
				}
				atomic.AddUint64(&bytesSent, uint64(payloadBytes))
			}
		}(i)
	}
	wg.Wait()

	var sum float64
	for _, v := range handshakeMs {
		sum += v
	}
	meanHandshake := 0.0
	if concurrency > 0 {
		meanHandshake = sum / float64(concurrency)
	}
	offeredMbps := (float64(atomic.LoadUint64(&bytesSent)) * 8.0) / float64(durationSecs) / 1_000_000.0

	csvLine("client", "throughput", concurrency, "handshake_setup_ms", meanHandshake, "ms")
	csvLine("client", "throughput", concurrency, "client_offered_mbps", offeredMbps, "Mbps")
}

func runLatency(tmpl *uritemplate.Template, clientCAs *x509.CertPool, targetAddr string, payloadBytes int) {
	const iterations = 200
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn, err := dial(ctx, tmpl, clientCAs, targetAddr)
	if err != nil {
		log.Fatalf("dial (latency): %v", err)
	}
	defer conn.Close()

	payload := make([]byte, payloadBytes)
	buf := make([]byte, 65536)
	rtts := make([]float64, 0, iterations)

	for i := 0; i < iterations; i++ {
		t0 := time.Now()
		if _, err := conn.WriteTo(payload, nil); err != nil {
			break
		}
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, _, err := conn.ReadFrom(buf); err != nil {
			break
		}
		rtts = append(rtts, float64(time.Since(t0).Microseconds())/1000.0)
	}

	sort.Float64s(rtts)
	p50 := percentile(rtts, 50)
	p99 := percentile(rtts, 99)
	csvLine("client", "latency", 1, "p50_ms", p50, "ms")
	csvLine("client", "latency", 1, "p99_ms", p99, "ms")
	csvLine("client", "latency", 1, "samples", float64(len(rtts)), "count")
}

func percentile(sorted []float64, pct float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(pct/100.0*float64(len(sorted)-1) + 0.5)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func runChurn(tmpl *uritemplate.Template, clientCAs *x509.CertPool, targetAddr string, concurrency, durationSecs int) {
	deadline := time.Now().Add(time.Duration(durationSecs) * time.Second)
	var wg sync.WaitGroup
	var completed, failed uint64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payload := make([]byte, 64)
			buf := make([]byte, 1500)
			for time.Now().Before(deadline) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				conn, err := dial(ctx, tmpl, clientCAs, targetAddr)
				cancel()
				if err != nil {
					atomic.AddUint64(&failed, 1)
					continue
				}
				ok := false
				if _, err := conn.WriteTo(payload, nil); err == nil {
					conn.SetReadDeadline(time.Now().Add(1 * time.Second))
					if _, _, err := conn.ReadFrom(buf); err == nil {
						ok = true
					}
				}
				conn.Close()
				if ok {
					atomic.AddUint64(&completed, 1)
				} else {
					atomic.AddUint64(&failed, 1)
				}
			}
		}()
	}
	wg.Wait()

	handshakesPerSec := float64(atomic.LoadUint64(&completed)) / float64(durationSecs)
	csvLine("client", "churn", concurrency, "handshakes_per_sec", handshakesPerSec, "per_sec")
	csvLine("client", "churn", concurrency, "failed_handshakes", float64(atomic.LoadUint64(&failed)), "count")
}
