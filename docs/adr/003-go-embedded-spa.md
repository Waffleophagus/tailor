# ADR 003: Go Backend with Embedded SPA

**Status**: Accepted

**Date**: 2026-05-25

## Context

We need a backend that can: (1) call the Tailscale LocalAPI, (2) call the Tailscale Cloud API, (3) serve a web frontend. We want a single deployable artifact with no external service dependencies in Phase 1.

## Decision

Build a **Go binary** using `net/http` that serves both the API and the embedded SPA via `//go:embed`.

The build pipeline is:
1. `npm run build` → generates `dist/` (Svelte + Cytoscape.js output)
2. `go build` → Go embeds `dist/*` into the binary via `//go:embed`
3. Single static executable: zero runtime dependencies, zero external asset files

## Alternatives Considered

### Separate Frontend Server (Node.js)
Run a Node.js server for the SPA and a Go server for the API. Proxy between them.
- **Cons**: Two runtimes, two processes, Docker complexity. We want a single artifact.

### Static File Server + API Proxy
Use nginx or Caddy as a reverse proxy. Go backend runs separately.
- **Cons**: Requires additional container/service. Violates "single binary" principle.

### WASM Go in the Browser
Compile Go to WASM and run entirely in the browser.
- **Cons**: Cannot access Tailscale LocalAPI Unix socket from browser context. Cannot call `api.tailscale.com` without CORS. Electron would be required, adding packaging complexity.

## Consequences

- The binary is self-contained. `scp` it anywhere, run it.
- `//go:embed` is a Go 1.16+ feature — widely supported.
- The binary size includes the entire frontend bundle (JS + CSS + HTML).
- No hot-reload in production. Development requires running both `vite dev` (frontend) and `go run` (backend) separately.
