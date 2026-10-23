# ADR 007: Identity-Based Editing Access

**Status**: Accepted

**Date**: 2026-10-22

**Supersedes**: the bootstrap-session fallback in `docs/tsnet-access-control-plan.md`; refines ADR 002.

## Context

In tsnet mode Tailor authorized editors only through its app capability grant. Before that grant existed, nobody could be recognised as an editor, so Tailor let *anyone* submit a Cloud API key and, if appending the grant failed, gave that browser a 15-minute bootstrap session with policy write access. This had problems:

- Any viewer could replace the process-wide API key with one for **their own** tailnet. Combined with the bootstrap fallback, this could be turned into temporary write access to the real tailnet.
- Possessing *an* API key is weak evidence of authority over *this* tailnet.
- The bootstrap and setup-session cookies were a second, time-limited authorization system that was easy to get subtly wrong.

Tailscale already has a notion of who may edit the policy file: Owners, Admins and Network admins. Tailor can read user roles through the Cloud API.

## Decision

1. **Mirror Tailscale's roles.** In tsnet mode a caller is `full` if the app capability grants them `admin`, **or** if their WhoIs login is an Owner, Admin or Network admin (active or idle) in the Cloud API user list and their device is untagged. Tagged devices act for no person and can only be authorized by the app capability. The user list is cached for ~30 seconds; lookups fail closed.
2. **Vet API keys before storing them.** In tsnet mode a submitted key must see Tailor's own node in its device list (so it belongs to Tailor's tailnet). If the submitter is not already an editor, the key's own user list must show them as an Owner, Admin or Network admin. Rejected keys are never stored.
3. **Only editors replace a configured key.** Once a key is configured, viewers get 403 on `/api/cloud/auth`.
4. **The app capability grant becomes optional.** Editors are offered the recommended grant to make access explicit in policy and to delegate it (to other users or `tag:tailor-acl-editor` devices); it also keeps access working if the Cloud API is unreachable. Grant failures are reported as errors.
5. **Remove the bootstrap session and setup-session cookie.**
6. **Bind MCP sessions to identity.** Each `Mcp-Session-Id` is bound to the login and node that initialized it, and each tool call is authorized with the role resolved for *that* HTTP request (the SDK otherwise reuses the initialize request's context for the life of the session).
7. **Block DNS rebinding where there is no identity layer.** In local and external modes, requests whose `Host` is not an IP, `localhost`, a single-label or local-only name, `*.ts.net`, or listed in `TAILOR_ALLOWED_HOSTS` are rejected.

Local and external modes (no tsnet identity) keep treating every caller as `full`; giving them real identities is tracked separately.

## Consequences

- First-run setup is one step for admins: enter the key, then optionally add the grant.
- Editing in tsnet mode depends on the Cloud API for role lookups unless the app capability grant is present; when the API is unavailable, admins without the grant fall back to view-only.
- Demoting an admin in Tailscale takes effect within the cache TTL (~30 s) for UI/API and on the next call for MCP sessions.
- Deployments behind a reverse proxy under a public domain in local/external mode must set `TAILOR_ALLOWED_HOSTS`.
