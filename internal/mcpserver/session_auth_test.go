package mcpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Waffleophagus/tailor/internal/authz"
	"github.com/Waffleophagus/tailor/internal/tailorcore"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// identitySwitch plays the role of Tailor's tailnet identity middleware, with an
// identity the test can change between requests.
type identitySwitch struct {
	mu       sync.Mutex
	identity authz.TailnetIdentity
}

func (s *identitySwitch) set(identity authz.TailnetIdentity) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.identity = identity
}

func (s *identitySwitch) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		identity := s.identity
		s.mu.Unlock()
		next.ServeHTTP(w, r.WithContext(authz.WithIdentity(r.Context(), identity)))
	})
}

var (
	aliceAdmin  = authz.TailnetIdentity{LoginName: "alice@example.com", NodeName: "laptop", Role: authz.RoleFull}
	aliceViewer = authz.TailnetIdentity{LoginName: "alice@example.com", NodeName: "laptop", Role: authz.RoleViewer}
	bobViewer   = authz.TailnetIdentity{LoginName: "bob@example.com", NodeName: "desktop", Role: authz.RoleViewer}
)

func newHTTPMCP(t *testing.T, who *identitySwitch) string {
	t.Helper()
	core := tailorcore.New(tailorcore.Options{})
	t.Cleanup(core.Close)
	server := httptest.NewServer(who.wrap(Handler(core, Config{Exposure: ExposureTailnet, Path: "/mcp"}, nil)))
	t.Cleanup(server.Close)
	return server.URL + "/mcp"
}

func connectHTTP(t *testing.T, endpoint string, client *http.Client) *mcp.ClientSession {
	t.Helper()
	transport := &mcp.StreamableClientTransport{Endpoint: endpoint, HTTPClient: client, MaxRetries: -1}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "test"}, nil).
		Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestMCPSessionCannotBeUsedByAnotherIdentity(t *testing.T) {
	who := &identitySwitch{identity: aliceAdmin}
	endpoint := newHTTPMCP(t, who)
	session := connectHTTP(t, endpoint, http.DefaultClient)

	who.set(bobViewer)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint,
		bytes.NewBufferString(`{"jsonrpc":"2.0","id":99,"method":"tools/list"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set(sessionIDHeader, session.ID())
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("hijacked session status = %d, want %d", response.StatusCode, http.StatusForbidden)
	}
}

func TestMCPRoleIsRecheckedOnEveryCall(t *testing.T) {
	who := &identitySwitch{identity: aliceAdmin}
	session := connectHTTP(t, newHTTPMCP(t, who), http.DefaultClient)

	assertToolNotDeniedAsViewOnly(t, session, "tailor_stage_policy_draft", map[string]any{"hujson": "{}"})

	// Alice's admin capability is revoked while her session is still open.
	who.set(aliceViewer)
	assertToolDenied(t, session, "tailor_stage_policy_draft", "view-only")
	assertToolDenied(t, session, "tailor_get_policy", "view-only")
}

type headerInjector struct {
	base   http.RoundTripper
	header string
	value  string
}

func (h headerInjector) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header.Set(h.header, h.value)
	return h.base.RoundTrip(request)
}

func TestMCPClientCannotForgeItsRole(t *testing.T) {
	who := &identitySwitch{identity: bobViewer}
	forging := &http.Client{Transport: headerInjector{base: http.DefaultTransport, header: callerRoleHeader, value: string(authz.RoleFull)}}
	session := connectHTTP(t, newHTTPMCP(t, who), forging)

	assertToolDenied(t, session, "tailor_stage_policy_draft", "view-only")
}
