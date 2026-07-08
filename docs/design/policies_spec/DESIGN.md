# policies.spec -- Parser + Group/Host Resolution Design (HVPN-P1-060 / CP-T6.1)

**Revision:** 1
**Last modified:** 2026-07-08T00:00:00Z
**Description:** Concrete design for parsing the `policies.spec` JSONB document,
resolving groups/hosts/ACLs, and compiling into per-node WireGuard peer maps.
**Authority:** design-authority
**Scope:** documents the Go parser, resolver, and compiler for `helix-go/internal/policy/`

---

## 1. Overview

The `policies.spec` JSONB document (stored in `policies.spec` column of the
`policies` table per [02-control-plane §2.2]) is the declarative access-control
policy for a tenant. This document specifies:

- The **complete JSON schema** with field-by-field type definitions, validation
  rules, and examples
- The **Go type definitions** that parse and validate the spec
- The **resolution algorithm** that expands symbolic names (groups, hosts,
  tags) into concrete device-IDs and CIDRs
- The **compilation algorithm** that produces per-node `NetworkMap`-compatible
  peer lists with WireGuard `AllowedIPs`
- The **test specification** that auto-verifies policy correctness

### 1.1 Position in the system

```
policies.spec (JSONB, user-authored)
        |
        v
  [Parser]        -- parse + validate (this design)
        |
        v
  [Resolver]      -- expand groups/hosts/tags -> device-IDs + CIDRs
        |
        v
  [Compiler]      -- produce CompiledPolicy { AllowedIPs, Verdicts, ExitNodes }
        |
        v
  [Coordinator]   -- per-node buildMap() -> NetworkMap -> WatchNetworkMap stream
        |
        v
  [Rust edge]     -- map.rs reconcile -> WireGuard peer add/remove
```

### 1.2 Governing constraints

| # | Invariant | Source |
|---|---|---|
| P1 | **Default-deny, fail-closed.** A peer is visible only if an explicit rule grants it. | [02 §0.1 C4] |
| P2 | **Pure, deterministic.** Same spec + same DB state => byte-identical `CompiledPolicy`. | [02 §7.2] |
| P3 | **Fail-closed validation.** Unknown group/host/tag, uncovered CIDR, revoked device -> REJECT. | [02 §7.3] |
| P4 | **Two output artifacts.** Coarse WG `AllowedIPs` (CIDR-only) + fine port-level verdict map (nftables/eBPF). | [02 §7.2] |
| P5 | **Parser is Go, not Rust.** The policy compiler lives in `helix-go/internal/policy/`; the Rust edge only consumes the compiled output. | [02 §1] |

---

## 2. JSONB Schema Definition

### 2.1 Top-level structure

```jsonc
{
  "groups":    { ... },       // §2.2 — named user/device collections
  "hosts":     { ... },       // §2.3 — IP/CIDR aliases
  "tagOwners": { ... },       // §2.4 — who may apply tags
  "acls":      [ ... ],       // §2.5 — access control rules
  "exitNodes": [ ... ],       // §2.6 — exit-node declarations
  "tests":     [ ... ]        // §2.7 — policy verification tests
}
```

All top-level keys are **optional** with these defaults:
- `groups` absent => `{}` (no named groups)
- `hosts` absent => `{}` (no host aliases)
- `tagOwners` absent => `{}` (only the empty set may apply tags)
- `acls` absent/empty => **deny all** (default-deny, P1)
- `exitNodes` absent => `[]` (no exit nodes)
- `tests` absent => `[]` (no tests)

### 2.2 `groups` -- named collections

```jsonc
"groups": {
  "group:admins":      ["alice@example.com", "bob@example.com"],
  "group:engineering": ["charlie@example.com"],
  "group:contractors": ["dave@external.org"]
}
```

**Type:** `map[string][]string` (JSON object of string arrays)

**Validation rules:**
- Every key MUST start with the literal prefix `group:` (reject otherwise)
- Group names are case-sensitive; `"group:Admins"` != `"group:admins"`
- Members are user identifiers (email or OIDC sub, matching `users.email` or `users.oidc_sub`)
- **Nested groups are NOT supported** (no `"group:outer": ["group:inner"]`) -- this simplifies
  resolution and matches the Tailscale model
- Empty member lists are allowed (a group with zero members resolves to an empty set)
- Duplicate members within a group are tolerated (deduplicated at resolve time)

**Error codes:**
| Code | When |
|---|---|
| `INVALID_GROUP_FORMAT` | Key does not start with `group:` |
| `NESTED_GROUPS` | A member value starts with `group:` |
| `DUPLICATE_GROUP` | Same group name appears twice |

### 2.3 `hosts` -- IP/CIDR aliases

```jsonc
"hosts": {
  "warehouse-cams":   "10.10.0.0/24",
  "office-lan":       "192.168.50.0/24",
  "db-primary":       "10.20.0.2",
  "internal-vpc":     "172.16.0.0/12"
}
```

**Type:** `map[string]string` (JSON object of string values)

**Validation rules:**
- Every value MUST be a valid IPv4 or IPv6 address, optionally with CIDR prefix length
- A bare IP (no `/NN`) is treated as a `/32` (IPv4) or `/128` (IPv6) host route
- Values are parsed via Go `netip.ParsePrefix`; `netip.Addr.IsPrivate()` is NOT required
  (a host alias may point to a public IP if that is the advertised prefix)
- Host names are case-sensitive and MUST be unique
- Host names MUST NOT start with `group:`, `tag:`, or `autogroup:`

**Error codes:**
| Code | When |
|---|---|
| `INVALID_HOST_IP` | Value is not a valid IP or CIDR |
| `DUPLICATE_HOST` | Same host name appears twice |
| `RESERVED_HOST_PREFIX` | Host name starts with a reserved prefix |

