package tailserve

import (
	"testing"

	"tailscale.com/ipn"
)

func TestListenAddrToProxyURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{":8080", "http://127.0.0.1:8080"},
		{"0.0.0.0:8080", "http://127.0.0.1:8080"},
		{"127.0.0.1:9090", "http://127.0.0.1:9090"},
		{"", "http://127.0.0.1:8080"},
	}

	for _, tc := range tests {
		got, err := listenAddrToProxyURL(tc.in)
		if err != nil {
			t.Fatalf("listenAddrToProxyURL(%q) error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("listenAddrToProxyURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestParseMode(t *testing.T) {
	t.Parallel()

	if ParseMode("") != ModeAuto {
		t.Fatal("empty mode should be auto")
	}
	if ParseMode("off") != ModeOff {
		t.Fatal("off should disable serve")
	}
	if ParseMode("on") != ModeOn {
		t.Fatal("on should enable serve")
	}
}

func TestAlreadyConfigured(t *testing.T) {
	t.Parallel()

	sc := &ipn.ServeConfig{}
	sc.SetWebHandler(
		&ipn.HTTPHandler{Proxy: "http://127.0.0.1:8080"},
		"tailor.example.ts.net",
		443,
		"/",
		true,
		"example.ts.net",
	)

	if !alreadyConfigured(sc, "tailor.example.ts.net", 443, "http://127.0.0.1:8080") {
		t.Fatal("expected matching serve config to be detected")
	}
	if alreadyConfigured(sc, "tailor.example.ts.net", 443, "http://127.0.0.1:9090") {
		t.Fatal("different proxy URL should not match")
	}
}

func TestPortInUseByOther(t *testing.T) {
	t.Parallel()

	sc := &ipn.ServeConfig{}
	sc.SetWebHandler(
		&ipn.HTTPHandler{Proxy: "http://127.0.0.1:3000"},
		"tailor.example.ts.net",
		443,
		"/",
		true,
		"example.ts.net",
	)

	if !portInUseByOther(sc, "tailor.example.ts.net", 443, "http://127.0.0.1:8080", "example.ts.net") {
		t.Fatal("expected existing foreign serve config to block")
	}
}

func TestPortInUseByOtherNonProxyHandler(t *testing.T) {
	t.Parallel()

	sc := &ipn.ServeConfig{}
	sc.SetWebHandler(
		&ipn.HTTPHandler{Path: "/srv/www"},
		"tailor.example.ts.net",
		443,
		"/",
		true,
		"example.ts.net",
	)

	if !portInUseByOther(sc, "tailor.example.ts.net", 443, "http://127.0.0.1:8080", "example.ts.net") {
		t.Fatal("expected existing non-proxy handler at / to block")
	}
}
