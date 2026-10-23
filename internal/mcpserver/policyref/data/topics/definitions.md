# Definitions

Use when:
- Editing `groups`, `tagOwners`, `hosts`, or `ipsets`.
- Creating reusable selectors.
- Checking ownership and nesting restrictions.

Groups:
- Shape: `"group:eng": ["alice@example.com", "bob@github"]`. Names start with `group:`.
- Members are user logins (email, `@github`, or `@passkey` forms).
- Groups cannot contain other groups.
- Synced identity-provider groups (`group:<name>@<domain>`) can be referenced in rules but are not edited in the policy file.

Tag owners:
- Shape: `"tag:web": ["group:ops", "alice@example.com"]`. Tag names start with `tag:`.
- An owner can be a user login, a group, an autogroup, or another tag.
- Define a tag in `tagOwners` before using it in rules.
- `[]` is shorthand for `["autogroup:admin"]`. It does not mean nobody: Owners, Admins, and Network admins can apply any tag, even ones they do not own.

Hosts:
- Shape: `"db": "100.64.0.10"` or `"office-net": "198.51.100.0/24"`.
- Host names cannot contain `@`.
- Reference a host by its bare name in grants and ACLs, and as `host:<name>` inside `ipsets`. Hosts are policy aliases, not MagicDNS names.

IP sets:
- `ipsets` maps `ipset:<name>` to `add`/`remove` operations, e.g. `"ipset:prod": ["add 192.0.2.0/24", "remove 192.0.2.33"]`.
- Full syntax, allowed targets, and limits are in the `ip_sets` topic.

Live Markdown: https://tailscale.com/docs/reference/syntax/policy-file.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/features/tags
- https://tailscale.com/docs/features/tailnet-policy-file/ip-sets
