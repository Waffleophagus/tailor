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
	"github.com/Waffleophagus/tailor/internal/policy"
	"github.com/Waffleophagus/tailor/internal/topology"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Options struct {
	LocalAPIEndpoint string
	Logger           *slog.Logger
}

type Server struct {
	logger   *slog.Logger
	localAPI *localapi.Client
	cloudAPI *cloudapi.Client
	deploy   deploy.Environment
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
	server := &Server{
		logger:   logger,
		localAPI: localapi.New(opts.LocalAPIEndpoint, localapi.WithLogger(logger)),
		cloudAPI: cloudapi.New(cloudapi.WithLogger(logger)),
		deploy:   deployEnv,
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
	mux.HandleFunc("POST /api/policy/save", server.handlePolicySave)
	server.registerDevRoutes(mux)

	spa := spaHandler(http.FileServer(frontend.FileSystem()))
	mux.Handle("/", spa)

	return AccessMiddleware(logger, mux)
}

func (s *Server) handleCloudStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, cloudAuthStatusResponse(s.cloudAPI.Status()))
}

func (s *Server) handleCloudAuth(w http.ResponseWriter, r *http.Request) {
	var request api.CloudAuthRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid cloud auth JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}

	status, err := s.cloudAPI.Authenticate(r.Context(), cloudapi.AuthRequest{
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
	policy, err := s.cloudAPI.Policy(r.Context())
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
	status := s.cloudAPI.Status()
	writeJSON(w, http.StatusOK, api.PolicyResponse{
		Tailnet: status.Tailnet,
		HuJSON:  policy,
	})
}

func (s *Server) handlePolicyMap(w http.ResponseWriter, r *http.Request) {
	rawPolicy, err := s.cloudAPI.Policy(r.Context())
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy map requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before fetching the policy map.")
			return
		}
		logAPIError(s.logger, r, http.StatusBadGateway, err, "policy map fetch failed")
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	policyMap, err := policy.StructuredMap(rawPolicy)
	if err != nil {
		logAPIError(s.logger, r, http.StatusBadGateway, err, "policy map parse failed")
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	policyMap.Tailnet = s.cloudAPI.Status().Tailnet
	writeJSON(w, http.StatusOK, policyMap)
}

func (s *Server) handlePolicyDraft(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyDraftRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid policy draft JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	rule, err := draftRule(request)
	if err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid policy draft rule")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	current, err := s.cloudAPI.Policy(r.Context())
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy draft requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before drafting a policy change.")
			return
		}
		logAPIError(s.logger, r, http.StatusBadGateway, err, "policy draft fetch failed")
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	draft, err := policy.AppendACLRule(current, rule)
	if err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "policy draft append failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := s.cloudAPI.Status()
	writeJSON(w, http.StatusOK, api.PolicyDraftResponse{Tailnet: status.Tailnet, Rule: rule, HuJSON: draft})
}

func (s *Server) handlePolicyMutate(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyMutationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid policy mutate JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	base := strings.TrimSpace(request.HuJSON)
	if base == "" {
		current, err := s.cloudAPI.Policy(r.Context())
		if err != nil {
			if errors.Is(err, cloudapi.ErrNotAuthenticated) {
				logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy mutate requires auth")
				writeError(w, http.StatusUnauthorized, "Enable ACL editing before mutating policy.")
				return
			}
			logAPIError(s.logger, r, http.StatusBadGateway, err, "policy mutate fetch failed")
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		base = current
	}
	draft, err := policy.ApplyMutation(base, request.Mutation)
	if err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "policy mutate failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := s.cloudAPI.Status()
	writeJSON(w, http.StatusOK, api.PolicyMutationResponse{
		Tailnet: status.Tailnet,
		HuJSON:  draft,
		Summary: request.Mutation.Type,
	})
}

func (s *Server) handlePolicyEvaluateDraft(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyEvaluateDraftRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid evaluate draft JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	if strings.TrimSpace(request.HuJSON) == "" {
		logAPIError(s.logger, r, http.StatusBadRequest, nil, "draft policy required")
		writeError(w, http.StatusBadRequest, "Draft policy is required.")
		return
	}
	current, err := s.cloudAPI.Policy(r.Context())
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "evaluate draft requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before evaluating a policy draft.")
			return
		}
		logAPIError(s.logger, r, http.StatusBadGateway, err, "evaluate draft policy fetch failed")
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	devices, err := s.topologyDevicesLogged(r.Context(), "evaluate_draft")
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, localapi.ErrUnavailable) {
			status = http.StatusServiceUnavailable
		}
		logAPIError(s.logger, r, status, err, "evaluate draft topology unavailable")
		unavailable := api.LocalAPIStatusResponse{
			Available:        false,
			LocalAPIEndpoint: s.localAPI.Endpoint(),
			Error:            err.Error(),
		}
		s.attachSetup(&unavailable, false, 0)
		writeJSON(w, status, unavailable)
		return
	}
	evaluation, err := policy.EvaluateDraft(current, request.HuJSON, devices, policy.EdgeOptions{Perspective: request.Perspective})
	if err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "evaluate draft failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	evaluation.Tailnet = s.cloudAPI.Status().Tailnet
	writeJSON(w, http.StatusOK, evaluation)
}

