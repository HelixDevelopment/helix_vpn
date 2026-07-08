# HelixVPN -- 4via6 Overlay Addressing Design

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Status:** active -- concrete design for IPv4 LAN collision resolution via 4via6 + ULA /48
**Authority:** This document is the binding design for the 4via6 overlay addressing scheme.
It synthesizes the IPAM service spec (`v03-control-plane/svc-ipam.md`), the routing-and-addressing
spec (`v02-data-plane/routing-and-addressing.md`), and external research on Tailscale 4via6 and
RFC 4193. Where this document disagrees with those specs, the specs win until this document is
amended and the specs updated.

---

## 1. Problem Statement

When multiple client sites connect to the same HelixVPN tenant, they may use **identical private
IPv4 ranges** -- e.g. Site A and Site B both use `192.168.1.0/24`. Without a collision-resolution
mechanism, a client cannot address hosts on both sites simultaneously: `192.168.1.5` is ambiguous.

HelixVPN solves this through **4via6**: embedding each site's IPv4 addresses inside unique IPv6
overlay addresses so that every host behind every connector has a **globally unique, routable
overlay address** regardless of its underlying private IPv4 assignment. The mechanism is
Tailscale-proven: "4via6" is "IPv4-via-IPv6" -- IPv4 addresses expressed within an IPv6 prefix.

### 1.1 The design goal

| Requirement | Mechanism |
|---|---|
| Each tenant gets a unique overlay address space | RFC 4193 ULA /48 per tenant, random 40-bit Global ID |
| Each connector site gets a unique namespace within its tenant | 16-bit site ID allocated monotonically per tenant |
| Every host behind a connector gets a unique overlay address | Embed the host's IPv4 into the low 32 bits of the site's /96 prefix |
| Colliding IPv4 LANs never collide in the overlay | Distinct site IDs => distinct overlay prefixes |
| IPv6-native overlay end-to-end | ACLs and routing target the v6 overlay, never raw IPv4 |
| Fallback for IPv4-only clients | CGNAT 100.64/10 1:1 NAT (documented, not Phase 1) |

### 1.2 Prior art

- **Tailscale 4via6:** Uses ULA prefix `fd7a:115c:a1e0::/48` with a fixed 64-bit extension
  (`fd7a:115c:a1e0:b1a`) for the 4via6 class, followed by `0:XXXX` (site-translator ID, 32 bits
  where only lower 16 are usable) and `YYYY:YYYY` (IPv4 as two hex words). Connects "hundreds/
  thousands of identical [overlapping-CIDR] networks." **Critical implementation detail:**
  Tailscale does NOT do true IP-level NAT64; for TCP, the subnet router terminates the TCP
  connection and opens a new connection to the destination, splicing them together. For UDP,
  datagrams are forwarded with address rewriting. ICMPv6 is not translated to ICMPv4 on Windows.
  HelixVPN's connector MUST evaluate whether true packet-level 4via6 translation or Tailscale-
  style TCP termination is the correct approach for its threat model and requirements.
- **RFC 4193 (ULA):** Defines the `fd00::/8` prefix with 40-bit pseudo-random Global ID
  (generated via SHA-1 over time-of-day + EUI-64 system identifier per RFC 4193 SS3.2.2) for
  locally unique but globally-unrouted address space.
- **RFC 6598 (CGNAT):** Defines `100.64.0.0/10` as Shared Address Space, used here as the
  documented fallback for pure-IPv4 environments.
- **WireGuard/Nebula/Classic VPNs:** Resolve overlapping RFC 1918 via SNAT/MASQUERADE on the
  VPN interface, translating local addresses into a unique, non-colliding overlay range. This
  breaks applications that embed IP addresses in payloads and requires per-host DNAT rules for
  inbound connections. Tailscale's 4via6 is the only production system that solves this at the
  addressing layer without NAT state tables.

---

## 2. ULA /48 Provisioning

### 2.1 Why ULA

