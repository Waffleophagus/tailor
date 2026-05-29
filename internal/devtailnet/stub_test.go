//go:build !dev

package devtailnet_test

import (
	"errors"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/devtailnet"
)

func TestProductionBuildExcludesDevTailnet(t *testing.T) {
	if devtailnet.Enabled {
		t.Fatal("production build must not enable dev tailnet")
	}
	if devtailnet.APIKey != "" {
		t.Fatalf("APIKey = %q, want empty", devtailnet.APIKey)
	}
	if devtailnet.IsDevAPIKey("tskey-api-tailor-dev") {
		t.Fatal("dev API key must not authenticate in production builds")
	}
	if devtailnet.Devices() != nil {
		t.Fatal("Devices() should be nil in production builds")
	}
	_, err := devtailnet.SpawnDevices(api.DevSpawnDevicesRequest{Count: 1})
	if !errors.Is(err, devtailnet.ErrUnavailable) {
		t.Fatalf("SpawnDevices err = %v, want ErrUnavailable", err)
	}
	online := true
	_, err = devtailnet.PatchDevices(api.DevPatchDevicesRequest{
		Devices: []api.DevPatchDeviceSpec{{Name: "x", Online: &online}},
	})
	if !errors.Is(err, devtailnet.ErrUnavailable) {
		t.Fatalf("PatchDevices err = %v, want ErrUnavailable", err)
	}
}
