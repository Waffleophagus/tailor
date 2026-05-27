# LocalAPI status graph tracer bullet

Labels: ready-for-human
Type: AFK

## Status

Done.

## What to build

Build the first end-to-end Phase 1 path: the app starts, connects to the Tailscale LocalAPI, reads Tailscale Status, serves the SPA, and renders visible Devices as a Cytoscape.js graph. When LocalAPI is unavailable, the UI shows a connect-to-Tailscale state instead of failing silently.

This slice should prove the single-binary shape, the REST API boundary, the Device model, and the first graph render using real LocalAPI data when available.

## Acceptance criteria

- [x] The backend exposes `GET /api/status`, `GET /api/topology`, and `GET /api/topology/socket` for LocalAPI connectivity and normalized Device data.
- [x] The backend can use a configurable LocalAPI socket path and reports a clear unavailable state when the socket cannot be reached.
- [x] The frontend renders Device nodes in Cytoscape.js from the topology socket's initial snapshot.
- [x] The frontend shows a connect-to-Tailscale state when LocalAPI is unavailable.
- [x] Tests cover Tailscale Status JSON parsing into the internal Device model.
- [x] No active network probing is introduced.

## Notes

- Phase 1 live updates now flow over `GET /api/topology/socket`.
- `GET /api/topology` remains as a snapshot endpoint for the same normalized topology shape.
- The socket sends `topology.snapshot` payloads when LocalAPI is available and `localapi.unavailable` payloads when the socket cannot be reached.

## Blocked by

None - can start immediately
