# MCP ACL Reference Plan

## Goal

Add an MCP-accessible reference system that helps agents understand and safely edit Tailscale tailnet policy HuJSON before they evaluate or stage ACL changes.

The reference should not be one large document returned in full. It should be split into small, task-oriented topics so an agent can retrieve only the syntax, constraints, examples, and gotchas relevant to the edit it is about to make.

## Current Context

Tailor already exposes MCP tools from `internal/mcpserver/tools.go`:

- `tailor_get_tailnet_state`
- `tailor_get_policy`
- `tailor_evaluate_policy_draft`
- `tailor_stage_policy_draft`

The backend already has a structured policy map implementation:

- `internal/policy/policy.go`: `StructuredMap`
- `internal/tailorcore/core.go`: `PolicyMap`
- HTTP API: `GET /api/policy/map`

The ACL and HuJSON source-of-truth reference lives in `docs/tailscale refs.md`. The MCP reference topics should be curated slices derived from that document, not a separate competing reference.

## Design Principles

1. Keep documentation lookup read-only and available in read-only MCP mode.
2. Prefer compact, topic-specific responses over a single exhaustive payload.
3. Make the agent workflow explicit: read relevant docs, inspect policy map, draft, evaluate, then stage.
4. Treat `docs/tailscale refs.md` as the source of truth for Tailor's embedded ACL reference.
5. Keep derived MCP reference content static and embedded in the binary so MCP works without filesystem dependencies.
6. Include official source links and a validation date in every topic.
7. Structure content for model use, not human prose density.

## Proposed MCP Surface

### `tailor_get_policy_map`

Expose the existing structured policy map through MCP.

Purpose:

- Let agents inspect policy sections, counts, entries, unsupported sections, and raw section values.
- Avoid forcing agents to repeatedly parse the full HuJSON when they only need section inventory.

Input:

```json
{}
```

Output:

Use the existing `api.PolicyMapResponse`.

Notes:

- Requires Cloud API authentication because it reads the policy.
- Should return the same auth error behavior as `tailor_get_policy`.
- This should be available in read-only MCP mode.

### `tailor_acl_reference_index`

Return a compact index of reference topics.

Purpose:

- Help the agent decide which topic to fetch.
- Avoid listing huge topic bodies in MCP tool descriptions.

Input:

```json
{}
```

Output shape:

```json
{
  "version": "2026-06-04",
  "topics": [
    {
      "id": "grants",
      "title": "Grants",
      "useWhen": [
        "Adding or changing modern network access rules",
        "Adding application capabilities",
        "Using route filtering with via"
      ],
      "related": ["selectors", "posture", "tests", "gotchas"]
    }
  ]
}
```

### `tailor_acl_reference`

Return one reference topic by ID.

Input:

```json
{
  "topic": "grants"
}
```

Output shape:

```json
{
  "id": "grants",
  "title": "Grants",
  "lastValidated": "2026-06-04",
  "sourceUrls": [
    "https://tailscale.com/docs/reference/syntax/grants"
  ],
  "contentMarkdown": "..."
}
```

Behavior:

- Unknown topic returns an error with the valid topic IDs.
- Topic IDs are stable and lowercase.
- Content should be concise enough for direct model context.

### Optional: `tailor_acl_reference_search`

Keyword search over topic metadata and content.

Input:

```json
{
  "query": "autogroup:self tagged devices ssh"
}
```

Output shape:

```json
{
  "matches": [
    {
      "topic": "gotchas",
      "title": "Gotchas",
      "score": 7,
      "snippets": [
        "autogroup:self only applies to user-owned devices and does not match tagged devices."
      ]
    }
  ]
}
```

Implementation can be simple at first:

- Lowercase token matching.
- Score by number of query terms found.
- Return top 5 topics.

This is useful, but not required for the first implementation.

## Topic Taxonomy

The source document has 23 sections. The MCP reference should group them by editing intent.

### `overview`

Use for:

- Understanding policy file structure.
- Choosing grants over legacy ACLs.
- Remembering deny-by-default behavior.

Includes:

- Top-level sections.
- Access model summary.
- Grants preferred over ACLs.
- Rule evaluation at a high level.

### `hujson_editing`

Use for:

- Editing raw policy text.
- Preserving comments and trailing commas.
- Understanding what HuJSON does and does not allow.

