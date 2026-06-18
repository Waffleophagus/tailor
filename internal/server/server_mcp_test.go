package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPEndpointDisabledByDefault(t *testing.T) {
	t.Setenv("TAILOR_MCP", "")

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
