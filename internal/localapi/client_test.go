package localapi

import (
	"encoding/json"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
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
			ID:     "node-self",
			Name:   "laptop.tailnet.ts.net",
			IP:     "100.64.0.1",
			OS:     "linux",
			Online: true,
			Owner:  "alice@example.com",
			Tags:   []string{"tag:workstation"},
		},
		{
			ID:       "node-peer",
			Name:     "server.tailnet.ts.net",
			IP:       "100.64.0.2",
			OS:       "linux",
			Online:   false,
			Owner:    "ops@example.com",
			Tags:     []string{"tag:server", "tag:database"},
			LastSeen: "2026-05-25T12:30:00Z",
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

func assertDevice(t *testing.T, got, want api.Device) {
	t.Helper()
	if got.ID != want.ID || got.Name != want.Name || got.IP != want.IP || got.OS != want.OS ||
		got.Online != want.Online || got.Owner != want.Owner || got.LastSeen != want.LastSeen {
		t.Fatalf("device mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
	if len(got.Tags) != len(want.Tags) {
		t.Fatalf("tags mismatch: got %#v want %#v", got.Tags, want.Tags)
	}
	for i := range want.Tags {
		if got.Tags[i] != want.Tags[i] {
			t.Fatalf("tags mismatch: got %#v want %#v", got.Tags, want.Tags)
		}
	}
}
