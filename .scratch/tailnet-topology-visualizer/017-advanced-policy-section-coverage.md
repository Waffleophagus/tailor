# Advanced policy section coverage

Labels: ready-for-agent
Type: AFK

## What to build

Round out coverage for advanced policy sections that do not fit the first builders but still need a clean non-raw-HuJSON path: tests, SSH tests, auto-approvers, node attributes, posture/device conditions, and any unknown policy sections discovered in real policy files.

The target is a structured hybrid editor: safe known fields get real controls, complex or unknown values get typed structured JSON controls with validation, comments preserved by the HuJSON round-trip layer.

## Acceptance criteria

- [ ] The policy workbench lists advanced sections with counts, search, empty states, and raw HuJSON fallback.
- [ ] Tests and SSH tests can be edited with structured expected-access fields and validation feedback.
- [ ] Auto-approvers can be edited for routes, exit nodes, and advertised tags where present.
- [ ] Node attributes and posture/device conditions can be edited as structured key/value or JSON objects with schema-aware validation where available.
- [ ] Unknown sections are preserved, visible, and editable through an explicit advanced raw-value editor.
- [ ] Advanced edits enter draft state, update the staged tray, and validate before save.
- [ ] Tests cover known advanced sections, unknown section preservation, and validation error display.

## Blocked by

- 009-structured-policy-map.md
- 016-staged-commit-tray-and-hujson-diff.md

