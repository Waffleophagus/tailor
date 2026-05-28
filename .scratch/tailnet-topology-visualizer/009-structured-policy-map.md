# Structured policy map

Labels: ready-for-human
Type: AFK

## What to build

Create the read-only structured policy map that turns the fetched Policy File into normalized sections the UI can inspect without raw HuJSON. This should cover the main policy surfaces Tailor needs to explain and eventually edit: ACLs, grants, SSH rules, groups, tag owners, hosts, IP sets, posture/device conditions, node attributes, auto-approvers, and tests where present.

The UI should present this as a policy workbench section navigator, not as the final editor. Raw HuJSON remains available as an advanced view.

## Acceptance criteria

- [x] The backend exposes a structured policy map derived from the fetched HuJSON while preserving the original HuJSON for round-trip mutation.
- [x] Unknown top-level sections are preserved and surfaced as unsupported sections rather than dropped.
- [x] The frontend renders read-only policy sections with search, counts, empty states, and links back to raw HuJSON.
- [x] Policy selectors are displayed consistently across sections: user, group, tag, autogroup, host, IP, IP set, and raw selector.
- [x] Parse errors are shown as actionable UI errors without losing access to the raw policy text.
- [x] Tests cover recognized sections, unknown section preservation, empty policy sections, and malformed HuJSON.

## Blocked by

- 004-cloud-api-auth-policy-fetch.md

## Notes

- Implemented with `GET /api/policy/map`.
- The workbench was read-only at ship time; the **Policy Workbench shell** ([019](019-policy-workbench-shell.md)) replaces the flat panel as the structured editing surface. See [018 roadmap](018-policy-scenario-roadmap.md).
- Unsupported sections are listed with their decoded raw value so future editors can preserve them.
