# HelixVPN -- Overlay DNS Management Service Design

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Status:** active -- concrete design for the overlay DNS server, zone management,
and split-DNS resolver shim
**Authority:** This document is the binding design for the HelixVPN overlay DNS
service. It extends the IPAM design ([ipam §7]) with concrete architecture, Go
types, zone structure, and dynamic-update mechanics. Where this document
disagrees with [ipam §7], [ipam §7] wins until this document is amended and that
spec updated.

---

## 1. Problem Statement

The HelixVPN overlay assigns every LAN host a unique 4via6 address (see [ipam]).
But an IPv6 address is not usable by an application unless the application can
**discover** it. Users type hostnames, not `fd12:3456:789a:4636:0:7:c0a8:14`.

The overlay DNS service solves three problems:

| Problem | DNS mechanism |
|---|---|
| Hostname-to-4via6 resolution | AAAA records mapping `<host>.<connector>.<tenant>.helix.vpn.` to 4via6 addresses |
| Reverse lookup (4via6-to-hostname) | PTR records in the `ip6.arpa` tree derived from the ULA /48 |
| Service discovery | SRV records for connector-exposed services (e.g. `_http._tcp.warehouse.acme-corp.helix.vpn.`) |
| IPv4-only client compatibility | Synthesized A records (via DNS64-like prefix embedding) for legacy apps |
| Split DNS | Overlay queries route to the VPN DNS server; everything else goes to the local resolver |

Without this service, every client would need a static `/etc/hosts` mapping
for every LAN host across every connector -- unmaintainable at any scale.

### 1.1 Prior art

- **RFC 1035 (DNS):** The authoritative wire protocol this server implements
  for zone data (SOA, NS, AAAA, PTR, SRV).
- **RFC 3596 (AAAA):** IPv6 address records in DNS -- used here for 4via6.
- **RFC 2782 (SRV):** Service location records -- used here for
  connector-exposed services.
- **RFC 6147 (DNS64):** IPv4-to-IPv6 DNS synthesis for IPv6-only clients --
  analogous to our A-to-AAAA synthesis for LAN hosts behind 4via6.
- **CoreDNS:** The plugin-based DNS server in Go. HelixVPN's DNS server
  reuses `github.com/miekg/dns` (the library CoreDNS is built on) for
  wire-format handling, but implements its own authoritative zone backend.
- **Tailscale MagicDNS:** Resolves `<hostname>.<tailnet-name>.ts.net` to
  Tailscale IPv4/IPv6 addresses. HelixVPN's overlay DNS is architecturally
  similar but operates over 4via6 addresses in a tenant-scoped ULA /48.

---

## 2. Architecture

### 2.1 System topology

```
                         Overlay Network (WG tunnels)
  ┌──────────┐                ┌──────────────┐
  │  Client  │◄──────────────►│   Gateway    │
  │ (alice)  │   WG tunnel    │ fd..::1      │
  │          │                │              │
  │ helix-   │   DNS query    │ helix-dns    │
  │ dns-shim│◄──────────────►│ :53 (UDP)    │
  │ 127.0.0.1│                │              │
  │ :53      │                │ zone store   │
  └──────────┘                │ (auth. data) │
                              └──────┬───────┘
                                     │
                              control-plane events
                              (device.enrolled,
                               connector.attached,
                               connector.prefixes_changed)
                                     │
                              ┌──────▼───────┐
                              │  Coordinator │
                              │  (zone mut.) │
                              └──────────────┘
```

### 2.2 Component placement

| Component | Where it runs | Role |
|---|---|---|
| **helix-dns** (authoritative server) | Gateway node | Serves authoritative DNS for `<tenant>.helix.vpn.`; listens on `<ula48>::1:53` UDP+TCP |
| **zone store** | Gateway (in-memory, hydrated from Postgres at startup) | The backing data: zones, records, SOA serials |
| **zone mutator** | Coordinator (control plane) | Updates zone data when topology/policy events fire |
| **helix-dns-shim** (split-DNS resolver) | Every client device | Intercepts DNS queries, routes overlay-domain queries to gateway DNS, forwards the rest to system resolver |

### 2.3 Why a dedicated authoritative server instead of CoreDNS+plugin

CoreDNS with a custom `helixvpn` plugin is the obvious alternative. This
design chooses a standalone `helix-dns` server instead because:

1. **The zone data is dynamic, not file-backed.** Records come from the
   control-plane event stream, not static zone files. A custom server keeps
   the zone-mutation path simple (no CoreDNS reloads or etcd middleware).
2. **The DNS server IS the single source of truth for hostname-to-4via6.**
   The IPAM service derives 4via6 addresses; the DNS service derives
   hostname-to-4via6 mappings. These are coupled concerns that belong in one
   codebase.
3. **One binary, one responsibility.** A small `miekg/dns`-based server with
   in-memory zone storage and an event-driven mutation path is ~500 lines of
   Go, testable end-to-end without external infra.

The server still uses `github.com/miekg/dns` for wire-format encode/decode
so it speaks proper DNS on the wire -- it just owns its own zone backend.

---

## 3. Zone Structure

### 3.1 Zone hierarchy

```
helix.vpn.                            (SOA -- root of the overlay DNS namespace)
  |
  +-- <tenant>.helix.vpn.             (tenant zone -- one per tenant)
        |
        +-- <connector>.<tenant>.helix.vpn.   (connector sub-zone -- one per connector)
        |     |
        |     +-- cam1.warehouse.acme-corp.helix.vpn.     AAAA  fd..:4636:0:7:c0a8:14
        |     +-- printer.warehouse.acme-corp.helix.vpn.  AAAA  fd..:4636:0:7:c0a8:32
        |     +-- _http._tcp.warehouse.acme-corp.helix.vpn.  SRV  0 10 8080 cam1...
        |
        +-- office.acme-corp.helix.vpn.
              +-- cam1.office.acme-corp.helix.vpn.  AAAA  fd..:4636:0:8:c0a8:14
```

