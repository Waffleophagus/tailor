<p align="center">
  <img src="tailor-logo.png" alt="Tailor logo" width="180">
</p>

<h1 align="center">Tailor</h1>
<p align="center"><em>The easiest way to tailor ACLs for your tailnet.</em></p>

> ⚠️ **WARNING: Not Production Ready**
> Tailor is under active development. Features, UI, and APIs may change. It's **not quite ready**. I don't really foresee massive breaking changes but still. Contributions and feedback are very welcome!



## What is Tailor?

Tailor is a visual tool that helps you understand and manage your [Tailscale](https://tailscale.com) tailnet's Access Control Lists (ACLs). It turns your raw policy file into an interactive, explorable map of your network.

Instead of editing HuJSON by hand, you can:

- **Visualize your tailnet** — See all your devices, who owns them, and how they are grouped in an interactive graph.
- **Understand access instantly** — Click any device to see which other devices it can reach, and inspect the exact ACL rules that allow it.
- **Edit policies safely** — Modify ACL rules, groups, and tags through a guided UI. Changes are staged as a diff so you can review before committing them to your tailnet.

## How it works

Tailor runs as a single binary with an embedded web frontend. It talks to Tailscale in two ways:

1. **Local discovery** — Tailor reads your local `tailscaled` daemon to list every device in your tailnet, its owner, tags, and online status.
2. **Policy editing** *(optional)* — Provide a Tailscale API key to fetch your tailnet's ACL policy. Tailor overlays effective access paths onto the graph and lets you edit the policy with a live preview and validation.

## Quick Start

### With Docker Compose (recommended)

The container image is hosted on `ghcr.io`. You can run Tailor in two modes: **embedded** (the container runs its own `tailscaled`) or **external** (it piggybacks on your host's already-running `tailscaled`).

Copy the example `compose.yaml` and start it:

```sh
docker compose up
```

Then open [http://localhost:8080](http://localhost:8080).

#### Embedded mode (default)

The container starts its own `tailscaled`. To make it useful for discovering your tailnet, provide a [Tailscale auth key](https://tailscale.com/kb/1085/auth-keys) so the container joins your tailnet as its own node:

```yaml
environment:
  TAILSCALE_AUTHKEY: "tskey-auth-..."
  TAILSCALE_HOSTNAME: "tailor"
```

If you don't provide an auth key, Tailor still starts in embedded mode but only has access to the local daemon — you'll want to switch to external mode (below) or add an auth key for full discovery.

#### External mode (host socket)

On Linux hosts that already run `tailscaled`, you can mount the host's Unix socket instead of running a second daemon inside the container:

```yaml
environment:
  TAILOR_TAILSCALE_MODE: "external"
  TAILOR_LOCALAPI_SOCKET: "/var/run/tailscale/tailscaled.sock"
volumes:
  - /var/run/tailscale/tailscaled.sock:/var/run/tailscale/tailscaled.sock:ro
```

### Prebuilt binary

The fastest way to run Tailor without Docker or build tools.

1. Download the latest release for your platform from the [Releases](../../releases) page.
2. Extract the binary and run it:

```sh
./tailor
```

The binary is fully self-contained — the web frontend is embedded, so no Node.js or Go toolchain is required.

By default, Tailor reads your local `tailscaled` socket and serves the app on `http://localhost:8080`. You can override the address or socket path with environment variables:

```sh
TAILOR_ADDR=127.0.0.1:9090 ./tailor
TAILOR_LOCALAPI_SOCKET=/var/run/tailscale/tailscaled.sock ./tailor
```

### Build from source

Requires [Go](https://go.dev) 1.26+, [Node.js](https://nodejs.org) 24+, and [pnpm](https://pnpm.io).

```sh
# Install frontend dependencies and build the UI
pnpm --dir web install
pnpm --dir web build

# Compile the Go binary
go build -o tailor ./cmd/tailor

# Run it. By default it reads your local tailscaled socket.
./tailor
```

The server listens on `:8080` by default. Override with the `TAILOR_ADDR` environment variable:

```sh
TAILOR_ADDR=127.0.0.1:9090 ./tailor
```

If your local `tailscaled.sock` lives somewhere non-standard, point to it with `TAILOR_LOCALAPI_SOCKET`:

```sh
TAILOR_LOCALAPI_SOCKET=/var/run/tailscale/tailscaled.sock ./tailor
```

## License

MIT
