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
	"sync/atomic"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"github.com/Waffleophagus/tailor/internal/devtailnet"
	"github.com/Waffleophagus/tailor/internal/localapi"
	"github.com/Waffleophagus/tailor/internal/policy"
	"github.com/Waffleophagus/tailor/internal/topology"
	"tailscale.com/client/local"
)

type Service struct {
	logger      *slog.Logger
	localAPI    *localapi.Client
	cloudAPI    *cloudapi.Client
	policyCache policy.Cache
	warmups     atomic.Int32

	mu           sync.Mutex
	stagedDrafts map[string]stagedDraft

	cleanupDone chan struct{}
	cleanupStop chan struct{}
	closeOnce   sync.Once
}

const (
	stagedDraftTTL             = 24 * time.Hour
	stagedDraftCleanupInterval = time.Hour
	maxStagedDrafts            = 100
	policyCacheWarmupTimeout   = 5 * time.Minute
)

var ErrPolicyFetch = errors.New("policy fetch failed")
var ErrStagedDraftNotFound = errors.New("staged draft not found")
var ErrStagedDraftHashMismatch = errors.New("staged draft hash mismatch")
var ErrStagedDraftBaseMismatch = errors.New("staged draft base policy mismatch")

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
	LocalClient      *local.Client
	CloudAPIOptions  []cloudapi.Option
	Logger           *slog.Logger
}

func New(options Options) *Service {
	logger := options.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	service := &Service{
		logger:       logger,
		localAPI:     localapi.NewWithLocalClient(options.LocalClient, options.LocalAPIEndpoint, localapi.WithLogger(logger)),
		cloudAPI:     cloudapi.New(append(options.CloudAPIOptions, cloudapi.WithLogger(logger))...),
		stagedDrafts: map[string]stagedDraft{},
		cleanupDone:  make(chan struct{}),
		cleanupStop:  make(chan struct{}),
	}
	go service.cleanupStagedDrafts()
	return service
}

func (s *Service) Close() {
	s.closeOnce.Do(func() {
		close(s.cleanupStop)
		<-s.cleanupDone
	})
}

func (s *Service) LocalAPIEndpoint() string {
	return s.localAPI.Endpoint()
}

func (s *Service) CloudStatus() cloudapi.AuthStatus {
	return s.cloudAPI.Status()
}

func (s *Service) CloudPolicy(ctx context.Context) (string, error) {
	return s.cloudAPI.Policy(ctx)
}

func (s *Service) RefreshCloudPolicy(ctx context.Context) (string, error) {
	raw, err := s.cloudAPI.RefreshPolicy(ctx)
	if err == nil {
		s.policyCache.Invalidate()
	}
	return raw, err
}

func (s *Service) ValidateCloudPolicy(ctx context.Context, draft string) error {
	return s.cloudAPI.ValidatePolicy(ctx, draft)
}

func (s *Service) SaveCloudPolicy(ctx context.Context, draft string) (string, error) {
	raw, err := s.cloudAPI.SavePolicy(ctx, draft)
	if err == nil {
		s.policyCache.Invalidate()
	}
	return raw, err
}

func (s *Service) AuthenticateCloud(ctx context.Context, request cloudapi.AuthRequest) (cloudapi.AuthStatus, error) {
	status, err := s.cloudAPI.Authenticate(ctx, request)
	if err == nil {
		s.policyCache.Invalidate()
		s.warmups.Add(1)
		go s.warmPolicyCache()
	}
	return status, err
}

func (s *Service) warmPolicyCache() {
	defer s.warmups.Add(-1)

	ctx, cancel := context.WithTimeout(context.Background(), policyCacheWarmupTimeout)
	defer cancel()

	rawPolicy, err := s.cloudAPI.Policy(ctx)
	if err != nil {
		s.logger.Debug("policy cache warmup failed", "stage", "policy", "error", err.Error())
		return
	}
	if _, err := s.policyCache.StructuredMap(rawPolicy); err != nil {
		s.logger.Debug("policy cache warmup failed", "stage", "structured_map", "error", err.Error())
		return
	}
	devices, err := s.TopologyDevicesLogged(ctx, "policy_cache_warmup")
	if err != nil {
		s.logger.Debug("policy cache warmup failed", "stage", "devices", "error", err.Error())
		return
	}
	if _, err := s.policyCache.EffectiveAccessEdges(rawPolicy, devices, policy.EdgeOptions{}); err != nil {
		s.logger.Debug("policy cache warmup failed", "stage", "effective_edges", "error", err.Error())
		return
	}
	s.logger.Debug("policy cache warmup complete", "devices", len(devices))
}

