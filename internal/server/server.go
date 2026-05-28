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
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"github.com/Waffleophagus/tailor/internal/frontend"
	"github.com/Waffleophagus/tailor/internal/localapi"
	"github.com/Waffleophagus/tailor/internal/policy"
	"github.com/Waffleophagus/tailor/internal/topology"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Options struct {
	LocalAPIEndpoint string
}

type Server struct {
	localAPI *localapi.Client
	cloudAPI *cloudapi.Client
}

func New(options ...Options) http.Handler {
	var opts Options
	if len(options) > 0 {
		opts = options[0]
	}

	server := &Server{
		localAPI: localapi.New(opts.LocalAPIEndpoint),
		cloudAPI: cloudapi.New(),
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

	spa := spaHandler(http.FileServer(frontend.FileSystem()))
	mux.Handle("/", spa)

	return mux
}

func (s *Server) handleCloudStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, cloudAuthStatusResponse(s.cloudAPI.Status()))
}

func (s *Server) handleCloudAuth(w http.ResponseWriter, r *http.Request) {
	var request api.CloudAuthRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
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
		writeError(w, statusCode, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, cloudAuthStatusResponse(status))
}

func (s *Server) handlePolicy(w http.ResponseWriter, r *http.Request) {
	policy, err := s.cloudAPI.Policy(r.Context())
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before fetching the policy file.")
			return
		}
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
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before fetching the policy map.")
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	policyMap, err := policy.StructuredMap(rawPolicy)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	policyMap.Tailnet = s.cloudAPI.Status().Tailnet
	writeJSON(w, http.StatusOK, policyMap)
}

func (s *Server) handlePolicyDraft(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyDraftRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	rule, err := draftRule(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	current, err := s.cloudAPI.Policy(r.Context())
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before drafting a policy change.")
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	draft, err := policy.AppendACLRule(current, rule)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := s.cloudAPI.Status()
	writeJSON(w, http.StatusOK, api.PolicyDraftResponse{Tailnet: status.Tailnet, Rule: rule, HuJSON: draft})
}

func (s *Server) handlePolicyMutate(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyMutationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	base := strings.TrimSpace(request.HuJSON)
	if base == "" {
		current, err := s.cloudAPI.Policy(r.Context())
		if err != nil {
			if errors.Is(err, cloudapi.ErrNotAuthenticated) {
				writeError(w, http.StatusUnauthorized, "Enable ACL editing before mutating policy.")
				return
			}
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		base = current
	}
	draft, err := policy.ApplyMutation(base, request.Mutation)
	if err != nil {
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
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	if strings.TrimSpace(request.HuJSON) == "" {
		writeError(w, http.StatusBadRequest, "Draft policy is required.")
		return
	}
	current, err := s.cloudAPI.Policy(r.Context())
	if err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before evaluating a policy draft.")
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
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
	evaluation, err := policy.EvaluateDraft(current, request.HuJSON, devices, policy.EdgeOptions{Perspective: request.Perspective})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	evaluation.Tailnet = s.cloudAPI.Status().Tailnet
	writeJSON(w, http.StatusOK, evaluation)
}

func (s *Server) handlePolicyValidate(w http.ResponseWriter, r *http.Request) {
	var request api.PolicyValidateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
		return
	}
	if err := s.cloudAPI.ValidatePolicy(r.Context(), request.HuJSON); err != nil {
		if errors.Is(err, cloudapi.ErrNotAuthenticated) {
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before validating a policy change.")
			return
		}
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
			writeError(w, http.StatusUnauthorized, "Enable ACL editing before saving a policy change.")
			return
		}
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, api.PolicySaveResponse{
		Saved:   true,
		Tailnet: s.cloudAPI.Status().Tailnet,
		HuJSON:  policy,
	})
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

	writeJSON(w, http.StatusOK, s.topologySnapshot(r.Context(), devices))
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
		Payload: s.topologySnapshot(ctx, devices),
	}
}

func (s *Server) topologySnapshot(ctx context.Context, devices []api.Device) api.TopologyResponse {
	edges := topology.Phase1Edges(devices)
	if status := s.cloudAPI.Status(); status.Authenticated && status.HasPolicy {
		rawPolicy, err := s.cloudAPI.Policy(ctx)
		if err == nil {
			if accessEdges, err := policy.EffectiveAccessEdges(rawPolicy, devices, policy.EdgeOptions{}); err == nil {
				edges = accessEdges
			}
		}
	}

	tailnet := ""
	if tn, err := s.localAPI.TailnetName(ctx); err == nil {
		tailnet = tn
	}

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
