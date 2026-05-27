# Definition workbench editors

Labels: ready-for-agent
Type: AFK

## What to build

Add structured editors for reusable policy definitions: groups, tag owners, hosts, and IP sets. These definitions are the nouns that make access rules understandable, so editing them should include where-used context and graph preview impact.

This is intentionally not a clone of a raw section table. Each editor should help an admin understand which Devices or selectors are affected by the definition.

## Acceptance criteria

- [ ] Groups can be created and edited with member validation, where-used references, and draft graph preview.
- [ ] Tag owners can be created and edited with owner selector validation and affected tag context.
- [ ] Hosts can be created and edited with IP/CIDR validation and selector resolution preview.
- [ ] IP sets can be created and edited with target validation and where-used references.
- [ ] Definition edits are staged as draft policy changes and do not save immediately.
- [ ] Unknown or unsupported definition shapes remain visible and fall back to raw HuJSON editing.
- [ ] Tests cover at least one create and one edit path per supported definition type.

## Blocked by

- 009-structured-policy-map.md
- 010-draft-policy-evaluation-api.md
- 016-staged-commit-tray-and-hujson-diff.md

