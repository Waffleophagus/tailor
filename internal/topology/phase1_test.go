package topology

import (
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
)

func TestPhase1EdgesInferOwnerTagAndSubnetRelationships(t *testing.T) {
	devices := []api.Device{
		{ID: "a", Owner: "alice@example.com", Tags: []string{"tag:server"}, RoutedSubnets: []string{"10.0.0.0/24"}},
		{ID: "b", Owner: "alice@example.com", Tags: []string{"tag:server"}, RoutedSubnets: []string{"10.0.0.0/24"}},
		{ID: "c", Owner: "bob@example.com", Tags: []string{"tag:client"}},
	}

	edges := Phase1Edges(devices)
	if len(edges) != 3 {
		t.Fatalf("got %d edges, want 3: %#v", len(edges), edges)
	}

	kinds := map[api.EdgeKind]bool{}
	for _, edge := range edges {
		kinds[edge.Kind] = true
		if edge.From != "a" || edge.To != "b" {
			t.Fatalf("edge should connect a to b: %#v", edge)
		}
	}
	for _, kind := range []api.EdgeKind{api.EdgeKindOwner, api.EdgeKindTag, api.EdgeKindSubnet} {
		if !kinds[kind] {
			t.Fatalf("missing %s edge in %#v", kind, edges)
		}
	}
}
