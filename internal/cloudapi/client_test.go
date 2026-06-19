package cloudapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAuthenticateWithAPIKeyFetchesPolicy(t *testing.T) {
	var sawPolicyRequest bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/tailnet/example.com/acl":
			sawPolicyRequest = true
			if got := r.Header.Get("Authorization"); got != "Basic dHNrZXktYXBpLXRlc3Q6" {
				t.Fatalf("Authorization = %q", got)
			}
			_, _ = w.Write([]byte("{\n  // kept\n  \"acls\": []\n}\n"))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))

	status, err := client.Authenticate(context.Background(), AuthRequest{
		Tailnet: "example.com",
		APIKey:  "tskey-api-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawPolicyRequest {
		t.Fatal("expected policy request")
	}
	if !status.Authenticated || !status.HasPolicy || status.Tailnet != "example.com" {
		t.Fatalf("unexpected status: %#v", status)
	}

	policy, err := client.Policy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(policy, "// kept") {
		t.Fatalf("policy did not preserve HuJSON text: %q", policy)
	}
}

func TestAuthenticateReturnsSanitizedAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"invalid API key"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	_, err := client.Authenticate(context.Background(), AuthRequest{
		Tailnet: "example.com",
		APIKey:  "tskey-api-secret",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "tskey-api-secret") {
		t.Fatalf("error leaked secret: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("error did not include status: %q", err.Error())
	}
}

func TestFetchPolicyEscapesTailnet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/api/v2/tailnet/"+url.PathEscape("example.com@github")+"/acl" {
			t.Fatalf("path = %s", r.URL.EscapedPath())
		}
		_, _ = w.Write([]byte(`{"acls":[]}`))
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	policy, err := client.fetchPolicy(context.Background(), "example.com@github", "token")
	if err != nil {
		t.Fatal(err)
	}
	if policy == "" {
		t.Fatal("expected policy")
	}
}

func TestAuthenticateRequiresAPIKeyPrefix(t *testing.T) {
	client := New()
	_, err := client.Authenticate(context.Background(), AuthRequest{
		Tailnet: "example.com",
		APIKey:  "tskey-auth-not-right",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "tskey-api-") {
		t.Fatalf("unexpected error: %q", err.Error())
	}
}

func TestValidateAndSavePolicyUsesExplicitDraftAndRefreshesPolicy(t *testing.T) {
	var validated string
	var saved string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/tailnet/example.com/acl":
			if r.Method == http.MethodGet {
				if saved != "" {
					_, _ = w.Write([]byte(saved))
					return
				}
				_, _ = w.Write([]byte(`{"acls":[]}`))
				return
			}
			if r.Method == http.MethodPost {
				body, _ := io.ReadAll(r.Body)
				saved = string(body)
				_, _ = w.Write([]byte(saved))
				return
			}
			t.Fatalf("unexpected method %s", r.Method)
		case "/api/v2/tailnet/example.com/acl/validate":
			body, _ := io.ReadAll(r.Body)
			validated = string(body)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	if _, err := client.Authenticate(context.Background(), AuthRequest{Tailnet: "example.com", APIKey: "tskey-api-test"}); err != nil {
		t.Fatal(err)
	}
	draft := `{"acls":[{"action":"accept","src":["*"],"dst":["*:*"]}]}`
	if err := client.ValidatePolicy(context.Background(), draft); err != nil {
		t.Fatal(err)
	}
	if validated != draft {
		t.Fatalf("validated = %q, want draft", validated)
	}
	refreshed, err := client.SavePolicy(context.Background(), draft)
	if err != nil {
		t.Fatal(err)
	}
	if saved != draft || refreshed != draft {
		t.Fatalf("save did not use/refresh draft: saved=%q refreshed=%q", saved, refreshed)
	}
	if _, err := client.SavePolicy(context.Background(), ""); err == nil {
		t.Fatal("expected empty explicit draft save to fail")
	}
}

func TestDevicesUsersPostureAndVIPServicesFetchAuthenticatedMetadata(t *testing.T) {
	var sawDevicesQuery string
	var sawUsersQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v2/tailnet/example.com/acl":
			_, _ = w.Write([]byte(`{"acls":[]}`))
		case "/api/v2/tailnet/example.com/devices":
			sawDevicesQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"devices":[{"addresses":["100.64.0.1"],"nodeId":"n1","user":"alice@example.com","clientVersion":"1.94.2","os":"linux","isExternal":true}]}`))
		case "/api/v2/tailnet/example.com/users":
			sawUsersQuery = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"users":[{"id":"u1","loginName":"alice@example.com","role":"admin","type":"member"}]}`))
		case "/api/v2/device/n1/attributes":
			_, _ = w.Write([]byte(`{"attributes":{"custom:tier":"prod","node:osVersion":"6.8.0"}}`))
		case "/api/v2/tailnet/example.com/services":
			_, _ = w.Write([]byte(`{"vipServices":[{"name":"svc:web","addrs":["100.100.0.1"],"tags":["tag:web"]}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(WithBaseURL(server.URL))
	if _, err := client.Authenticate(context.Background(), AuthRequest{Tailnet: "example.com", APIKey: "tskey-api-test"}); err != nil {
		t.Fatal(err)
	}
	devices, err := client.Devices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sawDevicesQuery != "fields=all" {
		t.Fatalf("devices query = %q, want fields=all", sawDevicesQuery)
	}
	if len(devices) != 1 || !devices[0].IsExternal || devices[0].ClientVersion != "1.94.2" {
		t.Fatalf("devices = %#v", devices)
	}
	users, err := client.Users(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if sawUsersQuery != "role=all&type=all" {
		t.Fatalf("users query = %q, want role=all&type=all", sawUsersQuery)
	}
	if len(users) != 1 || users[0].LoginName != "alice@example.com" || users[0].Role != "admin" {
		t.Fatalf("users = %#v", users)
	}
	attrs, err := client.DevicePostureAttributes(context.Background(), "n1")
	if err != nil {
		t.Fatal(err)
	}
	if attrs.Attributes["custom:tier"] != "prod" || attrs.Attributes["node:osVersion"] != "6.8.0" {
		t.Fatalf("attrs = %#v", attrs.Attributes)
	}
	services, err := client.VIPServices(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 1 || services[0].Name != "svc:web" || services[0].Addrs[0] != "100.100.0.1" {
		t.Fatalf("services = %#v", services)
	}
}