ULA (Unique Local Address) is the correct choice for an overlay's address space because:

- **Probabilistically unique:** A random 40-bit Global ID gives collision probability
  ~9 x 10^-13 per tenant pair -- negligible.
- **Globally unrouted:** ULA addresses are never announced on the public Internet; they
  stay within the overlay.
- **Permanently assigned:** A tenant's ULA /48 is generated once at tenant creation and
  never changes.
- **Structured:** The 16-bit Subnet ID field naturally maps to connector site IDs.

### 2.2 Provisioning flow

```
Tenant create (admin/helixvpnctl)
  |
  --> ipam.ProvisionPool(tenantID)
        |
        --> genRandomGlobalID() -- 40-bit random, verify no collision with existing tenants
        --> ula_prefix = fd || globalID[40] || 0::/48
        --> INSERT INTO overlay_pools (tenant_id, ula_prefix, next_host=2, next_site=1)
```

The pool is provisioned once, idempotently (`ON CONFLICT (tenant_id) DO NOTHING`).

**ProvisionPool edge cases:**
- Random collision with existing tenant: regenerate (bounded 8-retry loop).
  After exhaustion, return `ErrULAPrefixCollision` (retryable).
- Double-call: returns existing pool unchanged (idempotent).

### 2.3 Pool state

```
overlay_pools row:
  tenant_id    : UUID (PK)
  ula_prefix   : fdXX:XXXX:XXXX::/48
  next_host    : bigint (starts at 2; ::0 is subnet anycast, ::1 reserved for gateway)
  next_site    : bigint (starts at 1; 0 is the tenant fabric /64)
```

---

## 3. 4via6 Encoding

### 3.1 Byte-level address format

A 4via6 overlay address maps to a 128-bit IPv6 address within the tenant's ULA /48:

```
 128-bit 4via6 address:

  0        8                       48        64        80        96               128
  +--------+-----------------------+---------+---------+---------+-----------------+
  | 0xfd   | 40-bit tenant Global ID| 0x4636  | 0x0000  | siteID  |   IPv4 (32 bit) |
  | ULA    |       (T)              | (Via6)  | (rsvd)  | (16 bit)|   big-endian    |
  +--------+-----------------------+---------+---------+---------+-----------------+
                                           +--------- Interface ID (64 bit) ---------+
```

**Field breakdown:**

| Bits | Field | Value | Notes |
|---|---|---|---|
| 0-7 | ULA prefix | `0xfd` | RFC 4193, L=1 (locally assigned) |
| 8-47 | Global ID | random 40-bit | Per-tenant, generated at ProvisionPool |
| 48-63 | Address Class | `0x4636` | Fixed constant ('F6' mnemonic) for 4via6 |
| 64-79 | Reserved | `0x0000` | Must be zero; non-zero rejected at ingest |
| 80-95 | Site ID | 1..65535 | Per-connector, allocated by AssignSite |
| 96-127 | IPv4 | host address | Big-endian; the host's IPv4 in the served LAN |

**Other address classes:**

| Class (bits 48-63) | Kind | Interface ID layout |
|---|---|---|
| `0x0000` | Node | 64-bit stable Node ID (for clients, connectors, gateway) |
| `0x4636` | Via6 | 0x0000 || siteID(16) || IPv4(32) |
| `0x0001` | Relay | Relay overlay endpoint (Phase 2) |

### 3.2 Route prefix synthesis

A connector advertising `192.168.1.0/24` with site ID `S` produces a 4via6 **route prefix**:

```
  via6_prefix = <ula48>:<S>:0:0::/96
  via6_route  = <ula48>:<S>:0:0:<ipv4_network>/<96 + N>
```

Where N is the advertised prefix length. Examples:

