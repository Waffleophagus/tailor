# Selectors

Use when:
- Choosing legal `src` or `dst` selectors.
- Editing tag owners, auto approvers, SSH, or tests.
- Checking autogroup behavior.

Selector forms:
- Users: `alice@example.com` (email login), `alice@github`, `alice@passkey`.
- Domain users: `user:*@example.com` matches direct tailnet members (including invited users) whose login is in that domain. Allowed in `src` and `dst`. Known shared domains such as `gmail.com` are rejected, partial wildcards such as `user:b*b@example.com` do not match, domain aliases must each be listed, and external invited users are excluded.
- Groups: `group:<name>` from `groups`. Synced identity-provider groups are referenced as `group:<name>@<domain>`; you can use them but not edit them in the policy.
- Tags: `tag:<name>`. Define it in `tagOwners` first. Tags are not joined: a rule cannot require a device to have both `tag:a` and `tag:b`; use a composite tag.
- Hosts: a bare alias from `hosts` in grants and ACLs; `host:<alias>` inside `ipsets`.
- IPs and CIDRs: `100.101.102.103`, `192.0.2.0/24`. Ranges such as `192.0.2.4-192.0.2.12` are only allowed inside `ipsets`.
- `ipset:<name>`: grant and ACL `src`/`dst`, and inside other IP sets.
- `svc:<name>`: a Tailscale Service. Destination only (documented for grants), plus test destinations and `autoApprovers.services` keys.
- `posture:<name>`: only in `srcPosture` and `defaultSrcPosture`.
- External tailnets (alpha): `group://<alias>/<group>` and `tag://<alias>/<tag>` as grant `src`. See `external_tailnets`.

Autogroups:
| Autogroup | Allowed in | Notes |
| --- | --- | --- |
| `autogroup:member` | src, dst, tagOwners, autoApprovers, nodeAttrs; SSH src (restricted) | Direct members, including invited users; not shared users |
| `autogroup:owner`, `admin`, `network-admin`, `it-admin`, `billing-admin`, `auditor` | src, dst, tagOwners, autoApprovers, nodeAttrs | Role autogroups. The last four need Standard, Premium, or Enterprise |
| `autogroup:tagged` | src, dst, tagOwners, autoApprovers, nodeAttrs; SSH dst; SSH src (restricted) | Any device with at least one tag |
| `autogroup:self` | dst only (grants, ACLs, SSH) | The source user's own user-owned devices. Never matches tagged devices. `src` must be users, groups, or role autogroups |
| `autogroup:internet` | dst only; inside `ipsets` | Public IPs: `0.0.0.0/0` and `2000::/3` minus private, CGNAT `100.64.0.0/10`, and link-local ranges. Grants exit node use |
| `autogroup:shared` | src only | Users who accepted a node-sharing invitation |
| `autogroup:danger-all` | grant/ACL src only | All sources, even outside the tailnet. Kept for backward compatibility; avoid |
| `autogroup:nonroot` | SSH `users` only | Any host user except `root` |
| `localpart:*@<domain>` | SSH `users` (Premium, Enterprise) | Host user matching the login's local part. The targets page calls it an SSH source; the syntax reference and SSH docs put it in `users` |
- `autogroup:members` is a legacy alias for `autogroup:member`. Do not use both in one policy.

Rules:
- `*` in grant or ACL `src`/`dst` matches all Tailscale IPs and approved subnet routes. It is not allowed in SSH `src`/`dst` or in tests.
- Synced groups cannot be used inside `ipsets` or `via`. Only tags are allowed in `via`.
- `nodeAttrs` targets: the syntax reference allows a tag, user, group, or `*`; the targets page marks users as not allowed. Prefer tags or groups.

Protocol aliases (grant `ip` and ACL `proto`):
| Alias | IANA number |
| --- | --- |
| `igmp` | 2 |
| `ipv4`, `ip-in-ip` | 4 |
| `tcp` | 6 |
| `egp` | 8 |
| `igp` | 9 |
| `udp` | 17 |
| `gre` | 47 |
| `esp` | 50 |
| `ah` | 51 |
| `sctp` | 132 |
- Any IANA number from 1 to 255 also works as a string, e.g. `"16"`.
- `icmp` is not in the official alias table, but the docs use it in `icmp:*` (grants) and `"proto": "icmp"` (tests). Use it only in those forms.
- Only TCP, UDP, and SCTP take ports.

Live Markdown: https://tailscale.com/docs/reference/targets-and-selectors.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/reference/targets-and-selectors
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/reference/syntax/grants
- https://tailscale.com/docs/features/tags
- https://tailscale.com/docs/features/tailnet-policy-file/ip-sets
