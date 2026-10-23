package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"tailscale.com/ipn/ipnstate"
)

const loginURL = "https://login.tailscale.com/a/abc123"

type fakeNodeStatus struct{ status *ipnstate.Status }

func (f fakeNodeStatus) StatusWithoutPeers(context.Context) (*ipnstate.Status, error) {
	return f.status, nil
}

func identityHandler(status *ipnstate.Status, gate *SetupGate, httpsPort string) http.Handler {
	return IdentityMiddleware(nil, &AuthOptions{
		TailnetMode:      true,
		TailnetStatus:    fakeNodeStatus{status: status},
		WhoIsClient:      fakeWhoIsClient{err: context.DeadlineExceeded},
		SetupGate:        gate,
		TailnetHTTPSPort: httpsPort,
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
}

func serve(handler http.Handler, host, target, remoteAddr string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://"+host+target, nil)
	request.RemoteAddr = remoteAddr
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

// exchange posts a setup token like the setup page's script does.
func exchange(handler http.Handler, host, token, remoteAddr string) *httptest.ResponseRecorder {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "http://"+host+"/setup/exchange",
		bytes.NewBufferString(`{"token":"`+token+`"}`))
	request.RemoteAddr = remoteAddr
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func setupCookie(t *testing.T, response *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == setupCookieName {
			if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode {
				t.Fatalf("setup cookie must be HttpOnly and SameSite=Strict: %+v", cookie)
			}
			return cookie
		}
	}
	t.Fatalf("no setup cookie (status %d, body %s)", response.Code, response.Body.String())
	return nil
}

var needsLogin = &ipnstate.Status{BackendState: "NeedsLogin", AuthURL: loginURL}

func TestNodeLoginLinkRequiresExchangedSetupToken(t *testing.T) {
	gate := NewSetupGate()
	handler := identityHandler(needsLogin, gate, "")

	for _, remote := range []string{"127.0.0.1:5000", "[::1]:5000", "192.168.1.20:5000", "172.17.0.1:5000", "100.101.102.103:5000"} {
		// A reverse proxy on loopback or a container network makes every visitor
		// look local, so the address alone must never reveal the link.
		bare := serve(handler, "localhost:8080", "/", remote)
		if strings.Contains(bare.Body.String(), loginURL) {
			t.Errorf("%s without a setup session: login link shown", remote)
		}
		if !strings.Contains(bare.Body.String(), "setup link printed in Tailor") {
			t.Errorf("%s without a setup session: should point at the setup link in the logs", remote)
		}

		response := exchange(handler, "localhost:8080", gate.Token(), remote)
		if response.Code != http.StatusNoContent {
			t.Fatalf("%s exchange status = %d", remote, response.Code)
		}
		withCookie := serve(handler, "localhost:8080", "/", remote, setupCookie(t, response))
		if !strings.Contains(withCookie.Body.String(), loginURL) {
			t.Errorf("%s with a setup session: login link not shown", remote)
		}
	}
}

func TestNodeLoginIgnoresTokenInQueryString(t *testing.T) {
	// Query strings end up in proxy access logs; only the fragment+exchange flow works.
	gate := NewSetupGate()
	handler := identityHandler(needsLogin, gate, "")
	if strings.Contains(serve(handler, "localhost:8080", "/?setup="+gate.Token(), "127.0.0.1:5000").Body.String(), loginURL) {
		t.Fatal("a token in the query string revealed the login link")
	}
}

func TestSetupExchangeRejectsWrongTokenAndPublicClients(t *testing.T) {
	gate := NewSetupGate()
	handler := identityHandler(needsLogin, gate, "")

	if code := exchange(handler, "localhost:8080", "guess", "127.0.0.1:5000").Code; code != http.StatusForbidden {
		t.Fatalf("wrong token: status = %d, want 403", code)
	}
	if code := exchange(handler, "localhost:8080", gate.Token(), "203.0.113.9:5000").Code; code != http.StatusForbidden {
		t.Fatalf("public client: status = %d, want 403", code)
	}
	if code := exchange(handler, "rebind.attacker.example:8080", gate.Token(), "127.0.0.1:5000").Code; code != http.StatusMisdirectedRequest {
		t.Fatalf("rebinding host: status = %d, want 421", code)
	}
}

func TestSetupGateResetInvalidatesTokensAndSessions(t *testing.T) {
	gate := NewSetupGate()
	handler := identityHandler(needsLogin, gate, "")
	oldToken := gate.Token()
	cookie := setupCookie(t, exchange(handler, "localhost:8080", oldToken, "127.0.0.1:5000"))

	gate.Reset() // the node came up; later re-login gets a fresh token

	if strings.Contains(serve(handler, "localhost:8080", "/", "127.0.0.1:5000", cookie).Body.String(), loginURL) {
		t.Fatal("a setup session survived the token rotation")
	}
	if code := exchange(handler, "localhost:8080", oldToken, "127.0.0.1:5000").Code; code != http.StatusForbidden {
		t.Fatalf("old token after rotation: status = %d, want 403", code)
	}
}

func TestNodeLoginLinkIsHiddenFromPublicClientsEvenWithSession(t *testing.T) {
	gate := NewSetupGate()
	handler := identityHandler(needsLogin, gate, "")
	cookie := setupCookie(t, exchange(handler, "localhost:8080", gate.Token(), "127.0.0.1:5000"))

	if strings.Contains(serve(handler, "localhost:8080", "/", "203.0.113.9:5000", cookie).Body.String(), loginURL) {
		t.Fatal("login link shown to a public client")
	}
	api := serve(handler, "localhost:8080", "/api/status", "203.0.113.9:5000", cookie)
	if api.Code != http.StatusServiceUnavailable || strings.Contains(api.Body.String(), loginURL) {
		t.Fatalf("api status=%d body=%s", api.Code, api.Body.String())
	}
}

func TestNodeLoginPageScriptRunsOnlyUnderItsNonce(t *testing.T) {
	response := serve(identityHandler(needsLogin, NewSetupGate(), ""), "localhost:8080", "/", "127.0.0.1:5000")
	csp := response.Header().Get("Content-Security-Policy")
	match := regexp.MustCompile(`script-src 'nonce-([0-9a-f]+)'`).FindStringSubmatch(csp)
	if match == nil {
		t.Fatalf("CSP must allow scripts only by nonce: %q", csp)
	}
	if !strings.Contains(response.Body.String(), `<script nonce="`+match[1]+`">`) {
		t.Fatal("setup script does not carry the CSP nonce")
	}
	if response.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Fatal("setup page must not leak its URL via Referer")
	}
}

func TestNodeLoginRejectsDNSRebindingHosts(t *testing.T) {
	response := serve(identityHandler(needsLogin, NewSetupGate(), ""), "rebind.attacker.example:8080", "/api/status", "127.0.0.1:5000")
	if response.Code != http.StatusMisdirectedRequest || strings.Contains(response.Body.String(), loginURL) {
		t.Fatalf("rebinding host: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestNodeLoginPageAsksForApprovalWhenRequired(t *testing.T) {
	handler := identityHandler(&ipnstate.Status{BackendState: "NeedsMachineAuth", Self: &ipnstate.PeerStatus{HostName: "tailor"}}, NewSetupGate(), "")

	if !strings.Contains(serve(handler, "localhost:8080", "/", "127.0.0.1:5000").Body.String(), "Approve") {
		t.Fatal("approval page missing")
	}
}

func TestNodeLoginPageExplainsRelogin(t *testing.T) {
	// After key expiry the node needs login but has no login URL until a new
	// interactive login starts; the page must not claim it is merely connecting.
	handler := identityHandler(&ipnstate.Status{BackendState: "NeedsLogin"}, NewSetupGate(), "")

	if !strings.Contains(serve(handler, "localhost:8080", "/", "127.0.0.1:5000").Body.String(), "Requesting a login link") {
		t.Fatal("re-login state missing")
	}
}

func TestRunningNodeRequiresIdentityAndLinksToTailnetURL(t *testing.T) {
	running := &ipnstate.Status{BackendState: "Running", Self: &ipnstate.PeerStatus{DNSName: "tailor.example.ts.net."}}

	for port, want := range map[string]string{"": "https://tailor.example.ts.net/", "443": "https://tailor.example.ts.net/", "8443": "https://tailor.example.ts.net:8443/"} {
		response := serve(identityHandler(running, NewSetupGate(), port), "localhost:8080", "/", "192.168.1.20:5000")
		if response.Code != http.StatusForbidden {
			t.Fatalf("port %q: status = %d, want %d", port, response.Code, http.StatusForbidden)
		}
		if !strings.Contains(response.Body.String(), `href="`+want+`"`) {
			t.Fatalf("port %q: private page should link to %s", port, want)
		}
	}
}
