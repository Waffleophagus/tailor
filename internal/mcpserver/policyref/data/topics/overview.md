# Overview

Use when:
- Understanding policy file structure.
- Choosing grants over legacy ACLs.
- Remembering deny-by-default behavior.

Top-level sections:
- Access control: `grants` (preferred), `acls` (legacy), `ssh`.
- Automation and attributes: `autoApprovers`, `nodeAttrs`, `postures`, `defaultSrcPosture`.
- Targets: `groups`, `tagOwners`, `hosts`, `ipsets`.
- Tests: `tests`, `sshTests`.
- Network policy options (most tailnets never need them): `derpMap`, `disableIPv4`, `OneCGNATRoute`, `randomizeClientPort`.
- Cross-tailnet sharing (alpha): `externalTailnets`. See the `external_tailnets` topic.

Rules:
- The policy file is one HuJSON object (JSON with comments and trailing commas).
- Access is deny-by-default. Grants have an implied accept; the only ACL `action` is `accept`.
- Prefer `grants` for new access rules. Grants are feature complete with ACLs; ACLs keep working but get no new features.
- Grants and ACLs are additive. When several rules match, their access is unioned. A narrower rule never overrides a broader one, so you cannot remove access by adding a rule.
- Tailscale SSH needs both an `ssh` rule and network access from the source to the destination on port 22 (a grant or ACL).
- `ssh` stays a separate section. When migrating ACLs to grants, keep SSH rules as they are.
- `tests` and `sshTests` run on every policy change. A failing test rejects the whole update.
- Editing workflow for agents: read the relevant reference topic, inspect the policy map, draft HuJSON, evaluate the draft, then stage it for human review.

Live Markdown: https://tailscale.com/docs/reference/syntax/policy-file.md (compare its "Last validated" line with this card's lastValidated; this card is the offline answer).

Sources:
- https://tailscale.com/docs/reference/syntax/policy-file
- https://tailscale.com/docs/reference/grants-vs-acls
- https://tailscale.com/docs/reference/migrate-acls-grants
- https://tailscale.com/docs/features/device-posture/postures-in-access-rules
- https://tailscale.com/docs/features/declarative-node-sharing
