package mcpserver

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Waffleophagus/tailor/internal/authz"
	"github.com/Waffleophagus/tailor/internal/tailorcore"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Exposure string

const (
	ExposureOff       Exposure = "off"
	ExposureLocalhost Exposure = "localhost"
	ExposureTailnet   Exposure = "tailnet"
)

type Config struct {
	Exposure Exposure
	Path     string
	ReadOnly bool
}

func ConfigFromEnv() Config {
	return Config{
		Exposure: parseExposure(os.Getenv("TAILOR_MCP")),
		Path:     envOr("TAILOR_MCP_PATH", "/mcp"),
		ReadOnly: parseBool(os.Getenv("TAILOR_MCP_READONLY")),
	}
}

func (c Config) Enabled() bool {
	if c.Exposure == ExposureOff {
		return false
	}
	return true
}

func Handler(core *tailorcore.Service, cfg Config, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "tailor",
		Title:   "Tailor",
		Version: "dev",
	}, &mcp.ServerOptions{
		Instructions: "Inspect Tailor tailnet topology and stage ACL policy drafts for human review. Before modifying policy HuJSON, read the relevant ACL reference topic, inspect the policy map when available, evaluate the draft, then stage it for human review. Never save or upload ACL policy to Tailscale.",
		Logger:       logger,
	})
	registerTools(server, core, cfg)

	handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, &mcp.StreamableHTTPOptions{
		JSONResponse:   false,
		Logger:         logger,
		SessionTimeout: 10 * time.Minute,
	})
	return authMiddleware(cfg, logger, handler)
}

func authMiddleware(cfg Config, logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cfg.Exposure == ExposureLocalhost && !isLoopbackRequest(r) {
			logger.Warn("mcp request rejected: non-localhost client")
			http.Error(w, "MCP is only available to localhost clients.", http.StatusForbidden)
			return
		}
		if cfg.Exposure == ExposureTailnet {
			if _, ok := authz.IdentityFromContext(r.Context()); !ok {
				logger.Warn("mcp request rejected: missing tailnet identity")
				http.Error(w, "MCP tailnet exposure requires tsnet identity.", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func parseExposure(raw string) Exposure {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "localhost":
		return ExposureLocalhost
	case "tailnet":
		return ExposureTailnet
	default:
		return ExposureOff
	}
}

func parseBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func envOr(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	if !strings.HasPrefix(value, "/") {
		return "/" + value
	}
	return value
}

func isLoopbackRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
