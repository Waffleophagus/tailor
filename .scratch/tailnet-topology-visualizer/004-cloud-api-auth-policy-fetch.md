# Cloud API auth and policy fetch

Labels: ready-for-agent
Type: AFK

## Status

Done.

## What to build

Add the explicit Phase 2 trust escalation path. A user clicks "Enable ACL Editing", enters tailnet name plus a Tailscale API key (`tskey-api-...`), the backend keeps that key in memory, fetches the Policy File from the Cloud API, and exposes the raw HuJSON policy to the UI.

## Acceptance criteria

- [x] The frontend provides an "Enable ACL Editing" flow with tailnet name and Tailscale API key fields.
- [x] The backend stores the Tailscale API key only in memory by default.
- [x] The backend fetches the tailnet Policy File from the Cloud API after authentication.
- [x] The UI visibly distinguishes authenticated Phase 2 from unauthenticated Phase 1.
- [x] A Raw HuJSON tab displays the fetched policy.
- [x] Auth and policy-fetch failures surface actionable errors without exposing secrets.

## Notes

- Added `internal/cloudapi` for API key auth, in-memory session state, and raw HuJSON policy fetch.
- Added `GET /api/cloud/status`, `POST /api/cloud/auth`, and `GET /api/policy`.
- Added frontend Phase 2 entry controls, authenticated phase indicator, and a raw HuJSON policy panel.
- API keys are never persisted and are cleared from frontend state after successful authentication.

## Blocked by

- 001-localapi-status-graph.md
