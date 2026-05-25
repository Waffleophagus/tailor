# ADR 002: Two-Phase Authentication Model

**Status**: Accepted

**Date**: 2026-05-25

## Context

Tailscale has two entirely separate API surfaces: the LocalAPI (Unix socket, no auth) and the Cloud API (HTTPS, requires OAuth/API key). The LocalAPI gives device topology but not ACL policy. The Cloud API gives ACL policy but requires admin credentials. We want users to see value immediately without pasting a key.

## Decision

Implement a **two-phase application model**:

- **Phase 1**: Unauthenticated. Uses LocalAPI only. Renders all devices with inferred topology edges (shared owner, shared tag, subnet relationships). No ACL resolution.
- **Phase 2**: Authenticated. Uses Cloud API with an OAuth Client token. Unlocks effective access edges, perspective filtering, and ACL editing.

## Alternatives Considered

### Single Phase (Auth-First)
Require an API key on first launch before showing any graph.
- **Cons**: Kills the "wow" factor. Most Tailscale users are members, not admins, and cannot generate keys.

### Implicit Trust Escalation
Assume that being on the tailnet implies admin access and try to "hijack" an existing Tailscale browser session.
- **Cons**: Impossible. Tailscale daemon sessions do not expose Cloud API tokens. Browser Same-Origin Policy prevents cross-site token extraction.

### OAuth Forwarding via TSIDP
Use Tailscale as an OIDC provider to authenticate the user, then infer API access.
- **Cons**: Tailscale IDP authenticates identity, not admin privileges. An OIDC token from TSIDP cannot be exchanged for a Cloud API token with ACL scopes.

## Consequences

- The app must clearly communicate what Phase 1 can and cannot show.
- A prominent "Enable ACL Editing" CTA bridges the phases.
- Phase 2 requires explicit admin action: creating an OAuth Client in the Tailscale admin console.
