//go:build !dev

package tailorcore

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"tailscale.com/client/local"
)

// fakeACLServer serves a tailnet policy that can be edited out-of-band, the way an
// admin-console or GitOps change would be.
type fakeACLServer struct {
	mu        sync.Mutex
	policy    string
	saveCalls int
	// changeAfterGet, when set, replaces the policy right after the next GET is
	// served: an edit landing between Tailor's read and its write.
	changeAfterGet string
	// noETag drops the ETag header, as a proxy or API change might.
	noETag bool
}

func (f *fakeACLServer) etagLocked() string {
	return fmt.Sprintf("%q", fmt.Sprintf("%x", sha256.Sum256([]byte(f.policy))))
}

func (f *fakeACLServer) setPolicy(policy string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.policy = policy
}

func (f *fakeACLServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.URL.Path == "/api/v2/tailnet/example.com/acl/validate" {
		_, _ = io.WriteString(w, `{}`)
		return
	}
	if r.URL.Path != "/api/v2/tailnet/example.com/acl" {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		if ifMatch := r.Header.Get("If-Match"); ifMatch != "" && ifMatch != f.etagLocked() {
			http.Error(w, `{"message":"precondition failed"}`, http.StatusPreconditionFailed)
			return
		}
		f.saveCalls++
		body, _ := io.ReadAll(r.Body)
		f.policy = string(body)
		_, _ = w.Write(body)
		return
	}
	if !f.noETag {
		w.Header().Set("ETag", f.etagLocked())
	}
	_, _ = io.WriteString(w, f.policy)
	if f.changeAfterGet != "" {
		f.policy, f.changeAfterGet = f.changeAfterGet, ""
	}
}

// newTestService returns a Service talking to a fake Cloud API at cloudURL and a
// fake local tailscaled, so tests never depend on Tailscale running on the host.
func newTestService(t *testing.T, cloudURL string) *Service {
	t.Helper()
	tailscaled := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/localapi/v0/status" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"BackendState":"Running","Self":{"ID":"nTailor","HostName":"tailor","DNSName":"tailor.example.ts.net."},"Peer":{}}`)
	}))
	t.Cleanup(tailscaled.Close)
	localClient := &local.Client{Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		var dialer net.Dialer
		return dialer.DialContext(ctx, "tcp", tailscaled.Listener.Addr().String())
	}}
	service := New(Options{
		LocalClient:      localClient,
		LocalAPIEndpoint: "fake tailscaled",
		CloudAPIOptions:  []cloudapi.Option{cloudapi.WithBaseURL(cloudURL)},
	})
	t.Cleanup(service.Close)
	return service
}

