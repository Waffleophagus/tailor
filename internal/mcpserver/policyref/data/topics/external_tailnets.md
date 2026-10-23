# External Tailnets

Use when:
- Sharing resources with another trusted tailnet through policy (declarative node sharing).
- Editing the `externalTailnets` section.
- Using `group://` or `tag://` selectors.

Status: alpha. Turn it on under General settings > Feature previews > Declarative Node Sharing > Manage in policy. It assumes, but does not establish, trust between the two tailnets' admins. To share one machine with one outside user, use link-based node sharing instead.

How it works (double opt-in; nothing happens until both sides are configured):
- The receiving tailnet lists the sharing tailnet and chooses which of its local groups and tags the other side may reference.
- The sharing tailnet lists the receiving tailnet, allows incoming connections, and writes grants from those external groups or tags to local resources.

Receiving tailnet:
```json
"externalTailnets": {
  "<tailnet-alias>": {
    "externalID": "<sharing-tailnet-id>",
    "allowIncomingConnections": false,
    "allowExternalReferencesTo": ["group:<group-name>"]
  }
}
```
- `allowExternalReferencesTo` lists local groups or tags the external tailnet may reference. They must be defined in this policy file; synced groups (SCIM, Google) are not supported.

Sharing tailnet:
```json
"externalTailnets": {
  "<tailnet-alias>": {
    "externalID": "<receiving-tailnet-id>",
    "allowIncomingConnections": true
  }
},
"grants": [
  {"src": ["group://<tailnet-alias>/<group-name>"], "dst": ["<resource>"], "ip": ["*"]}
]
```
- `allowIncomingConnections` must be `true` on the sharing side.
- External selectors go in grant `src`: `group://<tailnet-alias>/<group-name>` or `tag://<tailnet-alias>/<tag-name>`.
- A grant that references a group or tag missing from the other tailnet's `allowExternalReferencesTo` is ignored.
- `externalID` is the other tailnet's ID, shown under Unique IDs on the General settings page.

Using the share:
- Connect with the resource's Tailscale IP or its fully qualified MagicDNS name (such as `machine.example.ts.net`). Short MagicDNS names do not work across tailnets.
- There is no invitation or acceptance step. Existing link-based shares keep working.

Limits:
- Tailscale Services and exit nodes cannot be shared, and subnet routes are not advertised to the other tailnet.
- Alpha limits: at most three entries in `externalTailnets`. If either tailnet has more than 500 nodes, sharing grants between them are dropped.

Live Markdown: https://tailscale.com/docs/features/declarative-node-sharing.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/features/declarative-node-sharing
