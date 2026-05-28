# Policy Workbench shell

Labels: ready-for-agent
Type: AFK

## What to build

Create the **Policy Workbench** — a Tailscale-shaped editing surface that replaces the flat read-only `PolicyPanel` as the primary policy UI. The graph remains visible as the live preview ("here's how it will look"); the workbench is where admins configure ACLs and definitions.

Reference: [`screenshots of ACL in site/1-general-access-rules-outside.png`](../../screenshots%20of%20ACL%20in%20site/1-general-access-rules-outside.png) — sidebar POLICY / DEFINITIONS structure, Visual editor / JSON editor toggle.

Master roadmap: [018-policy-scenario-roadmap.md](018-policy-scenario-roadmap.md) Phase 1.

Graph model: drop hypothetical `perspective:*` root; highlight real source devices per [018 § Graph rendering model](018-policy-scenario-roadmap.md#graph-rendering-model-default--no-hypothetical-root).

## Layout

- **Workbench panel/flyout** — opens from existing Policy entry point; does not hide the graph permanently (split or overlay per viewport).
- **Left nav** — two sections matching Tailscale:
  - **POLICY:** General access rules, Tailscale SSH, Tests, Auto-approvers
  - **DEFINITIONS:** Groups, Tags, IP sets, Hosts, Device posture, Node attributes
- **Main area** — route per nav item; placeholder list + empty state until section editors land.
- **Header** — "Access controls" title, Visual editor / JSON editor toggle (JSON stub → raw HuJSON escape hatch), close/minimize.
- **Search** — per-section where Tailscale shows it (rules table, groups table, etc.).

## Acceptance criteria

- [ ] Workbench opens from the app policy control and replaces the old flat section browser as the default policy surface.
- [ ] All Tailscale nav items exist with correct labels and route to a section view (placeholder content OK).
- [ ] Section views show entry count, search, and empty state from `policyMap` data where available ([009](009-structured-policy-map.md)).
- [ ] Unsupported sections from the policy map appear under an "Advanced" or inline fallback, not dropped.
- [ ] Visual / JSON editor toggle: Visual is default; JSON opens read-only or editable raw HuJSON audit view (reuse existing raw panel behavior).
- [ ] Existing "View as" shortcuts from policy entries wire to the scenario bar (no regression).
- [ ] Graph remains visible and interactive while workbench is open.
- [ ] **Graph rendering:** no hypothetical `perspective:*` node; scenario active → real source devices highlighted, focused subgraph per [018](018-policy-scenario-roadmap.md#graph-rendering-model-default--no-hypothetical-root).
- [ ] Remove legacy remap/collapse code (`perspective/device.ts`, `perspective/edges.ts` usage in `App.svelte`).
- [ ] Component or integration test covers nav routing and at least one section rendering entries from mock `policyMap`.

## Out of scope (follow-up issues)

- Full add/edit forms per section → [020](020-general-access-rules-editor.md), [014](014-definition-workbench-editors.md), [015](015-ssh-and-posture-access-builder.md), [017](017-advanced-policy-section-coverage.md)
- Staged commit tray → [016](016-staged-commit-tray-and-hujson-diff.md)
- Scenario persistence → [021](021-policy-scenario-state.md)

## Suggested implementation notes

- New module: `web/src/lib/workbench/` (routes, nav config, shell component).
- Nav config is data-driven — one array drives POLICY + DEFINITIONS items, section name mapping, and simulation tier badge.
- Migrate `PolicyPanel.svelte` read-only section rendering into shared section list components used by the shell.
- Deprecate duplicate policy UI entry points once shell ships.

## Blocked by

- [009-structured-policy-map.md](009-structured-policy-map.md) (done)
- [004-cloud-api-auth-policy-fetch.md](004-cloud-api-auth-policy-fetch.md) (done)

## Blocks

- [020-general-access-rules-editor.md](020-general-access-rules-editor.md)
- [014-definition-workbench-editors.md](014-definition-workbench-editors.md)
- [015-ssh-and-posture-access-builder.md](015-ssh-and-posture-access-builder.md)
- [017-advanced-policy-section-coverage.md](017-advanced-policy-section-coverage.md)
- [016-staged-commit-tray-and-hujson-diff.md](016-staged-commit-tray-and-hujson-diff.md)
