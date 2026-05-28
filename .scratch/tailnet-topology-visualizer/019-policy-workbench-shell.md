# Policy Workbench shell

Labels: ready-for-agent
Type: AFK

## What to build

Create the **Policy Workbench** — a Tailscale-shaped editing surface that replaces the flat read-only `PolicyPanel` as the primary policy UI. The graph remains visible as the live preview ("here's how it will look"); the workbench is where admins configure ACLs and definitions.

Reference: [`screenshots of ACL in site/1-general-access-rules-outside.png`](../../screenshots%20of%20ACL%20in%20site/1-general-access-rules-outside.png) — POLICY / DEFINITIONS nav structure, section tables, search, empty states. **Functional parity** with Tailscale (same nav items, fields, section structure, interaction patterns). Tailor design tokens — not a pixel-perfect clone of Tailscale chrome.

Master roadmap: [018-policy-scenario-roadmap.md](018-policy-scenario-roadmap.md) Phase 1.

Graph model: drop hypothetical `perspective:*` root; highlight real source devices per [018 § Graph rendering model](018-policy-scenario-roadmap.md#graph-rendering-model-default--no-hypothetical-root).

## Layout (decided)

- **Right-side workbench panel** — slides in from the right edge of the graph column (not a bottom overlay, not a left flyout). Graph keeps maximum horizontal space for live preview.
- **Auto-dismiss sidebars** — opening the workbench closes `SidebarLeft` and `SidebarRight`; closing the workbench restores prior sidebar state (or leaves them closed — implementer's choice; prefer restore).
- **Fast motion** — open/close animation ~150–200ms ease-out; graph reflows immediately so edits are visible without lag.
- **Desktop-first** — no narrow-viewport / mobile layout work in this issue.
- **Left nav inside the panel** — two sections matching Tailscale (nav is internal to the right panel, mirroring Tailscale's sidebar IA):
  - **POLICY:** General access rules, Tailscale SSH, Tests, Auto-approvers
  - **DEFINITIONS:** Groups, Tags, IP sets, Hosts, Device posture, Node attributes
- **Main area** — route per nav item; list + search + empty state from `policyMap` until section editors land ([020](020-general-access-rules-editor.md), [014](014-definition-workbench-editors.md), …).
- **Header** — "Access controls" title, close button. **No Visual/JSON toggle in Phase 1** — raw HuJSON remains a separate advanced entry (header "Raw HuJSON" or later [016](016-staged-commit-tray-and-hujson-diff.md)).
- **Search** — per-section where Tailscale shows it (rules table, groups table, etc.).

## Migration stance

Non-production app: **breaking changes OK.** Delete or replace `PolicyPanel.svelte`, legacy perspective graph code, and duplicate policy entry points as needed. Do not preserve backward-compatible shims.

## Acceptance criteria

- [ ] Workbench opens from the app policy control (replace "Raw HuJSON" as primary ACL surface with "Access controls" or equivalent) and replaces the old flat section browser.
- [ ] Opening workbench auto-dismisses left and right sidebars; graph expands; fast slide animation.
- [ ] All Tailscale nav items exist with correct labels and route to a section view (placeholder list content OK; structure must match Tailscale functionally).
- [ ] Section views show entry count, search, and empty state from `policyMap` data where available ([009](009-structured-policy-map.md)).
- [ ] General access rules route covers **both** `acls` and `grants` (combined list or sub-tabs — match Tailscale behavior).
- [ ] Unsupported sections from the policy map appear under an "Advanced" or inline fallback, not dropped.
- [ ] **No Visual/JSON toggle** in Phase 1 (deferred).
- [ ] Existing "View as" shortcuts from policy entries wire to the scenario bar (no regression).
- [ ] Graph remains visible and interactive while workbench is open.
- [ ] **Graph rendering:** no hypothetical `perspective:*` node; scenario active → real source devices highlighted, focused subgraph per [018](018-policy-scenario-roadmap.md#graph-rendering-model-default--no-hypothetical-root).
- [ ] Remove legacy remap/collapse code (`perspective/device.ts`, `perspective/edges.ts`, related `App.svelte` logic).
- [ ] Component or integration test covers nav routing and at least one section rendering entries from mock `policyMap`.

## Out of scope (follow-up issues)

- Full add/edit forms per section → [020](020-general-access-rules-editor.md), [014](014-definition-workbench-editors.md), [015](015-ssh-and-posture-access-builder.md), [017](017-advanced-policy-section-coverage.md)
- Visual/JSON editor toggle + raw HuJSON audit in workbench header → [016](016-staged-commit-tray-and-hujson-diff.md) or later
- Staged commit tray → [016](016-staged-commit-tray-and-hujson-diff.md)
- Scenario persistence → [021](021-policy-scenario-state.md)
- Mobile / narrow viewport layout

## Suggested implementation notes

- New module: `web/src/lib/workbench/` (routes, nav config, shell component).
- Nav config is data-driven — one array drives POLICY + DEFINITIONS items, section name mapping, and simulation tier badge.
- Extract shared section list components from current `PolicyPanel` logic, then **delete** `PolicyPanel.svelte`.
- App shell: `[SidebarLeft | Graph | WorkbenchPanel]` when open; workbench width ~`min(28rem, 40vw)` or similar — tune for graph readability.
- Use `transition:fly` or CSS transform on the right panel; call `graphAPI.reflow()` on open/close.

## Blocked by

- [009-structured-policy-map.md](009-structured-policy-map.md) (done)
- [004-cloud-api-auth-policy-fetch.md](004-cloud-api-auth-policy-fetch.md) (done)

## Blocks

- [020-general-access-rules-editor.md](020-general-access-rules-editor.md)
- [014-definition-workbench-editors.md](014-definition-workbench-editors.md)
- [015-ssh-and-posture-access-builder.md](015-ssh-and-posture-access-builder.md)
- [017-advanced-policy-section-coverage.md](017-advanced-policy-section-coverage.md)
- [016-staged-commit-tray-and-hujson-diff.md](016-staged-commit-tray-and-hujson-diff.md)
