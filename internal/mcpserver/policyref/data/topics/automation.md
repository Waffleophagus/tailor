# Automation

Use when:
- Auto-approving subnet routes, exit node advertisements, app connector routes, or Service hosts.
- Managing `nodeAttrs`.
- Enabling Funnel or client network options.

Auto approvers:
```json
"autoApprovers": {
  "routes": {"192.0.2.0/24": ["group:eng", "tag:router"]},
  "exitNode": ["tag:exit"],
  "services": {
    "svc:web-server": ["tag:server"],
    "tag:prod-service": ["tag:prod-infra"]
  }
}
```
- `routes`: maps a CIDR to approvers who can advertise that route, or a subnet of it, without admin approval.
- `exitNode`: approvers whose devices can advertise themselves as exit nodes without admin approval. It does not let anyone use an exit node. Exit node use comes from a grant or ACL with `dst` `autogroup:internet`.
- `services`: maps a `svc:<name>`, or a tag assigned to Services, to approvers whose devices can become hosts for those Tailscale Services without admin approval.
- Approvers are user logins, groups, autogroups, or tags. (The targets page marks plain users as not allowed here, while the syntax reference shows `alice@example.com`.) Prefer tags: a user-approved route stops being advertised if another user re-authenticates the device or the approving user is suspended or deleted.
- Not retroactive: approval only happens when Tailscale first receives an advertisement. To approve an existing route, remove it from the router and advertise it again.
- App connectors have no dedicated `autoApprovers` key. Approve their routes under `routes` with the connector tag; the setup guide uses `"0.0.0.0/0"` and `"::/0"`. Custom apps always need this. Preset apps usually do not, except for routes learned through DNS.

Node attributes:
```json
"nodeAttrs": [
  {"target": ["autogroup:member"], "attr": ["funnel"]},
  {"target": ["tag:server"], "attr": ["nextdns:abc123", "nextdns:no-device-info"]}
]
```
- `target`: a tag, user, group, or `*` per the syntax reference; the targets page also lists role autogroups, `autogroup:tagged`, synced groups, IPs, and hosts.
- `attr` holds flag strings. Documented flags: `funnel`, `nextdns:<config-id>`, `nextdns:no-device-info`, `randomize-client-port`, and `disable-ipv4` (preferred over top-level `disableIPv4`).
- `nodeAttrs` flags are separate from posture attributes.
- `app` holds node-level capabilities. App connectors are configured this way:
```json
{"target": ["*"], "app": {"tailscale.com/app-connectors": [
  {"name": "GitHub", "connectors": ["tag:github-app-connector"], "domains": ["github.com", "*.github.com"]}
]}}
```
  Optional keys include `routes`, and preset apps use `presetAppID`. `*.example.com` does not include `example.com` itself. If GitOps or Terraform manages the policy, app connector edits made in the admin console are overwritten on the next sync.

App connector checklist: a `tagOwners` entry for the connector tag; `autoApprovers.routes` for that tag; a grant to the advertised routes (for example `autogroup:internet`, which also allows exit node use, optionally with `via` set to the connector tag); minimal access to the connector tag for DNS discovery (`"ip": ["tcp:53", "udp:53"]`); and the `nodeAttrs` `app` entry.

Network-wide options (rarely needed):
- `randomizeClientPort`: `true` makes devices use a random WireGuard port instead of 41641. Only use it after consulting Tailscale Support; the `randomize-client-port` node attribute scopes it per device.
- `disableIPv4`: `true` stops assigning Tailscale IPv4 addresses. Tailscale recommends the `disable-ipv4` node attribute instead.
- `OneCGNATRoute`: `""` (default heuristics), `"mac-always"`, or `"mac-never"`.
- `derpMap`: adds custom DERP servers or disables Tailscale's.

Live Markdown: https://tailscale.com/docs/reference/syntax/policy-file.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/features/app-connectors/how-to/setup
- https://tailscale.com/docs/features/app-connectors
- https://tailscale.com/docs/features/tailscale-services
- https://tailscale.com/docs/features/tailscale-funnel
- https://tailscale.com/docs/reference/targets-and-selectors
