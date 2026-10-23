package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostAllowed(t *testing.T) {
	extra := []string{"tailor.example.com", ".corp.example.net"}
	tests := map[string]bool{
		"localhost:8080":               true,
		"127.0.0.1:8080":               true,
		"[::1]:8080":                   true,
		"192.168.1.20:8080":            true,
		"nas:8080":                     true, // single-label LAN name
		"nas.lan":                      true,
		"printer.local":                true,
		"router.home.arpa":             true,
		"tailor.tail1234.ts.net":       true, // Tailscale Serve keeps the MagicDNS host
		"tailor.example.com":           true, // TAILOR_ALLOWED_HOSTS exact
		"tailor.corp.example.net":      true, // TAILOR_ALLOWED_HOSTS suffix
		"evil.example":                 false,
		"localhost.evil.example":       false,
		"rebind.attacker.example:8080": false,
		"example.com":                  false,
	}
	for host, want := range tests {
		if got := hostAllowed(host, extra); got != want {
			t.Errorf("hostAllowed(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestDNSRebindingHostIsRejectedInLocalMode(t *testing.T) {
	handler := New(Options{})
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://rebind.attacker.example:8080/api/health", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMisdirectedRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusMisdirectedRequest)
	}
}
