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

func TestValidateTailscaleConstraintsRejectsSSHCheckWithTagSource(t *testing.T) {
	raw := `{
		"ssh": [
			{
				"action": "check",
				"src": ["tag:ci", "group:eng"],
				"dst": ["tag:prod"],
				"users": ["autogroup:nonroot"]
			}
		]
	}`

	err := ValidateTailscaleConstraints(raw)
	if err == nil {
		t.Fatal("expected invalid policy error")
	}
	if !strings.Contains(err.Error(), `ssh[0] uses action "check" with tagged source "tag:ci"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTailscaleConstraintsAllowsSSHAcceptWithTagSource(t *testing.T) {
	raw := `{
		"ssh": [
			{
				"action": "accept",
				"src": ["tag:ci"],
				"dst": ["tag:prod"],
				"users": ["root"]
			}
		]
	}`

	if err := ValidateTailscaleConstraints(raw); err != nil {
		t.Fatalf("expected accept rule to be valid: %v", err)
	}
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

func TestDevicesForPerspectiveMemberIncludesTaggedSources(t *testing.T) {
	devices := []api.Device{
		{ID: "untagged", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "tagged", Owner: "alice@example.com", Tags: []string{"tag:server"}, TailscaleIPs: []string{"100.64.0.2"}},
	}
	matched := devicesForPerspective("autogroup:member", Policy{}, devices)
	if len(matched) != 2 {
		t.Fatalf("member perspective sources = %#v, want tagged and untagged devices", matched)
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

func TestEffectiveAccessEdgesMemberPerspectiveIncludesTaggedSource(t *testing.T) {
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
	assertEdge(t, edges, "server", "server", api.AccessScopeHTTP, []string{"443"})
}

func TestEffectiveAccessEdgesNormalizeUserAndHostSelectors(t *testing.T) {
	p := Policy{
		Hosts: map[string]string{"db": "100.64.0.10"},
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"user:alice@example.com"}, Dst: []string{"host:db:5432"}},
		},
	}
	devices := []api.Device{
		{ID: "alice", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "db", Owner: "ops@example.com", TailscaleIPs: []string{"100.64.0.10"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "alice", "db", api.AccessScopeLimited, []string{"5432"})
}

func TestEffectiveAccessEdgesRoleAutogroupUsesDeviceRoles(t *testing.T) {
	p := Policy{
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"autogroup:admin"}, Dst: []string{"tag:prod:443"}},
		},
	}
	devices := []api.Device{
		{ID: "admin", Owner: "admin@example.com", Roles: []string{"admin"}, TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "member", Owner: "member@example.com", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "prod", Owner: "ops@example.com", Tags: []string{"tag:prod"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "admin", "prod", api.AccessScopeHTTP, []string{"443"})
	for _, edge := range edges {
		if edge.From == "member" {
			t.Fatalf("member without admin role should not match autogroup:admin: %#v", edge)
		}
	}
}

func TestEffectiveAccessEdgesRoleAutogroupWithoutMetadataDoesNotMatchMembers(t *testing.T) {
	p := Policy{
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"autogroup:network-admin"}, Dst: []string{"tag:prod:443"}},
		},
	}
	devices := []api.Device{
		{ID: "member", Owner: "member@example.com", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "prod", Owner: "ops@example.com", Tags: []string{"tag:prod"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	if len(edges) != 0 {
		t.Fatalf("role autogroup without role metadata should not match all members: %#v", edges)
	}
}

func TestEffectiveAccessEdgesRoleAutogroupDoesNotMatchTaggedDevices(t *testing.T) {
	p := Policy{
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"autogroup:admin"}, Dst: []string{"tag:prod:443"}},
		},
	}
	devices := []api.Device{
		{ID: "tagged-admin-host", Owner: "admin@example.com", Roles: []string{"admin"}, Tags: []string{"tag:ci"}, TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "prod", Owner: "ops@example.com", Tags: []string{"tag:prod"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	if len(edges) != 0 {
		t.Fatalf("tagged device should not match role autogroup: %#v", edges)
	}
}

func TestEffectiveAccessEdgesSharedAutogroupUsesSharedMetadata(t *testing.T) {
	p := Policy{
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"autogroup:shared"}, Dst: []string{"tag:prod:443"}},
		},
	}
	devices := []api.Device{
		{ID: "shared", Owner: "external@example.com", Shared: true, TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "member", Owner: "member@example.com", TailscaleIPs: []string{"100.64.0.4"}},
		{ID: "prod", Owner: "ops@example.com", Tags: []string{"tag:prod"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "shared", "prod", api.AccessScopeHTTP, []string{"443"})
	for _, edge := range edges {
		if edge.From == "member" {
			t.Fatalf("non-shared member should not match autogroup:shared: %#v", edge)
		}
	}
}

func TestEffectiveAccessEdgesServiceSelectorUsesServiceNode(t *testing.T) {
	p := Policy{
		Grants: []Grant{
			{Src: []string{"autogroup:member"}, Dst: []string{"svc:web"}, IP: []string{"tcp:443"}},
		},
	}
	devices := []api.Device{
		{ID: "alice", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "svc:web", Kind: "service", Name: "svc:web", TailscaleIPs: []string{"100.100.0.1"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "alice", "svc:web", api.AccessScopeHTTP, []string{"443"})
}

func TestEffectiveAccessEdgesTaggedServiceDoesNotBecomeSource(t *testing.T) {
	p := Policy{
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"tag:web"}, Dst: []string{"tag:prod:443"}},
		},
	}
	devices := []api.Device{
		{ID: "svc:web", Kind: "service", Name: "svc:web", Tags: []string{"tag:web"}, TailscaleIPs: []string{"100.100.0.1"}},
		{ID: "prod", Owner: "ops@example.com", Tags: []string{"tag:prod"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	if len(edges) != 0 {
		t.Fatalf("service node should not resolve as an access source: %#v", edges)
	}
}

func TestEffectiveAccessEdgesSSHRulesMaterializeSSHAccess(t *testing.T) {
	p := Policy{
		SSH: []SSHRule{
			{Action: "check", Src: []string{"alice@example.com"}, Dst: []string{"tag:server"}, Users: []string{"autogroup:nonroot"}},
		},
	}
	devices := []api.Device{
		{ID: "alice", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "server", Tags: []string{"tag:server"}, TailscaleIPs: []string{"100.64.0.2"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "alice", "server", api.AccessScopeSSH, []string{"22"})
	if len(edges[0].PolicyRefs) != 1 || edges[0].PolicyRefs[0].Section != "ssh" {
		t.Fatalf("ssh policy refs = %#v", edges[0].PolicyRefs)
	}
}

func TestEffectiveAccessEdgesSSHSrcPostureFiltersSources(t *testing.T) {
	p := Policy{
		Postures: map[string][]string{"posture:new": {"node:tsVersion >= '1.90.0'"}},
		SSH: []SSHRule{
			{Action: "accept", Src: []string{"autogroup:member"}, Dst: []string{"tag:server"}, Users: []string{"root"}, SrcPosture: []string{"posture:new"}},
		},
	}
	devices := []api.Device{
		{ID: "new", Owner: "alice@example.com", PostureAttrs: map[string]any{"node:tsVersion": "1.94.2"}, TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "old", Owner: "bob@example.com", PostureAttrs: map[string]any{"node:tsVersion": "1.80.0"}, TailscaleIPs: []string{"100.64.0.3"}},
		{ID: "server", Tags: []string{"tag:server"}, TailscaleIPs: []string{"100.64.0.2"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "new", "server", api.AccessScopeSSH, []string{"22"})
	for _, edge := range edges {
		if edge.From == "old" {
			t.Fatalf("old source should not satisfy SSH posture: %#v", edge)
		}
	}
}

func TestEffectiveAccessEdgesSSHDoesNotTargetServices(t *testing.T) {
	p := Policy{
		SSH: []SSHRule{
			{Action: "accept", Src: []string{"alice@example.com"}, Dst: []string{"svc:web"}, Users: []string{"root"}},
		},
	}
	devices := []api.Device{
		{ID: "alice", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "svc:web", Kind: "service", TailscaleIPs: []string{"100.100.0.1"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	if len(edges) != 0 {
		t.Fatalf("ssh should not target service nodes: %#v", edges)
	}
}

func TestEffectiveAccessEdgesPostureFiltersSources(t *testing.T) {
	p := Policy{
		Postures: map[string][]string{
			"posture:mac": {"node:os == 'macos'"},
		},
		Grants: []Grant{
			{Src: []string{"autogroup:member"}, Dst: []string{"tag:prod"}, IP: []string{"tcp:443"}, SrcPosture: []string{"posture:mac"}},
		},
	}
	devices := []api.Device{
		{ID: "mac", Owner: "alice@example.com", OS: "macOS", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "linux", Owner: "bob@example.com", OS: "linux", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "prod", Owner: "ops@example.com", Tags: []string{"tag:prod"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "mac", "prod", api.AccessScopeHTTP, []string{"443"})
	for _, edge := range edges {
		if edge.From == "linux" {
			t.Fatalf("linux source should not satisfy mac posture: %#v", edge)
		}
	}
}

func TestEffectiveAccessEdgesPostureVersionComparisonsAndOr(t *testing.T) {
	p := Policy{
		Postures: map[string][]string{
			"posture:new":   {"node:tsVersion >= '1.90.0'", "node:osVersion > '14.3'"},
			"posture:linux": {"node:os == 'linux'"},
		},
		Grants: []Grant{
			{Src: []string{"autogroup:member"}, Dst: []string{"tag:prod"}, IP: []string{"tcp:443"}, SrcPosture: []string{"posture:new", "posture:linux"}},
		},
	}
	devices := []api.Device{
		{ID: "new-mac", Owner: "alice@example.com", PostureAttrs: map[string]any{"node:tsVersion": "1.94.2", "node:osVersion": "14.4"}, TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "linux", Owner: "bob@example.com", OS: "linux", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "old-mac", Owner: "carol@example.com", PostureAttrs: map[string]any{"node:tsVersion": "1.80.0", "node:osVersion": "14.4"}, TailscaleIPs: []string{"100.64.0.4"}},
		{ID: "prod", Owner: "ops@example.com", Tags: []string{"tag:prod"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "new-mac", "prod", api.AccessScopeHTTP, []string{"443"})
	assertEdge(t, edges, "linux", "prod", api.AccessScopeHTTP, []string{"443"})
	for _, edge := range edges {
		if edge.From == "old-mac" {
			t.Fatalf("old mac should not satisfy version posture: %#v", edge)
		}
	}
}

func TestEffectiveAccessEdgesPostureTypedOperators(t *testing.T) {
	p := Policy{
		Postures: map[string][]string{
			"posture:trusted": {
				"node:os IN ['macos', 'linux']",
				"custom:encrypted == true",
				"custom:score >= 80",
				"custom:tier != 'dev'",
				"custom:region NOT IN ['CN', 'RU']",
			},
		},
		Grants: []Grant{
			{Src: []string{"autogroup:member"}, Dst: []string{"tag:prod"}, IP: []string{"tcp:443"}, SrcPosture: []string{"posture:trusted"}},
		},
	}
	devices := []api.Device{
		{ID: "trusted", Owner: "alice@example.com", PostureAttrs: map[string]any{
			"node:os":          "macos",
			"custom:encrypted": true,
			"custom:score":     90.0,
			"custom:tier":      "prod",
			"custom:region":    "US",
		}, TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "wrong-types", Owner: "bob@example.com", PostureAttrs: map[string]any{
			"node:os":          "macos",
			"custom:encrypted": "true",
			"custom:score":     "90",
			"custom:tier":      "prod",
			"custom:region":    "US",
		}, TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "prod", Owner: "ops@example.com", Tags: []string{"tag:prod"}, TailscaleIPs: []string{"100.64.0.3"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "trusted", "prod", api.AccessScopeHTTP, []string{"443"})
	for _, edge := range edges {
		if edge.From == "wrong-types" {
			t.Fatalf("string values should not satisfy bool/number posture assertions: %#v", edge)
		}
	}
}

func TestEffectiveAccessEdgesPostureUnsetNeverMatches(t *testing.T) {
	for _, assertion := range []string{
		"custom:tier != 'prod'",
		"custom:tier NOT IN ['prod']",
		"custom:tier NOT SET",
	} {
		p := Policy{
			Postures: map[string][]string{"posture:test": {assertion}},
			Grants: []Grant{
				{Src: []string{"autogroup:member"}, Dst: []string{"tag:prod"}, IP: []string{"tcp:443"}, SrcPosture: []string{"posture:test"}},
			},
		}
		devices := []api.Device{
			{ID: "unset", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
			{ID: "prod", Owner: "ops@example.com", Tags: []string{"tag:prod"}, TailscaleIPs: []string{"100.64.0.3"}},
		}

		edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
		if len(edges) != 0 {
			t.Fatalf("%s should not match unset attrs: %#v", assertion, edges)
		}
	}
}

func TestEvaluateDraftReportsParsedButUnsupportedSections(t *testing.T) {
	raw := `{
		"acls": [],
		"nodeAttrs": [{"target": ["autogroup:member"], "attr": ["funnel"]}],
		"autoApprovers": {"routes": {"10.0.0.0/8": ["autogroup:admin"]}},
		"tests": [{"src": "alice@example.com", "accept": ["tag:prod:443"]}],
		"sshTests": [{"src": "alice@example.com", "dst": ["tag:prod"], "accept": ["root"]}]
	}`

	got, err := EvaluateDraft(raw, raw, nil, EdgeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"autoApprovers", "nodeAttrs", "sshTests", "tests"}
	if strings.Join(got.UnsupportedSections, ",") != strings.Join(want, ",") {
		t.Fatalf("unsupported sections = %#v, want %#v", got.UnsupportedSections, want)
	}
}

func TestEffectiveAccessEdgesGrantViaFiltersRouters(t *testing.T) {
	p := Policy{
		Grants: []Grant{
			{Src: []string{"alice@example.com"}, Dst: []string{"10.10.0.0/24"}, IP: []string{"tcp:443"}, Via: []string{"tag:router-a"}},
		},
	}
	devices := []api.Device{
		{ID: "alice", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "router-a", Owner: "ops@example.com", Tags: []string{"tag:router-a"}, TailscaleIPs: []string{"100.64.0.2"}, RoutedSubnets: []string{"10.10.0.0/24"}},
		{ID: "router-b", Owner: "ops@example.com", Tags: []string{"tag:router-b"}, TailscaleIPs: []string{"100.64.0.3"}, RoutedSubnets: []string{"10.10.0.0/24"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "alice", "router-a", api.AccessScopeHTTP, []string{"443"})
	for _, edge := range edges {
		if edge.To == "router-b" {
			t.Fatalf("via should exclude router-b: %#v", edge)
		}
	}
}

func TestEffectiveAccessEdgesIPRangeMatchesRoutes(t *testing.T) {
	p := Policy{
		ACLs: []ACLRule{
			{Action: "accept", Src: []string{"alice@example.com"}, Dst: []string{"10.10.0.10-10.10.0.20:443"}},
		},
	}
	devices := []api.Device{
		{ID: "alice", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "router", Owner: "ops@example.com", TailscaleIPs: []string{"100.64.0.2"}, RoutedSubnets: []string{"10.10.0.0/24"}},
	}

	edges := ResolveEffectiveAccess(p, devices, EdgeOptions{})
	assertEdge(t, edges, "alice", "router", api.AccessScopeHTTP, []string{"443"})
}

func TestValidateTailscaleConstraintsRejectsInvalidPortsAndVia(t *testing.T) {
	raw := `{
		"acls": [
			{"action": "accept", "src": ["alice@example.com"], "dst": ["tag:web:70000"]}
		],
		"grants": [
			{"src": ["alice@example.com"], "dst": ["tag:web"], "ip": ["tcp:443"], "via": ["group:ops"]}
		]
	}`

	err := ValidateTailscaleConstraints(raw)
	if err == nil {
		t.Fatal("expected invalid policy error")
	}
	if !strings.Contains(err.Error(), "invalid ports") {
		t.Fatalf("unexpected error: %v", err)
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
