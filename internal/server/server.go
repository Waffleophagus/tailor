package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/frontend"
	"github.com/Waffleophagus/tailor/internal/localapi"
	"github.com/Waffleophagus/tailor/internal/topology"
)

type Options struct {
	LocalAPISocketPath string
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
		localAPI: localapi.New(opts.LocalAPISocketPath),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/status", server.handleStatus)
	mux.HandleFunc("GET /api/tailnet", server.handleTailnet)
	mux.HandleFunc("GET /api/topology", server.handleTailnet)

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
			Available:  false,
			SocketPath: s.localAPI.SocketPath(),
			Error:      err.Error(),
		}
		writeJSON(w, http.StatusOK, status)
		return
	}

	writeJSON(w, http.StatusOK, api.LocalAPIStatusResponse{
		Available:  true,
		SocketPath: s.localAPI.SocketPath(),
	})
}

func (s *Server) handleTailnet(w http.ResponseWriter, r *http.Request) {
	devices, err := s.localAPI.Status(r.Context())
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, localapi.ErrUnavailable) {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, api.LocalAPIStatusResponse{
			Available:  false,
			SocketPath: s.localAPI.SocketPath(),
			Error:      err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, api.TopologyResponse{
		Devices: devices,
		Edges:   topology.Phase1Edges(devices),
	})
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
