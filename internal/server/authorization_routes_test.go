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
