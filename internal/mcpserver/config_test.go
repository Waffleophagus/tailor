package mcpserver

import (
	"net/http/httptest"
	"testing"
)

func TestConfigFromEnvDefaultsOff(t *testing.T) {
	t.Setenv("TAILOR_MCP", "")
	t.Setenv("TAILOR_MCP_PATH", "")
	t.Setenv("TAILOR_MCP_TOKEN", "")
	t.Setenv("TAILOR_MCP_READONLY", "")

	cfg := ConfigFromEnv()
	if cfg.Exposure != ExposureOff {
		t.Fatalf("exposure = %q, want %q", cfg.Exposure, ExposureOff)
	}
	if cfg.Path != "/mcp" {
		t.Fatalf("path = %q, want /mcp", cfg.Path)
	}
	if cfg.Enabled() {
		t.Fatal("default MCP config should be disabled")
	}
}

func TestPublicExposureRequiresToken(t *testing.T) {
	t.Setenv("TAILOR_MCP", "public")
	t.Setenv("TAILOR_MCP_TOKEN", "")

	cfg := ConfigFromEnv()
	if cfg.Enabled() {
		t.Fatal("public MCP config without token should be disabled")
	}
}

func TestLocalhostHeaderDoesNotEnableMCP(t *testing.T) {
	req := httptest.NewRequest("POST", "/mcp", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("X-Forwarded-For", "127.0.0.1")
	req.Header.Set("X-Real-IP", "localhost")

	if isLoopbackRequest(req) {
		t.Fatal("loopback check should ignore spoofable forwarded headers")
	}
}

func TestTailnetExposureEnabledWithoutToken(t *testing.T) {
	t.Setenv("TAILOR_MCP", "tailnet")
	t.Setenv("TAILOR_MCP_PATH", "custom-mcp")
	t.Setenv("TAILOR_MCP_TOKEN", "")
	t.Setenv("TAILOR_MCP_READONLY", "true")

	cfg := ConfigFromEnv()
	if cfg.Exposure != ExposureTailnet {
		t.Fatalf("exposure = %q, want %q", cfg.Exposure, ExposureTailnet)
	}
	if !cfg.Enabled() {
		t.Fatal("tailnet MCP config without token should be enabled for tsnet identity auth")
	}
	if cfg.Path != "/custom-mcp" {
		t.Fatalf("path = %q, want /custom-mcp", cfg.Path)
	}
	if !cfg.ReadOnly {
		t.Fatal("readonly should parse true")
	}
}

func TestPublicExposureEnabledWithToken(t *testing.T) {
	t.Setenv("TAILOR_MCP", "public")
	t.Setenv("TAILOR_MCP_PATH", "custom-mcp")
	t.Setenv("TAILOR_MCP_TOKEN", "secret")
	t.Setenv("TAILOR_MCP_READONLY", "true")

	cfg := ConfigFromEnv()
	if !cfg.Enabled() {
		t.Fatal("public MCP config with token should be enabled")
	}
	if cfg.Path != "/custom-mcp" {
		t.Fatalf("path = %q, want /custom-mcp", cfg.Path)
	}
	if !cfg.ReadOnly {
		t.Fatal("readonly should parse true")
	}
}
