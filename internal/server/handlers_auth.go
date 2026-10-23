package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/authz"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"github.com/Waffleophagus/tailor/internal/tailorcore"
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

	credentials := cloudapi.AuthRequest{Tailnet: request.Tailnet, APIKey: request.APIKey}
	if s.auth.TailnetMode && !cloudapi.IsDevCredential(credentials) {
		if statusCode, message, ok := s.vetCloudCredentials(r, credentials); !ok {
			writeError(w, statusCode, message)
			return
		}
	}

	status, err := s.core.AuthenticateCloud(r.Context(), credentials)
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
	// The caller's role was resolved before the key existed; with the key in place
	// a tailnet admin is now recognised, so re-resolve before reporting status.
	if identity, ok := authz.IdentityFromContext(r.Context()); ok && identity.Role != authz.RoleFull &&
		len(identity.NodeTags) == 0 && s.auth.PolicyAdmin != nil && s.auth.PolicyAdmin(r.Context(), identity.LoginName) {
		identity.Role = authz.RoleFull
		r = r.WithContext(authz.WithIdentity(r.Context(), identity))
	}
	writeJSON(w, http.StatusOK, cloudAuthStatusResponse(r, s, status))
}

// vetCloudCredentials decides, from the caller's tailnet identity, whether they may
// set Tailor's Cloud API credentials, and checks the key is for Tailor's own tailnet.
//
//   - Editors (app capability grant, or a tailnet admin once a key is configured)
//     may set or replace the key.
//   - Before any key is configured, nobody can be recognized as an admin yet, so the
//     submitted key itself is used to verify the caller is an Owner, Admin or
//     Network admin of that tailnet.
//   - Once a key is configured, everyone else is refused: viewers must not be able
//     to re-point Tailor at another tailnet or swap in their own key.
func (s *Server) vetCloudCredentials(r *http.Request, credentials cloudapi.AuthRequest) (int, string, bool) {
	ctx := r.Context()
	identity, ok := authz.IdentityFromContext(ctx)
	if !ok {
		return http.StatusForbidden, "Tailnet identity is required.", false
	}
	editor := authz.Allowed(ctx, authz.PermissionWritePolicy)
	if !editor && s.core.CloudStatus().Authenticated {
		logAPIError(s.logger, r, http.StatusForbidden, nil, "viewer attempted to replace cloud credentials")
		return http.StatusForbidden, "Only Tailor editors can change the Tailscale API key.", false
	}

	check := tailorcore.CredentialCheck{}
	if !editor {
		if len(identity.NodeTags) > 0 {
			return http.StatusForbidden, "Connect Tailor to the Tailscale API from a personal device, not a tagged one.", false
		}
		check.AdminLogin = identity.LoginName
	}
	self, err := s.selfNode(ctx)
	if err != nil {
		logAPIError(s.logger, r, http.StatusServiceUnavailable, err, "tailor node identity unavailable")
		return http.StatusServiceUnavailable, "Tailor cannot determine its own tailnet node yet. Try again shortly.", false
	}
	check.SelfNodeID, check.SelfIPs = self.nodeID, self.ips

	err = s.core.VerifyCloudCredentials(ctx, credentials, check)
	switch {
	case err == nil:
		return 0, "", true
	case errors.Is(err, tailorcore.ErrCredentialsOtherTailnet), errors.Is(err, tailorcore.ErrCredentialsNotAdmin):
		s.logger.Warn("cloud credentials rejected",
			"login", identity.LoginName,
			"reason", err.Error(),
			"request_id", RequestIDFromContext(ctx),
		)
		return http.StatusForbidden, capitalize(err.Error()) + ".", false
	default:
		statusCode := http.StatusBadGateway
		var apiErr *cloudapi.Error
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			statusCode = http.StatusBadRequest
		}
		logAPIError(s.logger, r, statusCode, err, "cloud credential check failed")
		return statusCode, err.Error(), false
	}
}

type selfNode struct {
	nodeID string
	ips    []string
}

func (s *Server) selfNode(ctx context.Context) (selfNode, error) {
	if s.auth.TailnetStatus == nil {
		return selfNode{}, errors.New("tailnet status client unavailable")
	}
	status, err := s.auth.TailnetStatus.StatusWithoutPeers(ctx)
	if err != nil {
		return selfNode{}, err
	}
	if status == nil || status.Self == nil {
		return selfNode{}, errors.New("tailnet status has no self node")
	}
	node := selfNode{nodeID: string(status.Self.ID)}
	for _, ip := range status.Self.TailscaleIPs {
		node.ips = append(node.ips, ip.String())
	}
	return node, nil
}

func capitalize(message string) string {
	if message == "" {
		return message
	}
	return strings.ToUpper(message[:1]) + message[1:]
}
