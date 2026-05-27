# PRD: Tailnet Topology Visualizer & ACL Editor

## 1. Overview

A self-hosted Go binary with an embedded Svelte+Cytoscape.js frontend that visualizes a user's Tailscale tailnet as an interactive force-directed graph. Phase 1 requires zero credentials and renders all visible devices. Phase 2 requires a Tailscale Cloud API token and unlocks ACL editing with staged commit review, perspective filtering, and effective access visualization.

## 2. Goals

- **Instant "wow" factor**: Run the binary, open a browser, and see your entire tailnet as a physics-simulated graph within 2 seconds.
- **Read-only topology discovery (Phase 1)**: Works with only the LocalAPI — no API keys, no auth, no Cloud API dependency.
- **Safe ACL editing (Phase 2)**: Click a device, open a modal, modify tags/groups, preview a diff, validate against the Cloud API, and commit changes.
- **Perspective simulation**: Let an admin view the graph as a selected policy subject (user, group, tag, or autogroup) to understand what that subject can reach without authenticating as that user.
- **Preserve admin intent**: Round-trip HuJSON mutation using Tailscale's official `hujson` library. Admin comments and formatting are preserved.
- **No active probing**: Never perform `ping`, `nc`, port scans, or any active network reconnaissance.

## 3. Architectural Principles

- **Single binary**: One Go executable serves both the REST API and the static SPA via `//go:embed`.
- **No external services in Phase 1**: The LocalAPI (`/var/run/tailscale/tailscaled.sock`) is the only backend dependency.
- **Explicit trust escalation**: Phase 2 requires a deliberate action (pasting a Tailscale API key) and is visually distinct from Phase 1.
- **Validate before save**: All policy mutations must pass `POST /api/v2/tailnet/-/acl/validate` before being pushed.
- **No hidden state**: The raw HuJSON policy file is always inspectable via a "Raw" tab. No magic translations or silent rewrites.

## 4. User Flows

### 4.1 Phase 1: Topology Discovery (Unauthenticated)

1. User runs `./tailnet-viz` (or `docker run ...`).
2. Go backend attempts to connect to the Tailscale LocalAPI.
3. If LocalAPI is unreachable → show "Connect to Tailscale" screen with instructions.
4. If LocalAPI is reachable → fetch `tailscale status --json` equivalent.
5. Parse into device nodes: name, Tailscale IP, OS, online status, tags, owner email.
6. Render nodes in Cytoscape.js with D3-Force layout.
7. Sidebar filters:
  - Show/hide offline devices
  - Show/hide subnet-routed devices
  - Color by tag / owner / OS
8. User clicks a node → detail panel shows:
  - Device metadata
  - Tags (if any)
  - Owner
  - "ACL editing requires authentication" banner

### 4.2 Phase 2: ACL Editing (Authenticated)

1. User clicks "Enable ACL Editing" (prominent button in Phase 1).
2. Modal prompts for:
  - Tailnet name (defaults to `-`)
  - Tailscale API key (`tskey-api-...`)
3. Backend keeps the key in memory only.
4. Backend fetches policy file via `GET /api/v2/tailnet/{tailnet}/acl`.
5. Backend resolves effective access rules into device-to-device edges with allowed port/protocol scopes and rule provenance.
6. Graph updates: edges now reflect actual reachability, colored by port/protocol and visually distinguish full access from limited access such as HTTPS-only without SSH.
7. User clicks "Perspective Filter" → dropdown of users, groups, tags, and autogroups.
8. Selecting a perspective simulates the graph from that policy subject's effective access. This is policy simulation only, not credential impersonation. Exact visual treatment of inaccessible edges and nodes is deferred to interactive UI prototyping.
9. User clicks a device node → modal:
  - Current tags
  - Matching policy subjects (owner user, owner groups, tags, autogroups, hosts/IPs)
  - "What can reach this?" list (resolved from ACL file)
  - "What can this reach?" list
  - Responsible ACL rules/grants for each effective access path
  - "Edit Policy" button
10. User clicks "Edit Policy" → opens a policy editing interface scoped to the matching groups, tags, ACL rules, and grants.
11. Changes are accumulated into a draft state.
12. User clicks "Review & Save" → diff viewer shows old HuJSON vs. new HuJSON.
13. User clicks "Validate" → backend sends to `POST /api/v2/tailnet/{tailnet}/acl/validate`.
14. If validation fails → show errors with line numbers.
15. If validation passes → "Save" button enables.
16. User clicks "Save" → backend sends `POST /api/v2/tailnet/{tailnet}/acl`.
17. On success → toast notification, graph refreshes.

