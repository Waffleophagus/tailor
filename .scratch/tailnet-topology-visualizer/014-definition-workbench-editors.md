# Definition workbench editors

Labels: ready-for-agent
Type: AFK

## What to build

Structured editors for reusable policy **definitions** inside the Policy Workbench DEFINITIONS nav — matching Tailscale's Groups, Tags, IP sets, and Hosts sections. Each editor includes where-used context and graph preview when selector resolution applies.

Reference screenshots: `5-groups-general.png` through `12-hosts-advanced.png` in [`screenshots of ACL in site/`](../../screenshots%20of%20ACL%20in%20site/).

Master roadmap: [018-policy-scenario-roadmap.md](018-policy-scenario-roadmap.md) Phase 3.

## Section routes (DEFINITIONS nav)

| Route | Policy section | Tailscale pattern |
|-------|----------------|-------------------|
| Groups | `groups` | Table: name, size, members; autogroups reference table (read-only counts) |
| Tags | `tagOwners` | Table: tag, owners |
| IP sets | `ipsets` | Table + add form with target validation |
| Hosts | `hosts` | Table: name → IP/CIDR |

## Acceptance criteria

- [ ] Groups: create/edit with member validation, where-used references, "View as group:X" → scenario bar.
- [ ] Tags: create/edit tag owners with owner selector validation and affected device count.
- [ ] Hosts: create/edit with IP/CIDR validation and selector resolution preview on graph.
- [ ] IP sets: create/edit with target validation and where-used references.
- [ ] Definition edits stage to draft only; graph updates via evaluate-draft when scenario active.
- [ ] Unknown or unsupported definition shapes fall back to structured JSON or raw HuJSON.
- [ ] Simulation tier badge: **Graph-simulated** for hosts/IP sets/groups/tags (selector resolution).
- [ ] Tests cover at least one create and one edit path per supported definition type.

## Blocked by

- [019-policy-workbench-shell.md](019-policy-workbench-shell.md)
- [009-structured-policy-map.md](009-structured-policy-map.md) (done)
- [010-draft-policy-evaluation-api.md](010-draft-policy-evaluation-api.md) (done)
- [016-staged-commit-tray-and-hujson-diff.md](016-staged-commit-tray-and-hujson-diff.md) (for save flow; editing can start before tray ships)

## Blocks

- [022-scenario-centric-editing.md](022-scenario-centric-editing.md) (partial — definition edits trigger re-eval)
