package localapi

import (
	"encoding/json"
	"net/netip"
	"testing"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/types/views"
)

func TestDevicesFromStatusParsesSelfAndPeers(t *testing.T) {
	const raw = `{
		"Self": {
			"ID": "node-self",
			"HostName": "laptop",
			"DNSName": "laptop.tailnet.ts.net.",
			"TailscaleIPs": ["100.64.0.1", "fd7a:115c:a1e0::1"],
			"OS": "linux",
			"UserID": 1,
			"Tags": ["tag:workstation"],
			"Online": true
		},
		"Peer": {
			"node-key": {
				"ID": "node-peer",
				"HostName": "server",
				"DNSName": "server.tailnet.ts.net.",
				"TailscaleIPs": ["100.64.0.2"],
				"OS": "linux",
				"UserID": 2,
				"Tags": ["tag:server", "tag:database"],
				"Online": false,
				"LastSeen": "2026-05-25T12:30:00Z"
			}
		},
		"User": {
			"1": {"ID": 1, "LoginName": "alice@example.com"},
			"2": {"ID": 2, "LoginName": "ops@example.com"}
		}
	}`

	var status Status
	if err := json.Unmarshal([]byte(raw), &status); err != nil {
		t.Fatal(err)
	}

	got := DevicesFromStatus(status)
	want := []api.Device{
		{
			ID:            "node-self",
			Name:          "laptop.tailnet.ts.net",
			IP:            "100.64.0.1",
			TailscaleIPs:  []string{"100.64.0.1", "fd7a:115c:a1e0::1"},
			OS:            "linux",
			Online:        true,
			Owner:         "alice@example.com",
			Tags:          []string{"tag:workstation"},
			RoutedSubnets: []string{},
		},
		{
			ID:            "node-peer",
			Name:          "server.tailnet.ts.net",
			IP:            "100.64.0.2",
			TailscaleIPs:  []string{"100.64.0.2"},
			OS:            "linux",
			Online:        false,
			Owner:         "ops@example.com",
			Tags:          []string{"tag:server", "tag:database"},
			RoutedSubnets: []string{},
			LastSeen:      "2026-05-25T12:30:00Z",
		},
	}

	if len(got) != len(want) {
		t.Fatalf("got %d devices, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		assertDevice(t, got[i], want[i])
	}
}

func TestDevicesFromStatusNormalizesMissingTagsToEmptyArray(t *testing.T) {
	got := DevicesFromStatus(Status{
		Self: &Peer{ID: "node-self", HostName: "laptop"},
	})

	if len(got) != 1 {
		t.Fatalf("got %d devices, want 1", len(got))
	}
	if got[0].Tags == nil {
		t.Fatal("tags should be an empty slice, not nil")
	}
}

func TestDevicesFromStatusParsesSubnetRoutes(t *testing.T) {
	got := DevicesFromStatus(Status{
		Self: &Peer{
			ID:            "node-self",
			HostName:      "router",
			TailscaleIPs:  []string{"100.64.0.1"},
			AllowedIPs:    []string{"100.64.0.1/32", "192.168.1.0/24"},
			PrimaryRoutes: []string{"10.0.0.0/24"},
		},
	})

	if len(got) != 1 {
		t.Fatalf("got %d devices, want 1", len(got))
	}
	if !got[0].SubnetRouter {
		t.Fatal("device should be marked as a subnet router")
	}
	if len(got[0].RoutedSubnets) != 1 || got[0].RoutedSubnets[0] != "10.0.0.0/24" {
		t.Fatalf("got routed subnets %#v, want [10.0.0.0/24]", got[0].RoutedSubnets)
	}
}

func TestDevicesFromIPNStatusParsesOfficialStatusShape(t *testing.T) {
	tags := views.SliceOf([]string{"tag:server"})
	primaryRoutes := views.SliceOf([]netip.Prefix{netip.MustParsePrefix("10.0.0.0/24")})
	lastSeen := time.Date(2026, 5, 25, 12, 30, 0, 0, time.UTC)

	got := DevicesFromIPNStatus(&ipnstate.Status{
		Self: &ipnstate.PeerStatus{
			ID:            "node-self",
			HostName:      "router",
			DNSName:       "router.tailnet.ts.net.",
			TailscaleIPs:  []netip.Addr{netip.MustParseAddr("100.64.0.1")},
			OS:            "linux",
			UserID:        1,
			Tags:          &tags,
			PrimaryRoutes: &primaryRoutes,
			Online:        false,
			LastSeen:      lastSeen,
		},
		User: map[tailcfg.UserID]tailcfg.UserProfile{
			1: {ID: 1, LoginName: "alice@example.com"},
		},
	})

	if len(got) != 1 {
		t.Fatalf("got %d devices, want 1", len(got))
	}
	assertDevice(t, got[0], api.Device{
		ID:            "node-self",
		Name:          "router.tailnet.ts.net",
		IP:            "100.64.0.1",
		TailscaleIPs:  []string{"100.64.0.1"},
		OS:            "linux",
		Owner:         "alice@example.com",
		Tags:          []string{"tag:server"},
		SubnetRouter:  true,
		RoutedSubnets: []string{"10.0.0.0/24"},
		LastSeen:      "2026-05-25T12:30:00Z",
	})
}

func assertDevice(t *testing.T, got, want api.Device) {
	t.Helper()
	if got.ID != want.ID || got.Name != want.Name || got.IP != want.IP || got.OS != want.OS ||
		got.Online != want.Online || got.Owner != want.Owner || got.SubnetRouter != want.SubnetRouter ||
		got.LastSeen != want.LastSeen {
		t.Fatalf("device mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
	assertStrings(t, "tags", got.Tags, want.Tags)
	assertStrings(t, "tailscale IPs", got.TailscaleIPs, want.TailscaleIPs)
	assertStrings(t, "routed subnets", got.RoutedSubnets, want.RoutedSubnets)
}

func assertStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s mismatch: got %#v want %#v", label, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s mismatch: got %#v want %#v", label, got, want)
		}
	}
}
