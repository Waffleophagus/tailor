package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"github.com/Waffleophagus/tailor/internal/deploy"
	"github.com/Waffleophagus/tailor/internal/devtailnet"
	"github.com/Waffleophagus/tailor/internal/frontend"
	"github.com/Waffleophagus/tailor/internal/localapi"
	"github.com/Waffleophagus/tailor/internal/mcpserver"
	"github.com/Waffleophagus/tailor/internal/policy"
	"github.com/Waffleophagus/tailor/internal/tailorcore"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Options struct {
	LocalAPIEndpoint string
	Logger           *slog.Logger
}

type Server struct {
	logger *slog.Logger
	core   *tailorcore.Service
	deploy deploy.Environment
}

func New(options ...Options) http.Handler {
	var opts Options
	if len(options) > 0 {
		opts = options[0]
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}

	deployEnv := deploy.Detect()
	core := tailorcore.New(tailorcore.Options{
		LocalAPIEndpoint: opts.LocalAPIEndpoint,
		Logger:           logger,
	})
	server := &Server{
		logger: logger,
		core:   core,
		deploy: deployEnv,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/status", server.handleStatus)
	mux.HandleFunc("GET /api/topology", server.handleTopology)
	mux.HandleFunc("GET /api/topology/socket", server.handleTopologySocket)
	mux.HandleFunc("GET /api/cloud/status", server.handleCloudStatus)
	mux.HandleFunc("POST /api/cloud/auth", server.handleCloudAuth)
	mux.HandleFunc("GET /api/policy", server.handlePolicy)
	mux.HandleFunc("GET /api/policy/map", server.handlePolicyMap)
	mux.HandleFunc("POST /api/policy/draft", server.handlePolicyDraft)
	mux.HandleFunc("POST /api/policy/mutate", server.handlePolicyMutate)
	mux.HandleFunc("POST /api/policy/evaluate-draft", server.handlePolicyEvaluateDraft)
	mux.HandleFunc("POST /api/policy/validate", server.handlePolicyValidate)
	mux.HandleFunc("GET /api/policy/staged", server.handlePolicyStaged)
	mux.HandleFunc("POST /api/policy/stage", server.handlePolicyStage)
	mux.HandleFunc("GET /api/policy/staged/{id}", server.handlePolicyStagedDraft)
	mux.HandleFunc("DELETE /api/policy/staged/{id}", server.handlePolicyDiscardStaged)
	mux.HandleFunc("POST /api/policy/save", server.handlePolicySave)
	server.registerDevRoutes(mux)

	mcpConfig := mcpserver.ConfigFromEnv()
	if mcpConfig.Enabled() {
		logger.Info("remote mcp enabled",
			"exposure", string(mcpConfig.Exposure),
			"path", mcpConfig.Path,
			"readonly", mcpConfig.ReadOnly,
		)
		mux.Handle(mcpConfig.Path, mcpserver.Handler(core, mcpConfig, logger))
	}

	spa := spaHandler(http.FileServer(frontend.FileSystem()))
	mux.Handle("/", spa)

	return AccessMiddleware(logger, mux)
}

func (s *Server) handleCloudStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, cloudAuthStatusResponse(s.core.CloudStatus()))
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
		if strings.Contains(err.Error(), "required") {
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
	writeJSON(w, http.StatusOK, cloudAuthStatusResponse(status))
}

func (s *Server) handlePolicy(w http.ResponseWriter, r *http.Request) {
	response, err := s.core.Policy(r.Context())
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy fetch requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before fetching the policy file.")
			return
		}
		logAPIError(s.logger, r, http.StatusBadGateway, err, "policy fetch failed")
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handlePolicyMap(w http.ResponseWriter, r *http.Request) {
	policyMap, err := s.core.PolicyMap(r.Context())
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy map requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before fetching the policy map.")
			return
		}
		logAPIError(s.logger, r, http.StatusBadGateway, err, "policy map failed")
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, policyMap)
}

