# Tailscale ACL & HuJSON Exhaustive Reference

> An LLM-friendly reference for understanding and editing Tailscale tailnet policy files.
> Covers every field, selector, autogroup, edge case, and editing concern.
> Sources: Tailscale docs (policy-file syntax, grants syntax, device posture, grant examples),
> the JWCC specification by Nigel Tao, and the tailscale/hujson GitHub repo.
> Last updated: 2026-06-04

---

## Table of Contents

1. [HuJSON / JWCC: The Format](#1-hujson--jwcc-the-format)
2. [Policy File Top-Level Structure](#2-policy-file-top-level-structure)
3. [Grants](#3-grants)
4. [ACLs (Legacy)](#4-acls-legacy)
5. [SSH](#5-ssh)
6. [Auto Approvers](#6-auto-approvers)
7. [Node Attributes (nodeAttrs)](#7-node-attributes-nodeattrs)
8. [Postures](#8-postures)
9. [Tag Owners](#9-tag-owners)
10. [Groups](#10-groups)
11. [Hosts](#11-hosts)
12. [IP Sets (ipsets)](#12-ip-sets-ipsets)
13. [Tests](#13-tests)
14. [SSH Tests](#14-ssh-tests)
15. [Network Policy Options](#15-network-policy-options)
16. [All Autogroups (Exhaustive)](#16-all-autogroups-exhaustive)
17. [All Protocol Aliases](#17-all-protocol-aliases)
18. [All Posture Attributes (Exhaustive)](#18-all-posture-attributes-exhaustive)
19. [Selector Reference (Complete)](#19-selector-reference-complete)
20. [Grants vs ACLs: Migration Guide](#20-grants-vs-acls-migration-guide)
21. [Worked Examples](#21-worked-examples)
22. [Editing HuJSON: Practical Guide](#22-editing-hujson-practical-guide)
23. [Edge Cases & Gotchas](#23-edge-cases--gotchas)

---

## 1. HuJSON / JWCC: The Format

The tailnet policy file is written in **HuJSON** (Human JSON), which Tailscale also calls **JWCC** (JSON With Commas and Comments). It is a strict superset of standard JSON — every valid JSON file is valid HuJSON, but HuJSON adds two convenience features:

### 1.1 Trailing Commas

A comma after the last element in an array or last member in an object is allowed:

```json
{
  "grants": [
    {
      "src": ["group:eng"],
      "dst": ["tag:prod"],
      "ip": ["*"],
    },
  ],
}
```

**Why this matters for editing**: When you add or remove a line in a multi-element list, you don't have to fiddle with the comma on the adjacent line. This produces cleaner diffs.

### 1.2 Comments

Both C-style block comments and C++-style line comments are supported, anywhere standard JSON allows whitespace:

```json
{
  // This is a line comment — terminated by \n (or EOF)
  "grants": [
    {
      /* This is a block comment
         that can span multiple lines */
      "src": ["group:eng"],
      "dst": ["tag:prod"],  // inline comment
      "ip": ["*"],
    },
  ],
}
```

**Comment rules**:
- `//` line comments: terminated by `\n`. A line comment at end of file may omit the final `\n`.
- `/* */` block comments: can span multiple lines.
- Comments are stripped during parsing; they are not preserved by the Tailscale policy engine.
- `#` comments are **not** supported (JWCC is a valid JavaScript superset, and `#` is not).

### 1.3 What HuJSON Does NOT Add

HuJSON deliberately avoids these extensions:
- **No unquoted strings** — all strings must be in double quotes.
- **No single-quoted strings** — only `"double quotes"`.
- **No multi-line strings** — no heredocs or triple-quoted strings.
- **No NaN/Infinity** — the number space matches JSON exactly.
- **No missing commas** — unlike JSON5, `[1 2 3]` is invalid. Commas are required between elements.
- **Duplicate keys** — allowed (matching JSON behavior), but duplicate key detection requires O(N) memory.

### 1.4 Formatting & Tooling

- **`hujsonfmt`** — Go-based formatter: `go install github.com/tailscale/hujson/cmd/hujsonfmt@latest`
- **VS Code** — treat `.hujson` as `jsonc` with trailing commas:
  ```json
  "files.associations": { "*.hujson": "jsonc" },
  "json.schemas": [{ "fileMatch": ["*.hujson"], "schema": { "allowTrailingCommas": true } }]
  ```
- **Converting to standard JSON** — `hujsonfmt -input-jwcc` strips comments and trailing commas.
- The Tailscale admin console and API accept HuJSON; they normalize it internally.

### 1.5 HuJSON Editing Semantics for MCP Servers

When an MCP server reads and writes the policy file:
1. **Preserve comments and trailing commas** on read — they are part of the file's character.
2. **Prefer emitting HuJSON** when writing — include trailing commas and optionally comments.
3. **The Tailscale API accepts both strict JSON and HuJSON** for the policy file — either works.
4. **Minimize diff churn** — add new array elements after the last existing element, preserving the trailing comma on the prior element.

---

## 2. Policy File Top-Level Structure

The policy file is a single HuJSON object. All sections are optional. Order of sections does not matter.

```json
{
  // Access control (prefer grants over acls)
  "grants": [...],
  "acls": [...],

  // SSH
  "ssh": [...],

  // Automation
  "autoApprovers": {...},

  // Attributes
  "nodeAttrs": [...],
  "postures": {...},

  // Targets / definitions
  "tagOwners": {...},
  "groups": {...},
  "hosts": {...},
  "ipsets": {...},

  // Tests
  "tests": [...],
  "sshTests": [...],

  // Network-wide settings (rarely needed)
  "derpMap": {...},
  "disableIPv4": false,
  "OneCGNATRoute": "",
  "randomizeClientPort": false,

  // Default posture (optional)
  "defaultSrcPosture": [...],
}
```

| Section | Key | Type | Purpose |
|---------|-----|------|---------|
| Grants | `grants` | Access control | Network + application layer policies with optional route filtering. **Preferred.** |
| ACLs | `acls` | Access control | Network layer policies only (legacy, no new features) |
| SSH | `ssh` | Access control | Tailscale SSH rules |
| Auto Approvers | `autoApprovers` | Automation | Auto-approve subnet routers, exit nodes, app connectors |
| Node Attributes | `nodeAttrs` | Attributes | Additional attributes on devices (flags, app connectors, funnel, etc.) |
| Postures | `postures` | Attributes | Device posture rule definitions (key-value assertions) |
| Tag Owners | `tagOwners` | Targets | Who can assign which tags |
| Groups | `groups` | Targets | Named groups of users |
| Hosts | `hosts` | Targets | Named aliases for IP addresses / CIDRs |
| IP Sets | `ipsets` | Targets | Named groups of network segments with add/remove composition |
| Tests | `tests` | Tests | Assertions about ACL/grant policy |
| SSH Tests | `sshTests` | Tests | Assertions about SSH policy |
| Default Src Posture | `defaultSrcPosture` | Posture | Baseline posture for rules without explicit `srcPosture` |

---

## 3. Grants

Grants are the recommended, modern access control syntax. They combine network and application layer capabilities.

### 3.1 Core Concepts

- **Deny by default** — access must be explicitly granted.
- **Implied accept** — unlike ACLs, there is no `action` field; grants only grant, never restrict.
- **Union semantics** — if multiple grants match a connection, the engine applies the union of all granted capabilities. More specific grants do not override less specific ones; they add to them.
- **Application capabilities require network access** — `app` capabilities only apply if the device also has network-level access (`ip`).

### 3.2 Grant Structure

```json
{
  "grants": [
    {
      "src": ["<list-of-sources>"],
      "dst": ["<list-of-destinations>"],
      "ip": ["<list-of-ports-or-protocols>"],
      "app": {
        "<domainName>/<capabilityName>": [
          { "<parameter>": "<value>" }
        ]
      },
      "srcPosture": ["<list-of-posture-conditions>"],
      "via": ["<list-of-routing-tags>"]
    }
  ]
}
```

All fields except `src` and `dst` are optional, but you must include at least one of `ip` or `app` (or both). If both are omitted, the source has no network access unless granted by another rule.

### 3.3 Source Selectors (`src`)

| Selector | Description | Example |
|----------|-------------|---------|
| `*` | All tailnet devices + approved subnets + `autogroup:shared`. Does NOT include non-Tailscale devices (unless approved route). | `*` |
| `group:<name>` | All members of a group | `group:prod` |
| `<user>@<domain>` | All devices of a specific user. GitHub: `<user>@github`. Passkey: `<user>@passkey`. | `alice@example.com` |
| `tag:<name>` | All devices with a specific tag | `tag:server` |
| `autogroup:<role>` | All members of a role (`admin`, `member`, `owner`, `it-admin`, `network-admin`, `billing-admin`, `auditor`) | `autogroup:admin` |
| `autogroup:tagged` | All devices with any tag | `autogroup:tagged` |
| `autogroup:shared` | Devices belonging to users who accepted a sharing invitation | `autogroup:shared` |
| `autogroup:danger-all` | All sources including those outside your tailnet | `autogroup:danger-all` |
| `<IP>/<CIDR>` or `<IP>` | Devices in a CIDR range or specific Tailscale IP | `192.0.2.0/24`, `100.100.123.123` |
| `<hostAlias>` | Host defined in the `hosts` section | `my-server` |
| `ipset:<name>` | IP set defined in the `ipsets` section | `ipset:prod` |
| `user:*@<domain>` | All tailnet members whose login is in the specified domain | `user:*@example.com` |

**Note**: When multiple selectors are in a `src` array, the policy engine uses the **union** of all matching entities.

### 3.4 Destination Selectors (`dst`)

All `src` selectors are available, plus these destination-only selectors:

| Selector | Description | Example |
|----------|-------------|---------|
| `autogroup:internet` | Access to internet through exit nodes | `autogroup:internet` |
| `autogroup:self` | User's own devices (when src is `autogroup:<role>`, `group:<name>`, or individual user) | `autogroup:self` |
| `svc:<name>` | A Tailscale Service (virtual IP) | `svc:web-server` |

**Critical distinction**: `autogroup:self` only applies to **user-owned** devices. It does NOT apply to tagged devices. You cannot use `autogroup:self` with `autogroup:tagged`.

**CIDR in grants**: Grants allowing a CIDR (e.g. `192.168.0.0/16`) control which traffic is **permitted**, not which routes are **injected**. Routes are injected only when a subnet router advertises them and an admin approves them.

### 3.5 Network Layer Capabilities (`ip`)

| Selector | Description | Example |
|----------|-------------|---------|
| `*` | All ports (implies TCP, UDP, ICMP) | `*` |
| `<port>` | Single port (implies TCP, UDP, ICMP) | `443` |
| `<port>-<port>` | Port range (implies TCP, UDP, ICMP) | `80-443` |
| `<proto>:*` | All ports of a specific protocol. Useful for portless protocols like ICMP. | `icmp:*`, `sctp:*` |
| `<proto>:<port>` | Specific protocol + port | `tcp:443`, `tcp:80-443` |

Protocol aliases (see [Section 17](#17-all-protocol-aliases) for full table):
`igmp` (2), `ipv4`/`ip-in-ip` (4), `tcp` (6), `egp` (8), `igp` (9), `udp` (17), `gre` (47), `esp` (50), `ah` (51), `sctp` (132).

You can also use raw IANA protocol numbers 1–255 as strings: `"16"`.

**Port rules**:
- If traffic is allowed for a given pair of IPs, ICMP is also allowed automatically.
- Only TCP, UDP, and SCTP support port specification. All other protocols only support `*` for the port.

### 3.6 Application Layer Capabilities (`app`)

The `app` field maps **capability identifiers** to arrays of parameter objects:

```json
"app": {
  "tailscale.com/cap/tailsql": [
    { "dataSrc": ["warehouse"] }
  ],
  "tailscale.com/cap/kubernetes": [
    { "impersonate": { "groups": ["system:masters"] } }
  ],
  "tailscale.com/cap/golink": [
    { "admin": true }
  ],
  "tailscale.com/cap/secrets": [
    { "action": ["put", "activate"], "secret": ["prod/*"] }
  ],
  "tailscale.com/cap/relay": []
}
```

**Naming convention**: `<domainName>/<capabilityName>` (e.g. `tailscale.com/cap/tailsql`).

**Reserved domains**: `tailscale.com` and `tailscale.io` are reserved for Tailscale products. Use a domain you control for custom capabilities.

**Opaque parameters**: The policy engine treats `app` parameters as opaque JSON. It compiles and distributes them but does **not** validate against any schema. Clients use these parameters for local authorization decisions.

**Known Tailscale capabilities**:

| Capability | Application | Parameters |
|------------|-------------|------------|
| `tailscale.com/cap/tailsql` | TailSQL (SQL playground) | `dataSrc` (array of data source names or `*`) |
| `tailscale.com/cap/kubernetes` | Kubernetes auth proxy | `impersonate.groups` (array of K8s groups) |
| `tailscale.com/cap/golink` | Golink (short links) | `admin` (boolean) |
| `tailscale.com/cap/secrets` | Setec (secrets manager) | `action` (array: `get`, `put`, `activate`, `info`), `secret` (array of glob patterns) |
| `tailscale.com/cap/relay` | Peer Relays | (empty array — presence of the capability enables relay) |

### 3.7 Device Posture Requirements (`srcPosture`)

An array of posture conditions that the source device must meet. The grant only applies if the source device satisfies **any** of the listed postures (OR logic between postures, AND logic within a posture).

```json
"srcPosture": ["posture:latestMac", "posture:approvedWindows"]
```

See [Section 8 (Postures)](#8-postures) for full posture syntax.

**Edge case**: `srcPosture` only applies to traffic originating from Tailscale nodes within the same network. Shared nodes and devices behind subnet routers bypass posture conditions — they are permitted if IP-based conditions match.

### 3.8 Routing Specifications (`via`)

The `via` field forces traffic through specific exit nodes, subnet routers, or app connectors:

```json
{
  "src": ["group:tor"],
  "dst": ["autogroup:internet"],
  "ip": ["*"],
  "via": ["tag:exit-node-tor"]
}
```

**Rules**:
- Only **tags** are allowed in `via` (no groups, users, or IPs).
- Omitting `via` (or `[]` or `null`) allows access through any routing device.
- If a specified routing device is unavailable, users in that group **cannot** access the destination until it returns or the policy is updated.
- Only accessible routers are candidates (depends on applicable access policies).

### 3.9 Grant Evaluation Semantics

1. Grants follow **deny by default**.
2. Multiple matching grants produce the **union** of all capabilities.
3. Application capabilities only apply if network-level access exists (from `ip` or another grant's `ip`).
4. `srcPosture` conditions use OR across the array, AND within a posture definition.
5. If `defaultSrcPosture` is set, it applies to all rules without explicit `srcPosture`. An explicit `srcPosture` **replaces** (not adds to) the default.

---

## 4. ACLs (Legacy)

ACLs are the first-generation access control syntax. Tailscale recommends migrating to grants. ACLs will continue to work indefinitely but will receive no new features.

### 4.1 ACL Rule Structure

```json
{
  "acls": [
    {
      "action": "accept",
      "src": ["<list-of-sources>"],
      "proto": "tcp",
      "dst": ["<host>:<port>"]
    }
  ]
}
```

**Key differences from grants**:
- `action` field required (only value: `"accept"`)
- `dst` combines host and port: `"tag:server:22"` or `"192.0.2.1:443"`
- No `ip` field — ports are part of `dst`
- No `app` field — no application capabilities
- No `via` field — no route filtering
- `srcPosture` is supported
- Legacy field names: `users` (instead of `src`) and `ports` (instead of `dst`)

### 4.2 ACL `action`

The only possible value is `"accept"`. Tailscale denies by default.

### 4.3 ACL `src`

Same selectors as grant `src` (see [Section 3.3](#33-source-selectors-src)).

### 4.4 ACL `proto` (Optional)

Same as grants — see [Section 3.5](#35-network-layer-capabilities-ip) and [Section 17](#17-all-protocol-aliases).

If omitted, applies to all TCP and UDP traffic.

### 4.5 ACL `dst`

Format: `"<host>:<port>"` where port can be `*`, a single number, comma-separated (`80,443`), or a range (`1000-2000`).

`<host>` supports: `*`, user email, `group:<name>`, Tailscale IP, host alias, CIDR, `tag:<name>`, `autogroup:internet`, `autogroup:self`, `autogroup:<role>`, `ipset:<name>`.

### 4.6 4via6 Subnet Routers

When targeting resources behind a 4via6 subnet router, use the **IPv6** CIDR/address as the destination, not the IPv4 address. Use `tailscale debug via` to get the IPv6 CIDR.

### 4.7 Taildrop Precedence

Taildrop permits file sharing between devices you're logged in to, even if ACLs restrict access.

---

## 5. SSH

Tailscale SSH rules define who can SSH to what devices and as which users. Both network access (port 22) and SSH access rules are required for a connection.

### 5.1 SSH Rule Structure

```json
{
  "ssh": [
    {
      "action": "accept",
      "src": ["<list-of-sources>"],
      "dst": ["<list-of-destinations>"],
      "users": ["<list-of-ssh-users>"],
      "checkPeriod": "20h",
      "acceptEnv": ["GIT_EDITOR", "GIT_COMMITTER_*", "CUSTOM_VAR_V?"],
      "srcPosture": ["<list-of-posture-conditions>"]
    }
  ]
}
```

### 5.2 SSH `action`

| Value | Description |
|-------|-------------|
| `accept` | Accept connections from already-authenticated tailnet users |
| `check` | Require periodic re-authentication per `checkPeriod` (default 12h) |

**Evaluation order**: Check policies are evaluated before accept policies. If a user matches both, the `check` rule takes precedence.

**Premium/Enterprise only**: `checkPeriod` and `localpart:*@<domain>` are Premium/Enterprise features.

### 5.3 SSH `src`

Same as general `src` selectors. **Cannot** use `*`, other users (than self), IP addresses, or hostnames as `dst`.

### 5.4 SSH `dst`

**Highly restricted** compared to general `dst`:
- A tag (e.g. `tag:prod`)
- `autogroup:self` (if src contains only users/groups)
- A single named user (if src contains only the same user)
- **Cannot** specify a port (only port 22)
- **Cannot** use `*`

**Why**: Tailscale prevents one user from SSHing into another user's personal device. Only user-to-own-device, user-to-tagged-device, or tagged-to-tagged connections are allowed.

### 5.5 SSH `users`

| Value | Description |
|-------|-------------|
| `autogroup:nonroot` | Any user that is not `root` |
| `localpart:*@<domain>` | Map login email local-part to SSH username (Premium/Enterprise) |
| Specific username | e.g. `"ubuntu"`, `"root"` |
| (omitted) | Uses the local host's current user |

**Gotcha**: If `dst` is changed from `autogroup:self` to a tag, consider removing `autogroup:nonroot` from `users` — otherwise anyone in `src` can SSH as any non-root user on the destination.

### 5.6 SSH `checkPeriod`

Format: `<number><unit>` where unit is `m` (minutes) or `h` (hours). Min 1m, max 168h (1 week). Special value: `"always"` (check every connection — can cause issues with automation like Ansible).

### 5.7 SSH `acceptEnv`

Requires Tailscale v1.76.0+ on the host. Allowlists environment variable names for forwarding.

Wildcard patterns:
- `*` matches zero or more characters
- `?` matches a single character

Examples:

| Pattern | Matches | Rejects |
|---------|---------|---------|
| `*` | Everything | — |
| `FOO_*` | `FOO_A`, `FOO_B` | `BAZ` |
| `FOO_?` | `FOO_A`, `FOO_B` | `FOO_OTHER` |
| `FOO_A` | `FOO_A` | `FOO_B` |

### 5.8 SSH `srcPosture`

Same as grant `srcPosture` — array of posture conditions.

### 5.9 SSH Connection Types Allowed

The only SSH connections Tailscale permits:
1. User → their own devices (as any user including root)
2. User → tagged device (as any user including root)
3. Tagged device → tagged device (cannot be check mode)
4. User → tagged device shared with them (if ACL allows SSH)

---

## 6. Auto Approvers

Auto approvers bypass the admin console approval for subnet routers, exit nodes, and app connectors.

### 6.1 Structure

```json
{
  "autoApprovers": {
    "routes": {
      "<CIDR>": ["<approvers>"]
    },
    "exitNode": ["<approvers>"],
    "appConnectors": {
      "<app>": ["<approvers>"]
    }
  }
}
```

### 6.2 Routes

```json
"routes": {
  "192.0.2.0/24": ["group:engineering", "alice@example.com", "tag:foo"],
  "198.51.100.0/24": ["autogroup:member"]
}
```

Auto-approver of a route can also advertise a **subnet** of that route.

### 6.3 Exit Nodes

```json
"exitNode": ["tag:bar"]
```

### 6.4 Approvers

Each approver can be: user email, `group:<name>`, `autogroup:<role>`, or `tag:<name>`.

**Important**: Use tags as auto-approvers to avoid routes stopping when the user who advertised them is re-authenticated by someone else, suspended, or deleted.

**Retroactivity**: Auto-approver policies only apply when Tailscale **first** receives a route advertisement. Updating the policy file does NOT retroactively approve existing unapproved routes. To trigger re-approval, remove the route from the subnet router and re-advertise it.

---

## 7. Node Attributes (nodeAttrs)

`nodeAttrs` apply additional attributes to specific devices. Unlike `postures`, node attributes are **flags** (not key-value pairs).

### 7.1 Structure

```json
{
  "nodeAttrs": [
    {
      "target": ["my-kid@my-home.com", "tag:server"],
      "attr": ["nextdns:abc123", "nextdns:no-device-info"]
    },
    {
      "target": ["autogroup:member"],
      "attr": ["funnel"]
    },
    {
      "target": ["tag:office-network", "group:sea-office"],
      "attr": ["randomize-client-port"]
    }
  ]
}
```

### 7.2 Fields

| Field | Type | Description |
|-------|------|-------------|
| `target` | Array of selectors | Devices the attributes apply to. Can be: `tag:<name>`, `<user>@<domain>`, `group:<name>`, `*`, `autogroup:<role>` |
| `attr` | Array of strings | Flag attributes to apply |
| `app` | Object | Application layer capabilities (same structure as grant `app`) |

### 7.3 Common Attributes

| Attribute | Description |
|-----------|-------------|
| `funnel` | Enables Tailscale Funnel on the targeted devices |
| `randomize-client-port` | Uses random port for WireGuard traffic (instead of default 41641) |
| `nextdns:<configID>` | Sets NextDNS configuration ID |
| `nextdns:no-device-info` | Disables sending device metadata to NextDNS |
| `disable-ipv4` | Stops assigning Tailscale IPv4 addresses (use instead of top-level `disableIPv4`) |

### 7.4 App Connectors via nodeAttrs

```json
{
  "nodeAttrs": [
    {
      "target": ["*"],
      "app": {
        "tailscale.com/app-connectors": [
          {
            "name": "example-app",
            "connectors": ["tag:example-connector"],
            "domains": ["example.com"],
            "routes": ["192.0.2.0/24"]
          }
        ]
      }
    }
  ]
}
```

---

## 8. Postures

Postures define sets of device posture assertions that can be referenced in `srcPosture` or `defaultSrcPosture`.

### 8.1 Structure

```json
{
  "postures": {
    "posture:latestMac": [
      "node:os == 'macos'",
      "node:tsReleaseTrack == 'stable'",
      "node:tsVersion >= '1.40'"
    ],
    "posture:anyMac": [
      "node:os == 'macos'",
      "node:tsReleaseTrack == 'stable'"
    ],
    "posture:trusted": [
      "node:os == 'windows'",
      "custom:edrScore >= 70"
    ]
  }
}
```

Each posture name must start with `posture:`. Each posture is a list of **assertion strings**. All assertions within a posture must be true (AND logic).

### 8.2 Operators

| Operator | Works with | Example |
|----------|-----------|---------|
| `==` | All types | `node:os == 'macos'` |
| `!=` | All types | `node:os != 'linux'` |
| `IN` | Strings | `node:os IN ['macos', 'linux', 'windows']` |
| `NOT IN` | Strings | `node:os NOT IN ['ios', 'android']` |
| `IS SET` | Custom attributes | `custom:tier IS SET` |
| `NOT SET` | Custom attributes | `custom:tier NOT SET` |
| `>=`, `>`, `<=`, `<` | Numbers, versions | `node:tsVersion >= '1.40'`, `custom:edrScore >= 70` |

**Version comparison**: `node:osVersion` and `node:tsVersion` use a comparator that handles mixed numeric/non-numeric version strings.

**Unset attributes**: If a posture attribute is unset for a device, the posture will **not** match that device, regardless of the operator — even negative operators like `!=` or `NOT SET` won't match for truly unset attributes that are referenced in the posture.

### 8.3 Posture Attributes (Built-In)

| Key | Namespace | Description | Values |
|-----|-----------|-------------|--------|
| `node:os` | node | Operating system | `macos`, `windows`, `linux`, `ios`, `android`, `freebsd`, `openbsd`, `illumos`, `js` |
| `node:osVersion` | node | OS version | Quoted version string |
| `node:tsAutoUpdate` | node | Auto-updates enabled | `true`, `false` |
| `node:tsReleaseTrack` | node | Release track | `stable`, `unstable` |
| `node:tsStateEncrypted` | node | Client state encrypted at rest | `true`, `false` |
| `node:tsVersion` | node | Tailscale version | Quoted version string |
| `ip:country` | ip | Country of public IP | ISO 3166-1 two-letter code (uppercase) |

**Note**: `node:tsAutoUpdate` is only `true` when Tailscale's built-in auto-update is enabled. External mechanisms (App Store, Google Play) result in `false`.

### 8.4 Posture Attributes (Custom)

- Set via the Posture Attributes API (Premium/Enterprise).
- Third-party integrations (CrowdStrike Falcon, etc.): Standard/Premium/Enterprise.
- Namespace `custom:` for user-defined attributes.
- Integration-specific namespaces (e.g. `falcon:ztaScore`).

```json
"posture:crowdstrike-trusted": [
  "falcon:ztaScore >= 70"
]
```

### 8.5 Using Postures in Access Rules

In grants:
```json
{
  "src": ["group:dev"],
  "dst": ["tag:production"],
  "ip": ["*"],
  "srcPosture": ["posture:latestMac"]
}
```

Multiple postures in `srcPosture` = **OR** logic (any one matching is sufficient):
```json
"srcPosture": ["posture:approvedMacs", "posture:approvedWindows", "posture:approvedLinux"]
```

### 8.6 Default Source Posture (`defaultSrcPosture`)

```json
"defaultSrcPosture": ["posture:basicWindows", "posture:basicMac", "posture:basicLinux"]
```

Applies to **all** rules that don't have an explicit `srcPosture`. **Not additive** — if a rule specifies its own `srcPosture`, the default is replaced entirely.

This can be used to create rules **more permissive** than the default (by specifying a less strict posture):
```json
{
  "grants": [
    {
      "src": ["autogroup:member"],
      "dst": ["tag:intranet"],
      "ip": ["*"]
      // defaultSrcPosture applies here
    },
    {
      "src": ["group:dev"],
      "dst": ["tag:production"],
      "ip": ["*"],
      "srcPosture": ["posture:prodMac"]
      // This replaces (not adds to) the default
    }
  ]
}
```

**Edge case**: `defaultSrcPosture` only applies to traffic from Tailscale nodes within the same network. Shared nodes and devices behind subnet routers bypass it.

### 8.7 Tailscale SSH + Posture

SSH Console connections use `node:os == 'js'`. To allow these through posture checks, include that assertion:

```json
"posture:allow-ssh-console": [
  "node:os IN ['macos', 'linux', 'js']"
]
```

---

## 9. Tag Owners

Defines which users/groups can assign each tag.

```json
{
  "tagOwners": {
    "tag:webserver": ["group:engineering"],
    "tag:secure-server": ["group:security-admins", "president@example.com"],
    "tag:corp": ["autogroup:member"],
    "tag:monitoring": []
  }
}
```

**Rules**:
- Tag names must start with `tag:`.
- Owners can be: user email, `group:<name>`, `autogroup:<role>`, or `tag:<name>`.
- **Shorthand**: `[]` (empty array) is equivalent to `["autogroup:admin"]`. Both `autogroup:admin` and `autogroup:network-admin` can assign all tags, so `[]` implicitly lets only these two roles assign tags.
- You must define a tag in `tagOwners` before using it in access rules.

---

## 10. Groups

Named groups of users for use in access rules.

```json
{
  "groups": {
    "group:engineering": ["dave@example.com", "laura@example.com"],
    "group:sales": ["brad@example.com", "alice@example.com"],
    "group:security-team@example.com": ["bob@example.com"]
  }
}
```

**Rules**:
- Group names must start with `group:`.
- Members are specified by full email/GitHub/Passkey.
- Groups **cannot** contain other groups (to avoid obfuscating membership).
- SCIM-synced groups can be referenced but not edited in the policy file.
- Synced groups use the format `group:<name>@<domain>`.

---

## 11. Hosts

Human-friendly aliases for IP addresses or CIDR ranges.

```json
{
  "hosts": {
    "example-host-1": "198.51.100.100",
    "example-network-1": "198.51.100.0/24"
  }
}
```

**Rule**: Hostnames cannot include the `@` character.

---

## 12. IP Sets (ipsets)

IP sets define named collections of network segments. They support **composition** via `add` and `remove` operations.

### 12.1 Structure

```json
{
  "ipsets": {
    "ipset:internet": [
      "add autogroup:internet",
      "remove ipset:cdn-edge",
      "remove ipset:partner-net"
    ],
    "ipset:cdn-edge": ["198.51.100.6", "198.51.100.7", "198.51.100.13", "198.51.100.14"],
    "ipset:partner-net": ["203.0.113.0/24"],
    "ipset:prod-infra": ["10.0.1.0/24", "10.0.2.0/24"],
    "ipset:stg-infra": ["10.1.1.0/24"],
    "ipset:dev-infra": ["10.2.1.0/24"]
  }
}
```

### 12.2 Composition Operations

| Operation | Description |
|-----------|-------------|
| `add <target>` | Include all IPs from the target (can be `autogroup:internet`, another `ipset:`, or a CIDR/IP) |
| `remove <target>` | Exclude all IPs from the target |

IP sets can reference:
- `autogroup:internet`
- Other `ipset:<name>` definitions
- Individual IP addresses
- CIDR ranges

### 12.3 Using IP Sets

In grants/ACLs, reference with `ipset:<name>`:

```json
{
  "grants": [
    {
      "src": ["group:sea"],
      "dst": ["ipset:internet"],
      "ip": ["*"],
      "via": ["tag:officerouter-sea"]
    }
  ]
}
```

**Note**: IP sets in the `ipsets` section define **which IPs are in the set**. To use an IP set in `dst`, reference it as `ipset:<name>`. CIDR selectors in grants control permitted traffic, not injected routes.

---

## 13. Tests

Tests are assertions about your access policies that run on every policy file change. If an assertion fails, Tailscale **rejects** the policy update.

### 13.1 Structure

```json
{
  "tests": [
    {
      "src": "dave@example.com",
      "srcPostureAttrs": {
        "node:os": "windows"
      },
      "proto": "tcp",
      "accept": ["example-host-1:22", "vega:80"],
      "deny": ["192.0.2.3:443"]
    }
  ]
}
```

### 13.2 Fields

| Field | Description |
|-------|-------------|
| `src` | User identity: email, `group:<name>`, `tag:<name>`, or host alias. Test runs from this device's perspective. |
| `srcPostureAttrs` | Key-value pairs of posture attributes to simulate. Only needed if access rules use posture conditions. |
| `proto` | IP protocol to test. Omitted = TCP or UDP. `"icmp"` for ICMP (use port `0`). |
| `accept` | Destinations that should be allowed. Format: `<host>:<port>` (single numeric port). |
| `deny` | Destinations that should be blocked. Same format. |

### 13.3 Test Destination Hosts

| Type | Example |
|------|---------|
| Tailscale IP | `100.100.123.123:22` |
| Host alias | `my-host:80` |
| User | `shreya@example.com:443` |
| Group | `group:security@example.com:443` |
| Tag | `tag:production:22` |
| Service | `svc:my-service:443` |

**Restrictions**:
- Cannot use CIDR notation — must specify individual IPs or hostnames.
- Cannot use `*` wildcards.
- `accept` destinations cannot be `tag:*`.
- Legacy `allow` key still works instead of `accept`, but `accept` is preferred.

### 13.4 ICMP Tests

```json
{
  "tests": [
    {
      "src": "alice@example.com",
      "proto": "icmp",
      "accept": ["tag:production:0"]
    }
  ]
}
```

### 13.5 Subnet/Exit Node Tests

To ensure a user can access a Tailscale IP but NOT a subnet route:
```json
{
  "tests": [
    {
      "src": "not-allowed@example.com",
      "accept": ["192.0.2.100:22"],
      "deny": ["198.51.100.7:22"]
    }
  ]
}
```

To ensure a user can't access public IPs (i.e. can't use exit nodes):
```json
{
  "tests": [
    {
      "src": "not-allowed@example.com",
      "accept": ["192.0.2.100:22"],
      "deny": ["198.51.100.8:22"]
    }
  ]
}
```

---

## 14. SSH Tests

Assertions about Tailscale SSH access rules. Same rejection-on-failure behavior as `tests`.

### 14.1 Structure

```json
{
  "sshTests": [
    {
      "src": "dave@example.com",
      "dst": ["example-host-1"],
      "accept": ["dave"],
      "check": ["admin"],
      "deny": ["root"],
      "srcPostureAttrs": {
        "node:os": "windows"
      }
    }
  ]
}
```

### 14.2 Fields

| Field | Description |
|-------|-------------|
| `src` | User identity attempting SSH |
| `dst` | Destination device(s) — email, group, tag, or host alias |
| `accept` | SSH usernames that should be accepted without re-check |
| `check` | SSH usernames that should require re-authentication |
| `deny` | SSH usernames that should be denied under all circumstances |
| `srcPostureAttrs` | Simulated posture attributes |

---

## 15. Network Policy Options

Rarely needed. Most networks should never specify these.

### 15.1 `derpMap`

Add custom DERP servers or disable Tailscale-provided DERP servers.

### 15.2 `disableIPv4`

**Deprecated** — use the `disable-ipv4` node attribute instead (set in `nodeAttrs`).

Stops assigning Tailscale IPv4 addresses. All devices receive IPv6 only. Devices without IPv6 support become unreachable.

### 15.3 `OneCGNATRoute`

Controls how Tailscale clients generate routes:

| Value | Behavior |
|-------|----------|
| `""` or omitted | Default heuristics (fine-grained `/32` routes on most platforms; one `100.64/10` route on macOS v1.28+) |
| Other | Override default |

### 15.4 `randomizeClientPort`

**Only use as workaround after consulting Tailscale Support.**

Makes devices use a random port for WireGuard traffic instead of the default static port `41641`. Alternatively, use `nodeAttrs` with `randomize-client-port` for per-device control.

---

## 16. All Autogroups (Exhaustive)

| Autogroup | Allowed In | Description | Plan |
|-----------|-----------|-------------|------|
| `autogroup:internet` | `dst` only | Internet through exit nodes | All |
| `autogroup:self` | `dst` only | User's own devices. NOT for tagged devices. | All |
| `autogroup:owner` | `src`, `dst`, `tagOwner`, `autoApprover` | Tailnet Owner | All |
| `autogroup:admin` | `src`, `dst`, `tagOwner`, `autoApprover` | Admin role | All |
| `autogroup:member` | `src`, `dst`, `tagOwner`, `autoApprover` | Direct tailnet members (incl. invited). NOT shared users. | All |
| `autogroup:tagged` | `src`, `dst`, `tagOwner`, `autoApprover` | Any tagged device | All |
| `autogroup:shared` | `src` only | Users who accepted a sharing invitation | All |
| `autogroup:danger-all` | `src` only | All sources including outside tailnet | All |
| `autogroup:auditor` | `src`, `dst`, `tagOwner`, `autoApprover` | Auditor role | Standard+ |
| `autogroup:billing-admin` | `src`, `dst`, `tagOwner`, `autoApprover` | Billing admin role | Standard+ |
| `autogroup:it-admin` | `src`, `dst`, `tagOwner`, `autoApprover` | IT admin role | Standard+ |
| `autogroup:network-admin` | `src`, `dst`, `tagOwner`, `autoApprover` | Network admin role | Standard+ |
| `autogroup:nonroot` | SSH `users` only | Any non-root user | All |
| `user:*@<domain>` | `src`, `dst`, `tagOwner`, `autoApprover` | All members with login in domain | All |
| `localpart:*@<domain>` | SSH `users` only | Map email local-part to SSH user | Premium+ |
| `autogroup:members` | (legacy) | Same as `autogroup:member` (use `member` instead) | All |

**Key restrictions**:
- `autogroup:self` only applies to **user-owned** devices — NOT tagged devices.
- Cannot use both `autogroup:member` and `autogroup:members` in the same policy file.
- Domain-based autogroups (`user:*@domain`) cannot use known shared domains (e.g. `gmail.com`).
- Domain wildcards (`*`) only match the entire local-part — `user:b*b@example.com` will NOT match `bob@example.com`.
- Domain-based autogroups only include direct tailnet members, not external invited users.
- If a tailnet uses domain aliases, you must specify each alias explicitly.

---

## 17. All Protocol Aliases

| Protocol | Named Alias | IANA Number |
|----------|-------------|-------------|
| Internet Group Management (IGMP) | `igmp` | 2 |
| IPv4 encapsulation | `ipv4`, `ip-in-ip` | 4 |
| Transmission Control (TCP) | `tcp` | 6 |
| Exterior Gateway Protocol (EGP) | `egp` | 8 |
| Any private interior gateway | `igp` | 9 |
| User Datagram (UDP) | `udp` | 17 |
| Generic Routing Encapsulation (GRE) | `gre` | 47 |
| Encap Security Payload (ESP) | `esp` | 50 |
| Authentication Header (AH) | `ah` | 51 |
| Stream Control Transmission Protocol (SCTP) | `sctp` | 132 |

You can also use raw IANA numbers 1–255 as strings (e.g. `"16"`).

**Notes**:
- Requires Tailscale v1.18.2+ for the `proto` field.
- ICMP is automatically allowed when traffic is allowed for a given IP pair.
- Only TCP, UDP, and SCTP support port specification. Others only support `*` for the port.

---

## 18. All Posture Attributes (Exhaustive)

### Built-In (All Plans)

| Key | Type | Description | Example Values |
|-----|------|-------------|----------------|
| `node:os` | String | Operating system | `macos`, `windows`, `linux`, `ios`, `android`, `freebsd`, `openbsd`, `illumos`, `js` |
| `node:osVersion` | String (version) | OS version | `'13.4.0'`, `'17.1'` |
| `node:tsAutoUpdate` | Boolean | Tailscale auto-updates | `true`, `false` |
| `node:tsReleaseTrack` | String | Release track | `'stable'`, `'unstable'` |
| `node:tsStateEncrypted` | Boolean | Client state encrypted at rest | `true`, `false` |
| `node:tsVersion` | String (version) | Tailscale client version | `'1.42.2'` |

### Control Plane (All Plans)

| Key | Type | Description | Example Values |
|-----|------|-------------|----------------|
| `ip:country` | String | Country of public IP | `'CA'`, `'US'`, `'GB'`, `'NL'` |

### Third-Party Integration (Standard+)

| Key | Type | Description | Example |
|-----|------|-------------|---------|
| `falcon:ztaScore` | Number | CrowdStrike Falcon ZTA score | `>= 70` |

### Custom (Premium+)

| Key Pattern | Type | Description | Example |
|-------------|------|-------------|---------|
| `custom:<name>` | Any | User-defined via Posture Attributes API | `custom:edrScore >= 70`, `custom:tier IS SET` |

---

## 19. Selector Reference (Complete)

### User Formats

| Format | When to use | Example |
|--------|-------------|---------|
| `<user>@<domain>` | Email login | `alice@example.com` |
| `<user>@github` | GitHub login | `alice@github` |
| `<user>@passkey` | Passkey login | `alice@passkey` |

### Prefix Summary

| Prefix | Section | Example |
|--------|---------|---------|
| `group:` | Groups | `group:engineering` |
| `tag:` | Tags | `tag:production` |
| `autogroup:` | Autogroups | `autogroup:admin` |
| `ipset:` | IP sets | `ipset:prod-infra` |
| `svc:` | Services | `svc:web-server` |
| `posture:` | Postures | `posture:latestMac` |
| `node:` | Posture attributes | `node:os` |
| `ip:` | Posture attributes | `ip:country` |
| `custom:` | Custom posture attributes | `custom:tier` |
| `falcon:` | CrowdStrike posture | `falcon:ztaScore` |

---

## 20. Grants vs ACLs: Migration Guide

| Feature | ACLs | Grants |
|---------|------|--------|
| Network layer access | ✅ | ✅ |
| Application layer access (`app`) | ❌ | ✅ |
| Route filtering (`via`) | ❌ | ✅ |
| Device posture (`srcPosture`) | ✅ | ✅ |
| `action` field required | Yes (`accept` only) | No (implied accept) |
| Port specification | Combined in `dst` (`tag:server:22`) | Separate `ip` field (`"tcp:22"`) |
| Protocol specification | `proto` field | Integrated in `ip` field (`tcp:443`) |
| Mixed in same policy | ✅ Allowed | ✅ Allowed |

**Migration pattern**:
```json
// Before (ACL)
{
  "acls": [{
    "action": "accept",
    "src": ["group:prod"],
    "dst": ["tag:server:22"]
  }]
}

// After (Grant)
{
  "grants": [{
    "src": ["group:prod"],
    "dst": ["tag:server"],
    "ip": ["tcp:22"]
  }]
}
```

**SSH will eventually unify into grants** — Tailscale has announced this intent but has not yet completed it.

---

## 21. Worked Examples

### 21.1 Minimal: Allow All

```json
{
  "grants": [
    {
      "src": ["*"],
      "dst": ["*"],
      "ip": ["*"]
    }
  ]
}
```

### 21.2 Self-Access Only

```json
{
  "grants": [
    {
      "src": ["autogroup:member"],
      "dst": ["autogroup:self"],
      "ip": ["*"]
    }
  ]
}
```

### 21.3 Group-Based Access with Tags

```json
{
  "groups": {
    "group:eng": ["alice@example.com", "bob@example.com"],
    "group:sales": ["carol@example.com"]
  },
  "tagOwners": {
    "tag:prod": ["group:eng"],
    "tag:internal-tools": ["autogroup:admin"]
  },
  "grants": [
    {
      "src": ["group:eng"],
      "dst": ["tag:prod"],
      "ip": ["*"]
    },
    {
      "src": ["group:sales"],
      "dst": ["tag:internal-tools"],
      "ip": ["tcp:443", "tcp:22"]
    }
  ]
}
```

### 21.4 Exit Nodes with Location-Based Routing

```json
{
  "groups": {
    "group:tor": ["alice@example.com"],
    "group:sea": ["bob@example.com"]
  },
  "grants": [
    {
      "src": ["group:tor"],
      "dst": ["autogroup:internet"],
      "ip": ["*"],
      "via": ["tag:exit-node-tor"]
    },
    {
      "src": ["group:sea"],
      "dst": ["autogroup:internet"],
      "ip": ["*"],
      "via": ["tag:exit-node-sea"]
    },
    {
      "src": ["group:eng"],
      "dst": ["autogroup:internet"],
      "ip": ["*"]
    }
  ]
}
```

### 21.5 Posture-Based Access

```json
{
  "postures": {
    "posture:latestMac": [
      "node:os == 'macos'",
      "node:osVersion == '13.4.0'",
      "node:tsReleaseTrack == 'stable'"
    ],
    "posture:trustedWindows": [
      "node:os == 'windows'",
      "node:tsVersion >= '1.40'"
    ]
  },
  "defaultSrcPosture": ["posture:latestMac", "posture:trustedWindows"],
  "grants": [
    {
      "src": ["autogroup:member"],
      "dst": ["tag:intranet"],
      "ip": ["*"]
    },
    {
      "src": ["group:dev"],
      "dst": ["tag:production"],
      "ip": ["*"],
      "srcPosture": ["posture:latestMac"]
    }
  ]
}
```

### 21.6 Application Capabilities (TailSQL)

```json
{
  "grants": [
    {
      "src": ["group:prod"],
      "dst": ["tag:tailsql"],
      "ip": ["443"],
      "app": {
        "tailscale.com/cap/tailsql": [
          { "dataSrc": ["*"] }
        ]
      }
    },
    {
      "src": ["group:analytics"],
      "dst": ["tag:tailsql"],
      "ip": ["443"],
      "app": {
        "tailscale.com/cap/tailsql": [
          { "dataSrc": ["warehouse"] }
        ]
      }
    }
  ]
}
```

### 21.7 Kubernetes Impersonation

```json
{
  "grants": [
    {
      "src": ["group:prod"],
      "dst": ["tag:k8s-operator"],
      "app": {
        "tailscale.com/cap/kubernetes": [
          { "impersonate": { "groups": ["system:masters"] } }
        ]
      }
    },
    {
      "src": ["group:k8s-user"],
      "dst": ["tag:k8s-operator"],
      "app": {
        "tailscale.com/cap/kubernetes": [
          { "impersonate": { "groups": ["group:k8s-user"] } }
        ]
      }
    }
  ]
}
```

### 21.8 IP Sets with Custom Internet

```json
{
  "ipsets": {
    "ipset:internet": [
      "add autogroup:internet",
      "remove ipset:cdn-edge",
      "remove ipset:partner-net"
    ],
    "ipset:cdn-edge": ["198.51.100.6", "198.51.100.7"],
    "ipset:partner-net": ["203.0.113.0/24"]
  },
  "grants": [
    {
      "src": ["group:sea"],
      "dst": ["ipset:internet"],
      "ip": ["*"],
      "via": ["tag:officerouter-sea"]
    }
  ]
}
```

### 21.9 App Connectors via nodeAttrs

```json
{
  "nodeAttrs": [
    {
      "target": ["*"],
      "app": {
        "tailscale.com/app-connectors": [
          {
            "name": "github-app",
            "connectors": ["tag:github-connector"],
            "domains": ["github.com"],
            "routes": []
          }
        ]
      }
    }
  ],
  "grants": [
    {
      "src": ["group:github-users"],
      "dst": ["autogroup:internet"],
      "ip": ["*"],
      "via": ["tag:github-connector"]
    }
  ]
}
```

### 21.10 Peer Relays

```json
{
  "grants": [
    {
      "src": ["tag:us-east-vpc"],
      "dst": ["tag:us-east-relays"],
      "app": {
        "tailscale.com/cap/relay": []
      }
    },
    {
      "src": ["autogroup:member"],
      "dst": ["tag:us-east-vpc"],
      "ip": ["tcp:80", "tcp:443"]
    }
  ]
}
```

### 21.11 Comprehensive Policy with Tests

```json
{
  "groups": {
    "group:eng": ["dave@example.com", "laura@example.com"],
    "group:sre": ["alice@example.com", "bob@example.com"]
  },
  "tagOwners": {
    "tag:prod": ["group:sre"],
    "tag:dev": ["group:eng", "group:sre"],
    "tag:monitoring": ["autogroup:admin"]
  },
  "postures": {
    "posture:stable": [
      "node:tsReleaseTrack == 'stable'",
      "node:tsVersion >= '1.40'"
    ]
  },
  "grants": [
    {
      "src": ["group:eng"],
      "dst": ["tag:dev"],
      "ip": ["*"]
    },
    {
      "src": ["group:sre"],
      "dst": ["tag:prod"],
      "ip": ["*"],
      "srcPosture": ["posture:stable"]
    },
    {
      "src": ["group:sre"],
      "dst": ["tag:dev"],
      "ip": ["*"]
    },
    {
      "src": ["tag:monitoring"],
      "dst": ["tag:prod", "tag:dev"],
      "ip": ["80", "443", "9100"]
    }
  ],
  "ssh": [
    {
      "action": "accept",
      "src": ["group:sre"],
      "dst": ["tag:prod"],
      "users": ["ubuntu", "root"]
    },
    {
      "action": "check",
      "src": ["group:eng"],
      "dst": ["tag:dev"],
      "users": ["autogroup:nonroot"],
      "checkPeriod": "8h"
    }
  ],
  "autoApprovers": {
    "routes": {
      "10.0.0.0/16": ["group:sre", "tag:prod"]
    },
    "exitNode": ["tag:prod"]
  },
  "tests": [
    {
      "src": "dave@example.com",
      "accept": ["tag:dev:22", "tag:dev:443"],
      "deny": ["tag:prod:22"]
    },
    {
      "src": "alice@example.com",
      "srcPostureAttrs": { "node:tsReleaseTrack": "stable", "node:tsVersion": "1.42.0" },
      "accept": ["tag:prod:22", "tag:prod:443"],
      "deny": ["tag:prod:8080"]
    }
  ],
  "sshTests": [
    {
      "src": "dave@example.com",
      "dst": ["tag:dev"],
      "check": ["dave", "ubuntu"],
      "deny": ["root"]
    },
    {
      "src": "alice@example.com",
      "dst": ["tag:prod"],
      "accept": ["ubuntu", "root"]
    }
  ]
}
```

---

## 22. Editing HuJSON: Practical Guide

### 22.1 Add a Grant

To add a new grant to an existing `grants` array, insert a new object before the trailing comma on the last element (or add a trailing comma to the current last element, then add the new one):

```json
// Before
"grants": [
  { "src": ["group:eng"], "dst": ["tag:dev"], "ip": ["*"] }
]

// After
"grants": [
  { "src": ["group:eng"], "dst": ["tag:dev"], "ip": ["*"] },
  { "src": ["group:sre"], "dst": ["tag:prod"], "ip": ["*"] },
]
```

### 22.2 Remove a Grant

Remove the object and its preceding comma (or the following comma if it's the first element):

```json
// Before
"grants": [
  { "src": ["group:eng"], "dst": ["tag:dev"], "ip": ["*"] },
  { "src": ["group:sre"], "dst": ["tag:prod"], "ip": ["*"] },
]

// After removing the first grant
"grants": [
  { "src": ["group:sre"], "dst": ["tag:prod"], "ip": ["*"] },
]
```

### 22.3 Add a Group Member

```json
// Before
"group:eng": ["dave@example.com", "laura@example.com"]

// After
"group:eng": ["dave@example.com", "laura@example.com", "newuser@example.com"]
```

### 22.4 Add a Tag Owner

```json
// Before
"tagOwners": {
  "tag:prod": ["group:sre"],
}

// After
"tagOwners": {
  "tag:prod": ["group:sre"],
  "tag:staging": ["group:eng"],
}
```

### 22.5 Add a Posture

```json
// Before
"postures": {
  "posture:stable": ["node:tsReleaseTrack == 'stable'"]
}

// After
"postures": {
  "posture:stable": ["node:tsReleaseTrack == 'stable'"],
  "posture:macos-latest": [
    "node:os == 'macos'",
    "node:tsVersion >= '1.40'"
  ],
}
```

### 22.6 Add a Test

```json
// Before
"tests": []

// After
"tests": [
  {
    "src": "newuser@example.com",
    "accept": ["tag:dev:443"],
    "deny": ["tag:prod:22"]
  },
]
```

### 22.7 Adding `srcPosture` to an Existing Grant

```json
// Before
{ "src": ["group:eng"], "dst": ["tag:prod"], "ip": ["*"] }

// After
{ "src": ["group:eng"], "dst": ["tag:prod"], "ip": ["*"], "srcPosture": ["posture:stable"] }
```

### 22.8 Converting ACL to Grant

```json
// Before (ACL)
"acls": [
  { "action": "accept", "src": ["group:eng"], "dst": ["tag:server:22"] }
]

// After (Grant) — remove action, split dst, add ip
"grants": [
  { "src": ["group:eng"], "dst": ["tag:server"], "ip": ["tcp:22"] }
]
```

### 22.9 Handling Comments When Editing Programmatically

Comments are not preserved by the Tailscale API — when you submit a policy file through the API, comments are stripped. If you need to preserve comments, you must manage the raw file yourself and only submit the processed version.

For MCP servers: consider reading the raw policy file, parsing with a HuJSON-aware parser, making edits, and re-serializing with comments intact where possible.

---

## 23. Edge Cases & Gotchas

### 23.1 `autogroup:self` with Tags

`autogroup:self` does NOT match tagged devices. If your `dst` uses `autogroup:self`, it only matches user-owned (untagged) devices. This is a common source of confusion.

### 23.2 SSH `users` with Non-Self Destinations

If you change SSH `dst` from `autogroup:self` to a tag, reconsider `autogroup:nonroot` in `users` — it allows anyone in `src` to SSH as any non-root user on the destination.

### 23.3 Posture Unset Attributes

If a posture attribute is unset for a device, the posture will not match, **even with negative operators** like `!=` or `NOT SET`. This is counterintuitive — `custom:tier != 'prod'` won't match a device that has no `custom:tier` attribute at all.

### 23.4 `defaultSrcPosture` is Replacing, Not Additive

An explicit `srcPosture` on a rule replaces the `defaultSrcPosture` entirely — it does not combine. You can use this to make a specific rule more permissive than the default.

### 23.5 Shared Nodes Bypass Posture

Posture conditions (`srcPosture`, `defaultSrcPosture`) only apply to traffic from tailnet nodes. Shared devices and devices behind subnet routers bypass posture and are permitted if IP conditions match.

### 23.6 CIDR in Grants ≠ Route Injection

A grant allowing `192.168.0.0/16` permits traffic to that range but does NOT inject routes. Routes are only injected when a subnet router advertises them and an admin approves.

### 23.7 ACL Exit Node Restriction

You **cannot** restrict the use of specific exit nodes using ACLs (issue #1567). Use grants with `via` for route filtering instead.

### 23.8 Auto-Approvers Are Not Retroactive

Updating the policy file to add auto-approvers does NOT retroactively approve existing unapproved routes. You must remove and re-advertise the route.

### 23.9 `autogroup:member` vs `autogroup:members`

`autogroup:members` (legacy) is equivalent to `autogroup:member`. You cannot use both in the same policy file. Use `autogroup:member`.

### 23.10 Domain-Based Autogroup Restrictions

- Cannot use known shared domains (e.g. `gmail.com`).
- No arbitrary wildcards: `user:b*b@example.com` won't match `bob@example.com`.
- Domain aliases must be explicitly listed.
- External invited users are not included.

### 23.11 4via6 Destinations

When writing ACLs/grants targeting resources behind 4via6 subnet routers, use the **IPv6** CIDR, not IPv4. Use `tailscale debug via` to get the correct IPv6 CIDR.

### 23.12 Taildrop Bypasses ACLs

Taildrop permits file sharing between your own devices, even if ACLs restrict access.

### 23.13 Grant Union Semantics

Multiple matching grants produce the union of all capabilities. A more specific grant does not override a less specific one — it adds to it. This means you cannot "narrow" access by adding more grants.

### 23.14 Application Capabilities Require Network Access

`app` capabilities in grants only take effect if the device also has network-level access (`ip` from the same or another grant). An `app`-only grant with no `ip` provides no actual access.

### 23.15 SSH Check Mode with Automation

Using `checkPeriod: "always"` can cause issues with automation tools (like Ansible) that open many SSH connections in quick succession. Each connection triggers a re-authentication prompt.

### 23.16 `[]` in tagOwners ≠ Empty

`"tag:monitoring": []` is equivalent to `["autogroup:admin"]`, not "nobody can assign this tag". `autogroup:network-admin` can also assign all tags.

### 23.17 Groups Cannot Nest

Groups cannot contain other groups. This prevents membership obfuscation but means you must explicitly list all members.

### 23.18 `disableIPv4` vs `disable-ipv4` Node Attribute

The top-level `disableIPv4` field is deprecated. Use the `disable-ipv4` attribute in `nodeAttrs` instead for per-device control.

### 23.19 IPv6 Address Format in ACLs

IPv6 addresses in `dst` must follow the format `[1:2:3::4]:80` (bracketed).

### 23.20 Test Destinations Cannot Use CIDRs

In `tests`, you cannot use `192.168.1.0/24:22` — you must specify individual IPs or host aliases.

---

*End of reference. This document was synthesized from the Tailscale tailnet policy file syntax reference, grants syntax reference, device posture management docs, grant examples, the JWCC specification by Nigel Tao, and the tailscale/hujson GitHub repository.*
