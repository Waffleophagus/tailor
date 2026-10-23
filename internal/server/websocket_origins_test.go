package server

import (
	"net/http"
	"testing"
)

func TestTopologyWebSocketOriginPatternsLoopbackProxy(t *testing.T) {
	r := &http.Request{
		Host: "127.0.0.1:8080",
		Header: http.Header{
			"Origin":           []string{"https://tailor.example.ts.net"},
			"X-Forwarded-Host": []string{"tailor.example.ts.net"},
		},
	}

	patterns := topologyWebSocketOriginPatterns(r)
	if !containsPattern(patterns, "tailor.example.ts.net") {
		t.Fatalf("patterns = %#v, want tailor.example.ts.net", patterns)
	}
}

func TestTopologyWebSocketOriginPatternsDirectHost(t *testing.T) {
	r := &http.Request{
		Host: "tailor.example.ts.net",
		Header: http.Header{
			"Origin": []string{"https://tailor.example.ts.net"},
		},
	}

	patterns := topologyWebSocketOriginPatterns(r)
	if containsPattern(patterns, "tailor.example.ts.net:*") {
		t.Fatalf("direct host should rely on same-origin check, got %#v", patterns)
	}
}

func TestTopologyWebSocketOriginPatternsIgnoreRequestOrigin(t *testing.T) {
	// A page on another site connecting to a Tailor bound to localhost.
	r := &http.Request{
		Host: "localhost:8080",
		Header: http.Header{
			"Origin": []string{"https://evil.example"},
		},
	}

	patterns := topologyWebSocketOriginPatterns(r)
	if containsPattern(patterns, "evil.example") || containsPattern(patterns, "evil.example:*") {
		t.Fatalf("cross-site origin must not be allowed, got %#v", patterns)
	}
}

func containsPattern(patterns []string, want string) bool {
	for _, pattern := range patterns {
		if pattern == want {
			return true
		}
	}
	return false
}
