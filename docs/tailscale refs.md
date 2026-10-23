# Tailscale ACL & HuJSON Exhaustive Reference

> An LLM-friendly reference for understanding and editing Tailscale tailnet policy files.
> Covers every field, selector, autogroup, edge case, and editing concern.
> Sources: Tailscale docs (policy-file syntax, grants syntax, targets and selectors, device posture,
> IP sets, Tailscale SSH, declarative node sharing, grant examples), the JWCC specification by
> Nigel Tao, and the tailscale/hujson GitHub repo. Canonical docs live under
> `https://tailscale.com/docs/...`; append `.md` to a page URL for its Markdown copy.
> Last updated: 2026-10-22

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
24. [External Tailnets (Declarative Node Sharing, Alpha)](#24-external-tailnets-declarative-node-sharing-alpha)

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
- Comments carry no policy meaning. The Tailscale API returns the policy file as HuJSON or as JSON depending on the `Accept` header; only the HuJSON form can carry comments.
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

  // Declarative node sharing (alpha)
  "externalTailnets": {...},
}
```

The syntax reference's own section table lists only the access control, automation, attribute, target, and test sections. `defaultSrcPosture` is documented with device posture, the network policy options further down the syntax reference, and `externalTailnets` on the declarative node sharing page.

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
| Network options | `derpMap`, `disableIPv4`, `OneCGNATRoute`, `randomizeClientPort` | Network | Rarely needed network-wide settings (see Section 15) |
| External tailnets | `externalTailnets` | Sharing (alpha) | Declarative node sharing with trusted tailnets (see Section 24) |

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
| `autogroup:self` | User's own devices. The `src` must contain only individual users, groups, or role autogroups (`autogroup:member`, `autogroup:admin`, ...) | `autogroup:self` |
| `svc:<name>` | A Tailscale Service (virtual IP) | `svc:web-server` |

**Critical distinction**: `autogroup:self` only applies to **user-owned** devices. It does NOT apply to tagged devices. You cannot use `autogroup:self` with `autogroup:tagged`, and a `src` of tags or `*` cannot be paired with it.

**Exit nodes**: only sources with access to `autogroup:internet` can use exit nodes. `autoApprovers.exitNode` only approves exit node *advertisements* (Section 6).

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

**Only the four forms above are documented.** A bare protocol name such as `"icmp"` is not one of them; write `"icmp:*"`. Note that `icmp` is not in the official alias table even though the docs use it in `icmp:*` and in tests (`"proto": "icmp"`).

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

(Illustrative only: in practice each capability goes in a grant whose `src`/`dst` fit that application.)

**Naming convention**: `<domainName>/<capabilityName>` (e.g. `tailscale.com/cap/tailsql`).

**Reserved domains**: `tailscale.com` and `tailscale.io` are reserved for Tailscale products. Use a domain you control for custom capabilities.

**Opaque parameters**: The policy engine treats `app` parameters as opaque JSON. It compiles and distributes them but does **not** validate against any schema. Clients use these parameters for local authorization decisions.

**Restrictions**: with `autogroup:self` as `dst`, the `src` must be users, groups, or role autogroups. App grants cannot use `autogroup:shared`, `autogroup:danger-all`, `autogroup:internet`, or admin autogroups such as `autogroup:admin` as destinations.

**Known Tailscale capabilities** (from the app capabilities page and feature docs):

| Capability | Application | Parameters |
|------------|-------------|------------|
| `tailscale.com/cap/drive` | Taildrive | `shares` (share names or `*`), `access` (`ro` or `rw`). `tailscale.com/cap/drive-sharer` is added to the source automatically |
| `tailscale.com/cap/kubernetes` | Kubernetes operator | `impersonate` (e.g. `impersonate.groups`), `recorders` (one recorder tag per rule), `enforceRecorder` |
| `tailscale.com/cap/pam` | Tailscale PAM | `version`, `permissions`; `dst` is a PAM service such as `svc:production-database` |
| `tailscale.com/cap/secrets` | setec (secrets manager) | `action` (array: `get`, `put`, `info`, `activate`, `delete`), `secret` (array of path patterns) |
| `tailscale.com/cap/tailsql` | TailSQL (SQL playground) | Data sources. **Docs conflict**: the grants syntax page uses `dataSrc`, the app capabilities page uses `src` |
| `tailscale.com/cap/golink` | Golink (short links) | `admin` (boolean) |
| `tailscale.com/cap/relay` | Peer relays | Empty array `[]`, no parameters. `src` = devices to reach through the relay, `dst` = relay devices |

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

A user, group, tag, `user:*@<domain>`, or autogroup. **Cannot** be `*`. IP addresses and hosts are not allowed in SSH rules. (The syntax reference's `src` paragraph mistakenly talks about `dst`; the SSH feature page and targets page give the list above.)

### 5.4 SSH `dst`

**Highly restricted** compared to general `dst`:
- A tag (e.g. `tag:prod`)
- `autogroup:self` (if src contains only users/groups)
- A single named user (if src contains only the same user)
- **Cannot** specify a port (only port 22)
- **Cannot** use `*`

**Why**: Tailscale prevents one user from SSHing into another user's personal device. Only user-to-own-device, user-to-tagged-device, or tagged-to-tagged connections are allowed.

**Docs note**: the SSH feature page says `dst` "can be a user, tag, or autogroup", and the targets page marks plain users as not allowed in SSH `dst`. The rules above follow the syntax reference.

### 5.5 SSH `users`

| Value | Description |
|-------|-------------|
| `autogroup:nonroot` | Any user that is not `root` |
| `localpart:*@<domain>` | Map login email local-part to SSH username (Premium/Enterprise) |
| Specific username | e.g. `"ubuntu"`, `"root"` |
| (omitted) | Uses the local host's current user |

**Gotcha**: If `dst` is changed from `autogroup:self` to a tag, consider removing `autogroup:nonroot` from `users` — otherwise anyone in `src` can SSH as any non-root user on the destination.

### 5.6 SSH `checkPeriod`

Only with `action: "check"`; Premium and Enterprise plans. Specify minutes or hours (e.g. `"5m"`, `"20h"`). Min 1m, max 168h (1 week), default 12h. Special value: `"always"` (check every connection — can cause issues with automation like Ansible).

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

Tagged devices therefore **cannot** SSH into user-owned devices.

### 5.10 SSH Session Recording (`recorder`, `enforceRecorder`)

```json
{
  "action": "accept",
  "src": ["group:sre"],
  "dst": ["tag:prod"],
  "users": ["root"],
  "recorder": ["tag:session-recorder"],
  "enforceRecorder": false
}
```

- `recorder`: tags of recorder nodes. Sessions allowed by the rule are streamed to recorders with that tag. No `recorder` means no recording.
- `enforceRecorder` (default `false`, "fail open"): when `true`, sessions that should be recorded are refused, and active ones terminated, if no recorder is reachable ("fail closed").
- If two matching rules disagree on `enforceRecorder`, the rule listed first in the policy file wins.
- Tailscale automatically permits traffic from SSH servers to recorders (port 80); no extra access rule is needed. Restrict access to the recorder web UI (port 443) yourself.
- **Deprecation**: Tailscale no longer offers session recording through `tsrecorder`. Support for existing users is scheduled to end in August 2027; Tailscale recommends Tailscale PAM instead.

---

## 6. Auto Approvers

Auto approvers bypass admin console approval for advertised subnet routes, exit node advertisements, app connector routes (through `routes`), and Tailscale Service hosts.

### 6.1 Structure

```json
{
  "autoApprovers": {
    "routes": {
      "<CIDR>": ["<approvers>"]
    },
    "exitNode": ["<approvers>"],
    "services": {
      "svc:<service-name>": ["<approvers>"],
      "tag:<service-tag>": ["<approvers>"]
    }
  }
}
```

There is no `appConnectors` key. (An earlier version of this document showed one; it does not appear in Tailscale's docs.)

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

This auto-approves devices that **advertise** themselves as exit nodes. It does not let anyone **use** an exit node: exit node use requires a grant or ACL with `autogroup:internet` as the destination.

### 6.3a Tailscale Services

```json
"services": {
  "svc:web-server": ["tag:server"],
  "tag:prod-service": ["tag:prod-infra"]
}
```

Keys are a `svc:<name>` or a tag assigned to Services. Devices matching the approvers can advertise themselves as hosts for those Services and are approved automatically.

### 6.3b App Connectors

App connector routes are approved through `routes`, with the connector tag as the approver. The setup guide approves all routes for the connector tag:

```json
"routes": {
  "0.0.0.0/0": ["tag:github-app-connector"],
  "::/0": ["tag:github-app-connector"]
}
```

Custom apps always need this. Preset apps usually don't, but routes they learn through DNS do.

### 6.4 Approvers

Each approver can be: user email, `group:<name>`, an autogroup, or `tag:<name>` (syntax reference). **Docs conflict**: the targets page marks plain users as not allowed in auto approvers, while the syntax reference's example uses `alice@example.com`.

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
| `target` | Array of selectors | Devices the attributes apply to. The syntax reference lists `tag:<name>`, `<user>@<domain>`, `group:<name>`, and `*`; the targets page also allows role autogroups, `autogroup:tagged`, synced groups, IPs, and host aliases, but marks users as not allowed (a docs conflict) |
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

Optional keys include `routes`; preset apps use `presetAppID` instead of `domains`. Wildcard domains such as `*.example.com` do not include the parent domain. If GitOps or Terraform manages the policy, admin console edits to app connectors are overwritten on the next sync.

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

**Object syntax (recommended)**:

```json
{
  "postures": {
    "posture:latestMac": {
      "assertions": [
        "node:os IN ['macos']",
        "node:tsReleaseTrack == 'stable'",
        "node:tsVersion >= '1.40'"
      ],
      "onFailure": {
        "showAlerts": ["warn-in-client"],
        "endUserMessage": "You need to update your Mac. Contact IT helpdesk for detailed instructions."
      }
    },
    "posture:anyMac": {
      "assertions": ["node:os == 'macos'", "node:tsReleaseTrack == 'stable'"]
    }
  }
}
```

- Each posture name must start with `posture:`.
- `assertions`: list of assertion strings. All assertions within a posture must be true (AND logic).
- `onFailure` (optional): how admins and users are told a device fails the posture. Without it, nobody is alerted.
  - `showAlerts`: any of `badge-in-console` (badge in the admin console Machines view), `warn-in-client` (warning on the Tailscale client icon; the user opens the app to read the message), `notify-in-client` (push notification).
  - `endUserMessage`: message shown to the user of the failing device.
  - Console badges and client warnings/notifications are beta. The Windows client does not display them. Messages are only sent when the failing posture affects the device's connectivity.

**Legacy array syntax**:

```json
"posture:latestMac": [
  "node:os IN ['macos']",
  "node:tsReleaseTrack == 'stable'",
  "node:tsVersion >= '1.40'"
]
```

The array syntax keeps working indefinitely but gets **no new features** (including alerts). Both syntaxes can appear in the same `postures` block. Prefer the object syntax for new postures.

### 8.2 Operators

| Operator | Works with | Example |
|----------|-----------|---------|
| `==` | All types | `node:os == 'macos'` |
| `!=` | All types | `node:os != 'linux'` |
| `IN` | Lists of values (and IPs/CIDRs/IP sets for `ip:publicAddress`) | `node:os IN ['macos', 'linux', 'windows']` |
| `NOT IN` | Same as `IN` | `node:os NOT IN ['ios', 'android']` |
| `IS SET` | Any attribute (the docs list it without restriction) | `custom:tier IS SET` |
| `NOT SET` | Any attribute | `custom:tier NOT SET` |
| `>=`, `>`, `<=`, `<` | Numbers and the version attributes `node:osVersion`, `node:tsVersion` only | `node:tsVersion >= '1.40'`, `custom:edrScore >= 70` |

**Version comparison**: `node:osVersion` and `node:tsVersion` use a comparator that handles mixed numeric/non-numeric version strings.

**Unset attributes**: If an attribute referenced in a posture is unset for a device, the posture will **not** match that device, "irrespective of the operator used" — even a negative condition like `custom:tier != 'prod'` fails. The docs do not spell out how this interacts with `NOT SET`; test with `srcPostureAttrs` before relying on it.

### 8.3 Posture Attributes (Built-In)

| Key | Namespace | Description | Values |
|-----|-----------|-------------|--------|
| `node:os` | node | Operating system | `macos`, `windows`, `linux`, `ios`, `android`, `freebsd`, `openbsd`, `illumos`, `js` |
| `node:osVersion` | node | OS version | Quoted version string |
| `node:tsAutoUpdate` | node | Auto-updates enabled | `true`, `false` |
| `node:tsReleaseTrack` | node | Release track | `stable`, `unstable` |
| `node:tsStateEncrypted` | node | Client state encrypted at rest | `true`, `false` |
| `node:tsVersion` | node | Tailscale version | Quoted version string |
| `ip:country` | ip | Country of public IP (Standard, Premium, Enterprise) | ISO 3166-1 two-letter code (uppercase) |
| `ip:publicAddress` | ip | Public IP used to reach the control plane (beta; Standard, Premium, Enterprise) | IPv4/IPv6 address, CIDR, or IP set; only `IN`, `NOT IN`, `==`, `!=` |

**Note**: `node:tsAutoUpdate` is only `true` when Tailscale's built-in auto-update is enabled. External mechanisms (App Store, Google Play) result in `false`.

**Note**: the docs' examples quote booleans inconsistently (`node:tsAutoUpdate == true` vs `node:tsStateEncrypted == 'true'`). Evaluate a draft to confirm.

### 8.4 Posture Attributes (Custom)

- Set via the Posture Attributes API, or assigned during device provisioning with OAuth apps (Premium/Enterprise).
- Third-party integrations (CrowdStrike Falcon, etc.): Standard/Premium/Enterprise.
- Namespace `custom:` for user-defined attributes.
- Integration-specific namespaces (e.g. `falcon:ztaScore`).

```json
"posture:crowdstrike-trusted": {
  "assertions": ["falcon:ztaScore >= 70"]
}
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