Every connector gets its own DNS sub-zone. This is load-bearing for the
4via6 collision-free property: `cam1.warehouse` and `cam1.office` have the
same LAN IPv4 (`192.168.1.20`) but resolve to **different** 4via6 AAAA
records because their connector sub-zones embed different site IDs.

### 3.2 FQDN naming convention

```
<hostname>.<connector-name>.<tenant-name>.helix.vpn.
```

| Component | Source | Example |
|---|---|---|
| `hostname` | Connector-side host discovery (mDNS/NetBIOS/ARP/DHCP lease hostname, or user-assigned label) | `cam1`, `printer-3` |
| `connector-name` | Connector's `Node.Name` from the devices table, lowercased | `warehouse`, `office-nyc` |
| `tenant-name` | Tenant's canonical name slug, lowercased | `acme-corp` |
| `helix.vpn.` | Fixed root domain | `helix.vpn.` |

**DNS label constraints (RFC 1035 §2.3.1 enforced):**
- Each label: 1-63 octets, `[a-z0-9]([a-z0-9-]*[a-z0-9])?`
- Transform: strip leading/trailing non-alphanum, lowercase, replace
  consecutive non-alphanum with single `-`, truncate >63 to 63.

### 3.3 Reverse zone (PTR)

PTR records live under `ip6.arpa` derived from the tenant's ULA /48:

```
<reversed-4via6>.ip6.arpa.  PTR  <fqdn>
```

Example for `fd12:3456:789a:4636:0:7:c0a8:14` (warehouse cam1):

```
4.1.0.0.8.a.0.c.0.0.0.0.7.0.0.0.0.0.0.0.6.3.6.4.a.9.8.7.6.5.4.3.2.1.d.f.ip6.arpa.
  PTR  cam1.warehouse.acme-corp.helix.vpn.
```

The reverse zone is synthesized at query time -- it is NOT pre-populated as
individual PTR records in the zone store. The DNS server computes the nibble
expansion of the 4via6 address, reverse-maps through the zone registry, and
returns the PTR if the host exists. This avoids storing 2N records (AAAA +
PTR) when the PTR is mechanically derivable.

### 3.4 Service discovery (SRV)

Connectors may expose services that clients discover via SRV records. The
connector's `AdvertisedPrefixes` carries an optional `Services` field:

```
_<service>._<proto>.<connector>.<tenant>.helix.vpn.  SRV  <priority> <weight> <port> <target>
```

Examples:

| Query | SRV Response |
|---|---|
| `_http._tcp.warehouse.acme-corp.helix.vpn.` | `0 10 8080 cam1.warehouse.acme-corp.helix.vpn.` |
| `_rtsp._tcp.warehouse.acme-corp.helix.vpn.` | `0 5 554 cam1.warehouse.acme-corp.helix.vpn.` |

Services are registered by the connector at enrollment/update time via the
`AdvertisePrefixes` + `Services` fields. The zone mutator synthesizes SRV
records from the service declarations.

---

## 4. Record Types

### 4.1 AAAA -- 4via6 overlay address (primary record)

Every LAN host behind a connector gets exactly one AAAA record mapping its
FQDN to its 4via6 overlay address.

```
cam1.warehouse.acme-corp.helix.vpn.  300  IN  AAAA  fd12:3456:789a:4636:0:7:c0a8:14
```

**Derivation (zero-allocation, no DB lookup per query):**

```go
// ResolveAAAA derives the AAAA record for a LAN host from zone metadata.
// It does NOT query a record table -- it encodes the 4via6 address from
// the zone's site ID and the host's IPv4, which are loaded once at zone
// creation and cached in the in-memory zone struct.
func (z *DNSZone) ResolveAAAA(hostname string) (netip.Addr, bool) {
    host, ok := z.Hosts[hostname]
    if !ok {
        return netip.Addr{}, false
    }
    return ipam.Encode4via6(z.TenantPrefix, z.SiteID, host.IPv4), true
}
```

**TTL:** 300 seconds (5 minutes). This is intentionally short because:
- A connector prefix change updates every AAAA in the zone.
- A LAN host IP change (DHCP renewal) updates one AAAA.
- Clients cache for 5 min; stale cache is better than no resolution.
- The zone data is in-memory at the gateway; query cost is negligible.

### 4.2 PTR -- reverse lookup

```
4.1.0.0.8.a.0.c.[...].1.d.f.ip6.arpa.  300  IN  PTR  cam1.warehouse.acme-corp.helix.vpn.
```

**Derivation (query-time synthesis):** The DNS server reverses the 4via6
address, extracts the (siteID, IPv4) pair via 4via6 decode, looks up the
siteID in the zone registry to find the connector zone, then looks up the
IPv4 in that zone's host map. No pre-populated PTR records. The query-time
cost is one 4via6 decode + two map lookups -- well under 1 microsecond.

### 4.3 SOA -- Start of Authority

Every zone apex carries an SOA record:

```
acme-corp.helix.vpn.  3600  IN  SOA  ns1.helix.vpn. admin.helix.vpn. (
    2026070801  ; serial (YYYYMMDDNN -- bumped on every zone mutation)
    3600        ; refresh
    600         ; retry
    86400       ; expire
    300         ; minimum TTL (negative caching)
)
```

The serial is bumped by the zone mutator on every mutation (host add/remove,
prefix change, service change). The serial format is `YYYYMMDDNN` where NN
is a per-day counter (00-99), giving predictable sorting for human operators
and enough headroom for 100 mutations/day (far above expected rate).

### 4.4 NS -- Name Server

```
acme-corp.helix.vpn.  3600  IN  NS  ns1.helix.vpn.
```

The gateway's overlay DNS is the single authoritative nameserver for the
tenant zone. There is no secondary NS in Phase 1 -- the gateway is a single
node and zone transfers to a standby are a Phase-2 concern. If the gateway
is unreachable, overlay DNS is unavailable (same as the overlay itself being
unavailable).

### 4.5 SRV -- Service discovery

```
_http._tcp.warehouse.acme-corp.helix.vpn.  300  IN  SRV  0 10 8080 cam1.warehouse.acme-corp.helix.vpn.
```

