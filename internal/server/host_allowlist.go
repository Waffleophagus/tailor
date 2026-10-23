package server

import (
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
)

// localOnlySuffixes are domains that cannot be registered publicly, so a remote
// website cannot make them resolve to this machine.
var localOnlySuffixes = []string{".localhost", ".local", ".lan", ".home", ".home.arpa", ".internal", ".localdomain", ".ts.net"}

// HostAllowlistMiddleware blocks DNS rebinding when Tailor has no tailnet identity
// layer (local and external modes): a malicious site could otherwise point its own
// domain at 127.0.0.1 and drive Tailor from the victim's browser as same-origin.
// Allowed hosts are IP literals, localhost, single-label names, local-only domains,
// Tailscale MagicDNS names (*.ts.net) and anything in TAILOR_ALLOWED_HOSTS
// (comma-separated; a leading dot allows a whole domain).
func HostAllowlistMiddleware(logger *slog.Logger, extra []string, next http.Handler) http.Handler {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hostAllowed(r.Host, extra) {
			logger.Warn("request rejected: host not allowed",
				"host", r.Host,
				"path", r.URL.Path,
				"hint", "add it to TAILOR_ALLOWED_HOSTS if Tailor is served under this name",
				"request_id", RequestIDFromContext(r.Context()),
			)
			http.Error(w, "Host not allowed. Set TAILOR_ALLOWED_HOSTS to serve Tailor under this name.", http.StatusMisdirectedRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// AllowedHostsFromEnv parses TAILOR_ALLOWED_HOSTS.
func AllowedHostsFromEnv() []string {
	var hosts []string
	for _, host := range strings.Split(os.Getenv("TAILOR_ALLOWED_HOSTS"), ",") {
		if host = strings.ToLower(strings.TrimSpace(host)); host != "" {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

func hostAllowed(hostport string, extra []string) bool {
	host := strings.ToLower(strings.TrimSuffix(hostWithoutPort(hostport), "."))
	if host == "" {
		return true // HTTP/1.0 without Host; not reachable through a browser rebinding attack
	}
	if net.ParseIP(strings.Trim(host, "[]")) != nil {
		return true
	}
	if host == "localhost" || !strings.Contains(host, ".") {
		return true
	}
	for _, suffix := range localOnlySuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	for _, allowed := range extra {
		if host == allowed || (strings.HasPrefix(allowed, ".") && strings.HasSuffix(host, allowed)) {
			return true
		}
	}
	return false
}
