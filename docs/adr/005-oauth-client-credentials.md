# ADR 005: OAuth Client Credentials for Cloud API Auth

**Status**: Accepted

**Date**: 2026-05-25

## Context

To edit Tailscale ACLs, we must authenticate to the Tailscale Cloud API. The API accepts two mechanisms: API Access Tokens (long-lived, user-scoped) and OAuth Client Credentials (short-lived tokens, machine-scoped). We need a model that is secure, standard, and easy for a Tailscale admin to set up.

## Decision

Support **OAuth 2.0 Client Credentials** as the primary Phase 2 authentication method.

The flow:
1. Admin creates an OAuth Client in Tailscale Admin Console → Settings → Keys → Trust Credentials
2. They assign scopes: `policy:read` and `policy:write`
3. They paste Client ID + Client Secret into the app
4. App exchanges credentials at `POST https://api.tailscale.com/api/v2/oauth/token` for a 1-hour Access Token
5. Access Token is stored in memory (not persisted to disk by default)
6. App auto-refreshes the token before expiry

API Access Tokens (`tskey-api-...`) are supported as a fallback.

## Alternatives Considered

### API Access Token Only
Require the user to generate a `tskey-api-...` token and paste it.
- **Cons**: Tokens expire (1-90 days). Must be manually regenerated. No auto-refresh. More friction for a DevRel demo.

### Browser Cookie Hijacking
Try to reuse the user's existing `login.tailscale.com` session cookie.
- **Cons**: Impossible. Same-Origin Policy and Tailscale's session architecture prevent this. The Cloud API does not accept browser session cookies.

### Local API Token Extraction
Query the Tailscale daemon for stored credentials.
- **Cons**: The LocalAPI exposes no mechanism to extract Cloud API tokens. Tokens are scoped to the control plane, not the local daemon.

## Consequences

- Users must be Tailscale Owners, Admins, Network Admins, or IT Admins to create OAuth Clients.
- The app must handle token expiry gracefully (auto-refresh or re-prompt).
- In-memory-only storage by default means token is lost on app restart. Optional encrypted file storage can be added later.
- The OAuth token endpoint (`api.tailscale.com/api/v2/oauth/token`) is a documented, stable Tailscale API.
