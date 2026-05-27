# Phase 1 graph exploration controls

Labels: ready-for-human
Type: AFK

## Status

Done.

## What to build

Make the read-only topology graph useful for exploration: infer Phase 1 relationship edges from shared owners, shared tags, and subnet routing metadata; add sidebar filters for online status, subnet-routed devices, tags, owners, and OS; and show a Device detail panel when a node is selected.

This slice keeps the app unauthenticated. The detail panel should make it clear that ACL editing requires Phase 2 authentication.

## Acceptance criteria

- [x] The graph shows inferred Phase 1 edges without treating them as Effective Access.
- [x] Users can show or hide offline Devices and subnet-routed Devices.
- [x] Users can filter or color Devices by Tag, owner, and OS.
- [x] Selecting a Device shows metadata, Tags, owner, online status, and Tailscale IPs.
- [x] The detail panel includes an ACL-editing-requires-authentication banner.
- [x] Graph pan, zoom, and fit-to-view controls are available.

## Blocked by

- 001-localapi-status-graph.md

## Notes

- The graph now updates from topology socket snapshots rather than one-time REST fetches.
- Phase 1 edges remain inferred relationship edges only: owner, tag, and subnet route relationships are not treated as Effective Access.