### 2.4 `tagOwners` -- tag authorization

```jsonc
"tagOwners": {
  "tag:prod-web":       ["group:admins"],
  "tag:prod-db":        ["group:admins"],
  "tag:ci-runner":      ["group:engineering"],
  "tag:monitoring":     ["group:admins", "alice@example.com"]
}
```

**Type:** `map[string][]string` (JSON object of string arrays)

**Validation rules:**
- Every key MUST start with the literal prefix `tag:`
- Tag names are case-sensitive
- Tag owners are user identifiers OR group references (`group:...`)
- Tag owner references MUST resolve to known users or groups at validation time
- Empty owner lists are rejected (a tag with no owners cannot be applied)

**Error codes:**
| Code | When |
|---|---|
| `INVALID_TAG_FORMAT` | Key does not start with `tag:` |
| `TAG_NO_OWNERS` | A tag has an empty owner list |
| `UNKNOWN_TAG_OWNER` | An owner reference does not resolve to a known user/group |
| `CIRCULAR_TAG_OWNER` | A tag owner chain references itself (group->tag->group cycle) |

### 2.5 `acls` -- access control rules

```jsonc
"acls": [
  {
    "action": "accept",
    "src":    ["group:admins"],
    "proto":  "tcp",
    "dst":    ["tag:prod-web:80,443", "tag:prod-db:5432"]
  },
  {
    "action": "accept",
    "src":    ["group:engineering"],
    "dst":    ["tag:ci-runner:*"]
  },
  {
    "action": "accept",
    "src":    ["group:contractors"],
    "dst":    ["warehouse-cams:554,80"]
  },
  {
    "action": "accept",
    "src":    ["*"],
    "dst":    ["office-lan:*"]
  }
]
```

**Type:** array of ACL rule objects

#### ACL rule fields

| Field | Required | Type | Description |
|---|---|---|---|
| `action` | Yes | `"accept"` | Only `"accept"` is valid; deny is implicit (default-deny, P1) |
| `src` | Yes | `[]string` | Source selectors (see §2.5.1) |
| `dst` | Yes | `[]string` | Destination selectors with optional ports (see §2.5.2) |
| `proto` | No | `string` | Protocol: `"tcp"`, `"udp"`, `"icmp"`, `"icmp4"`, `"icmp6"`, `"sctp"`, or IANA protocol number as string. Omitted => any protocol |

#### 2.5.1 Source selectors (`src`)

Each `src` entry is one of:

| Format | Example | Resolves to |
|---|---|---|
| `"*"` | `"*"` | All devices in the tenant |
| `"user@domain"` | `"alice@example.com"` | All devices owned by that user |
| `"group:name"` | `"group:admins"` | All devices owned by group members |
| `"tag:name"` | `"tag:prod-web"` | All devices with that tag |
| `"hostname"` | `"office-lan"` | The CIDR from the `hosts` map |
| `"ip/cidr"` | `"10.0.0.0/8"` | The literal IP range |
| `"autogroup:member"` | `"autogroup:member"` | All non-tagged (user-owned) devices |

#### 2.5.2 Destination selectors (`dst`)

Each `dst` entry is `<selector>:<ports>` where:

**Selector** (left of colon): same as source selectors (§2.5.1) PLUS `"*"` for any destination.

**Ports** (right of colon): one of:
- `"*"` -- all ports
- `"80"` -- single port
- `"80,443"` -- comma-separated list
- `"8000-9000"` -- inclusive range
- Combination: `"80,443,8000-9000"`

**Validation rules:**
- Port values must be 1-65535
- Port ranges must have `start <= end`
- `proto` + port compatibility: ICMP with ports is a warning (ICMP has no ports)
- `action` must be exactly `"accept"` (case-sensitive)

**ACL ordering:** Rules are evaluated in **array order, first-match-wins**.
The first rule whose `src` matches the source device and whose `dst` matches
the destination wins; subsequent rules for the same (src, dst) pair are ignored.

**Error codes:**
| Code | When |
|---|---|
| `INVALID_ACL_ACTION` | `action` is not `"accept"` |
| `INVALID_SRC_SELECTOR` | Source selector cannot be parsed |
| `INVALID_DST_FORMAT` | Destination missing port or has invalid port syntax |
| `INVALID_PORT` | Port value out of range 1-65535 |
| `INVALID_PORT_RANGE` | Range start > end |
| `UNKNOWN_GROUP` | A group name in src/dst is not in `groups` |
| `UNKNOWN_HOST` | A host name in src/dst is not in `hosts` |
| `UNKNOWN_TAG` | A tag name in src/dst is not in `tagOwners` |
| `UNCOVERED_DST_CIDR` | A `dst` CIDR is not advertised by any connector |
| `REVOKED_IN_RULE` | A rule grants access to a revoked device |
| `PROTO_PORT_MISMATCH` | ICMP used with specific ports (warning, not error) |

### 2.6 `exitNodes` -- exit-node declarations

```jsonc
"exitNodes": [
  "group:admins",
  "tag:exit-us-east"
]
```

**Type:** `[]string` -- array of selectors (same format as ACL `src`)

**Semantics:** The resolved set of devices may route **all** internet-destined
traffic through the gateway as a full-tunnel exit. Connectors and revoked
devices are excluded from the exit-node set even if a selector would include them.

