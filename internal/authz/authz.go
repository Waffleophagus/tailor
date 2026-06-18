package authz

import (
	"context"
	"encoding/json"
	"strings"

	"tailscale.com/tailcfg"
)

type contextKey int

const (
	identityKey contextKey = iota
	bootstrapKey
)

type Role string

const (
	RoleFull   Role = "full"
	RoleViewer Role = "viewer"
)

type Permission string

const (
	PermissionViewTopology Permission = "view-topology"
	PermissionReadPolicy   Permission = "read-policy"
	PermissionWritePolicy  Permission = "write-policy"
	PermissionUseMCPWrite  Permission = "use-mcp-write"
)

type TailnetIdentity struct {
	LoginName string
	NodeName  string
	NodeTags  []string
	CapMap    tailcfg.PeerCapMap
	Role      Role
}

type appCapabilityValue struct {
	Actions []string `json:"actions"`
}

func WithIdentity(ctx context.Context, identity TailnetIdentity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

func IdentityFromContext(ctx context.Context) (TailnetIdentity, bool) {
	identity, ok := ctx.Value(identityKey).(TailnetIdentity)
	return identity, ok
}

func RoleForCapability(capMap tailcfg.PeerCapMap, capability string) Role {
	if HasAdminAction(capMap, capability) {
		return RoleFull
	}
	return RoleViewer
}

func HasAdminAction(capMap tailcfg.PeerCapMap, capability string) bool {
	capability = strings.TrimSpace(capability)
	if capability == "" {
		return false
	}
	values := capMap[tailcfg.PeerCapability(capability)]
	for _, raw := range values {
		var value appCapabilityValue
		if err := json.Unmarshal([]byte(raw), &value); err != nil {
			continue
		}
		for _, action := range value.Actions {
			if action == "admin" {
				return true
			}
		}
	}
	return false
}

func WithBootstrap(ctx context.Context) context.Context {
	return context.WithValue(ctx, bootstrapKey, true)
}

func HasBootstrap(ctx context.Context) bool {
	active, ok := ctx.Value(bootstrapKey).(bool)
	return ok && active
}

func Allowed(ctx context.Context, permission Permission) bool {
	switch permission {
	case PermissionViewTopology:
		return true
	case PermissionReadPolicy, PermissionWritePolicy:
		if HasBootstrap(ctx) {
			return true
		}
		identity, ok := IdentityFromContext(ctx)
		if !ok {
			return true
		}
		return identity.Role == RoleFull
	case PermissionUseMCPWrite:
		identity, ok := IdentityFromContext(ctx)
		if !ok {
			return true
		}
		return identity.Role == RoleFull
	default:
		return false
	}
}