func (s *Service) PolicyCacheWarming() bool {
	return s.warmups.Load() > 0
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
	policyMap, err := s.policyCache.StructuredMap(rawPolicy)
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
		return api.PolicyEvaluateDraftResponse{}, errors.New("draft policy is required")
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
		ExpiresAt:  now.Add(stagedDraftTTL),
	}

	s.mu.Lock()
	s.evictExpiredStagedDraftsLocked(now)
	s.evictOldestStagedDraftsLocked(maxStagedDrafts - 1)
	s.stagedDrafts[draft.ID] = draft
	s.mu.Unlock()

	return api.PolicyStageResponse{Draft: draft.toAPI(true)}, nil
}

func (s *Service) StagedDrafts() api.PolicyStagedResponse {
	s.mu.Lock()
	s.evictExpiredStagedDraftsLocked(time.Now().UTC())
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
	s.evictExpiredStagedDraftsLocked(time.Now().UTC())
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
	current, err := s.cloudAPI.Policy(ctx)
	if err != nil {
		return api.PolicySaveResponse{}, PolicyFetchError{Err: err}
	}
	if policyHash(current) != draft.BaseHash {
		return api.PolicySaveResponse{}, ErrStagedDraftBaseMismatch
	}
	saved, err := s.cloudAPI.SavePolicy(ctx, draft.HuJSON)
	if err != nil {
		return api.PolicySaveResponse{}, err
	}
	s.policyCache.Invalidate()
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
	s.evictExpiredStagedDraftsLocked(time.Now().UTC())
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

func (s *Service) cleanupStagedDrafts() {
	defer close(s.cleanupDone)
	ticker := time.NewTicker(stagedDraftCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			s.mu.Lock()
			s.evictExpiredStagedDraftsLocked(now.UTC())
			s.mu.Unlock()
		case <-s.cleanupStop:
			return
		}
	}
}

func (s *Service) evictExpiredStagedDraftsLocked(now time.Time) {
	for id, draft := range s.stagedDrafts {
		if !draft.ExpiresAt.IsZero() && !draft.ExpiresAt.After(now) {
			delete(s.stagedDrafts, id)
		}
	}
}

func (s *Service) evictOldestStagedDraftsLocked(keep int) {
	for len(s.stagedDrafts) > keep {
		var oldestID string
		var oldestCreatedAt time.Time
		for id, draft := range s.stagedDrafts {
			if oldestID == "" || draft.CreatedAt.Before(oldestCreatedAt) {
				oldestID = id
				oldestCreatedAt = draft.CreatedAt
			}
		}
		delete(s.stagedDrafts, oldestID)
	}
}

func (s *Service) UseDevTailnet() bool {
	return s.cloudAPI.Status().DevMode
}

func (s *Service) TopologyDevices(ctx context.Context) ([]api.Device, error) {
	if s.UseDevTailnet() {
		return devtailnet.Devices(), nil
	}
	devices, err := s.localAPI.Status(ctx)
	if err != nil {
		return nil, err
	}
	return s.enrichTopologyFromCloud(ctx, devices), nil
}

func (s *Service) TopologyDevicesLogged(ctx context.Context, operation string) ([]api.Device, error) {
	if s.UseDevTailnet() {
		return devtailnet.Devices(), nil
	}
	devices, err := s.localAPI.StatusLogged(ctx, operation)
	if err != nil {
		return nil, err
	}
	return s.enrichTopologyFromCloud(ctx, devices), nil
}

func (s *Service) enrichTopologyFromCloud(ctx context.Context, devices []api.Device) []api.Device {
	if status := s.cloudAPI.Status(); !status.Authenticated || status.DevMode {
		return s.enrichTopologyFromLocalServices(ctx, devices)
	}
	cloudDevices, err := s.cloudAPI.Devices(ctx)
	if err != nil {
		s.logger.Debug("cloud device metadata unavailable", "error", err.Error())
	} else {
		enrichDevicesFromCloud(devices, cloudDevices)
		users, err := s.cloudAPI.Users(ctx)
		if err != nil {
			s.logger.Debug("cloud user metadata unavailable", "error", err.Error())
		} else {
			enrichDeviceRolesFromCloudUsers(devices, users)
		}
		enrichDevicePostureAttributes(ctx, s.cloudAPI, devices, cloudDevices, s.logger)
	}
	services, err := s.cloudAPI.VIPServices(ctx)
	if err != nil {
		s.logger.Debug("vip services unavailable", "error", err.Error())
		return devices
	}
	return append(devices, serviceDevicesFromCloud(services)...)
}

