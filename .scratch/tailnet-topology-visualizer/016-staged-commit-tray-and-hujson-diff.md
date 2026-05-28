# Staged commit tray and HuJSON diff

Labels: ready-for-agent
Type: AFK

## What to build

Create the persistent **draft review surface** for Policy Workbench editing. The tray accumulates pending changes from all workbench section editors and Policy Lens actions, summarizes graph impact, shows HuJSON diff on demand, validates with the Cloud API, and saves only after validation.

Raw HuJSON is an audit and escape hatch — accessed via workbench Visual/JSON toggle — not the primary editing experience.

Master roadmap: [018-policy-scenario-roadmap.md](018-policy-scenario-roadmap.md) Phase 6.

## Relationship to other issues

- **Change sources:** General access rules ([020](020-general-access-rules-editor.md)), definitions ([014](014-definition-workbench-editors.md)), SSH ([015](015-ssh-and-posture-access-builder.md)), advanced ([017](017-advanced-policy-section-coverage.md)), Policy Lens ([013](013-policy-lens-provenance-editor.md)).
- **Graph preview:** tray summarizes same impact visible on graph; Current/Draft/Diff modes ([011](011-graph-policy-preview-modes.md)).
- **Scenario:** discard returns to Current mode but preserves scenario subject if set ([021](021-policy-scenario-state.md)).

## Acceptance criteria

- [ ] Persistent draft tray lists pending changes with source surface (e.g. "General access rules — rule #3", "Groups — group:eng").
- [ ] Tray summarizes graph impact: added access, removed access, broad access, unresolved selectors, unsupported sections.
- [ ] HuJSON diff compares saved vs draft; comments preserved outside mutated sections.
- [ ] Validation uses Cloud API validate flow with actionable errors.
- [ ] Save enabled only after last successful validation.
- [ ] Discard clears draft state; graph returns to Current policy mode; scenario subject unchanged.
- [ ] Successful save refreshes policy map, workbench lists, and graph edges.
- [ ] Tests cover validate, save, discard, stale validation invalidation, comment preservation.

## Blocked by

- [019-policy-workbench-shell.md](019-policy-workbench-shell.md) (tray attaches to workbench chrome)
- [010-draft-policy-evaluation-api.md](010-draft-policy-evaluation-api.md) (done)
- [011-graph-policy-preview-modes.md](011-graph-policy-preview-modes.md) (partial)

## Blocks

- Full save workflow for [014](014-definition-workbench-editors.md), [015](015-ssh-and-posture-access-builder.md), [017](017-advanced-policy-section-coverage.md)
