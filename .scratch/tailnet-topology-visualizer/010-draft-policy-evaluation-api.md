# Draft policy evaluation API

Labels: ready-for-agent
Type: AFK

## What to build

Add a grants-aware draft evaluation path that can compare the saved Policy File with an unsaved draft and return graph-ready impact data. This is the engine that lets Tailor preview policy changes before validation and save.

The output should explain impact in terms admins care about: added access, removed access, unchanged access, broad access introduced, selectors that do not currently resolve to Devices, and policy references responsible for the change. ACL rules and grants should both contribute to effective access.

## Acceptance criteria

- [ ] The backend can evaluate effective access for both the saved policy and a draft policy in one request.
- [ ] Grants are resolved into effective access edges alongside ACL rules, including network-level `ip` access where present.
- [ ] Application-layer grant capabilities are preserved in the response as policy impact metadata even when they do not map cleanly to a network edge.
- [ ] The response includes added, removed, unchanged, and changed access edges with ports, protocols, access scope, and policy references.
- [ ] The response includes unresolved selectors and unsupported sections encountered during evaluation.
- [ ] Draft evaluation does not save, validate, or mutate the Cloud API session.
- [ ] Existing topology snapshots continue to return saved-policy access unless a caller explicitly asks for draft evaluation.
- [ ] Tests cover added access, removed access, access scope changes, grants, unresolved selectors, and invalid draft input.

## Blocked by

- 005-effective-access-edges.md
- 009-structured-policy-map.md
