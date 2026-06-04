# Selectors

Use when:
- Choosing legal `src` or `dst` selectors.
- Editing tag owners, auto approvers, SSH, or tests.
- Checking autogroup behavior.

Selector forms:
- Users: `alice@example.com`, domain users, or groups depending on section.
- Groups: `group:name`.
- Tags: `tag:name`.
- Autogroups: `autogroup:member`, `autogroup:admin`, `autogroup:owner`, `autogroup:self`, `autogroup:internet`, and other feature-specific autogroups.
- Hosts: names from `hosts`.
- IP sets: `ipset:name`.
- Services: `svc:name` where supported.
- Postures: `posture:name` in posture-related contexts.

Rules:
- `src` selectors and `dst` selectors have different allowed forms. Check the target section before editing.
- `autogroup:self` is a destination selector for user-owned devices. It does not match tagged devices.
- `tag:` selectors represent tagged devices, not tag owners.
- `group:` selectors cannot contain nested groups.
- Test destinations are more restricted than policy destinations and cannot use CIDRs.

Protocol aliases:
- Common aliases include `tcp`, `udp`, `icmp`, `sctp`, and protocol numbers where supported by the section.

Sources:
- https://tailscale.com/kb/1337/policy-syntax
- https://tailscale.com/kb/1018/acls
