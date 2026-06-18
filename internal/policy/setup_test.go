package policy

import (
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
)

const testCapability = "tailor.example.ts.net/cap/admin"

func TestHasTailorAppCapabilityGrant(t *testing.T) {
	raw := `{
		"grants": [
			{
				"src": ["autogroup:owner"],
				"dst": ["tag:tailor-acl-service"],
				"ip": ["tcp:443"],
				"app": {
					"tailor.example.ts.net/cap/admin": [{"actions": ["admin"]}]
				}
			}
		]
	}`
	if !HasTailorAppCapabilityGrant(raw, testCapability) {
		t.Fatal("expected grant to be detected")
	}
	if HasTailorAppCapabilityGrant(raw, "other.example.ts.net/cap/admin") {
		t.Fatal("expected wrong capability to be absent")
	}
	if HasTailorAppCapabilityGrant(`{"grants": []}`, testCapability) {
		t.Fatal("expected empty grants to be absent")
	}
}

func TestValidateSetupGrantRejectsUnrelatedGrant(t *testing.T) {
	grant := RecommendedSetupGrant("other.example.ts.net/cap/admin")
	if err := ValidateSetupGrant(grant, testCapability); err == nil {
		t.Fatal("unrelated capability should be rejected")
	}
}

func TestValidateSetupGrantRejectsWrongDestination(t *testing.T) {
	grant := RecommendedSetupGrant(testCapability)
	grant.Dst = []string{"tag:database"}
	if err := ValidateSetupGrant(grant, testCapability); err == nil {
		t.Fatal("non-service destination should be rejected")
	}
}

func TestValidateSetupGrantRejectsAdditionalPolicyScope(t *testing.T) {
	grant := RecommendedSetupGrant(testCapability)
	grant.Dst = append(grant.Dst, "tag:database")
	grant.App["other.example.ts.net/cap/admin"] = []map[string]any{{"actions": []string{"admin"}}}
	if err := ValidateSetupGrant(grant, testCapability); err == nil {
		t.Fatal("additional destinations or capabilities should be rejected")
	}
}

func TestValidateSetupGrantAcceptsEditedSources(t *testing.T) {
	grant := RecommendedSetupGrant(testCapability)
	grant.Src = []string{"group:security"}
	if err := ValidateSetupGrant(grant, testCapability); err != nil {
		t.Fatalf("valid edited setup grant rejected: %v", err)
	}
}

func TestRecommendedSetupGrant(t *testing.T) {
	grant := RecommendedSetupGrant(testCapability)
	if len(grant.Src) != 2 || grant.Dst[0] != TailorACLServiceTag {
		t.Fatalf("unexpected grant shape: %#v", grant)
	}
	snippet := FormatGrantSnippet(grant)
	if snippet == "" {
		t.Fatal("expected non-empty snippet")
	}
}

func TestAppendSetupGrant(t *testing.T) {
	raw := `{
		"acls": []
	}`
	grant := RecommendedSetupGrant(testCapability)
	updated, err := AppendSetupGrant(raw, grant)
	if err != nil {
		t.Fatalf("AppendSetupGrant() error = %v", err)
	}
	if !HasTailorAppCapabilityGrant(updated, testCapability) {
		t.Fatal("expected appended policy to contain setup grant")
	}
	_, err = AppendSetupGrant(updated, grant)
	if err == nil {
		t.Fatal("expected duplicate append to fail")
	}
}

func TestGrantHasAdminAction(t *testing.T) {
	app := map[string]any{
		testCapability: []map[string]any{{"actions": []string{"read"}}},
	}
	if grantHasAdminAction(app, testCapability) {
		t.Fatal("read-only action should not count as admin")
	}
	app[testCapability] = []map[string]any{{"actions": []string{"admin"}}}
	if !grantHasAdminAction(app, testCapability) {
		t.Fatal("admin action should be detected")
	}
}

func TestCapabilityFromGrant(t *testing.T) {
	grant := api.GrantDraft{App: map[string]any{testCapability: nil}}
	if got := capabilityFromGrant(grant); got != testCapability {
		t.Fatalf("capabilityFromGrant() = %q, want %q", got, testCapability)
	}
}
