package policy

import (
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
)

func TestCacheReusesPolicyDerivationsAndInvalidatesOnPolicyChange(t *testing.T) {
	var cache Cache
	raw := `{"acls":[{"action":"accept","src":["*"],"dst":["*:*"]}]}`
	devices := []api.Device{{ID: "one", IP: "100.64.0.1"}, {ID: "two", IP: "100.64.0.2"}}

	if _, err := cache.EffectiveAccessEdges(raw, devices, EdgeOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := len(cache.edges); got != 1 {
		t.Fatalf("edge cache entries = %d, want 1", got)
	}
	if _, err := cache.EffectiveAccessEdges(raw, devices, EdgeOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := len(cache.edges); got != 1 {
		t.Fatalf("repeated edge cache entries = %d, want 1", got)
	}

	changedDevices := append([]api.Device(nil), devices...)
	changedDevices[1].Online = true
	if _, err := cache.EffectiveAccessEdges(raw, changedDevices, EdgeOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := len(cache.edges); got != 2 {
		t.Fatalf("device-sensitive edge cache entries = %d, want 2", got)
	}

	if _, err := cache.StructuredMap(`{"groups":{"group:eng":["a@example.com"]}}`); err != nil {
		t.Fatal(err)
	}
	if got := len(cache.edges); got != 0 {
		t.Fatalf("edge cache entries after policy change = %d, want 0", got)
	}
}
