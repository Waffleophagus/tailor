# Tailor

Tailnet topology visualizer and ACL editor.

## Local Build

Build the embedded frontend first, then compile the Go binary:

```sh
pnpm --dir web install
pnpm --dir web build
go build -o tailor ./cmd/tailor
```

Run it:

```sh
./tailor
```

The binary listens on `:8080` by default. Override the address or LocalAPI socket path when needed:

```sh
TAILOR_ADDR=127.0.0.1:9090 TAILOR_LOCALAPI_SOCKET=/var/run/tailscale/tailscaled.sock ./tailor
```

## Docker Compose

```sh
docker compose up --build
```

The compose service mounts `/var/run/tailscale/tailscaled.sock` from the host and serves the app on `http://localhost:8080`.
