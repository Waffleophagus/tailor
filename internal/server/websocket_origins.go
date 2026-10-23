package server

import (
	"net"
	"net/http"
	"strings"
)

// topologyWebSocketOriginPatterns authorizes browser WebSocket handshakes.
// Reverse proxies (including Tailscale Serve) can forward to loopback while the
// browser Origin still names the public host, so the proxy's X-Forwarded-Host is
// trusted as well. The request's own Origin header must never be trusted here:
// any website could then open a socket to a Tailor listening on localhost.
func topologyWebSocketOriginPatterns(r *http.Request) []string {
	patterns := []string{"localhost:*", "127.0.0.1:*", "[::1]:*"}

	if !requestForwardedToLoopback(r) {
		return patterns
	}

	if host := hostWithoutPort(forwardedRequestHost(r)); host != "" {
		patterns = append(patterns, host, host+":*")
	}

	return patterns
}

func requestForwardedToLoopback(r *http.Request) bool {
	host := hostWithoutPort(r.Host)
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") || host == "127.0.0.1" || host == "::1" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func forwardedRequestHost(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return r.Host
}

func hostWithoutPort(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
