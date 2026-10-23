package mcpserver

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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
	// 404, like an unknown session, so probes cannot tell whether an ID exists.
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("hijacked session status = %d, want %d", response.StatusCode, http.StatusNotFound)
	}
}

func postRaw(t *testing.T, method, endpoint, sessionID string, header http.Header) int {
	t.Helper()
	request, err := http.NewRequestWithContext(context.Background(), method, endpoint,
		bytes.NewBufferString(`{"jsonrpc":"2.0","id":7,"method":"tools/list"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	request.Header.Set(sessionIDHeader, sessionID)
	for key, values := range header {
		request.Header[key] = values
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	return response.StatusCode
}

func TestMCPRejectedDeleteKeepsSessionOwner(t *testing.T) {
	who := &identitySwitch{identity: aliceAdmin}
	endpoint := newHTTPMCP(t, who)
	session := connectHTTP(t, endpoint, http.DefaultClient)

	// The SDK rejects this DELETE (bad protocol version) and keeps the session.
	status := postRaw(t, http.MethodDelete, endpoint, session.ID(), http.Header{"Mcp-Protocol-Version": {"1999-01-01"}})
	if status >= 200 && status < 300 {
		t.Fatalf("expected the SDK to reject the DELETE, got %d", status)
	}

	// Alice still owns her session...
	if status := postRaw(t, http.MethodPost, endpoint, session.ID(), nil); status == http.StatusNotFound {
		t.Fatal("a rejected DELETE dropped the owner of a live session")
	}
	// ...and nobody else can use it.
	who.set(bobViewer)
	if status := postRaw(t, http.MethodPost, endpoint, session.ID(), nil); status != http.StatusNotFound {
		t.Fatalf("after a rejected DELETE, another identity got status %d, want %d", status, http.StatusNotFound)
	}
}

func TestMCPUnknownSessionIsRejected(t *testing.T) {
	who := &identitySwitch{identity: aliceAdmin}
	endpoint := newHTTPMCP(t, who)
	if status := postRaw(t, http.MethodPost, endpoint, "not-a-real-session", nil); status != http.StatusNotFound {
		t.Fatalf("unknown session status = %d, want %d", status, http.StatusNotFound)
	}
}

func TestSessionOwnersRejectUnclaimedSessions(t *testing.T) {
	// A live SDK session whose owner record is missing must not become usable by
	// anyone; the SDK alone would accept it.
	owners := newSessionOwners(time.Hour)
	if owners.acquire("live-but-unclaimed", "bob|desktop") {
		t.Fatal("an unclaimed session ID was allowed")
	}
	owners.claim("s1", "alice|laptop")
	if !owners.acquire("s1", "alice|laptop") || owners.acquire("s1", "bob|desktop") {
		t.Fatal("ownership check is wrong for a claimed session")
	}
	owners.done("s1")
}

func TestSessionOwnersPurgeOnClaim(t *testing.T) {
	owners := newSessionOwners(-2 * time.Minute) // every entry is already past its lifetime
	owners.claim("abandoned", "alice|laptop")
	owners.lastPurge = time.Time{} // allow the next purge immediately
	owners.claim("fresh", "bob|desktop")
	owners.mu.Lock()
	defer owners.mu.Unlock()
	if _, ok := owners.owners["abandoned"]; ok {
		t.Fatal("initialize-only sessions are never purged")
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

func TestSessionOwnerOutlivesLongRequests(t *testing.T) {
	// The SDK pauses its idle timer during a request and restarts it when the
	// request finishes; ownership must not expire earlier than the SDK session.
	owners := newSessionOwners(10 * time.Minute)
	owners.claim("s1", "alice|laptop")
	if !owners.acquire("s1", "alice|laptop") {
		t.Fatal("owner could not use its session")
	}

	// A very long tool call: the request started long ago.
	owners.mu.Lock()
	owner := owners.owners["s1"]
	owner.lastSeen = time.Now().Add(-2 * time.Hour)
	owners.owners["s1"] = owner
	owners.lastPurge = time.Time{}
	owners.mu.Unlock()
	owners.claim("other", "bob|desktop") // triggers a purge
	owners.mu.Lock()
	_, alive := owners.owners["s1"]
	owners.mu.Unlock()
	if !alive {
		t.Fatal("a session with a request in flight was expired")
	}

	// When the request finishes, the idle clock restarts from now.
	owners.done("s1")
	owners.mu.Lock()
	owners.lastPurge = time.Time{}
	owners.mu.Unlock()
	if !owners.acquire("s1", "alice|laptop") {
		t.Fatal("session expired right after its long request finished")
	}
	owners.done("s1")
}
