package server

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Waffleophagus/tailor/internal/authz"
	"tailscale.com/client/tailscale/apitype"
)

type contextKey int

const requestIDKey contextKey = iota

// RequestIDFromContext returns the request ID attached by AccessMiddleware, if any.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

type AuthOptions struct {
	TailnetMode   bool
	WhoIsClient   WhoIsClient
	TailnetStatus TailnetStatusClient
	AppCapability string
	MCPPath       string

	mu                 sync.Mutex
	resolvedCapability string
}

func newRequestID() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf[:])
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Hijack forwards to the underlying ResponseWriter so WebSocket handlers work.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("http.ResponseWriter does not implement http.Hijacker")
	}
	return h.Hijack()
}

// Flush forwards to the underlying ResponseWriter when supported.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Unwrap returns the underlying writer for middleware feature detection.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// AccessMiddleware logs API requests with method, path, status, duration, and request ID.
func AccessMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		id := newRequestID()
		ctx := withRequestID(r.Context(), id)
		r = r.WithContext(ctx)

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		status := rec.status
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", id,
		}

		switch {
		case r.URL.Path == "/api/health":
			logger.Debug("api request", attrs...)
		case status >= 500:
			logger.Error("api request", attrs...)
		case status >= 400:
			logger.Warn("api request", attrs...)
		default:
			logger.Info("api request", attrs...)
		}
	})
}

// IdentityMiddleware attaches tsnet WhoIs identity to API and MCP requests when
// Tailor is serving directly on the tailnet.
func IdentityMiddleware(logger *slog.Logger, opts *AuthOptions, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if opts == nil {
		opts = &AuthOptions{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !opts.TailnetMode || opts.WhoIsClient == nil || !needsIdentity(r, opts.MCPPath) {
			next.ServeHTTP(w, r)
			return
		}

		who, err := opts.WhoIsClient.WhoIs(r.Context(), r.RemoteAddr)
		if err != nil {
			logger.Warn("tailnet identity lookup failed",
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"error", err.Error(),
				"request_id", RequestIDFromContext(r.Context()),
			)
			http.Error(w, "Tailnet identity is required.", http.StatusForbidden)
			return
		}

		appCapability := opts.resolveAppCapability(r.Context(), logger)
		identity := identityFromWhoIs(who, appCapability)
		logger.Debug("tailnet identity attached",
			"path", r.URL.Path,
			"login", identity.LoginName,
			"node", identity.NodeName,
			"role", string(identity.Role),
			"request_id", RequestIDFromContext(r.Context()),
		)
		next.ServeHTTP(w, r.WithContext(authz.WithIdentity(r.Context(), identity)))
	})
}

func (opts *AuthOptions) resolveAppCapability(ctx context.Context, logger *slog.Logger) string {
	if strings.TrimSpace(opts.AppCapability) != "" {
		return strings.TrimSpace(opts.AppCapability)
	}
	if !opts.TailnetMode {
		return ""
	}

	opts.mu.Lock()
	cached := opts.resolvedCapability
	opts.mu.Unlock()
	if cached != "" {
		return cached
	}
	if opts.TailnetStatus == nil {
		return ""
	}

	status, err := opts.TailnetStatus.StatusWithoutPeers(ctx)
	if err != nil {
		logger.Debug("tailor app capability unavailable", "error", err.Error())
		return ""
	}
	if status == nil || status.Self == nil {
		return ""
	}
	name := strings.TrimSuffix(strings.TrimSpace(status.Self.DNSName), ".")
	if name == "" {
		return ""
	}
	capability := name + "/cap/admin"
	opts.mu.Lock()
	opts.resolvedCapability = capability
	opts.mu.Unlock()
	return capability
}

func needsIdentity(r *http.Request, mcpPath string) bool {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		return true
	}
	return mcpPath != "" && r.URL.Path == mcpPath
}

func identityFromWhoIs(who *apitype.WhoIsResponse, appCapability string) authz.TailnetIdentity {
	identity := authz.TailnetIdentity{Role: authz.RoleViewer}
	if who == nil {
		return identity
	}
	if who.UserProfile != nil {
		identity.LoginName = who.UserProfile.LoginName
	}
	if who.Node != nil {
		identity.NodeName = strings.TrimSuffix(who.Node.Name, ".")
		identity.NodeTags = append([]string(nil), who.Node.Tags...)
	}
	if who.CapMap != nil {
		identity.CapMap = who.CapMap
		identity.Role = authz.RoleForCapability(who.CapMap, appCapability)
	}
	return identity
}

func logAPIError(logger *slog.Logger, r *http.Request, status int, err error, msg string) {
	if logger == nil {
		return
	}
	attrs := []any{
		"status", status,
		"path", r.URL.Path,
		"method", r.Method,
		"request_id", RequestIDFromContext(r.Context()),
	}
	if msg != "" {
		attrs = append(attrs, "message", msg)
	}
	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}

	switch {
	case status >= 500:
		logger.Error("api handler error", attrs...)
	case status == http.StatusUnauthorized:
		logger.Debug("api handler error", attrs...)
	default:
		logger.Warn("api handler error", attrs...)
	}
}
