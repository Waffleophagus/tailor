package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Waffleophagus/tailor/internal/deploy"
	"github.com/Waffleophagus/tailor/internal/devtailnet"
	tailorlog "github.com/Waffleophagus/tailor/internal/log"
	"github.com/Waffleophagus/tailor/internal/server"
	"github.com/Waffleophagus/tailor/internal/tailserve"
	"tailscale.com/client/local"
	"tailscale.com/tsnet"
)

func main() {
	logger, logCfg, _, err := tailorlog.Setup()
	if err != nil {
		slog.Error("logging setup failed", "error", err)
		os.Exit(1)
	}

	addr := os.Getenv("TAILOR_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	localAPIEndpoint := os.Getenv("TAILOR_LOCALAPI_ENDPOINT")
	if localAPIEndpoint == "" {
		localAPIEndpoint = os.Getenv("TAILOR_LOCALAPI_SOCKET")
	}

	build := "release"
	if devtailnet.Enabled {
		build = "dev"
	}
	deployEnv := deploy.Detect()
	logger.Info("starting tailor",
		"addr", addr,
		"build", build,
		"in_container", deployEnv.InContainer,
		"tailscale_mode", deployEnv.TailscaleMode,
		"has_auth_key", deployEnv.HasAuthKey,
		"wants_host_socket", deployEnv.WantsHostSocket,
		"log_dir", strings.TrimSpace(logCfg.LogDir),
	)

	var tsnetServer *tsnet.Server
	var tsnetLocalClient *local.Client
	if shouldUseTsnet(deployEnv) {
		configureTSNetForceLogin(deployEnv, tsnetStateDir(), logger)
		tsnetServer = newTSNetServer(logger)
		var err error
		tsnetLocalClient, err = tsnetServer.LocalClient()
		if err != nil {
			logger.Error("tsnet startup failed", "error", err)
			os.Exit(1)
		}
		defer tsnetServer.Close()
		localAPIEndpoint = "tsnet embedded"
	}

	handler := server.New(server.Options{
		LocalAPIEndpoint: localAPIEndpoint,
		LocalClient:      tsnetLocalClient,
		WhoIsClient:      tsnetLocalClient,
		TailnetStatus:    tsnetLocalClient,
		TailnetMode:      tsnetServer != nil,
		AppCapability:    os.Getenv("TAILOR_APP_CAPABILITY"),
		Logger:           logger,
	})

	serveMode := tailserve.ParseMode(os.Getenv("TAILOR_TAILSCALE_SERVE"))
	if serveMode != tailserve.ModeOff && tsnetServer == nil {
		servePort, err := tailserve.ParseHTTPSPort(os.Getenv("TAILOR_TAILSCALE_SERVE_PORT"))
		if err != nil {
			logger.Error("invalid TAILOR_TAILSCALE_SERVE_PORT", "error", err)
			os.Exit(1)
		}
		go tailserve.ConfigureWhenReady(context.Background(), tailserve.Options{
			LocalAPIEndpoint: localAPIEndpoint,
			ListenAddr:       addr,
			Mode:             serveMode,
			HTTPSPort:        servePort,
			Logger:           logger,
		})
	}

	errs := make(chan error, 2)

	localSrv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	if tsnetServer != nil {
		go func() {
			certDomain, err := prepareTSNetHTTPS(context.Background(), tsnetServer, tsnetLocalClient, logger)
			if err != nil {
				errs <- fmt.Errorf("tsnet https setup: %w", err)
				return
			}
			ln, err := tsnetServer.ListenTLS("tcp", tsnetListenAddr())
			if err != nil {
				errs <- fmt.Errorf("tsnet listen: %w", err)
				return
			}
			logger.Info("tsnet https server listening",
				"addr", ln.Addr().String(),
				"url", "https://"+certDomain+"/",
			)
			srv := &http.Server{
				Handler:           handler,
				ReadTimeout:       15 * time.Second,
				ReadHeaderTimeout: 5 * time.Second,
				// Streamable MCP may keep a response open while the session is active.
				WriteTimeout: 0,
				IdleTimeout:  60 * time.Second,
			}
			if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errs <- fmt.Errorf("tsnet server: %w", err)
			}
		}()
	}

	go func() {
		if err := localSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- fmt.Errorf("local server: %w", err)
		}
	}()

	if err := <-errs; err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func shouldUseTsnet(env deploy.Environment) bool {
	if env.TailscaleMode == "external" || env.WantsHostSocket {
		return false
	}
	if env.TailscaleMode == "embedded" {
		return true
	}
	return env.HasAuthKey || strings.TrimSpace(os.Getenv("TS_AUTHKEY")) != ""
}

