# HuJSON staged policy save loop

Labels: ready-for-agent
Type: AFK

## What to build

Implement the smallest safe ACL editing loop: users can make a scoped policy change, review old vs. new HuJSON, validate the draft with the Cloud API, and save only after validation succeeds. The HuJSON round-trip must preserve existing admin comments and formatting outside mutated sections.

This slice should focus on one narrow edit path, such as adding or changing a Device Tag owner or a simple ACL destination entry, rather than the full Policy Lens editor.

## Acceptance criteria

- [ ] The backend parses HuJSON with Tailscale's `github.com/tailscale/hujson` library using the accepted AST hybrid approach.
- [ ] The UI accumulates policy edits in draft state before save.
- [ ] A diff viewer shows old HuJSON and new HuJSON before validation.
- [ ] `POST /api/validate` calls the Cloud API validate endpoint and returns line-aware errors when available.
- [ ] `POST /api/acl` saves only a draft that has passed validation.
- [ ] On successful save, the app refreshes policy data and graph edges.
- [ ] Tests verify comments and formatting are preserved outside mutated policy sections.

## Blocked by

- 004-cloud-api-auth-policy-fetch.md
- 005-effective-access-edges.md
