package policy

import (
	"strings"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
)

func TestEffectiveAccessEdgesExpandSelectorsAndClassifyPorts(t *testing.T) {
	raw := `{
		// comments are valid HuJSON and must not break parsing
		"groups": {
			"group:eng": ["alice@example.com"],
		},
		"hosts": {
			"dbhost": "100.64.0.10",
		},
		"acls": [
			{"action": "accept", "src": ["group:eng"], "dst": ["tag:web:443"]},
			{"action": "accept", "src": ["bob@example.com"], "dst": ["dbhost:22"]},
			{"action": "accept", "src": ["autogroup:member"], "dst": ["10.10.0.0/24:80,443"]},
		],
	}`
	devices := []api.Device{
		{ID: "alice-laptop", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "bob-laptop", Owner: "bob@example.com", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "web", Owner: "ops@example.com", Tags: []string{"tag:web"}, TailscaleIPs: []string{"100.64.0.20"}},
		{ID: "db", Owner: "ops@example.com", TailscaleIPs: []string{"100.64.0.10"}},
		{ID: "router", Owner: "ops@example.com", TailscaleIPs: []string{"100.64.0.30"}, RoutedSubnets: []string{"10.10.0.0/24"}},
	}

	edges, err := EffectiveAccessEdges(raw, devices, EdgeOptions{})
	if err != nil {
		t.Fatal(err)
	}

	assertEdge(t, edges, "alice-laptop", "web", api.AccessScopeHTTP, []string{"443"})
	assertEdge(t, edges, "bob-laptop", "db", api.AccessScopeSSH, []string{"22"})
	assertEdge(t, edges, "alice-laptop", "router", api.AccessScopeHTTP, []string{"443", "80"})
	assertEdge(t, edges, "bob-laptop", "router", api.AccessScopeHTTP, []string{"443", "80"})
}

func TestEffectiveAccessEdgesCanFilterByPerspective(t *testing.T) {
	p := Policy{
		Groups: map[string][]string{"group:eng": {"alice@example.com"}},
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"group:eng"}, Dst: []string{"tag:web:443"}},
			{Action: "accept", Src: []string{"bob@example.com"}, Dst: []string{"tag:web:22"}},
		},
	}
	devices := []api.Device{
		{ID: "alice", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "bob", Owner: "bob@example.com", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "web", Owner: "ops@example.com", Tags: []string{"tag:web"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{Perspective: "alice@example.com"})
	if len(edges) != 1 {
		t.Fatalf("got %d edges, want 1: %#v", len(edges), edges)
	}
	assertEdge(t, edges, "alice", "web", api.AccessScopeHTTP, []string{"443"})
	if len(edges[0].Perspectives) != 1 || edges[0].Perspectives[0] != "alice@example.com" {
		t.Fatalf("missing perspective provenance: %#v", edges[0].Perspectives)
	}
}

func TestAppendACLRulePreservesExistingHuJSONAndAppendsRule(t *testing.T) {
	raw := `{
	// keep this comment
	"groups": {
		"group:eng": ["alice@example.com"],
	},
	"acls": [
		{"action": "accept", "src": ["*"], "dst": ["*:*"]},
	],
}`
	draft, err := AppendACLRule(raw, api.ACLDraft{
		Action: "accept",
		Src:    []string{"alice@example.com"},
		Dst:    []string{"tag:web:443"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(draft, "// keep this comment") {
		t.Fatalf("draft did not preserve comment:\n%s", draft)
	}
	if !strings.Contains(draft, `"src":["alice@example.com"]`) {
		t.Fatalf("draft missing appended src:\n%s", draft)
	}
	if _, err := Parse(draft); err != nil {
		t.Fatalf("draft is not parseable: %v\n%s", err, draft)
	}
}

func assertEdge(t *testing.T, edges []api.Edge, from, to string, scope api.AccessScope, ports []string) {
	t.Helper()
	for _, edge := range edges {
		if edge.From == from && edge.To == to {
			if edge.AccessScope != scope {
				t.Fatalf("%s -> %s scope = %q, want %q: %#v", from, to, edge.AccessScope, scope, edge)
			}
			if !equalStrings(edge.Ports, ports) {
				t.Fatalf("%s -> %s ports = %#v, want %#v", from, to, edge.Ports, ports)
			}
			if len(edge.PolicyRefs) == 0 {
				t.Fatalf("%s -> %s missing policy refs", from, to)
			}
			return
		}
	}
	t.Fatalf("missing edge %s -> %s in %#v", from, to, edges)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