Tailscale SSH Console connections are subject to posture conditions and report `node:os == 'js'`. To allow these through posture checks, include that value:

```json
"posture:allow-ssh-console": {
  "assertions": ["node:os IN ['macos', 'linux', 'js']"]
}
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
- **Shorthand**: `[]` (empty array) is equivalent to `["autogroup:admin"]`. It does not mean "nobody": Owners, Admins, and Network admins can apply any tag, even ones they don't own, so `[]` effectively limits the tag to those roles.
- You must define a tag in `tagOwners` before using it in access rules.

---

## 10. Groups

Named groups of users for use in access rules.

```json
{
  "groups": {
    "group:engineering": ["dave@example.com", "laura@example.com"],
    "group:sales": ["brad@example.com", "alice@example.com"]
  }
}
```

**Rules**:
- Group names must start with `group:`.
- Members are specified by full email/GitHub/Passkey.
- Groups **cannot** contain other groups (to avoid obfuscating membership).
- Groups synced from an identity provider (SCIM) are referenced as `group:<name>@<domain>` (e.g. `group:security-team@example.com`). You can use them in rules and `tagOwners` but cannot define or edit them in the `groups` section.

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

**Rules**:
- Hostnames cannot include the `@` character.
- Reference a host by its bare name in grants and ACLs, and as `host:<name>` inside `ipsets`. Hosts are policy aliases, not MagicDNS names.

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
| `add <target>` | Include all IPs from the target |
| `remove <target>` | Exclude all IPs from the target |

Operations are processed in order. Every entry needs an `add` or `remove` prefix unless the IP set only adds, in which case the prefix may be omitted (`"ipset:prod": ["192.0.2.0/24"]`). Tailscale converts each IP set to a list of IP address ranges.

**Targets**:

| Target | Syntax | Example |
|--------|--------|---------|
| CIDR | `<cidr>` | `192.0.2.0/24`, `2001:db8::/32` |
| IP address | `<ip>` | `192.0.2.33`, `2001:db8::` |
| IP address range | `<start>-<end>` | `192.0.2.50-192.0.2.100` |
| Host | `host:<name>` (from `hosts`, not MagicDNS) | `host:sql-server-1` |
| Autogroup | `autogroup:internet` only | `autogroup:internet` |
| IP set | `ipset:<name>` | `ipset:prod` |

**Limitations**:
- No tags, users, or groups (including synced groups) in IP sets.
- `autogroup:internet` is the only supported autogroup.
- No circular references between IP sets.

### 12.3 Using IP Sets

`ipset:<name>` can be used in grant `src`/`dst`, ACL `src`/`dst` (with a port, e.g. `"ipset:prod:*"`), inside other IP sets, and as a value for the beta `ip:publicAddress` posture attribute.

Example grant:

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
| `src` | A single identity string (not a list): email, `group:<name>`, `tag:<name>`, or host alias. Test runs from this device's perspective. |
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
| `dst` | List of one or more destinations — email, group, tag, or host alias |
| `accept` | SSH usernames that should be accepted without re-check (the syntax reference's wording says "disallow", which reads as a typo) |
| `check` | SSH usernames that should require re-authentication (same wording issue) |
| `deny` | SSH usernames that should be denied under all circumstances |
| `srcPostureAttrs` | Simulated posture attributes |

---

## 15. Network Policy Options

Rarely needed. Most networks should never specify these.

### 15.1 `derpMap`

Add custom DERP servers or disable Tailscale-provided DERP servers.

### 15.2 `disableIPv4`

**Not recommended** — Tailscale recommends the `disable-ipv4` node attribute (set in `nodeAttrs`) instead of this field.

Stops assigning Tailscale IPv4 addresses. All devices receive IPv6 only. Devices without IPv6 support become unreachable.

### 15.3 `OneCGNATRoute`

Controls how Tailscale clients generate routes:

| Value | Behavior |
|-------|----------|
| `""` or omitted | Default heuristics (fine-grained `/32` routes on most platforms; one `100.64/10` route on macOS v1.28+ unless other interfaces route that range) |
| `"mac-always"` | macOS clients always add one `100.64/10` route |
| `"mac-never"` | macOS clients always add fine-grained `/32` routes |

### 15.4 `randomizeClientPort`

**Only use as workaround after consulting Tailscale Support.**

Makes devices use a random port for WireGuard traffic instead of the default static port `41641`. Alternatively, use `nodeAttrs` with `randomize-client-port` for per-device control.

---

## 16. All Autogroups (Exhaustive)

| Autogroup | Allowed In | Description | Plan |
|-----------|-----------|-------------|------|
| `autogroup:internet` | `dst` only; inside `ipsets` | Public IPs (`0.0.0.0/0` and `2000::/3` minus private, CGNAT, link-local); access through exit nodes | All |
| `autogroup:self` | `dst` only (grants/ACLs, SSH) | User's own devices. NOT for tagged devices. `src` must be users, groups, or role autogroups | All |
| `autogroup:owner` | `src`, `dst`, `tagOwner`, `autoApprover` | Tailnet Owner | All |
| `autogroup:admin` | `src`, `dst`, `tagOwner`, `autoApprover` | Admin role | All |
| `autogroup:member` | `src`, `dst`, `tagOwner`, `autoApprover` | Direct tailnet members (incl. invited). NOT shared users. | All |
| `autogroup:tagged` | `src`, `dst`, `tagOwner`, `autoApprover`, `nodeAttrs`; SSH `dst` | Any tagged device | All |
| `autogroup:shared` | `src` only | Users who accepted a sharing invitation | All |
| `autogroup:danger-all` | grant/ACL `src` only | All sources including outside tailnet. Kept for backward compatibility; avoid | All |
| `autogroup:auditor` | `src`, `dst`, `tagOwner`, `autoApprover` | Auditor role | Standard+ |
| `autogroup:billing-admin` | `src`, `dst`, `tagOwner`, `autoApprover` | Billing admin role | Standard+ |
| `autogroup:it-admin` | `src`, `dst`, `tagOwner`, `autoApprover` | IT admin role | Standard+ |
| `autogroup:network-admin` | `src`, `dst`, `tagOwner`, `autoApprover` | Network admin role | Standard+ |
| `autogroup:nonroot` | SSH `users` only | Any non-root user | All |
| `user:*@<domain>` | `src`, `dst` (also SSH `src`) | All direct members with login in domain | All |
| `localpart:*@<domain>` | SSH `users` | Map email local-part to SSH user. The targets page says "SSH source"; the syntax reference and SSH docs put it in `users` | Premium+ |
| `autogroup:members` | (legacy) | Same as `autogroup:member` (use `member` instead) | All |

**Key restrictions**:
- `autogroup:self` only applies to **user-owned** devices — NOT tagged devices.
- Cannot use both `autogroup:member` and `autogroup:members` in the same policy file.
- Domain-based autogroups (`user:*@domain`) cannot use known shared domains (e.g. `gmail.com`).
- Domain wildcards (`*`) only match the entire local-part — `user:b*b@example.com` will NOT match `bob@example.com`.
- Domain-based autogroups only include direct tailnet members, not external invited users.
- If a tailnet uses domain aliases, you must specify each alias explicitly.
- `autogroup:member` and the role autogroups are only allowed in SSH `src` with restrictions; role autogroups are not allowed in SSH `dst`.

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
- `icmp` is **not** in this official alias table, but the docs use it in grants (`icmp:*`) and tests (`"proto": "icmp"`). Use it only in those documented forms.
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

### Geolocation (Standard+)

| Key | Type | Description | Example Values |
|-----|------|-------------|----------------|
| `ip:country` | String | Country of public IP | `'CA'`, `'US'`, `'GB'`, `'NL'` |
| `ip:publicAddress` | IP (beta) | Public IP used to reach the control plane | `ip:publicAddress IN ['203.0.113.0/24']` |

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
| `user:*@<domain>` | All direct members with a login in the domain | `user:*@example.com` |

### Prefix Summary

| Prefix | Section | Example |
|--------|---------|---------|
| `group:` | Groups (synced groups: `group:<name>@<domain>`) | `group:engineering` |
| `tag:` | Tags | `tag:production` |
| `host:` | Host alias, inside `ipsets` only | `host:sql-server-1` |
| `localpart:` | SSH `users` | `localpart:*@example.com` |
| `group://` | External tailnet group (alpha, grant `src`) | `group://partner/eng` |
| `tag://` | External tailnet tag (alpha, grant `src`) | `tag://partner/ci` |
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
    "ip": ["22"]
  }]
}
```

- If the ACL has no `proto`, keep the grant port bare (`"22"`); an ACL without `proto` covers TCP and UDP. If the ACL has `"proto": "tcp"`, use `"tcp:22"`.
- If one ACL gives different ports to different hosts (e.g. `tag:web:80` and `tag:dev:22`), write one grant per host.
- **SSH rules stay in the separate `ssh` section.** Keep them in their original format during migration.

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
    "posture:latestMac": {
      "assertions": [
        "node:os == 'macos'",
        "node:osVersion == '13.4.0'",
        "node:tsReleaseTrack == 'stable'"
      ]
    },
    "posture:trustedWindows": {
      "assertions": [
        "node:os == 'windows'",
        "node:tsVersion >= '1.40'"
      ]
    }
  },
  // A device matching either posture satisfies the default.
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

The full setup also needs a `tagOwners` entry for the connector tag, `autoApprovers.routes` for that tag (Section 6.3b), and minimal access to the connector tag for DNS discovery (`"ip": ["tcp:53", "udp:53"]`).

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
    "posture:stable": {
      "assertions": [
        "node:tsReleaseTrack == 'stable'",
        "node:tsVersion >= '1.40'"
      ]
    }
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
  "posture:stable": {"assertions": ["node:tsReleaseTrack == 'stable'"]}
}

// After (use the object syntax for new postures; legacy array postures can stay as they are)
"postures": {
  "posture:stable": {"assertions": ["node:tsReleaseTrack == 'stable'"]},
  "posture:macos-latest": {
    "assertions": [
      "node:os == 'macos'",
      "node:tsVersion >= '1.40'"
    ]
  },
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

// After (Grant) — remove action, split dst, move the port to ip
// (no proto in the ACL, so the port stays bare)
"grants": [
  { "src": ["group:eng"], "dst": ["tag:server"], "ip": ["22"] }
]
```

