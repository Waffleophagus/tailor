# Structured policy map

Labels: ready-for-agent
Type: AFK

## What to build

Create the read-only structured policy map that turns the fetched Policy File into normalized sections the UI can inspect without raw HuJSON. This should cover the main policy surfaces Tailor needs to explain and eventually edit: ACLs, grants, SSH rules, groups, tag owners, hosts, IP sets, posture/device conditions, node attributes, auto-approvers, and tests where present.

The UI should present this as a policy workbench section navigator, not as the final editor. Raw HuJSON remains available as an advanced view.

## Acceptance criteria

- [ ] The backend exposes a structured policy map derived from the fetched HuJSON while preserving the original HuJSON for round-trip mutation.
- [ ] Unknown top-level sections are preserved and surfaced as unsupported sections rather than dropped.
- [ ] The frontend renders read-only policy sections with search, counts, empty states, and links back to raw HuJSON.
- [ ] Policy selectors are displayed consistently across sections: user, group, tag, autogroup, host, IP, IP set, and raw selector.
- [ ] Parse errors are shown as actionable UI errors without losing access to the raw policy text.
- [ ] Tests cover recognized sections, unknown section preservation, empty policy sections, and malformed HuJSON.

## Blocked by

- 004-cloud-api-auth-policy-fetch.md

