# Policy Lens provenance editor

Labels: ready-for-agent
Type: AFK

## What to build

Upgrade the right-side Policy Lens from passive Device details into an active explanation and editing surface. Selecting a Device or access edge should answer why access exists, which selectors matched, which policy sections are involved, and what safe edits are available.

This should make policy editing feel tied to concrete network impact instead of detached section tables.

## Acceptance criteria

- [ ] Selecting a Device shows "can reach", "reachable by", and "rules affecting this device" views.
- [ ] Selecting an access edge shows the policy references, matched selectors, ports/protocols, and access scope.
- [ ] The lens can jump to the relevant structured policy section from 009-structured-policy-map.md.
- [ ] The lens offers draft actions for narrow edits such as remove appended rule, duplicate and narrow rule, or change ports where the mutation is safe.
- [ ] All lens mutations update draft state and graph preview before validation.
- [ ] Ambiguous mutations explain why they are not offered and point to the policy section editor or raw HuJSON.
- [ ] Tests cover provenance display and at least one safe draft mutation from an edge.

## Blocked by

- 009-structured-policy-map.md
- 010-draft-policy-evaluation-api.md
- 011-graph-policy-preview-modes.md

