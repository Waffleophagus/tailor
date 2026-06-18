package server

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Waffleophagus/tailor/internal/authz"
	"tailscale.com/client/local"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/tailcfg"
)

type fakeHijacker struct {
	http.ResponseWriter
	hijacked bool
}

type fakeWhoIsClient struct {
	response *apitype.WhoIsResponse
	err      error
}

func (f fakeWhoIsClient) WhoIs(context.Context, string) (*apitype.WhoIsResponse, error) {
	return f.response, f.err
}

func (f *fakeHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	f.hijacked = true
	return nil, nil, nil
}

func TestStatusRecorderForwardsHijack(t *testing.T) {
	underlying := &fakeHijacker{ResponseWriter: httptest.NewRecorder()}
	rec := &statusRecorder{ResponseWriter: underlying, status: http.StatusOK}

	hj, ok := any(rec).(http.Hijacker)
	if !ok {
		t.Fatal("statusRecorder should implement http.Hijacker")
	}
	if _, _, err := hj.Hijack(); err != nil {
		t.Fatalf("Hijack() error = %v", err)
	}
	if !underlying.hijacked {
		t.Fatal("expected underlying Hijack to be called")
	}
}

func TestStatusRecorderUnwrapReturnsUnderlyingWriter(t *testing.T) {
	underlying := httptest.NewRecorder()
	rec := &statusRecorder{ResponseWriter: underlying, status: http.StatusOK}
	unwrapped := any(rec).(interface{ Unwrap() http.ResponseWriter })
	if unwrapped.Unwrap() != underlying {
		t.Fatal("Unwrap should return the underlying writer")
	}
}

func TestStatusRecorderHijackUnsupported(t *testing.T) {
	rec := &statusRecorder{ResponseWriter: httptest.NewRecorder(), status: http.StatusOK}
	if _, ok := any(rec).(http.Hijacker); !ok {
		t.Fatal("statusRecorder should implement http.Hijacker")
	}
	if _, _, err := rec.Hijack(); err == nil {
		t.Fatal("expected error when underlying writer is not hijackable")
	}
}

func TestIdentityMiddlewareAttachesFullRoleFromCapability(t *testing.T) {
	const cap = "tailor.example.ts.net/cap/admin"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := authz.IdentityFromContext(r.Context())
		if !ok {
			t.Fatal("expected identity in context")
		}
		if identity.Role != authz.RoleFull {
			t.Fatalf("role = %q, want full", identity.Role)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	handler := IdentityMiddleware(nil, &AuthOptions{
		TailnetMode:   true,
		WhoIsClient:   fakeWhoIsClient{response: &apitype.WhoIsResponse{CapMap: tailcfg.PeerCapMap{tailcfg.PeerCapability(cap): []tailcfg.RawMessage{`{"actions":["admin"]}`}}}},
		AppCapability: cap,
	}, next)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/cloud/status", nil)
	req.RemoteAddr = "100.64.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestIdentityMiddlewareRejectsUnauthenticatedPageWithTailnetGuidance(t *testing.T) {
	nextCalled := false
	handler := IdentityMiddleware(nil, &AuthOptions{
		TailnetMode: true,
		WhoIsClient: fakeWhoIsClient{err: errors.New("no tailnet identity")},
	}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if nextCalled {
		t.Fatal("unauthenticated request reached application handler")
	}
	if body := rec.Body.String(); !strings.Contains(body, "This Tailor instance is private") ||
		!strings.Contains(body, "No topology or policy data was loaded") ||
		!strings.Contains(body, "MagicDNS address") {
		t.Fatalf("response did not contain tailnet guidance: %q", body)
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("content type = %q, want HTML", contentType)
	}
}

func TestIdentityMiddlewareRejectsUnauthenticatedAPIWithoutHTML(t *testing.T) {
	handler := IdentityMiddleware(nil, &AuthOptions{
		TailnetMode: true,
		WhoIsClient: fakeWhoIsClient{err: errors.New("no tailnet identity")},
	}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unauthenticated request reached application handler")
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/health", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Fatal("API rejection should not return the browser access page")
	}
}

func TestIdentityMiddlewareFailsClosedWithoutWhoIsClient(t *testing.T) {
	handler := IdentityMiddleware(nil, &AuthOptions{TailnetMode: true}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request reached application handler without WhoIs client")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestResolveAppCapabilitySkipsTailnetStatusOutsideTailnetMode(t *testing.T) {
	var typedNil *local.Client
	opts := AuthOptions{TailnetStatus: typedNil}

	if got := opts.resolveAppCapability(context.Background(), nil); got != "" {
		t.Fatalf("capability = %q, want empty outside tailnet mode", got)
	}
}
