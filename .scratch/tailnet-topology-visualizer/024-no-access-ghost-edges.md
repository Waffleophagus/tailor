# No-access ghost edges

Labels: ready-for-agent
Type: AFK

## What to build

In **focused scenario** graph mode, optionally show ghost edges or nodes for candidate connections that would exist topologically but are **denied** by policy — helping admins see gaps without cluttering whole-tailnet view.

Master roadmap: [018](018-policy-scenario-roadmap.md) Phase 10. Sub-issue of [011](011-graph-policy-preview-modes.md).

## Acceptance criteria

- [ ] Ghost no-access candidates appear only when a Policy Scenario is active and graph mode is focused.
- [ ] Ghost styling is distinct from allowed edges (dashed, muted, non-interactive or lens-only).
- [ ] Large tailnets: cap or filter ghosts (e.g. scenario subject → visible destinations only) to stay readable.
- [ ] Toggle to hide ghosts without clearing scenario.
- [ ] Tests or component tests for ghost rendering guard (focused + scenario required).

## Blocked by

- [021-policy-scenario-state.md](021-policy-scenario-state.md)
- [011-graph-policy-preview-modes.md](011-graph-policy-preview-modes.md) (partial)
