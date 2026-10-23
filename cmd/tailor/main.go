package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Waffleophagus/tailor/internal/deploy"
	"github.com/Waffleophagus/tailor/internal/devtailnet"
	tailorlog "github.com/Waffleophagus/tailor/internal/log"
	"github.com/Waffleophagus/tailor/internal/server"
	"github.com/Waffleophagus/tailor/internal/tailserve"
	"tailscale.com/client/local"
	"tailscale.com/envknob"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/logtail"
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
	closeTSNet := func() {
		if tsnetServer != nil {
			if err := tsnetServer.Close(); err != nil {
				logger.Warn("tsnet close failed", "error", err)
			}
			tsnetServer = nil
		}
	}
	exit := func(code int) {
		closeTSNet()
		os.Exit(code)
	}

	var tsnetLocalClient *local.Client
	setupToken := ""
	if shouldUseTsnet(deployEnv) {
		setupToken = newSetupToken()
		stateDir := tsnetStateDir(deployEnv)
		hardenTSNetLogging(logger)
		configureTSNetForceLogin(deployEnv, stateDir, logger)
		tsnetServer = newTSNetServer(logger, stateDir)
		var err error
		tsnetLocalClient, err = tsnetServer.LocalClient()
		if err != nil {
			logger.Error("tsnet startup failed", "error", err, "state_dir", stateDir)
			exit(1)
		}
		go reportNodeLogin(context.Background(), tsnetLocalClient, setupPageURL(addr, setupToken), logger)
		localAPIEndpoint = "tsnet embedded"
	}

	handler := server.New(server.Options{
		LocalAPIEndpoint: localAPIEndpoint,
		LocalClient:      tsnetLocalClient,
		WhoIsClient:      tsnetLocalClient,
		TailnetStatus:    tsnetLocalClient,
		TailnetMode:      tsnetServer != nil,
		AppCapability:    os.Getenv("TAILOR_APP_CAPABILITY"),
		SetupToken:       setupToken,
		TailnetHTTPSPort: strings.TrimPrefix(listenPort(tsnetListenAddr()), ":"),
		Logger:           logger,
	})

	serveMode := tailserve.ParseMode(os.Getenv("TAILOR_TAILSCALE_SERVE"))
	if serveMode != tailserve.ModeOff && tsnetServer == nil {
		servePort, err := tailserve.ParseHTTPSPort(os.Getenv("TAILOR_TAILSCALE_SERVE_PORT"))
		if err != nil {
			logger.Error("invalid TAILOR_TAILSCALE_SERVE_PORT", "error", err)
			exit(1)
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

	tsnetSrv := &http.Server{
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		// Streamable MCP may keep a response open while the session is active.
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
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
			if err := tsnetSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errs <- fmt.Errorf("tsnet server: %w", err)
			}
		}()
	}

	go func() {
		if err := localSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs <- fmt.Errorf("local server: %w", err)
		}
	}()

	// Stop cleanly on docker stop / Ctrl-C so in-flight requests finish and the
	// embedded tsnet node shuts down instead of being killed mid-write.
	signals, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	select {
	case err := <-errs:
		logger.Error("server failed", "error", err)
		exit(1)
	case <-signals.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		// Drain both servers in parallel within the one grace period, then close tsnet.
		var drained sync.WaitGroup
		for _, srv := range []*http.Server{localSrv, tsnetSrv} {
			drained.Go(func() {
				if err := srv.Shutdown(shutdownCtx); err != nil {
					logger.Warn("http shutdown incomplete", "error", err)
				}
			})
		}
		drained.Wait()
		closeTSNet()
	}
}

const shutdownTimeout = 10 * time.Second

func shouldUseTsnet(env deploy.Environment) bool {
	// Tailor joins the tailnet as its own machine unless told to use the host's
	// tailscaled. Without an auth key, it is logged in from the setup page.
	return env.TailscaleMode != "external" && !env.WantsHostSocket
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

func newTSNetServer(logger *slog.Logger, stateDir string) *tsnet.Server {
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
		Dir:           stateDir,
		Hostname:      hostname,
		AuthKey:       authKey,
		AdvertiseTags: advertiseTags(),
		UserLogf: func(format string, args ...any) {
			logger.Info("tsnet", "message", fmt.Sprintf(format, args...))
		},
	}
}

// containerStateDir is the tsnet state directory in containers (a volume in compose).
const containerStateDir = "/var/lib/tailor-tsnet"

