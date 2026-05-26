FROM node:24-alpine AS web-build

WORKDIR /src

RUN corepack enable

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

FROM scratch

COPY --from=go-build /out/tailor /usr/local/bin/tailor
COPY --from=go-build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER 10001:10001
EXPOSE 8080
ENV TAILOR_ADDR=:8080

ENTRYPOINT ["/usr/local/bin/tailor"]
