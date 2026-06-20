package tailorcore

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
)

func TestEnrichDevicesFromCloudUsesExternalAndPostureMetadata(t *testing.T) {
	devices := []api.Device{{
		ID:           "local",
		Owner:        "alice@example.com",
		TailscaleIPs: []string{"100.64.0.1", "fd7a:115c:a1e0::1"},
		PostureAttrs: map[string]any{"custom:existing": "kept"},
	}}
	enrichDevicesFromCloud(devices, []cloudapi.Device{{
		Addresses:       []string{"fd7a:115c:a1e0::1"},
		ClientVersion:   "1.94.2",
		OS:              "linux",
		IsExternal:      true,
		PostureIdentity: map[string]any{"serial": "abc123"},
	}})

	if !devices[0].Shared {
		t.Fatalf("expected cloud external device to mark topology device shared: %#v", devices[0])
	}
	if devices[0].PostureAttrs["node:tsVersion"] != "1.94.2" || devices[0].PostureAttrs["node:os"] != "linux" {
		t.Fatalf("posture attrs = %#v", devices[0].PostureAttrs)
	}
	if devices[0].PostureAttrs["custom:existing"] != "kept" {
		t.Fatalf("existing posture attr was not preserved: %#v", devices[0].PostureAttrs)
	}
	identity, ok := devices[0].PostureAttrs["node:postureIdentity"].(map[string]any)
	if !ok || identity["serial"] != "abc123" {
		t.Fatalf("posture identity = %#v", devices[0].PostureAttrs["node:postureIdentity"])
	}
}

func TestEnrichDeviceRolesFromCloudUsersUsesRealUserRoleMetadata(t *testing.T) {
	devices := []api.Device{
		{ID: "admin-laptop", Owner: "alice@example.com", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "member-laptop", Owner: "bob@example.com", TailscaleIPs: []string{"100.64.0.2"}},
		{ID: "tagged-server", Owner: "alice@example.com", Tags: []string{"tag:prod"}, TailscaleIPs: []string{"100.64.0.3"}},
	}
	enrichDeviceRolesFromCloudUsers(devices, []cloudapi.User{
		{LoginName: "alice@example.com", Role: "admin"},
		{LoginName: "bob@example.com", Role: "member"},
	})

	if len(devices[0].Roles) != 1 || devices[0].Roles[0] != "admin" {
		t.Fatalf("admin roles = %#v", devices[0].Roles)
	}
	if len(devices[1].Roles) != 1 || devices[1].Roles[0] != "member" {
		t.Fatalf("member roles = %#v", devices[1].Roles)
	}
	if len(devices[2].Roles) != 0 {
		t.Fatalf("tagged device should not inherit user role metadata: %#v", devices[2].Roles)
	}
}

func TestEnrichDeviceRolesFromCloudUsersMergesWithoutDuplicates(t *testing.T) {
	devices := []api.Device{
		{ID: "admin-laptop", Owner: "alice@example.com", Roles: []string{"Admin"}, TailscaleIPs: []string{"100.64.0.1"}},
	}
	enrichDeviceRolesFromCloudUsers(devices, []cloudapi.User{
		{LoginName: "alice@example.com", Role: "admin"},
	})

	if len(devices[0].Roles) != 1 || devices[0].Roles[0] != "Admin" {
		t.Fatalf("roles = %#v, want existing role without duplicate", devices[0].Roles)
	}
}

func TestEnrichDevicePostureAttributesUsesCloudAttributeEndpoint(t *testing.T) {
	devices := []api.Device{{
		ID:           "local",
		TailscaleIPs: []string{"100.64.0.1"},
		PostureAttrs: map[string]any{"custom:existing": "kept"},
	}}
	client := fakePostureAttributeClient{
		attrs: map[string]cloudapi.DevicePostureAttributes{
			"n1": {Attributes: map[string]any{"custom:tier": "prod", "node:osVersion": "6.8.0"}},
		},
	}
	enrichDevicePostureAttributes(context.Background(), client, devices, []cloudapi.Device{{
		Addresses: []string{"100.64.0.1"},
		NodeID:    "n1",
	}}, slog.Default())

	if devices[0].PostureAttrs["custom:tier"] != "prod" || devices[0].PostureAttrs["node:osVersion"] != "6.8.0" {
		t.Fatalf("posture attrs = %#v", devices[0].PostureAttrs)
	}
	if devices[0].PostureAttrs["custom:existing"] != "kept" {
		t.Fatalf("existing posture attr was not preserved: %#v", devices[0].PostureAttrs)
	}
}

type fakePostureAttributeClient struct {
	attrs map[string]cloudapi.DevicePostureAttributes
}

func (f fakePostureAttributeClient) DevicePostureAttributes(_ context.Context, deviceID string) (cloudapi.DevicePostureAttributes, error) {
	attrs, ok := f.attrs[deviceID]
	if !ok {
		return cloudapi.DevicePostureAttributes{}, errors.New("not found")
	}
	return attrs, nil
}

func TestEnrichDevicePostureAttributesContinuesAfterDeviceError(t *testing.T) {
	devices := []api.Device{
		{ID: "missing", TailscaleIPs: []string{"100.64.0.1"}},
		{ID: "matched", TailscaleIPs: []string{"100.64.0.2"}},
	}
	client := fakePostureAttributeClient{
		attrs: map[string]cloudapi.DevicePostureAttributes{
			"n2": {Attributes: map[string]any{"custom:tier": "prod"}},
		},
	}
	enrichDevicePostureAttributes(context.Background(), client, devices, []cloudapi.Device{
		{Addresses: []string{"100.64.0.1"}, NodeID: "n1"},
		{Addresses: []string{"100.64.0.2"}, NodeID: "n2"},
	}, slog.Default())

	if devices[0].PostureAttrs != nil {
		t.Fatalf("missing device attrs = %#v, want nil", devices[0].PostureAttrs)
	}
	if devices[1].PostureAttrs["custom:tier"] != "prod" {
		t.Fatalf("matched device attrs = %#v", devices[1].PostureAttrs)
	}
}

func TestServiceDevicesFromCloudCreatesServiceNodes(t *testing.T) {
	devices := serviceDevicesFromCloud([]cloudapi.VIPService{
		{Name: " ", Addrs: []string{"100.100.0.9"}},
		{Name: "svc:web", Addrs: []string{"100.100.0.1", "fd7a:115c:a1e0::1"}, Tags: []string{"tag:web"}},
	})

	if len(devices) != 1 {
		t.Fatalf("got %d devices, want 1", len(devices))
	}
	got := devices[0]
	if got.ID != "svc:web" || got.Kind != "service" || got.IP != "100.100.0.1" || !got.Online {
		t.Fatalf("service node = %#v", got)
	}
	if len(got.Tags) == 0 || len(got.TailscaleIPs) == 0 {
		t.Fatalf("service arrays must be non-empty: %#v", got)
	}
	if got.RoutedSubnets == nil {
		t.Fatalf("routed subnets must be an empty, non-nil slice: %#v", got)
	}
	if len(got.TailscaleIPs) != 2 || got.TailscaleIPs[1] != "fd7a:115c:a1e0::1" {
		t.Fatalf("service addrs = %#v", got.TailscaleIPs)
	}
}
