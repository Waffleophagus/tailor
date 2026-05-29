package devtailnet

import (
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/policy"
)

func TestDevTailnetPolicyProducesVariedAccessScopes(t *testing.T) {
	edges, err := policy.EffectiveAccessEdges(Policy(), Devices(), policy.EdgeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(edges) < 8 {
		t.Fatalf("expected a rich edge set, got %d: %#v", len(edges), edges)
	}

	scopes := map[api.AccessScope]bool{}
	for _, edge := range edges {
		scopes[edge.AccessScope] = true
	}
	for _, want := range []api.AccessScope{
		api.AccessScopeHTTP,
		api.AccessScopeSSH,
		api.AccessScopeLimited,
	} {
		if !scopes[want] {
			t.Fatalf("missing scope %q in %#v", want, scopes)
		}
	}
}

func TestDevTailnetPerspectiveLeavesUnreachableTargets(t *testing.T) {
	edges := policy.ResolveEffectiveAccess(mustParsePolicy(t), Devices(), policy.EdgeOptions{
		Perspective: "alice@demo.tailor.ts.net",
	})
	reachable := map[string]bool{}
	for _, edge := range edges {
		reachable[edge.To] = true
	}
	if reachable["dev-db-primary"] {
		t.Fatal("alice should not reach db-primary directly in saved policy")
	}
}

func TestDevAPIKeyConstant(t *testing.T) {
	if !IsDevAPIKey(APIKey) {
		t.Fatal("APIKey must match IsDevAPIKey")
	}
	if IsDevAPIKey("tskey-api-real-key") {
		t.Fatal("real-looking key must not match dev key")
	}
}

func mustParsePolicy(t *testing.T) policy.Policy {
	t.Helper()
	p, err := policy.Parse(Policy())
	if err != nil {
		t.Fatal(err)
	}
	return p
}
