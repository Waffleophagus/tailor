<p align="center">
  <img src="tailor-logo.png" alt="Tailor logo" width="180">
</p>

<h1 align="center">Tailor</h1>
<p align="center"><em>Visualize and edit your Tailscale tailnet's access controls.</em></p>


## What is Tailor?

Tailor is a self-hosted visual tool for [Tailscale](https://tailscale.com) administrators. It maps out your tailnet as an interactive, explorable graph and lets you edit your ACL policy with validation and a live preview.

- **Visualize your tailnet** — See every device as a live, force-directed graph that updates in real time as devices come and go.
- **Filter and colorize** — Filter by tag, owner, OS, online status, or subnet-router role. Color nodes by status, tag, owner, or OS.
- **Inspect devices and access** — Click any node to see its owner, tags, IPs, OS, and which other devices it can reach (when authenticated with the Cloud API).
- **Edit ACL policies** — Authenticate with a Tailscale API key to fetch your tailnet's HuJSON policy. Edit directly, validate against Tailscale's Cloud API, preview the result on the graph, and save when it looks right.

## How it works

Tailor is a single Go binary with an embedded Svelte frontend. It connects to Tailscale in two ways:

1. **LocalAPI** — Reads your local `tailscaled` daemon (via Unix socket or TCP) for the live device list. No credentials required.
2. **Cloud API** *(optional)* — With a Tailscale API key, Tailor fetches your tailnet's ACL policy, resolves effective access into graph edges, and enables editing with validation.

## Quick start

### Docker Compose (recommended)

```sh
docker compose up
```

Open [http://localhost:8080](http://localhost:8080).

**Embedded mode** (container runs its own `tailscaled`):
```yaml
environment:
  TAILSCALE_AUTHKEY: "tskey-auth-..."
  TAILSCALE_HOSTNAME: "tailor"
```

**External mode** (use your host's already-running `tailscaled`):
```yaml
environment:
  TAILOR_TAILSCALE_MODE: "external"
  TAILOR_LOCALAPI_SOCKET: "/var/run/tailscale/tailscaled.sock"
volumes:
  - /var/run/tailscale/tailscaled.sock:/var/run/tailscale/tailscaled.sock:ro
```

### Prebuilt binary

Download the latest release for your platform from the [Releases](../../releases) page, extract, and run:

```sh
./tailor
```

The binary is fully self-contained. By default it listens on `:8080` and reads your local `tailscaled` socket.

### Build from source

Requires [Go](https://go.dev) 1.26+, [Node.js](https://nodejs.org) 24+, and [pnpm](https://pnpm.io).

```sh
# Install frontend dependencies and build the UI
pnpm --dir web install
pnpm --dir web build

# Compile the Go binary
go build -o tailor ./cmd/tailor

# Run it
./tailor
```

## Configuration

| Variable | Description | Default |
|---|---|---|
| `TAILOR_ADDR` | HTTP listen address | `:8080` |
| `TAILOR_LOCALAPI_SOCKET` | Path to `tailscaled.sock` (Linux) | auto-detected |
| `TAILOR_LOCALAPI_ENDPOINT` | TCP endpoint for LocalAPI (Windows) | — |
| `TAILOR_TAILSCALE_MODE` | `auto`, `embedded`, or `external` | `auto` |
| `TAILSCALE_AUTHKEY` | Tailscale auth key for embedded mode | — |
| `TAILSCALE_HOSTNAME` | Hostname when joining tailnet | `tailor` |

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
pnpm --dir web check && pnpm --dir web test  # frontend
go test ./...                                 # backend
```

## License

MIT
