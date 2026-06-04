# Definitions

Use when:
- Editing `groups`, `tagOwners`, `hosts`, or `ipsets`.
- Creating reusable selectors.
- Checking ownership and nesting restrictions.

Groups:
- Shape: `"group:eng": ["alice@example.com", "bob@example.com"]`.
- Groups contain users, not other groups.

Tag owners:
- Shape: `"tag:web": ["group:ops"]`.
- Owners can be users, groups, tags, or autogroups depending on policy.
- An empty owner array does not mean nobody can own the tag; it can mean the tag is unowned/restricted by existing assignment behavior. Check intent before changing.

Hosts:
- Shape: `"db": "100.64.0.10"` or a CIDR where supported.
- Hosts create reusable destination names.

IP sets:
- Shape: `"ipset:corp": ["192.0.2.0/24"]`.
- Composition can use `add` and `remove` entries for include/exclude behavior.
- Use `ipset:` selectors where the section permits them.

Sources:
- https://tailscale.com/kb/1337/policy-syntax
- https://tailscale.com/kb/1068/acl-tags