**Validation rules:**
- Each entry must be a valid selector (§2.5.1)
- Selectors resolving to connectors are rejected (connectors are not exits)
- Selectors resolving to revoked devices are silently excluded (not an error --
  they simply don't become exits)

**Error codes:**
| Code | When |
|---|---|
| `EXIT_NODE_IS_CONNECTOR` | Selector resolves to a connector device |
| `UNKNOWN_EXIT_SELECTOR` | Selector cannot be resolved |

### 2.7 `tests` -- policy verification tests

```jsonc
"tests": [
  {
    "src":    "alice@example.com",
    "dst":    "tag:prod-web:443",
    "proto":  "tcp",
    "accept": true
  },
  {
    "src":    "carol@external.org",
    "dst":    "tag:prod-db:5432",
    "proto":  "tcp",
    "deny":   true
  }
]
```

**Type:** array of test case objects

| Field | Required | Type | Description |
|---|---|---|---|
| `src` | Yes | `string` | Single source selector (user, group, tag) |
| `dst` | Yes | `string` | Single destination with a SINGLE port (no ranges/lists) |
| `proto` | No | `string` | Protocol (default: `"tcp"`) |
| `accept` | One of | `bool` | Assert the traffic SHOULD be allowed |
| `deny` | One of | `bool` | Assert the traffic SHOULD be denied |

Exactly one of `accept` or `deny` must be `true`.

**Test execution:** After compilation, each test is evaluated against the
compiled policy. A test that asserts `accept: true` and finds no matching rule
is a FAIL. A test that asserts `deny: true` and finds a matching rule is a FAIL.
All tests must pass for the policy to be considered valid.

**Error codes:**
| Code | When |
|---|---|
| `TEST_NEEDS_ACCEPT_OR_DENY` | Neither accept nor deny is true |
| `TEST_BOTH_ACCEPT_AND_DENY` | Both accept and deny are true |
| `TEST_INVALID_DST_PORT` | Destination has a port range or port list |
| `TEST_FAILED_ACCEPT` | `accept: true` but traffic is denied |
| `TEST_FAILED_DENY` | `deny: true` but traffic is allowed |

---

## 3. Go Type Definitions

### 3.1 Top-level types

```go
// internal/policy/spec.go

package policy

import (
    "net/netip"
    "encoding/json"
)

// Spec is the parsed and validated policies.spec document.
type Spec struct {
    Groups    GroupMap    `json:"groups,omitempty"`
    Hosts     HostMap     `json:"hosts,omitempty"`
    TagOwners TagOwnerMap `json:"tagOwners,omitempty"`
    ACLs      []ACLRule   `json:"acls,omitempty"`
    ExitNodes []Selector  `json:"exitNodes,omitempty"`
    Tests     []TestCase  `json:"tests,omitempty"`
}

// GroupMap maps group names to their member user identifiers.
// Keys MUST start with "group:". Values are email or OIDC-sub strings.
type GroupMap map[GroupName][]UserID

// HostMap maps host aliases to IP prefixes.
// Values are validated via netip.ParsePrefix; bare IPs get /32 or /128.
type HostMap map[HostName]netip.Prefix

// TagOwnerMap maps tag names to their authorized owner selectors.
// Keys MUST start with "tag:".
type TagOwnerMap map[TagName][]OwnerSelector
```

### 3.2 Strongly-typed string aliases

```go
// GroupName is a validated group identifier, e.g. "group:admins".
// The "group:" prefix is guaranteed present.
type GroupName string

// UserID is a user identifier -- an email or OIDC sub.
type UserID string

// HostName is a validated host alias, e.g. "warehouse-cams".
type HostName string

// TagName is a validated tag identifier, e.g. "tag:prod-web".
type TagName string

// Selector is a raw string from the policy document -- a user, group,
// tag, host, CIDR, wildcard, or autogroup reference.
type Selector string

// OwnerSelector is a selector for tag ownership -- user or group reference.
type OwnerSelector string

// Protocol is a network protocol: "tcp", "udp", "icmp", "icmp4", "icmp6",
// "sctp", or an IANA protocol number as a string (e.g. "6" for TCP).
type Protocol string

// Port is a single port number, 1-65535.
type Port uint16

// PortRange is an inclusive port range start-end.
type PortRange struct {
    Start Port
    End   Port
}

// PortSet is a set of ports specified in ACL destination syntax.
// It can be a single port, a comma-separated list, a range, or "*".
type PortSet struct {
    All    bool        // true for "*" (all ports)
    Ports  []Port      // individual ports (from "80,443")
    Ranges []PortRange // ranges (from "8000-9000")
}
```

### 3.3 ACL types

```go
// ACLRule is a single access-control rule.
type ACLRule struct {
    Action    ACLAction  `json:"action"`
    Sources   []Selector `json:"src"`
    Dests     []DestRef  `json:"dst"`
    Protocol  Protocol   `json:"proto,omitempty"`
}

// ACLAction is the action for an ACL rule. Only "accept" is valid;
// deny is implicit (default-deny).
type ACLAction string

const ActionAccept ACLAction = "accept"

// DestRef is a parsed destination reference: selector + port set.
// Parsed from the "selector:ports" syntax in the JSON.
type DestRef struct {
    Selector Selector
    Ports    PortSet
}
```

### 3.4 Test types

```go
// TestCase is a single policy verification test.
type TestCase struct {
    Source  Selector `json:"src"`
    Dest    DestRef  `json:"dst"`
    Proto   Protocol `json:"proto,omitempty"`
    Accept  bool     `json:"accept,omitempty"`
    Deny    bool     `json:"deny,omitempty"`
}
```

### 3.5 Resolution types (intermediate representations)

```go
// ResolvedSelector is a Selector expanded to a concrete set of device IDs
// and/or IP prefixes. This is the output of the resolution phase.
type ResolvedSelector struct {
    DeviceIDs []uuid.UUID      // concrete device UUIDs
    Prefixes  []netip.Prefix   // CIDR ranges
    IsWildcard bool             // true if the selector was "*"
}

// ResolvedRule is an ACL rule after resolution -- all selectors expanded.
type ResolvedRule struct {
    SrcDevices  []uuid.UUID
    SrcPrefixes []netip.Prefix
    DstDevices  []uuid.UUID
    DstPrefixes []netip.Prefix
    Protocol    Protocol
    Ports       PortSet
    Index       int        // original position in the ACL array (for first-match-wins)
}

// ExitNodeSet is the resolved set of devices permitted as exit nodes.
type ExitNodeSet struct {
    DeviceIDs []uuid.UUID
}
```

### 3.6 Compiled output types (compatible with map.rs / protobuf)

```go
// CompiledPolicy is the output of the policy compiler.
// This is what coordinator.buildMap() consumes.
type CompiledPolicy struct {
    Version    int64
    VisibleTo  map[uuid.UUID][]uuid.UUID           // device -> visible peers (need-to-know)
    AllowedIPs map[uuid.UUID]map[uuid.UUID][]netip.Prefix // device -> peer -> allowed CIDRs
    Verdicts   map[uuid.UUID]map[uuid.UUID][]PortRule     // device -> peer -> port-level rules
    ExitNodes  []uuid.UUID                         // exit-node device IDs
}

// PortRule is a single port-level allow entry for nftables/eBPF edge enforcement.
type PortRule struct {
    Protocol Protocol
    Ports    PortSet
}

// NetworkMapPeer is the per-peer entry the coordinator emits into
// the NetworkMap.peers array (protobuf: Peer message).
type NetworkMapPeer struct {
    DeviceID   uuid.UUID
    WgPubkey   []byte          // 32-byte Curve25519 public key
    AllowedIPs []netip.Prefix  // compiled CIDRs for WG AllowedIPs
    PortRules  []PortRule      // fine-grained port verdicts (edge enforces)
    IsExitNode bool            // true if this peer is an exit node
}
```

---

## 4. Parser Implementation

### 4.1 Parsing pipeline

```
JSON bytes
   |
   v
[unmarshalSpec]     -- json.Unmarshal into Spec struct; basic type checks
   |
   v
[validateSpec]      -- semantic validation (groups exist, ports valid, etc.)
   |
   v
[normalizeSpec]     -- apply defaults (bare IP -> /32, empty proto -> any)
   |
   v
validated Spec
```

### 4.2 Parser function signatures

```go
// internal/policy/parser.go

// ParseSpec parses a policies.spec JSON document and returns a validated Spec.
// It performs all validation synchronously -- a returned Spec is guaranteed
// structurally and semantically valid.
func ParseSpec(jsonBytes []byte) (*Spec, error)

// ParseSpecDryRun is like ParseSpec but also runs the resolver and compiler
// to catch resolution-time errors (unknown hosts, uncovered CIDRs) before
// the policy is persisted.
func ParseSpecDryRun(jsonBytes []byte, resolver *Resolver) (*Spec, error)
```

### 4.3 Validation function

```go
// internal/policy/validate.go

// ValidationError is a structured error with a machine-readable code
// and human-readable message.
type ValidationError struct {
    Code    string `json:"code"`    // e.g. "INVALID_GROUP_FORMAT"
    Field   string `json:"field"`   // JSON path, e.g. "groups.group:admins[0]"
    Message string `json:"message"` // human-readable description
    Value   string `json:"value"`   // the offending value (may be empty)
}

// ValidationErrors is a list of ValidationError, implementing the error interface.
type ValidationErrors []ValidationError

func (ve ValidationErrors) Error() string { ... }

// Validate performs all validation passes and returns nil or ValidationErrors.
func (s *Spec) Validate(tx TenantContext) error
```

### 4.4 Validation passes (ordered)

1. **Structural pass** -- JSON schema conformance (right types, no missing fields)
2. **Group validation** -- keys start with `group:`, no nested groups, no empty names
3. **Host validation** -- values parse as IP/prefix, no reserved-prefix names
4. **TagOwner validation** -- keys start with `tag:`, owners reference known users/groups
5. **ACL validation** -- action is `"accept"`, sources/dests parse, ports in range
6. **Cross-reference validation** -- groups/hosts/tags referenced in ACLs exist
7. **ExitNode validation** -- no connectors in the resolved set
8. **Test validation** -- exactly one of accept/deny, single-port dests

### 4.5 Custom JSON unmarshaling

```go
// DestRef has custom unmarshaling that parses the "selector:ports" format.
func (d *DestRef) UnmarshalJSON(b []byte) error {
    var raw string
    if err := json.Unmarshal(b, &raw); err != nil {
        return err
    }
    return d.parse(raw) // splits on last ':', parses ports
}

// PortSet has custom unmarshaling that parses "80,443,8000-9000" or "*".
func (ps *PortSet) UnmarshalJSON(b []byte) error { ... }

// HostMap has custom unmarshaling that appends /32 to bare IPv4 addresses
// and /128 to bare IPv6 addresses.
func (hm *HostMap) UnmarshalJSON(b []byte) error { ... }
```

---

## 5. Group Resolution

### 5.1 Algorithm

```
resolveGroup(name: "group:admins", spec, DB):
  1. Look up "group:admins" in spec.Groups
     -> ["alice@example.com", "bob@example.com"]
  2. For each member:
     a. If member starts with "group:" -> ERROR (nested groups forbidden)
     b. Look up user in DB users table by email
        -> user.ID = uuid
     c. Find all non-revoked devices owned by that user
        -> [deviceA, deviceB]
  3. Return union of all device IDs
```

### 5.2 Resolution functions

```go
// internal/policy/resolve.go

// Resolver resolves symbolic selectors to concrete device IDs and prefixes.
// It requires a database handle for user and device lookups.
type Resolver struct {
    db  *db.Queries  // sqlc-generated, tenant-scoped
}

// ResolveSelector resolves a single Selector to a ResolvedSelector.
func (r *Resolver) ResolveSelector(ctx context.Context, s Selector) (*ResolvedSelector, error)

// ResolveGroup resolves a group name to its member device IDs.
func (r *Resolver) ResolveGroup(ctx context.Context, g GroupName) ([]uuid.UUID, error)

// ResolveHost resolves a host alias to its CIDR prefix.
func (r *Resolver) ResolveHost(h HostName) (netip.Prefix, error)

// ResolveTag resolves a tag to the device IDs of nodes with that tag.
func (r *Resolver) ResolveTag(ctx context.Context, t TagName) ([]uuid.UUID, error)

// ResolveUser resolves a user identifier to their device IDs.
func (r *Resolver) ResolveUser(ctx context.Context, u UserID) ([]uuid.UUID, error)

// ResolveAutogroup resolves an autogroup: selector.
func (r *Resolver) ResolveAutogroup(ctx context.Context, a string) ([]uuid.UUID, error)
```

### 5.3 Selector dispatch table

| Selector prefix/pattern | Dispatch function | DB query |
|---|---|---|
| `"*"` | `allNonRevokedDevices()` | `SELECT id FROM devices WHERE tenant_id=$1 AND revoked_at IS NULL` |
| `"group:"` | `ResolveGroup()` | `SELECT d.id FROM devices d JOIN users u ON d.user_id=u.id WHERE u.email = ANY($1)` |
| `"tag:"` | `ResolveTag()` | Look up in tagOwnerMap (NodeView.HasTag) |
| `"autogroup:member"` | `ResolveAutogroup()` | `SELECT id FROM devices WHERE user_id IS NOT NULL AND revoked_at IS NULL` |
| `"autogroup:self"` | per-node (special) | The calling device's own devices |
| contains `@` | `ResolveUser()` | `SELECT d.id FROM devices d JOIN users u ON d.user_id=u.id WHERE u.email=$1` |
| contains `/` | `netip.ParsePrefix` | No DB -- literal IP range |
| matches host name in `spec.Hosts` | `ResolveHost()` | No DB -- literal prefix from HostMap |
| otherwise | error `UNKNOWN_SELECTOR` | -- |

### 5.4 Ambiguity resolution

When a selector could match multiple types (e.g. `"alice"` could be a host or
a user), the dispatch order is:

1. Literal `"*"` (wildcard)
2. Prefix `"group:"` (group)
3. Prefix `"tag:"` (tag)
4. Prefix `"autogroup:"` (autogroup)
5. Contains `@` (user email)
6. Contains `/` (CIDR)
7. Exact match in `spec.Hosts` (host alias)
8. Parse as bare IP (netip.ParseAddr)

---

## 6. Host Resolution

### 6.1 Algorithm

```
resolveHost(name: "warehouse-cams", spec):
  1. Look up "warehouse-cams" in spec.Hosts
     -> "10.10.0.0/24" (already parsed to netip.Prefix during validation)
  2. Cross-check against advertised_prefixes:
     a. Query DB: SELECT connector_id FROM advertised_prefixes
        WHERE tenant_id=$1 AND cidr >>= $2  (the advertised prefix covers this CIDR)
     b. If one connector covers it -> return prefix + connector device ID
     c. If multiple connectors cover it -> emit ROUTE_CONFLICT warning; all are returned
     d. If NO connector covers it -> VALIDATION ERROR (UNCOVERED_DST_CIDR)
  3. Return { prefix, connectorIDs }
```

### 6.2 Cross-check function

```go
// ResolveHostToConnector finds which connector(s) advertise a prefix
// covering the given host's CIDR.
func (r *Resolver) ResolveHostToConnector(
    ctx context.Context, h HostName,
) (prefix netip.Prefix, connectors []uuid.UUID, warnings []string, err error)
```

### 6.3 Route conflict handling

When two connectors advertise overlapping CIDRs (e.g., ConnectorA advertises
`10.0.0.0/8` and ConnectorB advertises `10.10.0.0/16`), the host reference
`warehouse-cams -> 10.10.0.0/24` resolves to both connectors. This is a
**warning**, not a rejection -- the `4via6` disambiguation (D4, [02 §3.1])
allows the client to reach the correct network through the correct connector.

---

## 7. ACL Compilation

### 7.1 Algorithm

```
compile(spec, resolver) -> CompiledPolicy:
  1. Resolve all ACL rules:
     for each rule in spec.ACLs (in order):
        resolved.src = resolveSelectors(rule.src)
        resolved.dst = resolveSelectors(rule.dst)
        resolved.ports = rule.dst ports
        resolved.proto = rule.proto
        resolved.index = position

  2. Build per-device visibility (need-to-know, C4/P1):
     for each device d in tenant:
        visible[d] = {}
        for each resolved rule r:
           if d is in r.src:
              for each target t in r.dst:
                 visible[d][t] = true
        // Apply first-match-wins within same (src, dst) pair
        // Later rules with same (src, dst) at a more specific level
        // do NOT override earlier rules -- but since action is always
        // "accept" and deny is implicit, first-match-wins means the
        // FIRST accept for a (src, dst) pair wins. Subsequent rules
        // for the same pair are irrelevant.

  3. Build AllowedIPs (coarse, WG-compatible):
     for each device d:
        allowedIPs[d] = {}
        for each peer p in visible[d]:
           allowedIPs[d][p] = union of all CIDRs from rules where
              d in src AND p in dst

  4. Build Verdicts (fine, port-level):
     for each device d:
        verdicts[d] = {}
        for each peer p in visible[d]:
           verdicts[d][p] = [(proto, ports), ...] from rules

  5. Build ExitNodes:
     resolved = resolveSelectors(spec.exitNodes)
     exitNodes = resolved.deviceIDs where device.kind != "connector"

  return { version, visible, allowedIPs, verdicts, exitNodes }
```

### 7.2 First-match-wins semantics

Rules are evaluated in array order. Within a given (source-device, destination)
pair, the FIRST rule whose `src` includes the source and whose `dst` includes
the destination determines the access. Since the only action is `"accept"` and
the default is deny, this means:

- If the first matching rule accepts, access is allowed.
- If no rule matches, access is denied (default-deny, P1).
- Two rules that both accept the same (src, dst) pair at different port/proto
  levels produce a UNION of the allowed ports from the first matching rule only.

### 7.3 Compiler function signatures

```go
// internal/policy/compile.go

// Compile runs the full compilation pipeline and returns a CompiledPolicy.
// The returned policy is deterministic: same spec + same DB state =>
// byte-identical CompiledPolicy (verified by property test).
func Compile(
    ctx context.Context,
    spec *Spec,
    resolver *Resolver,
) (*CompiledPolicy, error)

// CompileAndActivate compiles and marks the policy as active in one call.
// This is the transactional boundary: the compiled policy is persisted
// alongside the spec version bump.
func CompileAndActivate(
    ctx context.Context,
    tenantID uuid.UUID,
    spec *Spec,
    resolver *Resolver,
    store *Store,
) (int64, error)
```

### 7.4 WireGuard Peer Mapping (how policy becomes allowed_ips)

The compiled `AllowedIPs` map is the direct source for the `Peer.allowed_ips`
field in the `NetworkMap` (protobuf) which maps to `map.rs`'s `Peer.allowed_ips`
(Vec<String> of CIDR strings).

```
CompiledPolicy.AllowedIPs[deviceA][peerB] = ["10.10.0.0/24", "192.168.50.0/24"]
                                              |
                                              v
NetworkMap.peers[i].allowed_ips = ["10.10.0.0/24", "192.168.50.0/24"]
                                              |
                                              v
WireGuard Peer { ..., allowed_ips: 10.10.0.0/24, 192.168.50.0/24 }
```

**Key property:** ACL denials become **missing allowed_ips** -- WireGuard
enforces access at the kernel level because a packet from a peer whose source
IP is not in `allowed_ips` is silently dropped by the kernel. No explicit
"deny" rules are needed in WireGuard configuration.

**Port-level enforcement** is handled separately: the per-device `Verdicts`
map is shipped alongside the NetworkMap (in a separate protobuf field or as
metadata) and enforced at the edge via nftables or eBPF. WireGuard itself
has no port awareness -- it only filters by source IP (via `allowed_ips`).

### 7.5 Exit node routing

When a device is in the `ExitNodes` set, its `allowed_ips` in every peer's
configuration includes `0.0.0.0/0` and `::/0` (the default routes). This
tells WireGuard to route ALL traffic through that peer -- implementing the
full-tunnel VPN exit node pattern.

For the HelixVPN gateway specifically:
```json
// In the NetworkMap for a client whose policy permits exit-node use:
{
  "peers": [
    {
      "name": "gw-helix",
      "wg_pubkey": "gwkey123",
      "allowed_ips": ["0.0.0.0/0", "::/0"],
      // ... plus any LAN prefixes from connectors the client may reach
    }
  ]
}
```

---

## 8. Test Execution

### 8.1 Algorithm

```
runTests(spec, compiledPolicy, resolver):
  for each test in spec.Tests:
    srcDevices = resolveSelector(test.src)
    dstDevices, dstPrefixes = resolveDestRef(test.dst)
    matchingRule = findFirstMatchingRule(
        compiledPolicy, srcDevices, dstDevices, dstPrefixes,
        test.dst.Ports, test.proto)

    if test.accept:
        if matchingRule == nil:  FAIL("expected accept, got deny")
        else:                    PASS
    if test.deny:
        if matchingRule == nil:  PASS
        else:                    FAIL("expected deny, got accept")
```

### 8.2 Test runner function

```go
// internal/policy/test.go

// TestResult is the outcome of a single policy test case.
type TestResult struct {
    Index   int      // position in the tests array
    Passed  bool
    Want    string   // "accept" or "deny"
    Got     string   // "accept" or "deny"
    Message string   // human-readable explanation (on failure)
}

// RunTests evaluates all spec.Tests against the compiled policy.
// Returns results in the same order as the tests array.
func RunTests(spec *Spec, compiled *CompiledPolicy, resolver *Resolver) []TestResult

// RunTestsOrReject runs tests and returns an error if any test fails.
// This is called during ParseSpecDryRun to fail-closed on test failure.
func RunTestsOrReject(spec *Spec, compiled *CompiledPolicy, resolver *Resolver) error
```

---

## 9. Error Model

### 9.1 Error types

```go
// PolicyError is the top-level error type for all policy operations.
type PolicyError struct {
    Op      string           // "parse", "validate", "resolve", "compile", "test"
    Errors  ValidationErrors // may be empty if the error is not validation-related
    Err     error            // wrapped underlying error
}

func (e *PolicyError) Error() string { ... }
func (e *PolicyError) Unwrap() error { ... }

// IsValidationError checks whether an error is a validation error
// with a specific error code.
func IsValidationError(err error, code string) bool { ... }
```

### 9.2 Error code catalog

| Code | Category | Severity |
|---|---|---|
| `INVALID_JSON` | parse | ERROR |
| `INVALID_GROUP_FORMAT` | validate | ERROR |
| `NESTED_GROUPS` | validate | ERROR |
| `DUPLICATE_GROUP` | validate | ERROR |
| `INVALID_HOST_IP` | validate | ERROR |
| `DUPLICATE_HOST` | validate | ERROR |
| `RESERVED_HOST_PREFIX` | validate | ERROR |
| `INVALID_TAG_FORMAT` | validate | ERROR |
| `TAG_NO_OWNERS` | validate | ERROR |
| `UNKNOWN_TAG_OWNER` | validate | ERROR |
| `CIRCULAR_TAG_OWNER` | validate | ERROR |
| `INVALID_ACL_ACTION` | validate | ERROR |
| `INVALID_SRC_SELECTOR` | validate | ERROR |
| `INVALID_DST_FORMAT` | validate | ERROR |
| `INVALID_PORT` | validate | ERROR |
| `INVALID_PORT_RANGE` | validate | ERROR |
| `UNKNOWN_GROUP` | resolve | ERROR |
| `UNKNOWN_HOST` | resolve | ERROR |
| `UNKNOWN_TAG` | resolve | ERROR |
| `UNCOVERED_DST_CIDR` | resolve | ERROR |
| `REVOKED_IN_RULE` | resolve | WARN |
| `PROTO_PORT_MISMATCH` | validate | WARN |
| `EXIT_NODE_IS_CONNECTOR` | resolve | ERROR |
| `UNKNOWN_EXIT_SELECTOR` | resolve | ERROR |
| `ROUTE_CONFLICT` | resolve | WARN |
| `TEST_NEEDS_ACCEPT_OR_DENY` | validate | ERROR |
| `TEST_BOTH_ACCEPT_AND_DENY` | validate | ERROR |
| `TEST_INVALID_DST_PORT` | validate | ERROR |
| `TEST_FAILED_ACCEPT` | test | ERROR |
| `TEST_FAILED_DENY` | test | ERROR |

### 9.3 Error response format (API)

```json
{
  "error": "policy validation failed",
  "code": "VALIDATION_ERROR",
  "details": [
    {
      "code": "INVALID_GROUP_FORMAT",
      "field": "groups.admins",
      "message": "group name must start with 'group:' prefix",
      "value": "admins"
    },
    {
      "code": "UNKNOWN_HOST",
      "field": "acls[2].dst[0]",
      "message": "host 'warehouse-cams' is not defined in the hosts section",
      "value": "warehouse-cams"
    }
  ]
}
```

---

## 10. Test Fixtures

### 10.1 Valid minimal policy

```json
{
  "acls": [
    { "action": "accept", "src": ["*"], "dst": ["*:*"] }
  ]
}
```

### 10.2 Full multi-group policy

```json
{
  "groups": {
    "group:admins":      ["alice@corp.com"],
    "group:engineering": ["bob@corp.com", "charlie@corp.com"],
    "group:contractors": ["dave@external.org"]
  },
  "hosts": {
    "warehouse-cams": "10.10.0.0/24",
    "office-lan":     "192.168.50.0/24",
    "db-primary":     "10.20.0.2"
  },
  "tagOwners": {
    "tag:prod-web":   ["group:admins"],
    "tag:prod-db":    ["group:admins"],
    "tag:ci-runner":  ["group:engineering"]
  },
  "acls": [
    { "action": "accept", "src": ["group:admins"],      "proto": "tcp", "dst": ["*:*"] },
    { "action": "accept", "src": ["group:engineering"], "proto": "tcp", "dst": ["tag:ci-runner:*", "tag:prod-web:80,443"] },
    { "action": "accept", "src": ["group:contractors"], "proto": "tcp", "dst": ["warehouse-cams:554,80"] }
  ],
  "exitNodes": ["group:admins"],
  "tests": [
    { "src": "alice@corp.com",    "dst": "tag:prod-db:5432", "proto": "tcp", "accept": true },
    { "src": "dave@external.org", "dst": "tag:prod-db:5432", "proto": "tcp", "deny":   true },
    { "src": "dave@external.org", "dst": "warehouse-cams:554", "proto": "tcp", "accept": true }
  ]
}
```

### 10.3 Expected compiled output (from 10.2)

Given devices:
- `deviceA` (alice@corp.com, kind=client, tag=none)
- `deviceB` (tag:prod-web, kind=connector, advertises 10.10.0.0/24)
- `deviceC` (tag:prod-db, kind=connector, advertises 10.20.0.0/24)
- `deviceD` (dave@external.org, kind=client, tag=none)

Compiled `AllowedIPs`:
```
deviceA -> deviceB: [10.10.0.0/24]       (alice is admin, *:* includes everything)
deviceA -> deviceC: [10.20.0.0/24]
deviceA -> deviceD: []                    (no rule grants admin -> contractor)

deviceD -> deviceB: [10.10.0.0/24]       (contractor -> warehouse-cams)
deviceD -> deviceC: []                   (denied -- no rule)
deviceD -> deviceA: []                   (denied -- no rule)
```

ExitNodes: [deviceA] (alice is in group:admins)

### 10.4 Edge case fixtures

| Fixture | Input | Expected |
|---|---|---|
| Empty acls | `"acls": []` | deny all (no rules match) |
| Bare IPv4 host | `"hosts": {"srv": "10.0.0.1"}` | normalized to `10.0.0.1/32` |
| Group with zero members | `"group:empty": []` | resolves to empty device set |
| Duplicate group members | `"group:admins": ["alice@c.com", "alice@c.com"]` | deduplicated at resolve |
| Port range edge | `"dst": ["tag:web:1-65535"]` | valid, all ports |
| Invalid port | `"dst": ["tag:web:0"]` | INVALID_PORT |
| Range reversed | `"dst": ["tag:web:100-50"]` | INVALID_PORT_RANGE |
| ICMP with ports | `"proto": "icmp", "dst": ["*:80"]` | PROTO_PORT_MISMATCH (warning) |
| Exit node is connector | `"exitNodes": ["tag:prod-web"]` where tag:prod-web is connector | EXIT_NODE_IS_CONNECTOR |
| Test accept fails | `accept: true` on denied traffic | TEST_FAILED_ACCEPT |
| Test deny fails | `deny: true` on allowed traffic | TEST_FAILED_DENY |

---

## 11. Integration with the existing reconciler

The compiled policy output feeds directly into the coordinator's `buildMap()`
function, which produces `NetworkMap` entries matching the existing `map.rs`
schema. The data flow:

```
CompiledPolicy
    |
    v
coordinator.buildMap(device)
    |
    v
NetworkMap {
    self:    { overlay_ip, transport },
    gateway: { endpoint, wg_pubkey, masque_sni },
    peers:   [
        Peer {
            name:        "<device-name>",
            wg_pubkey:   "<hex-encoded pubkey>",
            allowed_ips: ["10.10.0.0/24", "192.168.50.0/24", ...]
        },
        ...
    ],
    dns:     ["fd7a:helix:1::1"]
}
    |
    v  (protobuf WatchNetworkMap stream)
    |
    v
map.rs: NetworkMap -> reconcile() -> ReconcileDelta
    |
    v
helix_orch: add_route / remove_route / switch_transport
```

The policy compiler is the **only** source of the `allowed_ips` values on
each peer entry. There is no separate mechanism for populating allowed_ips --
they are always the compiled output of the policy engine. This guarantees
that need-to-know (C4/P1) is enforced by construction: a peer's allowed_ips
contain only prefixes the policy explicitly grants.

---

## 12. File Layout

```
helix-go/internal/policy/
  spec.go         -- Spec, GroupMap, HostMap, TagOwnerMap, ACLRule, DestRef,
                     TestCase, Protocol, PortSet, PortRange types
  parser.go       -- ParseSpec, ParseSpecDryRun, unmarshalSpec
  validate.go     -- Spec.Validate, ValidationError, ValidationErrors,
                     all validation passes (§4.4)
  resolve.go      -- Resolver, ResolveSelector, ResolveGroup, ResolveHost,
                     ResolveTag, ResolveUser, ResolveAutogroup,
                     ResolveHostToConnector
  compile.go      -- Compile, CompileAndActivate, CompiledPolicy, PortRule,
                     NetworkMapPeer
  test.go         -- RunTests, RunTestsOrReject, TestResult
  iface.go        -- Compiler interface (per [02 §1.3]), CompiledPolicy
  errors.go       -- PolicyError, IsValidationError, error code constants
  spec_test.go    -- unit tests for parsing + validation
  resolve_test.go -- unit tests for resolution (with mock DB)
  compile_test.go -- unit tests for compilation + determinism property test
  test_test.go    -- unit tests for test execution
```

---

## 13. Determinism Property Test

```go
// internal/policy/compile_test.go

// TestCompileIsDeterministic verifies that compiling the same spec
// against the same DB state produces byte-identical CompiledPolicy
// (deep equality), satisfying the P2 invariant.
func TestCompileIsDeterministic(t *testing.T) {
    spec := loadFixture(t, "full_policy.json")
    resolver := newMockResolver(t, mockDevices, mockUsers, mockConnectors)

    // Compile twice independently.
    cp1, err1 := Compile(context.Background(), spec, resolver)
    cp2, err2 := Compile(context.Background(), spec, resolver)

    require.NoError(t, err1)
    require.NoError(t, err2)

    // Deep equality -- every field, every map key order (sorted).
    assert.Equal(t, cp1.Version, cp2.Version)
    assert.Equal(t, cp1.VisibleTo, cp2.VisibleTo)
    assert.Equal(t, cp1.AllowedIPs, cp2.AllowedIPs)
    assert.Equal(t, cp1.Verdicts, cp2.Verdicts)
    assert.Equal(t, cp1.ExitNodes, cp2.ExitNodes)

    // Deterministic output ordering: sorting by device ID UUID.
    for _, allowed := range cp1.AllowedIPs {
        for _, prefixes := range allowed {
            assert.True(t, sort.SliceIsSorted(prefixes, ...))
        }
    }
}
```

---

## Sources verified

- [Tailscale ACL policy syntax reference](https://tailscale.com/docs/reference/syntax/policy-file) -- groups, hosts, ACLs, grants, tests, autogroups
- [Tailscale grants syntax](https://tailscale.com/docs/reference/syntax/grants) -- next-gen access control, protocol/port specification
- [Headscale v2 policy types](https://pkg.go.dev/github.com/juanfont/headscale@v0.27.1/hscontrol/policy/v2) -- Go type definitions, Alias interface, validation logic
- [Headscale ACL documentation](https://headscale.net/0.27.1/ref/acls/) -- policy file format, special identifiers, edge cases
- [Headscale policy source code](https://raw.githubusercontent.com/juanfont/headscale/main/hscontrol/policy/v2/policy.go) -- PolicyManager, compilation pipeline, filter rule generation
- [Headscale policy types source](https://raw.githubusercontent.com/juanfont/headscale/main/hscontrol/policy/v2/types.go) -- alias resolution, autogroups, port parsing, validation
- [Tailscale WireGuard peer mapping](https://deepwiki.com/tailscale/tailscale/3.4-control-knobs-and-policy) -- ACL-to-AllowedIPs compilation, filter rules
- [Tailscale data plane](https://deepwiki.com/tailscale/tailscale-rs/4-data-plane) -- ingress filter, source IP verification
- HelixVPN control plane specification [02 §7] -- policy model, compiler interface, compiled output shape
- HelixVPN `map.rs` [helix_core/crates/helix-core/src/map.rs] -- NetworkMap, Peer, allowed_ips schema
- HelixVPN `reconciler.rs` [helix_core/crates/helix-core/src/reconciler.rs] -- live reconciliation, route table convergence

Date verified: 2026-07-08
