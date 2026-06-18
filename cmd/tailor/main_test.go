package main

import (
	"reflect"
	"testing"

	"github.com/Waffleophagus/tailor/internal/deploy"
)

func TestShouldUseTsnet(t *testing.T) {
	t.Setenv("TS_AUTHKEY", "")

	if !shouldUseTsnet(deploy.Environment{TailscaleMode: "auto", HasAuthKey: true}) {
		t.Fatal("auth key in auto mode should use tsnet")
	}
	if !shouldUseTsnet(deploy.Environment{TailscaleMode: "embedded"}) {
		t.Fatal("explicit embedded mode should use tsnet")
	}

	t.Setenv("TS_AUTHKEY", "tskey-auth-test")
	if !shouldUseTsnet(deploy.Environment{TailscaleMode: "auto"}) {
		t.Fatal("TS_AUTHKEY should use tsnet")
	}

	if shouldUseTsnet(deploy.Environment{TailscaleMode: "external", HasAuthKey: true}) {
		t.Fatal("external mode should not use tsnet")
	}
	if shouldUseTsnet(deploy.Environment{TailscaleMode: "auto", WantsHostSocket: true, HasAuthKey: true}) {
		t.Fatal("host socket mode should not use tsnet")
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
