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

func TestEffectiveAccessEdgesPerspectiveLimitsSourcesToSubject(t *testing.T) {
	p := Policy{
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"autogroup:member"}, Dst: []string{"tag:web:443"}},
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
		t.Fatalf("got %d edges, want 1 (alice only): %#v", len(edges), edges)
	}
	assertEdge(t, edges, "alice", "web", api.AccessScopeHTTP, []string{"443"})
}

func TestEffectiveAccessEdgesAutogroupSelfOwnDevices(t *testing.T) {
	p := Policy{
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"autogroup:member"}, Dst: []string{"autogroup:self:22"}},
		},
	}
	devices := []api.Device{
		{ID: "alice-laptop", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "alice-phone", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "bob-laptop", Owner: "bob@example.com", TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{Perspective: "alice@example.com"})
	assertEdge(t, edges, "alice-laptop", "alice-phone", api.AccessScopeSSH, []string{"22"})
	assertEdge(t, edges, "alice-phone", "alice-laptop", api.AccessScopeSSH, []string{"22"})
	for _, edge := range edges {
		if edge.From == "bob-laptop" || edge.To == "bob-laptop" {
			t.Fatalf("bob should not appear in alice perspective: %#v", edge)
		}
	}
}

func TestEffectiveAccessEdgesTagPerspective(t *testing.T) {
	p := Policy{
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"tag:ci"}, Dst: []string{"tag:web:443"}},
		},
	}
	devices := []api.Device{
		{ID: "ci-runner", Owner: "ops@example.com", Tags: []string{"tag:ci"}, TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "ci-runner-2", Owner: "ops@example.com", Tags: []string{"tag:ci"}, TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "web", Owner: "ops@example.com", Tags: []string{"tag:web"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{Perspective: "tag:ci"})
	if len(edges) != 2 {
		t.Fatalf("got %d edges, want 2: %#v", len(edges), edges)
	}
	assertEdge(t, edges, "ci-runner", "web", api.AccessScopeHTTP, []string{"443"})
	assertEdge(t, edges, "ci-runner-2", "web", api.AccessScopeHTTP, []string{"443"})
}

func TestEffectiveAccessEdgesGroupPerspective(t *testing.T) {
	p := Policy{
		Groups: map[string][]string{"group:eng": {"alice@example.com", "bob@example.com"}},
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"group:eng"}, Dst: []string{"tag:web:443"}},
		},
	}
	devices := []api.Device{
		{ID: "alice", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "bob", Owner: "bob@example.com", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "web", Owner: "ops@example.com", Tags: []string{"tag:web"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{Perspective: "group:eng"})
	if len(edges) != 2 {
		t.Fatalf("got %d edges, want 2: %#v", len(edges), edges)
	}
	assertEdge(t, edges, "alice", "web", api.AccessScopeHTTP, []string{"443"})
	assertEdge(t, edges, "bob", "web", api.AccessScopeHTTP, []string{"443"})
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

func TestDevicesForPerspectiveMemberExcludesTaggedSources(t *testing.T) {
	devices := []api.Device{
		{ID: "untagged", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "tagged", Owner: "alice@example.com", Tags: []string{"tag:server"}, TailscaleIPs: []string{"100.64.0.2"}},
	}
	matched := devicesForPerspective("autogroup:member", Policy{}, devices)
	if len(matched) != 1 || matched[0].ID != "untagged" {
		t.Fatalf("member perspective sources = %#v, want untagged only", matched)
	}
}

func TestDevicesForPerspectiveMemberTaggedUnion(t *testing.T) {
	devices := []api.Device{
		{ID: "untagged", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "tagged", Owner: "bob@example.com", Tags: []string{"tag:server"}, TailscaleIPs: []string{"100.64.0.2"}},
	}
	matched := devicesForPerspective("cohort:member+tagged", Policy{}, devices)
	if len(matched) != 2 {
		t.Fatalf("union perspective sources = %#v, want both devices", matched)
	}
}

func TestEffectiveAccessEdgesMemberPerspectiveTaggedDeviceIsDestinationOnly(t *testing.T) {
	p := Policy{
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"autogroup:member"}, Dst: []string{"tag:server:443"}},
		},
	}
	devices := []api.Device{
		{ID: "member", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "server", Owner: "alice@example.com", Tags: []string{"tag:server"}, TailscaleIPs: []string{"100.64.0.2"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{Perspective: "autogroup:member"})
	assertEdge(t, edges, "member", "server", api.AccessScopeHTTP, []string{"443"})
	for _, edge := range edges {
		if edge.From == "server" {
			t.Fatalf("tagged device should not be a source under member perspective: %#v", edge)
		}
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

func TestVisibleDeviceIDsTrimsNetmapByEffectiveAccess(t *testing.T) {
	p, err := Parse(`{
		"groups": {
			"group:eng": ["alice@example.com"]
		},
		"acls": [
			{"action": "accept", "src": ["alice@example.com"], "dst": ["tag:web:443"]},
			{"action": "accept", "src": ["bob@example.com"], "dst": ["tag:db:22"]}
		]
	}`)
	if err != nil {
		t.Fatal(err)
	}
	devices := []api.Device{
		{ID: "alice-laptop", Owner: "alice@example.com"},
		{ID: "alice-phone", Owner: "alice@example.com"},
		{ID: "bob-laptop", Owner: "bob@example.com"},
		{ID: "web", Owner: "ops@example.com", Tags: []string{"tag:web"}},
		{ID: "db", Owner: "ops@example.com", Tags: []string{"tag:db"}},
		{ID: "secret", Owner: "ops@example.com", Tags: []string{"tag:secret"}},
	}

	aliceVisible := VisibleDeviceIDs(p, devices, "alice@example.com")
	aliceSet := map[string]bool{}
	for _, id := range aliceVisible {
		aliceSet[id] = true
	}
	if !aliceSet["alice-laptop"] || !aliceSet["alice-phone"] || !aliceSet["web"] {
		t.Fatalf("alice netmap = %#v, want alice devices and web", aliceVisible)
	}
	if aliceSet["secret"] {
		t.Fatalf("alice should not see unrelated secret host: %#v", aliceVisible)
	}

	allVisible := VisibleDeviceIDs(p, devices, "")
	if len(allVisible) != len(devices) {
		t.Fatalf("empty perspective should show all devices, got %d want %d", len(allVisible), len(devices))
	}
}

func TestEvaluateDraftComparesAccessAndReportsRisk(t *testing.T) {
	saved := `{
		"acls": [
			{"action": "accept", "src": ["alice@example.com"], "dst": ["tag:web:443"]},
			{"action": "accept", "src": ["bob@example.com"], "dst": ["tag:web:22"]},
		],
	}`
	draft := `{
		"acls": [
			{"action": "accept", "src": ["alice@example.com"], "dst": ["tag:web:22"]},
			{"action": "accept", "src": ["tag:web"], "dst": ["*:0-65535"]},
			{"action": "accept", "src": ["group:missing"], "dst": ["tag:web:443"]},
		],
		"grants": [
			{"src": ["alice@example.com"], "dst": ["tag:db"], "ip": ["tcp:5432"]},
			{"src": ["alice@example.com"], "dst": ["tag:web"], "app": {"tailscale.com/cap/file-sharing": [{"shares": ["eng"]}]}},
		],
		"customThing": true,
	}`
	devices := []api.Device{
		{ID: "alice", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "bob", Owner: "bob@example.com", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "web", Owner: "ops@example.com", Tags: []string{"tag:web"}, TailscaleIPs: []string{"100.64.0.3"}},
		{ID: "db", Owner: "ops@example.com", Tags: []string{"tag:db"}, TailscaleIPs: []string{"100.64.0.4"}},
	}

	evaluation, err := EvaluateDraft(saved, draft, devices, EdgeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(evaluation.Changed) != 1 {
		t.Fatalf("changed = %#v, want one access-scope change", evaluation.Changed)
	}
	if evaluation.Changed[0].Saved == nil || evaluation.Changed[0].Saved.AccessScope != api.AccessScopeHTTP {
		t.Fatalf("changed saved edge = %#v, want HTTP saved edge", evaluation.Changed[0].Saved)
	}
	if evaluation.Changed[0].Draft == nil || evaluation.Changed[0].Draft.AccessScope != api.AccessScopeSSH {
		t.Fatalf("changed draft edge = %#v, want SSH draft edge", evaluation.Changed[0].Draft)
	}
	assertChangeEdge(t, evaluation.Added, "alice", "db", api.AccessScopeLimited)
	assertChangeEdge(t, evaluation.Removed, "bob", "web", api.AccessScopeSSH)
	assertChangeEdge(t, evaluation.Added, "web", "alice", api.AccessScopeBroad)
	if len(evaluation.BroadAccess) == 0 {
		t.Fatal("expected broad access risk")
	}
	if len(evaluation.UnresolvedSelectors) != 1 || evaluation.UnresolvedSelectors[0].Selector != "group:missing" {
		t.Fatalf("unresolved selectors = %#v, want group:missing", evaluation.UnresolvedSelectors)
	}
	if len(evaluation.UnsupportedSections) != 1 || evaluation.UnsupportedSections[0] != "customThing" {
		t.Fatalf("unsupported sections = %#v, want customThing", evaluation.UnsupportedSections)
	}
	if len(evaluation.ApplicationGrants) != 1 {
		t.Fatalf("application grants = %#v, want one app-layer grant", evaluation.ApplicationGrants)
	}
	if evaluation.ApplicationGrants[0].Capabilities[0] != "tailscale.com/cap/file-sharing" {
		t.Fatalf("application grant capabilities = %#v", evaluation.ApplicationGrants[0].Capabilities)
	}
	if evaluation.UnresolvedSelectors == nil || evaluation.UnsupportedSections == nil {
		t.Fatal("evaluate-draft should return empty slices, not nil")
	}
}

func TestStructuredMapSurfacesRecognizedAndUnknownSections(t *testing.T) {
	raw := `{
		"groups": {
			"group:eng": ["alice@example.com"],
		},
		"tagOwners": {
			"tag:web": ["group:eng"],
		},
		"hosts": {
			"db": "100.64.0.10",
		},
		"acls": [
			{"action": "accept", "src": ["group:eng"], "dst": ["tag:web:443"]},
		],
		"customThing": {
			"kept": true,
		},
	}`

	policyMap, err := StructuredMap(raw)
	if err != nil {
		t.Fatal(err)
	}
	if policyMap.ParseError != "" {
		t.Fatalf("unexpected parse error: %s", policyMap.ParseError)
	}

	acls := findSection(policyMap.Sections, "acls")
	if acls == nil || !acls.Supported || acls.Count != 1 {
		t.Fatalf("bad acls section: %#v", acls)
	}
	if len(acls.Entries) != 1 || acls.Entries[0].Summary != "group:eng -> tag:web:443" {
		t.Fatalf("bad acl entries: %#v", acls.Entries)
	}

	unknown := findSection(policyMap.Sections, "customThing")
	if unknown == nil || unknown.Supported || unknown.Count != 1 || unknown.Raw == nil {
		t.Fatalf("unknown section not preserved: %#v", unknown)
	}
}

func TestStructuredMapReturnsParseErrorWithRawPolicy(t *testing.T) {
	raw := `{"acls": [`
	policyMap, err := StructuredMap(raw)
	if err != nil {
		t.Fatal(err)
	}
	if policyMap.HuJSON != raw {
		t.Fatalf("raw policy not preserved: %q", policyMap.HuJSON)
	}
	if policyMap.ParseError == "" {
		t.Fatal("expected parse error")
	}
	if len(policyMap.Sections) != 0 {
		t.Fatalf("sections = %#v, want none", policyMap.Sections)
	}
}

func findSection(sections []api.PolicySection, name string) *api.PolicySection {
	for i := range sections {
		if sections[i].Name == name {
			return &sections[i]
		}
	}
	return nil
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

func assertChangeEdge(t *testing.T, changes []api.PolicyEdgeChange, from, to string, scope api.AccessScope) {
	t.Helper()
	for _, change := range changes {
		if change.Edge.From == from && change.Edge.To == to {
			if change.Edge.AccessScope != scope {
				t.Fatalf("%s -> %s scope = %q, want %q: %#v", from, to, change.Edge.AccessScope, scope, change)
			}
			return
		}
	}
	t.Fatalf("missing change edge %s -> %s in %#v", from, to, changes)
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