| Advertised LAN | Site ID | 4via6 Overlay Route |
|---|---|---|
| `192.168.1.0/24` | 7 | `fdXX:XXXX:XXXX:7::c0a8:100/120` |
| `10.0.0.0/8` | 7 | `fdXX:XXXX:XXXX:7::a00:0/104` |
| `172.16.0.1/32` (single host) | 7 | `fdXX:XXXX:XXXX:7::ac10:1/128` |
| `192.168.1.0/24` | 8 | `fdXX:XXXX:XXXX:8::c0a8:100/120`  _(different prefix!)_ |

**Collision firewalls:** The `OverlapAdvertise` validation reject ensures no two peers in the
same RouteMap advertise the same overlay prefix. Because two connectors with the same IPv4 LAN
get **different site IDs**, their 4via6 prefixes are always distinct -- the collision is
mechanically prevented.

### 3.3 Site-ID allocation

Site IDs are allocated by the control plane IPAM service (`AssignSite`):

```
Connector enroll (kind=connector)
  |
  --> ipam.AssignSite(tenantID, connectorID)
        |
        --> SELECT site_id FROM connector_sites WHERE connector_id = X  (idempotency probe)
        --> If existing: return existing site (OutcomeExisting)
        --> If new:
              SELECT ... FOR UPDATE on overlay_pools  (serializes allocation)
              claimed = BumpOverlayNextSite  (atomic, returns old next_site)
              INSERT INTO connector_sites (tenant_id, connector_id, site_id)
              Emit connector.attached event
              Return site (OutcomeCreated)
```

**Site-ID rules:**
- Range: `1..65535` (`0` reserved for the tenant fabric /64)
- Monotonic, never recycled (even after connector deletion)
- Unique per tenant (`UNIQUE(tenant_id, site_id)` constraint)
- Idempotent per connector (same connector re-attaches => same site ID)

### 3.4 4via6 encode/decode (reference implementation)

Rust (data plane, `helix-route/src/via6.rs`):

```rust
pub fn encode_host(t: TenantPrefix, site: SiteId, v4: Ipv4Addr) -> OverlayAddr {
    let mut o = [0u8; 16];
    o[..6].copy_from_slice(&t.bytes);
    o[6..8].copy_from_slice(&0x4636u16.to_be_bytes());  // AddrClass::Via6
    // o[8..10] = 0x0000 (reserved, already zero)
    o[10..12].copy_from_slice(&site.to_be_bytes());
    o[12..16].copy_from_slice(&v4.octets());
    OverlayAddr(Ipv6Addr::from(o))
}

pub fn encode_route(t: TenantPrefix, site: SiteId, v4: Ipv4Net) -> Result<Ipv6Net, RouteError> {
    let base = encode_host(t, site, v4.network()).0;
    let plen = 96u8 + v4.prefix_len();   // /24 -> /120, /32 -> /128
    Ipv6Net::new(base, plen).map_err(|_| RouteError::BadPrefixLen(plen))
}

pub fn decode(a: &OverlayAddr) -> Result<(SiteId, Ipv4Addr), RouteError> {
    if a.class()? != AddrClass::Via6 { return Err(RouteError::NotVia6); }
    let o = a.0.octets();
    if o[8] != 0 || o[9] != 0 { return Err(RouteError::ReservedNonZero); }
    let site = u16::from_be_bytes([o[10], o[11]]);
    let v4   = Ipv4Addr::new(o[12], o[13], o[14], o[15]);
    Ok((site, v4))
}
```

Go (control plane reference, `internal/ipam/via6.go`):

```go
func via6Prefix(ula48 netip.Prefix, site uint16) netip.Prefix {
    a := ula48.Addr().As16()
    binary.BigEndian.PutUint16(a[6:8], site)
    // bytes 8..15 already zero (the low half of /96)
    return netip.PrefixFrom(netip.AddrFrom16(a), 96)
}

// embedV4 is the reference algorithm for tests + Console preview.
// The data-path (per-packet) embedding lives in the Rust client core.
func embedV4(via6 netip.Prefix, v4 netip.Addr) netip.Addr {
    a := via6.Addr().As16()
    copy(a[12:16], v4.As4())
    return netip.AddrFrom16(a)
}
```

