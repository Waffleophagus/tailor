package mcpserver

import (
	"crypto/subtle"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Waffleophagus/tailor/internal/tailorcore"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Exposure string

const (
	ExposureOff       Exposure = "off"
	ExposureLocalhost Exposure = "localhost"
	ExposureTailnet   Exposure = "tailnet"
	ExposurePublic    Exposure = "public"
)

type Config struct {
	Exposure Exposure
	Path     string
	Token    string
	ReadOnly bool
}

func ConfigFromEnv() Config {
	return Config{
		Exposure: parseExposure(os.Getenv("TAILOR_MCP")),
		Path:     envOr("TAILOR_MCP_PATH", "/mcp"),
		Token:    strings.TrimSpace(os.Getenv("TAILOR_MCP_TOKEN")),
		ReadOnly: parseBool(os.Getenv("TAILOR_MCP_READONLY")),
	}
}

func (c Config) Enabled() bool {
	if c.Exposure == ExposureOff {
		return false
	}
	if c.Exposure == ExposurePublic && c.Token == "" {
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
		if cfg.Exposure != ExposureLocalhost {
			if cfg.Token == "" {
				logger.Warn("mcp request rejected: missing token configuration")
				http.Error(w, "MCP bearer token is required.", http.StatusForbidden)
				return
			}
			if !validBearerToken(r.Header.Get("Authorization"), cfg.Token) {
				logger.Warn("mcp request rejected: invalid bearer token")
				http.Error(w, "MCP bearer token is required.", http.StatusUnauthorized)
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
	case "public":
		return ExposurePublic
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
	if ip := requestHeaderIP(r.Header.Get("X-Forwarded-For")); ip != nil {
		return ip.IsLoopback()
	}
	if ip := requestHeaderIP(r.Header.Get("X-Real-IP")); ip != nil {
		return ip.IsLoopback()
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validBearerToken(header, token string) bool {
	expected := "Bearer " + token
	return len(header) == len(expected) && subtle.ConstantTimeCompare([]byte(header), []byte(expected)) == 1
}

func requestHeaderIP(value string) net.IP {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if before, _, found := strings.Cut(value, ","); found {
		value = strings.TrimSpace(before)
	}
	return net.ParseIP(value)
}
