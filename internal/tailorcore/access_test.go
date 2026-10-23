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

func TestContainsNodeRequiresOwnedStableNodeID(t *testing.T) {
	devices := []cloudapi.Device{
		{NodeID: "nA", Addresses: []string{"100.64.0.1"}},
		// Another tailnet's device reusing Tailor's address, and Tailor shared in.
		{NodeID: "nOther", Addresses: []string{"100.64.0.9"}},
		{NodeID: "nShared", Addresses: []string{"100.64.0.7"}, IsExternal: true},
	}
	if !containsNode(devices, "nA") {
		t.Error("owned device with matching stable node ID should match")
	}
	if containsNode(devices, "nB") {
		t.Error("an address match must not count: tailnet owners can reuse any 100.x address")
	}
	if containsNode(devices, "nShared") {
		t.Error("a shared-in (external) device is not a member of the tailnet")
	}
	if containsNode(devices, "") {
		t.Error("an empty node ID must never match")
	}
}

func TestIsTailnetPolicyAdminFailsClosedWithoutCloudSession(t *testing.T) {
	service := New(Options{})
	defer service.Close()
	if service.IsTailnetPolicyAdmin(context.Background(), "owner@example.com") {
		t.Fatal("no Cloud API session, so nobody can be verified as admin")
	}
}
