package server

import (
	"context"
	"html/template"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"tailscale.com/ipn/ipnstate"
)

// nodeStateTTL bounds how often the identity middleware asks tsnet for its state.
const nodeStateTTL = 2 * time.Second

// tailnetCGNAT is the Tailscale IPv4 range; clients there are on some tailnet.
var tailnetCGNAT = netip.MustParsePrefix("100.64.0.0/10")

// nodeState is the cached state of Tailor's own tsnet node.
type nodeState struct {
	backendState string
	authURL      string
	dnsName      string
	hostname     string
}

func (s nodeState) running() bool { return s.backendState == "Running" }

// tailorURL is where Tailor is served on the tailnet once the node is running.
// httpsPort is the tailnet HTTPS port; empty or "443" is omitted from the URL.
func (s nodeState) tailorURL(httpsPort string) string {
	if s.dnsName == "" {
		return ""
	}
	if httpsPort != "" && httpsPort != "443" {
		return "https://" + net.JoinHostPort(s.dnsName, httpsPort) + "/"
	}
	return "https://" + s.dnsName + "/"
}

type nodeStateCache struct {
	mu        sync.Mutex
	state     nodeState
	ok        bool
	checkedAt time.Time
}

// get returns the node state, refreshed at most every nodeStateTTL. ok is false when
// the state is unknown (no status client, or tsnet errored).
func (c *nodeStateCache) get(ctx context.Context, client TailnetStatusClient) (nodeState, bool) {
	if client == nil {
		return nodeState{}, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.checkedAt) < nodeStateTTL {
		return c.state, c.ok
	}
	status, err := client.StatusWithoutPeers(ctx)
	c.checkedAt = time.Now()
	if err != nil || status == nil {
		c.state, c.ok = nodeState{}, false
		return c.state, c.ok
	}
	c.state, c.ok = nodeStateFromStatus(status), true
	return c.state, c.ok
}

func nodeStateFromStatus(status *ipnstate.Status) nodeState {
	state := nodeState{backendState: status.BackendState, authURL: status.AuthURL}
	if status.Self != nil {
		state.dnsName = strings.TrimSuffix(strings.TrimSpace(status.Self.DNSName), ".")
		state.hostname = status.Self.HostName
	}
	return state
}

// fromPrivateNetwork reports whether the client is on this machine, a private LAN
// or container network, or a tailnet. This alone is not enough to show the login
// link (a reverse proxy makes every visitor look local); see setupAuthorized.
func fromPrivateNetwork(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	addr, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	return addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() || tailnetCGNAT.Contains(addr)
}

