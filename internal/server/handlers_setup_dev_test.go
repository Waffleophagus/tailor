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

func TestDevKeyAlwaysOffersAndSimulatesSetupGrant(t *testing.T) {
	mux := httptest.NewServer(New())
	t.Cleanup(mux.Close)

	body, _ := json.Marshal(api.CloudAuthRequest{Tailnet: "-", APIKey: devtailnet.APIKey})
	resp, err := http.Post(mux.URL+"/api/cloud/auth", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var auth api.CloudAuthStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}
	if !auth.NeedsSetupGrant || !auth.CanEditPolicy || auth.CallerRole != "full" || auth.AppCapability == "" || auth.SetupGrantSnippet == "" {
		t.Fatalf("dev auth did not offer setup grant: %+v", auth)
	}

	save, err := http.Post(mux.URL+"/api/cloud/setup-grant", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer save.Body.Close()
	var setup api.SetupGrantResponse
	if err := json.NewDecoder(save.Body).Decode(&setup); err != nil {
		t.Fatal(err)
	}
	if save.StatusCode != http.StatusOK || !setup.HasAppCapabilityGrant || !setup.CanEditPolicy {
		t.Fatalf("dev setup grant was not simulated: status=%d response=%+v", save.StatusCode, setup)
	}
}