### 22.9 Handling Comments When Editing Programmatically

The Tailscale API accepts the policy file as HuJSON or JSON, and returns it as HuJSON or JSON depending on the `Accept` header. Only the HuJSON form can carry comments; a JSON round trip drops them. The docs do not promise byte-for-byte preservation of formatting, so don't rely on it.

For MCP servers: read the raw HuJSON, parse it with a HuJSON-aware parser, make minimal edits, and re-serialize with comments intact where possible.

---

## 23. Edge Cases & Gotchas

### 23.1 `autogroup:self` with Tags

`autogroup:self` does NOT match tagged devices. If your `dst` uses `autogroup:self`, it only matches user-owned (untagged) devices. This is a common source of confusion.

### 23.2 SSH `users` with Non-Self Destinations

If you change SSH `dst` from `autogroup:self` to a tag, reconsider `autogroup:nonroot` in `users` — it allows anyone in `src` to SSH as any non-root user on the destination.

### 23.3 Posture Unset Attributes

If a posture attribute is unset for a device, the posture will not match, **even with negative operators** like `!=`. This is counterintuitive — `custom:tier != 'prod'` won't match a device that has no `custom:tier` attribute at all. The docs say this holds "irrespective of the operator" but don't spell out `NOT SET`; test it.

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

Tailscale recommends the `disable-ipv4` attribute in `nodeAttrs` instead of the top-level `disableIPv4` field.

