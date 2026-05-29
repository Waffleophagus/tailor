//go:build !dev

package devtailnet

import (
	"errors"

	"github.com/Waffleophagus/tailor/internal/api"
)

// Enabled reports whether this binary was built with the dev tag.
const Enabled = false

// APIKey is empty in production builds.
const APIKey = ""

// Name is empty in production builds.
const Name = ""

var ErrUnavailable = errors.New("dev tailnet is not available in this build")

func IsDevAPIKey(string) bool { return false }

func Devices() []api.Device { return nil }

func Policy() string { return "" }

func SpawnDevices(api.DevSpawnDevicesRequest) ([]api.Device, error) {
	return nil, ErrUnavailable
}

func PatchDevices(api.DevPatchDevicesRequest) ([]api.Device, error) {
	return nil, ErrUnavailable
}
