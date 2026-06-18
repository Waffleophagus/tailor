package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/authz"
	"github.com/Waffleophagus/tailor/internal/policy"
	"tailscale.com/ipn"
)

func (s *Server) handleSetupGrantRecommendation(w http.ResponseWriter, r *http.Request) {
	if !s.requireCloudAuth(w, r, "cloud auth required for setup grant recommendation") {
		return
	}
	status := s.core.CloudStatus()
	appCapability := s.auth.resolveAppCapability(r.Context(), s.logger)
	if status.DevMode {
		appCapability = "tailor.demo.tailor.ts.net/cap/admin"
	}
	if appCapability == "" {
		writeError(w, http.StatusBadRequest, "Tailor app capability is unavailable. Set TAILOR_APP_CAPABILITY or wait for MagicDNS.")
		return
	}
	raw, err := s.core.CloudPolicy(r.Context())
	if err != nil {
		logAPIError(s.logger, r, http.StatusBadGateway, err, "policy fetch failed for setup recommendation")
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if policy.HasTailorAppCapabilityGrant(raw, appCapability) {
		writeJSON(w, http.StatusOK, setupGrantResponse(r, s, appCapability, true))
		return
	}
	grant := policy.RecommendedSetupGrant(appCapability)
	writeJSON(w, http.StatusOK, api.SetupGrantResponse{
		Tailnet:               status.Tailnet,
		AppCapability:         appCapability,
		HasAppCapabilityGrant: false,
		SetupGrantSnippet:     policy.FormatGrantSnippet(grant),
		StatusMessage:         "Tailor should add an app capability grant so ACL editing access is controlled by your tailnet policy.",
	})
}

func (s *Server) handleSetupGrantSave(w http.ResponseWriter, r *http.Request) {
	if !s.requireCloudAuth(w, r, "cloud auth required for setup grant save") {
		return
	}
	if s.auth.TailnetMode && !s.consumeSetupSession(r) {
		writeError(w, http.StatusForbidden, "Re-enter the Cloud API key before applying the Tailor access grant.")
		return
	}
	status := s.core.CloudStatus()
	appCapability := s.auth.resolveAppCapability(r.Context(), s.logger)
	if status.DevMode {
		appCapability = "tailor.demo.tailor.ts.net/cap/admin"
	}
	if appCapability == "" {
		writeError(w, http.StatusBadRequest, "Tailor app capability is unavailable. Set TAILOR_APP_CAPABILITY or wait for MagicDNS.")
		return
	}
	if status.DevMode {
		grant := policy.RecommendedSetupGrant(appCapability)
		var request api.SetupGrantRequest
		if r.ContentLength > 0 {
			if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
				writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
				return
			}
			if request.Grant != nil {
				grant = *request.Grant
			}
		}
		writeJSON(w, http.StatusOK, api.SetupGrantResponse{
			Tailnet:               status.Tailnet,
			AppCapability:         appCapability,
			HasAppCapabilityGrant: true,
			CallerRole:            "full",
			CanEditPolicy:         true,
			StatusMessage:         "Demo access grant applied. No tailnet policy was changed.",
			SetupGrantSnippet:     policy.FormatGrantSnippet(grant),
		})
		return
	}

	// The recommendation may have been open while another editor changed the
	// tailnet policy. Bypass the authenticated session's cached policy before
	// appending so those changes are preserved.
	raw, err := s.core.RefreshCloudPolicy(r.Context())
	if err != nil {
		logAPIError(s.logger, r, http.StatusBadGateway, err, "policy fetch failed for setup grant save")
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if policy.HasTailorAppCapabilityGrant(raw, appCapability) {
		writeJSON(w, http.StatusOK, setupGrantResponse(r, s, appCapability, true))
		return
	}

	grant := policy.RecommendedSetupGrant(appCapability)
	var request api.SetupGrantRequest
	if r.ContentLength > 0 {
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
			logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid setup grant JSON")
			writeError(w, http.StatusBadRequest, "Request body must be valid JSON.")
			return
		}
		if request.Grant != nil {
			grant = *request.Grant
		}
	}
	if err := policy.ValidateSetupGrant(grant, appCapability); err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "invalid setup grant")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	updated, err := policy.AppendSetupGrant(raw, grant)
	if err != nil {
		logAPIError(s.logger, r, http.StatusBadRequest, err, "setup grant append failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.core.ValidateCloudPolicy(r.Context(), updated); err != nil {
		s.issueBootstrapFallback(w, r, grant)
		return
	}
	if _, err := s.core.SaveCloudPolicy(r.Context(), updated); err != nil {
		s.issueBootstrapFallback(w, r, grant)
		return
	}
	if err := s.activateServiceTag(r.Context()); err != nil {
		s.logger.Warn("tailor service tag activation failed",
			"tag", policy.TailorACLServiceTag,
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
	}

	// Policy propagation is asynchronous. Briefly re-check WhoIs so the common
	// case unlocks immediately without requiring a browser refresh.
	if identity, ok := s.waitForAdminCapability(r.Context(), r.RemoteAddr, appCapability); ok {
		r = r.WithContext(authz.WithIdentity(r.Context(), identity))
	}
	writeJSON(w, http.StatusOK, setupGrantResponse(r, s, appCapability, true))
}

func (s *Server) consumeSetupSession(r *http.Request) bool {
	if s.setup == nil {
		return false
	}
	identity, ok := authz.IdentityFromContext(r.Context())
	if !ok {
		return false
	}
	return s.setup.Consume(setupTokenFromRequest(r), identity.LoginName, identity.NodeName)
}

func (s *Server) activateServiceTag(ctx context.Context) error {
	if !s.auth.TailnetMode || s.tailnetPrefs == nil {
		return nil
	}
	prefs, err := s.tailnetPrefs.GetPrefs(ctx)
	if err != nil {
		return err
	}
	tags := append([]string(nil), prefs.AdvertiseTags...)
	for _, tag := range tags {
		if tag == policy.TailorACLServiceTag {
			return nil
		}
	}
	tags = append(tags, policy.TailorACLServiceTag)
	_, err = s.tailnetPrefs.EditPrefs(ctx, &ipn.MaskedPrefs{
		Prefs:            ipn.Prefs{AdvertiseTags: tags},
		AdvertiseTagsSet: true,
	})
	return err
}

const (
	setupCapabilityPollTimeout  = 3 * time.Second
	setupCapabilityPollInterval = 200 * time.Millisecond
)

func (s *Server) waitForAdminCapability(ctx context.Context, remoteAddr, appCapability string) (authz.TailnetIdentity, bool) {
	if !s.auth.TailnetMode || s.auth.WhoIsClient == nil {
		return authz.TailnetIdentity{}, false
	}
	pollCtx, cancel := context.WithTimeout(ctx, setupCapabilityPollTimeout)
	defer cancel()

	var last authz.TailnetIdentity
	for {
		who, err := s.auth.WhoIsClient.WhoIs(pollCtx, remoteAddr)
		if err == nil {
			last = identityFromWhoIs(who, appCapability)
			if last.Role == authz.RoleFull {
				return last, true
			}
		}

		select {
		case <-pollCtx.Done():
			return last, false
		case <-time.After(setupCapabilityPollInterval):
		}
	}
}

func (s *Server) issueBootstrapFallback(w http.ResponseWriter, r *http.Request, grant api.GrantDraft) {
	identity, ok := authz.IdentityFromContext(r.Context())
	if !ok || s.bootstrap == nil {
		writeError(w, http.StatusBadGateway, "Tailor could not apply the app capability grant automatically.")
		return
	}
	token, expiresAt := s.bootstrap.Create(identity.LoginName, identity.NodeName)
	setBootstrapCookie(w, token, expiresAt)
	ctx := authz.WithBootstrap(r.Context())
	response := cloudAuthStatusResponse(r.WithContext(ctx), s, s.core.CloudStatus())
	response.BootstrapActive = true
	response.BootstrapExpiresAt = expiresAt.Format(time.RFC3339)
	response.CanEditPolicy = true
	response.SetupGrantSnippet = policy.FormatGrantSnippet(grant)
	response.StatusMessage = cloudStatusMessage(response)
	writeJSON(w, http.StatusOK, api.SetupGrantResponse{
		Tailnet:            response.Tailnet,
		AppCapability:      response.AppCapability,
		CallerRole:         response.CallerRole,
		CanEditPolicy:      true,
		BootstrapActive:    true,
		BootstrapExpiresAt: response.BootstrapExpiresAt,
		StatusMessage:      response.StatusMessage,
		SetupGrantSnippet:  response.SetupGrantSnippet,
	})
}

func setupGrantResponse(r *http.Request, s *Server, appCapability string, hasGrant bool) api.SetupGrantResponse {
	cloud := cloudAuthStatusResponse(r, s, s.core.CloudStatus())
	return api.SetupGrantResponse{
		Tailnet:               cloud.Tailnet,
		AppCapability:         appCapability,
		HasAppCapabilityGrant: hasGrant,
		CallerRole:            cloud.CallerRole,
		CanEditPolicy:         cloud.CanEditPolicy,
		BootstrapActive:       cloud.BootstrapActive,
		BootstrapExpiresAt:    cloud.BootstrapExpiresAt,
		StatusMessage:         cloud.StatusMessage,
		SetupGrantSnippet:     cloud.SetupGrantSnippet,
	}
}
