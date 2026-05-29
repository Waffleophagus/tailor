# Live tailnet E2E tests

Playwright tests against a running Tailor dev stack. **`pnpm test:e2e` starts the stack for you** when it is not already running.

## Prerequisites

1. Go toolchain (to build `./cmd/tailor` with `-tags dev`).
2. `web/.env` with an API key (see setup below).

Playwright’s `webServer` hook will:

1. `go build -tags dev` the backend to `web/.cache/tailor-e2e` and run it on `:8080` (unless already healthy).
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

The backend serves a 15-device sample fleet (including a super-admin debug user) with varied ACL scopes, draft evaluation, and in-memory validate/save.

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
| `TAILOR_E2E_SUPER_USER`      | no       | `group:superuser`                  | Broad _:_ ACL subject for graph debugging     |
| `TAILOR_E2E_SUPER_DEVICE`    | no       | `superadmin-console`               | Demo device owned by the super-user           |
| `TAILOR_E2E_DESTINATION`     | no       | `tag:web`                          | ACL destination selector                      |

`.env` is gitignored — never commit API keys.

### Dev-only spawn API

When running a **dev build** (`go build -tags dev`) and authenticated with `tskey-api-tailor-dev`:

```http
POST /api/dev/spawn-devices
Content-Type: application/json

{"count": 4, "prefix": "worker", "owner": "spawn@demo.tailor.ts.net", "os": "linux", "tags": ["tag:ci"]}
```

Per-device spawn (varied owners, tags, subnet routers, offline provisioning):

```http
POST /api/dev/spawn-devices
{"specs": [{"name": "k8s-prod-worker-04", "owner": "platform-ops@demo.tailor.ts.net", "tags": ["tag:k8s-prod"]}]}
```

Bring provisioned nodes online after spawn:

```http
POST /api/dev/patch-devices
{"devices": [{"name": "compliance-archive-primary", "online": true}]}
```

Returns the spawned devices plus the full demo fleet. New nodes appear on the topology websocket within ~2 seconds. This route is **not compiled into production builds** (`go build` without tags → 404).

Production vs dev backend builds (from `web/`):

```sh
pnpm backend:build      # release — no demo key, no /api/dev/*
pnpm backend:build:dev  # local dev, E2E, and demo tailnet
pnpm backend:run:dev    # run the dev binary (after build:dev)
pnpm dev:stack          # build:dev + run:dev (backend only; pair with pnpm dev)
pnpm dev:spawn          # staggered demo fleet rollout (~16 nodes, k8s waves, offline→online)
```

## Run

```sh
cd web
pnpm test:e2e
```

Interactive mode:

```sh
pnpm test:e2e:ui
```

### Production ACL save (real tailnet)

`pnpm test:e2e` **does not** run the production save test (it never writes to Tailscale Cloud). To exercise the full enable-auth → snapshot → mutate → save → revert loop against your real tailnet:

```sh
cd web
pnpm test:e2e:production
```

Requires a real `TAILSCALE_API_KEY` in `web/.env` (not `tskey-api-tailor-dev`). The test enables ACL editing through the UI, appends a reversible probe ACL rule, saves twice (change + revert), and restores the initial policy if it fails mid-run.

| Variable                        | Required | Purpose                                               |
| ------------------------------- | -------- | ----------------------------------------------------- |
| `TAILOR_E2E_SKIP_GLOBAL_AUTH`   | auto     | Set by `test:e2e:production` so the UI auth flow runs |
| `TAILOR_E2E_INCLUDE_PRODUCTION` | auto     | Set by `test:e2e:production` to run only that spec    |

## What is covered

- Topology + policy fetch for an authenticated tailnet
- Workbench ACL staging, draft tray, simulate, validate (discard cleanup — no save)
- Scenario bar focused mode + ghost denied toggle
- `/api/policy/mutate`, evaluate-draft, and validate
- **Production only** (`pnpm test:e2e:production`): real Cloud ACL save + revert round-trip

## Graph styling (unit tests)

Every edge color/line-style variant is table-driven in `src/lib/graph/style-cases.ts` and tested with Vitest:

```sh
pnpm test
```

In dev, the running app exposes `window.__tailorGraphDebug()` — returns each visible edge’s classes and resolved `{ lineColor, lineStyle, … }` for spot-checking in the browser console or Playwright via `page.evaluate`.