func (s *Server) handlePolicyValidate(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyValidateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&request); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid policy validate JSON")
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	if err := s.cloudAPI.ValidatePolicy(r.Context(), request.HuJSON); err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy validate requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before validating a policy change.")
			return
		}
		s.logger.Info("policy validation failed",
			"tailnet", s.cloudAPI.Status().Tailnet,
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
		writeJSON(w, http.StatusBadGateway, api.PolicyValidateResponse{
			Valid:   false,
			Tailnet: s.cloudAPI.Status().Tailnet,
			Errors:  []string{err.Error()},
		})
		return
	}
	writeJSON(w, http.StatusOK, api.PolicyValidateResponse{Valid: true, Tailnet: s.cloudAPI.Status().Tailnet})
}

func (s *Server) handlePolicySave(w http.ResponseWriter, r *http.Request) {
	policy, err := s.cloudAPI.SaveValidatedPolicy(r.Context())
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			logAPIError(s.logger, r, http.StatusUnauthorized, err, "policy save requires auth")
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before saving a policy change.")
			return
		}
		s.logger.Warn("policy save failed",
			"tailnet", s.cloudAPI.Status().Tailnet,
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
		logAPIError(s.logger, r, http.StatusBadGateway, err, "policy save failed")
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	status := s.cloudAPI.Status()
	s.logger.Info("policy saved",
		"tailnet", status.Tailnet,
		"dev_mode", status.DevMode,
		"policy_bytes", len(policy),
		"request_id", RequestIDFromContext(r.Context()),
	)
	writeJSON(w, http.StatusOK, api.PolicySaveResponse{
		Saved:   true,
		Tailnet: status.Tailnet,
		HuJSON:  policy,
	})
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
	if s.useDevTailnet() {
		writeJSON(w, http.StatusOK, api.LocalAPIStatusResponse{
			Available:        true,
			LocalAPIEndpoint: "dev tailnet (" + devtailnet.Name + ")",
		})
		return
	}

	_, err := s.localAPI.StatusLogged(r.Context(), "status")
	if err != nil {
		status := api.LocalAPIStatusResponse{
			Available:        false,
			LocalAPIEndpoint: s.localAPI.Endpoint(),
			Error:            err.Error(),
		}
		s.attachSetup(&status, false, 0)
		writeJSON(w, http.StatusOK, status)
		return
	}

	status := api.LocalAPIStatusResponse{
		Available:        true,
		LocalAPIEndpoint: s.localAPI.Endpoint(),
	}
	s.attachSetup(&status, true, 0)
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleTopology(w http.ResponseWriter, r *http.Request) {
	devices, err := s.topologyDevicesLogged(r.Context(), "topology")
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, localapi.ErrUnavailable) {
			status = http.StatusServiceUnavailable
		}
		logAPIError(s.logger, r, status, err, "topology unavailable")
		unavailable := api.LocalAPIStatusResponse{
			Available:        false,
			LocalAPIEndpoint: s.localAPI.Endpoint(),
			Error:            err.Error(),
		}
		s.attachSetup(&unavailable, false, 0)
		writeJSON(w, status, unavailable)
		return
	}

	s.logger.Info("topology fetched",
		"device_count", len(devices),
		"request_id", RequestIDFromContext(r.Context()),
	)
	snapshot := s.topologySnapshot(r.Context(), devices)
	s.attachTopologySetup(&snapshot, true)
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) handleTopologySocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:*", "127.0.0.1:*"},
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
	devices, err := s.topologyDevices(ctx)
	if err != nil {
		unavailable := api.LocalAPIStatusResponse{
			Available:        false,
			LocalAPIEndpoint: s.localAPI.Endpoint(),
			Error:            err.Error(),
		}
		s.attachSetup(&unavailable, false, 0)
		return api.SocketMessage{
			Type:    api.SocketMessageLocalAPIUnavailable,
			Payload: unavailable,
		}
	}

	snapshot := s.topologySnapshot(ctx, devices)
	s.attachTopologySetup(&snapshot, true)
	return api.SocketMessage{
		Type:    api.SocketMessageTopologySnapshot,
		Payload: snapshot,
	}
}

