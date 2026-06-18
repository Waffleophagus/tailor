package authz

import (
	"context"
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

func TestPermissionMatrix(t *testing.T) {
	viewer := WithIdentity(context.Background(), TailnetIdentity{Role: RoleViewer})
	full := WithIdentity(context.Background(), TailnetIdentity{Role: RoleFull})
	bootstrap := WithBootstrap(viewer)

	tests := []struct {
		name       string
		ctx        context.Context
		permission Permission
		want       bool
	}{
		{"viewer topology", viewer, PermissionViewTopology, true},
		{"viewer policy read", viewer, PermissionReadPolicy, false},
		{"viewer policy write", viewer, PermissionWritePolicy, false},
		{"viewer MCP write", viewer, PermissionUseMCPWrite, false},
		{"full policy read", full, PermissionReadPolicy, true},
		{"full policy write", full, PermissionWritePolicy, true},
		{"full MCP write", full, PermissionUseMCPWrite, true},
		{"bootstrap policy read", bootstrap, PermissionReadPolicy, true},
		{"bootstrap policy write", bootstrap, PermissionWritePolicy, true},
		{"bootstrap MCP write", bootstrap, PermissionUseMCPWrite, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Allowed(tt.ctx, tt.permission); got != tt.want {
				t.Fatalf("Allowed() = %v, want %v", got, tt.want)
			}
		})
	}
}
