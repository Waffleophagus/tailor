package authz

import (
	"testing"

	"tailscale.com/tailcfg"
)

func TestRoleForCapabilityRequiresAdminAction(t *testing.T) {
	const cap = "tailor.example.ts.net/cap/admin"
	tests := []struct {
		name string
		caps tailcfg.PeerCapMap
		want Role
	}{
		{
			name: "admin action",
			caps: tailcfg.PeerCapMap{
				tailcfg.PeerCapability(cap): []tailcfg.RawMessage{`{"actions":["admin"]}`},
			},
			want: RoleFull,
		},
		{
			name: "other action",
			caps: tailcfg.PeerCapMap{
				tailcfg.PeerCapability(cap): []tailcfg.RawMessage{`{"actions":["read"]}`},
			},
			want: RoleViewer,
		},
		{
			name: "wrong capability",
			caps: tailcfg.PeerCapMap{
				tailcfg.PeerCapability("other.example.ts.net/cap/admin"): []tailcfg.RawMessage{`{"actions":["admin"]}`},
			},
			want: RoleViewer,
		},
		{
			name: "invalid value",
			caps: tailcfg.PeerCapMap{
				tailcfg.PeerCapability(cap): []tailcfg.RawMessage{`not-json`},
			},
			want: RoleViewer,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RoleForCapability(tt.caps, cap); got != tt.want {
				t.Fatalf("RoleForCapability() = %q, want %q", got, tt.want)
			}
		})
	}
}
