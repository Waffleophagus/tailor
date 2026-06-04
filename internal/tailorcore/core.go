package tailorcore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"github.com/Waffleophagus/tailor/internal/devtailnet"
	"github.com/Waffleophagus/tailor/internal/localapi"
	"github.com/Waffleophagus/tailor/internal/policy"
	"github.com/Waffleophagus/tailor/internal/topology"
)

type Service struct {
	logger   *slog.Logger
	localAPI *localapi.Client
	cloudAPI *cloudapi.Client

	mu           sync.Mutex
	stagedDrafts map[string]stagedDraft
}

var ErrPolicyFetch = errors.New("policy fetch failed")
var ErrStagedDraftNotFound = errors.New("staged draft not found")
var ErrStagedDraftHashMismatch = errors.New("staged draft hash mismatch")

type PolicyFetchError struct {
	Err error
}

func (e PolicyFetchError) Error() string {
	return e.Err.Error()
}

func (e PolicyFetchError) Unwrap() error {
	return e.Err
}

func (e PolicyFetchError) Is(target error) bool {
	return target == ErrPolicyFetch
}

type Options struct {
	LocalAPIEndpoint string
	Logger           *slog.Logger
}

func New(options Options) *Service {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Service{
		logger:       logger,
		localAPI:     localapi.New(options.LocalAPIEndpoint, localapi.WithLogger(logger)),
		cloudAPI:     cloudapi.New(cloudapi.WithLogger(logger)),
		stagedDrafts: map[string]stagedDraft{},
	}
}

func (s *Service) LocalAPIEndpoint() string {
	return s.localAPI.Endpoint()
}

func (s *Service) CloudStatus() cloudapi.AuthStatus {
	return s.cloudAPI.Status()
}

func (s *Service) AuthenticateCloud(ctx context.Context, request cloudapi.AuthRequest) (cloudapi.AuthStatus, error) {
	return s.cloudAPI.Authenticate(ctx, request)
}

func (s *Service) Policy(ctx context.Context) (api.PolicyResponse, error) {
	rawPolicy, err := s.cloudAPI.Policy(ctx)
	if err != nil {
		return api.PolicyResponse{}, err
	}
	return api.PolicyResponse{
		Tailnet: s.cloudAPI.Status().Tailnet,
		HuJSON:  rawPolicy,
	}, nil
}

func (s *Service) PolicyMap(ctx context.Context) (api.PolicyMapResponse, error) {
	rawPolicy, err := s.cloudAPI.Policy(ctx)
	if err != nil {
		return api.PolicyMapResponse{}, err
	}
	policyMap, err := policy.StructuredMap(rawPolicy)
	if err != nil {
		return api.PolicyMapResponse{}, err
	}
	policyMap.Tailnet = s.cloudAPI.Status().Tailnet
	return policyMap, nil
}

func (s *Service) DraftPolicy(ctx context.Context, request api.PolicyDraftRequest) (api.PolicyDraftResponse, error) {
	rule, err := DraftRule(request)
	if err != nil {
		return api.PolicyDraftResponse{}, err
	}
	current, err := s.cloudAPI.Policy(ctx)
	if err != nil {
		return api.PolicyDraftResponse{}, PolicyFetchError{Err: err}
	}
	draft, err := policy.AppendACLRule(current, rule)
	if err != nil {
		return api.PolicyDraftResponse{}, err
	}
	return api.PolicyDraftResponse{
		Tailnet: s.cloudAPI.Status().Tailnet,
		Rule:    rule,
		HuJSON:  draft,
	}, nil
}

func (s *Service) MutatePolicy(ctx context.Context, request api.PolicyMutationRequest) (api.PolicyMutationResponse, error) {
	base := strings.TrimSpace(request.HuJSON)
	if base == "" {
		current, err := s.cloudAPI.Policy(ctx)
		if err != nil {
			return api.PolicyMutationResponse{}, PolicyFetchError{Err: err}
		}
		base = current
	}
	draft, err := policy.ApplyMutation(base, request.Mutation)
	if err != nil {
		return api.PolicyMutationResponse{}, err
	}
	return api.PolicyMutationResponse{
		Tailnet: s.cloudAPI.Status().Tailnet,
		HuJSON:  draft,
		Summary: request.Mutation.Type,
	}, nil
}

