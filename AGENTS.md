## Project

This is tailor, it has a simple goal: to map out a tailscale tailnet and allow you to visualize changes to your ACL policies. The goal is to have as simple an interface as possible to simplify the ability for the user to, at a glance, see how their tailnet is structured and what devices/users can talk to.


## Skill

The `impeccable` skill (from `skills-lock.json`) is available for UI work. Use it, abuse it, the skills are invaluable.

## Issue tracker

Issues are tracked in-repo as needed. See `docs/agents/issue-tracker.md` for the current workflow.

### Triage labels

`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.


## Rules
When you are done modifying anything in the front end, run the following from `web/` (or `pnpm --dir web …` from the repo root) in order
```
pnpm format && pnpm lint
```
then fix issues, and then run
```
pnpm check && pnpm test
```
and make sure you have not introduced issues.

Graph edge styling is covered by Vitest (`pnpm --dir web test`) — see `web/src/lib/graph/style-cases.ts`.

Live tailnet E2E tests (Playwright runs `pnpm backend:e2e` + Vite when not already running; set `TAILSCALE_API_KEY` in `web/.env` — see `web/.env.example`; use `tskey-api-tailor-dev` for the built-in demo tailnet):
```
pnpm --dir web test:e2e
```

Production ACL save (`pnpm --dir web test:e2e:production`) hits a real tailnet and is **not** run in CI.

GitHub Actions (`.github/workflows/ci.yml`) runs lint, check, Vitest, `go test` (+ `-tags dev`), demo-tailnet E2E, build, and Docker smoke on every push/PR. Pushes to `main` also publish a semver release (starting at `v0.1.0`): git tag, GitHub Release with cross-platform production binaries, and `ghcr.io/<repo>` Docker tags (`:version`, `:vversion`, `:latest`) via `GITHUB_TOKEN`.

Backend scripts (from `web/`): `pnpm backend:build` (release), `pnpm backend:build:dev` + `pnpm backend:run:dev` (demo tailnet), `pnpm backend:test:dev`.
See `web/e2e/README.md`.