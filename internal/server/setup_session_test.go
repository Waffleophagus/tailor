package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Waffleophagus/tailor/internal/authz"
)

func TestSetupSessionIsIdentityBoundAndSingleUse(t *testing.T) {
	sessions := NewSetupSessions()
	token, _ := sessions.Create("admin@example.com", "workstation")

	if sessions.Consume(token, "other@example.com", "workstation") {
		t.Fatal("setup session should reject a different identity")
	}
	if sessions.Consume(token, "admin@example.com", "workstation") {
		t.Fatal("setup session should be invalidated after a mismatched identity")
	}

	token, _ = sessions.Create("admin@example.com", "workstation")
	if !sessions.Consume(token, "admin@example.com", "workstation") {
		t.Fatal("setup session should accept its bound identity")
	}
	if sessions.Consume(token, "admin@example.com", "workstation") {
		t.Fatal("setup session should be single use")
	}
}

func TestSetupAndBootstrapSessionsExpire(t *testing.T) {
	setup := NewSetupSessions()
	setupToken, _ := setup.Create("admin@example.com", "workstation")
	setup.mu.Lock()
	session := setup.sessions[setupToken]
	session.expiresAt = time.Now().Add(-time.Second)
	setup.sessions[setupToken] = session
	setup.mu.Unlock()
	if setup.Consume(setupToken, "admin@example.com", "workstation") {
		t.Fatal("expired setup session should be rejected")
	}

	bootstrap := NewBootstrapSessions()
	bootstrapToken, _ := bootstrap.Create("admin@example.com", "workstation")
	bootstrap.mu.Lock()
	bootstrapSession := bootstrap.sessions[bootstrapToken]
	bootstrapSession.expiresAt = time.Now().Add(-time.Second)
	bootstrap.sessions[bootstrapToken] = bootstrapSession
	bootstrap.mu.Unlock()
	if valid, _ := bootstrap.Valid(bootstrapToken, "admin@example.com", "workstation"); valid {
		t.Fatal("expired bootstrap session should be rejected")
	}
}

func TestSetupGrantSaveRequiresFreshIdentityBoundSession(t *testing.T) {
	s := &Server{setup: NewSetupSessions(), auth: AuthOptions{TailnetMode: true}}
	identity := authz.TailnetIdentity{LoginName: "admin@example.com", NodeName: "workstation"}

	missing := httptest.NewRequest("POST", "/api/cloud/setup-grant", nil)
	missing = missing.WithContext(authz.WithIdentity(missing.Context(), identity))
	if s.consumeSetupSession(missing) {
		t.Fatal("missing fresh setup session should be rejected")
	}

	token, expiresAt := s.setup.Create(identity.LoginName, identity.NodeName)
	valid := httptest.NewRequest("POST", "/api/cloud/setup-grant", nil)
	valid.AddCookie(&http.Cookie{Name: setupCookieName, Value: token, Expires: expiresAt})
	valid = valid.WithContext(authz.WithIdentity(valid.Context(), identity))
	if !s.consumeSetupSession(valid) {
		t.Fatal("fresh identity-bound setup session should be accepted")
	}
}

func TestHTTPPolicyPermissionMatrix(t *testing.T) {
	s := &Server{}
	tests := []struct {
		name       string
		ctx        func(*http.Request) *http.Request
		permission authz.Permission
		want       int
	}{
		{"viewer read denied", withRole(authz.RoleViewer), authz.PermissionReadPolicy, http.StatusForbidden},
		{"viewer write denied", withRole(authz.RoleViewer), authz.PermissionWritePolicy, http.StatusForbidden},
		{"full read allowed", withRole(authz.RoleFull), authz.PermissionReadPolicy, http.StatusOK},
		{"full write allowed", withRole(authz.RoleFull), authz.PermissionWritePolicy, http.StatusOK},
		{"bootstrap HTTP write allowed", withBootstrapRole(authz.RoleViewer), authz.PermissionWritePolicy, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := tt.ctx(httptest.NewRequest("GET", "/api/policy", nil))
			allowed := s.requirePermission(recorder, request, tt.permission, "denied")
			got := recorder.Code
			if allowed {
				got = http.StatusOK
			}
			if got != tt.want {
				t.Fatalf("status = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBootstrapMiddlewareRejectsExpiredOrDifferentIdentity(t *testing.T) {
	bootstrap := NewBootstrapSessions()
	token, _ := bootstrap.Create("admin@example.com", "workstation")
	bootstrap.mu.Lock()
	session := bootstrap.sessions[token]
	session.expiresAt = time.Now().Add(-time.Second)
	bootstrap.sessions[token] = session
	bootstrap.mu.Unlock()
	validToken, _ := bootstrap.Create("admin@example.com", "workstation")

	server := &Server{bootstrap: bootstrap}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authz.Allowed(r.Context(), authz.PermissionWritePolicy) {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusForbidden)
	})
	tests := []struct {
		name     string
		token    string
		identity authz.TailnetIdentity
	}{
		{"expired", token, authz.TailnetIdentity{Role: authz.RoleViewer, LoginName: "admin@example.com", NodeName: "workstation"}},
		{"different identity", validToken, authz.TailnetIdentity{Role: authz.RoleViewer, LoginName: "other@example.com", NodeName: "workstation"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/policy/stage", nil)
			request.AddCookie(&http.Cookie{Name: bootstrapCookieName, Value: tt.token})
			request = request.WithContext(authz.WithIdentity(request.Context(), tt.identity))
			response := httptest.NewRecorder()
			BootstrapMiddleware(server, next).ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusForbidden)
			}
		})
	}
}

func withRole(role authz.Role) func(*http.Request) *http.Request {
	return func(r *http.Request) *http.Request {
		return r.WithContext(authz.WithIdentity(r.Context(), authz.TailnetIdentity{Role: role}))
	}
}

func withBootstrapRole(role authz.Role) func(*http.Request) *http.Request {
	return func(r *http.Request) *http.Request {
		ctx := authz.WithIdentity(r.Context(), authz.TailnetIdentity{Role: role})
		return r.WithContext(authz.WithBootstrap(ctx))
	}
}
