package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/Waffleophagus/tailor/internal/deploy"
	"tailscale.com/ipn/ipnstate"
)

func TestShouldUseTsnet(t *testing.T) {
	t.Setenv("TS_AUTHKEY", "")

	if !shouldUseTsnet(deploy.Environment{TailscaleMode: "auto", HasAuthKey: true}) {
		t.Fatal("auth key in auto mode should use tsnet")
	}
	if !shouldUseTsnet(deploy.Environment{TailscaleMode: "embedded"}) {
		t.Fatal("explicit embedded mode should use tsnet")
	}

	if !shouldUseTsnet(deploy.Environment{TailscaleMode: "auto"}) {
		t.Fatal("auto without an auth key should join the tailnet via interactive login")
	}

	if shouldUseTsnet(deploy.Environment{TailscaleMode: "external", HasAuthKey: true}) {
		t.Fatal("external mode should not use tsnet")
	}
	if shouldUseTsnet(deploy.Environment{TailscaleMode: "auto", WantsHostSocket: true, HasAuthKey: true}) {
		t.Fatal("host socket mode should not use tsnet")
	}
}

func TestTSNetStateDir(t *testing.T) {
	t.Setenv("TS_STATE_DIR", "")
	t.Setenv("TAILSCALE_STATE_DIR", "")

	if got := tsnetStateDir(deploy.Environment{InContainer: true}); got != containerStateDir {
		t.Fatalf("container state dir = %q, want %q", got, containerStateDir)
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		t.Skip("no user config dir:", err)
	}
	if !hasTSNetMachineState(containerStateDir) {
		want := filepath.Join(configDir, "tailor", "tsnet")
		if got := tsnetStateDir(deploy.Environment{}); got != want {
			t.Fatalf("desktop state dir = %q, want %q", got, want)
		}
	}

	t.Setenv("TAILSCALE_STATE_DIR", "/srv/tailor")
	if got := tsnetStateDir(deploy.Environment{InContainer: true}); got != "/srv/tailor" {
		t.Fatalf("explicit state dir ignored: %q", got)
	}
}

type fakeLoginClient struct {
	mu         sync.Mutex
	states     []*ipnstate.Status
	calls      int
	loginStart int
}

func (f *fakeLoginClient) StatusWithoutPeers(context.Context) (*ipnstate.Status, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	status := f.states[min(f.calls, len(f.states)-1)]
	f.calls++
	return status, nil
}

func (f *fakeLoginClient) StartLoginInteractive(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loginStart++
	return nil
}

func TestReportNodeLoginRecoversFromKeyExpiry(t *testing.T) {
	defer func(poll, wait, retry time.Duration) {
		nodeLoginPollInterval, nodeLoginURLWait, nodeLoginRetryWait = poll, wait, retry
	}(nodeLoginPollInterval, nodeLoginURLWait, nodeLoginRetryWait)
	nodeLoginPollInterval, nodeLoginURLWait, nodeLoginRetryWait = time.Millisecond, 5*time.Millisecond, time.Hour

	// Running, then the key expires: NeedsLogin with no login URL.
	client := &fakeLoginClient{states: []*ipnstate.Status{
		{BackendState: "Running"},
		{BackendState: "NeedsLogin"},
	}}
	gate := &fakeGate{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		reportNodeLogin(ctx, client, gate, ":8080", slog.New(slog.DiscardHandler))
		close(done)
	}()
	deadline := time.After(2 * time.Second)
	for {
		client.mu.Lock()
		started := client.loginStart
		client.mu.Unlock()
		if started > 0 {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatal("reportNodeLogin never started a new login after the node lost its login")
		case <-time.After(time.Millisecond):
		}
	}
	cancel()
	<-done

	client.mu.Lock()
	defer client.mu.Unlock()
	if client.loginStart != 1 {
		t.Fatalf("StartLoginInteractive called %d times, want 1 (retries are rate limited)", client.loginStart)
	}
	if gate.resets() != 1 {
		t.Fatalf("setup token rotated %d times, want once when the node came up", gate.resets())
	}
}

type fakeGate struct {
	mu sync.Mutex
	n  int
}

func (g *fakeGate) Token() string { return "tok" }

func (g *fakeGate) Reset() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
}

func (g *fakeGate) resets() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.n
}

func TestSetupPageURL(t *testing.T) {
	if got := setupPageURL(":8080", "tok"); got != "http://localhost:8080/#setup=tok" {
		t.Fatalf("setupPageURL = %q", got)
	}
}

func TestListenPort(t *testing.T) {
	for addr, want := range map[string]string{":8080": ":8080", "127.0.0.1:9000": ":9000", "": ""} {
		if got := listenPort(addr); got != want {
			t.Errorf("listenPort(%q) = %q, want %q", addr, got, want)
		}
	}
}

