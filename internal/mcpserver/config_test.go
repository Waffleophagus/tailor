package mcpserver

import "testing"

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

func TestTailnetExposureEnabledWithToken(t *testing.T) {
	t.Setenv("TAILOR_MCP", "tailnet")
	t.Setenv("TAILOR_MCP_PATH", "custom-mcp")
	t.Setenv("TAILOR_MCP_TOKEN", "secret")
	t.Setenv("TAILOR_MCP_READONLY", "true")

	cfg := ConfigFromEnv()
	if !cfg.Enabled() {
		t.Fatal("tailnet MCP config with token should be enabled")
	}
	if cfg.Path != "/custom-mcp" {
		t.Fatalf("path = %q, want /custom-mcp", cfg.Path)
	}
	if !cfg.ReadOnly {
		t.Fatal("readonly should parse true")
	}
}
