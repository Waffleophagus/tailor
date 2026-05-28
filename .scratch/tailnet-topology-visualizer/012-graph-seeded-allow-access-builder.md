# Graph-seeded allow access builder

Labels: wontfix
Type: AFK

**Superseded by [020-general-access-rules-editor.md](020-general-access-rules-editor.md).**

The primary ACL/grant editing path now lives in the Policy Workbench **General access rules** section (Tailscale-shaped list + Add rule form). Graph and Policy Lens still **seed** that form with pre-filled src/dst — see 020 acceptance criteria.

Master roadmap: [018-policy-scenario-roadmap.md](018-policy-scenario-roadmap.md).

## Historical intent

Replace raw selector-first ACL form with intent builder starting from graph selection. That behavior is retained as seeding into [020](020-general-access-rules-editor.md), not a separate builder surface.

## Original acceptance criteria (for reference)

- Builder opened from graph nodes, sidebar devices, or policy workbench
- Source/destination selector support
- Access presets, draft staging, graph preview

## Blocked by

- N/A — do not implement this issue
