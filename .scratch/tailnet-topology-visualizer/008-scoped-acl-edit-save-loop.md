# Scoped ACL edit, validate, and save loop

Labels: ready-for-human
Type: AFK

## What to build

Implement the smallest reversible Phase 2 edit path that can actually change the tailnet policy, validate the draft, save it through the Cloud API, and refresh the graph so visual access changes can be tested against a real tailnet.

This slice exists because preview-only validation is weak when the current tailnet is intentionally open. The user already has their own local backup process for development, so the app should focus on making one narrow ACL change, validating it, saving it, and then showing the effective-access graph update from the newly fetched policy.

Start with one scoped edit path rather than a general policy editor. Preferred first operation: add a new simple ACL rule with selected sources, selected destinations, and selected ports/protocols. This is easier to revert than mutating an existing rule in place and gives the graph a clear before/after delta.

## Acceptance criteria

- [x] The UI provides one scoped ACL edit form for adding a simple accept rule:
  - source selectors: user, group, tag, autogroup, or raw selector
  - destination selector: device/tag/host/IP/raw selector
  - ports: common presets for SSH (`22`), HTTP/S (`80,443`), and custom port entry
- [x] The backend creates a draft HuJSON policy by appending the new ACL rule while preserving existing comments and formatting outside the mutated ACL section.
- [x] The UI shows a before/after HuJSON diff or at minimum the exact rule to be added before validation.
- [x] `POST /api/policy/validate` validates the draft against the Tailscale Cloud API and returns actionable errors without exposing API keys.
- [x] `POST /api/policy/save` saves only the last successfully validated draft.
- [x] After save, the backend refreshes policy data from the Cloud API and topology snapshots use the updated effective-access edges.
- [x] The graph visibly updates after save, including access scope changes such as SSH, HTTP/S, broad, custom, or limited access.
- [x] Tests cover draft policy creation, validation/save request handling, secret scrubbing, and policy refresh after save.

## Blocked by

- 005-effective-access-edges.md

## Notes

- This intentionally prioritizes a real save loop over a perfect editor because real policy changes are needed to validate graph behavior on an open tailnet.
- The first edit operation should append a new rule rather than rewriting an existing rule, because append-only changes are easier for a human to inspect and revert.
- This does not replace the later Policy Lens editor or graph-backed policy workbench. It is the tracer bullet for safe edits, validation, save, and graph refresh.
- Implemented with `POST /api/policy/draft`, `POST /api/policy/validate`, and `POST /api/policy/save`.
- The UI currently shows the exact appended rule and draft HuJSON, not a side-by-side diff.
- Follow-up work should use the graph as the preview and review surface, with raw HuJSON available as an advanced audit view.
