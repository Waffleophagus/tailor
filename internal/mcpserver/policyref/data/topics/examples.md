# Examples

Use when:
- Looking up compact policy examples by scenario.
- Converting examples into draft edits.
- Adding tests alongside changes.

These are fragments. Every `tag:` used must be defined in `tagOwners`, and every `group:` in `groups` (unless synced).

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

Group to tag HTTPS, with a test:
```json
{
  "groups": {"group:eng": ["alice@example.com"]},
  "tagOwners": {"tag:web": ["group:eng"]},
  "grants": [
    {"src": ["group:eng"], "dst": ["tag:web"], "ip": ["tcp:443"]}
  ],
  "tests": [
    {"src": "alice@example.com", "accept": ["tag:web:443"], "deny": ["tag:web:22"]}
  ]
}
```

Exit nodes by location (use comes from `autogroup:internet`; `via` picks the exit nodes):
```json
{
  "grants": [
    {"src": ["group:sea"], "dst": ["autogroup:internet"], "ip": ["*"], "via": ["tag:exit-node-sea"]}
  ],
  "autoApprovers": {"exitNode": ["tag:exit-node-sea"]}
}
```

Posture-gated access (any listed posture is enough):
```json
{
  "postures": {
    "posture:trustedMac": {"assertions": ["node:os == 'macos'", "node:tsReleaseTrack == 'stable'"]},
    "posture:trustedWindows": {"assertions": ["node:os == 'windows'", "node:tsVersion >= '1.40'"]}
  },
  "grants": [
    {"src": ["group:eng"], "dst": ["tag:prod"], "ip": ["tcp:443"], "srcPosture": ["posture:trustedMac", "posture:trustedWindows"]}
  ]
}
```

Legacy ACL to grant (no `proto`, so the port stays bare):
```json
{
  "acls": [
    {"action": "accept", "src": ["group:eng"], "dst": ["tag:web:443"]}
  ],
  "grants": [
    {"src": ["group:eng"], "dst": ["tag:web"], "ip": ["443"]}
  ]
}
```

Tailscale Service access with host auto-approval:
```json
{
  "grants": [
    {"src": ["autogroup:member"], "dst": ["svc:web-server"], "ip": ["443"]}
  ],
  "autoApprovers": {"services": {"svc:web-server": ["tag:server"]}}
}
```

SSH to a tagged server (needs both the network grant and the SSH rule):
```json
{
  "grants": [
    {"src": ["group:sre"], "dst": ["tag:prod"], "ip": ["22"]}
  ],
  "ssh": [
    {"action": "check", "src": ["group:sre"], "dst": ["tag:prod"], "users": ["ubuntu"], "checkPeriod": "1h"}
  ],
  "sshTests": [
    {"src": "alice@example.com", "dst": ["tag:prod"], "check": ["ubuntu"], "deny": ["root"]}
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

Live Markdown: https://tailscale.com/docs/reference/examples/grants.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/reference/examples/grants
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/features/access-control/grants/grants-via
- https://tailscale.com/docs/reference/migrate-acls-grants
- https://tailscale.com/docs/features/tailscale-services
- https://tailscale.com/docs/features/tailscale-ssh
