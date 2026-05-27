# SSH and posture access builder

Labels: ready-for-agent
Type: AFK

## What to build

Add a focused builder for Tailscale SSH and conditional access. Admins should be able to define who can SSH, which destinations are eligible, which login users are allowed, and what check or accept behavior applies. Where posture/device conditions, via routing, or app capabilities are available in the policy shape, the builder should expose them as advanced controls and preview the resulting graph impact.

The builder must clearly distinguish network reachability from SSH permission. SSH policy alone should not imply that the network path exists.

## Acceptance criteria

- [ ] The builder creates and edits SSH rules with source, destination, users, action, and check mode where applicable.
- [ ] The UI explains when SSH permission also requires network access from ACLs or grants.
- [ ] Advanced controls can reference posture/device conditions and via routing where those fields are present in the policy.
- [ ] Generated changes enter draft state and update graph preview before validation.
- [ ] The Policy Lens can show SSH permission separately from network access for a selected Device or edge.
- [ ] Unsupported SSH rule shapes remain visible and fall back to raw HuJSON editing.
- [ ] Tests cover SSH rule generation, SSH/network distinction, and validation error surfacing.

## Blocked by

- 009-structured-policy-map.md
- 010-draft-policy-evaluation-api.md
- 011-graph-policy-preview-modes.md

