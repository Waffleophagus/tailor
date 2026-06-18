package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/authz"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"github.com/Waffleophagus/tailor/internal/deploy"
	"github.com/Waffleophagus/tailor/internal/frontend"
	"github.com/Waffleophagus/tailor/internal/mcpserver"
	"github.com/Waffleophagus/tailor/internal/policy"
	"github.com/Waffleophagus/tailor/internal/tailorcore"
	"tailscale.com/client/local"
	"tailscale.com/client/tailscale/apitype"
	"tailscale.com/ipn/ipnstate"
)

type WhoIsClient interface {
	WhoIs(ctx context.Context, remoteAddr string) (*apitype.WhoIsResponse, error)
}

type TailnetStatusClient interface {
	StatusWithoutPeers(ctx context.Context) (*ipnstate.Status, error)
}

type Options struct {
	LocalAPIEndpoint string
	LocalClient      *local.Client
	WhoIsClient      WhoIsClient
	TailnetStatus    TailnetStatusClient
	TailnetMode      bool
	AppCapability    string
	Logger           *slog.Logger
}

type Server struct {
	logger    *slog.Logger
	core      *tailorcore.Service
	deploy    deploy.Environment
	auth      AuthOptions
	bootstrap *BootstrapSessions
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
		LocalClient:      opts.LocalClient,
		Logger:           logger,
	})
	server := &Server{
		logger: logger,
		core:   core,
		deploy: deployEnv,
		auth: AuthOptions{
			TailnetMode:   opts.TailnetMode,
			WhoIsClient:   opts.WhoIsClient,
			TailnetStatus: opts.TailnetStatus,
			AppCapability: appCapability(opts.AppCapability),
		},
		bootstrap: NewBootstrapSessions(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/status", server.handleStatus)
	mux.HandleFunc("GET /api/topology", server.handleTopology)
	mux.HandleFunc("GET /api/topology/socket", server.handleTopologySocket)
	mux.HandleFunc("GET /api/cloud/status", server.handleCloudStatus)
	mux.HandleFunc("POST /api/cloud/auth", server.handleCloudAuth)
	mux.HandleFunc("GET /api/cloud/setup-grant", server.handleSetupGrantRecommendation)
	mux.HandleFunc("POST /api/cloud/setup-grant", server.handleSetupGrantSave)
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
	server.auth.MCPPath = mcpConfig.Path
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

	return AccessMiddleware(logger, BootstrapMiddleware(server, IdentityMiddleware(logger, &server.auth, mux)))
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

func (s *Server) requirePermission(w http.ResponseWriter, r *http.Request, permission authz.Permission, message string) bool {
	if authz.Allowed(r.Context(), permission) {
		return true
	}
	logAPIError(s.logger, r, http.StatusForbidden, nil, string(permission)+" denied")
	writeError(w, http.StatusForbidden, message)
	return false
}

func cloudAuthStatusResponse(r *http.Request, s *Server, status cloudapi.AuthStatus) api.CloudAuthStatusResponse {
	role := "full"
	loginName := ""
	nodeName := ""
	if identity, ok := authz.IdentityFromContext(r.Context()); ok {
		role = string(identity.Role)
		loginName = identity.LoginName
		nodeName = identity.NodeName
	}

	appCapability := s.auth.resolveAppCapability(r.Context(), s.logger)
	if status.DevMode {
		appCapability = "tailor.demo.tailor.ts.net/cap/admin"
	}
	hasGrant := false
	if status.Authenticated && status.HasPolicy && appCapability != "" {
		if raw, err := s.core.CloudPolicy(r.Context()); err == nil {
			hasGrant = policy.HasTailorAppCapabilityGrant(raw, appCapability)
		}
	}

	bootstrapActive, bootstrapExpiresAt := s.bootstrapState(r, loginName, nodeName)
	canEdit := authz.Allowed(r.Context(), authz.PermissionWritePolicy)
	needsSetup := status.Authenticated && appCapability != "" && !hasGrant && !bootstrapActive && (status.DevMode || s.auth.TailnetMode)

	response := api.CloudAuthStatusResponse{
		Authenticated:         status.Authenticated,
		Tailnet:               status.Tailnet,
		HasPolicy:             status.HasPolicy,
		DevMode:               status.DevMode,
		CallerRole:            role,
		CanEditPolicy:         canEdit,
		HasAppCapabilityGrant: hasGrant,
		AppCapability:         appCapability,
		NeedsSetupGrant:       needsSetup,
		BootstrapActive:       bootstrapActive,
		BootstrapExpiresAt:    bootstrapExpiresAt,
	}

	if needsSetup {
		grant := policy.RecommendedSetupGrant(appCapability)
		response.SetupGrantSnippet = policy.FormatGrantSnippet(grant)
	}

	response.StatusMessage = cloudStatusMessage(response)
	return response
}

func cloudStatusMessage(status api.CloudAuthStatusResponse) string {
	switch {
	case status.BootstrapActive:
		return "Tailor could not apply the app capability grant automatically. ACL editing is temporarily available in this browser session. Add the grant below to your tailnet policy and restart Tailor to use grant-based access without a time limit."
	case status.NeedsSetupGrant:
		return "Tailor should add an app capability grant so ACL editing access is controlled by your tailnet policy."
	case status.Authenticated && status.CallerRole == "viewer" && status.HasAppCapabilityGrant:
		return "API key accepted, but your current device or user is view-only."
	case status.Authenticated && status.CallerRole == "viewer":
		return "Tailor access was configured, but your current device or user is view-only."
	default:
		return ""
	}
}

func (s *Server) bootstrapState(r *http.Request, loginName, nodeName string) (active bool, expiresAt string) {
	if s.bootstrap == nil {
		return false, ""
	}
	token := bootstrapTokenFromRequest(r)
	if token == "" {
		return false, ""
	}
	valid, expiry := s.bootstrap.Valid(token, loginName, nodeName)
	if !valid {
		return false, ""
	}
	return true, expiry.Format(time.RFC3339)
}

func appCapability(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	return strings.TrimSpace(os.Getenv("TAILOR_APP_CAPABILITY"))
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
