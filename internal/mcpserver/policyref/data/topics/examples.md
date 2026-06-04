# Examples

Use when:
- Looking up compact policy examples by scenario.
- Converting examples into draft edits.
- Adding tests alongside changes.

Allow all with grants:
```json
{
  "grants": [
    {"src": ["*"], "dst": ["*"], "ip": ["*"]}
  ]
}
```

Self-access:
```json
{
  "grants": [
    {"src": ["autogroup:member"], "dst": ["autogroup:self"], "ip": ["*"]}
  ]
}
```

Group to tag HTTPS:
```json
{
  "groups": {"group:eng": ["alice@example.com"]},
  "grants": [
    {"src": ["group:eng"], "dst": ["tag:web"], "ip": ["tcp:443"]}
  ]
}
```

Posture-gated access:
```json
{
  "postures": {"posture:trusted": ["node:os == 'macos'"]},
  "grants": [
    {"src": ["group:eng"], "dst": ["tag:prod"], "ip": ["tcp:443"], "srcPosture": ["posture:trusted"]}
  ]
}
```

Legacy ACL to grant:
```json
{
  "acls": [
    {"action": "accept", "src": ["group:eng"], "dst": ["tag:web:443"]}
  ],
  "grants": [
    {"src": ["group:eng"], "dst": ["tag:web"], "ip": ["tcp:443"]}
  ]
}
```

SSH self non-root with check:
```json
{
  "ssh": [
    {"action": "check", "src": ["autogroup:member"], "dst": ["autogroup:self"], "users": ["autogroup:nonroot"]}
  ]
}
```

Sources:
- https://tailscale.com/kb/1018/acls
- https://tailscale.com/kb/1324/grants
- https://tailscale.com/kb/1193/tailscale-ssh