Includes:

- Comments.
- Trailing commas.
- Invalid JSON5-style features.
- Diff-minimizing edit guidance.
- Comment preservation limits when using Tailscale API normalization.

### `grants`

Use for:

- Adding or changing modern access rules.
- Adding application capabilities.
- Using `via`.

Includes:

- Required fields: `src`, `dst`, and at least one of `ip` or `app`.
- `ip` syntax.
- `app` syntax.
- `via` restrictions.
- Union semantics.
- App capabilities require network access.

### `selectors`

Use for:

- Deciding whether a selector is legal in `src`, `dst`, tag owners, auto approvers, SSH, or tests.

Includes:

- User formats.
- `group:`, `tag:`, `autogroup:`, `ipset:`, `svc:`, `posture:`.
- Source vs destination restrictions.
- Exhaustive autogroup table.
- Protocol aliases can either live here or in `grants`.

### `legacy_acls`

Use for:

- Understanding existing `acls`.
- Migrating ACLs to grants.

Includes:

- ACL rule shape.
- `action: accept`.
- `src`, `proto`, `dst`.
- ACL destination port encoding.
- ACL to grant conversion examples.
- Limitations such as no `via` and no `app`.

### `ssh`

Use for:

- Adding or modifying Tailscale SSH rules.

Includes:

- SSH rule shape.
- `accept` vs `check`.
- Allowed `src`, `dst`, and `users`.
- `checkPeriod`.
- `acceptEnv`.
- SSH connection types allowed.
- Common `autogroup:nonroot` gotcha.

### `definitions`

Use for:

- Editing reusable policy definitions.

Includes:

- `groups`.
- `tagOwners`.
- `hosts`.
- `ipsets`.
- Group nesting restriction.
- Tag owner empty-array behavior.
- IP set composition with `add` and `remove`.

### `posture`

Use for:

- Adding or reasoning about device posture controls.

Includes:

- `postures`.
- `srcPosture`.
- `defaultSrcPosture`.
- Operators.
- Built-in posture attributes.
- Custom and integration attributes.
- Unset attribute behavior.
- Shared nodes and subnet-routed devices bypassing posture.

### `automation`

Use for:

- Auto-approving routes, exit nodes, or app connectors.
- Managing device-level attributes.

Includes:

- `autoApprovers`.
- Route, exit node, and app connector approval.
- Non-retroactivity.
- `nodeAttrs`.
- Funnel, NextDNS, randomize client port, disable IPv4.
- App connectors through `nodeAttrs`.

### `tests`

Use for:

- Adding validation tests to policy drafts.
- Proving intended allow and deny behavior.

Includes:

- `tests`.
- `sshTests`.
- `srcPostureAttrs`.
- ICMP tests.
- Destination restrictions.
- Why tests should accompany policy changes.

### `gotchas`

Use for:

- Pre-flight checks before evaluating or staging a policy draft.

Includes:

- `autogroup:self` does not match tagged devices.
- Grants are additive.
- App capabilities need network access.
- `defaultSrcPosture` is replacing, not additive.
- Auto approvers are not retroactive.
- `tagOwners` empty array does not mean nobody.
- CIDR grants do not inject routes.
- Test destinations cannot use CIDRs.
- IPv6 ACL destination formatting.

### `examples`

Use for:

- Getting compact examples by scenario.

Includes:

- Minimal allow-all.
- Self-access only.
- Group to tag access.
- Exit node routing with `via`.
- Posture-gated access.
- TailSQL app capability.
- IP sets.
- App connectors.
- SSH examples.
- Tests.

If this topic grows too large, split it into `examples_basic`, `examples_advanced`, and `examples_tests`.

## Reference Content Format

Store each topic as a model-facing markdown card:

```md
# Grants

Use when:
- Adding or changing modern network access rules.
- Adding application capabilities.
- Routing traffic through selected tagged routers with via.

Rules:
- A grant requires src and dst.
- Include at least one of ip or app.
- Grants are additive. A narrower grant does not override a broader grant.

Syntax:
...

Common mistakes:
- App-only grants do not provide useful access without network access.
- via only accepts tags.

Sources:
- https://tailscale.com/docs/reference/syntax/grants
```

Keep each topic focused. Dense tables are fine when they prevent ambiguity, but avoid copying the entire exhaustive document into every topic.

