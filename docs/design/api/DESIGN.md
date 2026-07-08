# Helix VPN -- REST API + Connect Handler Design (HVPN-P1-080, P1-081)

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Status:** active
**Scope:** HVPN-P1-080 (Connect/gRPC handler) + HVPN-P1-081 (REST routes, auth, RBAC, RLS)
**Authority:** This document is a **synthesizing design** document. It consolidates the detailed
nano-specs for the `internal/api` package (`v03-control-plane/svc-api.md`), the `Coordinator`
service contract (`v03-control-plane/protobuf-spec.md`), the identity model
(`docs/design/identity/DESIGN.md`), and the policy model (`docs/design/policies_spec/DESIGN.md`)
into a single implementation-ready design. Where this document and a nano-detail spec conflict on
a wire-level, DB-schema, or cryptographic detail, the nano-detail spec wins.

---

## Table of contents

- [1. Architecture overview](#1-architecture-overview)
- [2. Multiplexed server -- one TLS listener, three surfaces](#2-multiplexed-server--one-tls-listener-three-surfaces)
- [3. mTLS device authentication](#3-mtls-device-authentication)
- [4. Go type definitions -- concrete structs](#4-go-type-definitions--concrete-structs)
- [5. REST routes (P1-081.1)](#5-rest-routes-p1-0811)
- [6. Auth + RBAC + RLS (P1-081.2, P1-081.3)](#6-auth--rbac--rls-p1-0812-p1-0813)
- [7. Connect handler -- device-facing RPCs (P1-080)](#7-connect-handler--device-facing-rpcs-p1-080)
- [8. WebSocket / SSE live stream](#8-websocket--sse-live-stream)
- [9. Error taxonomy (unified REST + Connect)](#9-error-taxonomy-unified-rest--connect)
- [10. Rate limiting](#10-rate-limiting)
- [11. Metrics](#11-metrics)
- [12. File layout in helix-go](#12-file-layout-in-helix-go)
- [13. Phase-2 forward seams](#13-phase-2-forward-seams)
- [Sources verified](#sources-verified)

---

## 1. Architecture overview

Helix VPN surfaces two distinct audiences through **one** multiplexed HTTP server on a single
TLS listener (`:8443`):

| Audience | Surface | Protocol | Auth |
|---|---|---|---|
| Admin Console (browser) | Gin REST | HTTP/1.1 or HTTP/2 | OIDC session cookie OR API token (Bearer) |
| Automation (CI, scripts) | Gin REST | HTTP/1.1 or HTTP/2 | API token (Bearer) |
| Devices (agents) | Connect (gRPC / gRPC-Web) | HTTP/2 (gRPC) or HTTP/1.1 (Connect) | mTLS device cert |

```
                    +-------------------------------------------------+
                    |          TLS Listener :8443                     |
                    |  ALPN: h2, http/1.1                             |
                    |  ClientAuth: VerifyClientCertIfGiven            |
                    +---------------------+---------------------------+
                                          |
                                   +-----v------+
                                   | top-level   |
                                   | http.ServeMux|
                                   +-----+------+
                                         |
                    +--------------------+--------------------+
                    |                                          |
                    v                                          v
        +-----------+----------+               +---------------+--------------+
        | Connect mux           |               | Gin Engine                    |
        | /helix.coordinator.v1.|               | /v1/* (REST)                  |
        |  * (agent RPC paths)  |               | /v1/stream (WS/SSE)           |
        +-----------+----------+               | /healthz, /readyz, /metrics    |
                    |                          +---------------+--------------+
        +-----------+--------------+                           |
        |                          |             +-------------+----------+
        v                          v             v                        v
   +---------+             +-----------+   +----------+            +-----------+
   | Enroll  |             | Enroll    |   | authn:   |            | Ops       |
   | token   |             | handler   |   | session  |            | handlers  |
   | verify  |             +-----------+   | or API   |            +-----------+
   +---------+             | Coordinator|   | token    |
                           | streams    |   +-----+----+
                           +-----------+         |
                                      +---------v------+
                                      | rbac:           |
                                      | requireRole     |
                                      +---------+------+
                                                |
                                      +---------v------+
                                      | store.WithTenant|
                                      | (RLS backstop)  |
                                      +-----------------+
```

### 1.1 Design principles

| # | Principle | Enforced by |
|---|---|---|
| D1 | **One listener, no sidecar proxy.** A single `*http.Server` hosts REST + Connect + WS/SSE. | `http.ServeMux` path-prefix routing |
| D2 | **No `WriteTimeout`.** `WatchNetworkMap` and `/v1/stream` are intentionally infinite long-lived streams. | Liveness via per-stream keepalive (20 s), `ReadHeaderTimeout` for slowloris |
| D3 | **ClientAuth is `VerifyClientCertIfGiven`**, not `RequireAndVerify`. REST/WS callers present no client cert. Connect callers present a device mTLS cert. Absence is permitted; presented-and-invalid aborts the handshake. | TLS config + per-route auth enforcement |
| D4 | **Connect is the agent protocol.** `buf` Connect (connectrpc.com/connect) serves `Coordinator` as gRPC, gRPC-Web, and Connect protocol on one handler. | connect-go library |
| D5 | **Schema-first, no-drift.** Protobuf generates Go/Dart/Rust stubs. OpenAPI generates TS/Dart app clients. `buf breaking` + `oasdiff` in CI prevent wire-incompatible changes. | CI gates |
| D6 | **Fail-closed, default-deny.** Every `/v1/*` route carries `requireRole`. An unguarded route aborts startup. | Startup route-audit |

### 1.2 Go type: Server

```go
// internal/api/server.go

// Server is the multiplexed HTTP server hosting Gin REST + Connect RPC + WS/SSE.
type Server struct {
    gin   *gin.Engine      // REST + WS/SSE
    conn  http.Handler     // Connect mux (Coordinator handlers)
    http  *http.Server
    deps  Deps             // injected service interfaces
}

// Deps is the dependency injection bundle consumed by all api handlers.
type Deps struct {
    Registry    Registry        // device/connector CRUD
    PKI         PKI             // cert issue/revoke/verify
    Policy      policy.Compiler // policy parse/compile/activate
    Bus         events.Bus      // event publish/subscribe
    Store       *store.Store    // Postgres pool (RLS tenant-tx)
    Coordinator *coordinator.Coordinator // map build + stream fan-out
    OIDC        *oidc.Verifier  // OIDC token verification
}
```

---

## 2. Multiplexed server -- one TLS listener, three surfaces

### 2.1 Mux wiring

Connect handler paths are prefixed with the proto package path `"/helix.coordinator.v1."`.
Everything else is routed to Gin.

```go
const connectPathPrefix = "/helix.coordinator.v1."

func (s *Server) handler() http.Handler {
    mux := http.NewServeMux()
    mux.Handle(connectPathPrefix, s.conn) // routes to Connect mux
    mux.Handle("/", s.gin)                // routes to Gin (REST/WS/SSE/ops)
    return mux
}
```

### 2.2 Startup

```go
func (s *Server) Run(ctx context.Context, addr string, tlsCfg *tls.Config) error {
    h2s := &http2.Server{}
    s.http = &http.Server{
        Addr:              addr,
        Handler:           h2c.NewHandler(s.handler(), h2s), // h2c for plaintext dev
        TLSConfig:         tlsCfg,
        ReadHeaderTimeout: 5 * time.Second,
        IdleTimeout:       120 * time.Second,
        // NO WriteTimeout (D2)
    }
    go func() { <-ctx.Done(); _ = s.http.Shutdown(context.Background()) }()
    if tlsCfg != nil {
        return s.http.ListenAndServeTLS("", "")
    }
    return s.http.ListenAndServe()
}
```

### 2.3 TLS configuration

```go
tlsCfg := &tls.Config{
    MinVersion: tls.VersionTLS13,
    NextProtos: []string{"h2", "http/1.1"},
    ClientAuth: tls.VerifyClientCertIfGiven, // D3
    // ClientCAs: loaded per-request from the tenant CA pool (§3)
}
```

---

## 3. mTLS device authentication

### 3.1 `authDevice` -- resolve identity from client cert

The `AuthDevice` function resolves a verified TLS client certificate to an
`AuthedDevice` struct using `pki.VerifyCertChain`. It is invoked from the Connect
interceptor chain and from `AuthDeviceMiddleware` for any mTLS-authenticated Gin routes.

```go
// AuthedDevice is the resolved device identity from a verified mTLS client cert.
type AuthedDevice struct {
    DeviceID  uuid.UUID
    TenantID  uuid.UUID
    Kind      DeviceKind // client or connector
    Serial    string     // cert serial number (hex)
    NotAfter  time.Time  // cert expiry
    RevokedAt *time.Time // nil => active
    UserID    *uuid.UUID // nil for anonymous
}

// DeviceKind distinguishes the two agent classes.
type DeviceKind string

const (
    DeviceKindClient    DeviceKind = "client"
    DeviceKindConnector DeviceKind = "connector"
)

// AuthDevice resolves the device identity from a verified mTLS client certificate.
//
// Steps:
//   1. Extract *x509.Certificate from the TLS connection state.
//   2. Resolve the device row from device_certs.serial -> devices.
//   3. Verify the cert chain against the tenant CA pool via pki.VerifyCertChain.
//   4. Reject if device is revoked, cert is expired, or chain verification fails.
//   5. Return AuthedDevice with tenant, kind, serial, validity.
//
// The Enroll RPC is certificate-exempt (it authenticates by enroll token).
// All other Connect RPCs REQUIRE a valid, non-revoked device cert.
func AuthDevice(ctx context.Context, pki PKI) (AuthedDevice, error)
```

### 3.2 Connect interceptor chain

```
incoming Connect request
         |
    +----v----+
    | recover |   panic -> CodeInternal + audit
    +----+----+
         |
    +----v----+
    | requestID|  propagate/issue X-Request-Id
    +----+----+
         |
    +----v----+
    | validate |  protovalidate (buf.validate field constraints)
    +----+----+
         |
    +----v----+
    |deviceAuth|  mTLS cert -> AuthedDevice
    | (skip    |  Enroll is TOKEN-authenticated, cert-exempt
    |  Enroll) |
    +----+----+
         |
    +----v----+
    | metrics  |  helix_rpc_* histograms
    +----+----+
         |
    handler
```

```go
// internal/api/connect.go

func deviceAuthInterceptor(pki PKI) connect.UnaryInterceptorFunc {
    return func(next connect.UnaryFunc) connect.UnaryFunc {
        return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
            // Enroll is cert-exempt; token-authenticated instead.
            if req.Spec().Procedure == "helix.coordinator.v1.Coordinator/Enroll" {
                return next(ctx, req)
            }
            cert := peerCert(ctx)
            if cert == nil {
                return nil, connect.NewError(connect.CodeUnauthenticated,
                    errors.New("mTLS client certificate required"))
            }
            dev, err := pki.ResolveDeviceBySerial(ctx, cert.SerialNumber)
            if err != nil || dev.Revoked || time.Now().After(dev.CertNotAfter) {
                return nil, connect.NewError(connect.CodePermissionDenied,
                    errors.New("device certificate revoked or expired"))
            }
            ctx = context.WithValue(ctx, deviceCtxKey, dev)
            return next(ctx, req)
        }
    }
}
```

---

## 4. Go type definitions -- concrete structs

### 4.1 Principal (authenticated identity carried on the request context)

```go
// Principal is the authenticated identity of the caller, set on the
// gin.Context or context.Context by the authn/authz layer.
type Principal struct {
    TenantID uuid.UUID // the tenant this principal belongs to
    UserID   uuid.UUID // user row id; nil for device principals
    DeviceID uuid.UUID // device id; nil for user principals
    Role     string    // "admin" | "operator" | "member" | "" (device)
    Subject  string    // human-readable identifier for audit logs
    Kind     string    // "user" | "device"
}
```

### 4.2 REST request/response types

#### Enroll tokens

```go
// MintEnrollTokenReq is the request body for POST /v1/enroll-tokens.
type MintEnrollTokenReq struct {
    Kind       string `json:"kind"        binding:"required,oneof=client connector"`
    BindUserID string `json:"bind_user_id" binding:"omitempty,uuid"`
    TTLSeconds int    `json:"ttl_seconds"  binding:"omitempty,min=60,max=86400"` // default 900
    MaxUses    int    `json:"max_uses"     binding:"omitempty,min=1,max=1000"`    // default 1
}

// MintEnrollTokenResp is the response for POST /v1/enroll-tokens.
// The Token field is the ONLY place the raw token appears; it is never stored.
type MintEnrollTokenResp struct {
    ID        string `json:"id"`
    Token     string `json:"token"`        // shown ONCE; sha256(token) is all that's stored
    QRPNG     string `json:"qr_png_b64"`   // base64 PNG for Connector/Access scan
    Kind      string `json:"kind"`
    MaxUses   int    `json:"max_uses"`
    ExpiresAt string `json:"expires_at"`   // RFC 3339
}

// EnrollTokenView is the metadata-only view returned by GET /v1/enroll-tokens.
type EnrollTokenView struct {
    ID           string `json:"id"`
    Kind         string `json:"kind"`
    MaxUses      int    `json:"max_uses"`
    UsedCount    int    `json:"used_count"`
    ExpiresAt    string `json:"expires_at"`
    ConsumedAt   string `json:"consumed_at,omitempty"`
    ConsumedByID string `json:"consumed_device_id,omitempty"`
    CreatedBy    string `json:"created_by"`
    CreatedAt    string `json:"created_at"`
}
```

#### Devices

```go
// DeviceView is a tenant-scoped device summary returned by GET /v1/devices.
type DeviceView struct {
    ID         string `json:"id"`
    Kind       string `json:"kind"`         // client | connector
    Name       string `json:"name"`
    OverlayIP  string `json:"overlay_ip"`   // ULA /48 address
    OS         string `json:"os,omitempty"`
    Online     bool   `json:"online"`       // from Redis presence (ephemeral)
    LastSeenAt string `json:"last_seen_at,omitempty"` // COARSE
    Revoked    bool   `json:"revoked"`
    OwnerEmail string `json:"owner_email,omitempty"`  // null in anonymous mode
    EnrolledAt string `json:"enrolled_at"`
}

// ListDevicesResp is the paginated device list response.
type ListDevicesResp struct {
    Devices    []DeviceView `json:"devices"`
    NextCursor string       `json:"next_cursor,omitempty"` // keyset pagination
}

// PatchDeviceReq is the request body for PATCH /v1/devices/:id.
type PatchDeviceReq struct {
    Name   *string `json:"name,omitempty"    binding:"omitempty,min=1,max=128"`
    UserID *string `json:"user_id,omitempty" binding:"omitempty,uuid"`
}
```

#### Policies

```go
// CreatePolicyReq submits a new policy spec for dry-run compilation.
type CreatePolicyReq struct {
    Spec json.RawMessage `json:"spec" binding:"required"`
}

// CreatePolicyResp returns the compilation result.
type CreatePolicyResp struct {
    Version      int64            `json:"version"`
    Active       bool             `json:"active"` // false until /activate
    CompileStats CompileStats     `json:"compile_stats"`
    Conflicts    []RouteConflict  `json:"conflicts,omitempty"`
}

// CompileStats summarizes the compiled policy.
type CompileStats struct {
    DeviceCount     int `json:"device_count"`
    ACLRuleCount    int `json:"acl_rule_count"`
    PeerEdgeCount   int `json:"peer_edge_count"`
    ExitNodeCount   int `json:"exit_node_count"`
    TestPassCount   int `json:"test_pass_count"`
    TestFailCount   int `json:"test_fail_count"`
}

// RouteConflict is a warning about overlapping advertised CIDRs.
type RouteConflict struct {
    CIDR       string   `json:"cidr"`
    Connectors []string `json:"connector_ids"` // who advertises it
    Severity   string   `json:"severity"`      // "warning" | "error"
}
```

#### Networks

```go
// NetworksResp is the tenant overlay network summary.
type NetworksResp struct {
    ULAPrefix   string          `json:"ula_prefix"`     // "fd7a:helix:<rand>::/48"
    GatewayIP   string          `json:"gateway_ip"`     // "::1"
    DeviceCount int             `json:"device_count"`
    Connectors  []ConnectorView `json:"connectors"`
}

// ConnectorView summarizes a connector and its advertised prefixes.
type ConnectorView struct {
    DeviceID string   `json:"device_id"`
    SiteName string   `json:"site_name"`
    SiteID   uint32   `json:"site_id"` // for 4via6 disambiguation
    Prefixes []string `json:"prefixes"`
    Via6     []Via6   `json:"via6"`    // {ipv4_cidr, via6_prefix}
}

// Via6 is a 4via6 route mapping for IPv4 collision disambiguation.
type Via6 struct {
    IPv4CIDR   string `json:"ipv4_cidr"`
    Via6Prefix string `json:"via6_prefix"`
}
```

#### Audit

```go
// AuditEventView is a single control-action audit entry.
type AuditEventView struct {
    ID       int64  `json:"id"`
    Actor    string `json:"actor"`
    Action   string `json:"action"`   // e.g. "device.revoke", "policy.activate"
    Target   string `json:"target"`
    TS       string `json:"ts"`
    Meta     map[string]any `json:"meta,omitempty"`
}

// ListAuditResp is the paginated audit log response.
type ListAuditResp struct {
    Events     []AuditEventView `json:"events"`
    NextCursor string           `json:"next_cursor,omitempty"`
}
```

#### API tokens

```go
// MintAPITokenReq mints a programmatic API token.
type MintAPITokenReq struct {
    Name      string `json:"name"       binding:"required,min=1,max=128"`
    Role      string `json:"role"       binding:"required,oneof=admin operator member"`
    ExpiresAt string `json:"expires_at" binding:"omitempty"` // RFC 3339; omit => non-expiring
}

// MintAPITokenResp returns the once-shown token.
type MintAPITokenResp struct {
    ID    string `json:"id"`
    Token string `json:"token"` // shown ONCE; sha256 stored
    Name  string `json:"name"`
    Role  string `json:"role"`
}
```

---

## 5. REST routes (P1-081.1)

### 5.1 Route table (authoritative)

All routes are versioned under `/v1`. RBAC roles use the closed set `admin > operator > member`.

| Method | Path | Purpose | Admin | Operator | Member |
|---|---|---|---|---|---|
| POST | `/v1/enroll-tokens` | Mint a device enroll token | YES | YES | -- |
| GET | `/v1/enroll-tokens` | List unconsumed tokens (metadata only) | YES | YES | -- |
| DELETE | `/v1/enroll-tokens/:id` | Revoke an unconsumed token | YES | YES | -- |
| GET | `/v1/devices` | List tenant devices | YES | YES | YES |
| GET | `/v1/devices/:id` | One device detail | YES | YES | YES |
| PATCH | `/v1/devices/:id` | Rename / reassign owner | YES | YES | -- |
| POST | `/v1/devices/:id/revoke` | Revoke device | YES | -- | -- |
| GET | `/v1/connectors` | List connectors + prefixes | YES | YES | YES |
| POST | `/v1/connectors/:id/prefixes` | Set advertised CIDRs | YES | YES | -- |
| GET | `/v1/groups` | List policy groups | YES | YES | YES |
| POST | `/v1/groups` | Create group | YES | YES | -- |
| PUT | `/v1/groups/:id/members` | Replace group membership | YES | YES | -- |
| GET | `/v1/policies` | List policy versions (active flagged) | YES | YES | YES |
| POST | `/v1/policies` | Submit new policy spec (dry-run compile) | YES | YES | -- |
| POST | `/v1/policies/:v/activate` | Activate policy version v (atomic flip) | YES | YES | -- |
| GET | `/v1/networks` | Tenant overlay summary | YES | YES | YES |
| GET | `/v1/audit` | Control-action audit log (paginated) | YES | YES | -- |
| POST | `/v1/api-tokens` | Mint a programmatic API token | YES | -- | -- |
| DELETE | `/v1/api-tokens/:id` | Revoke an API token | YES | -- | -- |
| GET | `/v1/stream` | WS/SSE live event subscription | YES | YES | YES |
| GET | `/v1/me` | Current principal (user/role/tenant) | YES | YES | YES |
| GET | `/healthz` `/readyz` `/metrics` | Ops endpoints (no auth) | -- | -- | -- |

### 5.2 Route registration (Go)

RBAC is declared at route registration, never inside handlers.

```go
// internal/api/routes.go

func (s *Server) registerREST(r *gin.Engine, h *Handlers) {
    v1 := r.Group("/v1", s.authn())     // every /v1 route authenticates first
    {
        // Enroll tokens -- admin + operator
        et := v1.Group("/enroll-tokens", requireRole("admin", "operator"))
        et.POST("",       h.MintEnrollToken)
        et.GET("",        h.ListEnrollTokens)
        et.DELETE("/:id", h.RevokeEnrollToken)

        // Devices
        dev := v1.Group("/devices")
        dev.GET("",            requireRole("admin", "operator", "member"), h.ListDevices)
        dev.GET("/:id",        requireRole("admin", "operator", "member"), h.GetDevice)
        dev.PATCH("/:id",      requireRole("admin", "operator"),           h.PatchDevice)
        dev.POST("/:id/revoke", requireRole("admin"),                       h.RevokeDevice)

        // Connectors
        con := v1.Group("/connectors")
        con.GET("",              requireRole("admin", "operator", "member"), h.ListConnectors)
        con.POST("/:id/prefixes", requireRole("admin", "operator"),           h.SetPrefixes)

        // Groups
        grp := v1.Group("/groups")
        grp.GET("",            requireRole("admin", "operator", "member"), h.ListGroups)
        grp.POST("",           requireRole("admin", "operator"),           h.CreateGroup)
        grp.PUT("/:id/members", requireRole("admin", "operator"),           h.SetGroupMembers)

        // Policies
        pol := v1.Group("/policies")
        pol.GET("",             requireRole("admin", "operator", "member"), h.ListPolicies)
        pol.POST("",            requireRole("admin", "operator"),           h.CreatePolicy)
        pol.POST("/:v/activate", requireRole("admin", "operator"),           h.ActivatePolicy)

        // Networks (read-only)
        v1.GET("/networks", requireRole("admin", "operator", "member"), h.GetNetworks)

        // Audit
        v1.GET("/audit", requireRole("admin", "operator"), h.ListAudit)

        // API tokens
        at := v1.Group("/api-tokens", requireRole("admin"))
        at.POST("",       h.MintAPIToken)
        at.DELETE("/:id", h.RevokeAPIToken)

        // Live stream + self
        v1.GET("/stream", requireRole("admin", "operator", "member"), h.Stream)
        v1.GET("/me",     requireRole("admin", "operator", "member"), h.Me)
    }
}
```

### 5.3 Startup route-audit (default-deny enforcement)

A boot-time check walks every registered `/v1/*` route and asserts it carries a `requireRole`
handler. An unguarded route aborts startup.

```go
// internal/api/audit_routes.go

func assertEveryV1RouteGuarded(routes gin.RoutesInfo) error {
    for _, rt := range routes {
        if strings.HasPrefix(rt.Path, "/v1/") && !chainHasRBAC(rt.HandlerName) {
            return fmt.Errorf("route %s %s missing requireRole: §11.4.6/C4 violation",
                rt.Method, rt.Path)
        }
    }
    return nil
}
```

### 5.4 Key handler patterns

**`POST /v1/devices/:id/revoke` (admin only):**

```go
func (h *Handlers) RevokeDevice(c *gin.Context) {
    id := mustUUID(c.Param("id"))
    p := principalOf(c)
    err := h.store.WithTenant(c, p.TenantID, func(q *db.Queries) error {
        if err := q.MarkDeviceRevoked(c, id); err != nil {
            return err
        }
        if err := q.RevokeDeviceCerts(c, id); err != nil {
            return err
        }
        _, err := h.bus.Publish(c, "events:devices",
            events.New("device.revoked", p.TenantID, p.Subject,
                map[string]any{"device_id": id.String()}))
        return err
    })
    writeResultOrError(c, err, http.StatusAccepted)
    // 202 Accepted: DB write + event publish are synchronous;
    // WG-peer removal at the edge is event-driven within the < 1 s convergence SLO.
}
```

**`POST /v1/policies` (dry-run compile, fail-closed):**

```go
func (h *Handlers) CreatePolicy(c *gin.Context) {
    var req CreatePolicyReq
    if err := c.ShouldBindJSON(&req); err != nil {
        writeError(c, http.StatusBadRequest, "validation_failed", err.Error())
        return
    }
    p := principalOf(c)
    spec, err := policy.ParseSpec(req.Spec)
    if err != nil {
        writePolicyError(c, err) // 422 with structured PolicyCompileError
        return
    }
    // Dry-run compile BEFORE any DB insert (fail-closed: nothing written on error).
    compiled, err := h.deps.Policy.Compile(c, p.TenantID, spec)
    if err != nil {
        writePolicyError(c, err)
        return
    }
    // Insert policies row (spec stored, not yet active).
    var version int64
    err = h.store.WithTenant(c, p.TenantID, func(q *db.Queries) error {
        v, err := q.InsertPolicy(c, /* spec, compiled, false */)
        version = v
        return err
    })
    if err != nil {
        writeDBError(c, err)
        return
    }
    c.JSON(http.StatusCreated, CreatePolicyResp{
        Version:  version,
        Active:   false,
        CompileStats: compileStats(compiled),
    })
}
```

---

## 6. Auth + RBAC + RLS (P1-081.2, P1-081.3)

### 6.1 Four-layer authorization chain

```
incoming request
      |
      v
+-------------+
| L1: authn   |  Who are you?
| OIDC session|  - Console browser: session cookie (sha256 hashed)
| OR API token|  - Automation: Bearer hvpn_<token> (sha256 hashed)
| OR mTLS cert|  - Devices: verified client cert -> device row
+------+------+
       |
       v
+-------------+
| L2: rbac    |  May this role access this route?
| requireRole |  - admin > operator > member (closed set)
+------+------+  - Declared at route registration, not inside handlers
       |
       v
+-------------+
| L3: RLS     |  Only this tenant's rows -- enforced AT THE DATABASE
| WithTenant  |  - SET LOCAL app.tenant_id per transaction
| + FORCE RLS |  - Postgres FORCE ROW LEVEL SECURITY
+------+------+  - Non-superuser role (helix_app), never the table owner
       |
       v
  [handler logic]
```

### 6.2 OIDC/session auth (console browser)

The Console authenticates via standard OIDC Authorization Code + PKCE (RFC 7636) against
any IdP (Keycloak, Authentik, Auth0, Entra, Okta). Helix VPN is a Relying Party, never an IdP.

```go
// internal/api/authn.go

func (s *Server) authn() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Bearer API token? "Authorization: Bearer hvpn_<...>"
        if raw, ok := bearerToken(c); ok && strings.HasPrefix(raw, "hvpn_") {
            p, err := s.resolveAPIToken(c, sha256sum(raw))
            if err != nil {
                abort(c, http.StatusUnauthorized, "invalid_token", nil)
                return
            }
            setPrincipal(c, p)
            c.Next()
            return
        }
        // 2. Session cookie? (Console browser)
        if cookie, err := c.Cookie("helix_session"); err == nil {
            p, err := s.resolveSession(c, sha256sum(cookie))
            if err != nil {
                abort(c, http.StatusUnauthorized, "invalid_session", nil)
                return
            }
            setPrincipal(c, p)
            c.Next()
            return
        }
        abort(c, http.StatusUnauthorized, "missing_credentials", nil)
    }
}

func (s *Server) resolveSession(ctx context.Context, tokenHash string) (Principal, error) {
    row, err := s.deps.Store.LookupSession(ctx, tokenHash)
    if err != nil || row.Revoked || time.Now().After(row.ExpiresAt) {
        return Principal{}, ErrSessionInvalid
    }
    return Principal{
        TenantID: row.TenantID,
        UserID:   row.UserID,
        Role:     row.Role,
        Subject:  row.UserEmail,
        Kind:     "user",
    }, nil
}

func (s *Server) resolveAPIToken(ctx context.Context, tokenHash string) (Principal, error) {
    row, err := s.deps.Store.LookupAPIToken(ctx, tokenHash)
    if err != nil || row.Revoked || (row.ExpiresAt != nil && time.Now().After(*row.ExpiresAt)) {
        return Principal{}, ErrTokenInvalid
    }
    // Async coarse last_used_at update (throttled to at-most-once-per-minute, C3).
    go s.markTokenUsed(row.ID)
    return Principal{
        TenantID: row.TenantID,
        UserID:   row.UserID,
        Role:     string(row.Role),
        Subject:  row.Name,
        Kind:     "user",
    }, nil
}
```

The OIDC login flow (Authorization Code + PKCE) is:
1. `GET /v1/auth/login?provider=<idp_id>` redirects to the IdP
2. The IdP redirects back to `/v1/auth/callback?code=...&state=...`
3. The callback exchanges the code for tokens, validates the ID token against JWKS,
   upserts the `users` row keyed on `(tenant_id, oidc_sub)`, mints a `sessions` row
   with a random 256-bit session value (only sha256 stored), and sets the
   `helix_session` cookie.

**Role from group claim.** Role is asserted from an IdP group claim mapped per tenant via
`oidc_providers.role_map`. The claim is re-evaluated on every login. A user removed from
all mapped groups falls to `default_role`. `SuspendedAt` user => `ErrUserSuspended`.

```go
type TenantOIDC struct {
    TenantID   uuid.UUID
    Issuer     string            // pinned; mismatch => reject
    ClientID   string            // aud must contain this
    JWKSURL    string            // refreshed on kid-miss with backoff
    GroupClaim string            // e.g. "groups"
    RoleMap    map[string]string // group name -> role
}
```

### 6.3 API token auth (automation)

```go
// Token format: hvpn_<32 random bytes, base64url>
// Stored as sha256(token) in api_tokens.token_hash.
// The raw token is shown ONCE at mint (POST /v1/api-tokens).
// Roles: admin, operator, member -- the token carries a fixed role.
```

### 6.4 RBAC middleware

```go
// requireRole returns a handler that blocks requests from principals
// whose role is not in the allowed set.
func requireRole(allowed ...string) gin.HandlerFunc {
    set := toSet(allowed)
    return func(c *gin.Context) {
        if !set[principalOf(c).Role] {
            abort(c, http.StatusForbidden, "insufficient_role", nil)
            return
        }
        c.Next()
    }
}
```

### 6.5 RLS backstop (defense-in-depth)

Every handler runs DB work through `store.WithTenant`, which sets `app.tenant_id` per
transaction. Postgres `FORCE ROW LEVEL SECURITY` enforces tenant isolation even if RBAC
is misconfigured or a `WHERE` clause is forgotten.

```go
// internal/store/tenant.go

// WithTenant runs fn inside a transaction whose RLS scope is pinned to tenantID.
func (s *Store) WithTenant(ctx context.Context, tenantID uuid.UUID,
    fn func(q *db.Queries) error) error {
    tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return err
    }
    defer func() { _ = tx.Rollback(ctx) }() // no-op after Commit
    if _, err := tx.Exec(ctx,
        "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
        return fmt.Errorf("store: pin tenant: %w", err)
    }
    if err := fn(db.New(tx)); err != nil {
        return err
    }
    return tx.Commit(ctx)
}
```

**RBAC roles (closed set):**

| Role | Description |
|---|---|
| `admin` | Full tenant control: revoke devices, mint tokens, manage all resources. |
| `operator` | Operational control: manage devices, policies, groups, enroll tokens. Cannot revoke devices or mint API tokens. |
| `member` | Read-only access to devices, connectors, groups, policies, networks, stream. |

### 6.6 API-owned DDL (sessions, API tokens, enroll tokens)

These tables exist solely to serve authentication. They are tenant-scoped and RLS-protected.
All secrets are stored as `sha256` hashes only; the raw value appears once at mint.

```sql
-- Browser/console sessions (OIDC login)
CREATE TABLE sessions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id      uuid NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    token_hash   bytea NOT NULL,            -- sha256(opaque cookie value)
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    revoked_at   timestamptz,
    user_agent   text,                      -- COARSE; NO IP address (C3)
    UNIQUE (tenant_id, token_hash)
);

-- Programmatic API tokens (CI / automation)
CREATE TABLE api_tokens (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id      uuid REFERENCES users(id) ON DELETE SET NULL,
    name         text NOT NULL,
    token_hash   bytea NOT NULL,            -- sha256 of the displayed-once secret
    role         user_role NOT NULL,        -- the role the token carries
    created_at   timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,               -- COARSE; updated at most once/min, async
    expires_at   timestamptz,               -- nullable: non-expiring (discouraged)
    revoked_at   timestamptz,
    UNIQUE (tenant_id, name),
    UNIQUE (tenant_id, token_hash)
);

-- Device enroll tokens (anonymous + managed)
CREATE TABLE enroll_tokens (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash         bytea NOT NULL,      -- sha256 of the raw token
    kind               device_kind NOT NULL,-- the device_kind this token may enroll
    max_uses           int NOT NULL DEFAULT 1,
    used_count         int NOT NULL DEFAULT 0,
    bound_user_id      uuid REFERENCES users(id) ON DELETE SET NULL,
    created_by         uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    expires_at         timestamptz NOT NULL,
    consumed_device_id uuid REFERENCES devices(id) ON DELETE SET NULL,
    UNIQUE (tenant_id, token_hash)
);
```

---

## 7. Connect handler -- device-facing RPCs (P1-080)

### 7.1 `Coordinator` service (proto package `helix.coordinator.v1`)

| RPC | Kind | Auth | Description |
|---|---|---|---|
| `Enroll` | unary | Enroll token | Exchange a single-use token for identity + cert. The ONLY unauthenticated-by-cert RPC. |
| `WatchNetworkMap` | **server-stream** | Device mTLS | The spine: snapshot then deltas, p99 < 1 s convergence. |
| `AdvertisePrefixes` | unary | Device mTLS (connector) | A connector pushes its advertised CIDRs. Idempotent. |
| `ReportStatus` | unary | Device mTLS | Presence + transport + RTT heartbeat. Carries NO traffic data (C3). |

### 7.2 Enroll RPC -- flow

```
Device (helix-core)                  helixd (api/identity/pki)
     |                                        |
     |-- generate WG+leaf keypairs            |
     |   LOCALLY (S2: keys never leave)       |
     |                                        |
     |-- Connect:Enroll --------------------->|
     |   {token, wg_pubkey, csr, os,          |
     |    name, kind}                         |
     |                                        |
     |              [atomic consume token]    |
     |              [IPAM.AllocOverlayIP]     |
     |              [PKI.IssueDeviceCert]     |
     |              [INSERT devices + certs   |
     |               + wg_keys -- ONE TX]     |
     |              [emit device.enrolled]    |
     |                                        |
     |<-- EnrollResponse ---------------------|
     |   {device_id, overlay_ip,              |
     |    device_cert, ca_chain,              |
     |    gateway, cert_ttl_s}                |
     |                                        |
     |-- persist cert+keys in                 |
     |   OS keystore                          |
     |                                        |
     |-- WatchNetworkMap (mTLS) ------------>|
     |   (authenticated by the NEW cert)      |
```

### 7.3 Connect handler wiring

```go
// internal/api/connect.go

func newConnectMux(d Deps) http.Handler {
    interceptors := connect.WithInterceptors(
        recoverInterceptor(),           // panic -> CodeInternal + audit
        requestIDInterceptor(),         // X-Request-Id propagation
        validateInterceptor(),          // protovalidate
        deviceAuthInterceptor(d.PKI),   // mTLS -> AuthedDevice (skips Enroll)
        metricsInterceptor(),           // helix_rpc_* histograms
    )
    path, handler := coordinatorv1connect.NewCoordinatorHandler(
        &coordinatorServer{d: d}, interceptors)
    mux := http.NewServeMux()
    mux.Handle(path, handler)
    return mux
}

type coordinatorServer struct{ d Deps }

func (s *coordinatorServer) Enroll(
    ctx context.Context, req *connect.Request[coordinatorv1.EnrollRequest],
) (*connect.Response[coordinatorv1.EnrollResponse], error) {
    // 1. Verify enroll token (constant-time argon2id comparison)
    // 2. Atomic consume: UPDATE ... WHERE consumed_at IS NULL ... RETURNING
    // 3. IPAM alloc overlay IP
    // 4. PKI sign device cert from CSR
    // 5. INSERT devices + certs + wg_keys (ONE WithTenant TX)
    // 6. Publish device.enrolled event
    // 7. Return EnrollResponse with device_id, cert, ca_chain, gateway
}

func (s *coordinatorServer) WatchNetworkMap(
    ctx context.Context, req *connect.Request[coordinatorv1.WatchRequest],
    stream *connect.ServerStream[coordinatorv1.MapUpdate],
) error {
    // Delegates to coordinator.Coordinator.WatchNetworkMap
    return s.d.Coordinator.WatchNetworkMap(ctx, req, stream)
}

func (s *coordinatorServer) AdvertisePrefixes(
    ctx context.Context, req *connect.Request[coordinatorv1.AdvertiseRequest],
) (*connect.Response[coordinatorv1.AdvertiseResponse], error) { /* ... */ }

func (s *coordinatorServer) ReportStatus(
    ctx context.Context, req *connect.Request[coordinatorv1.StatusReport],
) (*connect.Response[coordinatorv1.StatusAck], error) { /* ... */ }
```

### 7.4 `WatchNetworkMap` stream lifecycle

The connect-go handler delegates to the coordinator's stream loop:

1. **Authenticate** -- mTLS cert -> `AuthedDevice`; reject revoked/expired
2. **`known_version == 0`** => send full `NetworkMap` snapshot
3. **`0 < known_version <= current`** => replay compacted deltas since that version (cheap reconnect)
4. **`known_version > current`** => defensive full snapshot
5. **Streaming** -- forward deltas on event, keepalive every 20 s
6. **Terminate** on: device revoked, stream error, client disconnect, slow consumer (bounded queue full)

**Streaming-error caveat:** Connect protocol returns HTTP 200 for streaming responses; the real
status rides in the gRPC trailer. The handler returns `connect.NewError(code, ...)` for the
agent client to read from the trailer.

### 7.5 Enroll RPC security properties

| Property | Mechanism |
|---|---|
| Token never stored plaintext | `sha256(token)` in `enroll_tokens.token_hash` |
| Atomic single-use | `UPDATE ... WHERE consumed_at IS NULL ... RETURNING` |
| Constant-time comparison | Candidate hashes compared in constant time |
| No field oracle | Failure returns opaque `PermissionDenied` |
| Kind must match | `EnrollRequest.kind` must match `enroll_tokens.kind` |
| Concurrency-safe | Exactly one `Enroll` wins the atomic consume per token |
| All-or-nothing TX | Token consume + device insert + IP alloc + cert issue are ONE transaction |
| Rate-limited | Per-source-IP token bucket (ephemeral key, not logged -- C3) |

---

## 8. WebSocket / SSE live stream

`GET /v1/stream` provides Console real-time event delivery. It upgrades to WebSocket if the
client sends `Upgrade: websocket`; otherwise it serves SSE (`Accept: text/event-stream`).

### 8.1 Protocol negotiation

Both WS and SSE deliver the identical event envelope:

```json
{
  "type": "device.online",
  "tenant_id": "<uuid>",
  "ts": "RFC3339",
  "data": { "device_id": "<uuid>", "name": "alice-laptop" },
  "trace_id": "<request-or-event-id>"
}
```

### 8.2 Subscribable event types

| `type` | Role floor |
|---|---|
| `device.online` / `device.offline` | member |
| `device.enrolled` / `device.revoked` | operator |
| `route.changed` | member |
| `route.conflict` | operator |
| `policy.compiled` | operator |
| `handshake.failing` | operator |
| `gateway.failover` | operator |

### 8.3 Hub mechanics

```go
// internal/api/stream.go

type subscriber struct {
    tenant uuid.UUID
    role   string
    out    chan []byte // bounded (cap 256); full => drop subscriber
    done   chan struct{}
}

func (h *Hub) run(ctx context.Context) {
    evs := h.bus.Subscribe(ctx, "coordinator-ws", eventStreams...)
    for {
        select {
        case <-ctx.Done():
            return
        case env := <-evs:
            for _, s := range h.subsFor(env.TenantID) {
                if !roleMaySee(s.role, env.Type) {
                    continue
                }
                msg := projectAppEnvelope(env, s.role)
                select {
                case s.out <- msg:
                    // delivered
                default:
                    h.drop(s) // slow consumer -> drop, not unbounded memory
                }
            }
        }
    }
}
```

A WS keepalive ping every 20 s (and SSE comment heartbeat) proves liveness without a
global `WriteTimeout`.

---

## 9. Error taxonomy (unified REST + Connect)

### 9.1 APIError type

```go
// APIError is the unified error response for both REST (JSON) and Connect (trailer detail).
type APIError struct {
    Code    string `json:"code"`              // stable machine token
    Message string `json:"message"`           // human-readable, safe to surface
    Details any    `json:"details,omitempty"` // structured detail (e.g. PolicyCompileError)
}

// PolicyCompileError is the structured detail for 422 policy compile failures.
type PolicyCompileError struct {
    Stage  string `json:"stage"`  // "parse" | "resolve_groups" | "resolve_hosts" | "validate"
    Field  string `json:"field"`  // JSON path into spec, e.g. "acls[1].dst[0]"
    Reason string `json:"reason"` // unknown_group | unknown_host | cidr_not_advertised | ...
    Detail string `json:"detail"` // human elaboration
}
```

### 9.2 Error code table

| Code | REST HTTP | Connect Code | When |
|---|---|---|---|
| `missing_credentials` | 401 | Unauthenticated | No session/api-token/cert |
| `invalid_token` | 401 | Unauthenticated | Bad/expired/revoked API token |
| `invalid_session` | 401 | Unauthenticated | Bad/expired/revoked session |
| `insufficient_role` | 403 | PermissionDenied | RBAC gate fails |
| `device_revoked` | 403 | PermissionDenied | Revoked/expired device cert |
| `enroll_token_invalid` | 403 | PermissionDenied | Token missing/expired/consumed |
| `not_found` | 404 | NotFound | Resource absent in tenant (RLS-filtered) |
| `validation_failed` | 400 | InvalidArgument | Binding/protovalidate failure |
| `policy_compile_failed` | 422 | InvalidArgument | Dry-run compile rejected (details=PolicyCompileError) |
| `prefix_conflict` | 409 | AlreadyExists | Overlapping advertised CIDR |
| `rate_limited` | 429 | ResourceExhausted | Token bucket exhausted |
| `internal` | 500 | Internal | Unexpected (panic-recovered, audited, NO stack to client) |

### 9.3 Gin middleware chain

```go
func newGin(d Deps) *gin.Engine {
    r := gin.New() // gin.New(), never gin.Default() in prod
    r.Use(requestID())          // X-Request-Id
    r.Use(structuredLogger())   // route + status + latency + tenant; NO client IP (C3)
    r.Use(recovery())           // panic -> 500 + audit, no stack leak
    r.Use(securityHeaders())    // HSTS, X-Content-Type-Options, no-store on auth routes
    r.Use(limitBody(256 << 10)) // 256 KiB global body cap
    return r
}
```

---

## 10. Rate limiting

Redis-backed token buckets, keyed by `(tenant_id, principal_id, bucket_class)`, fleet-wide
correct across multi-replica deployments. Fail-open on Redis unavailability (C2: losing Redis
loses no durable state; the API stays reachable).

| Bucket class | Scope | Capacity | Refill | Rationale |
|---|---|---|---|---|
| `enroll_token_mint` | per (tenant, principal) | 20 | 1 / 3 s | Prevents admin-session abuse or buggy automation |
| `enroll_consume` | per source (ephemeral, not logged -- C3) | 10 | 1 / 6 s | Anti-brute-force on enroll tokens |
| `api_token_mint` | per (tenant, admin) | 5 | 1 / 60 s | Token minting is rare and high-privilege |
| `policy_write` | per tenant | 30 | 1 / 2 s | Protects compiler + fan-out from policy-thrash |
| `rest_general` | per (tenant, principal) | 300 | 5 / s | Generous default for Console polling |
| `connect_unary` | per device | 60 | 1 / s | ~20x headroom above normal keepalive cadence |
| `stream_open` | per device | 10 | 1 / 10 s | Bounds reconnect-storm |

**Numbers are design defaults, not yet load-tuned.** Exact values are operator-tunable via
`Config` and MUST NOT be hardcoded as un-overridable constants (§11.4.6 honesty).

```go
// internal/api/ratelimit.go

func rateLimited(class BucketClass, keyFn func(*gin.Context) string) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := fmt.Sprintf("ratelimit:%s:%s", class, keyFn(c))
        allowed, retryAfter, err := redisTokenBucket(c, key, class.Capacity(), class.RefillRate())
        if err != nil {
            // Redis down: fail OPEN for availability (C2)
            metrics.RateLimiterDegraded.Inc()
            c.Next()
            return
        }
        if !allowed {
            c.Header("Retry-After", strconv.Itoa(retryAfter))
            abort(c, http.StatusTooManyRequests, "rate_limited", nil)
            return
        }
        c.Next()
    }
}
```

---

## 11. Metrics

Prometheus metrics exposed at `GET /metrics` (scrape-only network, never public).

```
helix_http_requests_total{route,method,status}       # REST counter
helix_http_request_seconds{route,method}             # REST latency histogram
helix_rpc_seconds{procedure,code}                    # Connect unary/stream histogram
helix_ws_subscribers{tenant}                         # open /v1/stream subscriptions
helix_ws_dropped_total{reason="slow_consumer"}       # backpressure drops
helix_watch_streams_open{tenant}                     # open WatchNetworkMap streams
helix_reconcile_seconds                              # event->Send histogram (the SLO)
helix_rate_limiter_degraded_total                    # fail-open on Redis-down
```

---

## 12. File layout in helix-go

```
helix-go/internal/api/
  server.go          -- Server, Deps, Run(), handler() mux wiring
  rest.go            -- newGin() engine + middleware chain
  routes.go          -- registerREST() route registration with RBAC
  audit_routes.go    -- assertEveryV1RouteGuarded startup audit
  authn.go           -- authn() handler: OIDC session / API token resolution
  rbac.go            -- requireRole() middleware
  connect.go         -- newConnectMux(), deviceAuthInterceptor, coordinatorServer
  stream.go          -- WS/SSE hub, subscriber, projector
  ratelimit.go       -- Redis-backed rate-limited() middleware
  handlers_enroll_tokens.go
  handlers_devices.go
  handlers_connectors.go
  handlers_groups.go
  handlers_policies.go
  handlers_networks.go
  handlers_audit.go
  handlers_api_tokens.go
  handlers_stream.go
  handlers_me.go
  handlers_ops.go     -- /healthz, /readyz, /metrics
  errors.go           -- APIError, writeError(), abort()
  iface.go            -- exported interfaces (Handlers, Server)
```

---

## 13. Phase-2 forward seams

The api layer's design permits Phase-2 growth without reshaping:

- **WASM Console** calling `Coordinator` over Connect/HTTP-1.1 (already supported).
- **`/v1/stream` `Last-Event-ID` replay** becomes a NATS JetStream-backed durable
  subscription when the `events.Bus` swaps from Redis Streams.
- **GitOps policy endpoints** for policy-as-code (additive REST routes).
- **PQ-PSK TLS layer** at the TLS config level -- no handler change.
- **Stateless multi-replica api pods** behind an LB -- no api handler holds durable
  state; the graph/streams live in `coordinator`, rebuilt from Postgres + events.

---

## Sources verified

- [svc-api.md] `v03-control-plane/svc-api.md` -- the full api service nano-detail spec:
  listener topology, mux wiring, full REST route table with schemas, authn/rbac/RLS chain,
  Connect handler interceptor chain, WS/SSE hub, error taxonomy, rate limiting, metrics,
  test points, forward seams. Rev 2 (2026-07-04). (Read 2026-07-08.)
- [protobuf-spec.md] `v03-control-plane/protobuf-spec.md` -- canonical `Coordinator`
  `.proto` (package `helix.coordinator.v1`), field-number registry, Connect transport
  binding, `WatchNetworkMap` state machine, version negotiation, versioning policy.
  Rev 2 (2026-07-04). (Read 2026-07-08.)
- [02-control-plane.md] `02-control-plane.md` -- module architecture (modular monolith,
  wiring rules, inter-module interfaces), data model DDL + RLS, IPAM, coordinator
  graph, policy compiler, event bus, SLOs. Rev 2 (2026-07-04). (Read 2026-07-08.)
- [identity/DESIGN.md] `docs/design/identity/DESIGN.md` -- identity model (managed OIDC +
  anonymous enroll tokens), Enroll RPC wire contract, certificate management, revocation
  flow, SLOs. (Read 2026-07-08.)
- [policies_spec/DESIGN.md] `docs/design/policies_spec/DESIGN.md` -- policies.spec JSONB
  schema, Go types, parser, resolver, compiler, compiled output shape. (Read 2026-07-08.)
- [04-security-privacy-pki.md] `04-security-privacy-pki.md` -- trust model, identity,
  PKI, edge hardening, audit, PQ handshake. (Read 2026-07-08.)
- Connect (buf) -- connectrpc.com/connect: one service handler, three protocols
  (gRPC, gRPC-Web, Connect), server-streaming, trailer-based error codes.
  (Read 2026-07-08.)
- Gin -- gin-gonic/gin v1.12.0: HTTP web framework, middleware chains, route groups.
  (Read 2026-07-08.)
- sqlc + pgx -- sqlc.dev (compile-checked SQL), github.com/jackc/pgx/v5 (Postgres driver).
  (Read 2026-07-08.)
- Tailscale ACL policy syntax -- tailscale.com/docs/reference/syntax/policy-file
  (reference for groups, hosts, tags, ACLs, tests, autogroups). (Read 2026-07-08.)
- OIDC Device Authorization Grant RFC 8628 + DPoP draft-parecki-oauth-dpop-device-flow
  (September 2025 -- DPoP binding for device flow). (Read 2026-07-08.)
- Mullvad anonymous enrollment -- 16-digit account number, zero PII, cash/Monero payments.
  The Verge (2023-04-21) confirmed zero customer data in a 2023 Swedish police raid.
  (Read 2026-07-08.)
- Constitution: §11.4.6 (no-guessing -- UNVERIFIED marks, latency stated as targets),
  §11.4.8 (deep web research), §11.4.10 (credentials never logged/git-tracked),
  §11.4.17 (universal-vs-project classification), §11.4.35 (canonical-root inheritance),
  §11.4.44 (revision header), §11.4.99 (latest-source verification),
  §11.4.108 (runtime-signature), §11.4.110 (build-readiness verdict).

Date verified: 2026-07-08
