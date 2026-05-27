FROM node:24-alpine AS web-build

WORKDIR /src

RUN corepack enable && corepack prepare pnpm@11.2.2 --activate

COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./web/
WORKDIR /src/web
RUN pnpm install --frozen-lockfile

WORKDIR /src
COPY web ./web
COPY internal/frontend ./internal/frontend
RUN pnpm --dir web build

FROM golang:1.26-alpine AS go-build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY tools ./tools
COPY --from=web-build /src/internal/frontend/dist ./internal/frontend/dist

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/tailor ./cmd/tailor

FROM tailscale/tailscale:stable AS tailscale

FROM alpine:3.22

RUN apk add --no-cache ca-certificates

COPY --from=tailscale /usr/local/bin/tailscale /usr/local/bin/tailscale
COPY --from=tailscale /usr/local/bin/tailscaled /usr/local/bin/tailscaled
COPY --from=go-build /out/tailor /usr/local/bin/tailor
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

EXPOSE 8080
ENV TAILOR_ADDR=:8080
ENV TAILOR_TAILSCALE_MODE=auto

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
