# Tests

Use when:
- Adding validation tests to policy drafts.
- Proving intended allow or deny behavior.
- Testing SSH or posture-dependent access.

Network tests:
```json
"tests": [
  {
    "src": "dave@example.com",
    "srcPostureAttrs": {"node:os": "windows"},
    "proto": "tcp",
    "accept": ["tag:web:443", "svc:my-service:443"],
    "deny": ["192.0.2.3:22"]
  }
]
```
- Tests run on every policy change. A failing test rejects the whole update.
- `src` is one identity string, not a list: a user, group, tag, or host that maps to an IP.
- `accept` and `deny` hold `host:port` entries with a single numeric port. Hosts can be a Tailscale IP (IPv6 as `[1:2:3::4]:80`), a `hosts` alias, a user, a group, a tag, or `svc:<name>`.
- No CIDRs in destinations; list individual IPs or hosts instead.
- No `*` wildcards in `src`, `accept`, or `deny`.
- `proto` is optional. Without it, the test checks TCP or UDP access. For ICMP use `"proto": "icmp"` and port `0`, e.g. `"accept": ["tag:production:0"]`.
- `srcPostureAttrs` maps posture attributes to values. Only needed when rules use posture conditions.
- The legacy key `allow` still works in place of `accept`. Write `accept`.

SSH tests:
```json
"sshTests": [
  {
    "src": "dave@example.com",
    "dst": ["example-host-1"],
    "accept": ["dave"],
    "check": ["admin"],
    "deny": ["root"]
  }
]
```
- `src` is one identity string: a user, group, tag, or host.
- `dst` is a list of one or more destinations: users, groups, tags, or hosts.
- `accept`: host usernames expected to connect without a check. `check`: usernames expected to need a check. `deny`: usernames that must be refused. (The syntax reference's wording for `accept` and `check` says "disallow", which reads as a typo.)
- `srcPostureAttrs` works the same as in `tests`.

Useful patterns:
- Prove a subnet stays closed: `accept` the router's Tailscale IP and `deny` an IP inside the subnet.
- Prove exit node use stays closed: `deny` a public IP for users who should not have `autogroup:internet`.

Common mistakes:
- Testing only allow behavior and not the intended deny boundary.
- Using CIDR or `*` destinations, or a port range, in tests.
- Writing `sshTests` `dst` as a string instead of a list.
- Forgetting that grants are additive, so an unrelated broader rule may make a deny test fail.

Live Markdown: https://tailscale.com/docs/reference/syntax/policy-file.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/reference/targets-and-selectors
- https://tailscale.com/docs/features/tailscale-services
