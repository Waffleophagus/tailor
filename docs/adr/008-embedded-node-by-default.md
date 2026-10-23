# ADR 008: Embedded Node by Default, Joined by Interactive Login

**Status**: Accepted

**Date**: 2026-10-22

## Context

Tailor could join the tailnet as its own machine (embedded tsnet) only when given a `tskey-auth-` key; otherwise `auto` mode used the host's `tailscaled`. Creating auth keys is extra friction, and the host-daemon mode has no per-request identity, so it cannot use the identity-based access model from ADR 007. Tailscale's normal login flow works for tsnet nodes too: an unenrolled node exposes a login URL.

## Decision

- `auto` (the default) always runs the embedded node. The host daemon is used only with `TAILOR_TAILSCALE_MODE=external` or an explicit `TAILOR_LOCALAPI_SOCKET`/`TAILOR_LOCALAPI_ENDPOINT`. It stays supported but is not the focus.
- Without an auth key, the node is logged in interactively. Until it is running, the local listener serves a setup page.
- The Tailscale login link is shown only to requests carrying a random setup token, which Tailor prints in its logs as `http://localhost:<port>/?setup=<token>`. The request must also come from a loopback, private, link-local or tailnet client, and its `Host` must pass the DNS-rebinding allow-list. Client addresses alone are not enough, because a reverse proxy or NAT makes every visitor look local. The token's trust boundary matches the logs, which contain the login URL anyway. ADR 007's key check would also reject API keys from another tailnet.
- The node is watched for its whole lifetime. If it needs login again (e.g. key expiry) and has no login URL, Tailor starts a new interactive login and logs the fresh setup link.
- Auth keys remain supported for unattended installs.
- State lives in `/var/lib/tailor-tsnet` in containers (or where a node already exists) and in the user config directory for desktop binaries.
- Like CLIProxyAPI, tsnet log upload is disabled and `TS_DEBUG_REGISTER` is ignored.
- Dev and E2E scripts run the demo tailnet with `TAILOR_TAILSCALE_MODE=external` (no identity layer).

## Consequences

- Upgrading binary users get the setup page instead of silently using their machine's Tailscale; `TAILOR_TAILSCALE_MODE=external` restores the old behaviour.
- Interactively added nodes are user-owned and inherit key expiry; Tailor warns and recommends tagging the node or disabling expiry.
