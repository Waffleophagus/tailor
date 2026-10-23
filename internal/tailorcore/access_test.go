package tailorcore

import (
	"context"
	"testing"

	"github.com/Waffleophagus/tailor/internal/cloudapi"
)

func TestIsPolicyAdminMirrorsTailscalePolicyEditingRoles(t *testing.T) {
	users := []cloudapi.User{
		{LoginName: "owner@example.com", Role: "owner", Status: "active"},
		{LoginName: "Admin@Example.com", Role: "admin", Status: "idle"},
		{LoginName: "netadmin@example.com", Role: "network-admin"},
		{LoginName: "itadmin@example.com", Role: "it-admin", Status: "active"},
		{LoginName: "auditor@example.com", Role: "auditor", Status: "active"},
		{LoginName: "billing@example.com", Role: "billing-admin", Status: "active"},
		{LoginName: "member@example.com", Role: "member", Status: "active"},
		{LoginName: "suspended@example.com", Role: "admin", Status: "suspended"},
	}
	tests := map[string]bool{
		"owner@example.com":     true,
		"admin@example.com":     true, // login comparison is case-insensitive
		"netadmin@example.com":  true,
		"itadmin@example.com":   false, // IT admins cannot edit the policy file
		"auditor@example.com":   false,
		"billing@example.com":   false,
		"member@example.com":    false,
		"suspended@example.com": false,
		"stranger@example.com":  false,
		"":                      false,
	}
	for login, want := range tests {
		if got := isPolicyAdmin(users, login); got != want {
			t.Errorf("isPolicyAdmin(%q) = %v, want %v", login, got, want)
		}
	}
}

func TestContainsNodeMatchesStableIDOrAddress(t *testing.T) {
	devices := []cloudapi.Device{{NodeID: "nA", Addresses: []string{"100.64.0.1", "fd7a::1"}}}
	if !containsNode(devices, "nA", nil) {
		t.Error("stable node ID should match")
	}
	if !containsNode(devices, "", []string{"fd7a::1"}) {
		t.Error("tailnet address should match")
	}
	if containsNode(devices, "nB", []string{"100.64.0.2"}) {
		t.Error("unrelated node matched")
	}
}

func TestIsTailnetPolicyAdminFailsClosedWithoutCloudSession(t *testing.T) {
	service := New(Options{})
	defer service.Close()
	if service.IsTailnetPolicyAdmin(context.Background(), "owner@example.com") {
		t.Fatal("no Cloud API session, so nobody can be verified as admin")
	}
}
