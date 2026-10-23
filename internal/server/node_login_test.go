package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tailscale.com/ipn/ipnstate"
)

const (
	loginURL   = "https://login.tailscale.com/a/abc123"
	setupToken = "0123456789abcdef0123456789abcdef"
)

type fakeNodeStatus struct{ status *ipnstate.Status }

func (f fakeNodeStatus) StatusWithoutPeers(context.Context) (*ipnstate.Status, error) {
	return f.status, nil
}

func identityHandler(status *ipnstate.Status, httpsPort string) http.Handler {
	return IdentityMiddleware(nil, &AuthOptions{
		TailnetMode:      true,
		TailnetStatus:    fakeNodeStatus{status: status},
		WhoIsClient:      fakeWhoIsClient{err: context.DeadlineExceeded},
		SetupToken:       setupToken,
		TailnetHTTPSPort: httpsPort,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
}

func serve(handler http.Handler, host, target, remoteAddr string) *httptest.ResponseRecorder {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+host+target, nil)
	request.RemoteAddr = remoteAddr
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

var needsLogin = &ipnstate.Status{BackendState: "NeedsLogin", AuthURL: loginURL}

func TestNodeLoginLinkRequiresSetupToken(t *testing.T) {
	handler := identityHandler(needsLogin, "")

	for _, remote := range []string{"127.0.0.1:5000", "[::1]:5000", "192.168.1.20:5000", "172.17.0.1:5000", "100.101.102.103:5000"} {
		with := serve(handler, "localhost:8080", "/?setup="+setupToken, remote)
		if with.Code != http.StatusOK || !strings.Contains(with.Body.String(), loginURL) {
			t.Errorf("%s with token: status=%d, link shown=%v", remote, with.Code, strings.Contains(with.Body.String(), loginURL))
		}
		// A reverse proxy on loopback or a container network makes every visitor
		// look local, so the address alone must never reveal the link.
		without := serve(handler, "localhost:8080", "/", remote)
		if strings.Contains(without.Body.String(), loginURL) {
			t.Errorf("%s without token: login link shown", remote)
		}
		if !strings.Contains(without.Body.String(), "setup link printed in Tailor") {
			t.Errorf("%s without token: should point at the setup link in the logs", remote)
		}
	}

	wrong := serve(handler, "localhost:8080", "/?setup=guess", "127.0.0.1:5000")
	if strings.Contains(wrong.Body.String(), loginURL) {
		t.Fatal("login link shown for a wrong setup token")
	}
}

func TestNodeLoginLinkIsHiddenFromPublicClientsEvenWithToken(t *testing.T) {
	handler := identityHandler(needsLogin, "")

	response := serve(handler, "localhost:8080", "/?setup="+setupToken, "203.0.113.9:5000")
	if strings.Contains(response.Body.String(), loginURL) {
		t.Fatal("login link shown to a public client")
	}
	api := serve(handler, "localhost:8080", "/api/status", "203.0.113.9:5000")
	if api.Code != http.StatusServiceUnavailable || strings.Contains(api.Body.String(), loginURL) {
		t.Fatalf("api status=%d body=%s", api.Code, api.Body.String())
	}
}

func TestNodeLoginRejectsDNSRebindingHosts(t *testing.T) {
	handler := identityHandler(needsLogin, "")

	response := serve(handler, "rebind.attacker.example:8080", "/api/status?setup="+setupToken, "127.0.0.1:5000")
	if response.Code != http.StatusMisdirectedRequest || strings.Contains(response.Body.String(), loginURL) {
		t.Fatalf("rebinding host: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestNodeLoginPageAsksForApprovalWhenRequired(t *testing.T) {
	handler := identityHandler(&ipnstate.Status{BackendState: "NeedsMachineAuth", Self: &ipnstate.PeerStatus{HostName: "tailor"}}, "")

	response := serve(handler, "localhost:8080", "/", "127.0.0.1:5000")
	if !strings.Contains(response.Body.String(), "Approve") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestNodeLoginPageExplainsRelogin(t *testing.T) {
	// After key expiry the node needs login but has no login URL until a new
	// interactive login starts; the page must not claim it is merely connecting.
	handler := identityHandler(&ipnstate.Status{BackendState: "NeedsLogin"}, "")

	response := serve(handler, "localhost:8080", "/?setup="+setupToken, "127.0.0.1:5000")
	if !strings.Contains(response.Body.String(), "Requesting a login link") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestRunningNodeRequiresIdentityAndLinksToTailnetURL(t *testing.T) {
	running := &ipnstate.Status{BackendState: "Running", Self: &ipnstate.PeerStatus{DNSName: "tailor.example.ts.net."}}

	for port, want := range map[string]string{"": "https://tailor.example.ts.net/", "443": "https://tailor.example.ts.net/", "8443": "https://tailor.example.ts.net:8443/"} {
		response := serve(identityHandler(running, port), "localhost:8080", "/", "192.168.1.20:5000")
		if response.Code != http.StatusForbidden {
			t.Fatalf("port %q: status = %d, want %d", port, response.Code, http.StatusForbidden)
		}
		if !strings.Contains(response.Body.String(), `href="`+want+`"`) {
			t.Fatalf("port %q: private page should link to %s: %s", port, want, response.Body.String())
		}
	}
}
