package deploy

import (
	"os"
	"strings"
)

// Environment describes how Tailor was started (Docker, Tailscale mode, credentials).
type Environment struct {
	InContainer       bool
	TailscaleMode     string
	HasAuthKey        bool
	WantsHostSocket   bool
	LikelyDockerDesktop bool
}

// Detect reads the process environment and runtime cues (e.g. /.dockerenv).
func Detect() Environment {
	mode := strings.TrimSpace(os.Getenv("TAILOR_TAILSCALE_MODE"))
	if mode == "" {
		mode = "auto"
	}

	env := Environment{
		InContainer:     inContainer(),
		TailscaleMode:   mode,
		HasAuthKey:      strings.TrimSpace(os.Getenv("TAILSCALE_AUTHKEY")) != "",
		WantsHostSocket: wantsHostSocket(mode),
	}
	env.LikelyDockerDesktop = env.InContainer && likelyDockerDesktopKernel()
	return env
}

// NeedsTailscaleSetup reports whether the deployment cannot show a useful tailnet yet.
func (e Environment) NeedsTailscaleSetup(localAPIAvailable bool, deviceCount int) bool {
	if !e.InContainer || e.HasAuthKey {
		return false
	}
	return !localAPIAvailable || deviceCount == 0
}

// SetupInfo returns UI-facing setup guidance when NeedsTailscaleSetup is true.
func (e Environment) SetupInfo(localAPIAvailable bool, deviceCount int) *SetupInfo {
	if !e.NeedsTailscaleSetup(localAPIAvailable, deviceCount) {
		return nil
	}
	return &SetupInfo{
		Required: true,
		Hints:    e.hints(localAPIAvailable),
	}
}

func wantsHostSocket(mode string) bool {
	if mode == "external" {
		return true
	}
	if mode == "embedded" {
		return false
	}
	return strings.TrimSpace(os.Getenv("TAILOR_LOCALAPI_SOCKET")) != ""
}

func inContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}
	data, err := os.ReadFile("/proc/1/cgroup")
	if err != nil {
		return hasContainerEnviron()
	}
	content := string(data)
	if strings.Contains(content, "docker") ||
		strings.Contains(content, "containerd") ||
		strings.Contains(content, "kubelet") {
		return true
	}
	if strings.Contains(content, "0::/") {
		mountinfo, err := os.ReadFile("/proc/1/mountinfo")
		if err == nil {
			mi := string(mountinfo)
			if strings.Contains(mi, "kubepods") || strings.Contains(mi, "containerd") {
				return true
			}
		}
	}
	return hasContainerEnviron()
}

func hasContainerEnviron() bool {
	data, err := os.ReadFile("/proc/1/environ")
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "container=")
}

func likelyDockerDesktopKernel() bool {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(data)), "microsoft")
}

func (e Environment) hints(localAPIAvailable bool) []SetupHint {
	var hints []SetupHint

	if e.WantsHostSocket {
		msg := "Host socket mode only works on Linux when you mount the host tailscaled.sock into the container."
		if e.LikelyDockerDesktop {
			msg += " On Docker Desktop (Windows or macOS) that socket is not available — use TAILSCALE_AUTHKEY instead."
		}
		hints = append(hints, SetupHint{ID: "host-socket", Message: msg})
	}

	if !e.HasAuthKey {
		hints = append(hints, SetupHint{
			ID: "auth-key",
			Message: "Set TAILSCALE_AUTHKEY to a tskey-auth-… key so this container joins your tailnet as its own node. " +
				"Optional: TAILSCALE_HOSTNAME=tailor. Tailor then exposes HTTPS via Tailscale Serve at https://tailor.<your-tailnet>.ts.net/.",
		})
	}

	if !localAPIAvailable && !e.WantsHostSocket {
		hints = append(hints, SetupHint{
			ID: "localapi-wait",
			Message: "Tailscale is starting inside the container. After you set TAILSCALE_AUTHKEY, restart the container.",
		})
	}

	return hints
}
