# SSH

Use when:
- Adding or modifying Tailscale SSH rules.
- Choosing `accept` versus `check`.
- Configuring SSH session recording.
- Writing `sshTests`.

Two rules are required for a connection:
- An `ssh` rule covering the source, destination, and host user.
- Network access from the source to the destination on port 22, from a grant or ACL.

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
Optional fields: `checkPeriod`, `acceptEnv`, `srcPosture`, `recorder`, `enforceRecorder`.

Fields:
- `action`: `accept` lets already-authenticated users connect. `check` makes users re-authenticate according to `checkPeriod`. If both a `check` and an `accept` rule match, the `check` rule applies, and only its `users` count.
- `src`: user, group, tag, `user:*@<domain>`, or autogroup. Not `*`.
- `dst`: a tag; `autogroup:self` if `src` contains only users or groups; or a single named user if `src` is only that same user. Not `*`, and no port (always 22). The SSH feature page phrases this more loosely; follow the syntax reference.
- `users`: host account names (`root`, `ubuntu`), `autogroup:nonroot`, or `localpart:*@<domain>` (Premium, Enterprise). Accounts must already exist on the host. If `users` is omitted, Tailscale tries the connecting user's local username.
- `checkPeriod` (only with `check`; Premium and Enterprise plans): minutes or hours from `1m` to `168h`. The default is `12h`. `"always"` checks every connection, which can break automation such as Ansible.
- `acceptEnv`: environment variable names clients may send with `SendEnv`/`SetEnv`. Supports `*` and `?`. The host needs Tailscale v1.76.0 or later.
- `srcPosture`: posture conditions on the source device (see `posture`).

Allowed connection types (everything else is rejected):
- A user to their own devices, as any host user including `root`.
- A user to a tagged device, including a tagged device shared with them.
- A tagged device to another tagged device. Tagged sources cannot SSH to user-owned devices, and a rule from a tagged source cannot use `check`.

Session recording:
- `"recorder": ["tag:<recorder-tag>"]` streams sessions allowed by that rule to recorder nodes with that tag. Without `recorder`, sessions are not recorded.
- `"enforceRecorder": true` fails closed: if no recorder is reachable, new sessions are refused and active ones end. The default is `false` (fail open). If matching rules disagree, the rule listed first in the policy file wins.
- Tailscale no longer offers session recording through `tsrecorder`. Support for existing users is scheduled to end in August 2027, and Tailscale points to Tailscale PAM instead. Keep existing `recorder` fields intact, and confirm with the owner before adding new ones.

Common mistakes:
- Forgetting the port 22 network rule.
- Putting `autogroup:nonroot` in `src` or `dst`; it belongs in `users`.
- Changing `dst` from `autogroup:self` to a tag while keeping `autogroup:nonroot`. Everyone in `src` can then log in as any non-root user on that tag.
- `autogroup:member` in `src` also covers external invited users when the destination is shared with them.
- Using `check` on a rule whose source is a tag.
- Tailscale SSH Console sessions are posture-checked; allow them with `node:os == 'js'` if postures apply.

Live Markdown: https://tailscale.com/docs/features/tailscale-ssh.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/features/tailscale-ssh
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/reference/targets-and-selectors
- https://tailscale.com/docs/features/tailscale-ssh/tailscale-ssh-session-recording
- https://tailscale.com/docs/features/tsrecorder
- https://tailscale.com/docs/features/device-posture/postures-and-attributes
