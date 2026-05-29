//go:build dev

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
	if len(edges) < 12 {
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

func TestSuperUserBroadAccess(t *testing.T) {
	edges := policy.ResolveEffectiveAccess(mustParsePolicy(t), Devices(), policy.EdgeOptions{
		Perspective: SuperUserEmail,
	})
	if len(edges) == 0 {
		t.Fatal("super user should produce policy edges")
	}

	reachable := map[string]bool{}
	broad := false
	for _, edge := range edges {
		reachable[edge.To] = true
		if edge.AccessScope == api.AccessScopeBroad {
			broad = true
		}
	}
	if !broad {
		t.Fatal("super user *:* rule should classify as broad access")
	}

	deviceCount := len(Devices())
	if len(reachable) < deviceCount-2 {
		t.Fatalf("super user should reach most tailnet devices, got %d/%d reachable: %#v", len(reachable), deviceCount, reachable)
	}

	aliceEdges := policy.ResolveEffectiveAccess(mustParsePolicy(t), Devices(), policy.EdgeOptions{
		Perspective: "alice@demo.tailor.ts.net",
	})
	if len(edges) <= len(aliceEdges) {
		t.Fatalf("super user should have more edges than alice (%d vs %d)", len(edges), len(aliceEdges))
	}

	visible := policy.VisibleDeviceIDs(mustParsePolicy(t), Devices(), SuperUserEmail)
	aliceVisible := policy.VisibleDeviceIDs(mustParsePolicy(t), Devices(), "alice@demo.tailor.ts.net")
	if len(visible) < deviceCount-1 {
		t.Fatalf("super user netmap should include nearly all devices, got %d/%d", len(visible), deviceCount)
	}
	if len(visible) <= len(aliceVisible) {
		t.Fatalf("super user should see more devices than alice (%d vs %d)", len(visible), len(aliceVisible))
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
	if !Enabled {
		t.Fatal("dev build must set Enabled")
	}
	if !IsDevAPIKey(APIKey) {
		t.Fatal("APIKey must match IsDevAPIKey")
	}
	if IsDevAPIKey("tskey-api-real-key") {
		t.Fatal("real-looking key must not match dev key")
	}
}

func TestSpawnDevicesAppendsToStore(t *testing.T) {
	ResetStore()
	before := len(Devices())
	spawned, err := SpawnDevices(api.DevSpawnDevicesRequest{Count: 4, Prefix: "burst"})
	if err != nil {
		t.Fatal(err)
	}
	if len(spawned) != 4 {
		t.Fatalf("spawned = %d, want 4", len(spawned))
	}
	after := Devices()
	if len(after) != before+4 {
		t.Fatalf("device count = %d, want %d", len(after), before+4)
	}
}

func TestSpawnDevicesUsesExplicitNames(t *testing.T) {
	ResetStore()
	spawned, err := SpawnDevices(api.DevSpawnDevicesRequest{
		Names: []string{"compliance-archive-primary", "audit-trail-ingest"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(spawned) != 2 {
		t.Fatalf("spawned = %d, want 2", len(spawned))
	}
	if spawned[0].Name != "compliance-archive-primary" || spawned[1].Name != "audit-trail-ingest" {
		t.Fatalf("unexpected names: %#v", spawned)
	}
}

func TestSpawnDevicesUsesSpecs(t *testing.T) {
	ResetStore()
	spawned, err := SpawnDevices(api.DevSpawnDevicesRequest{
		Specs: []api.DevSpawnDeviceSpec{
			{
				Name:          "k8s-prod-worker-99",
				Owner:         "platform-ops@demo.tailor.ts.net",
				Tags:          []string{"tag:k8s-prod"},
				SubnetRouter:  true,
				RoutedSubnets: []string{"10.30.99.0/24"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(spawned) != 1 {
		t.Fatalf("spawned = %d, want 1", len(spawned))
	}
	if !spawned[0].SubnetRouter || len(spawned[0].RoutedSubnets) != 1 {
		t.Fatalf("expected subnet router, got %#v", spawned[0])
	}
}

func TestPatchDevicesTogglesOnline(t *testing.T) {
	ResetStore()
	spawned, err := SpawnDevices(api.DevSpawnDevicesRequest{
		Specs: []api.DevSpawnDeviceSpec{
			{Name: "provision-test", Online: boolPtr(false)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if spawned[0].Online {
		t.Fatal("expected offline spawn")
	}

	patched, err := PatchDevices(api.DevPatchDevicesRequest{
		Devices: []api.DevPatchDeviceSpec{{Name: "provision-test", Online: boolPtr(true)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !patched[0].Online || patched[0].LastSeen != "" {
		t.Fatalf("expected online with cleared lastSeen, got %#v", patched[0])
	}
}

func boolPtr(v bool) *bool { return &v }

func TestAuditTrailIngestConnectivity(t *testing.T) {
	ResetStore()
	_, err := SpawnDevices(api.DevSpawnDevicesRequest{
		Specs: []api.DevSpawnDeviceSpec{{
			Name:  "audit-trail-ingest",
			Owner: "platform-ops@demo.tailor.ts.net",
			Tags:  []string{"tag:prod", "tag:platform"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	devices := Devices()
	var targetID string
	for _, d := range devices {
		if d.Name == "audit-trail-ingest" {
			targetID = d.ID
			break
		}
	}
	if targetID == "" {
		t.Fatal("audit-trail-ingest not found")
	}

	edges, err := policy.EffectiveAccessEdges(Policy(), devices, policy.EdgeOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var incoming, outgoing int
	incomingPeers := map[string]int{}
	outgoingPeers := map[string]int{}
	nameByID := map[string]string{}
	for _, d := range devices {
		nameByID[d.ID] = d.Name
	}
	for _, edge := range edges {
		if edge.To == targetID {
			incoming++
			incomingPeers[nameByID[edge.From]]++
		}
		if edge.From == targetID {
			outgoing++
			outgoingPeers[nameByID[edge.To]]++
		}
	}
	t.Logf("audit-trail-ingest id=%s incoming=%d outgoing=%d", targetID, incoming, outgoing)
	if incoming+outgoing == 0 {
		t.Fatal("expected policy edges involving audit-trail-ingest")
	}
	if incoming < 5 {
		t.Logf("warning: only %d incoming edges (expected many from group:platform → tag:prod/platform)", incoming)
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
