package server

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// topologyWebSocketOriginPatterns authorizes browser WebSocket handshakes.
// Tailscale Serve and other reverse proxies forward to loopback while the
// browser Origin still names the public MagicDNS host — the static localhost
// patterns alone reject those connections.
func topologyWebSocketOriginPatterns(r *http.Request) []string {
	patterns := []string{"localhost:*", "127.0.0.1:*", "[::1]:*"}

	if !requestForwardedToLoopback(r) {
		return patterns
	}

	if originHost := originHostname(r.Header.Get("Origin")); originHost != "" {
		patterns = append(patterns, originHost, originHost+":*")
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

func originHostname(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return ""
	}
	return hostWithoutPort(parsed.Host)
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