func TestSaveStagedPolicyRejectsOutOfBandPolicyChange(t *testing.T) {
	ctx := context.Background()
	fake := &fakeACLServer{policy: `{"acls":[{"action":"accept","src":["*"],"dst":["*:*"]}]}`}
	cloud := httptest.NewServer(fake)
	defer cloud.Close()

	service := newTestService(t, cloud.URL)

	if _, err := service.AuthenticateCloud(ctx, cloudapi.AuthRequest{
		Tailnet: "example.com",
		APIKey:  "tskey-api-test",
	}); err != nil {
		t.Fatal(err)
	}
	current, err := service.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := service.StagePolicyDraft(ctx, api.PolicyStageRequest{
		HuJSON: current.HuJSON,
		Source: "ui",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Someone edits the policy outside Tailor; Tailor's session cache still holds the old one.
	fake.setPolicy(`{"acls":[{"action":"accept","src":["autogroup:admin"],"dst":["*:*"]}]}`)

	_, err = service.SaveStagedPolicy(ctx, api.PolicySaveRequest{
		DraftID:   staged.Draft.ID,
		DraftHash: staged.Draft.DraftHash,
	})
	if !errors.Is(err, ErrStagedDraftBaseMismatch) {
		t.Fatalf("SaveStagedPolicy error = %v, want %v", err, ErrStagedDraftBaseMismatch)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.saveCalls != 0 {
		t.Fatalf("stale draft was written to the tailnet (%d save calls)", fake.saveCalls)
	}
}

func TestSaveStagedPolicyRejectsChangeBetweenReadAndWrite(t *testing.T) {
	ctx := context.Background()
	fake := &fakeACLServer{policy: `{"acls":[{"action":"accept","src":["*"],"dst":["*:*"]}]}`}
	cloud := httptest.NewServer(fake)
	defer cloud.Close()

	service := newTestService(t, cloud.URL)

	if _, err := service.AuthenticateCloud(ctx, cloudapi.AuthRequest{
		Tailnet: "example.com",
		APIKey:  "tskey-api-test",
	}); err != nil {
		t.Fatal(err)
	}
	current, err := service.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := service.StagePolicyDraft(ctx, api.PolicyStageRequest{HuJSON: current.HuJSON, Source: "ui"})
	if err != nil {
		t.Fatal(err)
	}

	fake.mu.Lock()
	fake.changeAfterGet = `{"acls":[{"action":"accept","src":["autogroup:admin"],"dst":["*:*"]}]}`
	fake.mu.Unlock()

	_, err = service.SaveStagedPolicy(ctx, api.PolicySaveRequest{
		DraftID:   staged.Draft.ID,
		DraftHash: staged.Draft.DraftHash,
	})
	if !errors.Is(err, ErrStagedDraftBaseMismatch) {
		t.Fatalf("SaveStagedPolicy error = %v, want %v", err, ErrStagedDraftBaseMismatch)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.saveCalls != 0 {
		t.Fatalf("draft overwrote a concurrent change (%d successful saves)", fake.saveCalls)
	}
}

func TestStaleEditorDraftCannotOverwriteNewerSave(t *testing.T) {
	ctx := context.Background()
	fake := &fakeACLServer{policy: `{"acls":[{"action":"accept","src":["*"],"dst":["*:*"]}]}`}
	cloud := httptest.NewServer(fake)
	defer cloud.Close()
	service := newTestService(t, cloud.URL)
	if _, err := service.AuthenticateCloud(ctx, cloudapi.AuthRequest{Tailnet: "example.com", APIKey: "tskey-api-test"}); err != nil {
		t.Fatal(err)
	}

	// Editors A and B both load policy P0.
	p0, err := service.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if p0.Revision == "" {
		t.Fatal("policy response has no revision")
	}

	// A saves P1.
	stagedA, err := service.StagePolicyDraft(ctx, api.PolicyStageRequest{
		HuJSON: `{"acls":[{"action":"accept","src":["autogroup:admin"],"dst":["*:*"]}]}`, Source: "ui", BaseRevision: p0.Revision,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveStagedPolicy(ctx, api.PolicySaveRequest{DraftID: stagedA.Draft.ID, DraftHash: stagedA.Draft.DraftHash}); err != nil {
		t.Fatalf("editor A's save failed: %v", err)
	}

	// B, still editing from P0, stages and saves afterwards.
	stagedB, err := service.StagePolicyDraft(ctx, api.PolicyStageRequest{
		HuJSON: `{"acls":[{"action":"accept","src":["autogroup:member"],"dst":["*:*"]}]}`, Source: "ui", BaseRevision: p0.Revision,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.SaveStagedPolicy(ctx, api.PolicySaveRequest{DraftID: stagedB.Draft.ID, DraftHash: stagedB.Draft.DraftHash})
	if !errors.Is(err, ErrStagedDraftBaseMismatch) {
		t.Fatalf("editor B's stale save error = %v, want %v", err, ErrStagedDraftBaseMismatch)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if !strings.Contains(fake.policy, "autogroup:admin") {
		t.Fatalf("editor A's change was overwritten: %s", fake.policy)
	}
}

func TestSaveStagedPolicyRefusesWithoutPolicyVersion(t *testing.T) {
	ctx := context.Background()
	fake := &fakeACLServer{policy: `{"acls":[{"action":"accept","src":["*"],"dst":["*:*"]}]}`, noETag: true}
	cloud := httptest.NewServer(fake)
	defer cloud.Close()
	service := newTestService(t, cloud.URL)
	if _, err := service.AuthenticateCloud(ctx, cloudapi.AuthRequest{Tailnet: "example.com", APIKey: "tskey-api-test"}); err != nil {
		t.Fatal(err)
	}
	current, err := service.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := service.StagePolicyDraft(ctx, api.PolicyStageRequest{HuJSON: current.HuJSON, Source: "ui", BaseRevision: current.Revision})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.SaveStagedPolicy(ctx, api.PolicySaveRequest{DraftID: staged.Draft.ID, DraftHash: staged.Draft.DraftHash})
	if !errors.Is(err, cloudapi.ErrPolicyVersionUnavailable) {
		t.Fatalf("err = %v, want ErrPolicyVersionUnavailable", err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if fake.saveCalls != 0 {
		t.Fatalf("saved %d times without conflict protection", fake.saveCalls)
	}
}
