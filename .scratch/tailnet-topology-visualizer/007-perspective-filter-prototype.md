# Perspective filter visual treatment decision

Labels: ready-for-human
Type: HITL

## What to build

Decide and document the Phase 2 Perspective visual treatment before implementation. The PRD intentionally defers how inaccessible edges and nodes should appear when an admin selects a simulated Perspective such as a User, Group, Tag, or Autogroup. Produce a small interactive prototype or design note that resolves this behavior.

Perspective is policy simulation only. It must not imply real credential impersonation, user auth token extraction, or active probing from that user's device.

The result should be concrete enough for an AFK agent to implement without guessing.

## Acceptance criteria

- [ ] The chosen Perspective behavior is documented for selected Users, Groups, Tags, and Autogroups.
- [ ] The decision states whether inaccessible Devices are hidden, faded, grouped, or otherwise represented.
- [ ] The decision states how blocked or no-access edges differ from allowed Effective Access edges.
- [ ] The decision states how limited access is represented when only some ports/protocols are available, such as HTTPS (`tcp:443`) but not SSH (`tcp:22`).
- [ ] The decision defines visual categories for SSH, HTTP/S, broad/custom access, limited/partial access, and no-access without depending on color alone.
- [ ] The decision accounts for graph readability on large tailnets.
- [ ] Follow-up implementation criteria are clear enough to create an AFK issue.

## Blocked by

- 005-effective-access-edges.md
