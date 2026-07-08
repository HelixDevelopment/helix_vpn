# Helix VPN -- PKI Identity Model & Enroll RPC Design

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Status:** active
**Scope:** HVPN-P1-030 (PKI identity model) + HVPN-P1-031 (Enroll RPC)
**Authority:** This document is a synthesizing **design** document. It references the authoritative nano-detail
specifications produced during the MVP research phase (under `docs/research/mvp/final/`) for DDL, state machines,
algorithms, threat models, and test matrices. Where this document and a nano-detail spec conflict on a
cryptographic or DDL detail, the nano-detail spec wins. This document adds: consolidated protobuf definitions,
fresh 2025/2026 web research on DPoP and Tailscale patterns, concrete SLO targets, and the full Enroll RPC
request/response contract.

---

## Table of contents

- [1. Identity model overview](#1-identity-model-overview)
- [2. Managed mode: OIDC device authorization (RFC 8628 + DPoP)](#2-managed-mode-oidc-device-authorization-rfc-8628--dpop)
- [3. Anonymous mode: device enroll tokens](#3-anonymous-mode-device-enroll-tokens)
- [4. Enroll RPC: wire contract (P1-031)](#4-enroll-rpc-wire-contract-p1-031)
- [5. Certificate management (building on P1-032)](#5-certificate-management-building-on-p1-032)
- [6. Security considerations](#6-security-considerations)
- [7. SLOs & performance targets](#7-slos--performance-targets)
- [8. Phase-2 forward seams](#8-phase-2-forward-seams)
- [Sources verified](#sources-verified)

---

## 1. Identity model overview

HelixVPN ships **two identity postures** that bind to the **same** `devices` table and the **same** x509
certificate chain. The posture decides *how a human (or none) is associated with a device*; it does **not**
change the cryptographic device-enrollment spine (the key-never-leaves invariant, S2).

```
                    +---------------------------+
                    |   Managed posture (orgs)   |
                    |  OIDC IdP (Keycloak /     |
                    |  Authentik / Auth0 /       |
                    |  Entra / Okta / Zitadel)   |
                    |  Authorization Code+PKCE   |
                    |  + Device Grant (RFC 8628) |
                    |  users row: kind=oidc      |
                    +------------+--------------+
                                 |
                    +------------+--------------+
                    |      Shared device spine   |
                    |  +-- devices table         |
                    |  +-- WG public key (32B)   |
                    |  +-- mTLS device leaf cert |
                    |      (Ed25519, <= 24h)     |
                    |  +-- IPAM allocation       |
                    +------------+--------------+
                                 |
                    +------------+--------------+
                    |  Anonymous posture (end    |
                    |  users, Mullvad-like)      |
                    |  enroll tokens / account # |
                    |  NO email, NO SSO, NO PII  |
                    |  users row: kind=anonymous  |
                    |  email=NULL, oidc_sub=NULL |
                    +---------------------------+
```

### 1.1 Key invariants (non-negotiable)

| # | Invariant | Source |
|---|---|---|
| S2 | **Device private keys never leave the device.** Control plane only ever sees the 32-byte Curve25519 *public* key. | [04-security-privacy-pki.md SS0.1] |
| S4 | **Short-lived control-channel mTLS.** Every agent authenticates with a <= 24 h device cert. | [04-security-privacy-pki.md SS0.1] |
| S5 | **Revocation latency == convergence SLO** (p99 < 1 s, a TARGET). | [04-security-privacy-pki.md SS0.1] |
| S11 | **The CA root + Postgres are the only secrets to protect.** | [04-security-privacy-pki.md SS0.1] |
| -- | **Anonymous-by-default for end users** -- Mullvad "account number, no PII" stance. | [04_P1 S6.1] |

### 1.2 Two keypairs, two planes (deliberately uncoupled)

The device generates **two** keypairs at enroll, on **independent** key planes:

| Plane | Keypair | Purpose | Crosses the wire? |
|---|---|---|---|
| Control-identity | Ed25519 leaf (CSR) | Authenticate the mTLS control channel (`WatchNetworkMap`, `RenewCert`) | CSR (public) only |
| Data | X25519 WG static | Decrypt the tunnel (WG Noise IK) | 32-byte public key only |

Compromising the device leaf cert does **not** yield the WG private key, and stealing the WG key
does **not** authenticate the control RPC -- both must be valid for a device to *learn* its peers
(map) **and** *reach* them (tunnel). [04-security-privacy-pki.md SS1.2]

---

## 2. Managed mode: OIDC device authorization (RFC 8628 + DPoP)

### 2.1 Two OIDC flows for two surfaces

Managed tenants authenticate through **two** distinct OIDC flows, corresponding to the two
surfaces HelixVPN exposes:

| Surface | Flow | RFC | Auth material |
|---|---|---|---|
| Console / REST API (browser) | Authorization Code + PKCE | RFC 7636 | ID token from IdP JWKS; session cookie |
| CLI / device-first enrollment | Device Authorization Grant + DPoP | RFC 8628 + RFC 9449 | device_code + DPoP-bound access token |

The Console flow (Authorization Code + PKCE) is fully specified in
[identity-and-enrollment.md SS4]. This section focuses on the **device-first enrollment flow**
which is the new addition for P1-030/P1-031.

### 2.2 Device Authorization Grant flow (RFC 8628 with DPoP binding)

The Device Authorization Grant (RFC 8628) is the correct OAuth flow for input-constrained
devices (Linux servers, headless connectors, CLI clients) that have a browser available on a
*separate* device. The 2025 security landscape demands **DPoP (RFC 9449) binding** to prevent
device-code theft and replay attacks.

```
Device (CLI / headless)                    helixd (identity)                  Browser (separate device)
       |                                         |                                    |
       |-- POST /v1/oidc/{provider}/device ----->|                                    |
       |   client_id, scope, DPoP proof JWT      |                                    |
       |                                         |-- verify DPoP, mint device_code    |
       |<-- { device_code, user_code,            |   bind DPoP key to device_code     |
       |      verification_uri, interval,        |                                    |
       |      expires_in }                       |                                    |
       |                                         |                                    |
       |   Show: "Go to https://...              |                                    |
       |    and enter code: XXXX-XXXX"           |                                    |
       |                                         |                                    |
       |   [polling loop with DPoP proofs]       |            [user opens browser]    |
       |-- POST /v1/oauth/token (DPoP) -------->|       <----| visits verification_uri|
       |   grant_type=urn:ietf:params:oauth:     |            | enters user_code       |
       |   grant-type:device_code                |            | authenticates via IdP  |
       |<-- { authorization_pending }            |       ---->| consent granted        |
       |                                         |                                    |
       |   [... poll every 'interval' seconds]   |                                    |
       |                                         |                                    |
       |-- POST /v1/oauth/token (DPoP) -------->|                                    |
       |   grant_type=device_code                |-- verify DPoP proof matches bound  |
       |<-- { DPoP-bound access_token,           |   key; IdP confirmed authorization |
       |      refresh_token, expires_in }        |   JIT-provision users row           |
       |                                         |                                    |
       |   Token stored in OS keychain           |                                    |
       |   (Keychain / Keystore / DPAPI /        |                                    |
       |    libsecret) -- NEVER a plaintext file |                                    |
```

### 2.3 DPoP binding -- the 2025 security requirement

Per the IETF `draft-parecki-oauth-dpop-device-flow` (September 2025), DPoP binds the
device code to a cryptographic key at the start of the flow:

1. The device generates an Ed25519 keypair at the start of the flow.
2. Every request to `/device_authorization` and `/token` carries a DPoP proof JWT
   signed with the device's private key.
3. The authorization server binds the `device_code` to the DPoP public key at creation.
4. Resulting access and refresh tokens are **DPoP-bound** (sender-constrained,
   `"token_type": "DPoP"`), mitigating token leakage.

This closes the device-code theft vector: an attacker who obtains the `device_code` cannot
poll the token endpoint without the corresponding DPoP private key.

### 2.4 Claim -> role binding

Role is asserted from an IdP **group claim**, mapped per tenant, re-evaluated on every
login -- no stale privilege retention. The trust boundary is explicit: HelixVPN trusts
the IdP to assert group membership. A compromised IdP is out of scope.
[identity-and-enrollment.md SS4.2]

| Rule | Property |
|---|---|
| Role from group claim, mapped per tenant (`role_map`), never client-supplied | client cannot self-elevate |
| Role refreshed on every login | IdP-side group removal demotes on next sign-in |
| Removed from all mapped groups => falls to `default_role` | never silently retains privilege |
| `SuspendedAt` user => `ErrUserSuspended`, no session | suspension denies login immediately |

---

## 3. Anonymous mode: device enroll tokens

### 3.1 Two anonymous front doors (both no-PII)

| Front door | Use | Secret format | Stored as |
|---|---|---|---|
| **Device enroll token** | one device joins (paste/QR) | 256-bit base32-Crockford string (`het_...`) | `argon2id(token)` -- hash only |
| **Account number** | human-memorable login for privacy accounts (Mullvad-style) | 16-digit decimal, grouped `####-####-####-####` | `argon2id(number)` -- hash only |

The anonymous path is the **default for end users** -- it is the LINDDUN *Identifying* mitigation
LP-I-1: there is no human to which the tunnel can be tied.

### 3.2 Enroll token lifecycle

```
              +---------+
    Minted -->| PENDING |--(now > expires_at)--> EXPIRED
              +----+----+
                   |
          (Enroll RPC, conditional UPDATE wins)
                   |
              +----+----+
              | CONSUMED |  (single-use; atomic conditional consume)
              +---------+
                   |
          (operator revokes)
                   |
              +----+----+
              | REVOKED  |
              +---------+
```

The only legal terminal-by-success transition is `PENDING -> CONSUMED`. Every other terminal
state rejects redemption. A second `Enroll` with the same raw token updates ZERO rows
=> `ErrTokenUnusable`. Atomicity is a conditional `UPDATE ... WHERE consumed_at IS NULL`
inside the enroll transaction. [identity-and-enrollment.md SS6.2]

### 3.3 Token security properties

| Property | Mechanism |
|---|---|
| High entropy | 256-bit `randomBytes(32)`, `het_` prefix, base32-no-pad |
| Hash-at-rest only | `argon2id(raw)` stored; raw shown ONCE at mint |
| Short TTL | clamped `[5m, 24h]`, default 1h |
| Single / bounded use | `max_uses` (default 1); atomic conditional consume |
| Kind-bound | `bind_kind in {client, connector}` enforced at `Enroll` |
| Audited mint | `enroll_token.minted` event carries `{token_id, kind}` -- never the raw token |
| Constant-time check | candidate hashes compared in constant time; failure returns opaque `PermissionDenied` with no field-level oracle |

### 3.4 Account number entropy

16 decimal digits = approximately 53.1 bits of entropy. Online-guessing resistance comes from:
(a) storing only `argon2id(number)`, (b) rate-limiting per `(tenant, ua_hash)` bucket, and
(c) treating the number as a bearer credential the operator must store securely.

This is modeled on Mullvad's 16-digit account number system, which was proven by a 2023
Swedish police raid -- they raided Mullvad's office and found **zero customer data** because
none exists. [Mullvad VPN raid, The Verge 2023]

---

## 4. Enroll RPC: wire contract (P1-031)

### 4.1 gRPC service definition

The `Enroll` RPC is the **only** unauthenticated agent RPC on the control plane. It validates
a hashed single-use enroll token, allocates an overlay IP, issues a short-lived mTLS device
cert, and persists the device row and WG public key in one atomic database transaction.

```protobuf
// proto/helix/coordinator/v1/coordinator.proto
// Package: helix.coordinator.v1
// This is the authoritative Enroll definition -- extends the existing Coordinator service.

syntax = "proto3";
package helix.coordinator.v1;
option go_package = "github.com/helixdevelopment/helix-proto/gen/go/coordinator/v1;coordinatorv1";

import "buf/validate/validate.proto";
import "google/protobuf/timestamp.proto";

// ---------------------------------------------------------------------------
// Enroll -- the single unauthenticated agent RPC
// ---------------------------------------------------------------------------

service Coordinator {
  // ... existing RPCs (WatchNetworkMap, AdvertisePrefixes, ReportStatus, RenewCert, RotateWGKey) ...

  // Enroll a new device. This is the ONLY unauthenticated agent RPC.
  // Authenticated by a single-use (or bounded-use) hashed enroll token.
  // The device generates its WG + leaf keypairs LOCALLY; only public material crosses the wire (S2).
  rpc Enroll(EnrollRequest) returns (EnrollResponse);
}

message EnrollRequest {
  // Enroll token -- plaintext as provided by the admin out-of-band (paste/QR).
  // Verified against argon2id(token) stored in enroll_tokens.token_hash.
  // Single-use (or bounded-use); TTL'd; constant-time comparison.
  string enroll_token = 1 [(buf.validate.field).string.min_len = 1];

  // 32-byte Curve25519 PUBLIC key. The device generates the keypair locally;
  // the private key NEVER leaves the device (invariant S2 / K2).
  bytes wg_pubkey = 2 [(buf.validate.field).bytes.len = 32];

  // PKCS#10 CSR for the Ed25519 mTLS leaf key, generated ON the device.
  // Proof-of-possession verified before signing -- the leaf public key
  // in the CSR MUST match the signer (closes T-PKI-S-1).
  bytes csr = 3 [(buf.validate.field).bytes.min_len = 1];

  // Operating system of the enrolling device.
  string os = 4 [(buf.validate.field).string = {
    in: ["ios", "android", "linux", "windows", "macos", "harmonyos", "aurora"]
  }];

  // Human-readable device name (user-assigned or auto-generated).
  string name = 5 [(buf.validate.field).string.max_len = 128];

  // Device kind -- MUST match the token's bind_kind when set.
  DeviceKind kind = 6 [(buf.validate.field).enum.defined_only = true];

  // OPTIONAL platform key-attestation blob (Apple App Attest / Android Key
  // Attestation / Windows TPM). Empty in MVP; Phase-2 adds hardware binding.
  bytes attestation = 7;
}

message EnrollResponse {
  // Stable device identifier (UUID v7, time-ordered).
  string device_id = 1;

  // Allocated overlay IPv6 address, e.g. "fd7a:helix:1::2".
  string overlay_ip = 2;

  // Short-lived mTLS device leaf certificate (X.509 DER), signed by the
  // tenant issuing CA. not_after = now + cert_ttl_s (default 86400 = 24h).
  bytes device_cert = 3;

  // CA certificate chain for pinning -- [root_ca, issuing_ca].
  // The agent pins the root CA public key and verifies every future
  // leaf/issuing cert against it (closes control-plane MITM substitution).
  repeated bytes ca_chain = 4;

  // Gateway information for the initial connection.
  GatewayInfo gateway = 5;

  // Certificate TTL hint (seconds). The agent renews at
  // max(1h, 0.2 * cert_ttl_s) before expiry.
  uint32 cert_ttl_s = 6;

  // Timestamp when this enrollment was processed (server time).
  google.protobuf.Timestamp enrolled_at = 7;
}

// Device kind enum -- matches the DB device_kind type.
enum DeviceKind {
  DEVICE_KIND_UNSPECIFIED = 0;
  DEVICE_KIND_CLIENT      = 1;  // end-user VPN client
  DEVICE_KIND_CONNECTOR   = 2;  // network-side agent (advertises CIDRs)
}

// Gateway bootstrap info delivered at enrollment.
message GatewayInfo {
  // Public IPv4/IPv6 endpoint of the gateway.
  string endpoint = 1;

  // Gateway WireGuard public key (32 bytes).
  bytes wg_pubkey = 2;

  // Gateway overlay IPv6 address.
  string overlay_ip = 3;

  // Allowed IPs for the initial WG tunnel to the gateway.
  repeated string allowed_ips = 4;
}
```

### 4.2 Enroll flow (end-to-end)

```
Admin/Console                  identity (Go)       pki (Go)      ipam (Go)      Postgres      events bus
     |                              |                  |              |              |              |
     |-- POST /v1/enroll-tokens --->|                  |              |              |              |
     |   {kind, ttl, max_uses}      |                  |              |              |              |
     |                              |-- token=rand256()|              |              |              |
     |                              |-- store argon2id(token) ------->|              |              |
     |<-- {token, qr} (SHOWN ONCE)  |                  |              |              |              |
     |                              |                  |              |              |              |
     |   [out-of-band: paste/QR]    |                  |              |              |              |
     |                              |                  |              |              |              |
Device (helix-core)                 |                  |              |              |              |
     |                              |                  |              |              |              |
     |-- generate WG+leaf keypairs  |                  |              |              |              |
     |   LOCALLY (S2/K1)            |                  |              |              |              |
     |                              |                  |              |              |              |
     |-- Connect:Enroll ----------->|                  |              |              |              |
     |   {token, wg_pubkey, csr}    |                  |              |              |              |
     |                              |-- verify token   |              |              |              |
     |                              |   (const-time)   |              |              |              |
     |                              |-- ON VALID:      |              |              |              |
     |                              |   allocate IP ---|------------->|              |              |
     |                              |                  |              |-- allocate -->|              |
     |                              |                  |              |<-- overlay_ip|              |
     |                              |   SignDeviceCert |              |              |              |
     |                              |----------------->|              |              |              |
     |                              |                  |-- CSR PoP --->|              |              |
     |                              |                  |-- sign leaf-->|              |              |
     |                              |                  |<-- cert ------|              |              |
     |                              |   INSERT devices + certs + wg_keys + consume token (ONE TX)
     |                              |----------------------------------------------->|              |
     |                              |   emit device.enrolled -------------------------------------->|
     |   ON INVALID:                |                  |              |              |              |
     |<-- PermissionDenied          |                  |              |              |              |
     |   (no field oracle, SS6.2)   |                  |              |              |              |
     |                              |                  |              |              |              |
     |<-- EnrollResponse -----------|                  |              |              |              |
     |   {device_id, overlay_ip,    |                  |              |              |              |
     |    device_cert, ca_chain,    |                  |              |              |              |
     |    gateway, cert_ttl_s}      |                  |              |              |              |
     |                              |                  |              |              |              |
     |-- persist cert+keys in       |                  |              |              |              |
     |   OS keystore (S2/K4)        |                  |              |              |              |
     |                              |                  |              |              |              |
     |-- WatchNetworkMap (mTLS) --->|  (authenticated by the NEW cert)               |              |
```

### 4.3 Error paths

| Error | Connect/gRPC code | Trigger | Caller action |
|---|---|---|---|
| `ErrTokenUnusable` | `PermissionDenied` | Token consumed / expired / revoked / wrong tenant | Mint a fresh token |
| `ErrKindMismatch` | `PermissionDenied` | `DeviceKind` doesn't match `token.bind_kind` | Use the correct token type |
| `ErrCSRInvalid` | `InvalidArgument` | CSR fails proof-of-possession verification | Regenerate keypair + CSR |
| `ErrWGKeyDuplicate` | `AlreadyExists` | WG pubkey already bound (tenant unique constraint) | Regenerate WG keypair |
| `ErrIPAMExhausted` | `ResourceExhausted` | Tenant ULA /48 fully allocated | Operator expands or GCs |
| `ErrRateLimited` | `ResourceExhausted` | Per-IP / per-tenant rate limit hit | Retry with exponential backoff |
| `ErrNoActiveCA` | `FailedPrecondition` | Tenant has no active issuing CA | Provision CA (operator action) |

All error responses are **opaque** -- no field-level detail leaks (no enumeration oracle,
no timing side-channel). Constant-time token comparison ensures a hit and a miss cost the
same wall-clock time. [identity-and-enrollment.md SS6.2]

### 4.4 Anti-replay & abuse controls

| Control | Mechanism | Threat closed |
|---|---|---|
| Constant-time token check | Candidate `enroll_tokens` rows hash-compared in constant time | Token enumeration |
| CSR proof-of-possession | CSR signature verified before signing; leaf pubkey in CSR must match signer | T-PKI-S-1 (sign a key you don't hold) |
| Rate limiting | Per-source-IP and per-tenant Redis token buckets; exponential backoff | Enroll flood (T-PKI-D-1, T-CONN-D-1) |
| One pubkey, one device | `UNIQUE (tenant_id, wg_pubkey)` | WG-key reuse across devices |
| `kind` must match token | A `connector` token cannot enroll a `client` device | Privilege-shape confusion |
| All-or-nothing transaction | Token consume + device insert + IP alloc + cert issue + WG key register are ONE transaction | No partial enrollment, no orphan certs |

### 4.5 SLO: enroll -> first NetworkMap < 2 s

The target for P1-031 is **enroll call -> first `WatchNetworkMap` snapshot delivered < 2 seconds**,
measured as `helix_enroll_to_map_seconds` histogram. The budget breaks down as:

| Phase | Target |
|---|---|
| Token verify + IPAM allocate + cert issue + device insert (one tx) | < 150 ms |
| `device.enrolled` event publish -> coordinator consume | < 50 ms |
| Coordinator builds initial NetworkMap (single-device, single-gateway) | < 100 ms |
| `WatchNetworkMap` stream open + first snapshot push | < 100 ms |
| Network round-trip (agent -> gateway) + TLS handshake | < 500 ms |
| **Total target** | **< 2 s** |

This is a **design target**, not a measured result. Until the SS9 soak captures it, the figure is
**UNVERIFIED** per SS11.4.6.

---

## 5. Certificate management (building on P1-032)

### 5.1 x509 device cert structure

Every device leaf cert follows this profile, enforced in the signer (never taken from a request field):

| Field | Value |
|---|---|
| Key algorithm | Ed25519 (ECDSA-P256 fallback for platforms that reject Ed25519 client certs) |
| Validity | <= 24 h |
| `subject` | `CN=<device_id>, O=<tenant_id>` |
| `subjectAltName` | URI `spiffe://<tenant_id>/device/<device_id>` (structured binding, not IP/hostname) |
| `keyUsage` | `digitalSignature` |
| `extKeyUsage` | `clientAuth` (+ `serverAuth` for the gateway-facing edge cert) |
| `basicConstraints` | `CA:FALSE` |
| `serialNumber` | 20 random bytes (mTLS lookup key + revocation-set key) |
| Custom ext | `helix.deviceKind = client | connector` |

### 5.2 CA hierarchy

```
Tenant Root CA (Ed25519, ~10y, OFFLINE / KMS / HSM)
  |-- CA:TRUE, pathlen:1, keyUsage keyCertSign + cRLSign
  |
  +-- Issuing CA (Ed25519, ~1y, ONLINE signer in helixd/pki)
        |-- CA:TRUE, pathlen:0, signs every device leaf
        |
        +-- Device control-channel leaf (Ed25519, <= 24h)
        |     |-- CA:FALSE, EKU clientAuth + serverAuth
        |     |-- SAN URI spiffe://<tenant>/device/<id>
        |
        +-- Device attestation leaf (Ed25519, ~90d, Phase-2)
              |-- TPM / Secure-Enclave-bound when present
```

Each tenant gets its **own** isolated certificate hierarchy -- there is no shared global root.
Multi-tenant isolation reaches into the trust anchor, not just the row filter.
[pki-and-certs.md SS2], [svc-pki.md SS2]

### 5.3 Auto-renewal before expiry (P1-033)

The agent drives renewal over the **existing authenticated channel** at
`T - renewSkew` where `renewSkew = max(1h, 0.2 * TTL)` => for a 24 h TTL, renewal starts
~4.8 h before expiry. Key properties:

- Renewal is authenticated by the *current still-valid* leaf -- **never needs the enroll token again**
- Make-before-break: both old (flipped to `renewed`) and new leaves are valid during the overlap
- `AuthDevice` accepts `renewed` status for the remainder of its `not_after` -- no mid-stream cutoff
- WG key is unchanged (renewal != WG rotation)
- A missed renewal (agent offline past expiry) degrades to full re-enrollment with a fresh token -- **never a silent failure**

### 5.4 Revocation flow (P1-033)

Revocation is the security analogue of the convergence SLO: the same push-don't-poll machinery
that propagates a route change propagates a revocation.

```
Admin POST /v1/devices/{id}/revoke
  |
  identity/pki: ONE tx
  |-- devices.revoked_at = now
  |-- device_certs.status = 'revoked' (tooth a: stop renewing)
  |-- device_wg_keys.active = false (tooth b: instant data-path cutoff)
  |-- audit "device.revoke"
  |-- add serial to in-memory revoke cache (instant hot-path reject, no DB round-trip)
  |
  events bus: XADD device.revoked {device_id, wg_pubkey, serial}
  |
  coordinator (fan-out):
  |-- remove WG peer from every affected edge
  |-- push MapDelta{remove peer device_id} to every open WatchNetworkMap stream
  |-- force-close the revoked device's OWN stream
  |
  edge (kernel WG):
  |-- remove kernel WG peer => data path cut, ZERO restarts
```

Two cooperating teeth (belt-and-suspenders):
1. **Stop renewing** the short-lived mTLS cert => it self-expires within <= 24 h (defence-in-depth floor)
2. **Remove the WG public key** from every peer set => instant data-path cutoff

There is **no CRL/OCSP on the data path** -- short-lived certs + active push obviate online
revocation checking. [pki-and-certs.md SS7]

### 5.5 Issuing CA compromise -- fast path

Scheduled rotation (safe, 24 h overlap) and compromise rotation (fast, zero overlap) are
**two distinct paths**:

- **Scheduled:** root signs new issuing CA; both old (`retiring`) and new (`active`) are trusted
  for an overlap window = max device-cert TTL + renew-sweeper margin.
- **Compromise:** the compromised issuing CA is flipped directly to `status='compromised'` with
  **zero overlap** -- `AuthDevice` rejects any leaf whose `issuing_ca_id` resolves to a
  `compromised` row, regardless of `not_after`. Every affected device is force-revoked and
  re-enrolled. [svc-pki.md SS2.3]

### 5.6 SLO: revoke -> edge enforcement < 1 s (P1-033)

Targeted at **p99 < 1 s**, measured by `helix_pki_revoke_seconds` histogram. The budget:

| Phase | Mechanism | Target |
|---|---|---|
| DB tx (revoke cert + retire WG key + audit) | Single-row UPDATE, primary-key lookup | < 10 ms |
| Revoke-cache add | In-memory `map[serial]struct{}` insert | < 1 micros |
| Event publish | Redis `XADD events:devices` | < 5 ms |
| Coordinator consume + map recompute | In-memory node removal; minimal affected-set diff | < 50 ms |
| Push to affected `WatchNetworkMap` streams | gRPC `stream.Send(MapDelta)` per open stream | < 100 ms |
| Edge drops kernel WG peer | `ip link del dev wg0 peer <pubkey>` | < 10 ms |
| **Total target** | | **< 1 s p99** |

This is a design **target**, not a measured result. The sub-second race window is honest, not
zero -- residual R-RACE. The floor is fail-static (a stale grant never fails *open*), and
<= 24 h cert expiry is the hard ceiling even if a push is missed.

---

## 6. Security considerations

### 6.1 Private key never leaves the device (S2)

The WG private key and the leaf Ed25519 private key are generated **on the device** by a CSPRNG.
The wire contract names only `wg_pubkey` (32 bytes) and `csr` (PKCS#10, public material).
The control plane has no `*_private*` / `*_secret*` column for device keys. A schema-lint in CI
fails the build if such a column appears. A paired SS1.1 mutation test feeds a private-shaped
key into the enroll handler and asserts it MUST reject.

Platform keystore targets:
| Platform | Sealed by | Hardware-backed when present |
|---|---|---|
| iOS / macOS | Keychain | Secure Enclave |
| Android | Keystore | StrongBox / TEE |
| Windows | DPAPI | TPM |
| Linux | libsecret | TPM2 |

A software fallback is permitted **only with a logged warning**. [identity-and-enrollment.md SS2.3]

### 6.2 Token hashing (argon2id)

All enroll tokens and account numbers are stored as `argon2id(raw)` -- never plaintext, never a
fast hash. Argon2id parameters are calibrated to 30-150 ms per verification on the target
hardware, providing memory-hardness against offline brute-force. A DB read never yields a usable
token (AS-TOKEN; S11-class hygiene). Token lookup is constant-time: candidate hashes are compared
in constant time, and failure returns an opaque `PermissionDenied` with no field-level detail.

### 6.3 Rate limiting

| Surface | Mechanism |
|---|---|
| Enroll RPC | Per-source-IP and per-tenant Redis token buckets; exponential backoff on denial |
| Account login | Per `(tenant, ua_hash)` bucket; constant-time hit/miss |
| OIDC token endpoint | Standard OAuth rate limiting per `(client_id, grant_type)` |
| Certificate renewal | Per-device rate limit (at most one renewal per `renewSkew` window) |

Rate-limit buckets hold **counters + timestamps only** -- never IPs, destinations, or flows
(privacy invariant: this is abuse metering, not a connection log; LP-L-1 stays clean).

### 6.4 Audit log

Every identity mutation emits an audit event on the Redis Streams bus:

| Event | Payload |
|---|---|
| `enroll_token.minted` | `{token_id, kind, bind_kind, max_uses, not_after, created_by}` -- never the raw token |
| `device.enrolled` | `{device_id, os, kind, wg_pubkey_prefix_8b}` |
| `cert.issued` | `{device_id, serial, not_after}` |
| `cert.renewed` | `{device_id, serial, prior_serial}` |
| `device.revoked` | `{device_id, wg_pubkey, serial, reason}` |
| `oidc.login` | `{user_id, provider, tenant_id}` -- never the raw ID token |

The audit trail itself never carries secrets (S11 / AS-TOKEN). Control actions are audited;
traffic is not (S7). [04-security-privacy-pki.md SS7]

### 6.5 Threat mapping

| Mechanism | Invariant | Threats closed |
|---|---|---|
| Device keys generated on-device, sealed in OS keystore | S2 | T-CLI-I-1, T-PKI-S-1 (partial) |
| CSR proof-of-possession at sign | -- | T-PKI-S-1 |
| mTLS leaf <= 24 h + auto-renew | S4 | T-CLI-S-1, T-CP-S-1, T-COORD-S-1 |
| OIDC PKCE + JWKS pin + nonce/state single-use + DPoP for device flow | S4 | T-CP-S-2 |
| Anonymous enroll (no email / no SSO) | -- | LP-I-1 (identifying), LP-N-1 (partial) |
| Enroll token: hashed, single-use, TTL, kind-bound | S11 | AS-TOKEN abuse, token enumeration |
| Constant-time token check, no field oracle | -- | Token enumeration oracle |
| Sub-second revocation cascade | S5 | T-PKI-E-1 |
| RLS `FORCE ROW LEVEL SECURITY` | P7 | T-CP-T-1, T-CP-I-1 (cross-tenant) |

---

## 7. SLOs & performance targets

All figures are **design targets**, not measured results (SS11.4.6). They become measured SLOs
once the SS9 soak captures them on real hardware.

| Metric | Target | Histogram |
|---|---|---|
| Enroll -> first NetworkMap delivered | **p99 < 2 s** | `helix_enroll_to_map_seconds` |
| Device revoke -> edge WG-peer removed | **p99 < 1 s** | `helix_pki_revoke_seconds` |
| Cert issue (enroll path) | < 150 ms | `helix_pki_issue_seconds` |
| Cert renew round-trip | < 150 ms | `helix_pki_renew_seconds` |
| `AuthDevice` hot path (cache hit) | < 1 ms | `helix_pki_authdevice_seconds` |
| Enroll token verify (const-time argon2id) | 30-150 ms | `helix_token_verify_seconds` |
| OIDC token exchange (device flow) | < 500 ms | `helix_oidc_device_token_seconds` |

---

## 8. Phase-2 forward seams

The Phase-1 interfaces extend without reshaping:

- **Platform attestation** -- the reserved `EnrollRequest.attestation` field carries Apple
  App Attest / Android Key Attestation / Windows TPM attestation blobs. Phase-2 tenants can
  require hardware-bound keys, closing residual R-DEV.
- **SCIM provisioning + group sync** -- `oidc_providers` gains push-provisioning for large
  tenants; role mapping stays the same.
- **WebAuthn / passkey step-up** -- `sessions` gains a hardware-second-factor for `admin`
  actions (revoke, CA rotation).
- **Anonymous recovery codes** -- `accounts` gains optional recovery codes (still no PII)
  and a multi-tenant `number->tenant` directory behind operator opt-in.
- **PQ flip** -- `device_pq_material.enabled=true` with `kem='x25519_mlkem768'` (Mullvad-parity
  adds `+mceliece`). Config, not a schema change -- the table + proto fields already exist.
- **Service extraction** -- the `PKI` and `Identity` interfaces are the seams along which
  standalone services split from the monolith.

---

## Sources verified

- [identity-and-enrollment.md] HelixVPN -- full identity + enrollment security spec. SS1 identity
  model (two postures), SS2 key-never-leaves invariant + proof obligations (K1-K5), SS3 enrollment
  protocol + Enroll wire contract, SS4 OIDC SSO (auth-code + PKCE + claim->role), SS5 anonymous
  device-token path (account-number, enroll-token), SS6 enroll-token security (argon2id, atomic
  single-use, constant-time), SS7 re-enrollment + revocation cascade, SS8 threat mapping, SS9
  test matrix (SS11.4.169), SS10 Phase-2 seams. (Read 2026-07-08.)
- [pki-and-certs.md] HelixVPN -- full PKI spec. SS1 WireGuard-native-identity baseline +
  SPIFFE two-identity split, SS2 CA hierarchy (three tiers, Ed25519, offline root + online
  issuing), SS3 cert profiles (fields, TTLs, `spiffe://` SAN), SS4 lifecycle state machine,
  SS5 short-lived mTLS + auto-renew (renewSkew, make-before-break), SS6 rotation schedule,
  SS7 revocation sub-second target + two-teeth, SS8 mTLS everywhere + `AuthDevice` hot path,
  SS9 PQ material slot (hybrid never PQ-only), SS10 key storage (KMS/HSM + device secure element),
  SS11 threat mapping + test matrix. (Read 2026-07-08.)
- [svc-pki.md] HelixVPN -- Go implementation spec for `internal/pki`. SS1 scope + WG reality
  baseline, SS2 CA hierarchy + compromise fast-path (Rev 2, 2026-07-04), SS3 DDL/RLS (ca_keys,
  device_certs extended, device_wg_keys, device_pq_material), SS4 Go interface + types
  (`PKI`, `CASigner`, `AuthedDevice`), SS5 issuance algorithm + state machine, SS6 auto-renew
  (renewSkew, sweeper), SS7 WG-key rotation (overlap-then-retire), SS8 protobuf + `AuthDevice`
  hot path + revoke cache, SS9 events + revocation algorithm, SS10 error taxonomy + SLOs,
  SS11 test points, SS12 task plan, SS13 Phase-2 seams. (Read 2026-07-08.)
- [04-security-privacy-pki.md] HelixVPN security spine. SS0.1 invariants S1-S11, SS1 trust
  model + two-channel auth, SS2 identity (OIDC managed + anonymous device tokens), SS3
  enrollment, SS4 PKI. (Read 2026-07-08.)
- [04_P1] HelixVPN-Phase1-MVP.md SS6 (two identity modes, enrollment flow, key-never-leaves,
  24h cert, revoke < 1 s). (Read 2026-07-08.)
- [research-pki_pq_nat] SS1.1 WireGuard native identity (public key IS identity, no built-in
  PKI), SS1.2 SPIFFE/SPIRE two-identity split, SS2.1 hybrid PQ PSK. Access date 2026-06-25.
- Tailscale identity architecture (2025) -- multi-layer identity (machine key + node key +
  user identity + tags), tsidp OIDC provider (zero-click auth), workload identity federation
  (short-lived OIDC tokens for CI/CD). Retrieved from tailscale.com/docs/concepts/tailscale-identity,
  tailscale.com/blog/zero-trust-with-zero-clicks-a-new-take-on-idps,
  tailscale.com/blog/workload-identity-beta. Access date 2026-07-08.
- OIDC Device Authorization Grant RFC 8628 + DPoP (2025 best practices) -- draft-parecki-oauth-dpop-device-flow
  (September 2025) formally binds DPoP to device flow; 2024-2025 attack landscape (ShinyHunters/UNC6040)
  exploits device flow via social engineering; DPoP binding closes device-code theft vector.
  Implementation: Zitadel/oidc, GitHub CLI, Stripe CLI, AWS SSO. Retrieved from
  datatracker.ietf.org/doc/html/draft-parecki-oauth-dpop-device-flow-00,
  dev.to/deepakgupta/preventing-oauth-device-flow-attacks. Access date 2026-07-08.
- Mullvad anonymous enrollment -- 16-digit random account number, zero PII, cash/Monero payments,
  2023 Swedish police raid found zero customer data (The Verge, 2023-04-21). Confirmed Mullvad's
  design as the privacy-VPN gold standard. Retrieved from theverge.com/2023/4/21/23692580/mullvad-vpn-raid-sweden-police.
  Access date 2026-07-08.
- Constitution: SS11.4.6 (no-guessing -- UNVERIFIED marks, latency stated as targets), SS11.4.8
  (deep web research), SS11.4.10 (credentials never logged/git-tracked), SS11.4.27/SS11.4.85/
  SS11.4.107/SS11.4.169/SS1.1 (test-type coverage + self-validated analyzers), SS11.4.35
  (canonical-root inheritance), SS11.4.44 (revision header), SS11.4.99 (latest-source verification).