var nodeLoginPage = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta http-equiv="refresh" content="5">
  <title>Connect Tailor to your tailnet · Tailor</title>
  <style>
    :root { color-scheme: light; font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    * { box-sizing: border-box; }
    body { min-height: 100vh; margin: 0; display: grid; place-items: center; padding: 1.5rem; color: #172126; background: #f3f6f4; }
    main { width: min(100%, 38rem); padding: clamp(1.5rem, 5vw, 3rem); border: 1px solid #d9e1dd; border-radius: 16px; background: #fff; }
    .brand { display: flex; align-items: center; gap: .75rem; margin-bottom: 2.5rem; color: #3a4a44; font-weight: 650; }
    .mark { display: inline-grid; place-items: center; width: 2.5rem; height: 2.5rem; border-radius: 10px; color: #fff; background: #315e4e; font-weight: 750; }
    .status { display: inline-flex; align-items: center; gap: .5rem; margin-bottom: 1rem; color: #315044; font-size: .875rem; font-weight: 650; }
    .status::before { width: .625rem; height: .625rem; border-radius: 50%; background: #b0892f; content: ""; }
    h1 { margin: 0; font-size: clamp(1.75rem, 6vw, 2.25rem); line-height: 1.12; letter-spacing: -0.025em; text-wrap: balance; }
    p { max-width: 60ch; margin: 1rem 0 0; color: #586761; font-size: 1rem; line-height: 1.65; text-wrap: pretty; }
    code { padding: .1rem .3rem; border-radius: 5px; background: #edf2ef; font-size: .9em; }
    .note { margin-top: 1.5rem; padding: .875rem 1rem; border-radius: 10px; color: #315044; background: #edf8f2; font-size: .925rem; }
    a.button { display: inline-flex; margin-top: 2rem; padding: .7rem 1rem; border-radius: 9px; color: #fff; background: #315e4e; font-weight: 650; text-decoration: none; }
    a.button:hover { background: #25483c; }
    a.button:focus-visible { outline: 3px solid #9bc4b5; outline-offset: 3px; }
  </style>
</head>
<body>
  <main>
    <div class="brand"><span class="mark" aria-hidden="true">T</span> Tailor</div>
    {{if eq .State "NeedsMachineAuth"}}
    <div class="status">Waiting for approval</div>
    <h1>Approve Tailor in your tailnet</h1>
    <p>Tailor has logged in, but your tailnet requires new devices to be approved. Approve the <strong>{{.Hostname}}</strong> machine in the Tailscale admin console. This page refreshes on its own.</p>
    {{else if .AuthURL}}
    <div class="status">Not connected</div>
    <h1>Connect Tailor to your tailnet</h1>
    <p>Tailor runs as its own machine on your tailnet. Log in with your Tailscale account to add it — no auth key needed.</p>
    <a class="button" href="{{.AuthURL}}" target="_blank" rel="noopener noreferrer">Log in to Tailscale</a>
    <p class="note">After you log in, this page sends you on. Tailor is then served at <code>https://&lt;hostname&gt;.&lt;your-tailnet&gt;.ts.net/</code> and only answers people on your tailnet.</p>
    {{else if .Hidden}}
    <div class="status">Not connected</div>
    <h1>Connect Tailor to your tailnet</h1>
    <p>Tailor is waiting to be logged in to a tailnet. To keep the login link private, open the <strong>setup link printed in Tailor's logs</strong> (it looks like <code>http://localhost:8080/#setup=…</code>). The Tailscale login link is also printed there.</p>
    {{else if eq .State "NeedsLogin"}}
    <div class="status">Not connected</div>
    <h1>Requesting a login link…</h1>
    <p>Tailor needs to be logged in to your tailnet again (for example after its key expired) and is asking Tailscale for a new login link. This page refreshes on its own.</p>
    {{else}}
    <div class="status">Starting</div>
    <h1>Connecting to your tailnet…</h1>
    <p>Tailor is starting its Tailscale connection. This page refreshes on its own.</p>
    {{end}}
  </main>
  <script nonce="{{.Nonce}}">
    // The setup token arrives in the URL fragment, which browsers never send to
    // servers or proxies. Drop it from the address bar and swap it for a cookie.
    (function () {
      var match = location.hash.match(/(?:^#|&)setup=([0-9a-f]+)/);
      if (!match) return;
      history.replaceState(null, "", location.pathname);
      fetch("/setup/exchange", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: match[1] }),
        credentials: "same-origin"
      }).finally(function () { location.reload(); });
    })();
  </script>
</body>
</html>`))

// setupAuthorized reports whether a request may see the node login link: it needs
// a setup session cookie (obtained by exchanging the token from Tailor's logs)
// and a local or private client. The token is what actually protects the link:
// anyone with it can already read the logs, which contain the Tailscale login URL
// too. The address check alone would let a reverse proxy expose the link.
func setupAuthorized(r *http.Request, gate *SetupGate) bool {
	if gate == nil || !fromPrivateNetwork(r.RemoteAddr) {
		return false
	}
	cookie, err := r.Cookie(setupCookieName)
	return err == nil && gate.validSession(cookie.Value)
}

// writeNodeLogin answers requests while Tailor's own node is not on a tailnet yet:
// an HTML login page for browsers, JSON for API clients.
func writeNodeLogin(w http.ResponseWriter, r *http.Request, state nodeState, gate *SetupGate, allowedHosts []string) {
	// Unauthenticated responses must not be readable through DNS rebinding.
	if !hostAllowed(r.Host, allowedHosts) {
		http.Error(w, "Host not allowed. Set TAILOR_ALLOWED_HOSTS to serve Tailor under this name.", http.StatusMisdirectedRequest)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path == "/setup/exchange" {
		handleSetupExchange(w, r, gate)
		return
	}
	authURL := ""
	hidden := false
	if state.authURL != "" {
		if setupAuthorized(r, gate) {
			authURL = state.authURL
		} else {
			hidden = true
		}
	}
	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/mcp" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "Tailor is not connected to a tailnet yet. Open Tailor in a browser to log it in.",
			"state":   state.backendState,
			"authUrl": authURL,
		})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	nonce := randomHex()
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-"+nonce+"'; connect-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'")
	w.Header().Set("Referrer-Policy", "no-referrer")
	// A normal page (it refreshes itself), so not an error status in the access log.
	w.WriteHeader(http.StatusOK)
	_ = nodeLoginPage.Execute(w, struct {
		State, AuthURL, Hostname, Nonce string
		Hidden                          bool
	}{state.backendState, authURL, state.hostname, nonce, hidden})
}
