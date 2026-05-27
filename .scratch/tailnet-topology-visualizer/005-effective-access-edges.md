# Effective access graph edges

Labels: ready-for-human
Type: AFK

## Status

Mostly done. Remaining visual treatment is moved to 011-graph-policy-preview-modes.md.

## What to build

Resolve ACL Rules, Grants, Groups, Tags, Autogroups, hosts, and IP selectors into Effective Access edges between Devices after Phase 2 authentication. Replace Phase 1 inferred relationship edges with access edges in the graph, and expose enough port/protocol scope plus rule provenance for later Policy Lens, limited-access coloring, and Perspective simulation work.

## Acceptance criteria

- [x] The backend parses the fetched Policy File into typed policy structures while preserving access to the original HuJSON.
- [x] The backend expands Groups, Tags, Autogroups, hosts, and IP ranges against known Devices.
- [x] The backend returns Effective Access edges with source Device, destination Device, allowed ports, protocols, access scope classification, and responsible policy references.
- [x] Edge data preserves enough detail to distinguish limited access such as HTTPS-only (`tcp:443`) from SSH (`tcp:22`) and broader/custom access.
- [x] The resolver can calculate effective access for all devices and for a selected policy subject as a simulation input, without authenticating as that subject.
- [x] The frontend renders authenticated graph edges from Effective Access rather than inferred Phase 1 relationships.
- [x] Edge payloads provide enough data for SSH, HTTP/S, broad, custom, limited, and no-access visual treatment in the graph workbench.
- [x] Tests cover representative user, group, tag, autogroup, host, and IP selector expansion.
- [x] Tests cover port/protocol classification and perspective-subject filtering for representative users, groups, tags, and autogroups.

## Notes

- Added `internal/policy` as a conservative Phase 2 resolver for ACL rules.
- Topology snapshots now return ACL effective-access edges when Phase 2 policy data is loaded; Phase 1 inferred edges remain the fallback.
- Edge payloads now carry protocols, ports, access scope classification, policy references, and perspective provenance.
- Remaining visual styling work belongs in 011-graph-policy-preview-modes.md, which covers the current/draft/diff graph vocabulary.

## Blocked by

- 004-cloud-api-auth-policy-fetch.md
