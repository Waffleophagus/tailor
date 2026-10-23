//go:build !dev

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
)

const (
	integrationCapability = "tailor.example.ts.net/cap/admin"
	selfNodeID            = "nSelfTailor"
	selfNodeIP            = "100.64.0.10"
	adminLogin            = "admin@example.com"
	memberLogin           = "member@example.com"
)

// cloudPolicyFixture fakes the parts of the Tailscale API Tailor uses for setup:
// the policy file, its validation, and the device and user lists used to vet keys.
type cloudPolicyFixture struct {
	mu             sync.Mutex
	policy         string
	validateStatus int
	saveStatus     int
	validateCalls  int
	saveCalls      int
	// otherTailnet makes the key's device list omit Tailor's own node.
	otherTailnet bool
}

func (f *cloudPolicyFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.URL.Path {
	case "/api/v2/tailnet/example.com/acl/validate":
		f.validateCalls++
		if f.validateStatus != 0 {
			http.Error(w, "validation failed", f.validateStatus)
		}
	case "/api/v2/tailnet/example.com/devices":
		nodeID, address := selfNodeID, selfNodeIP
		if f.otherTailnet {
			nodeID, address = "nSomeoneElse", "100.100.1.1"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"devices": []map[string]any{
			{"id": "1", "nodeId": nodeID, "addresses": []string{address}},
		}})
	case "/api/v2/tailnet/example.com/users":
		_ = json.NewEncoder(w).Encode(map[string]any{"users": []map[string]any{
			{"loginName": adminLogin, "role": "admin", "status": "active"},
			{"loginName": memberLogin, "role": "member", "status": "active"},
		}})
	case "/api/v2/tailnet/example.com/acl":
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
	default:
		http.NotFound(w, r)
	}
}

// switchableWhoIs identifies every request as the current login, from an untagged
// device without the Tailor app capability.
type switchableWhoIs struct {
	mu    sync.Mutex
	login string
	tags  []string
}

func (w *switchableWhoIs) set(login string, tags ...string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.login, w.tags = login, tags
}

func (w *switchableWhoIs) WhoIs(context.Context, string) (*apitype.WhoIsResponse, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	response := viewerWhoIs(w.login, "workstation.example.ts.net.")
	response.Node.Tags = w.tags
	return response, nil
}

type selfStatus struct{}

func (selfStatus) StatusWithoutPeers(context.Context) (*ipnstate.Status, error) {
	return &ipnstate.Status{Self: &ipnstate.PeerStatus{
		ID:           tailcfg.StableNodeID(selfNodeID),
		TailscaleIPs: []netip.Addr{netip.MustParseAddr(selfNodeIP)},
	}}, nil
}

func viewerWhoIs(login, node string) *apitype.WhoIsResponse {
	return &apitype.WhoIsResponse{
		UserProfile: &tailcfg.UserProfile{LoginName: login},
		Node:        &tailcfg.Node{Name: node},
	}
}

type setupApp struct {
	url    string
	who    *switchableWhoIs
	client *http.Client
}

func newSetupIntegrationApp(t *testing.T, fixture *cloudPolicyFixture) setupApp {
	t.Helper()
	cloud := httptest.NewServer(fixture)
	t.Cleanup(cloud.Close)
	who := &switchableWhoIs{login: adminLogin}
	app := httptest.NewServer(New(Options{
		TailnetMode:     true,
		WhoIsClient:     who,
		TailnetStatus:   selfStatus{},
		AppCapability:   integrationCapability,
		CloudAPIOptions: []cloudapi.Option{cloudapi.WithBaseURL(cloud.URL)},
	}))
	t.Cleanup(app.Close)
	return setupApp{url: app.URL, who: who, client: &http.Client{}}
}

func (a setupApp) authenticate(t *testing.T) *http.Response {
	t.Helper()
	body, _ := json.Marshal(api.CloudAuthRequest{Tailnet: "example.com", APIKey: "tskey-api-test"})
	response, err := postJSON(context.Background(), a.client, a.url+"/api/cloud/auth", body)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func (a setupApp) mustAuthenticate(t *testing.T) {
	t.Helper()
	response := a.authenticate(t)
	defer closeResponseBody(t, response)
	if response.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(response.Body)
		t.Fatalf("cloud auth status=%d body=%s", response.StatusCode, data)
	}
}

