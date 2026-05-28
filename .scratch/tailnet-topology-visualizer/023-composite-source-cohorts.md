# Composite source cohorts

Labels: ready-for-agent
Type: AFK

## What to build

Fix Tailscale-accurate **member** source semantics and add a **member ∪ tagged** composite cohort for simulation. Enables "show me everything members OR tagged servers can initiate" in one scenario.

Graph shows the union as **highlighted real source devices**, not a synthetic center node ([018 § Graph rendering model](018-policy-scenario-roadmap.md#graph-rendering-model-default--no-hypothetical-root)).

Master roadmap: [018](018-policy-scenario-roadmap.md) Phase 9.

Reference: [Tailscale autogroups](https://tailscale.com/docs/reference/syntax/policy-file)

## Backend (`internal/policy/policy.go`)

- Add `devicesWithOwnerUntagged(devices)` — `owner != ""` and `len(tags) == 0`.
- Update `devicesForPerspective`: `autogroup:member` uses untagged only.
- Add encoded selector `cohort:member+tagged` — union of member + tagged device sets.
- Update `selectorIncludesPerspective` for union: match rules where src is `autogroup:member`, `autogroup:tagged`, matching tags, or `*`.
- Go tests:
  - Member excludes tagged-only devices as **sources**
  - Union includes both cohorts
  - Member perspective still sees tagged devices as **destinations**

Keep `perspective: string` on evaluate-draft (Option A — no OpenAPI change required for MVP).

## Frontend (`catalog.ts`, `subjects.ts`, scenario picker)

- Relabel `autogroup:member` → "Member devices (user-owned, untagged)"
- Relabel `autogroup:tagged` → "Tagged devices"
- Add `cohort:member+tagged` → "Members and tagged devices (all initiators)"
- Mirror cohort resolution in `subjectDeviceIds` for source highlight and scenario bar count.
- Scenario bar shows preset label + source device count.

## Acceptance criteria

- [ ] Simulate `cohort:member+tagged`: all member and tagged source devices highlighted; edges from either cohort; visible source count = union size.
- [ ] Simulate `autogroup:member`: tagged devices with owners are **not** highlighted as sources unless reached as destinations.
- [ ] Backend tests pass; frontend catalog shows correct counts.

## Blocked by

- None (can ship in parallel with workbench shell)

## Blocks

- None (enhances scenario bar; not blocking workbench)
