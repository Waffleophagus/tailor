# Posture

Use when:
- Adding device posture controls.
- Reasoning about `srcPosture` or `defaultSrcPosture`.
- Checking unset attribute behavior.

Define postures (object syntax, recommended):
```json
"postures": {
  "posture:latestMac": {
    "assertions": [
      "node:os IN ['macos']",
      "node:tsReleaseTrack == 'stable'",
      "node:tsVersion >= '1.40'"
    ],
    "onFailure": {
      "showAlerts": ["warn-in-client"],
      "endUserMessage": "Update your Mac. Contact the IT helpdesk for help."
    }
  }
}
```
- Names start with `posture:`.
- `assertions`: a device matches the posture only if every assertion holds (AND).
- `onFailure` (optional): `showAlerts` lists any of `badge-in-console` (badge in the admin console), `warn-in-client` (warning on the client icon), `notify-in-client` (push notification). `endUserMessage` is the text shown to the user. These alerts are beta, the Windows client does not show warnings or notifications, and messages only go to devices whose access is affected.
- Legacy array syntax, `"posture:x": ["assertion", ...]`, works indefinitely but gets no new features, including alerts. Both syntaxes can coexist in one `postures` block. Use the object syntax for new postures.

Apply postures:
- `srcPosture` on a grant, ACL, or SSH rule. With several postures listed, the condition passes if the device matches ANY one of them (OR).
- `defaultSrcPosture` (top level) lists postures for every access rule without its own `srcPosture`. It is also ANY-of.
- A rule's own `srcPosture` replaces the default; it does not add to it. A rule can therefore be looser than the default.
- Defining a posture alone enforces nothing.

Assertions are `<attribute> <operator> <value>`, with strings in single quotes.

Operators: `==`, `!=`, `IN`, `NOT IN` (list such as `['a', 'b']`), `IS SET`, `NOT SET`, and `<`, `<=`, `>=`, `>` (numbers and the version attributes `node:osVersion` and `node:tsVersion` only).

Attributes:
| Key | Values | Plan |
| --- | --- | --- |
| `node:os` | `macos`, `windows`, `linux`, `ios`, `android`, `freebsd`, `openbsd`, `illumos`, `js` | All |
| `node:osVersion` | Quoted version, e.g. `'13.4.0'` | All |
| `node:tsAutoUpdate` | `true`/`false`. True only with Tailscale's built-in auto-update; app-store updates report false | All |
| `node:tsReleaseTrack` | `'stable'`, `'unstable'` | All |
| `node:tsStateEncrypted` | `true`/`false` (client state encrypted at rest) | All |
| `node:tsVersion` | Quoted version, e.g. `'1.42.2'` | All |
| `ip:country` | Uppercase ISO 3166-1 two-letter code, e.g. `ip:country IN ['CA', 'US']` | Standard, Premium, Enterprise |
| `ip:publicAddress` (beta) | IPv4/IPv6 address, CIDR, or IP set; only `IN`, `NOT IN`, `==`, `!=` | Standard, Premium, Enterprise |
| `custom:<name>` | Set with the API or device provisioning | Premium, Enterprise |
| Integration keys (e.g. `falcon:ztaScore`) | Synced from EDR/MDM integrations | Standard, Premium, Enterprise |
- The docs quote booleans inconsistently (`node:tsAutoUpdate == true` but `node:tsStateEncrypted == 'true'`). Evaluate the draft to confirm.

Unset attributes:
- If a device does not have an attribute the posture references, the posture does not match that device, whatever the operator. For example, `custom:tier != 'prod'` fails on a device with no `custom:tier`.
- The docs do not say how this interacts with `NOT SET`. Test it with `srcPostureAttrs` before relying on it.

Who is not posture-checked:
- Posture only applies to source devices in your own tailnet. Devices shared in from other tailnets and devices behind subnet routers bypass posture and get access if the IP-based conditions match.
- Tailscale SSH Console sessions are posture-checked and report `node:os == 'js'`.

Live Markdown: https://tailscale.com/docs/features/device-posture/postures-and-attributes.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer). The attribute tables are missing from that Markdown, so check the HTML page for them.

Sources:
- https://tailscale.com/docs/features/device-posture/postures-and-attributes
- https://tailscale.com/docs/features/device-posture/postures-in-access-rules
- https://tailscale.com/docs/features/device-posture/monitoring-postures
- https://tailscale.com/docs/features/device-posture
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/integrations/crowdstrike-zta
