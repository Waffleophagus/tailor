# Gotchas

Use when:
- Running pre-flight checks before evaluating or staging.
- Checking common ACL and grant mistakes.
- Reviewing subtle selector semantics.

Checklist:
- `autogroup:self` applies to user-owned devices and does not match tagged devices.
- Grants are additive. A narrow grant does not override a broad grant.
- Application capabilities require matching network access.
- `defaultSrcPosture` is replacing, not additive with explicit `srcPosture`.
- Auto approvers are not retroactive.
- Empty arrays in `tagOwners` do not necessarily mean nobody can use the tag.
- CIDR grants allow access to a destination range but do not inject or advertise routes.
- Test destinations cannot use CIDRs.
- ACL IPv6 destinations need the ACL-specific formatting.
- Groups cannot nest.
- Taildrop has ACL precedence behavior; check before assuming file transfer follows ordinary access edges.
- ACLs cannot express `via` or `app`.

Pre-stage workflow:
- Fetch the current policy map.
- Read the specific reference topic for the section being edited.
- Add or update tests for the intended allow and deny behavior when practical.
- Evaluate the draft before staging.

Sources:
- https://tailscale.com/kb/1018/acls
- https://tailscale.com/kb/1324/grants
- https://tailscale.com/kb/1337/policy-syntax
