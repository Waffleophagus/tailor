# Graph policy preview modes

Labels: ready-for-agent
Type: AFK

## What to build

The graph is the **"here's how it will look"** preview surface for the Policy Workbench. Add explicit Current, Draft, and Diff modes that render saved access, draft access, and the delta between them — always in the context of an optional **Policy Scenario** (simulated subject).

Master roadmap: [018-policy-scenario-roadmap.md](018-policy-scenario-roadmap.md) Phase 10.

## Relationship to other issues

- **Workbench edits** stage to draft → graph reflects via evaluate-draft ([020](020-general-access-rules-editor.md), [016](016-staged-commit-tray-and-hujson-diff.md)).
- **Scenario bar** sets who initiates ([021](021-policy-scenario-state.md)).
- **Ghost no-access** → [024-no-access-ghost-edges.md](024-no-access-ghost-edges.md).

## Acceptance criteria

- [ ] The graph has visible Current, Draft, and Diff modes when a draft exists.
- [ ] Added, removed, unchanged, broad, SSH, HTTP/S, custom, limited, and unresolved access states are visually distinct without relying on color alone.
- [ ] Selecting a changed edge shows the responsible saved and draft policy references.
- [x] Perspective selection for User, Group, Tag, and Autogroup recalculates the focused graph and labels the view as simulated policy subject access.
- [ ] No-access candidates are shown only in focused scenario contexts ([024](024-no-access-ghost-edges.md)) so large tailnets stay readable.
- [ ] The graph remains usable on large tailnets through focused mode, filtering, and all-connections fallback.
- [ ] UI tests or component tests cover mode switching and representative edge state rendering.

## Blocked by

- [010-draft-policy-evaluation-api.md](010-draft-policy-evaluation-api.md) (done)
- [007-perspective-filter-prototype.md](007-perspective-filter-prototype.md) (done)

## Notes

- Perspective simulation backend + selector UI are shipped. Graph **rendering** migrates to real-device highlight (no hypothetical root) in [019](019-policy-workbench-shell.md) — see [018 § Graph rendering model](018-policy-scenario-roadmap.md#graph-rendering-model-default--no-hypothetical-root).
- Diff edge styling is partial today via `RenderEdge.state` in `web/src/lib/graph/engine.ts`.
