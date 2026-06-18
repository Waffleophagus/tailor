package server

import (
	"context"
	"slices"
	"testing"

	"github.com/Waffleophagus/tailor/internal/authz"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/ipn"
	"tailscale.com/tailcfg"
)

type fakePrefsClient struct {
	prefs *ipn.Prefs
	edit  *ipn.MaskedPrefs
}

func (f *fakePrefsClient) GetPrefs(context.Context) (*ipn.Prefs, error) {
	return f.prefs, nil
}

func (f *fakePrefsClient) EditPrefs(_ context.Context, prefs *ipn.MaskedPrefs) (*ipn.Prefs, error) {
	f.edit = prefs
	return &prefs.Prefs, nil
}

type sequenceWhoIsClient struct {
	responses []*apitype.WhoIsResponse
	calls     int
}

func (f *sequenceWhoIsClient) WhoIs(context.Context, string) (*apitype.WhoIsResponse, error) {
	index := f.calls
	f.calls++
	if index >= len(f.responses) {
		index = len(f.responses) - 1
	}
	return f.responses[index], nil
}

func TestWaitForAdminCapabilityObservesPropagation(t *testing.T) {
	const capability = "tailor.example.ts.net/cap/admin"
	viewer := &apitype.WhoIsResponse{}
	full := &apitype.WhoIsResponse{
		CapMap: tailcfg.PeerCapMap{
			tailcfg.PeerCapability(capability): []tailcfg.RawMessage{`{"actions":["admin"]}`},
		},
	}
	client := &sequenceWhoIsClient{responses: []*apitype.WhoIsResponse{viewer, full}}
	server := &Server{auth: AuthOptions{TailnetMode: true, WhoIsClient: client}}

	identity, ok := server.waitForAdminCapability(context.Background(), "100.64.0.1:1234", capability)
	if !ok {
		t.Fatal("expected propagated capability to be observed")
	}
	if identity.Role != authz.RoleFull {
		t.Fatalf("role = %q, want full", identity.Role)
	}
	if client.calls != 2 {
		t.Fatalf("WhoIs calls = %d, want 2", client.calls)
	}
}

func TestWaitForAdminCapabilitySkipsNonTailnetMode(t *testing.T) {
	client := &sequenceWhoIsClient{responses: []*apitype.WhoIsResponse{{}}}
	server := &Server{auth: AuthOptions{WhoIsClient: client}}

	if _, ok := server.waitForAdminCapability(context.Background(), "127.0.0.1:1234", "cap"); ok {
		t.Fatal("non-tailnet mode should not poll")
	}
	if client.calls != 0 {
		t.Fatalf("WhoIs calls = %d, want 0", client.calls)
	}
}

func TestActivateServiceTagPreservesExistingAdvertisedTags(t *testing.T) {
	client := &fakePrefsClient{prefs: &ipn.Prefs{AdvertiseTags: []string{"tag:existing"}}}
	server := &Server{
		auth:         AuthOptions{TailnetMode: true},
		tailnetPrefs: client,
	}

	if err := server.activateServiceTag(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.edit == nil || !client.edit.AdvertiseTagsSet {
		t.Fatal("expected advertised tags to be updated")
	}
	want := []string{"tag:existing", "tag:tailor-acl-service"}
	if !slices.Equal(client.edit.AdvertiseTags, want) {
		t.Fatalf("advertised tags = %#v, want %#v", client.edit.AdvertiseTags, want)
	}
}

func TestActivateServiceTagDoesNotRewriteExistingServiceTag(t *testing.T) {
	client := &fakePrefsClient{prefs: &ipn.Prefs{AdvertiseTags: []string{"tag:tailor-acl-service"}}}
	server := &Server{
		auth:         AuthOptions{TailnetMode: true},
		tailnetPrefs: client,
	}

	if err := server.activateServiceTag(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.edit != nil {
		t.Fatal("existing service tag should not be rewritten")
	}
}