Fields per RFC 2782: `Priority Weight Port Target`. The Target MUST be an
FQDN with an AAAA record in the same zone (no CNAME chains, no out-of-zone
targets). The zone mutator validates this at insertion time.

---

## 5. Dynamic Updates

### 5.1 Event-driven zone mutation

The DNS zone is NOT edited via DNS UPDATE (RFC 2136) messages. It is mutated
by the **zone mutator** -- a coordinator component that subscribes to the
control-plane event stream and applies zone deltas in response to topology
changes.

```
Control Plane Event              Zone Mutation
────────────────────             ─────────────
connector.attached               CREATE sub-zone <connector>.<tenant>.helix.vpn.
                                 with SOA + NS + AAAA records for every
                                 registered LAN host

connector.prefixes_changed       UPDATE all AAAA records in the connector's
                                 sub-zone (site ID changed or LAN hosts
                                 added/removed)

connector.services_changed       UPSERT SRV records for the connector's
                                 declared services

device.enrolled                  (no zone mutation -- clients don't get
                                 sub-zones unless they are connectors)

device.revoked (connector)       DELETE sub-zone and all its records;
                                 downstream nodes still caching get
                                 NXDOMAIN after TTL expires

host.discovered (connector)      INSERT AAAA + synthesized PTR for the
                                 newly discovered LAN host

host.lost (connector)            DELETE AAAA for the departed LAN host

tenant.deleted                   DELETE the entire tenant zone and all
                                 sub-zones
```

### 5.2 Zone mutator algorithm

```
OnEvent(event):
  switch event.Type:
    case ConnectorAttached:
      zone := NewDNSZone(event.TenantID, event.ConnectorID,
                         event.ConnectorName, event.SiteID,
                         event.TenantPrefix)
      for each host in event.InitialHosts:
        zone.UpsertHost(host.Name, host.IPv4)
      store.UpsertZone(zone)

    case ConnectorPrefixesChanged:
      zone := store.Zone(event.ConnectorID)
      // Rebuild host set from delta
      for each host in event.AddedHosts:
        zone.UpsertHost(host.Name, host.IPv4)
      for each host in event.RemovedHosts:
        zone.DeleteHost(host.Name)
      zone.BumpSerial()

    case ConnectorServicesChanged:
      zone := store.Zone(event.ConnectorID)
      zone.ReplaceServices(event.Services)
      zone.BumpSerial()

    case DeviceRevoked:
      if event.Kind == Connector:
        store.DeleteZone(event.DeviceID)

    case TenantDeleted:
      store.DeleteTenantZones(event.TenantID)
```

### 5.3 SOA serial bump contract

Every zone mutation that changes record data MUST bump the SOA serial.
Bumping is done by the zone mutator, not by the DNS query path. The bump
is atomic with the mutation (the zone store uses an `sync.RWMutex`; the
mutator holds the write lock, appends the mutation, bumps the serial, and
releases).

Two mutations in the same event (e.g. `ConnectorPrefixesChanged` adding 5
hosts and removing 2) produce exactly ONE serial bump -- the event is the
unit of mutation, not the individual record.

### 5.4 NOTIFY for cache invalidation

When a zone mutates, the DNS server sends DNS NOTIFY (RFC 1996) to all
registered secondaries. In Phase 1 there are no secondaries, so NOTIFY is
a no-op. The mechanism is wired but dormant -- the `NotifyPeers` field on
the zone is populated from an allowlist, and the server sends NOTIFY
messages to each peer after each serial bump. This is future-proofed, not
tested against real secondaries.

---

## 6. Split DNS

### 6.1 The problem

A client device has two DNS namespaces:

1. **Overlay namespace** (`*.helix.vpn.`) -- resolved by the gateway's overlay
   DNS server at `<ula48>::1:53`.
2. **Internet namespace** (everything else) -- resolved by the system's
   configured DNS resolver (DHCP-provided, `8.8.8.8`, etc.).

The client MUST route overlay queries to the VPN DNS and everything else to
the system resolver. Leaking overlay queries to the internet is a privacy
leak (internal hostnames exposed to public DNS). Leaking internet queries to
the VPN DNS adds latency (tunnel round-trip) for no benefit (the VPN DNS is
not a recursive resolver).

### 6.2 helix-dns-shim architecture

```
Application                    helix-dns-shim                 Upstream
  │                                │                              │
  │-- resolve("cam1.wh...") -->    │                              │
  │                                │-- match *.helix.vpn.?        │
  │                                │   YES --> forward to         │
  │                                │   [fd..::1]:53 (via overlay) │
  │                                │                              │
  │-- resolve("google.com") -->    │                              │
  │                                │-- match *.helix.vpn.?        │
  │                                │   NO --> forward to          │
  │                                │   system resolver ──────────►│
```

The shim is a local DNS proxy listening on `127.0.0.1:53` (or `::1:53`).
It is installed by the client agent on enrollment and removed on disconnect.

### 6.3 Shim implementation

```go
// DNSShim is a split-DNS forwarding proxy that runs on every client device.
type DNSShim struct {
    ListenAddr   string       // "127.0.0.1:53"
    OverlayDNS   string       // "[fd12:3456:789a::1]:53" -- gateway DNS
    OverlayDomain string      // "acme-corp.helix.vpn." -- tenant root zone
    Upstream     []string     // system resolver addresses (from /etc/resolv.conf)
    Transport    net.PacketConn // UDP socket bound to ListenAddr
}

func (s *DNSShim) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
    q := r.Question[0]

    if strings.HasSuffix(q.Name, s.OverlayDomain) || q.Name == "helix.vpn." {
        // Overlay query: forward to gateway DNS over the WireGuard tunnel.
        // The overlay DNS server is reachable at <ula48>::1:53 via the
        // WireGuard interface -- no separate transport needed.
        resp, err := s.forwardOverlay(r)
        if err != nil {
            // Gateway unreachable: return SERVFAIL (do not fall through to
            // internet resolver -- that would leak internal hostnames).
            m := new(dns.Msg).SetRcode(r, dns.RcodeServerFailure)
            w.WriteMsg(m)
            return
        }
        w.WriteMsg(resp)
        return
    }

    // Non-overlay query: forward to system resolver.
    resp, err := s.forwardUpstream(r)
    if err != nil {
        m := new(dns.Msg).SetRcode(r, dns.RcodeServerFailure)
        w.WriteMsg(m)
        return
    }
    w.WriteMsg(resp)
}
```

