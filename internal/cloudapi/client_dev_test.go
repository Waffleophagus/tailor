//go:build dev

package cloudapi

import (
	"context"
	"strings"
	"testing"
)

func TestAuthenticateWithDevKeyUsesInMemoryPolicy(t *testing.T) {
	client := New()
	status, err := client.Authenticate(context.Background(), AuthRequest{
		Tailnet: "-",
		APIKey:  "tskey-api-tailor-dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Authenticated || !status.HasPolicy || !status.DevMode {
		t.Fatalf("unexpected status: %#v", status)
	}
	if status.Tailnet != "demo.tailor.ts.net" {
		t.Fatalf("tailnet = %q", status.Tailnet)
	}

	policy, err := client.Policy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(policy, "group:eng") {
		t.Fatalf("expected demo policy, got: %q", policy)
	}
}

func TestValidateAndSavePolicyInDevMode(t *testing.T) {
	client := New()
	if _, err := client.Authenticate(context.Background(), AuthRequest{
		Tailnet: "-",
		APIKey:  "tskey-api-tailor-dev",
	}); err != nil {
		t.Fatal(err)
	}
	draft := `{"acls":[{"action":"accept","src":["*"],"dst":["*:*"]}]}`
	if err := client.ValidatePolicy(context.Background(), draft); err != nil {
		t.Fatal(err)
	}
	saved, err := client.SavePolicy(context.Background(), draft)
	if err != nil {
		t.Fatal(err)
	}
	if saved != draft {
		t.Fatalf("saved = %q, want draft", saved)
	}
}

func TestSavePolicyInvalidInDevMode(t *testing.T) {
	client := New()
	if _, err := client.Authenticate(context.Background(), AuthRequest{
		Tailnet: "-",
		APIKey:  "tskey-api-tailor-dev",
	}); err != nil {
		t.Fatal(err)
	}

	invalidDraft := `{"acls":[`
	saved, err := client.SavePolicy(context.Background(), invalidDraft)
	if err == nil {
		t.Fatalf("SavePolicy(%q) error = nil, saved = %q", invalidDraft, saved)
	}
}
