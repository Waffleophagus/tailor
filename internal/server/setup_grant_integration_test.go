//go:build !dev

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/tailcfg"
)

const integrationCapability = "tailor.example.ts.net/cap/admin"

type cloudPolicyFixture struct {
	mu             sync.Mutex
	policy         string
	validateStatus int
	saveStatus     int
	validateCalls  int
	saveCalls      int
}

func (f *cloudPolicyFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path == "/api/v2/tailnet/example.com/acl/validate" {
		f.validateCalls++
		if f.validateStatus != 0 {
			http.Error(w, "validation failed", f.validateStatus)
		}
		return
	}
	if r.URL.Path != "/api/v2/tailnet/example.com/acl" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		f.saveCalls++
		if f.saveStatus != 0 {
			http.Error(w, "save failed", f.saveStatus)
			return
		}
		body, _ := io.ReadAll(r.Body)
		f.policy = string(body)
		return
	}
	_, _ = io.WriteString(w, f.policy)
}

type propagatingWhoIs struct {
	mu        sync.Mutex
	calls     int
	fullAfter int
}

func (w *propagatingWhoIs) WhoIs(_ context.Context, _ string) (*apitype.WhoIsResponse, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.calls++
	response := viewerWhoIs("admin@example.com", "workstation.example.ts.net.")
	if w.fullAfter > 0 && w.calls >= w.fullAfter {
		response.CapMap = tailcfg.PeerCapMap{
			tailcfg.PeerCapability(integrationCapability): []tailcfg.RawMessage{`{"actions":["admin"]}`},
		}
	}
	return response, nil
}

func viewerWhoIs(login, node string) *apitype.WhoIsResponse {
	return &apitype.WhoIsResponse{
		UserProfile: &tailcfg.UserProfile{LoginName: login},
		Node:        &tailcfg.Node{Name: node},
	}
}

func TestSetupGrantFailuresCreateBrowserBootstrapAccess(t *testing.T) {
	tests := []struct {
		name           string
		validateStatus int
		saveStatus     int
	}{
		{name: "validation failure", validateStatus: http.StatusBadRequest},
		{name: "save failure", saveStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := &cloudPolicyFixture{policy: `{"grants":[]}`, validateStatus: tt.validateStatus, saveStatus: tt.saveStatus}
			app, cloud, client := newSetupIntegrationApp(t, fixture, &propagatingWhoIs{})
			defer app.Close()
			defer cloud.Close()

			authenticateSetupClient(t, client, app.URL)
			response := postSetupGrant(t, client, app.URL)
			defer response.Body.Close()
			var setup api.SetupGrantResponse
			if err := json.NewDecoder(response.Body).Decode(&setup); err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusOK || !setup.BootstrapActive || !setup.CanEditPolicy {
				t.Fatalf("status=%d setup=%+v", response.StatusCode, setup)
			}
			if setup.SetupGrantSnippet == "" || !strings.Contains(setup.StatusMessage, "temporarily available") {
				t.Fatalf("bootstrap instructions missing: %+v", setup)
			}

			assertBootstrapPolicyRoutesAreAuthorized(t, client, app.URL)
		})
	}
}

func TestSuccessfulSetupGrantUnlocksOnlyAfterWhoIsPropagation(t *testing.T) {
	fixture := &cloudPolicyFixture{policy: `{"grants":[]}`}
	who := &propagatingWhoIs{fullAfter: 3}
	app, cloud, client := newSetupIntegrationApp(t, fixture, who)
	defer app.Close()
	defer cloud.Close()

	authenticateSetupClient(t, client, app.URL)
	response := postSetupGrant(t, client, app.URL)
	defer response.Body.Close()
	var setup api.SetupGrantResponse
	if err := json.NewDecoder(response.Body).Decode(&setup); err != nil {
		t.Fatal(err)
	}
	if !setup.HasAppCapabilityGrant || !setup.CanEditPolicy || setup.CallerRole != "full" || setup.BootstrapActive {
		t.Fatalf("setup did not unlock through propagated WhoIs capability: %+v", setup)
	}
	if fixture.validateCalls != 1 || fixture.saveCalls != 1 {
		t.Fatalf("validate calls=%d save calls=%d, want 1 each", fixture.validateCalls, fixture.saveCalls)
	}
}

func TestExistingSetupGrantIsNotWrittenAgain(t *testing.T) {
	fixture := &cloudPolicyFixture{policy: `{"grants":[{"src":["autogroup:admin"],"dst":["tag:tailor-acl-service"],"ip":["tcp:443"],"app":{"tailor.example.ts.net/cap/admin":[{"actions":["admin"]}]}}]}`}
	app, cloud, client := newSetupIntegrationApp(t, fixture, &propagatingWhoIs{})
	defer app.Close()
	defer cloud.Close()

	authenticateSetupClient(t, client, app.URL)
	response := postSetupGrant(t, client, app.URL)
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	if fixture.validateCalls != 0 || fixture.saveCalls != 0 {
		t.Fatalf("existing grant triggered validate=%d save=%d", fixture.validateCalls, fixture.saveCalls)
	}
}

func newSetupIntegrationApp(t *testing.T, fixture *cloudPolicyFixture, who WhoIsClient) (*httptest.Server, *httptest.Server, *http.Client) {
	t.Helper()
	cloud := httptest.NewServer(fixture)
	app := httptest.NewServer(New(Options{
		TailnetMode:     true,
		WhoIsClient:     who,
		AppCapability:   integrationCapability,
		CloudAPIOptions: []cloudapi.Option{cloudapi.WithBaseURL(cloud.URL)},
	}))
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return app, cloud, &http.Client{Jar: jar}
}

func authenticateSetupClient(t *testing.T, client *http.Client, appURL string) {
	t.Helper()
	body, _ := json.Marshal(api.CloudAuthRequest{Tailnet: "example.com", APIKey: "tskey-api-test"})
	response, err := client.Post(appURL+"/api/cloud/auth", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(response.Body)
		t.Fatalf("cloud auth status=%d body=%s", response.StatusCode, data)
	}
}

func postSetupGrant(t *testing.T, client *http.Client, appURL string) *http.Response {
	t.Helper()
	response, err := client.Post(appURL+"/api/cloud/setup-grant", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func assertBootstrapPolicyRoutesAreAuthorized(t *testing.T, client *http.Client, appURL string) {
	t.Helper()
	tests := []struct{ method, path string }{
		{http.MethodGet, "/api/policy"},
		{http.MethodGet, "/api/policy/map"},
		{http.MethodPost, "/api/policy/draft"},
		{http.MethodPost, "/api/policy/mutate"},
		{http.MethodPost, "/api/policy/evaluate-draft"},
		{http.MethodPost, "/api/policy/validate"},
		{http.MethodGet, "/api/policy/staged"},
		{http.MethodPost, "/api/policy/stage"},
		{http.MethodGet, "/api/policy/staged/missing"},
		{http.MethodDelete, "/api/policy/staged/missing"},
		{http.MethodPost, "/api/policy/save"},
	}
	for _, tt := range tests {
		request, _ := http.NewRequest(tt.method, appURL+tt.path, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode == http.StatusForbidden {
			t.Errorf("bootstrap request %s %s was forbidden", tt.method, tt.path)
		}
	}
}
