# Policy scenario state

Labels: ready-for-agent
Type: AFK

## What to build

Introduce **Policy Scenario** as a first-class persistent simulation setup. Replaces loose `policyPerspective`, `simulatedPerspective`, and `policyGraphViewMode` state with one object that survives policy mode toggles and optionally page refresh.

Master roadmap: [018](018-policy-scenario-roadmap.md) Phase 7.

## Data model

Extract to `web/src/lib/scenario/state.ts`:

```typescript
interface PolicyScenario {
  id: string;                       // uuid for session
  sourceSelector: string;             // e.g. alice@…, group:eng, cohort:member+tagged
  policyMode: 'current' | 'draft' | 'diff';
  graphMode: 'focused' | 'all';
  simulatedAt?: number;             // last successful evaluate timestamp
  label?: string;                   // optional display name
}
```

- **Simulate** commits input → `PolicyScenario`; **Clear** resets to whole-tailnet (null scenario).
- Persist last scenario in `sessionStorage` (named library in `localStorage` → Phase 12).

## UX

- **Scenario bar** above graph (evolve `PerspectiveBar` / `PerspectiveSelector`):
  - Chip: `Viewing as Member ∪ Tagged · 18 sources · draft`
  - Subject picker, policy mode toggles, graph mode toggles
- Mode toggles (Current / Draft / Diff) mutate `scenario.policyMode` **without** clearing subject.
- Workbench header shows subtle active-scenario indicator when simulating.

## Acceptance criteria

- [ ] Single scenario object drives evaluate-draft `perspective` param and graph focused mode.
- [ ] Toggling Current/Draft/Diff does not reset source selector.
- [ ] Clear scenario returns graph to whole-tailnet view.
- [ ] Optional: refresh restores last scenario from sessionStorage.
- [ ] Tests cover mode toggle persistence and clear behavior.

## Blocked by

- [019-policy-workbench-shell.md](019-policy-workbench-shell.md) (scenario bar layout)
- Existing perspective simulation (shipped)

## Blocks

- [022-scenario-centric-editing.md](022-scenario-centric-editing.md)