func (s *Service) EvaluatePolicyDraft(ctx context.Context, request api.PolicyEvaluateDraftRequest) (api.PolicyEvaluateDraftResponse, error) {
	if strings.TrimSpace(request.HuJSON) == "" {
		return api.PolicyEvaluateDraftResponse{}, errors.New("Draft policy is required.")
	}
	current, err := s.cloudAPI.Policy(ctx)
	if err != nil {
		return api.PolicyEvaluateDraftResponse{}, PolicyFetchError{Err: err}
	}
	devices, err := s.TopologyDevicesLogged(ctx, "evaluate_draft")
	if err != nil {
		return api.PolicyEvaluateDraftResponse{}, err
	}
	evaluation, err := policy.EvaluateDraft(current, request.HuJSON, devices, policy.EdgeOptions{Perspective: request.Perspective})
	if err != nil {
		return api.PolicyEvaluateDraftResponse{}, err
	}
	evaluation.Tailnet = s.cloudAPI.Status().Tailnet
	return evaluation, nil
}

func (s *Service) ValidatePolicy(ctx context.Context, hujson string) (api.PolicyValidateResponse, error) {
	if err := policy.ValidateTailscaleConstraints(hujson); err != nil {
		return api.PolicyValidateResponse{}, err
	}
	if err := s.cloudAPI.ValidatePolicy(ctx, hujson); err != nil {
		return api.PolicyValidateResponse{}, err
	}
	return api.PolicyValidateResponse{Valid: true, Tailnet: s.cloudAPI.Status().Tailnet}, nil
}

func (s *Service) StagePolicyDraft(ctx context.Context, request api.PolicyStageRequest) (api.PolicyStageResponse, error) {
	if _, err := s.ValidatePolicy(ctx, request.HuJSON); err != nil {
		return api.PolicyStageResponse{}, err
	}
	evaluation, err := s.EvaluatePolicyDraft(ctx, api.PolicyEvaluateDraftRequest{HuJSON: request.HuJSON})
	if err != nil {
		return api.PolicyStageResponse{}, err
	}
	current, err := s.cloudAPI.Policy(ctx)
	if err != nil {
		return api.PolicyStageResponse{}, PolicyFetchError{Err: err}
	}

	now := time.Now().UTC()
	draft := stagedDraft{
		ID:         newDraftID(),
		Source:     stagedSource(request.Source),
		Tailnet:    s.cloudAPI.Status().Tailnet,
		BaseHash:   policyHash(current),
		DraftHash:  policyHash(request.HuJSON),
		HuJSON:     request.HuJSON,
		Valid:      true,
		Evaluation: evaluation,
		Summary:    stageSummary(request.Summary, evaluation),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	s.mu.Lock()
	s.stagedDrafts[draft.ID] = draft
	s.mu.Unlock()

	return api.PolicyStageResponse{Draft: draft.toAPI(true)}, nil
}

func (s *Service) StagedDrafts() api.PolicyStagedResponse {
	s.mu.Lock()
	drafts := make([]stagedDraft, 0, len(s.stagedDrafts))
	for _, draft := range s.stagedDrafts {
		drafts = append(drafts, draft)
	}
	s.mu.Unlock()
	sort.Slice(drafts, func(i, j int) bool {
		return drafts[i].CreatedAt.After(drafts[j].CreatedAt)
	})

	response := api.PolicyStagedResponse{Drafts: make([]api.StagedDraft, 0, len(drafts))}
	for _, draft := range drafts {
		response.Drafts = append(response.Drafts, draft.toAPI(false))
	}
	return response
}

func (s *Service) StagedDraft(id string) (api.PolicyStagedDraftResponse, error) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	draft, exists := s.stagedDrafts[id]
	s.mu.Unlock()
	if !exists {
		return api.PolicyStagedDraftResponse{}, ErrStagedDraftNotFound
	}
	return api.PolicyStagedDraftResponse{Draft: draft.toAPI(true)}, nil
}

func (s *Service) DiscardStagedDraft(id string) (api.PolicyDiscardStagedResponse, error) {
	id = strings.TrimSpace(id)
	s.mu.Lock()
	_, exists := s.stagedDrafts[id]
	if exists {
		delete(s.stagedDrafts, id)
	}
	s.mu.Unlock()
	if !exists {
		return api.PolicyDiscardStagedResponse{}, ErrStagedDraftNotFound
	}
	return api.PolicyDiscardStagedResponse{Discarded: true, DraftID: id}, nil
}

