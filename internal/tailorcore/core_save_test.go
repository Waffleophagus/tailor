//go:build !dev

package tailorcore

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
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
	w.Header().Set("ETag", f.etagLocked())
	_, _ = io.WriteString(w, f.policy)
	if f.changeAfterGet != "" {
		f.policy, f.changeAfterGet = f.changeAfterGet, ""
	}
}

func TestSaveStagedPolicyRejectsOutOfBandPolicyChange(t *testing.T) {
	ctx := context.Background()
	fake := &fakeACLServer{policy: `{"acls":[{"action":"accept","src":["*"],"dst":["*:*"]}]}`}
	cloud := httptest.NewServer(fake)
	defer cloud.Close()

	service := New(Options{CloudAPIOptions: []cloudapi.Option{cloudapi.WithBaseURL(cloud.URL)}})
	defer service.Close()

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

	service := New(Options{CloudAPIOptions: []cloudapi.Option{cloudapi.WithBaseURL(cloud.URL)}})
	defer service.Close()

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
