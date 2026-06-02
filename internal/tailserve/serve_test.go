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

	if alreadyConfigured(sc, "tailor.example.ts.net", 443, "http://127.0.0.1:8080", "308:https://tailor.example.ts.net${REQUEST_URI}") {
		t.Fatal("expected missing HTTP redirect to be incomplete")
	}

	sc.SetWebHandler(
		&ipn.HTTPHandler{Redirect: "308:https://tailor.example.ts.net${REQUEST_URI}"},
		"tailor.example.ts.net",
		80,
		"/",
		false,
		"example.ts.net",
	)

	if !alreadyConfigured(sc, "tailor.example.ts.net", 443, "http://127.0.0.1:8080", "308:https://tailor.example.ts.net${REQUEST_URI}") {
		t.Fatal("expected matching serve config to be detected")
	}
	if alreadyConfigured(sc, "tailor.example.ts.net", 443, "http://127.0.0.1:9090", "308:https://tailor.example.ts.net${REQUEST_URI}") {
		t.Fatal("different proxy URL should not match")
	}
}

func TestHTTPRedirectInUseByOtherAllowsMatchingRedirect(t *testing.T) {
	t.Parallel()

	sc := &ipn.ServeConfig{}
	sc.SetWebHandler(
		&ipn.HTTPHandler{Redirect: "308:https://tailor.example.ts.net${REQUEST_URI}"},
		"tailor.example.ts.net",
		80,
		"/",
		false,
		"example.ts.net",
	)

	if httpRedirectInUseByOther(sc, "tailor.example.ts.net", "308:https://tailor.example.ts.net${REQUEST_URI}", "example.ts.net") {
		t.Fatal("expected matching HTTP redirect to be accepted")
	}
}

func TestHTTPRedirectInUseByOtherAllowsStaleLocalProxy(t *testing.T) {
	t.Parallel()

	sc := &ipn.ServeConfig{}
	sc.SetWebHandler(
		&ipn.HTTPHandler{Proxy: "http://127.0.0.1:80"},
		"tailor.example.ts.net",
		80,
		"/",
		false,
		"example.ts.net",
	)

	if httpRedirectInUseByOther(sc, "tailor.example.ts.net", "308:https://tailor.example.ts.net${REQUEST_URI}", "example.ts.net") {
		t.Fatal("expected stale local HTTP proxy to be repairable")
	}
}

func TestHTTPRedirectInUseByOtherBlocksDifferentRedirect(t *testing.T) {
	t.Parallel()

	sc := &ipn.ServeConfig{}
	sc.SetWebHandler(
		&ipn.HTTPHandler{Redirect: "308:https://other.example.ts.net${REQUEST_URI}"},
		"tailor.example.ts.net",
		80,
		"/",
		false,
		"example.ts.net",
	)

	if !httpRedirectInUseByOther(sc, "tailor.example.ts.net", "308:https://tailor.example.ts.net${REQUEST_URI}", "example.ts.net") {
		t.Fatal("expected different HTTP redirect to block")
	}
}

func TestHTTPRedirectInUseByOtherWithMismatchedRedirectAndLocalProxy(t *testing.T) {
	t.Parallel()

	sc := &ipn.ServeConfig{}
	sc.SetWebHandler(
		&ipn.HTTPHandler{
			Redirect: "308:https://other.example.ts.net${REQUEST_URI}",
			Proxy:    "http://127.0.0.1:8080",
		},
		"tailor.example.ts.net",
		80,
		"/",
		false,
		"example.ts.net",
	)

	if !httpRedirectInUseByOther(sc, "tailor.example.ts.net", "308:https://tailor.example.ts.net${REQUEST_URI}", "example.ts.net") {
		t.Fatal("expected mismatched redirect to block even with local proxy")
	}
}

func TestHTTPSRedirectURL(t *testing.T) {
	t.Parallel()

	if got := httpsRedirectURL("tailor.example.ts.net", 443); got != "308:https://tailor.example.ts.net${REQUEST_URI}" {
		t.Fatalf("default redirect = %q", got)
	}
	if got := httpsRedirectURL("tailor.example.ts.net", 8443); got != "308:https://tailor.example.ts.net:8443${REQUEST_URI}" {
		t.Fatalf("custom-port redirect = %q", got)
	}
}

func TestPortInUseByOtherAllowsStaleLocalProxy(t *testing.T) {
	t.Parallel()

	sc := &ipn.ServeConfig{}
	sc.SetWebHandler(
		&ipn.HTTPHandler{Proxy: "http://127.0.0.1:80"},
		"tailor.example.ts.net",
		443,
		"/",
		true,
		"example.ts.net",
	)

	if portInUseByOther(sc, "tailor.example.ts.net", 443, "http://127.0.0.1:8080", "example.ts.net") {
		t.Fatal("expected existing local proxy to be repairable")
	}
}

func TestRepairableLocalProxy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"ipv4 loopback with port", "http://127.0.0.2:8080", true},
		{"ipv6 loopback with port", "http://[::1]:8080", true},
		{"localhost case insensitive", "http://LOCALHOST:8080", true},
		{"ipv4 loopback without port", "http://127.0.0.2", true},
		{"ipv6 loopback without port", "http://[::1]", true},
		{"foreign ip", "http://192.0.2.10:3000", false},
		{"foreign hostname", "http://example.com:3000", false},
	}

	for _, tc := range tests {
		if got := repairableLocalProxy(tc.in); got != tc.want {
			t.Fatalf("%s: repairableLocalProxy(%q) = %v, want %v", tc.name, tc.in, got, tc.want)
		}
	}
}

func TestPortInUseByOtherBlocksForeignProxy(t *testing.T) {
	t.Parallel()

	sc := &ipn.ServeConfig{}
	sc.SetWebHandler(
		&ipn.HTTPHandler{Proxy: "http://192.0.2.10:3000"},
		"tailor.example.ts.net",
		443,
		"/",
		true,
		"example.ts.net",
	)

	if !portInUseByOther(sc, "tailor.example.ts.net", 443, "http://127.0.0.1:8080", "example.ts.net") {
		t.Fatal("expected existing foreign proxy config to block")
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
