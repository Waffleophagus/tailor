# ADR 006: API Key for Cloud API Auth

**Status**: Accepted

**Date**: 2026-05-26

## Context

Phase 2 needs Cloud API access to fetch and later validate/save the tailnet policy file. ADR 005 selected OAuth Client Credentials, but that adds setup friction before the app has a full staged editing loop.

Tailscale API access tokens use the `tskey-api-` prefix and can call the same Cloud API policy endpoints when granted sufficient permissions.

## Decision

Use a pasted **Tailscale API key** as the primary Phase 2 authentication method.

The flow:

1. Admin generates an API access token in the Tailscale admin console.
2. Admin clicks "Enable ACL Editing" in Tailor.
3. Admin enters tailnet name and the `tskey-api-...` key.
4. Tailor stores the key in backend memory only.
5. Tailor uses the key as HTTP Basic auth username with an empty password for Cloud API requests.
6. Tailor clears the key from frontend state after successful policy fetch.

## Consequences

- Phase 2 setup is much simpler for local development and early product validation.
- Keys are not persisted by default, so restarting Tailor requires re-entering the key.
- The key's lifetime and privileges are controlled outside Tailor in the Tailscale admin console.
- OAuth Client Credentials can still be added later if short-lived machine credentials become worth the extra setup.
