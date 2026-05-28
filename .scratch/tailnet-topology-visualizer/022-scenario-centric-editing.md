# Scenario-centric editing

Labels: ready-for-agent
Type: AFK

## What to build

Close the **edit → view** loop: workbench and Policy Lens mutations automatically re-evaluate the graph under the **active Policy Scenario** without manual Simulate clicks. New rules default their source to the scenario subject.

Master roadmap: [018](018-policy-scenario-roadmap.md) Phase 8.

## Auto re-evaluate

After draft mutations from any workbench editor or Policy Lens action:

- If scenario active → call evaluate-draft with `scenario.sourceSelector`
- Refresh graph edges for current `scenario.policyMode`
- Debounce 300ms when typing in forms

Triggers: append rule, edit definition, remove rule, validate-side-effect refresh.

## Scenario-centric seeding

- **General access rules** add form: when scenario active, pre-fill **src** with scenario selector (or decomposed cohort ACL selectors).
- **Definition editors:** "View as this group/tag" sets scenario (already partially shipped — wire through scenario object).
- **Source device selected** (member of active scenario cohort): primary action "Add rule for this subject".
- Deprecate device-only `seedBuilder` default when scenario is active (`App.svelte`).

## Acceptance criteria

- [ ] Append ACL rule while simulating user X → graph updates without Simulate click.
- [ ] Edit group membership in draft → graph updates under active scenario.
- [ ] Debounced re-eval does not spam API during rapid typing.
- [ ] No scenario → existing manual Simulate behavior unchanged.
- [ ] Tests cover auto re-eval on rule append and debounce.

## Backend

No change required if perspective / cohort strings from [023](023-composite-source-cohorts.md) already work on evaluate-draft.

## Blocked by

- [021-policy-scenario-state.md](021-policy-scenario-state.md)
- [020-general-access-rules-editor.md](020-general-access-rules-editor.md) (or minimal draft staging from workbench)

## Blocks

- [013-policy-lens-provenance-editor.md](013-policy-lens-provenance-editor.md) (partial — lens mutations expect auto re-eval)
