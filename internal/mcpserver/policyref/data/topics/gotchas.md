# Gotchas

Use when:
- Running pre-flight checks before evaluating or staging.
- Checking common ACL and grant mistakes.
- Reviewing subtle selector semantics.

Checklist:
- `autogroup:self` only matches the source user's own user-owned devices, never tagged devices. As a `dst` it needs a `src` of users, groups, or role autogroups; not tags or `*`.
- Grants and ACLs are additive. A narrow rule does not override a broad rule.
- Application capabilities require matching network access.
- Grant `ip` takes `*`, `<port>`, `<proto>:*`, or `<proto>:<port>`. Write `"icmp:*"`, not `"icmp"`. A bare port such as `"443"` covers TCP, UDP, and ICMP.
- Several postures in `srcPosture` or `defaultSrcPosture` are OR: matching any one is enough. Assertions inside one posture are AND.
- `defaultSrcPosture` is replaced by an explicit `srcPosture`, not combined with it.
- An unset posture attribute makes the posture fail, even with `!=`.
- Shared-in devices and devices behind subnet routers bypass posture checks.
- `autoApprovers.exitNode` approves exit node advertisements. Exit node use is granted with `autogroup:internet`.
- Auto approvers are not retroactive.
- `tagOwners` `[]` means `autogroup:admin`, not nobody. Owners, Admins, and Network admins can apply any tag.
- Tailscale SSH needs an `ssh` rule and network access on port 22. Tagged sources cannot SSH to user-owned devices and cannot use `check`.
- CIDR grants allow traffic to a range but do not inject or advertise routes.
- Test destinations cannot use CIDRs or `*`. `sshTests` `dst` is a list.
- ACL IPv6 destinations need brackets: `[1:2:3::4]:80`.
- Groups cannot nest. IP sets cannot contain tags, users, or groups.
- Taildrop lets users share files between their own devices even when ACLs restrict access.
- ACLs cannot express `via` or `app`, and cannot restrict which exit node is used.
- When migrating an ACL without `proto`, keep grant ports bare (`"443"`, not `"tcp:443"`).

Pre-stage workflow:
- Fetch the current policy map.
- Read the specific reference topic for the section being edited.
- Add or update tests for the intended allow and deny behavior when practical.
- Evaluate the draft before staging.

Live Markdown: https://tailscale.com/docs/reference/syntax/policy-file.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/reference/syntax/grants
- https://tailscale.com/docs/reference/targets-and-selectors
- https://tailscale.com/docs/features/device-posture/postures-in-access-rules
- https://tailscale.com/docs/features/device-posture/postures-and-attributes
- https://tailscale.com/docs/reference/migrate-acls-grants
- https://tailscale.com/docs/features/tags
- https://tailscale.com/docs/features/tailnet-policy-file/ip-sets