func TestConfigureTSNetForceLoginForAuthKeyDeployment(t *testing.T) {
	t.Setenv("TSNET_FORCE_LOGIN", "")
	configureTSNetForceLogin(deploy.Environment{HasAuthKey: true}, t.TempDir(), slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if got := os.Getenv("TSNET_FORCE_LOGIN"); got != "1" {
		t.Fatalf("TSNET_FORCE_LOGIN = %q, want 1", got)
	}
}

func TestConfigureTSNetForceLoginIgnoresEmptyStateStore(t *testing.T) {
	stateDir := t.TempDir()
	stateFile := filepath.Join(stateDir, "tailscaled.state")
	if err := os.WriteFile(stateFile, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TSNET_FORCE_LOGIN", "")
	configureTSNetForceLogin(deploy.Environment{HasAuthKey: true}, stateDir, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if got := os.Getenv("TSNET_FORCE_LOGIN"); got != "1" {
		t.Fatalf("TSNET_FORCE_LOGIN = %q, want 1 for empty state store", got)
	}
}

func TestConfigureTSNetForceLoginPreservesExistingMachineState(t *testing.T) {
	stateDir := t.TempDir()
	stateFile := filepath.Join(stateDir, "tailscaled.state")
	if err := os.WriteFile(stateFile, []byte(`{"_machinekey":"privkey:test"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TSNET_FORCE_LOGIN", "")
	configureTSNetForceLogin(deploy.Environment{HasAuthKey: true}, stateDir, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if got := os.Getenv("TSNET_FORCE_LOGIN"); got != "" {
		t.Fatalf("TSNET_FORCE_LOGIN = %q, want empty when machine state exists", got)
	}
}

func TestConfigureTSNetForceLoginPreservesExplicitValue(t *testing.T) {
	t.Setenv("TSNET_FORCE_LOGIN", "0")
	configureTSNetForceLogin(deploy.Environment{HasAuthKey: true}, t.TempDir(), slog.New(slog.NewTextHandler(os.Stderr, nil)))

	if got := os.Getenv("TSNET_FORCE_LOGIN"); got != "0" {
		t.Fatalf("TSNET_FORCE_LOGIN = %q, want explicit value preserved", got)
	}
}

func TestTsnetListenAddr(t *testing.T) {
	t.Setenv("TAILOR_TSNET_PORT", "")
	if got := tsnetListenAddr(); got != ":443" {
		t.Fatalf("default listen addr = %q, want :443", got)
	}

	t.Setenv("TAILOR_TSNET_PORT", "8443")
	if got := tsnetListenAddr(); got != ":8443" {
		t.Fatalf("port listen addr = %q, want :8443", got)
	}

	t.Setenv("TAILOR_TSNET_PORT", "127.0.0.1:8443")
	if got := tsnetListenAddr(); got != "127.0.0.1:8443" {
		t.Fatalf("explicit listen addr = %q, want 127.0.0.1:8443", got)
	}
}

func TestAdvertiseTags(t *testing.T) {
	t.Setenv("TS_ADVERTISE_TAGS", "tag:tailor-acl-service, tag:tailor-acl-editor")
	t.Setenv("TAILSCALE_ADVERTISE_TAGS", "")
	t.Setenv("TAILSCALE_UP_EXTRA_ARGS", "")

	want := []string{"tag:tailor-acl-service", "tag:tailor-acl-editor"}
	if got := advertiseTags(); !reflect.DeepEqual(got, want) {
		t.Fatalf("advertiseTags = %#v, want %#v", got, want)
	}
}

func TestAdvertiseTagsFallbacks(t *testing.T) {
	t.Setenv("TS_ADVERTISE_TAGS", "")
	t.Setenv("TAILSCALE_ADVERTISE_TAGS", "tag:tailor-acl-service")
	t.Setenv("TAILSCALE_UP_EXTRA_ARGS", "")

	if got := advertiseTags(); !reflect.DeepEqual(got, []string{"tag:tailor-acl-service"}) {
		t.Fatalf("TAILSCALE_ADVERTISE_TAGS fallback = %#v", got)
	}

	t.Setenv("TAILSCALE_ADVERTISE_TAGS", "")
	t.Setenv("TAILSCALE_UP_EXTRA_ARGS", "--accept-dns=false --advertise-tags tag:one,tag:two")
	if got := advertiseTags(); !reflect.DeepEqual(got, []string{"tag:one", "tag:two"}) {
		t.Fatalf("TAILSCALE_UP_EXTRA_ARGS fallback = %#v", got)
	}
}

func TestAdvertiseTagsHasNoImplicitDefault(t *testing.T) {
	t.Setenv("TS_ADVERTISE_TAGS", "")
	t.Setenv("TAILSCALE_ADVERTISE_TAGS", "")
	t.Setenv("TAILSCALE_UP_EXTRA_ARGS", "")

	if got := advertiseTags(); got != nil {
		t.Fatalf("defaults = %#v, want nil", got)
	}
}
