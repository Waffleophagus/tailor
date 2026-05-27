package cloudapi

import (
	"context"
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
