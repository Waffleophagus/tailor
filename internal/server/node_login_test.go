package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tailscale.com/ipn/ipnstate"
)

const loginURL = "https://login.tailscale.com/a/abc123"

type fakeNodeStatus struct{ status *ipnstate.Status }

func (f fakeNodeStatus) StatusWithoutPeers(context.Context) (*ipnstate.Status, error) {
	return f.status, nil
}

func identityHandler(status *ipnstate.Status) http.Handler {
	return IdentityMiddleware(nil, &AuthOptions{
		TailnetMode:   true,
		TailnetStatus: fakeNodeStatus{status: status},
		WhoIsClient:   fakeWhoIsClient{err: context.DeadlineExceeded},
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
}

func serve(handler http.Handler, path, remoteAddr string) *httptest.ResponseRecorder {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost:8080"+path, nil)
	request.RemoteAddr = remoteAddr
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func TestNodeLoginPageShowsLinkToLocalAndPrivateClients(t *testing.T) {
	handler := identityHandler(&ipnstate.Status{BackendState: "NeedsLogin", AuthURL: loginURL})

	for _, remote := range []string{"127.0.0.1:5000", "[::1]:5000", "192.168.1.20:5000", "172.17.0.1:5000", "100.101.102.103:5000"} {
		response := serve(handler, "/", remote)
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), loginURL) {
			t.Errorf("%s: status=%d, login link shown=%v", remote, response.Code, strings.Contains(response.Body.String(), loginURL))
		}
	}
}

func TestNodeLoginLinkIsHiddenFromPublicClients(t *testing.T) {
	handler := identityHandler(&ipnstate.Status{BackendState: "NeedsLogin", AuthURL: loginURL})

	response := serve(handler, "/", "203.0.113.9:5000")
	if strings.Contains(response.Body.String(), loginURL) {
		t.Fatal("login link shown to a public client")
	}
	if !strings.Contains(response.Body.String(), "logs") {
		t.Fatal("public clients should be pointed at the logs")
	}

	api := serve(handler, "/api/cloud/status", "203.0.113.9:5000")
	if api.Code != http.StatusServiceUnavailable || strings.Contains(api.Body.String(), loginURL) {
		t.Fatalf("api status=%d body=%s", api.Code, api.Body.String())
	}
}

func TestNodeLoginPageAsksForApprovalWhenRequired(t *testing.T) {
	handler := identityHandler(&ipnstate.Status{BackendState: "NeedsMachineAuth", Self: &ipnstate.PeerStatus{HostName: "tailor"}})

	response := serve(handler, "/", "127.0.0.1:5000")
	if !strings.Contains(response.Body.String(), "Approve") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestRunningNodeRequiresIdentityAndLinksToTailnetURL(t *testing.T) {
	handler := identityHandler(&ipnstate.Status{
		BackendState: "Running",
		Self:         &ipnstate.PeerStatus{DNSName: "tailor.example.ts.net."},
	})

	response := serve(handler, "/", "192.168.1.20:5000")
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if !strings.Contains(response.Body.String(), `href="https://tailor.example.ts.net/"`) {
		t.Fatalf("private page should link to the tailnet URL: %s", response.Body.String())
	}
}
