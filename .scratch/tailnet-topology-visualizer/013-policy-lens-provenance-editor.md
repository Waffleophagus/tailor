# Policy Lens provenance editor

Labels: ready-for-agent
Type: AFK

## What to build

Upgrade the right-side **Policy Lens** from passive device details into an active explanation and editing surface that bridges the **graph preview** and the **Policy Workbench**. Selecting a device or access edge answers why access exists and offers safe edits — always under the active **Policy Scenario** when one is set.

Master roadmap: [018-policy-scenario-roadmap.md](018-policy-scenario-roadmap.md) Phase 11.

## Relationship to other issues

- **Jump to workbench** — lens links open the relevant workbench route ([019](019-policy-workbench-shell.md)): General access rules, SSH, Groups, Tags, etc.
- **Draft mutations** — stage to draft and auto re-eval under scenario ([022](022-scenario-centric-editing.md)).
- **Graph modes** — respect Current/Draft/Diff ([011](011-graph-policy-preview-modes.md)).

## Acceptance criteria

- [ ] Selecting a Device shows "can reach", "reachable by", and "rules affecting this device" views.
- [ ] Selecting an access edge shows policy references, matched selectors, ports/protocols, and access scope.
- [ ] SSH permission is shown separately from network ACL/grant reachability where both apply.
- [ ] The lens can jump to the relevant Policy Workbench section (not the old flat policy panel).
- [ ] The lens offers draft actions for narrow safe edits: remove appended rule, duplicate and narrow, change ports where unambiguous.
- [ ] All lens mutations update draft state and graph preview before validation; auto re-eval when scenario active.
- [ ] Ambiguous mutations explain why they are not offered and point to the workbench section editor or raw HuJSON.
- [ ] Tests cover provenance display and at least one safe draft mutation from an edge.

## Blocked by

- [019-policy-workbench-shell.md](019-policy-workbench-shell.md)
- [020-general-access-rules-editor.md](020-general-access-rules-editor.md) (jump target for ACL edits)
- [021-policy-scenario-state.md](021-policy-scenario-state.md)
- [022-scenario-centric-editing.md](022-scenario-centric-editing.md)
- [011-graph-policy-preview-modes.md](011-graph-policy-preview-modes.md) (partial)
