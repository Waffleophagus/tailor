package tailserve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"tailscale.com/client/local"
	"tailscale.com/ipn"
	"tailscale.com/ipn/ipnstate"
)

const defaultHTTPSPort = 443
const httpRedirectPort = 80

// Mode controls whether Tailor configures Tailscale Serve on startup.
type Mode string

const (
	ModeAuto Mode = "auto"
	ModeOn   Mode = "on"
	ModeOff  Mode = "off"
)

// Options configures automatic Tailscale Serve for the Tailor HTTP server.
type Options struct {
	LocalAPIEndpoint string
	ListenAddr       string
	Mode             Mode
	HTTPSPort        uint16
	Logger           *slog.Logger
}

// ConfigureWhenReady waits for tailscaled and an active tailnet session, then
// exposes Tailor over HTTPS via Tailscale Serve (MagicDNS name, port 443).
func ConfigureWhenReady(ctx context.Context, opts Options) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	mode := opts.Mode
	if mode == "" {
		mode = ModeAuto
	}
	if mode == ModeOff {
		return
	}

	httpsPort := opts.HTTPSPort
	if httpsPort == 0 {
		httpsPort = defaultHTTPSPort
	}

	proxyURL, err := listenAddrToProxyURL(opts.ListenAddr)
	if err != nil {
		if mode == ModeOn {
			logger.Error("tailscale serve disabled: invalid listen address", "addr", opts.ListenAddr, "error", err)
		}
		return
	}

	lc := &local.Client{Socket: opts.LocalAPIEndpoint}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if err := tryConfigure(ctx, lc, proxyURL, httpsPort, logger); err == nil {
			return
		} else if mode == ModeOn {
			logger.Warn("tailscale serve not ready yet", "error", err)
		}

		select {
		case <-ctx.Done():
			if mode == ModeOn {
				logger.Error("tailscale serve setup cancelled", "error", ctx.Err())
			}
			return
		case <-ticker.C:
		}
	}
}

func tryConfigure(ctx context.Context, lc *local.Client, proxyURL string, httpsPort uint16, logger *slog.Logger) error {
	st, err := lc.StatusWithoutPeers(ctx)
	if err != nil {
		return fmt.Errorf("localapi status: %w", err)
	}
	if st == nil || st.BackendState != "Running" {
		return errors.New("tailnet not running")
	}
	if st.Self == nil || strings.TrimSpace(st.Self.DNSName) == "" {
		return errors.New("tailnet DNS name unavailable")
	}

	dnsName := strings.TrimSuffix(st.Self.DNSName, ".")
	mds := ""
	if st.CurrentTailnet != nil {
		mds = st.CurrentTailnet.MagicDNSSuffix
	}

	sc, err := lc.GetServeConfig(ctx)
	if err != nil {
		return fmt.Errorf("get serve config: %w", err)
	}
	if sc == nil {
		sc = new(ipn.ServeConfig)
	}

	redirectURL := httpsRedirectURL(dnsName, httpsPort)
	if alreadyConfigured(sc, dnsName, httpsPort, proxyURL, redirectURL) {
		logger.Info("tailscale serve already configured",
			"url", fmt.Sprintf("https://%s/", dnsName),
			"proxy", proxyURL,
		)
		return nil
	}

	if portInUseByOther(sc, dnsName, httpsPort, proxyURL, mds) {
		return fmt.Errorf("port %d already has a different tailscale serve config", httpsPort)
	}
	if httpRedirectInUseByOther(sc, dnsName, redirectURL, mds) {
		return fmt.Errorf("port %d already has a different tailscale serve config", httpRedirectPort)
	}

	handler := &ipn.HTTPHandler{Proxy: proxyURL}
	sc.SetWebHandler(handler, dnsName, httpsPort, "/", true, mds)
	sc.SetWebHandler(&ipn.HTTPHandler{Redirect: redirectURL}, dnsName, httpRedirectPort, "/", false, mds)

	if err := lc.SetServeConfig(ctx, sc); err != nil {
		return fmt.Errorf("set serve config: %w", err)
	}

	logger.Info("tailscale serve enabled",
		"url", fmt.Sprintf("https://%s/", dnsName),
		"proxy", proxyURL,
	)
	return nil
}

func alreadyConfigured(sc *ipn.ServeConfig, dnsName string, port uint16, proxyURL, redirectURL string) bool {
	if sc == nil {
		return false
	}
	handler := sc.GetWebHandler("", ipn.HostPort(net.JoinHostPort(dnsName, strconv.Itoa(int(port)))), "/")
	redirect := sc.GetWebHandler("", ipn.HostPort(net.JoinHostPort(dnsName, strconv.Itoa(httpRedirectPort))), "/")
	return handler != nil && handler.Proxy == proxyURL && redirect != nil && redirect.Redirect == redirectURL
}