// tsnetStateDir returns where the embedded node keeps its identity: TS_STATE_DIR or
// TAILSCALE_STATE_DIR, else /var/lib/tailor-tsnet in containers (or wherever a node
// already lives there), else the user's config directory for desktop binaries.
func tsnetStateDir(env deploy.Environment) string {
	for _, name := range []string{"TS_STATE_DIR", "TAILSCALE_STATE_DIR"} {
		if dir := strings.TrimSpace(os.Getenv(name)); dir != "" {
			return dir
		}
	}
	if env.InContainer || hasTSNetMachineState(containerStateDir) {
		return containerStateDir
	}
	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		return filepath.Join(configDir, "tailor", "tsnet")
	}
	return containerStateDir
}

// hardenTSNetLogging must run before the tsnet server starts. tsnet otherwise
// uploads its backend logs to log.tailscale.com, and TS_DEBUG_REGISTER would log
// registration requests that include the auth key.
func hardenTSNetLogging(logger *slog.Logger) {
	logtail.Disable()
	if os.Getenv("TS_DEBUG_REGISTER") != "" {
		logger.Warn("ignoring TS_DEBUG_REGISTER; it would log registration requests including the auth key")
		envknob.Setenv("TS_DEBUG_REGISTER", "")
	}
}

// nodeLoginClient is the part of the tsnet LocalAPI client reportNodeLogin uses.
type nodeLoginClient interface {
	StatusWithoutPeers(context.Context) (*ipnstate.Status, error)
	StartLoginInteractive(context.Context) error
}

// Recovery timing for reportNodeLogin; tests shorten these.
var (
	nodeLoginPollInterval = 3 * time.Second
	// A node that needs login but has no login URL for this long (e.g. after its
	// key expired) gets a fresh interactive login started.
	nodeLoginURLWait   = 10 * time.Second
	nodeLoginRetryWait = time.Minute
)

// reportNodeLogin runs for the life of the process: it logs the setup page (with
// its token) and Tailscale login URL whenever the node needs login, starts a new
// interactive login when the node needs one but has no URL (key expiry), and
// warns about key expiry each time the node comes up.
func reportNodeLogin(ctx context.Context, client nodeLoginClient, setupURL string, logger *slog.Logger) {
	lastState, lastURL := "", ""
	var needsURLSince, lastLoginStart time.Time
	ticker := time.NewTicker(nodeLoginPollInterval)
	defer ticker.Stop()
	for {
		if status, err := client.StatusWithoutPeers(ctx); err == nil && status != nil {
			state := status.BackendState
			switch {
			case state == "Running":
				if lastState != "Running" {
					warnIfNodeKeyExpires(status, logger)
				}
				lastURL = ""
			case state == "NeedsMachineAuth":
				if lastState != state {
					logger.Warn("tailor is logged in but waiting for device approval in the Tailscale admin console")
				}
			case status.AuthURL != "":
				if status.AuthURL != lastURL {
					logger.Warn("tailor is not connected to a tailnet yet; open the setup page to log it in with your Tailscale account",
						"setup_page", setupURL,
						"login_url", status.AuthURL,
					)
					lastURL = status.AuthURL
				}
				needsURLSince = time.Time{}
			case state == "NeedsLogin":
				now := time.Now()
				if needsURLSince.IsZero() {
					needsURLSince = now
				}
				if now.Sub(needsURLSince) >= nodeLoginURLWait && now.Sub(lastLoginStart) >= nodeLoginRetryWait {
					lastLoginStart = now
					logger.Warn("tailor needs to log in to its tailnet again; requesting a new login link")
					if err := client.StartLoginInteractive(ctx); err != nil {
						logger.Warn("could not start tailnet login", "error", err)
					}
				}
			}
			if state != "NeedsLogin" {
				needsURLSince = time.Time{}
			}
			lastState = state
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// newSetupToken returns the random token that unlocks the node login link on the
// setup page. It is only ever printed to Tailor's own logs.
func newSetupToken() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic("setup token generation failed: " + err.Error())
	}
	return hex.EncodeToString(buf[:])
}

func setupPageURL(addr, token string) string {
	return "http://localhost" + listenPort(addr) + "/?setup=" + token
}

func warnIfNodeKeyExpires(status *ipnstate.Status, logger *slog.Logger) {
	self := status.Self
	if self == nil || self.KeyExpiry == nil || (self.Tags != nil && self.Tags.Len() > 0) {
		return
	}
	logger.Warn("tailor's node key expires; tag the node (e.g. tag:tailor-acl-service) or disable key expiry for it in the Tailscale admin console",
		"expires", self.KeyExpiry.Format(time.RFC3339),
	)
}

func listenPort(addr string) string {
	if _, port, err := net.SplitHostPort(addr); err == nil && port != "" {
		return ":" + port
	}
	return ""
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
		return "", errors.New("tailscale HTTPS certificates are not enabled for this tailnet; enable HTTPS certificates in Tailscale DNS settings")
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
