# Graph policy preview modes

Labels: ready-for-agent
Type: AFK

## What to build

Turn the graph into the primary policy preview surface. Add explicit Current, Draft, and Diff modes that render saved access, draft access, and the delta between them. This implements the visual decision from 007-perspective-filter-prototype.md.

The goal is that an admin can understand the blast radius of a policy change from the graph before reading generated HuJSON.

## Acceptance criteria

- [ ] The graph has visible Current, Draft, and Diff modes when a draft exists.
- [ ] Added, removed, unchanged, broad, SSH, HTTP/S, custom, limited, and unresolved access states are visually distinct without relying on color alone.
- [ ] Selecting a changed edge shows the responsible saved and draft policy references.
- [ ] No-access candidates are shown only in focused contexts so large tailnets do not become unreadable.
- [ ] Perspective selection for User, Group, Tag, and Autogroup recalculates the focused graph and labels the view as simulated policy subject access.
- [ ] The graph remains usable on large tailnets through focused mode, filtering, and all-connections fallback.
- [ ] UI tests or component tests cover mode switching and representative edge state rendering.

## Blocked by

- 007-perspective-filter-prototype.md
- 010-draft-policy-evaluation-api.md
