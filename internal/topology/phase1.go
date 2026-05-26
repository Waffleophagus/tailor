package topology

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Waffleophagus/tailor/internal/api"
)

func Phase1Edges(devices []api.Device) []api.Edge {
	var edges []api.Edge
	edges = append(edges, relationshipEdges(devices, api.EdgeKindOwner, ownerKey)...)
	edges = append(edges, relationshipEdges(devices, api.EdgeKindTag, tagKeys)...)
	edges = append(edges, relationshipEdges(devices, api.EdgeKindSubnet, subnetKeys)...)
	sort.Slice(edges, func(i, j int) bool {
		return edges[i].ID < edges[j].ID
	})
	return edges
}

func relationshipEdges(devices []api.Device, kind api.EdgeKind, keysFor func(api.Device) []string) []api.Edge {
	groups := map[string][]api.Device{}
	for _, device := range devices {
		for _, key := range keysFor(device) {
			if key == "" {
				continue
			}
			groups[key] = append(groups[key], device)
		}
	}

	var edges []api.Edge
	seen := map[string]bool{}
	for key, members := range groups {
		if len(members) < 2 {
			continue
		}
		sort.Slice(members, func(i, j int) bool {
			return members[i].ID < members[j].ID
		})
		for i := 0; i < len(members)-1; i++ {
			from := members[i].ID
			to := members[i+1].ID
			id := edgeID(kind, key, from, to)
			if seen[id] {
				continue
			}
			seen[id] = true
			edges = append(edges, api.Edge{
				ID:     id,
				From:   from,
				To:     to,
				Kind:   kind,
				Labels: []string{key},
			})
		}
	}
	return edges
}

func ownerKey(device api.Device) []string {
	if device.Owner == "" {
		return nil
	}
	return []string{device.Owner}
}

func tagKeys(device api.Device) []string {
	return device.Tags
}

func subnetKeys(device api.Device) []string {
	return device.RoutedSubnets
}

func edgeID(kind api.EdgeKind, label, from, to string) string {
	return fmt.Sprintf("%s:%s:%s:%s", kind, sanitize(label), sanitize(from), sanitize(to))
}

func sanitize(value string) string {
	value = strings.ToLower(value)
	value = strings.NewReplacer(" ", "-", "/", "_", ":", "_", "@", "_").Replace(value)
	return value
}