func (a setupApp) cloudStatus(t *testing.T) api.CloudAuthStatusResponse {
	t.Helper()
	response, err := a.client.Get(a.url + "/api/cloud/status")
	if err != nil {
		t.Fatal(err)
	}
	defer closeResponseBody(t, response)
	var status api.CloudAuthStatusResponse
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	return status
}

func (a setupApp) postSetupGrant(t *testing.T) *http.Response {
	t.Helper()
	response, err := postJSON(context.Background(), a.client, a.url+"/api/cloud/setup-grant", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func expectStatus(t *testing.T, response *http.Response, want int, bodyContains string) {
	t.Helper()
	defer closeResponseBody(t, response)
	data, _ := io.ReadAll(response.Body)
	if response.StatusCode != want || !strings.Contains(string(data), bodyContains) {
		t.Fatalf("status=%d body=%s, want %d containing %q", response.StatusCode, data, want, bodyContains)
	}
}

func TestTailnetAdminConnectsCloudAPIAndCanEditWithoutGrant(t *testing.T) {
	app := newSetupIntegrationApp(t, &cloudPolicyFixture{policy: `{"grants":[]}`})

	response := app.authenticate(t)
	defer closeResponseBody(t, response)
	var authStatus api.CloudAuthStatusResponse
	if err := json.NewDecoder(response.Body).Decode(&authStatus); err != nil {
		t.Fatal(err)
	}
	// The UI uses this response directly, so it must already reflect admin access.
	if response.StatusCode != http.StatusOK || !authStatus.CanEditPolicy {
		t.Fatalf("auth response status=%d body=%+v, want editable", response.StatusCode, authStatus)
	}

	status := app.cloudStatus(t)
	if !status.CanEditPolicy || status.CallerRole != "full" {
		t.Fatalf("tailnet admin should edit without a grant: %+v", status)
	}
	if !status.NeedsSetupGrant || status.SetupGrantSnippet == "" {
		t.Fatalf("editors should be offered the optional grant: %+v", status)
	}
	assertPolicyRoutesAreAuthorized(t, app.client, app.url)
}

func TestNonAdminCannotConnectCloudAPI(t *testing.T) {
	app := newSetupIntegrationApp(t, &cloudPolicyFixture{policy: `{"grants":[]}`})
	app.who.set(memberLogin)

	expectStatus(t, app.authenticate(t), http.StatusForbidden, "Owners, Admins and Network admins")
	if app.cloudStatus(t).Authenticated {
		t.Fatal("a rejected key was stored")
	}
}

func TestTaggedDeviceCannotConnectCloudAPI(t *testing.T) {
	app := newSetupIntegrationApp(t, &cloudPolicyFixture{policy: `{"grants":[]}`})
	app.who.set(adminLogin, "tag:server")

	expectStatus(t, app.authenticate(t), http.StatusForbidden, "tagged")
}

func TestCloudAPIKeyForAnotherTailnetIsRejected(t *testing.T) {
	app := newSetupIntegrationApp(t, &cloudPolicyFixture{policy: `{"grants":[]}`, otherTailnet: true})

	expectStatus(t, app.authenticate(t), http.StatusForbidden, "different tailnet")
	if app.cloudStatus(t).Authenticated {
		t.Fatal("a key for another tailnet was stored")
	}
}

func TestViewerCannotReplaceConfiguredCloudKey(t *testing.T) {
	app := newSetupIntegrationApp(t, &cloudPolicyFixture{policy: `{"grants":[]}`})
	app.mustAuthenticate(t)

	app.who.set(memberLogin)
	expectStatus(t, app.authenticate(t), http.StatusForbidden, "Only Tailor editors")
}

func TestViewerCannotApplySetupGrant(t *testing.T) {
	fixture := &cloudPolicyFixture{policy: `{"grants":[]}`}
	app := newSetupIntegrationApp(t, fixture)
	app.mustAuthenticate(t)

	app.who.set(memberLogin)
	expectStatus(t, app.postSetupGrant(t), http.StatusForbidden, "editors")
	if fixture.saveCalls != 0 {
		t.Fatalf("viewer request wrote the policy (%d saves)", fixture.saveCalls)
	}
}

func TestSetupGrantFailuresAreReportedWithoutGrantingAccess(t *testing.T) {
	tests := []struct {
		name           string
		validateStatus int
		saveStatus     int
		wantStatus     int
		wantMessage    string
	}{
		{name: "validation failure", validateStatus: http.StatusBadRequest, wantStatus: http.StatusBadRequest, wantMessage: "rejected the access grant"},
		{name: "save failure", saveStatus: http.StatusInternalServerError, wantStatus: http.StatusBadGateway, wantMessage: "could not save the access grant"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := &cloudPolicyFixture{policy: `{"grants":[]}`, validateStatus: tt.validateStatus, saveStatus: tt.saveStatus}
			app := newSetupIntegrationApp(t, fixture)
			app.mustAuthenticate(t)

			expectStatus(t, app.postSetupGrant(t), tt.wantStatus, tt.wantMessage)

			// Non-admins get nothing out of a failed grant.
			app.who.set(memberLogin)
			if status := app.cloudStatus(t); status.CanEditPolicy {
				t.Fatalf("member gained edit access after a failed grant: %+v", status)
			}
		})
	}
}

func TestSuccessfulSetupGrantIsSavedOnce(t *testing.T) {
	fixture := &cloudPolicyFixture{policy: `{"grants":[]}`}
	app := newSetupIntegrationApp(t, fixture)
	app.mustAuthenticate(t)

	response := app.postSetupGrant(t)
	defer closeResponseBody(t, response)
	var setup api.SetupGrantResponse
	if err := json.NewDecoder(response.Body).Decode(&setup); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !setup.HasAppCapabilityGrant || !setup.CanEditPolicy {
		t.Fatalf("status=%d setup=%+v", response.StatusCode, setup)
	}
	if fixture.validateCalls != 1 || fixture.saveCalls != 1 {
		t.Fatalf("validate calls=%d save calls=%d, want 1 each", fixture.validateCalls, fixture.saveCalls)
	}
	if !strings.Contains(fixture.policy, integrationCapability) {
		t.Fatalf("saved policy is missing the grant: %s", fixture.policy)
	}
}

func TestExistingSetupGrantIsNotWrittenAgain(t *testing.T) {
	fixture := &cloudPolicyFixture{policy: `{"grants":[{"src":["autogroup:admin"],"dst":["tag:tailor-acl-service"],"ip":["tcp:443"],"app":{"tailor.example.ts.net/cap/admin":[{"actions":["admin"]}]}}]}`}
	app := newSetupIntegrationApp(t, fixture)
	app.mustAuthenticate(t)

	expectStatus(t, app.postSetupGrant(t), http.StatusOK, "")
	if fixture.validateCalls != 0 || fixture.saveCalls != 0 {
		t.Fatalf("existing grant triggered validate=%d save=%d", fixture.validateCalls, fixture.saveCalls)
	}
}

func postJSON(ctx context.Context, client *http.Client, url string, body []byte) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	return client.Do(request)
}

func closeResponseBody(t *testing.T, response *http.Response) {
	t.Helper()
	if response == nil || response.Body == nil {
		return
	}
	if err := response.Body.Close(); err != nil {
		t.Fatalf("response.Body.Close() error = %v", err)
	}
}

func assertPolicyRoutesAreAuthorized(t *testing.T, client *http.Client, appURL string) {
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
		request, err := http.NewRequestWithContext(context.Background(), tt.method, appURL+tt.path, strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		closeResponseBody(t, response)
		if response.StatusCode == http.StatusForbidden {
			t.Errorf("admin request %s %s was forbidden", tt.method, tt.path)
		}
	}
}
