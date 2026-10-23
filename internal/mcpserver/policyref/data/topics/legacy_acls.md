# Legacy ACLs

Use when:
- Understanding existing `acls`.
- Migrating ACLs to grants.
- Editing legacy ACL destination ports.

ACL shape:
```json
{
  "action": "accept",
  "src": ["group:eng"],
  "dst": ["tag:web:443"],
  "proto": "tcp"
}
```

Rules:
- `action` must be `accept`.
- `src` is a list of sources: `*`, users, groups, Tailscale IPs, CIDRs, hosts, tags, autogroups, `autogroup:danger-all`, `ipset:<name>`.
- `dst` is a list of `<host>:<ports>`. Ports are `*`, `22`, `80,443`, or `1000-2000`. Hosts include `*`, users, groups, Tailscale IPs, `hosts` aliases, CIDRs, tags, `autogroup:internet`, `autogroup:self`, role autogroups, and `ipset:<name>` (e.g. `ipset:prod:*`).
- IPv6 addresses with a port use brackets: `[1:2:3::4]:80`.
- `proto` is optional. Without it the rule covers TCP and UDP. It takes an IANA number or alias (see `selectors`); only TCP, UDP, and SCTP support ports.
- If traffic is allowed between two IPs, ICMP is also allowed.
- `srcPosture` is optional and works as in grants.
- Legacy field names `users` (for `src`) and `ports` (for `dst`) still work. Write `src` and `dst`.
- ACLs cannot express `app` capabilities or `via`. You cannot restrict use of specific exit nodes with ACLs.
- ACLs are additive and can coexist with grants.
- For 4via6 subnet routers, use the IPv6 address or CIDR as the destination.

Migration:
- Remove `action`. `src` and `srcPosture` stay the same.
- Split `dst`: the host goes to grant `dst`, the ports go to grant `ip`.
- If the ACL has no `proto`, keep the grant port bare: `tag:web:443` becomes `"ip": ["443"]`.
- If the ACL has `"proto": "tcp"`, prefix the port: `"ip": ["tcp:443"]`.
- If one ACL gives different ports to different hosts, write one grant per host.
- Keep tests near migrations to prove old and new behavior match.

Common mistakes:
- Forgetting ports in ACL `dst`.
- Adding `tcp:` during migration when the ACL had no `proto`. This silently drops UDP.
- Using IPv6 destinations without brackets.
- Assuming ACLs can route with `via`.

Live Markdown: https://tailscale.com/docs/reference/migrate-acls-grants.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/reference/migrate-acls-grants
- https://tailscale.com/docs/features/access-control/acls
- https://tailscale.com/docs/reference/grants-vs-acls