func configureTSNetForceLogin(env deploy.Environment, stateDir string, logger *slog.Logger) {
	if !env.HasAuthKey && strings.TrimSpace(os.Getenv("TS_AUTHKEY")) == "" {
		return
	}
	if strings.TrimSpace(os.Getenv("TSNET_FORCE_LOGIN")) != "" {
		return
	}
	if hasTSNetMachineState(stateDir) {
		return
	}
	if err := os.Setenv("TSNET_FORCE_LOGIN", "1"); err != nil {
		logger.Warn("could not set TSNET_FORCE_LOGIN", "error", err)
		return
	}
	logger.Info("tsnet force login enabled for auth-key deployment")
}

func hasTSNetMachineState(stateDir string) bool {
	stateFile := filepath.Join(stateDir, "tailscaled.state")
	data, err := os.ReadFile(stateFile)
	if err != nil || len(data) == 0 {
		return false
	}
	var state map[string]json.RawMessage
	if err := json.Unmarshal(data, &state); err != nil {
		return false
	}
	machineKey := state["_machinekey"]
	return len(machineKey) > 0 && string(machineKey) != "null"
}

func newTSNetServer(logger *slog.Logger) *tsnet.Server {
	hostname := strings.TrimSpace(os.Getenv("TS_HOSTNAME"))
	if hostname == "" {
		hostname = strings.TrimSpace(os.Getenv("TAILSCALE_HOSTNAME"))
	}
	if hostname == "" {
		hostname = "tailor"
	}
	authKey := strings.TrimSpace(os.Getenv("TS_AUTHKEY"))
	if authKey == "" {
		authKey = strings.TrimSpace(os.Getenv("TAILSCALE_AUTHKEY"))
	}

	return &tsnet.Server{
		Dir:           tsnetStateDir(),
		Hostname:      hostname,
		AuthKey:       authKey,
		AdvertiseTags: advertiseTags(),
		UserLogf: func(format string, args ...any) {
			logger.Info("tsnet", "message", fmt.Sprintf(format, args...))
		},
	}
}

func tsnetStateDir() string {
	stateDir := strings.TrimSpace(os.Getenv("TS_STATE_DIR"))
	if stateDir == "" {
		stateDir = strings.TrimSpace(os.Getenv("TAILSCALE_STATE_DIR"))
	}
	if stateDir == "" {
		stateDir = "/var/lib/tailor-tsnet"
	}
	return stateDir
}

func prepareTSNetHTTPS(ctx context.Context, server *tsnet.Server, client *local.Client, logger *slog.Logger) (string, error) {
	if server == nil || client == nil {
		return "", errors.New("missing tsnet server or local client")
	}
	status, err := server.Up(ctx)
	if err != nil {
		return "", err
	}
	if status.CurrentTailnet == nil || !status.CurrentTailnet.MagicDNSEnabled {
		return "", errors.New("MagicDNS is required for https://<hostname>.<tailnet>.ts.net/; enable MagicDNS in Tailscale DNS settings")
	}
	if len(status.CertDomains) == 0 {
		return "", errors.New("Tailscale HTTPS certificates are not enabled for this tailnet; enable HTTPS certificates in Tailscale DNS settings")
	}
	certDomain := status.CertDomains[0]
	if status.Self != nil && strings.TrimSpace(status.Self.DNSName) != "" {
		selfDNS := strings.TrimSuffix(strings.TrimSpace(status.Self.DNSName), ".")
		for _, domain := range status.CertDomains {
			if domain == selfDNS {
				certDomain = domain
				break
			}
		}
	}
	logger.Info("prewarming tsnet https certificate", "domain", certDomain)
	if _, err := client.GetCertificate(&tls.ClientHelloInfo{ServerName: certDomain}); err != nil {
		return "", fmt.Errorf("prewarm certificate for %s: %w", certDomain, err)
	}
	return certDomain, nil
}

func tsnetListenAddr() string {
	port := strings.TrimSpace(os.Getenv("TAILOR_TSNET_PORT"))
	if port == "" {
		port = "443"
	}
	if strings.Contains(port, ":") {
		return port
	}
	return net.JoinHostPort("", port)
}

func advertiseTags() []string {
	raw := strings.TrimSpace(os.Getenv("TS_ADVERTISE_TAGS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("TAILSCALE_ADVERTISE_TAGS"))
	}
	if raw == "" {
		raw = advertiseTagsFromUpArgs(os.Getenv("TAILSCALE_UP_EXTRA_ARGS"))
	}
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

func advertiseTagsFromUpArgs(raw string) string {
	fields := strings.Fields(raw)
	for i, field := range fields {
		if strings.HasPrefix(field, "--advertise-tags=") {
			return strings.TrimPrefix(field, "--advertise-tags=")
		}
		if field == "--advertise-tags" && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}
