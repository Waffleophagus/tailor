package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/frontend"
)

func New() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/topology", handleTopology)

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

func handleTopology(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, api.TopologyResponse{
		Devices: []api.Device{},
		Edges:   []api.Edge{},
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