### 6.4 OS integration

| Platform | Mechanism |
|---|---|
| **Linux (systemd-resolved)** | `resolvectl domain <iface> '~helix.vpn.'` + `resolvectl dns <iface> <ula48>::1` -- routes overlay queries to VPN DNS, everything else to DHCP-provided |
| **Linux (resolv.conf)** | The shim replaces `/etc/resolv.conf` `nameserver` with `127.0.0.1`; the shim's upstream list is the original nameserver |
| **macOS** | `/etc/resolver/helix.vpn` with `nameserver <ula48>::1` |
| **Windows** | NRPT (Name Resolution Policy Table) rule via `Add-DnsClientNrptRule` |
| **Android** | `DnsResolver` API (API 29+) with per-network resolver; shim runs as a VPN service add-on |

In all cases the `NetworkMap.DNS` field carries the gateway's overlay DNS
address. The client agent reads it and configures the OS-level DNS routing.

---

## 7. Legacy IPv4 -- DNS64 Synthesis

### 7.1 The problem

An IPv4-only application on a client (e.g. a legacy NVR viewer that calls
`getaddrinfo` with `AF_INET`) cannot consume a 4via6 AAAA record. It needs
an **A record** (IPv4 address). But the LAN host's real IPv4 (`192.168.1.20`)
is not routable from the client's perspective -- the client is in a different
site, possibly with the same private range.

The solution: synthesize a **unique, routable A record** from the 4via6
address, using DNS64-like embedding.

### 7.2 A-record synthesis

When the DNS server receives an **A** query for a name in the overlay zone,
it synthesizes an A record by extracting the embedded IPv4 from the 4via6
address and returning it verbatim. The returned IPv4 is the LAN host's
**actual** IPv4 -- but it is only reachable through the overlay because the
client's routing table directs traffic through the WireGuard tunnel.

```
Query:  cam1.warehouse.acme-corp.helix.vpn.  IN  A
Answer: cam1.warehouse.acme-corp.helix.vpn.  300  IN  A  192.168.1.20
```

This is NOT DNS64 per RFC 6147 (which embeds IPv4 into a well-known /96 prefix).
It is simpler: the DNS server returns the host's real IPv4 because the
**routing layer** (WireGuard `AllowedIPs`) already ensures that packets
destined for that IPv4 are encapsulated and sent to the correct connector.

**When this works:** The client has a WireGuard route for the LAN's IPv4
prefix (e.g. `192.168.1.0/24`) pointing to the connector. The A record
returns `192.168.1.20`; the kernel routes it through the WireGuard tunnel;
the connector DNATs it to the real LAN host. This path exists in the
CGNAT fallback ([ipam §10]) and in the 4via6 path when the connector
advertises LAN prefixes directly.

**When this does NOT work:** Two connectors advertise the same IPv4 range
(e.g. both `192.168.1.0/24`). An A record returning `192.168.1.20` is
ambiguous -- the client cannot know which connector to route it through.
In this case the DNS server MUST return the AAAA (4via6) record and
MUST NOT synthesize an A record. The server detects the ambiguity by
checking whether the tenant has >1 connector advertising the same IPv4
prefix -- if so, A-record queries for hosts in those ranges return
`RCode=NXDOMAIN` for the A type while still returning AAAA.

### 7.3 A-record synthesis rules

```go
func (s *DNSServer) resolveA(zone *DNSZone, hostname string, qtype uint16) ([]dns.RR, bool) {
    host, ok := zone.Hosts[hostname]
    if !ok {
        return nil, false
    }

    // AAAA records are always synthesized (primary record type).
    if qtype == dns.TypeAAAA || qtype == dns.TypeANY {
        aaaa := s.synthesizeAAAA(zone, host)
        return []dns.RR{aaaa}, true
    }

    // A records are synthesized ONLY when the tenant has no overlapping
    // IPv4 LAN prefixes (single-connector or distinct ranges).
    if qtype == dns.TypeA || qtype == dns.TypeANY {
        if !zone.HasOverlappingPrefixes {
            a := s.synthesizeA(host)
            return []dns.RR{a}, true
        }
        // Ambiguous: return NXDOMAIN for A type, but the caller still
        // returns AAAA if qtype==ANY.
        if qtype == dns.TypeA {
            return nil, false // triggers NXDOMAIN in caller
        }
    }

    return nil, false
}
```

### 7.4 Client-side A-record preference

The `NetworkMap.DNS` field carries a flag indicating A-record availability:

```json
{
  "dns": {
    "servers": ["fd12:3456:789a::1"],
    "synthesize_a": false
  }
}
```

`synthesize_a: false` means the tenant has overlapping prefixes and A-record
synthesis is disabled. The client agent reads this and configures the shim to
not forward A queries for overlay names -- they will get NXDOMAIN, and
applications using `AF_INET` will see resolution failure (which is correct
behavior when routing is ambiguous).

---

## 8. Go Types

### 8.1 Core types (`helix-go/internal/dns/types.go`)

