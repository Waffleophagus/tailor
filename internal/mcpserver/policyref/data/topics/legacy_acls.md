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
- `src` is a list of source selectors.
- `dst` is a list of destination plus port selectors, such as `tag:web:443`, `db:22`, or `*:80,443`.
- `proto` is optional.
- ACLs cannot express grant-only features such as `app` capabilities or `via`.
- ACLs are additive.

Migration:
- ACL `src` usually maps to grant `src`.
- ACL destination host/tag usually maps to grant `dst`.
- ACL destination ports usually map to grant `ip`.
- Keep tests near migrations to prove old and new behavior match.

Common mistakes:
- Forgetting ports in ACL `dst`.
- Using IPv6 destinations without the required ACL formatting.
- Assuming ACLs can route with `via`.

Sources:
- https://tailscale.com/kb/1018/acls
- https://tailscale.com/kb/1324/grants
