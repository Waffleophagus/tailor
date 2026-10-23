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
func (s nodeState) tailorURL() string {
	if s.dnsName == "" {
		return ""
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
// or container network, or a tailnet. Only they are shown the node login link, so
// an internet client cannot enroll Tailor into a tailnet of its choosing.
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
    <p>Tailor is waiting to be logged in to a tailnet. For safety the login link is only shown on this machine or your local network. It is also printed in Tailor's logs.</p>
    {{else}}
    <div class="status">Starting</div>
    <h1>Connecting to your tailnet…</h1>
    <p>Tailor is starting its Tailscale connection. This page refreshes on its own.</p>
    {{end}}
  </main>
</body>
</html>`))

// writeNodeLogin answers requests while Tailor's own node is not on a tailnet yet:
// an HTML login page for browsers, JSON for API clients.
func writeNodeLogin(w http.ResponseWriter, r *http.Request, state nodeState, hostname string) {
	authURL := ""
	hidden := false
	if state.authURL != "" {
		if fromPrivateNetwork(r.RemoteAddr) {
			authURL = state.authURL
		} else {
			hidden = true
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/mcp" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error":   "Tailor is not connected to a tailnet yet. Open Tailor in a browser to log it in.",
			"state":   state.backendState,
			"authUrl": authURL,
		})
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'")
	// A normal page (it refreshes itself), so not an error status in the access log.
	w.WriteHeader(http.StatusOK)
	_ = nodeLoginPage.Execute(w, struct {
		State, AuthURL, Hostname string
		Hidden                   bool
	}{state.backendState, authURL, hostname, hidden})
}
