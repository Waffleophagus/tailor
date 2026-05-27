# Graph-backed policy workbench visual treatment decision

Labels: ready-for-human
Type: HITL

## Status

Decision captured. Implementation moved to 011-graph-policy-preview-modes.md.

## What to build

Decide and document the Phase 2 graph-backed policy workbench visual treatment before implementation. The workbench should make the graph the policy preview surface, not merely a topology background behind raw HuJSON editing.

Perspective is policy simulation only. It must not imply real credential impersonation, user auth token extraction, or active probing from that user's device.

The result should be concrete enough for an AFK agent to implement without guessing.

## Decision

The workbench has three graph states:

- Current: saved policy effective access.
- Draft: saved policy plus pending policy mutations.
- Diff: added access, removed access, unchanged access, unresolved selectors, and risky broad access.

Visual treatment:

- Existing access uses normal solid edges.
- Added draft access uses a dashed edge and a distinct "added" badge in the selected edge details.
- Removed draft access stays visible as a muted ghost edge with a removal marker.
- Broad access uses heavier weight and warning copy in the details pane.
- SSH, HTTP/S, custom, and limited access differ by label, stroke pattern, and icon, not color alone.
- No-access is not drawn as a full graph mesh by default. It appears when the user selects a source, destination, or Perspective, then inaccessible candidates are faded in the surrounding list and optionally shown as sparse ghost relationships.
- Unknown or unresolved selectors are shown as warnings in the draft tray and Policy Lens. They should not silently disappear.

Perspective behavior:

- Selecting a User, Group, Tag, or Autogroup recalculates the graph around what that policy subject can reach.
- Large tailnets default to focused view around the selected subject or selected device. All-connections mode remains available.
- Perspective labels must say "simulated policy subject" or equivalent clear copy in the UI surface where the selection is made.

## Acceptance criteria

- [x] The chosen Perspective behavior is documented for selected Users, Groups, Tags, and Autogroups.
- [x] The decision states whether inaccessible Devices are hidden, faded, grouped, or otherwise represented.
- [x] The decision states how blocked or no-access edges differ from allowed Effective Access edges.
- [x] The decision states how limited access is represented when only some ports/protocols are available, such as HTTPS (`tcp:443`) but not SSH (`tcp:22`).
- [x] The decision defines visual categories for SSH, HTTP/S, broad/custom access, limited/partial access, and no-access without depending on color alone.
- [x] The decision accounts for graph readability on large tailnets.
- [x] Follow-up implementation criteria are clear enough to create an AFK issue.

## Blocked by

- 005-effective-access-edges.md
