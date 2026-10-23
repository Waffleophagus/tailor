https://github.com/user-attachments/assets/385ca1b3-d3f5-44b7-9e8e-10170bee532f

<h1 align="center">Tailor</h1>
<p align="center"><em>Visualize and edit your Tailscale tailnet's access controls.</em></p>


## What is Tailor?

Tailor is a self-hosted visual tool for [Tailscale](https://tailscale.com) administrators. It maps out your tailnet as an interactive, explorable graph and lets you edit your ACL policy with validation and a live preview. An MCP server lets AI agents inspect your tailnet and draft policy changes for your review.

- **Visualize your tailnet** — See every device as a live, force-directed graph that updates in real time as devices come and go.
- **Filter and colorize** — Filter by tag, owner, OS, online status, or subnet-router role. Color nodes by status, tag, owner, or OS.
- **Inspect devices and access** — Click any node to see its owner, tags, IPs, OS, and which other devices it can reach (when authenticated with the Cloud API).
- **Edit ACL policies** — Authenticate with a Tailscale API key to fetch your tailnet's HuJSON policy. Edit directly, validate against Tailscale's Cloud API, preview the result on the graph, stage the draft, and save when you are ready. When Tailor is deployed as a Tailscale node, editing is gated by your tailnet policy so only authorized callers can modify ACLs.
- **AI agent integration** *(optional)* — Connect compatible agents via the built-in MCP server. Agents can inspect your topology, read embedded ACL reference docs, draft policy changes, evaluate their impact, and stage them for your review. When deployed as a Tailscale node, agents are authorized through your tailnet policy instead of separate bearer tokens.

## How it works

Tailor is a single Go binary with an embedded Svelte frontend and an optional MCP endpoint. It connects to Tailscale in two ways:

1. **LocalAPI** — Reads your local `tailscaled` daemon (via Unix socket or TCP) for the live device list. No credentials required.
2. **Cloud API** *(optional)* — With a Tailscale API key, Tailor fetches your tailnet's ACL policy, resolves effective access into graph edges, and enables editing with validation.

When Tailor joins your tailnet as its own node (the recommended Docker deployment), it serves HTTPS directly and every request is authenticated through Tailscale. Editing follows your tailnet's own roles: Owners, Admins and Network admins (the people Tailscale lets edit the policy file) can edit through Tailor, and everyone else can view the topology. You can also delegate access to other users or tagged devices with a Tailscale **app capability** — a permission you grant in your ACL policy — which Tailor offers to add for you.

The same Go backend serves both the web UI and the MCP endpoint, so agents see the same live topology and policy data you do in the browser.

## Quick start

### Docker (recommended)

Published images are on GitHub Container Registry:

```sh
docker pull ghcr.io/waffleophagus/tailor:latest
```

The recommended way to run Tailor is with `docker compose`. A reference [`compose.yaml`](compose.yaml) is included in the repo (it uses `build: .` for local builds). For the published image, create a `compose.yaml` like this:

```yaml
services:
  tailor:
    image: ghcr.io/waffleophagus/tailor:latest
    ports:
      - "8080:8080"
    environment:
      TAILOR_ADDR: ":8080"
      TAILOR_LOG_DIR: /var/log/tailor
      # Choose one of the two modes below:
```

**Embedded mode** (recommended and the default — the container joins your tailnet as its own machine). No auth key is needed: on first start, open `http://<host>:8080` and click **Log in to Tailscale** to add Tailor with your normal Tailscale login (the link is also printed in the logs).

```yaml
      TAILSCALE_HOSTNAME: "tailor"
      # Optional, for unattended installs: enroll with an auth key instead of logging in.
      # TAILSCALE_AUTHKEY: "tskey-auth-..."   # TS_AUTHKEY also works
      # Optional: advertise tags
      # TS_ADVERTISE_TAGS: "tag:tailor-acl-service"
```

Once the node joins your tailnet, Tailor serves HTTPS directly, so you can open **`https://tailor.<your-tailnet>.ts.net/`** — no `:8080` required. After that, port `8080` only shows a link to that address. The login link on the setup page is only shown to this machine and private networks.

> **Key expiry:** a machine added by logging in belongs to you and inherits your key expiry (180 days by default). Tag it (for example `tag:tailor-acl-service`, which Tailor applies when it adds its access grant) or disable key expiry for it in the admin console. Tailor logs a warning when this applies.

**External mode** (use your host's already-running `tailscaled` instead — Linux only; supported, but embedded mode is preferred):

```yaml
      TAILOR_TAILSCALE_MODE: "external"
      TAILOR_LOCALAPI_SOCKET: "/var/run/tailscale/tailscaled.sock"
    volumes:
      - /var/run/tailscale/tailscaled.sock:/var/run/tailscale/tailscaled.sock:ro
```

Add state and log volumes for either mode:

```yaml
    volumes:
      - ./tailor-state:/var/lib/tailor-tsnet
      - ./tailor-logs:/var/log/tailor
```

Then run:

```sh
docker compose up
```

Open [http://localhost:8080](http://localhost:8080).

Pin a release: `ghcr.io/waffleophagus/tailor:0.1.0` or `ghcr.io/waffleophagus/tailor:v0.1.0`.

### Prebuilt binary

Each [GitHub Release](../../releases) ships production binaries (no dev/demo tags) for Linux, macOS, and Windows, plus a `checksums.txt` file.

```sh
# Example: Linux amd64
curl -LO https://github.com/Waffleophagus/tailor/releases/latest/download/tailor_0.1.0_linux_amd64.tar.gz
tar -xzf tailor_0.1.0_linux_amd64.tar.gz
chmod +x tailor
./tailor
```

The binary is fully self-contained. By default it listens on `:8080` and joins your tailnet as its own machine: open [http://localhost:8080](http://localhost:8080) and click **Log in to Tailscale** on first run. Its node state is kept in your user config directory (for example `~/.config/tailor/tsnet`).

To use this computer's existing Tailscale connection instead of adding a machine, run with `TAILOR_TAILSCALE_MODE=external` (the LocalAPI socket is auto-detected; set `TAILOR_LOCALAPI_SOCKET` or, on Windows, `TAILOR_LOCALAPI_ENDPOINT` if needed). This mode is still supported, but embedded mode is the focus going forward.

### Build from source

Requires [Go](https://go.dev) 1.27+, [Node.js](https://nodejs.org) 24+, and [pnpm](https://pnpm.io).

```sh
# Install frontend dependencies and build the UI
pnpm --dir web install
pnpm --dir web build

# Compile the Go binary
go build -o tailor ./cmd/tailor

# Run it
./tailor
```

## Access Control

When Tailor is deployed as a Tailscale node, every browser and MCP request is authenticated through Tailscale, and Tailor decides what each caller may do from their Tailscale identity. See [ADR 007](docs/adr/007-identity-based-editing-access.md) for the design, and [ADR 008](docs/adr/008-embedded-node-by-default.md) for how Tailor joins the tailnet.

### What this means for you

1. **View-only by default.** Anyone who can reach the Tailor node can open it and see the live topology graph, but they cannot read or edit policy.
2. **A tailnet admin connects the Tailscale API.** Only an Owner, Admin or Network admin, on a personal (untagged) device, can enter the Tailscale API key. Tailor checks two things before storing it: that the key belongs to the tailnet Tailor runs in, and that the person entering it holds one of those roles there. Once a key is set, only editors can replace it.
3. **Tailnet admins can edit.** Owners, Admins and Network admins — the roles Tailscale itself allows to edit the policy file — get **full access** from their personal devices:
   - View topology and status
   - Read and edit ACL policy through the UI and API
   - Use MCP write tools (stage drafts)

   Role changes in Tailscale take effect in Tailor within about 30 seconds.
4. **Optional: delegate with an app capability.** To give editing to other users or to tagged devices (for example MCP runners), grant Tailor's app capability with the `admin` action. Tailor offers to append a recommended grant for you. The capability name is derived from Tailor's MagicDNS address and looks like:
   ```text
   tailor.<your-tailnet>.ts.net/cap/admin
   ```
   You can override this with `TAILOR_APP_CAPABILITY` if needed. Callers with the capability get full access even if the Tailscale API is unreachable.
5. **View-only stays view-only.** Everyone else is **view-only**:
   - View the UI and topology
   - Use read-only MCP tools (inspect topology, read ACL reference docs)
   - Cannot read raw policy HuJSON, edit policy, or stage drafts

### Tags

- **`tag:tailor-acl-service`** — Advertise this tag on the Tailor node so tailnet policy can target the Tailor service as a destination. Tailor can automatically add this tag during setup.
- **`tag:tailor-acl-editor`** *(optional, advanced)* — Apply this tag to trusted workstations or MCP runners. If the app capability grant includes this tag as a source, those devices receive full Tailor access regardless of user role.

### Example policy snippet

```json
{
  "tagOwners": {
    "tag:tailor-acl-service": ["autogroup:admin"]
  },
  "grants": [
    {
      "src": ["autogroup:owner", "autogroup:admin"],
      "dst": ["tag:tailor-acl-service"],
      "ip": ["tcp:443"],
      "app": {
        "tailor.<your-tailnet>.ts.net/cap/admin": [
          { "actions": ["admin"] }
        ]
      }
    },
    {
      "src": ["autogroup:member"],
      "dst": ["tag:tailor-acl-service"],
      "ip": ["tcp:443"]
    }
  ]
}
```

Replace `tailor.<your-tailnet>.ts.net/cap/admin` with your Tailor instance's actual MagicDNS-based capability name.

## AI Agent Integration

Tailor can expose an MCP (Model Context Protocol) server so compatible agents — Claude, Cursor, and others — can inspect and help manage your tailnet.

### What agents can do

- **Inspect topology** — Read your live device list and access edges. Reading current ACL policy requires full Tailor access.
- **Draft policy changes** — Agents use embedded Tailscale ACL reference docs to propose safe, syntax-correct HuJSON edits.
- **Evaluate impact** — Before staging, agents preview what would change on the graph: added, removed, and broadened access.
- **Stage for review** — Agents never save directly. Changes are staged inside Tailor for you to review in the UI and decide whether to apply or discard.

### Setup

1. **Enable the MCP server** by setting `TAILOR_MCP`:
   - `localhost` — for agents running on the same machine
   - `tailnet` — for agents anywhere on your tailnet (uses Tailscale identity and app-capability authorization)

2. **Restart Tailor** to pick up the changes.

3. **Find your MCP URL**:
   - Local: `http://localhost:8080/mcp`
   - Tailnet: `https://tailor.<your-tailnet>.ts.net/mcp` (when Tailor is deployed as a Tailscale node)
   - Custom: `http(s)://<your-host>:8080/mcp`

4. **Connect your agent**. Most MCP-compatible clients (Claude Desktop, Cursor, etc.) accept an HTTP SSE endpoint:
   - URL: your MCP URL from step 3

5. **Verify it's working** by asking the agent to describe your tailnet topology.

### Workflow

1. **Enable** — Set `TAILOR_MCP` to `localhost` or `tailnet` (see Configuration below).
2. **Connect** — Point your agent at `http(s)://tailor.<your-tailnet>.ts.net/mcp` (or `http://localhost:8080/mcp`).
3. **Explore** — The agent reads your topology and policy.
4. **Draft** — The agent edits HuJSON with embedded ACL reference guidance.
5. **Evaluate** — The agent previews the impact on your graph.
6. **Stage** — The agent submits the draft to Tailor.
7. **Review** — Open the Tailor UI, inspect the staged draft, and save or discard it.

**Note:** You will notice that the agent cannot save to Tailscale directly, this is very intentional.

### Security

| Setting | Exposure | Authorization |
|---|---|---|
| `localhost` | Only `127.0.0.1` / `::1` | None required |
| `tailnet` | Any tailnet client | Tailnet identity: tailnet admin role or app capability (`admin` action) |

Set `TAILOR_MCP_READONLY=true` to prevent agents from staging drafts — useful for observability-only setups.

In `tailnet` mode each MCP session is bound to the tailnet user and device that opened it, and permissions are re-checked on every tool call, so revoking someone's access takes effect immediately even for open sessions.

**Reverse proxies:** `localhost` mode only allows connections from the same machine. If Tailor sits behind a reverse proxy or load balancer, the proxy appears as a remote client and requests will be rejected. In that case, use `tailnet` mode so Tailor can verify the caller's Tailscale identity and app capability.

## Configuration

| Variable | Description | Default |
|---|---|---|
| `TAILOR_ADDR` | HTTP listen address | `:8080` |
| `TAILOR_LOCALAPI_SOCKET` | Path to `tailscaled.sock` (Linux) | auto-detected |
| `TAILOR_LOCALAPI_ENDPOINT` | TCP endpoint for LocalAPI (Windows) | — |
| `TAILOR_TAILSCALE_MODE` | `auto`/`embedded`: join the tailnet as Tailor's own machine; `external`: use the host's `tailscaled`. `auto` uses `external` only when `TAILOR_LOCALAPI_SOCKET` or `TAILOR_LOCALAPI_ENDPOINT` is set. | `auto` |
| `TAILSCALE_AUTHKEY` | Optional auth key to enroll the embedded node without the login page (unattended installs) | — |
| `TS_AUTHKEY` | Tailscale auth key alias | — |
| `TAILSCALE_HOSTNAME` | Hostname when joining tailnet | `tailor` |
| `TS_HOSTNAME` | Hostname alias | `tailor` |
| `TS_STATE_DIR` | State directory when Tailor joins your tailnet as a node | `/var/lib/tailor-tsnet` in containers, otherwise `<user config dir>/tailor/tsnet` |
| `TAILSCALE_STATE_DIR` | State directory alias | same as above |
| `TS_ADVERTISE_TAGS` | Comma-separated tags advertised by the Tailor node | — |
| `TAILSCALE_ADVERTISE_TAGS` | Advertise tags alias | — |
| `TAILOR_TSNET_PORT` | HTTPS listen port when Tailor joins your tailnet | `443` |
| `TAILOR_APP_CAPABILITY` | Override the auto-resolved app capability name (e.g. `tailor.<your-tailnet>.ts.net/cap/admin`) | — |
| `TAILOR_TAILSCALE_SERVE` | Legacy non-tsnet Tailscale Serve auto-configuration: `auto`, `on`, or `off` | `auto` |
| `TAILOR_TAILSCALE_SERVE_PORT` | HTTPS port for legacy Tailscale Serve auto-configuration | `443` |
| `TAILOR_MCP` | MCP server exposure: `off`, `localhost`, or `tailnet` | `off` |
| `TAILOR_ALLOWED_HOSTS` | When Tailor is not its own tailnet node: extra hostnames it may be served under, comma-separated (a leading `.` allows a whole domain, e.g. `.example.com`). IP addresses, `localhost`, single-label names, local-only domains (`.lan`, `.local`, `.home.arpa`, …) and `*.ts.net` are always allowed; other names are rejected to block DNS rebinding. | — |
| `TAILOR_MCP_PATH` | MCP endpoint path | `/mcp` |
| `TAILOR_MCP_READONLY` | Disallow staging from MCP (`true`/`false`) | `false` |
| `TAILOR_LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |
| `TAILOR_LOG_FORMAT` | Log format: `text`, `json`, or `auto` (JSON in containers) | `auto` |
| `TAILOR_LOG_DIR` | Optional directory for rotated log files (`tailor.log`); stdout always logged | — |
| `TAILOR_LOG_MAX_SIZE_MB` | Max size in MB before rotating the log file | `10` |
| `TAILOR_LOG_MAX_BACKUPS` | Number of rotated log files to retain | `5` |
| `TAILOR_LOG_MAX_AGE_DAYS` | Delete rotated logs older than this many days | `30` |

## Development

```sh
# Build and run the Go backend in dev mode (synthetic fleet)
pnpm --dir web dev:stack

# In another terminal, run the Vite dev server
pnpm --dir web dev:proxy
```

Dev mode compiles the backend with a built-in synthetic tailnet — a fake fleet of devices and a sample ACL policy — so you can work on the UI without joining a real tailnet. Use the demo API key `tskey-api-tailor-dev` to enable "Cloud API" editing against this synthetic data.

Run tests:
```sh
pnpm --dir web lint && pnpm --dir web check && pnpm --dir web test
pnpm --dir web test:e2e   # demo tailnet (see web/e2e/README.md)
go test ./... && go test -tags dev ./...
```

## PR oddities?
Ok, this is me trying something out. I [no longer](https://mrshu.github.io/github-statuses/) [trust](https://youtu.be/R7ex-Gt8dtw) [Github](https://dev.to/varshithvhegde/github-broke-git-the-merge-queue-bug-that-silently-deleted-your-code-4f7i). So I am using this repository to learn how to manage code both on and off Github. Feel free to open issues, or a PR! I will happily make it work. But that is why all my PRs are closed without being merged, they are being merged in my personal Forgejo instance.

## Consider hiring me?

If you like what you see here, I'm actively looking for my next role. Open to contract work or full time! Learn more at [d6software.com](https://d6software.com).

## License

MIT