```go
// Package dns implements the HelixVPN overlay DNS management service.
//
// The authoritative DNS server (DNSServer) serves AAAA, PTR, SRV, SOA,
// and NS records for the tenant overlay zone. Zone data is mutated by
// the zone mutator in response to control-plane events, not via DNS
// UPDATE messages.
//
// Design: docs/design/dns/DESIGN.md
package dns

import (
    "net/netip"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/miekg/dns"
)

// DNSZone is an authoritative DNS zone for a tenant or connector.
//
// A tenant has one root zone (<tenant>.helix.vpn.) and zero or more
// connector sub-zones (<connector>.<tenant>.helix.vpn.). Each zone
// carries SOA/NS apex records + AAAA/PTR/SRV records for its hosts.
type DNSZone struct {
    ID          uuid.UUID    // zone ID (PK in dns_zones table, or synthetic from connectorID)
    TenantID    uuid.UUID    // owning tenant
    Name        string       // zone apex FQDN, e.g. "warehouse.acme-corp.helix.vpn."
    ConnectorID *uuid.UUID   // nil for the tenant root zone; non-nil for connector sub-zones
    SiteID      uint16       // connector site ID (0 for tenant root zone)
    TenantPrefix netip.Prefix // the tenant's ULA /48 (needed for 4via6 encoding)

    // Hosts maps hostname (label only, e.g. "cam1") to its LAN IPv4.
    // The FQDN is <hostname>.<zone.Name>.
    Hosts map[string]HostEntry

    // Services maps service name (e.g. "_http._tcp") to SRV targets.
    Services map[string][]SRVTarget

    // Apex records (SOA + NS). Populated at zone creation, updated on
    // serial bump.
    SOA     DNSRecord
    NS      []DNSRecord

    // Serial is the SOA serial number. Bumped by the zone mutator on
    // every mutation event (host add/remove, service change). Format:
    // YYYYMMDDNN.
    Serial uint32

    // HasOverlappingPrefixes is true when >1 connector in the tenant
    // advertises the same IPv4 LAN prefix. When true, A-record
    // synthesis is suppressed (see DESIGN §7.2).
    HasOverlappingPrefixes bool

    // NotifyPeers is the list of secondary NS addresses to NOTIFY
    // (RFC 1996) on zone mutation. Empty in Phase 1 (no secondaries).
    NotifyPeers []netip.AddrPort

    mu         sync.RWMutex // guards Hosts, Services, Serial
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// HostEntry is a single LAN host registered in a connector's DNS zone.
type HostEntry struct {
    Name      string       // hostname label, e.g. "cam1"
    IPv4      netip.Addr   // LAN IPv4 address, e.g. 192.168.1.20
    MAC       string       // MAC address (from ARP table), e.g. "aa:bb:cc:dd:ee:ff"
    DiscoveredAt time.Time // when the connector first saw this host
    LastSeen  time.Time    // last ARP/NDP refresh
}

// SRVTarget is a single SRV record target.
type SRVTarget struct {
    Priority uint16
    Weight   uint16
    Port     uint16
    Target   string // FQDN of the target host (must have an AAAA in the same zone)
}

// DNSRecord is a single resource record ready for wire encoding.
// Values are stored as strings (presentation format); the server
// parses them into dns.RR at query time.
type DNSRecord struct {
    Name  string       // FQDN, e.g. "cam1.warehouse.acme-corp.helix.vpn."
    Type  uint16       // dns.TypeAAAA, dns.TypePTR, dns.TypeSRV, etc.
    Class uint16       // dns.ClassINET
    TTL   uint32       // seconds
    RData string       // presentation-format rdata, e.g. "fd12:3456:789a:4636:0:7:c0a8:14"
}

// Common TTL constants.
const (
    DefaultTTL     = 300   // 5 minutes -- standard host record TTL
    SOATTl         = 3600  // 1 hour -- SOA/NS records change rarely
    NegativeTTL    = 300   // 5 minutes -- SOA minimum for negative caching
)
```

### 8.2 Server type (`helix-go/internal/dns/server.go`)

```go
// DNSServer is the authoritative DNS server for the overlay.
//
// It listens on the gateway's overlay address (<ula48>::1:53) and serves
// authoritative responses for the tenant zone and its connector sub-zones.
// Non-overlay queries are forwarded to the configured upstream resolver
// (the gateway itself acts as a recursive resolver for internet names).
type DNSServer struct {
    Addr     netip.AddrPort // e.g. [fd12:3456:789a::1]:53

    // ZoneStore is the in-memory zone registry. The server reads
    // zones for query resolution; the zone mutator (in the coordinator)
    // writes zones in response to control-plane events.
    ZoneStore *ZoneStore

    // Upstream is the recursive resolver for non-overlay queries.
    // On the gateway, this is typically a local recursive resolver
    // (systemd-resolved or unbound). If empty, non-overlay queries
    // get REFUSED.
    Upstream string // e.g. "127.0.0.53:53" (systemd-resolved stub)

    // Transport is the miekg/dns server handle. Created in NewServer,
    // started in ListenAndServe.
    transport *dns.Server
}

// NewServer creates a new DNSServer bound to addr.
func NewServer(addr netip.AddrPort, store *ZoneStore, upstream string) *DNSServer

// ListenAndServe starts the DNS server. Blocks until the context is
// cancelled or the server encounters a fatal error.
func (s *DNSServer) ListenAndServe(ctx context.Context) error

// Shutdown gracefully stops the DNS server.
func (s *DNSServer) Shutdown() error

// ServeDNS implements the dns.Handler interface. It is the per-query
// entry point called by the miekg/dns server goroutine pool.
func (s *DNSServer) ServeDNS(w dns.ResponseWriter, r *dns.Msg)
```

### 8.3 Zone store (`helix-go/internal/dns/zonestore.go`)

```go
// ZoneStore is a thread-safe in-memory registry of DNS zones.
//
// Hydrated from Postgres at gateway startup; kept in sync by the zone
// mutator consuming the control-plane event stream. Lookups are O(1)
// via map keyed on zone name.
type ZoneStore struct {
    mu    sync.RWMutex
    zones map[string]*DNSZone // keyed by zone name (FQDN, e.g. "acme-corp.helix.vpn.")
    byID  map[uuid.UUID]string // zone ID -> zone name (reverse index)
}

// NewZoneStore creates an empty zone store.
func NewZoneStore() *ZoneStore

// Hydrate loads all zones for a tenant from Postgres into the store.
// Called once at gateway startup.
func (s *ZoneStore) Hydrate(ctx context.Context, tenantID uuid.UUID) error

// Zone returns the zone for the given FQDN, or nil if not found.
// Performs longest-suffix match so a query for
// "cam1.warehouse.acme-corp.helix.vpn." matches the
// "warehouse.acme-corp.helix.vpn." sub-zone first, then falls
// back to "acme-corp.helix.vpn.".
func (s *ZoneStore) Zone(name string) *DNSZone

// UpsertZone inserts or replaces a zone. Called by the zone mutator.
func (s *ZoneStore) UpsertZone(zone *DNSZone)

// DeleteZone removes a zone by connector ID. Called by the zone
// mutator on device.revoked.
func (s *ZoneStore) DeleteZone(connectorID uuid.UUID)

// DeleteTenantZones removes all zones for a tenant. Called by the
// zone mutator on tenant.deleted.
func (s *ZoneStore) DeleteTenantZones(tenantID uuid.UUID)

// LookupHost resolves a hostname to its zone and host entry.
// Returns (zone, host, found). The caller uses zone.SiteID and
// host.IPv4 to synthesize the 4via6 AAAA record.
func (s *ZoneStore) LookupHost(fqdn string) (*DNSZone, *HostEntry, bool)
```

