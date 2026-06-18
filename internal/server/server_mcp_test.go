package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPEndpointDisabledByDefault(t *testing.T) {
	t.Setenv("TAILOR_MCP", "")
	t.Setenv("TAILOR_MCP_TOKEN", "")

	mux := httptest.NewServer(New())
	defer mux.Close()

	resp, err := http.Post(mux.URL+"/mcp", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		t.Fatalf("disabled MCP endpoint returned auth status %d; route appears enabled", resp.StatusCode)
	}
}

func TestMCPRemoteEndpointRequiresBearerToken(t *testing.T) {
	t.Setenv("TAILOR_MCP", "public")
	t.Setenv("TAILOR_MCP_TOKEN", "secret")

	mux := httptest.NewServer(New())
	defer mux.Close()

	resp, err := http.Post(mux.URL+"/mcp", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated MCP status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestMCPRemoteEndpointAcceptsBearerToken(t *testing.T) {
	t.Setenv("TAILOR_MCP", "public")
	t.Setenv("TAILOR_MCP_TOKEN", "secret")

	mux := httptest.NewServer(New())
	defer mux.Close()

	req, err := http.NewRequest(http.MethodPost, mux.URL+"/mcp", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		t.Fatalf("authenticated MCP request returned auth status %d", resp.StatusCode)
	}
}
