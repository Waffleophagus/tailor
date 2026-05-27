prep the project, we will want to init a go project if it has not been initted yet, and use pnpm as our package manager for the front end, installing all the necessary dependencies, please use https://github.com/dmmulroy/better-result for error handling and transactions with the server, and zod for schema validation, and lets generate types via https://github.com/hey-api/openapi-ts for the front end. Take a look and use https://github.com/coder/guts to generate types for our front end.

## Status

Done.

## Notes

- Initialized the Go module as `github.com/Waffleophagus/tailor`.
- Added a Svelte/Vite TypeScript frontend under `web`, managed by pnpm.
- Installed `better-result`, `zod`, `cytoscape`, and `@hey-api/openapi-ts`.
- Added a Go backend entry point at `cmd/tailor` with `/api/health` and `/api/topology`.
- Added embedded SPA serving through `internal/frontend`.
- Added `docs/openapi.yaml` and `web/openapi-ts.config.ts` for generated API clients.
- Added `tools/generate-types` using `github.com/coder/guts` to generate frontend TypeScript types from `internal/api`.
