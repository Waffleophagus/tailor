# General access rules editor

Labels: ready-for-agent
Type: AFK

## What to build

Implement **General access rules** inside the Policy Workbench — the Tailscale POLICY nav item for `acls` and `grants`. Admins browse rules in a table, add/edit via a structured form, and see a live JSON preview per rule. Changes stage to draft and update the graph preview under the active scenario.

Reference:
- List: [`1-general-access-rules-outside.png`](../../screenshots%20of%20ACL%20in%20site/1-general-access-rules-outside.png)
- Add rule: [`2-general-access-rules-clicked-in.png`](../../screenshots%20of%20ACL%20in%20site/2-general-access-rules-clicked-in.png)

Supersedes intent of [012-graph-seeded-allow-access-builder.md](012-graph-seeded-allow-access-builder.md) for the primary editing path. Graph seeding remains via Policy Lens and scenario bar.

Master roadmap: [018](018-policy-scenario-roadmap.md) Phase 2.

## Rule list view

- Table columns aligned with Tailscale: **Sources**, **can access destinations**, **on port and protocol**.
- Expand row → human-readable breakdown + JSON block for that rule.
- Search filters by user, group, device, tag, port, IP (client-side over rule selectors).
- **+ Add rule** opens add/edit form.

## Add / edit rule form

**Core fields:**

- Source — selector picker (user, group, tag, autogroup, host, IP set, raw)
- Destination — selector picker (+ port suffix handling)
- Port and protocol — presets (All, SSH, HTTP/S, custom) + removable tags
- Note — free text stored as HuJSON comment above rule where supported

**Advanced options (collapsed by default):**

- Device posture — selector picker referencing `postures` entries
- Via — tag to route via
- App / Capability — grant-style fields where policy uses grants

**Live JSON preview** — updates as fields change (right column on wide layouts, below on narrow).

**Actions:** Save to draft (not Cloud save), Cancel.

Prefer **grants** when access shape is app/capability-native; fall back to **ACL** for port/protocol network access.

## Acceptance criteria

- [ ] Rules list renders ACL and grant entries from draft or saved policy with search.
- [ ] Add rule form produces valid draft HuJSON for representative ACL and grant shapes.
- [ ] Advanced fields (posture, via, app) serialize correctly when present; hidden when empty.
- [ ] Live JSON preview matches staged draft output.
- [ ] Saving to draft triggers evaluate-draft and updates graph (respecting active scenario when set).
- [ ] Unsupported rule shapes show read-only JSON with link to raw HuJSON editor.
- [ ] Can open builder pre-filled from Policy Lens / graph selection (src/dst seed).
- [ ] Tests cover ACL generation, grant generation, port preset handling, and draft staging.

## Simulation tier

**Graph-simulated** — ACL/grant selector resolution already supported by evaluate-draft. Posture/via on rules may be edit-only until backend eval catches up; label in UI if graph ignores them.

## Blocked by

- [019-policy-workbench-shell.md](019-policy-workbench-shell.md)
- [010-draft-policy-evaluation-api.md](010-draft-policy-evaluation-api.md) (done)

## Blocks

- [022-scenario-centric-editing.md](022-scenario-centric-editing.md) (partial — rule editor is primary edit surface)