### 8.4 Zone mutator (`helix-go/internal/coordinator/dns_mutator.go`)

```go
// DNSMutator subscribes to the control-plane event stream and mutates
// the DNS zone store in response to topology changes.
//
// It is a coordinator-side component that bridges the event stream
// (device.enrolled, connector.attached, etc.) to the DNS zone store.
// The mutated zone store is pushed to the gateway's DNS server via
// the WatchNetworkMap stream (the DNS field carries the server address;
// the zone data itself lives at the gateway, hydrated from Postgres
// and kept in sync via the event stream).
type DNSMutator struct {
    store  *ZoneStore
    events <-chan topology.Event
    log    *slog.Logger
}

// NewDNSMutator creates a DNSMutator subscribed to the given event channel.
func NewDNSMutator(store *ZoneStore, events <-chan topology.Event) *DNSMutator

// Run starts the mutator loop. Blocks until the context is cancelled
// or the event channel is closed.
func (m *DNSMutator) Run(ctx context.Context) error

// handleEvent dispatches a single topology event to the appropriate
// zone mutation method.
func (m *DNSMutator) handleEvent(ev topology.Event)

// createConnectorZone creates a new DNS sub-zone for a connector.
func (m *DNSMutator) createConnectorZone(ev topology.Event)

// updateConnectorPrefixes updates AAAA records when a connector's
// advertised prefixes or LAN hosts change.
func (m *DNSMutator) updateConnectorPrefixes(ev topology.Event)

// deleteConnectorZone removes a connector's DNS sub-zone.
func (m *DNSMutator) deleteConnectorZone(ev topology.Event)
```

### 8.5 NetworkMap DNS extension

The existing `NetworkMap.DNS` field ([buildmap §2.2]) is a `[]string`. This
design extends it to a structured type:

```go
// DNSConfig is the DNS configuration delivered to a client in its
// NetworkMap. It replaces the current `DNS []string` field.
type DNSConfig struct {
    // Servers is the list of overlay DNS server addresses.
    // Currently always one entry: the gateway's overlay address.
    // Future: may include secondary DNS servers for HA.
    Servers []string `json:"servers"`

    // Suffixes is the list of DNS search suffixes for the overlay
    // domain. The client agent configures the OS resolver to route
    // queries for these suffixes to the overlay DNS servers.
    Suffixes []string `json:"suffixes"`

    // SynthesizeA indicates whether the DNS server synthesizes
    // A records for overlay hostnames. False when the tenant has
    // overlapping IPv4 LAN prefixes (see DESIGN §7.2).
    SynthesizeA bool `json:"synthesize_a"`
}
```

The `buildMap` function populates this from the zone store:

```go
func buildDNSConfig(store *ZoneStore, tenantID uuid.UUID, gatewayOverlayIP string) DNSConfig {
    tenantZone := store.Zone(tenantZoneName(tenantID))
    return DNSConfig{
        Servers:     []string{gatewayOverlayIP},
        Suffixes:    []string{tenantZone.Name}, // e.g. "acme-corp.helix.vpn."
        SynthesizeA: !tenantZone.HasOverlappingPrefixes,
    }
}
```

---

## 9. Query Resolution Walkthrough

### 9.1 AAAA query

```
Client queries: cam1.warehouse.acme-corp.helix.vpn. IN AAAA

Gateway DNS server (helix-dns):
  1. Parse question: qname="cam1.warehouse.acme-corp.helix.vpn.", qtype=AAAA
  2. ZoneStore.LookupHost("cam1.warehouse.acme-corp.helix.vpn.")
     -> zone=DNSZone{Name:"warehouse.acme-corp.helix.vpn.", SiteID:7, ...}
     -> host=HostEntry{Name:"cam1", IPv4:192.168.1.20}
  3. SynthesizeAAAA(zone, host):
     via6 = ipam.Encode4via6(zone.TenantPrefix, zone.SiteID, host.IPv4)
     -> fd12:3456:789a:4636:0:7:c0a8:14
  4. Build DNS response:
     - Header: QR=1 (response), RA=0 (not recursive for overlay zone), RCODE=NOERROR
     - Question: cam1.warehouse.acme-corp.helix.vpn. IN AAAA
     - Answer: cam1.warehouse.acme-corp.helix.vpn. 300 IN AAAA fd12:3456:789a:4636:0:7:c0a8:14
     - Authority: warehoues.acme-corp.helix.vpn. 3600 IN NS ns1.helix.vpn.
  5. WriteMsg(response)
```

### 9.2 PTR query

```
Client queries: 4.1.0.0.8.a.0.c.[...nibbles...].ip6.arpa. IN PTR

Gateway DNS server:
  1. Parse qname, extract IPv6 from ip6.arpa nibble string
     -> fd12:3456:789a:4636:0:7:c0a8:14
  2. ipam.Decode4via6(addr)
     -> class=Via6, siteID=7, ipv4=192.168.1.20
  3. ZoneStore.ZoneBySiteID(7)
     -> zone=DNSZone{Name:"warehouse.acme-corp.helix.vpn."}
  4. zone.LookupHostByIPv4(192.168.1.20)
     -> host=HostEntry{Name:"cam1"}
  5. FQDN = "cam1.warehouse.acme-corp.helix.vpn."
  6. Build PTR response
```

