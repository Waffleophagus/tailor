# Overview

Use when:
- Understanding policy file structure.
- Choosing grants over legacy ACLs.
- Remembering deny-by-default behavior.

Rules:
- Tailnet policy is a HuJSON object with top-level sections such as `grants`, `acls`, `ssh`, `groups`, `tagOwners`, `hosts`, `ipsets`, `autoApprovers`, `nodeAttrs`, `postures`, `tests`, and `sshTests`.
- Access is deny-by-default. Devices cannot connect unless a grant, ACL, SSH rule, or other feature-specific rule allows it.
- Prefer `grants` for new network access. `acls` are legacy and remain useful when editing existing policies.
- Grants and ACLs are additive. A narrow rule does not override a broad rule.
- Editing workflow for agents: read the relevant reference topic, inspect the policy map, draft HuJSON, evaluate the draft, then stage it for human review.

Sources:
- https://tailscale.com/kb/1018/acls
- https://tailscale.com/kb/1337/policy-syntax
- https://tailscale.com/kb/1324/grants
