# SSH rules editor

Labels: ready-for-agent
Type: AFK

## What to build

Implement **Tailscale SSH** as a POLICY nav item in the Policy Workbench. Admins define who can SSH, destinations, login users, check/accept behavior, and advanced fields (posture, via) where present in the policy shape.

Reference: [`3-ssh-general.png`](../../screenshots%20of%20ACL%20in%20site/3-ssh-general.png), [`4-ssh-advanced.png`](../../screenshots%20of%20ACL%20in%20site/4-ssh-advanced.png).

Master roadmap: [018-policy-scenario-roadmap.md](018-policy-scenario-roadmap.md) Phase 4.

**Note:** Device posture *definitions* live under DEFINITIONS ([017](017-advanced-policy-section-coverage.md)); this issue covers SSH *rules* that reference them.

## Acceptance criteria

- [ ] SSH rule list + add/edit form inside workbench POLICY → Tailscale SSH route.
- [ ] Fields: source, destination, users, action, check mode where applicable.
- [ ] UI clearly distinguishes **network reachability** (ACLs/grants) from **SSH permission** — SSH alone does not imply network path on graph.
- [ ] Advanced controls: posture reference, via routing when fields present in policy.
- [ ] Generated changes stage to draft; graph shows network path where eval supports it; SSH permission surfaced in Policy Lens.
- [ ] Unsupported SSH rule shapes fall back to raw HuJSON.
- [ ] Simulation tier badge: **Graph-partial**.
- [ ] Tests cover SSH rule generation, SSH/network distinction, validation errors.

## Blocked by

- [019-policy-workbench-shell.md](019-policy-workbench-shell.md)
- [010-draft-policy-evaluation-api.md](010-draft-policy-evaluation-api.md) (done)

## Blocks

- [013-policy-lens-provenance-editor.md](013-policy-lens-provenance-editor.md) (partial — SSH vs network in lens)
