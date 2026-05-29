package deploy

import "testing"

func TestWantsHostSocket(t *testing.T) {
	t.Setenv("TAILOR_LOCALAPI_SOCKET", "")

	if wantsHostSocket("external") != true {
		t.Fatal("external mode should want host socket")
	}
	if wantsHostSocket("embedded") != false {
		t.Fatal("embedded mode should not want host socket")
	}

	t.Setenv("TAILOR_LOCALAPI_SOCKET", "/var/run/tailscale/tailscaled.sock")
	if wantsHostSocket("auto") != true {
		t.Fatal("auto with socket env should want host socket")
	}
}

func TestNeedsTailscaleSetup(t *testing.T) {
	base := Environment{InContainer: true}

	withKey := Environment{InContainer: true, HasAuthKey: true}
	if withKey.NeedsTailscaleSetup(false, 0) {
		t.Fatal("auth key configured should not need setup")
	}

	if !base.NeedsTailscaleSetup(false, 0) {
		t.Fatal("container without auth or socket should need setup when LocalAPI is down")
	}
	if !base.NeedsTailscaleSetup(true, 0) {
		t.Fatal("container without auth should need setup when there are no devices")
	}
	if base.NeedsTailscaleSetup(true, 2) {
		t.Fatal("container with peers should not need setup")
	}

	host := Environment{InContainer: true, WantsHostSocket: true}
	if !host.NeedsTailscaleSetup(false, 0) {
		t.Fatal("broken host socket mode should need setup")
	}
	if host.NeedsTailscaleSetup(true, 3) {
		t.Fatal("working host socket with devices should not need setup")
	}
}

func TestSetupInfoNilWhenConfigured(t *testing.T) {
	env := Environment{InContainer: true, HasAuthKey: true}
	if env.SetupInfo(false, 0) != nil {
		t.Fatal("expected nil setup when auth key is set")
	}
}

func TestSetupInfoHints(t *testing.T) {
	env := Environment{InContainer: true, LikelyDockerDesktop: true, WantsHostSocket: true}
	info := env.SetupInfo(false, 0)
	if info == nil || !info.Required || len(info.Hints) < 2 {
		t.Fatalf("setup = %#v, want required with multiple hints", info)
	}
}
