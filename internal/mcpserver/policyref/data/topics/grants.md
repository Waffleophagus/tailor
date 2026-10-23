# Grants

Use when:
- Adding or changing modern network access rules.
- Adding application capabilities.
- Routing traffic through selected tagged routers with `via`.

Rules:
- A grant needs `src`, `dst`, and at least one of `ip` or `app` (or both). `srcPosture` and `via` are optional.
- Every field is an array except `app`, which maps capability names to arrays of parameter objects.
- Grants have an implied accept and are unioned. If several grants match, their capabilities add up. A more specific grant never overrides a broader one.
- Several selectors in one `src` or `dst` array are a union.
- App capabilities only apply if the source also has network-level access to the destination.

Shape:
```json
{
  "src": ["group:eng"],
  "dst": ["tag:web"],
  "ip": ["tcp:443"]
}
```

`ip` entries (the only documented forms):
| Form | Meaning | Example |
| --- | --- | --- |
| `*` | All ports and protocols (TCP, UDP, ICMP) | `"*"` |
| `<port>` or `<low>-<high>` | Port or range; implies TCP, UDP, and ICMP | `"443"`, `"80-443"` |
| `<proto>:*` | Every port of one protocol; use for portless protocols | `"icmp:*"`, `"sctp:*"` |
| `<proto>:<port>` | Protocol plus port or range | `"tcp:443"`, `"tcp:80-443"` |
- A bare protocol name such as `"icmp"` is not a documented form. Write `"icmp:*"`.
- `<proto>` is an IANA protocol number (1-255) or a named alias. Aliases are listed in the `selectors` topic.
- Only TCP, UDP, and SCTP take ports. Other protocols only accept `*` as the port.

Destination notes:
- `svc:<name>` selects a Tailscale Service (destination only).
- `autogroup:internet` grants internet access through exit nodes. Only sources with access to it can use exit nodes.
- `autogroup:self` requires a `src` made of users, groups, or role autogroups such as `autogroup:member`.
- A CIDR destination permits traffic. It does not inject or approve routes; a subnet router must advertise them and they must be approved.

`app` (application capabilities):
- Shape: `"app": {"<domain>/<capability>": [{...params}]}`. The policy engine only checks that params are valid JSON; the application defines and enforces them.
- `tailscale.com` and `tailscale.io` are reserved for Tailscale. Use a domain you control for custom capabilities.
- App grants cannot target `autogroup:internet`, `autogroup:shared`, `autogroup:danger-all`, or admin autogroups such as `autogroup:admin`.
- First-party capabilities (check the feature page for the full parameter schema):
  - `tailscale.com/cap/drive` (Taildrive): `{"shares": ["*"], "access": "ro"}` or `"rw"`.
  - `tailscale.com/cap/kubernetes`: `impersonate`, `recorders`, `enforceRecorder`.
  - `tailscale.com/cap/pam` (Tailscale PAM): `dst` is a PAM service such as `svc:production-database`; params include `version` and `permissions`.
  - `tailscale.com/cap/secrets` (setec): `action` (`get`, `put`, `info`, `activate`, `delete`) and `secret` path patterns.
  - `tailscale.com/cap/relay` (peer relays): value `[]`. `src` is the devices to reach through the relay, `dst` is the relay devices.
  - `tailscale.com/cap/golink`: `{"admin": true}`.
  - `tailscale.com/cap/tailsql`: the docs disagree on the parameter name (`dataSrc` on the grants syntax page, `src` on the app capabilities page). Check the TailSQL version in use.

`via`:
- Accepts only tags. It names the exit nodes, subnet routers, or app connectors traffic must use, e.g. `dst` `autogroup:internet` with `via` `["tag:exit-sea"]`.
- Only routers the source can access are candidates.
- Omitting `via`, `[]`, or `null` allows any router.

Common mistakes:
- Creating an app-only grant without network access.
- Expecting a narrower grant to override a broader one.
- Using a non-tag selector in `via`.
- Writing `"icmp"` instead of `"icmp:*"` in `ip`.
- Treating a CIDR destination grant as route advertisement or route injection.

Live Markdown: https://tailscale.com/docs/reference/syntax/grants.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/reference/syntax/grants
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/features/access-control/grants/grants-app-capabilities
- https://tailscale.com/docs/features/access-control/grants/grants-via
- https://tailscale.com/docs/features/peer-relay
- https://tailscale.com/docs/reference/targets-and-selectors