---

## 4. AllocOverlayIP Flow

The `AllocOverlayIP` RPC allocates overlay addresses for devices (clients and connectors
themselves) in the **site-0 tenant fabric** (`<ula48>::/64`). 4via6 addresses for LAN hosts
are derived on-the-fly, not allocated.

### 4.1 Enrollment allocation sequence

```
Client enrolls with enroll_token:
  1. Resolve (tenant, wg_pubkey) -> existing device_id or create new
  2. AllocOverlayIP(tenantID, deviceID)
     - Idempotency probe: SELECT overlay_ip FROM devices WHERE id = X
       If found: return existing address (next_host unchanged)
     - If new:
         SELECT ... FOR UPDATE overlay_pools
         claimed = BumpOverlayNextHost (atomic, returns claimed value)
         address = embedHost(ula_prefix, claimed)
         -- caller writes devices.overlay_ip in same tx
  3. If kind=connector:
     AssignSite(tenantID, connectorID) -> siteID
  4. Emit device.enrolled + connector.attached events
  5. Return EnrollResponse{ device_id, overlay_ip, cert, gateway }
```

### 4.2 Fabric addressing (Node class)

```
  Node address (class 0x0000):

  0        8                       48              64                              128
  +--------+-----------------------+---------------+--------------------------------+
  | 0xfd   | 40-bit tenant Global ID| 0x0000        |    64-bit Node ID              |
  +--------+-----------------------+---------------+--------------------------------+
                                     Node class      monotonically allocated
```

| Address | Who |
|---|---|
| `<ula48>::1` | Gateway (reserved) |
| `<ula48>::2` | First enrolled device |
| `<ula48>::3` | Second enrolled device |
| ... | ... |

### 4.3 Idempotency guarantees

| Operation | Key | Repeat-call result |
|---|---|---|
| `ProvisionPool` | `tenant_id` (PK) | existing pool, no change |
| `AllocOverlayIP` | `device_id` | existing address, `next_host` unchanged |
| `AssignSite` | `connector_id` (PK) | existing site, `next_site` unchanged |

All three operations run in the same Postgres `WithTenant` transaction as the device insert,
so a rollback rolls back all counter bumps (no leaked addresses).

---

## 5. Route Advertisement

### 5.1 How routes propagate

```
Connector attaches + advertises CIDRs
  |
  --> ipam.AssignSite (allocates siteID)
  --> registry.AdvertisePrefixes (inserts into advertised_prefixes table)
  --> ipam.Via6RoutesFor(tenantID, connectorID)
        returns: Via6Route{ ipv4_cidr: "192.168.1.0/24",
                             via6_prefix: "fd..:4636:0:7::/96",
                             site_id: 7,
                             connector: uuid }
  |
  --> coordinator compiles RouteMap
        Peer.allowed_ips = [ "fd..::2/128", "fd..:4636:0:7::/96" ]
        Peer.via6       = [ { ipv4_cidr: "192.168.1.0/24", via6_prefix: "fd..:4636:0:7::/96" } ]
  |
  --> WatchNetworkMap stream pushes to all authorized nodes
  |
  --> Client helix-route reconciler:
        FIB builds: fd..:4636:0:7::/120 -> NextHop{ wg_pubkey=connectorA, via_site=7 }
        WG AllowedIPs includes fd..:4636:0:7::/96
```

### 5.2 Networking map protobuf

```protobuf
message Peer {
  string             device_id   = 1;
  bytes              wg_pubkey   = 2;
  repeated string    allowed_ips = 3;   // compiled AllowedIPs (includes via6 /96)
  bool               is_connector = 5;
  repeated Via6Route via6        = 6;   // present only for connector peers
}

message Via6Route {
  string ipv4_cidr   = 1;   // "192.168.1.0/24"
  string via6_prefix = 2;   // "fd7a:1122:3340:1::/96"
}
```

