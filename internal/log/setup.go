package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/DeRuina/timberjack"
	"github.com/Waffleophagus/tailor/internal/deploy"
)

const defaultLogFile = "tailor.log"

// Config holds resolved logging settings from the environment.
type Config struct {
	Level      slog.Level
	Format     string
	LogDir     string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
}

// Setup configures the process-wide default logger and returns it with the resolved config.
// The cleanup function closes any open log file; call it before removing the log directory
// (required on Windows). Process exit does not need cleanup.
func Setup() (*slog.Logger, Config, func(), error) {
	cfg := Config{
		Level:      parseLevel(envOr("TAILOR_LOG_LEVEL", "info")),
		Format:     strings.ToLower(strings.TrimSpace(envOr("TAILOR_LOG_FORMAT", "auto"))),
		LogDir:     strings.TrimSpace(os.Getenv("TAILOR_LOG_DIR")),
		MaxSizeMB:  envInt("TAILOR_LOG_MAX_SIZE_MB", 10),
		MaxBackups: envInt("TAILOR_LOG_MAX_BACKUPS", 5),
		MaxAgeDays: envInt("TAILOR_LOG_MAX_AGE_DAYS", 30),
	}

	format := cfg.Format
	if format == "auto" {
		if deploy.Detect().InContainer {
			format = "json"
		} else {
			format = "text"
		}
	}

	writer, fileEnabled, fileLogger, err := logWriter(cfg)
	if err != nil {
		return nil, cfg, nil, err
	}
	cleanup := func() {}
	if fileLogger != nil {
		cleanup = func() { _ = fileLogger.Close() }
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: cfg.Level}
	switch format {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "text":
		handler = slog.NewTextHandler(writer, opts)
	default:
		return nil, cfg, nil, fmt.Errorf("unknown TAILOR_LOG_FORMAT %q (want text, json, or auto)", cfg.Format)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	if cfg.LogDir != "" && !fileEnabled {
		logger.Warn("file logging disabled; continuing with stdout only", "log_dir", cfg.LogDir)
	} else if fileEnabled {
		logger.Info("file logging enabled", "log_dir", cfg.LogDir, "file", defaultLogFile)
	}

	return logger, cfg, cleanup, nil
}

func logWriter(cfg Config) (io.Writer, bool, *timberjack.Logger, error) {
	if cfg.LogDir == "" {
		return os.Stdout, false, nil, nil
	}

	if err := os.MkdirAll(cfg.LogDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "tailor: create log dir %q: %v (continuing with stdout only)\n", cfg.LogDir, err)
		return os.Stdout, false, nil, nil
	}

	fileLogger := &timberjack.Logger{
		Filename:   filepath.Join(cfg.LogDir, defaultLogFile),
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
	}

	return io.MultiWriter(os.Stdout, fileLogger), true, fileLogger, nil
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}
