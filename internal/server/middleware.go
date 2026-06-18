package server

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
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

// IdentityMiddleware requires and attaches tsnet WhoIs identity to every request
// when Tailor is serving directly on the tailnet. This prevents Funnel or another
// public proxy from exposing even the application shell without tailnet identity.
func IdentityMiddleware(logger *slog.Logger, opts *AuthOptions, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if opts == nil {
		opts = &AuthOptions{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !opts.TailnetMode {
			next.ServeHTTP(w, r)
			return
		}
		if opts.WhoIsClient == nil {
			logger.Error("tailnet identity lookup unavailable", "path", r.URL.Path)
			writeTailnetIdentityRequired(w, r)
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
			writeTailnetIdentityRequired(w, r)
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

func writeTailnetIdentityRequired(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/mcp" {
		http.Error(w, "Tailnet identity is required. Open Tailor from a device connected to its tailnet.", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'")
	w.WriteHeader(http.StatusForbidden)
	_, _ = io.WriteString(w, `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Tailnet access required · Tailor</title>
  <style>
    :root { color-scheme: light; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    * { box-sizing: border-box; }
    body { min-height: 100vh; margin: 0; display: grid; place-items: center; padding: 1.5rem; color: #172126; background: #f3f6f4; }
    main { width: min(100%, 38rem); padding: clamp(1.5rem, 5vw, 3rem); border: 1px solid #d9e1dd; border-radius: 16px; background: #fff; }
    .brand { display: flex; align-items: center; gap: .75rem; margin-bottom: 2.5rem; color: #3a4a44; font-weight: 650; }
    .mark { display: inline-grid; place-items: center; width: 2.5rem; height: 2.5rem; border-radius: 10px; color: #fff; background: #315e4e; font-weight: 750; }
    .status { display: inline-flex; align-items: center; gap: .5rem; margin-bottom: 1rem; color: #315044; font-size: .875rem; font-weight: 650; }
    .status::before { width: .625rem; height: .625rem; border-radius: 50%; background: #b0892f; content: ""; }
    h1 { max-width: 20ch; margin: 0; font-size: clamp(1.75rem, 6vw, 2.25rem); line-height: 1.12; letter-spacing: -0.025em; text-wrap: balance; }
    p { max-width: 60ch; margin: 1rem 0 0; color: #586761; font-size: 1rem; line-height: 1.65; text-wrap: pretty; }
    strong { color: #243d35; font-weight: 650; }
    .privacy { margin-top: 1.5rem; padding: .875rem 1rem; border-radius: 10px; color: #315044; background: #edf8f2; font-size: .925rem; }
    h2 { margin: 2rem 0 .75rem; color: #20332c; font-size: 1rem; }
    ol { margin: 0; padding-left: 1.4rem; color: #586761; line-height: 1.75; }
    a { display: inline-flex; margin-top: 2rem; padding: .7rem 1rem; border-radius: 9px; color: #fff; background: #315e4e; font-weight: 650; text-decoration: none; }
    a:hover { background: #25483c; }
    a:focus-visible { outline: 3px solid #9bc4b5; outline-offset: 3px; }
  </style>
</head>
<body>
  <main>
    <div class="brand"><span class="mark" aria-hidden="true">T</span> Tailor</div>
    <div class="status">Identity not available</div>
    <h1>This Tailor instance is private</h1>
    <p>Tailor could not verify a tailnet identity for this connection. Access is limited to authenticated devices on the tailnet that hosts this instance.</p>
    <p class="privacy"><strong>No topology or policy data was loaded.</strong> Tailor blocks requests before the application starts.</p>
    <h2>To continue</h2>
    <ol>
      <li>Connect this device to Tailscale.</li>
      <li>Confirm you are using the correct tailnet.</li>
      <li>Open Tailor using its MagicDNS address.</li>
    </ol>
    <a href="/">Try again</a>
  </main>
</body>
</html>`)
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
