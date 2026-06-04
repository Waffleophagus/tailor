# SSH

Use when:
- Adding or modifying Tailscale SSH rules.
- Choosing `accept` versus `check`.
- Writing `sshTests`.

SSH rule shape:
```json
{
  "action": "check",
  "src": ["group:eng"],
  "dst": ["autogroup:self"],
  "users": ["autogroup:nonroot"],
  "checkPeriod": "12h"
}
```

Rules:
- `action` is `accept` for no periodic re-auth or `check` for periodic user check.
- `src` selects who may initiate SSH.
- `dst` selects target devices.
- `users` selects Unix usernames allowed on the destination.
- `checkPeriod` applies to `check`.
- `acceptEnv` can allow specific environment variables.
- `srcPosture` can require posture checks.

Common mistakes:
- `autogroup:nonroot` belongs in `users`, not `src` or `dst`.
- SSH rules are separate from network-layer grants and ACLs.
- Use `sshTests` for expected allow/deny behavior.

Sources:
- https://tailscale.com/kb/1193/tailscale-ssh
- https://tailscale.com/kb/1337/policy-syntax
