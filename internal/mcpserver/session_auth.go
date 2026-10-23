package mcpserver

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Waffleophagus/tailor/internal/authz"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// callerRoleHeader carries the caller's role, as resolved for the current HTTP
// request, to tool handlers. The SDK runs tool calls with the context of the
// session's initialize request, so without this a revoked admin would keep their
// role for the life of the session. Client-supplied values are always discarded.
const callerRoleHeader = "X-Tailor-Caller-Role"

const sessionIDHeader = "Mcp-Session-Id"

// allowed authorizes a tool call against the role of the request that carried it,
// falling back to the session context for non-HTTP transports.
func allowed(ctx context.Context, req *mcp.CallToolRequest, permission authz.Permission) bool {
	if req != nil && req.Extra != nil && req.Extra.Header != nil {
		if role := req.Extra.Header.Get(callerRoleHeader); role != "" {
			identity, _ := authz.IdentityFromContext(ctx)
			identity.Role = authz.Role(role)
			ctx = authz.WithIdentity(ctx, identity)
		}
	}
	return authz.Allowed(ctx, permission)
}

// sessionOwners binds MCP session IDs to the tailnet identity that created them,
// so a leaked Mcp-Session-Id cannot be used from another user or device.
type sessionOwners struct {
	mu        sync.Mutex
	owners    map[string]sessionOwner
	ttl       time.Duration
	lastPurge time.Time
}

type sessionOwner struct {
	user     string
	lastSeen time.Time
}

func newSessionOwners(ttl time.Duration) *sessionOwners {
	return &sessionOwners{owners: map[string]sessionOwner{}, ttl: ttl}
}

// check reports whether user may use sessionID, refreshing its lifetime.
func (s *sessionOwners) check(sessionID, user string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.purgeLocked(now)
	owner, ok := s.owners[sessionID]
	if !ok {
		// Unknown sessions are rejected by the SDK itself ("session not found").
		return true
	}
	if owner.user != user {
		return false
	}
	owner.lastSeen = now
	s.owners[sessionID] = owner
	return true
}

func (s *sessionOwners) claim(sessionID, user string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.owners[sessionID]; !exists {
		s.owners[sessionID] = sessionOwner{user: user, lastSeen: time.Now()}
	}
}

func (s *sessionOwners) release(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.owners, sessionID)
}

func (s *sessionOwners) purgeLocked(now time.Time) {
	if now.Sub(s.lastPurge) < time.Minute {
		return
	}
	for id, owner := range s.owners {
		// Sessions idle past the SDK's timeout are gone; keep a margin.
		if now.Sub(owner.lastSeen) > s.ttl+time.Minute {
			delete(s.owners, id)
		}
	}
	s.lastPurge = now
}

func callerKey(ctx context.Context) string {
	identity, _ := authz.IdentityFromContext(ctx)
	return identity.LoginName + "|" + identity.NodeName
}

// bindSessions stamps the caller's current role onto each request and enforces
// that a session is only used by the identity that initialized it.
func bindSessions(owners *sessionOwners, logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del(callerRoleHeader)
		if identity, ok := authz.IdentityFromContext(r.Context()); ok {
			r.Header.Set(callerRoleHeader, string(identity.Role))
		}

		user := callerKey(r.Context())
		sessionID := r.Header.Get(sessionIDHeader)
		if sessionID != "" && !owners.check(sessionID, user) {
			logger.Warn("mcp request rejected: session belongs to another identity")
			http.Error(w, "This MCP session belongs to another user or device.", http.StatusForbidden)
			return
		}

		next.ServeHTTP(&sessionClaimWriter{ResponseWriter: w, owners: owners, user: user}, r)

		if r.Method == http.MethodDelete && sessionID != "" {
			owners.release(sessionID)
		}
	})
}

// sessionClaimWriter records the owner of a newly issued session ID as soon as the
// SDK writes the response headers (before any streamed body).
type sessionClaimWriter struct {
	http.ResponseWriter
	owners  *sessionOwners
	user    string
	claimed bool
}

func (w *sessionClaimWriter) claim() {
	if w.claimed {
		return
	}
	w.claimed = true
	if id := w.Header().Get(sessionIDHeader); id != "" {
		w.owners.claim(id, w.user)
	}
}

func (w *sessionClaimWriter) WriteHeader(status int) {
	w.claim()
	w.ResponseWriter.WriteHeader(status)
}

func (w *sessionClaimWriter) Write(b []byte) (int, error) {
	w.claim()
	return w.ResponseWriter.Write(b)
}

func (w *sessionClaimWriter) Flush() {
	w.claim()
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *sessionClaimWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