func (s *Server) handlePolicyDraft(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyDraftRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid policy draft JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	if _, err := tailorcore.DraftRule(request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid policy draft rule")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	response, err := s.core.DraftPolicy(r.Context(), request)
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy draft requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before drafting a policy change.")
			return
		}
		if errors.Is(err, tailorcore.ErrPolicyFetch) {
			logAPIError(s.logger, r, http.StatusBadGateway, err, "policy draft fetch failed")
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		logAPIError(s.logger, r, http.StatusBadRequest, err, "policy draft append failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handlePolicyMutate(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyMutationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid policy mutate JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	response, err := s.core.MutatePolicy(r.Context(), request)
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy mutate requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before mutating policy.")
			return
		}
		if errors.Is(err, tailorcore.ErrPolicyFetch) {
			logAPIError(s.logger, r, http.StatusBadGateway, err, "policy mutate fetch failed")
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		logAPIError(s.logger, r, http.StatusBadRequest, err, "policy mutate failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handlePolicyEvaluateDraft(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyEvaluateDraftRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid evaluate draft JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	evaluation, err := s.core.EvaluatePolicyDraft(r.Context(), request)
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "evaluate draft requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before evaluating a policy draft.")
			return
		}
		if errors.Is(err, tailorcore.ErrPolicyFetch) {
			logAPIError(s.logger, r, http.StatusBadGateway, err, "evaluate draft policy fetch failed")
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		if errors.Is(err, localapi.ErrUnavailable) {
			s.writeLocalAPIUnavailable(w, r, http.StatusServiceUnavailable, err, "evaluate draft topology unavailable")
			return
		}
		logAPIError(s.logger, r, http.StatusBadRequest, err, "evaluate draft failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, evaluation)
}

func (s *Server) handlePolicyValidate(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyValidateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid policy validate JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	response, err := s.core.ValidatePolicy(r.Context(), request.HuJSON)
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy validate requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before validating a policy change.")
			return
		}
		s.logger.Info("policy validation failed",
			"tailnet", s.core.CloudStatus().Tailnet,
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
		status := http.StatusBadGateway
		if errors.Is(err, policy.ErrInvalidPolicy) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, api.PolicyValidateResponse{
			Valid:   false,
			Tailnet: s.core.CloudStatus().Tailnet,
			Errors:  []string{err.Error()},
		})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handlePolicyStaged(w http.ResponseWriter, r *http.Request) {
	if !s.requireCloudAuth(w, r, "staged policy list requires auth") {
		return
	}
	writeJSON(w, http.StatusOK, s.core.StagedDrafts())
}

func (s *Server) handlePolicyStagedDraft(w http.ResponseWriter, r *http.Request) {
	if !s.requireCloudAuth(w, r, "staged policy fetch requires auth") {
		return
	}
	response, err := s.core.StagedDraft(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, tailorcore.ErrStagedDraftNotFound) {
			logAPIError(s.logger, r, http.StatusNotFound, err, "staged draft not found")
			writeError(w, http.StatusNotFound, "Staged draft not found.")
			return
		}
		logAPIError(s.logger, r, http.StatusBadRequest, err, "staged draft fetch failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handlePolicyStage(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyStageRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid policy stage JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	response, err := s.core.StagePolicyDraft(r.Context(), request)
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy stage requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before staging a policy change.")
			return
		}
		if errors.Is(err, tailorcore.ErrPolicyFetch) {
			logAPIError(s.logger, r, http.StatusBadGateway, err, "policy stage fetch failed")
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		if errors.Is(err, localapi.ErrUnavailable) {
			s.writeLocalAPIUnavailable(w, r, http.StatusServiceUnavailable, err, "policy stage topology unavailable")
			return
		}
		s.logger.Info("policy stage failed",
			"tailnet", s.core.CloudStatus().Tailnet,
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
		status := http.StatusBadGateway
		if errors.Is(err, policy.ErrInvalidPolicy) {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, api.PolicyValidateResponse{
			Valid:   false,
			Tailnet: s.core.CloudStatus().Tailnet,
			Errors:  []string{err.Error()},
		})
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handlePolicyDiscardStaged(w http.ResponseWriter, r *http.Request) {
	if !s.requireCloudAuth(w, r, "staged policy discard requires auth") {
		return
	}
	response, err := s.core.DiscardStagedDraft(r.PathValue("id"))
	if err != nil {
		if errors.Is(err, tailorcore.ErrStagedDraftNotFound) {
			logAPIError(s.logger, r, http.StatusNotFound, err, "staged draft not found")
			writeError(w, http.StatusNotFound, "Staged draft not found.")
			return
		}
		logAPIError(s.logger, r, http.StatusBadRequest, err, "discard staged draft failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handlePolicySave(w http.ResponseWriter, r *http.Request) {
	var request api.PolicySaveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid policy save JSON")
		writeError(w, http.StatusBadRequest, "Request body must include draftId and draftHash.")
		return
	}
	response, err := s.core.SaveStagedPolicy(r.Context(), request)
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy save requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before saving a policy change.")
			return
		}
		if errors.Is(err, tailorcore.ErrStagedDraftNotFound) {
			logAPIError(s.logger, r, http.StatusBadRequest, err, "policy save draft not found")
			writeError(w, http.StatusBadRequest, "Choose a staged draft before saving.")
			return
		}
		if errors.Is(err, tailorcore.ErrStagedDraftHashMismatch) {
			logAPIError(s.logger, r, http.StatusConflict, err, "policy save stale draft hash")
			writeError(w, http.StatusConflict, "Staged draft hash does not match.")
			return
		}
		if errors.Is(err, tailorcore.ErrStagedDraftBaseMismatch) {
			logAPIError(s.logger, r, http.StatusConflict, err, "policy save stale base policy")
			writeError(w, http.StatusConflict, "Staged draft is based on an older policy.")
			return
		}
		if errors.Is(err, policy.ErrInvalidPolicy) {
			logAPIError(s.logger, r, http.StatusBadRequest, err, "policy save invalid draft")
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.logger.Warn("policy save failed",
			"tailnet", s.core.CloudStatus().Tailnet,
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
		logAPIError(s.logger, r, http.StatusBadGateway, err, "policy save failed")
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	s.logger.Info("policy saved",
		"tailnet", response.Tailnet,
		"dev_mode", s.core.CloudStatus().DevMode,
		"policy_bytes", len(response.HuJSON),
		"request_id", RequestIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, response)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	build := "release"
	if devtailnet.Enabled {
		build = "dev"
	}
	writeJSON(w, http.StatusOK, api.HealthResponse{
		Status:  "ok",
		Version: "dev",
		Build:   build,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if s.core.UseDevTailnet() {
		writeJSON(w, http.StatusOK, api.LocalAPIStatusResponse{
			Available:        true,
			LocalAPIEndpoint: "dev tailnet (" + devtailnet.Name + ")",
		})
		return
	}

	_, err := s.core.TopologyDevicesLogged(r.Context(), "status")
	if err != nil {
		status := api.LocalAPIStatusResponse{
			Available:        false,
			LocalAPIEndpoint: s.core.LocalAPIEndpoint(),
			Error:            err.Error(),
		}
		s.attachSetup(&status, false, 0)
		writeJSON(w, http.StatusOK, status)
		return
	}

	status := api.LocalAPIStatusResponse{
		Available:        true,
		LocalAPIEndpoint: s.core.LocalAPIEndpoint(),
	}
	s.attachSetup(&status, true, 0)
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleTopology(w http.ResponseWriter, r *http.Request) {
	devices, err := s.core.TopologyDevicesLogged(r.Context(), "topology")
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, localapi.ErrUnavailable) {
			status = http.StatusServiceUnavailable
		}
		s.writeLocalAPIUnavailable(w, r, status, err, "topology unavailable")
		return
	}

	s.logger.Info("topology fetched",
		"device_count", len(devices),
		"request_id", RequestIDFromContext(r.Context()),
	)
	snapshot := s.core.TopologySnapshot(r.Context(), devices)
	s.attachTopologySetup(&snapshot, true)
	if s.core.CloudStatus().Authenticated {
		snapshot.StagedDrafts = s.core.StagedDrafts().Drafts
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleTopologySocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: topologyWebSocketOriginPatterns(r),
	})
	if err != nil {
		s.logger.Warn("topology websocket accept failed",
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	s.logger.Info("topology websocket connected",
		"remote", r.RemoteAddr,
		"request_id", RequestIDFromContext(r.Context()),
	)
	defer s.logger.Info("topology websocket disconnected",
		"remote", r.RemoteAddr,
		"request_id", RequestIDFromContext(r.Context()),
	)

	ctx := conn.CloseRead(r.Context())
	conn.SetReadLimit(64 << 10)

	var lastMessage []byte
	if err := s.writeTopologySocketMessage(ctx, conn, &lastMessage); err != nil {
		s.logger.Warn("topology websocket write failed",
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
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
				s.logger.Warn("topology websocket write failed",
					"error", err.Error(),
					"request_id", RequestIDFromContext(r.Context()),
				)
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
	devices, err := s.core.TopologyDevices(ctx)
	if err != nil {
		unavailable := api.LocalAPIStatusResponse{
			Available:        false,
			LocalAPIEndpoint: s.core.LocalAPIEndpoint(),
			Error:            err.Error(),
		}
		s.attachSetup(&unavailable, false, 0)
		return api.SocketMessage{
			Type:    api.SocketMessageLocalAPIUnavailable,
			Payload: unavailable,
		}
	}

	snapshot := s.core.TopologySnapshot(ctx, devices)
	s.attachTopologySetup(&snapshot, true)
	if s.core.CloudStatus().Authenticated {
		snapshot.StagedDrafts = s.core.StagedDrafts().Drafts
	}
	return api.SocketMessage{
		Type:    api.SocketMessageTopologySnapshot,
		Payload: snapshot,
	}
}

func (s *Server) writeLocalAPIUnavailable(w http.ResponseWriter, r *http.Request, status int, err error, message string) {
	logAPIError(s.logger, r, status, err, message)
	unavailable := api.LocalAPIStatusResponse{
		Available:        false,
		LocalAPIEndpoint: s.core.LocalAPIEndpoint(),
		Error:            err.Error(),
	}
	s.attachSetup(&unavailable, false, 0)
	writeJSON(w, status, unavailable)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	buf, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(buf)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, api.ErrorResponse{Error: message})
}

func (s *Server) requireCloudAuth(w http.ResponseWriter, r *http.Request, logMessage string) bool {
	if s.core.CloudStatus().Authenticated {
		return true
	}
	err := cloudapi.ErrNotAuthenticated
	logAPIError(s.logger, r, http.StatusUnauthorized, err, logMessage)
	writeError(w, http.StatusUnauthorized, "Enable ACL editing before reviewing staged policy changes.")
	return false
}

func cloudAuthStatusResponse(status cloudapi.AuthStatus) api.CloudAuthStatusResponse {
	return api.CloudAuthStatusResponse{
		Authenticated: status.Authenticated,
		Tailnet:       status.Tailnet,
		HasPolicy:     status.HasPolicy,
		DevMode:       status.DevMode,
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
