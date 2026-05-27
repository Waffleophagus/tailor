# Staged commit tray and HuJSON diff

Labels: ready-for-agent
Type: AFK

## What to build

Create the persistent draft review surface for graph-backed policy editing. The tray should accumulate pending changes from builders and Policy Lens actions, summarize graph impact, show generated HuJSON diff on demand, validate with the Cloud API, save only after validation, and discard safely.

Raw HuJSON is an audit and escape hatch here, not the primary editing experience.

## Acceptance criteria

- [ ] The UI has a persistent draft tray that lists pending policy changes and their source surface.
- [ ] The tray summarizes graph impact: added access, removed access, broad access introduced, unresolved selectors, and unsupported sections.
- [ ] A HuJSON diff view compares saved policy and draft policy while preserving existing comments outside mutated sections.
- [ ] Validation calls the existing Cloud API validate flow and displays actionable errors.
- [ ] Save is enabled only for the last successfully validated draft.
- [ ] Discard clears draft state and returns the graph to Current mode.
- [ ] Successful save refreshes policy, structured policy map, and graph edges.
- [ ] Tests cover validate, save, discard, stale validation invalidation, and comment preservation.

## Blocked by

- 010-draft-policy-evaluation-api.md
- 011-graph-policy-preview-modes.md