### 23.19 IPv6 Address Format in ACLs

IPv6 addresses in `dst` must follow the format `[1:2:3::4]:80` (bracketed).

### 23.20 Test Destinations Cannot Use CIDRs

In `tests`, you cannot use `192.168.1.0/24:22` — you must specify individual IPs or host aliases.

### 23.21 `autogroup:self` Source Restrictions

`autogroup:self` as a destination requires the `src` to be individual users, groups, or role autogroups (`autogroup:member`, `autogroup:admin`, ...). It cannot be paired with tags or `*`.

### 23.22 `autoApprovers.exitNode` Approves Advertising, Not Use

`exitNode` auto-approves devices that advertise as exit nodes. Using an exit node requires access to `autogroup:internet` in a grant or ACL.

### 23.23 SSH Needs Two Rules, and Tags Are Limited

A Tailscale SSH connection needs an `ssh` rule plus network access to port 22. Tagged sources can only SSH to other tagged devices and cannot use `check`.

### 23.24 No Bare Protocols in Grant `ip`

Grant `ip` entries are `*`, `<port>`, `<proto>:*`, or `<proto>:<port>`. Write `"icmp:*"`, not `"icmp"`. A bare port like `"443"` covers TCP, UDP, and ICMP.

### 23.25 Multiple Postures Are OR

