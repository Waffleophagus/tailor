package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/authz"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
)

func (s *Server) handleCloudStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, cloudAuthStatusResponse(r, s, s.core.CloudStatus()))
}

func (s *Server) handleCloudAuth(w http.ResponseWriter, r *http.Request) {
	var request api.CloudAuthRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid cloud auth JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}

	status, err := s.core.AuthenticateCloud(r.Context(), cloudapi.AuthRequest{
		Tailnet: request.Tailnet,
		APIKey:  request.APIKey,
	})
	if err != nil {
		statusCode := http.StatusBadGateway
		var apiErr *cloudapi.Error
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			statusCode = http.StatusBadRequest
		}
		s.logger.Warn("cloud auth failed",
			"tailnet", strings.TrimSpace(request.Tailnet),
			"status", statusCode,
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
		logAPIError(s.logger, r, statusCode, err, "cloud auth failed")
		writeError(w, statusCode, err.Error())
		return
	}

	s.logger.Info("cloud auth succeeded",
		"tailnet", status.Tailnet,
		"dev_mode", status.DevMode,
		"request_id", RequestIDFromContext(r.Context()),
	)
	if s.auth.TailnetMode && s.setup != nil {
		identity, ok := authz.IdentityFromContext(r.Context())
		if ok {
			token, expiresAt := s.setup.Create(identity.LoginName, identity.NodeName)
			setSetupCookie(w, r, token, expiresAt)
		}
	}
	writeJSON(w, http.StatusOK, cloudAuthStatusResponse(r, s, status))
}
