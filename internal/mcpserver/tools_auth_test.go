package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/Waffleophagus/tailor/internal/authz"
	"github.com/Waffleophagus/tailor/internal/tailorcore"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestViewerMCPToolPermissions(t *testing.T) {
	session := connectTestClient(t, authz.WithIdentity(context.Background(), authz.TailnetIdentity{Role: authz.RoleViewer}))

	assertToolSucceeds(t, session, "tailor_acl_reference_index", nil)
	assertToolDenied(t, session, "tailor_get_policy", "view-only")
	assertToolDenied(t, session, "tailor_get_policy_map", "view-only")
	assertToolDenied(t, session, "tailor_evaluate_policy_draft", "view-only")
	assertToolDenied(t, session, "tailor_stage_policy_draft", "view-only")
}

func TestBootstrapSessionNeverAuthorizesMCPWrite(t *testing.T) {
	ctx := authz.WithIdentity(context.Background(), authz.TailnetIdentity{Role: authz.RoleViewer})
	ctx = authz.WithBootstrap(ctx)
	session := connectTestClient(t, ctx)

	assertToolDenied(t, session, "tailor_stage_policy_draft", "view-only")
}

func TestFullAccessPassesMCPAuthorizationGates(t *testing.T) {
	session := connectTestClient(t, authz.WithIdentity(context.Background(), authz.TailnetIdentity{Role: authz.RoleFull}))

	assertToolNotDeniedAsViewOnly(t, session, "tailor_get_policy", nil)
	assertToolNotDeniedAsViewOnly(t, session, "tailor_evaluate_policy_draft", map[string]any{"hujson": "{}"})
	assertToolNotDeniedAsViewOnly(t, session, "tailor_stage_policy_draft", map[string]any{"hujson": "{}"})
}

func connectTestClient(t *testing.T, serverContext context.Context) *mcp.ClientSession {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "tailor-test", Version: "test"}, nil)
	registerTools(server, tailorcore.New(tailorcore.Options{}), Config{})
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(serverContext, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })
	return clientSession
}

func assertToolSucceeds(t *testing.T, session *mcp.ClientSession, name string, arguments map[string]any) {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("%s returned tool error: %s", name, toolText(result))
	}
}

func assertToolDenied(t *testing.T, session *mcp.ClientSession, name, message string) {
	t.Helper()
	arguments := map[string]any{}
	if name == "tailor_evaluate_policy_draft" || name == "tailor_stage_policy_draft" {
		arguments["hujson"] = "{}"
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError || !strings.Contains(toolText(result), message) {
		t.Fatalf("%s result = isError:%v content:%q, want error containing %q", name, result.IsError, toolText(result), message)
	}
}

func assertToolNotDeniedAsViewOnly(t *testing.T, session *mcp.ClientSession, name string, arguments map[string]any) {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(toolText(result), "view-only") {
		t.Fatalf("%s was denied by the full-access authorization gate: %s", name, toolText(result))
	}
}

func toolText(result *mcp.CallToolResult) string {
	var text strings.Builder
	for _, content := range result.Content {
		if item, ok := content.(*mcp.TextContent); ok {
			text.WriteString(item.Text)
		}
	}
	return text.String()
}