`srcPosture` and `defaultSrcPosture` pass if the device matches **any** listed posture. Assertions inside a single posture are AND.

### 23.26 ACL-to-Grant Protocol Narrowing

An ACL without `proto` covers TCP and UDP. Converting `tag:web:443` to `"tcp:443"` silently drops UDP; use `"443"`.

---

## 24. External Tailnets (Declarative Node Sharing, Alpha)

Declarative node sharing gives users and tagged devices in other trusted tailnets access to resources in your tailnet, configured in policy. It is in **alpha**: enable it under General settings > Feature previews > Declarative Node Sharing > **Manage in policy**. It assumes, but does not establish, trust between the two tailnets' administrators. To share one machine with one outside user, use link-based node sharing instead.

It is double opt-in: access only takes effect once both tailnets' policy files are configured.

**Receiving tailnet** (whose users or devices get access):

```json
{
  "externalTailnets": {
    "<tailnet-alias>": {
      "externalID": "<sharing-tailnet-id>",
      "allowIncomingConnections": false,
      "allowExternalReferencesTo": ["group:<group-name>"]
    }
  },
  "groups": {
    "group:<group-name>": ["<user-1>", "<user-2>"]
  }
}
```

`allowExternalReferencesTo` lists the local groups or tags the other tailnet may reference. They must be defined in the policy file; synced groups (SCIM, Google) are not supported.