### 5.3 Need-to-know routing (policy-filtered)

A `RouteMap` delivered to a node is **already policy-filtered**: a client that is not
authorized to reach a connector's LAN never receives that connector's Peer entry at all.
Even if it did, `AllowedIPs` would exclude that prefix, and WG would drop the packet.

### 5.4 Convergence SLO

From ipam mutation to agent receiving the delta: **p99 < 1 second**.

| Step | Budget |
|---|---|
| DB commit + Redis XADD | < 50 ms |
| Bus delivery | < 50 ms |
| Coordinator diff + push | < 100 ms |
| **End-to-end p99 target** | **< 1 second** |

---

## 6. IPv4-to-IPv6 Translation (at the Edge)

### 6.1 Client-side: DNS-to-4via6 mapping

When an application on a client resolves `host-alice.warehouse.local` and gets back
`192.168.1.5`, the **4via6 resolver shim** intercepts and rewrites the response to the
4via6 address. The DNS server/service provides the mapping context.

Two paths for DNS resolution (design decision -- see section 7):

1. **Overlay DNS server at the gateway** -- the client queries a DNS server at
   `<ula48>::1` (the gateway); the DNS server has a zone-per-connector mapping LAN
   hostnames to their 4via6 addresses.

2. **Split-DNS via the 4via6 resolver shim** -- the client's local DNS resolver
   intercepts queries for `.warehouse.local` (or equivalent), queries the control
   plane for the connector's DNS zone, and synthesizes 4via6 AAAA records.

### 6.2 Connector-side: decode + DNAT

When a connector receives a packet destined for a 4via6 address:

```
Packet arrives at connector (from WG tunnel):
  dst = fd..:4636:0:7:c0a8:10a    (4via6 for 192.168.1.10, site 7)

  1. decode(dst) -> (site=7, 192.168.1.10)
  2. Confirm site 7 == this connector's own site ID
  3. DNAT: rewrite dst to 192.168.1.10, forward into served LAN
  4. Reply comes back from 192.168.1.10 -> SNAT back to overlay
```

### 6.3 End-to-end packet flow

```
App on client                    Gateway edge                   Connector (site 7)
  |                                |                              |
  |-- DNS resolve "cam1.wh.local" --> ... returns 4via6          |
  |   fd..:4636:0:7:c0a8:14                                       |
  |                                                                 |
  |-- packet dst=fd..:4636:0:7:c0a8:14 -->                        |
  |   WG encrypt to connectorA's pubkey                             |
  |                                                                 |
  |           encrypted datagram -->                                |
  |                                verdict map: src,dst,proto = ?   |
  |                                ALLOW? --> forward               |
  |                                                                 |
  |                                             |--> decode: site=7, 192.168.1.20
  |                                             |--> DNAT into LAN
  |                                             |<-- reply from 192.168.1.20
  |                                             |--> SNAT back to overlay
  |           <-- encrypted reply              |
  |<-- decrypted, delivered to app              |
```

### 6.4 Overlapping CIDR handling (automatic, no operator action)

When two connectors serve the same IPv4 range:
- Connector A: `192.168.1.0/24`, site 7 -> `fd..:4636:0:7:c0a8:100/120`
- Connector B: `192.168.1.0/24`, site 8 -> `fd..:4636:0:8:c0a8:100/120`

These are **different overlay prefixes** -- no collision. The conflict is **detected**
(surfaced in Console UX as an informational event) but never blocks routing. 4via6
resolves it automatically.

---

## 7. DNS Integration

### 7.1 Design approach: Overlay-resident DNS

Phase 1 uses an **overlay DNS server** at the gateway to resolve LAN hostnames to 4via6
addresses. Each connector registers its served hosts with a naming convention.

**DNS zones per connector:**

