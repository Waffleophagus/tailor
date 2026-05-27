# Graph-seeded allow access builder

Labels: ready-for-agent
Type: AFK

## What to build

Replace the raw selector-first ACL form with an intent builder that starts from the graph. An admin should be able to choose a source, choose a destination, choose an access shape, and immediately preview the resulting access on the graph before validation.

Prefer grants where the requested access is naturally represented as a grant. Fall back to ACL rules for port-level network access that still belongs in ACLs.

## Acceptance criteria

- [ ] The builder can be opened from selected graph nodes, selected devices in the sidebar, or the policy workbench.
- [ ] Source selection supports user, group, tag, autogroup, and raw selector entry.
- [ ] Destination selection supports device-derived selectors, tag, host, IP, IP set, subnet, and raw selector entry.
- [ ] Access presets include SSH, HTTP/S, custom ports, all ports, and grant-style app capability where supported.
- [ ] Generated policy changes are added to draft state only, not saved immediately.
- [ ] The graph preview updates from the draft evaluation API before validation.
- [ ] Tests cover generated ACL/grant draft output and source/destination selector handling.

## Blocked by

- 010-draft-policy-evaluation-api.md
- 011-graph-policy-preview-modes.md

