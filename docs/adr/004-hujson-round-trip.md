# ADR 004: HuJSON Round-Trip via AST Hybrid

**Status**: Accepted

**Date**: 2026-05-25

## Context

Tailscale ACL files are written in HuJSON (Human JSON), a JSON superset that permits comments and trailing commas. Standard `encoding/json` in Go rejects them. Admin teams often add inline comments explaining why a rule exists. We want to edit the policy file programmatically while preserving those comments.

## Decision

Use **Tailscale's official `github.com/tailscale/hujson` library** with a **hybrid AST approach**:

1. Parse the original HuJSON into an AST (comments preserved)
2. `Clone()` the AST
3. `Standardize()` the clone (strip comments, remove trailing commas)
4. `Pack()` and `json.Unmarshal` into typed Go structs for editing
5. Modify the Go structs
6. Marshal back to JSON
7. Generate an RFC 6902 JSON Patch between original JSON and modified JSON
8. Apply the patch to the **original AST** (which still has comments)
9. `Pack()` original AST → round-tripped HuJSON with comments intact

## Alternatives Considered

### Direct Struct Unmarshal (Standard Go `json`)
Pre-process with `hujson.Standardize()` to strip comments, unmarshal into structs, marshal back.
- **Cons**: Admin comments are permanently lost on first save. Formatting is destroyed.

### Direct AST Manipulation Only
Never convert to Go structs. Navigate the `hujson.Value` tree and mutate `Object`/`Array` nodes directly.
- **Cons**: No type safety. Easy to produce malformed policy files. Error-prone for nested structures.

### Treat as Opaque Text
Store the raw HuJSON string. Only append rules by string manipulation.
- **Cons**: Fragile regex/string hacking. Will corrupt the file on edge cases.

## Consequences

- We depend on Tailscale's `hujson` library, which is currently maintained and stable.
- The patch/round-trip flow is more code than standard JSON unmarshal, but it preserves admin intent.
- New rules appended to the `acls` array will not have comments attached (since they are new). Existing rule comments are preserved.
