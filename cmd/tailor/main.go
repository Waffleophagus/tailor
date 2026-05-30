package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Waffleophagus/tailor/internal/deploy"
	"github.com/Waffleophagus/tailor/internal/devtailnet"
	tailorlog "github.com/Waffleophagus/tailor/internal/log"
	"github.com/Waffleophagus/tailor/internal/server"
	"github.com/Waffleophagus/tailor/internal/tailserve"
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

	handler := server.New(server.Options{
		LocalAPIEndpoint: localAPIEndpoint,
		Logger:           logger,
	})

	serveMode := tailserve.ParseMode(os.Getenv("TAILOR_TAILSCALE_SERVE"))
	if serveMode != tailserve.ModeOff {
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

	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}