### 9.3 NXDOMAIN (host not found)

```
Client queries: ghost.warehouse.acme-corp.helix.vpn. IN AAAA

Gateway DNS server:
  1. ZoneStore.LookupHost("ghost.warehouse.acme-corp.helix.vpn.")
  2. Zone exists (warehouse.acme-corp.helix.vpn.) but host "ghost" not in Hosts
  3. Return NXDOMAIN with SOA in authority section for negative caching
```

### 9.4 Non-overlay query (forwarding)

```
Client queries: google.com IN A

Gateway DNS server:
  1. ZoneStore.Zone("google.com") -> nil (not an overlay zone)
  2. Forward to upstream resolver (127.0.0.53:53)
  3. Return upstream's response verbatim
```

---

## 10. Performance

### 10.1 Query path cost

| Operation | Cost | Notes |
|---|---|---|
| `ZoneStore.LookupHost` | O(L) where L = number of labels (~3) | Longest-suffix match iterates labels |
| Map lookup in zone.Hosts | O(1) | `map[string]HostEntry` |
| 4via6 encode (AAAA synthesis) | O(1) | Byte copies, no allocation after warmup |
| 4via6 decode + arpa expand (PTR) | O(1) | Nibble expansion is 32 iterations of a 4-bit mask |
| DNS wire marshal (miekg/dns) | O(N) where N = response packet size | Typically < 512 bytes, well under 1 microsecond |

**Target:** < 100 microseconds per query from receive to send (single
goroutine, no blocking I/O inside the query path). The query path holds
the zone store's read lock only for the lookup duration; it does not
acquire the lock during wire marshal.

### 10.2 Concurrency

The `miekg/dns` server calls `ServeDNS` from a goroutine pool. The zone
store uses `sync.RWMutex` -- concurrent readers proceed in parallel.
Only the zone mutator (which runs in a single goroutine on the coordinator)
acquires the write lock, and only during mutation events (infrequent
relative to query volume).

### 10.3 Memory

Zone data is entirely in-memory at the gateway. For a tenant with 50
connectors and 500 LAN hosts per connector:

| Data structure | Size per entry | Total (25,000 hosts) |
|---|---|---|
| `HostEntry` | ~128 bytes (strings + netip.Addr + timestamps) | ~3.2 MB |
| `DNSZone` (50) | ~2 KB each (map overhead + SOA/NS records) | ~100 KB |
| `ZoneStore` maps | ~200 bytes per zone | ~10 KB |
| **Total** | | **~3.3 MB** |

Well within the gateway's memory budget.

---

## 11. Edge Cases

| # | Edge Case | Handling |
|---|---|---|
| EC-1 | Connector enrolls with no LAN hosts discovered yet | Zone created with empty `Hosts` map; SOA + NS records present. Queries for any host return NXDOMAIN. |
| EC-2 | Two connectors have a host with the same name (e.g. both have "cam1") | Different sub-zones (`cam1.warehouse...` vs `cam1.office...`) -- no collision. |
| EC-3 | Hostname contains characters illegal in DNS labels | Sanitized at registration per RFC 1035 §2.3.1 (see §3.2). Original name preserved in `HostEntry` metadata; DNS label is the sanitized form. |
| EC-4 | Connector's site ID changes (re-enrollment after delete) | New site ID, new sub-zone name (the connector name stays the same), new AAAA records. Old sub-zone deleted. Clients with cached old records get NXDOMAIN after TTL. |
| EC-5 | Gateway DNS server is unreachable (gateway down) | Client shim returns SERVFAIL. Applications see resolution failure. No fallback -- the overlay is unavailable without the gateway. |
| EC-6 | Tenant has >1 connector with same IPv4 LAN | `HasOverlappingPrefixes` set to true. A-record synthesis suppressed. AAAA records still served. A queries return NXDOMAIN. |
| EC-7 | DNS query flood (DoS from a compromised client) | The `miekg/dns` server has a built-in rate limiter. Additional per-client rate limiting at the gateway edge (policy layer, out of scope for DNS design). |
| EC-8 | Zone store not yet hydrated at gateway startup | `ZoneStore.Hydrate` runs before `DNSServer.ListenAndServe`. If hydration fails, the server does not start -- gateway health check fails, coordinator does not route traffic to it. |
| EC-9 | Concurrent host discovery + prefix change for the same connector | Zone mutator serializes events per connector (single goroutine). The `ConnectorPrefixesChanged` event is a batch delta; hosts added and removed in the same event are applied atomically with one serial bump. |
| EC-10 | Hostname is an IPv4 address literal (e.g. "192.168.1.20") | Treated as a hostname label. The DNS label becomes `192-168-1-20` after sanitization. The real hostname (from mDNS/DHCP) is preferred; literal-IP labels are a last-resort fallback. |

---

## 12. File Layout

```
helix-go/internal/dns/
  doc.go             -- package doc + honest scope
  types.go           -- DNSZone, HostEntry, SRVTarget, DNSRecord, DNSConfig
  zonestore.go       -- ZoneStore (in-memory registry, hydrate, lookup, upsert, delete)
  zonestore_test.go  -- unit tests: lookup, longest-suffix match, concurrent read/write
  server.go          -- DNSServer (miekg/dns handler, ServeDNS, query routing)
  server_test.go     -- integration tests: real UDP DNS server, AAAA/PTR/SRV/NXDOMAIN
  shim.go            -- DNSShim (split-DNS forwarding proxy for client devices)
  shim_test.go       -- unit tests: overlay vs non-overlay routing

helix-go/internal/coordinator/
  dns_mutator.go     -- DNSMutator (event subscriber, zone mutation logic)
  dns_mutator_test.go -- unit tests: event->zone-state transition table
```

---

## 13. Integration

### 13.1 Startup sequence

