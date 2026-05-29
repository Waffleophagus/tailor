# Live tailnet E2E tests

Playwright tests against a running Tailor dev stack. **`pnpm test:e2e` starts the stack for you** when it is not already running.

## Prerequisites

1. Go toolchain (to build `./cmd/tailor`).
2. `web/.env` with an API key (see setup below).

Playwright’s `webServer` hook will:

1. `go build` the backend to `web/.cache/tailor-e2e` and run it on `:8080` (unless already healthy).
2. Start Vite on `TAILOR_E2E_BASE_URL` (default `http://127.0.0.1:5173`) proxying `/api` to the backend.

Locally, an already-running `./tailor` or `pnpm dev` is reused (`reuseExistingServer`). In CI, both are always started fresh.

Tests authenticate automatically via `POST /api/cloud/auth` in global setup — you do not need to enable ACL editing in the UI first.

## Setup

```sh
cd web
cp .env.example .env
```

### Demo tailnet (recommended for local dev)

The default `.env.example` uses the built-in demo key — no real Tailscale Cloud API calls:

```
TAILSCALE_API_KEY=tskey-api-tailor-dev
```

The backend serves a 14-device sample fleet with varied ACL scopes, draft evaluation, and in-memory validate/save.

### Real tailnet

Set `TAILSCALE_API_KEY=tskey-api-…` from the Tailscale admin console. Local tailscaled access is required for topology unless you use the demo key.

| Variable                     | Required | Default                            | Purpose                                       |
| ---------------------------- | -------- | ---------------------------------- | --------------------------------------------- |
| `TAILSCALE_API_KEY`          | yes      | —                                  | Enables ACL editing on the dev backend        |
| `TAILOR_TAILNET`             | no       | from `/api/topology` or demo       | Tailnet name for real keys only               |
| `TAILOR_E2E_BASE_URL`        | no       | `http://127.0.0.1:5173`            | App under test (Vite dev server)              |
| `TAILOR_E2E_TAILOR_URL`      | no       | `http://127.0.0.1:8080/api/health` | Backend health probe                          |
| `TAILOR_E2E_TAILOR_PORT`     | no       | `8080`                             | Backend listen port when Playwright starts it |
| `TAILOR_E2E_PERSPECTIVE`     | no       | `alice@demo.tailor.ts.net`         | Primary simulate subject (demo tailnet)       |
| `TAILOR_E2E_ALT_PERSPECTIVE` | no       | `bob@demo.tailor.ts.net`           | Secondary user for draft rules                |
| `TAILOR_E2E_DESTINATION`     | no       | `tag:web`                          | ACL destination selector                      |

`.env` is gitignored — never commit API keys.

## Run

```sh
cd web
pnpm test:e2e
```

Interactive mode:

```sh
pnpm test:e2e:ui
```

## What is covered

- Topology + policy fetch for an authenticated tailnet
- Workbench ACL staging, draft tray, simulate, validate (discard cleanup — no save)
- Scenario bar focused mode + ghost denied toggle
- `/api/policy/mutate`, evaluate-draft, and validate

## Graph styling (unit tests)

Every edge color/line-style variant is table-driven in `src/lib/graph/style-cases.ts` and tested with Vitest:

```sh
pnpm test
```

In dev, the running app exposes `window.__tailorGraphDebug()` — returns each visible edge’s classes and resolved `{ lineColor, lineStyle, … }` for spot-checking in the browser console or Playwright via `page.evaluate`.