**Sharing tailnet** (which owns the resource):

```json
{
  "externalTailnets": {
    "<tailnet-alias>": {
      "externalID": "<receiving-tailnet-id>",
      "allowIncomingConnections": true
    }
  },
  "grants": [
    {
      "src": ["group://<tailnet-alias>/<group-name>"],
      "dst": ["<resource>"],
      "ip": ["*"]
    }
  ]
}
```

- `allowIncomingConnections` must be `true` on the sharing side.
- External selectors: `group://<tailnet-alias>/<group-name>` and `tag://<tailnet-alias>/<tag-name>`, used as the grant `src`.
- A grant that references a group or tag the receiving tailnet did not list in `allowExternalReferencesTo` is ignored.
- `externalID` is the other tailnet's ID (General settings > Unique IDs).
- Connect with the resource's Tailscale IP or fully qualified MagicDNS name (e.g. `machine.example.ts.net`); short names don't work across tailnets. There is no invitation or acceptance step.

**Limitations**: Tailscale Services and exit nodes cannot be shared, and subnet routes are not advertised to the other tailnet. Alpha limits: at most three entries in `externalTailnets`; sharing grants are dropped if either tailnet has more than 500 nodes.

---

*End of reference. This document was synthesized from the Tailscale tailnet policy file syntax reference, grants syntax reference, targets and selectors reference, device posture management docs, IP sets docs, Tailscale SSH and session recording docs, app connector and Tailscale Services docs, declarative node sharing docs, grant examples, the JWCC specification by Nigel Tao, and the tailscale/hujson GitHub repository.*