func (s *Service) SaveStagedPolicy(ctx context.Context, request api.PolicySaveRequest) (api.PolicySaveResponse, error) {
	draft, err := s.stagedDraft(request.DraftID, request.DraftHash)
	if err != nil {
		return api.PolicySaveResponse{}, err
	}
	if err := policy.ValidateTailscaleConstraints(draft.HuJSON); err != nil {
		return api.PolicySaveResponse{}, err
	}
	saved, err := s.cloudAPI.SavePolicy(ctx, draft.HuJSON)
	if err != nil {
		return api.PolicySaveResponse{}, err
	}
	s.mu.Lock()
	delete(s.stagedDrafts, draft.ID)
	s.mu.Unlock()
	return api.PolicySaveResponse{
		Saved:   true,
		Tailnet: s.cloudAPI.Status().Tailnet,
		HuJSON:  saved,
	}, nil
}

func (s *Service) stagedDraft(id, draftHash string) (stagedDraft, error) {
	id = strings.TrimSpace(id)
	draftHash = strings.TrimSpace(draftHash)
	if id == "" {
		return stagedDraft{}, ErrStagedDraftNotFound
	}
	s.mu.Lock()
	draft, exists := s.stagedDrafts[id]
	s.mu.Unlock()
	if !exists {
		return stagedDraft{}, ErrStagedDraftNotFound
	}
	if draftHash == "" || draft.DraftHash != draftHash {
		return stagedDraft{}, ErrStagedDraftHashMismatch
	}
	return draft, nil
}

func (s *Service) UseDevTailnet() bool {
	return s.cloudAPI.Status().DevMode
}

func (s *Service) TopologyDevices(ctx context.Context) ([]api.Device, error) {
	if s.UseDevTailnet() {
		return devtailnet.Devices(), nil
	}
	return s.localAPI.Status(ctx)
}

func (s *Service) TopologyDevicesLogged(ctx context.Context, operation string) ([]api.Device, error) {
	if s.UseDevTailnet() {
		return devtailnet.Devices(), nil
	}
	return s.localAPI.StatusLogged(ctx, operation)
}

func (s *Service) TopologyTailnet(ctx context.Context) string {
	if s.UseDevTailnet() {
		return devtailnet.Name
	}
	if tn, err := s.localAPI.TailnetName(ctx); err == nil {
		return tn
	}
	return ""
}

func (s *Service) TopologySnapshot(ctx context.Context, devices []api.Device) api.TopologyResponse {
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

	return api.TopologyResponse{
		Devices: devices,
		Edges:   edges,
		Tailnet: s.TopologyTailnet(ctx),
	}
}

func DraftRule(request api.PolicyDraftRequest) (api.ACLDraft, error) {
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

type stagedDraft struct {
	ID         string
	Source     string
	Tailnet    string
	BaseHash   string
	DraftHash  string
	HuJSON     string
	Valid      bool
	Errors     []string
	Evaluation api.PolicyEvaluateDraftResponse
	Summary    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (d stagedDraft) toAPI(includeHuJSON bool) api.StagedDraft {
	hujson := ""
	if includeHuJSON {
		hujson = d.HuJSON
	}
	return api.StagedDraft{
		ID:         d.ID,
		Source:     d.Source,
		Tailnet:    d.Tailnet,
		BaseHash:   d.BaseHash,
		DraftHash:  d.DraftHash,
		HuJSON:     hujson,
		Valid:      d.Valid,
		Errors:     nonNilStrings(d.Errors),
		Evaluation: d.Evaluation,
		Summary:    d.Summary,
		CreatedAt:  d.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:  d.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func newDraftID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "draft-" + hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return "draft-" + hex.EncodeToString(bytes[:])
}

func policyHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func stagedSource(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "ui"
	}
	return source
}

func stageSummary(summary string, evaluation api.PolicyEvaluateDraftResponse) string {
	summary = strings.TrimSpace(summary)
	if summary != "" {
		return summary
	}
	return fmt.Sprintf("%d added, %d removed, %d changed access edges", len(evaluation.Added), len(evaluation.Removed), len(evaluation.Changed))
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
