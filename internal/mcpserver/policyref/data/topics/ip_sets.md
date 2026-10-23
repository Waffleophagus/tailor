# IP Sets

Use when:
- Grouping IPs, CIDRs, or hosts into a named network segment.
- Excluding addresses from a broader range with `remove`.
- Customizing `autogroup:internet` for exit node access.

Shape:
```json
"ipsets": {
  "ipset:dev": ["host:sql-server-1"],
  "ipset:prod": [
    "add 192.0.2.0/24",
    "add 2001:db8::/32",
    "remove 192.0.2.33",
    "remove ipset:dev"
  ]
}
```

Rules:
- Each entry is an operation followed by a target: `add <target>` or `remove <target>`. Operations run in order.
- If an IP set only adds, the `add` keyword may be omitted: `"ipset:prod": ["192.0.2.0/24"]`.
- Tailscale converts each IP set into a list of IP address ranges.

Targets:
| Target | Syntax | Example |
| --- | --- | --- |
| CIDR | `<cidr>` | `192.0.2.0/24`, `2001:db8::/32` |
| IP address | `<ip>` | `192.0.2.33` |
| IP range | `<start>-<end>` | `192.0.2.50-192.0.2.100` |
| Host | `host:<name>` (from `hosts`, not MagicDNS) | `host:sql-server-1` |
| Autogroup | only `autogroup:internet` | `autogroup:internet` |
| IP set | `ipset:<name>` | `ipset:prod` |

Limits:
- No tags, users, or groups (including synced groups) in IP sets.
- `autogroup:internet` is the only supported autogroup.
- No circular references between IP sets.

Where `ipset:<name>` can be used:
- Grant `src` and `dst`.
- ACL `src` and `dst`, with a port, e.g. `"ipset:prod:*"`.
- Inside other IP sets.
- As a value for the beta `ip:publicAddress` posture attribute.

Customize exit node traffic:
```json
"ipsets": {
  "ipset:internet": ["add autogroup:internet", "remove ipset:partner-net"],
  "ipset:partner-net": ["52.23.40.0/24"]
},
"grants": [
  {"src": ["group:sea"], "dst": ["ipset:internet"], "ip": ["*"], "via": ["tag:officerouter-sea"]}
]
```

Common mistakes:
- Putting `tag:`, `group:`, or user selectors in an IP set.
- Writing a bare host alias inside an IP set instead of `host:<name>`.
- Mixing bare targets with `remove` entries; once you remove anything, prefix every entry with `add` or `remove`.

Live Markdown: https://tailscale.com/docs/features/tailnet-policy-file/ip-sets.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/features/tailnet-policy-file/ip-sets
- https://tailscale.com/docs/reference/targets-and-selectors
- https://tailscale.com/docs/features/device-posture/postures-and-attributes
