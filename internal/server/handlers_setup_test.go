package server

import (
	"context"
	"slices"
	"testing"

	"tailscale.com/ipn"
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