| Connector | Site ID | DNS zone (example) | LAN CIDR |
|---|---|---|---|
| warehouse | 7 | `wh.internal.helix` or `warehouse.local` | 192.168.1.0/24 |
| office | 8 | `office.internal.helix` or `office.local` | 192.168.1.0/24 |

**Hostname-to-4via6 mapping (AAAA records):**

| Hostname | AAAA Record |
|---|---|
| `cam1.warehouse.local` | `fd..:4636:0:7:c0a8:14` |
| `printer.warehouse.local` | `fd..:4636:0:7:c0a8:32` |
| `cam1.office.local` | `fd..:4636:0:8:c0a8:14` |

Note: `cam1.warehouse.local` and `cam1.office.local` both have the same IPv4 (`192.168.1.20`)
but resolve to **different** 4via6 addresses because they are in different sites.

### 7.2 DNS server placement

```
Gateway (overlay: <ula48>::1)
  |
  +-- helix-dns (internal service, listens on <ula48>::1:53)
        |
        +-- zone: <tenant>.internal.helix
              |
              +-- wh.<tenant>.internal.helix   -> AAAA records for site 7 hosts
              +-- office.<tenant>.internal.helix -> AAAA records for site 8 hosts
```

The `dns` field in the NetworkMap (`Vec<OverlayAddr>`) is populated with `<ula48>::1`
(the gateway's overlay DNS server address) and pushed to every client.

### 7.3 Dynamic host registration

When a LAN host behind a connector is discovered (ARP/NDP/mDNS), the connector
registers it with the control plane. The control plane's DNS service generates
AAAA records for the host's 4via6 address.

### 7.4 DNS resolution flow

```
Client App                     Client helix-dns-shim        Gateway helix-dns
  |                                |                            |
  |-- resolve "cam1.wh.local" -->  |                            |
  |                                |-- forward query -->        |
  |                                |   (over WG tunnel to ::1)  |
  |                                |                            |-- lookup: wh = site 7
  |                                |                            |-- cam1 = 192.168.1.20
  |                                |                            |-- encode: fd..:4636:0:7:c0a8:14
  |                                |                            |-- return AAAA
  |                                |<-- AAAA response           |
  |<-- 4via6 AAAA                 |                            |
  |-- connect to fd..:4636:0:7:c0a8:14 --> (normal overlay routing)
```

### 7.5 IPv4-only application compatibility (legacy resolver shim)

For applications that only understand IPv4, the **legacy resolver shim** presents a
synthesized IPv4 address from a reserved pool instead of the 4via6 address. The
connector's NAT then translates this synthesized IPv4 back to the 4via6 overlay address.

This is a Phase-2 optimization; Phase 1 assumes applications can consume IPv6 addresses
(which all modern operating systems and applications support).

---

## 8. Complete Example Walkthrough

### 8.1 Setup

**Tenant:** "acme-corp"
- ULA /48: `fd12:3456:789a::/48` (Global ID `0x123456789a`)

**Gateway:**
- Overlay address: `fd12:3456:789a::1` (site 0, host 1)

**Connector A: "warehouse"**
- LAN: `192.168.1.0/24`
- Enrolls as connector
- Assigned site ID: `1`
- Node overlay address: `fd12:3456:789a::2` (fabric, host 2)

**Connector B: "office"**
- LAN: `192.168.1.0/24` _(same IPv4 range!)_
- Enrolls as connector
- Assigned site ID: `2`
- Node overlay address: `fd12:3456:789a::3` (fabric, host 3)

**Client: "alice-laptop"**
- Enrolls as client
- Node overlay address: `fd12:3456:789a::4` (fabric, host 4)
- Policy grants: warehouse-cameras

### 8.2 4via6 address assignment

| LAN Host | IPv4 | Site | 4via6 Overlay Address |
|---|---|---|---|
| Warehouse cam1 | `192.168.1.20` | 1 | `fd12:3456:789a:4636:0:1:c0a8:14` |
| Warehouse printer | `192.168.1.50` | 1 | `fd12:3456:789a:4636:0:1:c0a8:32` |
| Office cam1 | `192.168.1.20` | 2 | `fd12:3456:789a:4636:0:2:c0a8:14` |
| Office laptop | `192.168.1.100` | 2 | `fd12:3456:789a:4636:0:2:c0a8:64` |

Key observation: Both cam1 hosts have the same IPv4 (`192.168.1.20`) but **different**
4via6 addresses because their site IDs differ. No collision.

### 8.3 Route map (delivered to alice-laptop)

```json
{
  "self": { "overlay_ip": "fd12:3456:789a::4/128" },
  "peers": [
    {
      "name": "warehouse-connector",
      "wg_pubkey": "<pubkey-A>",
      "allowed_ips": ["fd12:3456:789a::2/128", "fd12:3456:789a:1::/96"],
      "via6": [{ "ipv4_cidr": "192.168.1.0/24", "via6_prefix": "fd12:3456:789a:1::/96" }]
    }
  ],
  "dns": ["fd12:3456:789a::1"]
}
```

Note: Alice's laptop only receives the warehouse peer because she is only authorized for
warehouse-cameras. It never learns office exists.

### 8.4 Alice reaches warehouse cam1

```
1. Alice's app resolves "cam1.warehouse.local" -> AAAA fd12:3456:789a:4636:0:1:c0a8:14
2. App sends packet to fd12:3456:789a:4636:0:1:c0a8:14
3. Client FIB: LPM matches fd12:3456:789a:1::/96 -> NextHop(connectorA, via_site=1)
4. WG encrypts to connectorA's pubkey, sends via transport
5. Gateway edge: verdict map check -> ALLOW (policy: alice -> warehouse-cameras:any)
6. ConnectorA receives, decode() -> (site=1, 192.168.1.20)
7. Confirm site 1 == this connector's site
8. DNAT to 192.168.1.20, forward into LAN
9. Reply from 192.168.1.20 -> SNAT to overlay -> WG encrypt back to alice
```

### 8.5 Alice tries to reach office cam1 (denied)

```
1. Alice's app tries to resolve "cam1.office.local" -> NXDOMAIN (no DNS zone)
   OR: if the DNS zone exists but policy denies: the FIB has no entry for site 2's /96
2. Client FIB: LPM for fd12:3456:789a:2::/96 -> None
3. Default-deny: packet dropped before it leaves the client
4. No traffic leaks to office
```

---

## 9. Edge Cases

| # | Edge Case | Handling |
|---|---|---|
| EC-1 | Two clients enroll concurrently | `FOR UPDATE` on pool serializes; each gets a distinct `next_host` |
| EC-2 | Same device re-enrolls | Resolves to existing `device_id`; idempotent -- same overlay IP, counter unchanged |
| EC-3 | Device insert fails after host bump | Single tx rolls back the bump too; no leaked address |
| EC-4 | Three connectors advertise identical `/24` | Each gets a distinct site ID -> three distinct via6 prefixes; conflict detected but not blocking |
| EC-5 | Connector advertises IPv6 LAN | No 4via6 needed; shipped as plain `allowed_ips` CIDR |
| EC-6 | Connector deleted, re-created | Cascade frees `connector_sites` row; re-create allocates new (higher) `site_id` |
| EC-7 | Site space exhausted (>65535) | `ErrSiteSpaceExhausted`; OPERATOR-BLOCKED |
| EC-8 | ULA random collision across tenants | Regenerate (bounded retry); probability ~9e-13 per pair |
| EC-9 | 4via6 reserved bytes non-zero | `ReservedNonZero` reject at ingest; fail-closed |
| EC-10 | Connector roam (underlay IP change) | Overlay address unchanged; only `endpoint_candidates` update; FIB/policy untouched |

---

## 10. CGNAT Fallback (Documented, Not Phase 1)

For environments where 4via6 is impossible (IPv4-only client OS, no IPv6 stack on the
TUN interface), a documented CGNAT fallback using `100.64.0.0/10` (RFC 6598) provides
1:1 NAT per site.

**Address carve-out:**

```
100.64.0.0/10 partitioned into 256 tenant /18s:
  tenant_slot 0  -> 100.64.0.0/18
  tenant_slot 1  -> 100.64.64.0/18
  ...
  tenant_slot 255 -> 100.64.192.0/18

Each tenant /18 further partitioned into 64 site /24s:
  site 0  -> tenant_slot.0/24
  site 1  -> tenant_slot.1/24
  ...
  site 63 -> tenant_slot.63/24
```

**Trade-off vs 4via6:**
- CGNAT: 64 sites per tenant max (vs 4via6's 65,535)
- CGNAT: 256 tenants per gateway (shared `100.64.0.0/10` space)
- CGNAT: Remains IPv4-only; no v6 requirement for clients
- Both: Collision-free by construction; site ID disambiguates

Activation criterion per `decision-register.md`:
"4via6 client/OS support proves inadequate on a target platform -> fall back to
documented per-network NAT."

---

## 11. Frozen Contracts (Must Not Break)

| Contract | Where defined | What it means |
|---|---|---|
| `OverlayAddr` / `TenantPrefix` / `AddrClass` | `helix-route/src/addr.rs` | These are byte-level contracts; changing the class constant or the field layout breaks decode on every existing peer |
| 4via6 byte layout (§3.1) | `helix-route/src/via6.rs` + this document | byte-frozen; `0x4636` is a compile-time constant |
| `RouteMap` / `PeerRoute` | `helix-route/src/map.rs` | The protobuf `WatchNetworkMap` materializes this shape; breaking it breaks the data-plane protocol |
| `CompiledPolicy` / `VerdictRule` | `helix-route/src/policy.rs` | Policy compiler emits this; R4 mandates v6-only overlay targets |
| `Via6Route` protobuf message | `helix.coordinator.v1.Via6Route` | Wire-format: `ipv4_cidr` + `via6_prefix` pair |
| Postgres schema | `overlay_pools` + `connector_sites` tables | Durable state for address and site allocation; DDL under goose migration control |

---

## 12. Sources Verified

| Source | URL / Reference | Date verified |
|---|---|---|
| RFC 4193 -- Unique Local IPv6 Unicast Addresses | https://datatracker.ietf.org/doc/html/rfc4193 | 2026-07-08 |
| RFC 6598 -- IANA-Reserved IPv4 Prefix for Shared Address Space | https://datatracker.ietf.org/doc/html/rfc6598 | 2026-07-08 |
| Tailscale -- How NAT traversal works (4via6 addressing) | https://tailscale.com/blog/how-nat-traversal-works | 2026-07-08 |
| HelixVPN IPAM Service Spec (svc-ipam.md) | docs/research/mvp/final/v03-control-plane/svc-ipam.md (Rev 1, 2026-06-25) | 2026-07-08 |
| HelixVPN Overlay Routing & Addressing Spec (routing-and-addressing.md) | docs/research/mvp/final/v02-data-plane/routing-and-addressing.md (Rev 2, 2026-07-04) | 2026-07-08 |
| HelixVPN Architecture Refined (`04_VPN_CLD/HelixVPN-Architecture-Refined.md`) | decision D4: ULA /48 + 4via6 Camp A | 2026-07-08 |
| WireGuard whitepaper | https://www.wireguard.com/papers/wireguard.pdf | 2026-07-08 |

---

*End of HelixVPN 4via6 Overlay Addressing Design. For implementation, start with:
1. `internal/ipam` (Go -- tenant pool provisioning, site allocation, via6 route derivation)
2. `helix-route` (Rust -- via6 encode/decode, FIB compilation, policy compilation)
3. `helix-dns` (Go -- overlay DNS server, hostname-to-4via6 resolution)*
