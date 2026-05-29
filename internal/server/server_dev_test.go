//go:build dev

package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/devtailnet"
)

func TestDevSpawnDevicesRequiresDemoAuth(t *testing.T) {
	mux := httptest.NewServer(New())
	t.Cleanup(mux.Close)

	spawn := func() *http.Response {
		body, _ := json.Marshal(api.DevSpawnDevicesRequest{Count: 2, Prefix: "e2e"})
		resp, err := http.Post(mux.URL+"/api/dev/spawn-devices", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	unauth := spawn()
	defer unauth.Body.Close()
	if unauth.StatusCode != http.StatusForbidden {
		t.Fatalf("unauthenticated spawn status = %d, want 403", unauth.StatusCode)
	}

	authBody, _ := json.Marshal(api.CloudAuthRequest{Tailnet: "-", APIKey: devtailnet.APIKey})
	authResp, err := http.Post(mux.URL+"/api/cloud/auth", "application/json", bytes.NewReader(authBody))
	if err != nil {
		t.Fatal(err)
	}
	defer authResp.Body.Close()
	if authResp.StatusCode != http.StatusOK {
		t.Fatalf("auth status = %d", authResp.StatusCode)
	}

	ok := spawn()
	defer ok.Body.Close()
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("spawn status = %d, want 200", ok.StatusCode)
	}

	var payload api.DevSpawnDevicesResponse
	if err := json.NewDecoder(ok.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Spawned) != 2 {
		t.Fatalf("spawned = %d, want 2", len(payload.Spawned))
	}
	if len(payload.Devices) < 25 {
		t.Fatalf("devices = %d, want at least 25", len(payload.Devices))
	}
}
