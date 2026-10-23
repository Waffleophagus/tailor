package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"tailscale.com/client/tailscale/apitype"
)

func TestViewerCannotAccessAnyPolicyRoute(t *testing.T) {
	handler := New(Options{
		TailnetMode:   true,
		AppCapability: "tailor.example.ts.net/cap/admin",
		WhoIsClient:   fakeWhoIsClient{response: &apitype.WhoIsResponse{}},
	})

	tests := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/policy"},
		{http.MethodGet, "/api/policy/map"},
		{http.MethodPost, "/api/policy/draft"},
		{http.MethodPost, "/api/policy/mutate"},
		{http.MethodPost, "/api/policy/evaluate-draft"},
		{http.MethodPost, "/api/policy/validate"},
		{http.MethodGet, "/api/policy/staged"},
		{http.MethodPost, "/api/policy/stage"},
		{http.MethodGet, "/api/policy/staged/draft-1"},
		{http.MethodDelete, "/api/policy/staged/draft-1"},
		{http.MethodPost, "/api/policy/save"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			request := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, bytes.NewBufferString(`{}`))
			request.RemoteAddr = "100.64.0.1:12345"
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusForbidden, response.Body.String())
			}
		})
	}
}

func TestCrossSiteBrowserRequestsCannotChangeState(t *testing.T) {
	handler := New(Options{})

	tests := []struct {
		name        string
		header      http.Header
		wantBlocked bool
	}{
		{name: "cross-site fetch", header: http.Header{"Sec-Fetch-Site": {"cross-site"}}, wantBlocked: true},
		{name: "foreign origin without fetch metadata", header: http.Header{"Origin": {"https://evil.example"}}, wantBlocked: true},
		{name: "same-origin fetch", header: http.Header{"Sec-Fetch-Site": {"same-origin"}}},
		{name: "non-browser client", header: http.Header{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8080/api/policy/stage", bytes.NewBufferString(`{}`))
			for key, values := range tt.header {
				request.Header[key] = values
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			blocked := response.Code == http.StatusForbidden && bytes.Contains(response.Body.Bytes(), []byte("Cross-origin"))
			if blocked != tt.wantBlocked {
				t.Fatalf("blocked = %v, want %v (status %d, body %s)", blocked, tt.wantBlocked, response.Code, response.Body.String())
			}
		})
	}
}
