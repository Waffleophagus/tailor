# Automation

Use when:
- Auto-approving routes, exit nodes, or app connectors.
- Managing `nodeAttrs`.
- Enabling Funnel or client network options.

Auto approvers:
- `autoApprovers.routes` approves advertised subnet routes for matching devices or users.
- `autoApprovers.exitNode` approves exit node use for matching devices or users.
- App connector approval can be modeled through app connector policy fields where supported.
- Auto approvers are not retroactive; existing routes may need re-advertisement or manual action.

Node attributes:
- `nodeAttrs` applies device-level attributes to matching targets.
- Common attributes include Funnel, NextDNS, randomize client port, and disabling IPv4 behavior.
- App connectors can be configured through `nodeAttrs`.

Common mistakes:
- Expecting auto approvers to approve already-advertised routes retroactively.
- Confusing network option names with node attribute keys.
- Using selectors not legal in approver contexts.

Sources:
- https://tailscale.com/kb/1337/policy-syntax
- https://tailscale.com/kb/1223/funnel
- https://tailscale.com/kb/1281/app-connectors