## Proposed File Layout

```text
internal/mcpserver/policyref/
  reference.go
  reference_test.go
  data/
    index.json
    topics/
      overview.md
      hujson_editing.md
      grants.md
      selectors.md
      legacy_acls.md
      ssh.md
      definitions.md
      posture.md
      automation.md
      tests.md
      gotchas.md
      examples.md
```

`reference.go` responsibilities:

- Embed `data/index.json` and topic markdown files.
- Parse and validate the index at init or test time.
- Return topic lists and topic bodies.
- Optionally provide search.

`tools.go` responsibilities:

- Register the new MCP tools.
- Keep all documentation tools available even when `TAILOR_MCP_READONLY=true`.
- Add `tailor_get_policy_map`.

## MCP Server Instructions Update

Update `internal/mcpserver/config.go` instructions to make docs part of the default workflow.

Current instruction:

```text
Inspect Tailor tailnet topology and stage ACL policy drafts for human review. Never save or upload ACL policy to Tailscale.
```

Proposed instruction:

```text
Inspect Tailor tailnet topology and stage ACL policy drafts for human review. Before modifying policy HuJSON, read the relevant ACL reference topic, inspect the policy map when available, evaluate the draft, then stage it for human review. Never save or upload ACL policy to Tailscale.
```

## Implementation Sequence

### Phase 1: Backend MCP reference tools

1. Create `internal/mcpserver/policyref`.
2. Split `docs/tailscale refs.md` into the topic files listed above.
3. Add index parsing and topic lookup.
4. Register `tailor_acl_reference_index`.
5. Register `tailor_acl_reference`.
6. Add unit tests for:
   - every index topic has a markdown file
   - every markdown file is listed in the index
   - unknown topic produces a useful error
   - returned topic includes sources and content

### Phase 2: Policy map MCP tool

1. Add `tailor_get_policy_map`.
2. Reuse `core.PolicyMap(ctx)`.
3. Mirror Cloud API auth error handling from `tailor_get_policy`.
4. Add tests if there is already a convenient MCP tool registration test harness. Otherwise cover this through service-level tests or a focused handler test.

### Phase 3: Optional search

1. Add `tailor_acl_reference_search`.
2. Implement simple token search over title, use-when text, and markdown body.
3. Return top matches with short snippets.
4. Add unit tests for common queries:
   - `autogroup:self`
   - `via`
   - `ssh nonroot`
   - `posture unset`
   - `ipset remove`

### Phase 4: Workflow hardening

1. Update MCP server instructions.
2. Consider adding stronger tool descriptions:
   - `tailor_stage_policy_draft` should remind the agent to evaluate first.
   - `tailor_evaluate_policy_draft` should remind the agent to use reference topics for syntax questions.
3. Consider exposing reference topic IDs in tool descriptions only as a short list.

## Testing Plan

Backend-only change:

```bash
go test ./...
```

If generated API docs or frontend code is touched, also run from repo root:

```bash
pnpm --dir web format && pnpm --dir web lint
pnpm --dir web check && pnpm --dir web test
```

This plan does not require frontend changes unless we decide to display the MCP reference in the Tailor UI.

## Open Questions

1. Should the reference be exposed as MCP resources in addition to tools?

   Tools are more reliably used by agents. Resources are semantically cleaner. The first implementation should use tools. Add resources later if client support proves useful.

2. Should `examples` be one topic or several?

   Start with one compact `examples` topic. Split it if the response becomes too large.

3. Should topic files be generated from `docs/tailscale refs.md` or manually curated from it?

   Start with manually curated topic files derived from `docs/tailscale refs.md`. Generation is unnecessary until the reference changes often, and manual curation lets us optimize each card for agent retrieval.

4. Should the server include a direct "preflight checklist" tool?

   Not initially. The `gotchas` topic plus draft evaluation should cover the need. A checklist tool can be added after observing agent behavior.

## Acceptance Criteria

- Agents can discover available ACL reference topics through MCP.
- Agents can fetch one topic without receiving the entire reference.
- Agents can inspect the current policy through `tailor_get_policy_map`.
- MCP read-only mode still exposes all documentation and inspection tools.
- Draft-modification workflow is documented in MCP server instructions.
- Backend tests pass with `go test ./...`.
