# Single binary and container packaging

Labels: ready-for-human
Type: AFK

## Status

Done.

## What to build

Package the Phase 1 app so a user can run one Go executable or a Docker Compose service and see the tailnet graph. The Go binary should serve the embedded SPA and REST API on a configurable port, with the Tailscale socket path configurable for local and container runs.

## Acceptance criteria

- [x] The production build embeds the Svelte SPA into the Go binary with `//go:embed`.
- [x] The binary supports a configurable listen port with default `:8080`.
- [x] The binary supports a configurable LocalAPI socket path.
- [x] A multi-stage Dockerfile builds the app without runtime Node.js dependencies.
- [x] A Docker Compose file mounts the Tailscale socket and starts the app.
- [x] Basic build documentation explains local binary and Docker Compose usage.

## Blocked by

- 001-localapi-status-graph.md

## Notes

- `pnpm --dir web build` writes the SPA bundle to `internal/frontend/dist`.
- `go build ./cmd/tailor` embeds that bundle in the binary.
- `Dockerfile` builds frontend assets and the Go binary in separate stages, then copies only the binary into the runtime image.
- `compose.yaml` mounts the host Tailscale socket read-only and publishes the app on port 8080.
