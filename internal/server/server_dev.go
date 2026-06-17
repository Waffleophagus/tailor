//go:build dev

package server

import (
	"encoding/json"
	"net/http"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/devtailnet"
)

func (s *Server) registerDevRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/dev/spawn-devices", s.handleDevSpawnDevices)
	mux.HandleFunc("POST /api/dev/patch-devices", s.handleDevPatchDevices)
}

func (s *Server) handleDevSpawnDevices(w http.ResponseWriter, r *http.Request) {
	if !devtailnet.Enabled {
		http.NotFound(w, r)
		return
	}
	if !s.core.CloudStatus().DevMode {
		logAPIError(s.logger, r, http.StatusForbidden, nil, "spawn devices requires dev mode")
		writeError(w, http.StatusForbidden, "Spawn devices requires dev mode.")
		return
	}

	var request api.DevSpawnDevicesRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid spawn devices JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}

	spawned, err := devtailnet.SpawnDevices(request)
	if err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "spawn devices failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.logger.Info("dev devices spawned",
		"count", len(spawned),
		"request_id", RequestIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, api.DevSpawnDevicesResponse{
		Tailnet: devtailnet.Name,
		Spawned: spawned,
		Devices: devtailnet.Devices(),
	})
}

func (s *Server) handleDevPatchDevices(w http.ResponseWriter, r *http.Request) {
	if !devtailnet.Enabled {
		http.NotFound(w, r)
		return
	}
	if !s.core.CloudStatus().DevMode {
		logAPIError(s.logger, r, http.StatusForbidden, nil, "patch devices requires dev mode")
		writeError(w, http.StatusForbidden, "Patch devices requires dev mode.")
		return
	}

	var request api.DevPatchDevicesRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid patch devices JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}

	patched, err := devtailnet.PatchDevices(request)
	if err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "patch devices failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.logger.Info("dev devices patched",
		"count", len(patched),
		"request_id", RequestIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, api.DevPatchDevicesResponse{
		Tailnet: devtailnet.Name,
		Patched: patched,
		Devices: devtailnet.Devices(),
	})
}
