package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/Waffleophagus/tailor/internal/server"
)

func main() {
	addr := os.Getenv("TAILOR_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	socketPath := os.Getenv("TAILOR_LOCALAPI_SOCKET")

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.New(server.Options{LocalAPISocketPath: socketPath}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("starting tailor", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}
