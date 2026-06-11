package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestHandleTopologySocketAcceptsProxiedOrigin(t *testing.T) {
	ts := httptest.NewServer(New())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/topology/socket"
	conn, resp, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin":           []string{"https://tailor.example.ts.net"},
			"X-Forwarded-Host": []string{"tailor.example.ts.net"},
		},
	})
	if err != nil {
		t.Fatalf("dial: %v (status=%v)", err, resp)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
}