func (s *Server) useDevTailnet() bool {
	return s.cloudAPI.Status().DevMode
}

func (s *Server) topologyDevices(ctx context.Context) ([]api.Device, error) {
	if s.useDevTailnet() {
		return devtailnet.Devices(), nil
	}
	return s.localAPI.Status(ctx)
}

func (s *Server) topologyDevicesLogged(ctx context.Context, operation string) ([]api.Device, error) {
	if s.useDevTailnet() {
		return devtailnet.Devices(), nil
	}
	return s.localAPI.StatusLogged(ctx, operation)
}

func (s *Server) topologyTailnet(ctx context.Context) string {
	if s.useDevTailnet() {
		return devtailnet.Name
	}
	if tn, err := s.localAPI.TailnetName(ctx); err == nil {
		return tn
	}
	return ""
}

func (s *Server) topologySnapshot(ctx context.Context, devices []api.Device) api.TopologyResponse {
	edges := topology.Phase1Edges(devices)
	if status := s.cloudAPI.Status(); status.Authenticated && status.HasPolicy {
		rawPolicy, err := s.cloudAPI.Policy(ctx)
		if err == nil {
			if accessEdges, err := policy.EffectiveAccessEdges(rawPolicy, devices, policy.EdgeOptions{}); err == nil {
				edges = accessEdges
			} else {
				s.logger.Debug("effective access edges failed", "error", err.Error())
			}
		} else {
			s.logger.Debug("topology policy fetch failed", "error", err.Error())
		}
	}

	tailnet := s.topologyTailnet(ctx)

	return api.TopologyResponse{
		Devices: devices,
		Edges:   edges,
		Tailnet: tailnet,
	}
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

func cloudAuthStatusResponse(status cloudapi.AuthStatus) api.CloudAuthStatusResponse {
	return api.CloudAuthStatusResponse{
		Authenticated: status.Authenticated,
		Tailnet:       status.Tailnet,
		HasPolicy:     status.HasPolicy,
		DevMode:       status.DevMode,
	}
}

func draftRule(request api.PolicyDraftRequest) (api.ACLDraft, error) {
	sources := compactStrings(request.Sources)
	destinations := compactStrings(request.Destinations)
	ports := compactStrings(request.Ports)
	if len(sources) == 0 {
		return api.ACLDraft{}, errors.New("at least one source selector is required")
	}
	if len(destinations) == 0 {
		return api.ACLDraft{}, errors.New("at least one destination selector is required")
	}
	if len(ports) == 0 {
		return api.ACLDraft{}, errors.New("at least one destination port is required")
	}
	dst := make([]string, 0, len(destinations))
	portSet := strings.Join(ports, ",")
	for _, destination := range destinations {
		if strings.Contains(destination, ":") && strings.HasSuffix(destination, ":*") {
			dst = append(dst, destination)
			continue
		}
		dst = append(dst, destination+":"+portSet)
	}
	proto := strings.TrimSpace(request.Protocol)
	if proto == "tcp" {
		proto = ""
	}
	return api.ACLDraft{Action: "accept", Src: sources, Dst: dst, Proto: proto}, nil
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
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
