package policy

import (
	"fmt"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
)

func TestEffectiveAccessEdgeCacheIsBounded(t *testing.T) {
	var cache Cache
	raw := `{"acls":[{"action":"accept","src":["*"],"dst":["*:*"]}]}`
	for i := range maxEdgeCacheEntries * 3 {
		// Every device-state change (here: a different last-seen time) is a new cache key.
		devices := []api.Device{{ID: "a", Name: "a", LastSeen: fmt.Sprintf("2026-10-22T00:00:%02dZ", i%60), Online: i%2 == 0}}
		if _, err := cache.EffectiveAccessEdges(raw, devices, EdgeOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if len(cache.edges) > maxEdgeCacheEntries {
		t.Fatalf("edge cache holds %d entries, want at most %d", len(cache.edges), maxEdgeCacheEntries)
	}
}