func (s *Service) enrichTopologyFromLocalServices(ctx context.Context, devices []api.Device) []api.Device {
	services, err := s.localAPI.VIPServiceDevices(ctx)
	if err != nil {
		s.logger.Debug("localapi vip services unavailable", "error", err.Error())
		return devices
	}
	return append(devices, services...)
}

type postureAttributeClient interface {
	DevicePostureAttributes(context.Context, string) (cloudapi.DevicePostureAttributes, error)
}

func enrichDevicesFromCloud(devices []api.Device, cloudDevices []cloudapi.Device) {
	byIP := map[string]cloudapi.Device{}
	for _, device := range cloudDevices {
		for _, ip := range device.Addresses {
			if ip != "" {
				byIP[ip] = device
			}
		}
	}
	for i := range devices {
		var cloud cloudapi.Device
		var ok bool
		for _, ip := range devices[i].TailscaleIPs {
			cloud, ok = byIP[ip]
			if ok {
				break
			}
		}
		if !ok {
			continue
		}
		devices[i].Shared = devices[i].Shared || cloud.IsExternal
		if devices[i].PostureAttrs == nil {
			devices[i].PostureAttrs = map[string]any{}
		}
		if cloud.ClientVersion != "" {
			devices[i].PostureAttrs["node:tsVersion"] = cloud.ClientVersion
		}
		if cloud.OS != "" {
			devices[i].PostureAttrs["node:os"] = strings.ToLower(cloud.OS)
		}
		if len(cloud.PostureIdentity) > 0 {
			devices[i].PostureAttrs["node:postureIdentity"] = cloud.PostureIdentity
		}
	}
}

func enrichDeviceRolesFromCloudUsers(devices []api.Device, users []cloudapi.User) {
	rolesByLogin := map[string]string{}
	for _, user := range users {
		login := strings.TrimSpace(user.LoginName)
		role := strings.TrimSpace(user.Role)
		if login == "" || role == "" {
			continue
		}
		rolesByLogin[login] = role
	}
	for i := range devices {
		if devices[i].Owner == "" || len(devices[i].Tags) > 0 {
			continue
		}
		role := rolesByLogin[devices[i].Owner]
		if role == "" {
			continue
		}
		devices[i].Roles = mergeRoles(devices[i].Roles, []string{role})
	}
}

func enrichDevicePostureAttributes(ctx context.Context, client postureAttributeClient, devices []api.Device, cloudDevices []cloudapi.Device, logger *slog.Logger) {
	cloudByIP := map[string]cloudapi.Device{}
	for _, cloud := range cloudDevices {
		for _, ip := range cloud.Addresses {
			if ip != "" {
				cloudByIP[ip] = cloud
			}
		}
	}
	for i := range devices {
		var cloud cloudapi.Device
		var ok bool
		for _, ip := range devices[i].TailscaleIPs {
			cloud, ok = cloudByIP[ip]
			if ok {
				break
			}
		}
		if !ok {
			continue
		}
		deviceID := firstNonEmpty(cloud.NodeID, cloud.ID)
		if deviceID == "" {
			continue
		}
		attrs, err := client.DevicePostureAttributes(ctx, deviceID)
		if err != nil {
			if logger != nil {
				logger.Debug("cloud posture attributes unavailable", "device", deviceID, "error", err.Error())
			}
			continue
		}
		if len(attrs.Attributes) == 0 {
			continue
		}
		if devices[i].PostureAttrs == nil {
			devices[i].PostureAttrs = map[string]any{}
		}
		for key, value := range attrs.Attributes {
			if key != "" && value != nil {
				devices[i].PostureAttrs[key] = value
			}
		}
	}
}

func mergeRoles(existing, roles []string) []string {
	out := append([]string(nil), existing...)
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		found := false
		for _, existingRole := range out {
			if strings.EqualFold(existingRole, role) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, role)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func serviceDevicesFromCloud(services []cloudapi.VIPService) []api.Device {
	out := make([]api.Device, 0, len(services))
	for _, service := range services {
		name := strings.TrimSpace(service.Name)
		if name == "" {
			continue
		}
		ip := ""
		if len(service.Addrs) > 0 {
			ip = service.Addrs[0]
		}
		out = append(out, api.Device{
			ID:           name,
			Kind:         "service",
			Name:         name,
			IP:           ip,
			TailscaleIPs: append([]string(nil), service.Addrs...),
			Online:       true,
			Tags:         append([]string(nil), service.Tags...),
		})
	}
	return out
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
			if accessEdges, err := s.policyCache.EffectiveAccessEdges(rawPolicy, devices, policy.EdgeOptions{}); err == nil {
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
	ExpiresAt  time.Time
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
