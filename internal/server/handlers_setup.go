package server

import (
	"context"
	"encoding/json"
	"net/http"

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
		StatusMessage:         "Optionally add an app capability grant to make Tailor access explicit in your tailnet policy and delegate it to other users or tagged devices.",
	})
}

func (s *Server) handleSetupGrantSave(w http.ResponseWriter, r *http.Request) {
	if !s.requireCloudAuth(w, r, "cloud auth required for setup grant save") {
		return
	}
	status := s.core.CloudStatus()
	// The demo tailnet simulates the grant; everywhere else only editors may change policy.
	if !status.DevMode && !s.requirePermission(w, r, authz.PermissionWritePolicy, "Only Tailor editors can add the access grant.") {
		return
	}
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
		logAPIError(s.logger, r, http.StatusBadRequest, err, "setup grant failed validation")
		writeError(w, http.StatusBadRequest, "Tailscale rejected the access grant: "+err.Error())
		return
	}
	if _, err := s.core.SaveCloudPolicy(r.Context(), updated); err != nil {
		logAPIError(s.logger, r, http.StatusBadGateway, err, "setup grant save failed")
		writeError(w, http.StatusBadGateway, "Tailor could not save the access grant: "+err.Error())
		return
	}
	if err := s.activateServiceTag(r.Context()); err != nil {
		s.logger.Warn("tailor service tag activation failed",
			"tag", policy.TailorACLServiceTag,
			"error", err.Error(),
			"request_id", RequestIDFromContext(r.Context()),
		)
	}

	writeJSON(w, http.StatusOK, setupGrantResponse(r, s, appCapability, true))
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

func setupGrantResponse(r *http.Request, s *Server, appCapability string, hasGrant bool) api.SetupGrantResponse {
	cloud := cloudAuthStatusResponse(r, s, s.core.CloudStatus())
	return api.SetupGrantResponse{
		Tailnet:               cloud.Tailnet,
		AppCapability:         appCapability,
		HasAppCapabilityGrant: hasGrant,
		CallerRole:            cloud.CallerRole,
		CanEditPolicy:         cloud.CanEditPolicy,
		StatusMessage:         cloud.StatusMessage,
		SetupGrantSnippet:     cloud.SetupGrantSnippet,
	}
}