## 5. Functional Requirements

### 5.1 Backend (Go)

| ID | Requirement | Priority |
|---|---|---|
| BE-01 | Connect to Tailscale LocalAPI Unix socket (`/var/run/tailscale/tailscaled.sock` on Linux; handle Windows/macOS alternatives) | P0 |
| BE-02 | Parse `tailscale status` JSON into an internal device model | P0 |
| BE-03 | Serve SPA static files via `//go:embed` on a configurable port (default `:8080`) | P0 |
| BE-04 | Expose `GET /api/status`, `GET /api/topology`, and a topology/policy WebSocket at `GET /api/topology/socket` | P0 |
| BE-05 | Store Tailscale API keys in memory (not persisted to disk by default) | P0 |
| BE-06 | Implement API key authentication for Cloud API requests | P0 |
| BE-07 | Fetch and cache the tailnet policy file from Cloud API | P0 |
| BE-08 | Resolve ACL rules into effective access edges with group, tag, autogroup, host/IP, port, protocol, and provenance expansion | P0 |
| BE-15 | Support policy-perspective simulation for selected users, groups, tags, and autogroups without user credential impersonation | P0 |
| BE-09 | Parse HuJSON using `github.com/tailscale/hujson` AST (lossless round-trip) | P0 |
| BE-10 | Marshal modified ACL structs back to HuJSON preserving comments | P0 |
| BE-11 | Call Cloud API `validate` endpoint before saving | P0 |
| BE-12 | Support `--port` and `--socket-path` CLI flags | P1 |
| BE-13 | Support OAuth Client Credentials as an optional later auth path | P1 |
| BE-14 | Cache device list for configurable TTL | P1 |

### 5.2 Frontend (Svelte + Cytoscape.js)

| ID | Requirement | Priority |
|---|---|---|
| FE-01 | Cytoscape.js graph with D3-Force layout: nodes = devices, edges = relationships | P0 |
| FE-02 | Dark mode default (Obsidian-inspired) | P0 |
| FE-03 | Sidebar filter panel: tags, owners, OS, online status | P0 |
| FE-04 | Click node → detail panel with metadata | P0 |
| FE-05 | "Enable ACL Editing" button with API key modal | P0 |
| FE-06 | Perspective filter: select user, group, tag, or autogroup to recalculate the graph around that subject's effective access | P0 |
| FE-07 | Color-coded edges: SSH, HTTP/S, custom access, limited/partial access, and blocked/no-access states must be distinguishable | P1 |
| FE-08 | Node policy lens modal: matching policy subjects, responsible rules/grants, scoped policy editing | P1 |
| FE-09 | Diff viewer: side-by-side old/new HuJSON with syntax highlighting | P1 |
| FE-10 | Raw HuJSON tab (read-only in Phase 1, editable in Phase 2) | P1 |
| FE-11 | Toast notifications for save/validate events | P2 |
| FE-12 | Zoom, pan, fit-to-view controls | P0 |

### 5.3 Packaging

| ID | Requirement | Priority |
|---|---|---|
| PKG-01 | Single Go binary, zero runtime dependencies | P0 |
| PKG-02 | `Dockerfile`: multi-stage build, Alpine or distroless base | P0 |
| PKG-03 | `docker-compose.yml` with volume mount for Tailscale socket | P0 |
| PKG-04 | Electron wrapper (later): main process spawns Go binary, loads `localhost:<port>` in window | P2 |
| PKG-05 | GitHub Actions: build binary + Docker image on tag | P2 |

## 6. Non-Goals

- No enterprise scale optimization beyond Cytoscape.js's comfortable range (~500-1000 nodes)
- No active network probing (`ping`, `nmap`, port scanning)
- No real user credential impersonation or auth token extraction from Tailscale daemon. Admin-only policy-perspective simulation is in scope.
- No SSH key management or device provisioning
- No Tailscale DNS management (out of scope)
- No multi-tailnet switching (one instance per tailnet)

## 7. Success Criteria

- A user can download the binary, run it, and see their tailnet graph in under 10 seconds without any configuration.
- An admin can paste a Tailscale API key, edit a tag, validate, save, and see the change reflected in the Tailscale admin console within 60 seconds.
- The HuJSON diff viewer preserves all original comments and formatting except for the mutated sections.
