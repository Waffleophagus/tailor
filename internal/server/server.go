package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/frontend"
	"github.com/Waffleophagus/tailor/internal/localapi"
	"github.com/Waffleophagus/tailor/internal/topology"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Options struct {
	LocalAPIEndpoint string
}

type Server struct {
	localAPI *localapi.Client
}

func New(options ...Options) http.Handler {
	var opts Options
	if len(options) > 0 {
		opts = options[0]
	}

	server := &Server{
		localAPI: localapi.New(opts.LocalAPIEndpoint),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/status", server.handleStatus)
	mux.HandleFunc("GET /api/topology", server.handleTopology)
	mux.HandleFunc("GET /api/topology/socket", server.handleTopologySocket)

	spa := spaHandler(http.FileServer(frontend.FileSystem()))
	mux.Handle("/", spa)

	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.HealthResponse{
		Status:  "ok",
		Version: "dev",
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	_, err := s.localAPI.Status(r.Context())
	if err != nil {
		status := api.LocalAPIStatusResponse{
			Available:        false,
			LocalAPIEndpoint: s.localAPI.Endpoint(),
			Error:            err.Error(),
		}
		writeJSON(w, http.StatusOK, status)
		return
	}

	writeJSON(w, http.StatusOK, api.LocalAPIStatusResponse{
		Available:        true,
		LocalAPIEndpoint: s.localAPI.Endpoint(),
	})
}

func (s *Server) handleTopology(w http.ResponseWriter, r *http.Request) {
	devices, err := s.localAPI.Status(r.Context())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, localapi.ErrUnavailable) {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, api.LocalAPIStatusResponse{
			Available:        false,
			LocalAPIEndpoint: s.localAPI.Endpoint(),
			Error:            err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, topologySnapshot(devices))
}

func (s *Server) handleTopologySocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:*", "127.0.0.1:*"},
	})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx := conn.CloseRead(r.Context())
	conn.SetReadLimit(64 << 10)

	var lastMessage []byte
	if err := s.writeTopologySocketMessage(ctx, conn, &lastMessage); err != nil {
		return
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.writeTopologySocketMessage(ctx, conn, &lastMessage); err != nil {
				return
			}
		}
	}
}

func (s *Server) writeTopologySocketMessage(ctx context.Context, conn *websocket.Conn, lastMessage *[]byte) error {
	message := s.topologySocketMessage(ctx)
	encoded, err := json.Marshal(message)
	if err != nil {
		return err
	}
	if bytes.Equal(encoded, *lastMessage) {
		return nil
	}
	*lastMessage = encoded

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return wsjson.Write(writeCtx, conn, message)
}

func (s *Server) topologySocketMessage(ctx context.Context) api.SocketMessage {
	devices, err := s.localAPI.Status(ctx)
	if err != nil {
		return api.SocketMessage{
			Type: api.SocketMessageLocalAPIUnavailable,
			Payload: api.LocalAPIStatusResponse{
				Available:        false,
				LocalAPIEndpoint: s.localAPI.Endpoint(),
				Error:            err.Error(),
			},
		}
	}

	return api.SocketMessage{
		Type:    api.SocketMessageTopologySnapshot,
		Payload: topologySnapshot(devices),
	}
}

func topologySnapshot(devices []api.Device) api.TopologyResponse {
	return api.TopologyResponse{
		Devices: devices,
		Edges:   topology.Phase1Edges(devices),
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func spaHandler(fileServer http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		fileServer.ServeHTTP(w, r)
	})
}
