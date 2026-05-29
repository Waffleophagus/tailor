package log

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		"":        slog.LevelInfo,
		"unknown": slog.LevelInfo,
	}
	for raw, want := range cases {
		if got := parseLevel(raw); got != want {
			t.Errorf("parseLevel(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestSetupStdoutOnly(t *testing.T) {
	t.Setenv("TAILOR_LOG_DIR", "")
	t.Setenv("TAILOR_LOG_FORMAT", "text")
	t.Setenv("TAILOR_LOG_LEVEL", "info")

	logger, cfg, err := Setup()
	if err != nil {
		t.Fatal(err)
	}
	if logger == nil {
		t.Fatal("expected logger")
	}
	if cfg.LogDir != "" {
		t.Fatalf("log dir = %q, want empty", cfg.LogDir)
	}
}

func TestSetupFileLogging(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TAILOR_LOG_DIR", dir)
	t.Setenv("TAILOR_LOG_FORMAT", "json")
	t.Setenv("TAILOR_LOG_LEVEL", "debug")

	logger, cfg, err := Setup()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogDir != dir {
		t.Fatalf("log dir = %q, want %q", cfg.LogDir, dir)
	}

	logger.Info("setup test message")

	path := filepath.Join(dir, defaultLogFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "setup test message") {
		t.Fatalf("log file missing message: %q", string(data))
	}
}

func TestSetupInvalidFormat(t *testing.T) {
	t.Setenv("TAILOR_LOG_DIR", "")
	t.Setenv("TAILOR_LOG_FORMAT", "xml")

	_, _, err := Setup()
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}

func TestSetupLogDirFallback(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TAILOR_LOG_DIR", filepath.Join(blocker, "logs"))
	t.Setenv("TAILOR_LOG_FORMAT", "text")
	t.Setenv("TAILOR_LOG_LEVEL", "info")

	logger, _, err := Setup()
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("fallback test")
}
