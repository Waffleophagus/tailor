package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"github.com/Waffleophagus/tailor/internal/localapi"
	"github.com/Waffleophagus/tailor/internal/policy"
	"github.com/Waffleophagus/tailor/internal/tailorcore"
)

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
			logAPIError(s.logger, r, http.StatusNotFound, err, "policy save draft not found")
			writeError(w, http.StatusNotFound, "Choose a staged draft before saving.")
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