func portInUseByOther(sc *ipn.ServeConfig, dnsName string, port uint16, proxyURL, mds string) bool {
	if sc == nil {
		return false
	}
	hp := ipn.HostPort(net.JoinHostPort(dnsName, strconv.Itoa(int(port))))
	if web, ok := sc.Web[hp]; ok {
		for _, handler := range web.Handlers {
			if handler == nil {
				continue
			}
			if handler.Proxy != "" {
				if !repairableLocalProxy(handler.Proxy) {
					return true
				}
				continue
			}
			if handler.Redirect != "" || handler.Path != "" || handler.Text != "" {
				return true
			}
		}
	}
	_ = mds
	if tcp, ok := sc.TCP[port]; ok && tcp != nil {
		if tcp.TCPForward != "" {
			return true
		}
		if tcp.HTTPS || tcp.HTTP {
			handler := sc.GetWebHandler("", hp, "/")
			if handler == nil || !repairableLocalProxy(handler.Proxy) {
				return true
			}
		}
	}
	return false
}

func httpRedirectInUseByOther(sc *ipn.ServeConfig, dnsName, redirectURL, mds string) bool {
	if sc == nil {
		return false
	}
	hp := ipn.HostPort(net.JoinHostPort(dnsName, strconv.Itoa(httpRedirectPort)))
	if web, ok := sc.Web[hp]; ok {
		for _, handler := range web.Handlers {
			if handler == nil {
				continue
			}
			if handler.Redirect != "" && handler.Redirect != redirectURL {
				return true
			}
			if handler.Redirect == redirectURL {
				continue
			}
			if !repairableLocalProxy(handler.Proxy) {
				return true
			}
		}
	}
	_ = mds
	if tcp, ok := sc.TCP[httpRedirectPort]; ok && tcp != nil {
		if tcp.TCPForward != "" || tcp.HTTPS {
			return true
		}
		if tcp.HTTP {
			handler := sc.GetWebHandler("", hp, "/")
			if handler == nil {
				return true
			}
			if handler.Redirect != "" && handler.Redirect != redirectURL {
				return true
			}
			if handler.Redirect == redirectURL {
				return false
			}
			if !repairableLocalProxy(handler.Proxy) {
				return true
			}
		}
	}
	return false
}

func httpsRedirectURL(dnsName string, httpsPort uint16) string {
	host := dnsName
	if httpsPort != defaultHTTPSPort {
		host = net.JoinHostPort(dnsName, strconv.Itoa(int(httpsPort)))
	}
	return "308:https://" + host + "${REQUEST_URI}"
}

func repairableLocalProxy(rawURL string) bool {
	hostPort := strings.TrimPrefix(rawURL, "http://")
	host, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		host = hostPort
		if strings.HasPrefix(hostPort, "[") && strings.HasSuffix(hostPort, "]") {
			host = strings.TrimPrefix(strings.TrimSuffix(hostPort, "]"), "[")
		}
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return strings.EqualFold(host, "localhost")
}

func listenAddrToProxyURL(listenAddr string) (string, error) {
	addr := strings.TrimSpace(listenAddr)
	if addr == "" {
		addr = ":8080"
	}

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	if port == "" {
		return "", errors.New("listen address missing port")
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}

	return "http://" + net.JoinHostPort(host, port), nil
}

// ParseMode reads TAILOR_TAILSCALE_SERVE.
func ParseMode(raw string) Mode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "auto":
		return ModeAuto
	case "1", "true", "on", "yes":
		return ModeOn
	case "0", "false", "off", "no":
		return ModeOff
	default:
		return ModeAuto
	}
}

// ParseHTTPSPort reads TAILOR_TAILSCALE_SERVE_PORT.
func ParseHTTPSPort(raw string) (uint16, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultHTTPSPort, nil
	}
	port, err := strconv.ParseUint(raw, 10, 16)
	if err != nil {
		return 0, err
	}
	if port == 0 {
		return 0, errors.New("port must be greater than zero")
	}
	return uint16(port), nil
}

// StatusFromIPN is exported for tests.
func StatusFromIPN(st *ipnstate.Status) (dnsName, mds string, running bool) {
	if st == nil {
		return "", "", false
	}
	if st.BackendState != "Running" || st.Self == nil {
		return "", "", false
	}
	dnsName = strings.TrimSuffix(st.Self.DNSName, ".")
	if st.CurrentTailnet != nil {
		mds = st.CurrentTailnet.MagicDNSSuffix
	}
	return dnsName, mds, dnsName != ""
}
