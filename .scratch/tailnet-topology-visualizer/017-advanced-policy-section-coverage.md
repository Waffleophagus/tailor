# Advanced policy section coverage

Labels: ready-for-agent
Type: AFK

## What to build

Round out Policy Workbench coverage for sections that don't fit the first rule/definition editors: **Tests**, **Auto-approvers**, **Device posture** (definitions), **Node attributes**, and any unknown policy sections.

Reference: posture [`13-device-posture-general.png`](../../screenshots%20of%20ACL%20in%20site/13-device-posture-general.png), node attrs [`15-node-attributes-general.png`](../../screenshots%20of%20ACL%20in%20site/15-node-attributes-general.png).

Master roadmap: [018-policy-scenario-roadmap.md](018-policy-scenario-roadmap.md) Phase 5.

Structured hybrid editor: known fields get real controls; complex values get typed JSON with validation; HuJSON comments preserved on round-trip.

## Section placement

| Section | Workbench nav | Simulation tier |
|---------|---------------|-----------------|
| Device posture | DEFINITIONS → Device posture | Edit + validate only (graph eval needs device attributes) |
| Node attributes | DEFINITIONS → Node attributes | Edit + validate only |
| Tests | POLICY → Tests | Non-graph (test runner future) |
| Auto-approvers | POLICY → Auto-approvers | Non-graph |
| Unknown sections | Inline / Advanced fallback | Edit + validate |

## Acceptance criteria

- [ ] Each advanced route lists entries with counts, search, empty states.
- [ ] Device posture: add/edit with attribute key/value pairs (e.g. `node:os`, `node:tsStateEncrypted`).
- [ ] Node attributes: structured key/value or JSON object editing with validation.
- [ ] Tests and SSH tests: structured expected-access fields where schema known.
- [ ] Auto-approvers: routes, exit nodes, advertised tags where present.
- [ ] Unknown sections preserved, visible, editable via explicit raw-value editor.
- [ ] Simulation tier badges explain what the graph can and cannot preview.
- [ ] Advanced edits stage to draft, appear in staged tray, validate before save.
- [ ] Tests cover known sections, unknown preservation, validation errors.

## Blocked by

- [019-policy-workbench-shell.md](019-policy-workbench-shell.md)
- [009-structured-policy-map.md](009-structured-policy-map.md) (done)
- [016-staged-commit-tray-and-hujson-diff.md](016-staged-commit-tray-and-hujson-diff.md)