```
Gateway startup:
  1. Postgres connection pool established
  2. ZoneStore.Hydrate(tenantID) -- loads all zones from dns_zones table
  3. DNSServer = NewServer(addr, store, upstream)
  4. DNSServer.ListenAndServe(ctx) -- blocks, serves DNS

Coordinator startup:
  1. DNSMutator = NewDNSMutator(store, eventCh)
  2. go DNSMutator.Run(ctx) -- consumes event stream, mutates zones
```

### 13.2 Event flow

```
Connector discovers LAN host (ARP)
  |
  --> connector reports host to control plane
        |
        --> topology event: connector.prefixes_changed { addedHosts: [cam1] }
              |
              --> DNSMutator.handleEvent()
                    zone.UpsertHost("cam1", 192.168.1.20)
                    zone.BumpSerial()
                    store.UpsertZone(zone)
              |
              --> (future) NOTIFY secondaries (no-op in Phase 1)

Client resolves hostname
  |
  --> helix-dns-shim intercepts query
        |
        --> forwards to gateway DNS at <ula48>::1:53
              |
              --> DNSServer.ServeDNS()
                    zone := store.Zone(qname)
                    host := zone.Hosts["cam1"]
                    aaaa := synthesizeAAAA(zone, host)
                    write response
```

### 13.3 NetworkMap delivery

The `DNSConfig` is embedded in the `NetworkMap` and pushed to every client
via `WatchNetworkMap`. When the `DNSConfig` changes (e.g. a connector is
added and the suffix list grows, or `SynthesizeA` toggles), the client
agent reconfigures the DNS shim:

```
WatchNetworkMap push:
  NetworkMap {
    ...
    "dns": {
      "servers": ["fd12:3456:789a::1"],
      "suffixes": ["acme-corp.helix.vpn."],
      "synthesize_a": true
    }
  }

Client agent (map.rs reconcile):
  if dns_config_changed:
    update_shim_config(new_dns)
    update_os_resolver(new_dns)
```

---

## 14. Frozen Contracts (Must Not Break)

| Contract | Where defined | What it means |
|---|---|---|
| Zone FQDN naming: `<host>.<connector>.<tenant>.helix.vpn.` | This document §3.2 | Changing the name structure breaks every hostname resolution |
| 4via6 AAAA derivation: `Encode4via6(tenantPrefix, siteID, ipv4)` | [ipam §3.4] + this document §4.1 | The AAAA value IS the 4via6 address; changing encoding breaks routing |
| PTR derivation: `Decode4via6(addr) -> (siteID, ipv4) -> reverse lookup` | This document §4.2 | PTR must match AAAA; divergence breaks `getnameinfo` / reverse proxies |
| `DNSConfig` JSON schema | This document §8.5 | The wire format between coordinator and edge; changing field names breaks every client |
| SOA serial bump on every mutation | This document §5.3 | Stale serial = stale zone = downstream caches never invalidate |
| `NetworkMap.DNS` field shape | [buildmap §2.2] + this document §8.5 | The DNS field carries structured config; changing it breaks `map.rs` deserialization |
| `miekg/dns` wire format | RFC 1035 / `github.com/miekg/dns` | The server speaks standard DNS on the wire; non-standard responses break every DNS client |

---

## 15. Sources Verified

| Source | URL / Reference | Date verified |
|---|---|---|
| RFC 1035 -- Domain Names -- Implementation and Specification | https://datatracker.ietf.org/doc/html/rfc1035 | 2026-07-08 |
| RFC 3596 -- DNS Extensions to Support IP Version 6 (AAAA) | https://datatracker.ietf.org/doc/html/rfc3596 | 2026-07-08 |
| RFC 2782 -- A DNS RR for Specifying the Location of Services (SRV) | https://datatracker.ietf.org/doc/html/rfc2782 | 2026-07-08 |
| RFC 1996 -- A Mechanism for Prompt Notification of Zone Changes (NOTIFY) | https://datatracker.ietf.org/doc/html/rfc1996 | 2026-07-08 |
| RFC 6147 -- DNS64: DNS Extensions for Network Address Translation from IPv6 Clients to IPv4 Servers | https://datatracker.ietf.org/doc/html/rfc6147 | 2026-07-08 |
| RFC 4193 -- Unique Local IPv6 Unicast Addresses | https://datatracker.ietf.org/doc/html/rfc4193 | 2026-07-08 |
| miekg/dns -- Go DNS library (authoritative server, wire marshal/unmarshal) | https://github.com/miekg/dns | 2026-07-08 |
| CoreDNS -- plugin-based DNS server in Go (architectural reference, NOT a dependency) | https://coredns.io/ | 2026-07-08 |
| Tailscale MagicDNS -- `<hostname>.<tailnet>.ts.net` resolution (architectural analogy) | https://tailscale.com/kb/1081/magicdns | 2026-07-08 |
| [ipam] HelixVPN 4via6 Overlay Addressing Design (DNS Integration §7) | `docs/design/ipam/DESIGN.md` (Rev 1, 2026-07-08) | 2026-07-08 |
| [buildmap] Per-Node NetworkMap Computation Design (NetworkMap.DNS field §2.2) | `docs/design/buildmap/DESIGN.md` (Rev 1, 2026-07-08) | 2026-07-08 |
| [topology] Coordinator Topology Graph (Node structures) | `submodules/helix_go/internal/topology/graph.go` (2026-07-08) | 2026-07-08 |
| [identity] Identity & Enrollment Design | `docs/design/identity/DESIGN.md` (Rev 1) | 2026-07-08 |

---

*End of HelixVPN Overlay DNS Management Service Design. For implementation, start with:
1. `internal/dns/types.go` -- Go types (DNSZone, HostEntry, SRVTarget, DNSRecord, DNSConfig)
2. `internal/dns/zonestore.go` -- in-memory zone registry with hydration, lookup, upsert, delete
3. `internal/dns/server.go` -- DNSServer wrapping miekg/dns for authoritative overlay DNS
4. `internal/dns/shim.go` -- DNSShim split-DNS forwarding proxy for client devices
5. `internal/coordinator/dns_mutator.go` -- DNSMutator subscribing to control-plane events
6. `internal/buildmap/dnsconfig.go` -- DNSConfig population in buildMap*
