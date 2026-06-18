package api

import "encoding/json"

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Build   string `json:"build,omitempty"`
}

type SetupHint struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type TailscaleSetupInfo struct {
	Required bool        `json:"required"`
	Hints    []SetupHint `json:"hints,omitempty"`
}

type LocalAPIStatusResponse struct {
	Available        bool                `json:"available"`
	LocalAPIEndpoint string              `json:"localApiEndpoint"`
	Error            string              `json:"error,omitempty"`
	Setup            *TailscaleSetupInfo `json:"setup,omitempty"`
}

type Device struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	IP            string   `json:"ip"`
	TailscaleIPs  []string `json:"tailscaleIps"`
	OS            string   `json:"os"`
	Online        bool     `json:"online"`
	Owner         string   `json:"owner"`
	Tags          []string `json:"tags"`
	SubnetRouter  bool     `json:"subnetRouter"`
	RoutedSubnets []string `json:"routedSubnets"`
	LastSeen      string   `json:"lastSeen,omitempty"`
}

type EdgeKind string

const (
	EdgeKindOwner  EdgeKind = "owner"
	EdgeKindTag    EdgeKind = "tag"
	EdgeKindSubnet EdgeKind = "subnet"
	EdgeKindACL    EdgeKind = "acl"
)

type AccessScope string

const (
	AccessScopeUnknown AccessScope = ""
	AccessScopeSSH     AccessScope = "ssh"
	AccessScopeHTTP    AccessScope = "http"
	AccessScopeBroad   AccessScope = "broad"
	AccessScopeCustom  AccessScope = "custom"
	AccessScopeLimited AccessScope = "limited"
	AccessScopeNone    AccessScope = "none"
)

type Edge struct {
	ID           string      `json:"id"`
	From         string      `json:"from"`
	To           string      `json:"to"`
	Kind         EdgeKind    `json:"kind"`
	Labels       []string    `json:"labels,omitempty"`
	Protocols    []string    `json:"protocols,omitempty"`
	Ports        []string    `json:"ports,omitempty"`
	AccessScope  AccessScope `json:"accessScope,omitempty"`
	PolicyRefs   []PolicyRef `json:"policyRefs,omitempty"`
	Perspectives []string    `json:"perspectives,omitempty"`
}

type PolicyRef struct {
	Section string `json:"section"`
	Index   int    `json:"index"`
	Src     string `json:"src,omitempty"`
	Dst     string `json:"dst,omitempty"`
}

type TopologyResponse struct {
	Devices      []Device            `json:"devices"`
	Edges        []Edge              `json:"edges"`
	Tailnet      string              `json:"tailnet"`
	Setup        *TailscaleSetupInfo `json:"setup,omitempty"`
	StagedDrafts []StagedDraft       `json:"stagedDrafts,omitempty"`
}

type CloudAuthRequest struct {
	Tailnet string `json:"tailnet"`
	APIKey  string `json:"apiKey"`
}

type CloudAuthStatusResponse struct {
	Authenticated         bool   `json:"authenticated"`
	Tailnet               string `json:"tailnet,omitempty"`
	HasPolicy             bool   `json:"hasPolicy"`
	DevMode               bool   `json:"devMode,omitempty"`
	CallerRole            string `json:"callerRole,omitempty"`
	CanEditPolicy         bool   `json:"canEditPolicy"`
	HasAppCapabilityGrant bool   `json:"hasAppCapabilityGrant,omitempty"`
	AppCapability         string `json:"appCapability,omitempty"`
	NeedsSetupGrant       bool   `json:"needsSetupGrant,omitempty"`
	BootstrapActive       bool   `json:"bootstrapActive,omitempty"`
	BootstrapExpiresAt    string `json:"bootstrapExpiresAt,omitempty"`
	StatusMessage         string `json:"statusMessage,omitempty"`
	SetupGrantSnippet     string `json:"setupGrantSnippet,omitempty"`
}

type SetupGrantRequest struct {
	Grant *GrantDraft `json:"grant,omitempty"`
}

type SetupGrantResponse struct {
	Tailnet            string `json:"tailnet,omitempty"`
	AppCapability      string `json:"appCapability,omitempty"`
	HasAppCapabilityGrant bool `json:"hasAppCapabilityGrant"`
	CallerRole         string `json:"callerRole,omitempty"`
	CanEditPolicy      bool   `json:"canEditPolicy"`
	BootstrapActive    bool   `json:"bootstrapActive,omitempty"`
	BootstrapExpiresAt string `json:"bootstrapExpiresAt,omitempty"`
	StatusMessage      string `json:"statusMessage,omitempty"`
	SetupGrantSnippet  string `json:"setupGrantSnippet,omitempty"`
}

type PolicyResponse struct {
	Tailnet string `json:"tailnet"`
	HuJSON  string `json:"hujson"`
}

type PolicyMapResponse struct {
	Tailnet    string          `json:"tailnet"`
	HuJSON     string          `json:"hujson"`
	Sections   []PolicySection `json:"sections"`
	ParseError string          `json:"parseError,omitempty"`
}

type PolicySection struct {
	Name        string               `json:"name"`
	Type        string               `json:"type"`
	Supported   bool                 `json:"supported"`
	Count       int                  `json:"count"`
	Entries     []PolicySectionEntry `json:"entries,omitempty"`
	Raw         any                  `json:"raw,omitempty"`
	Description string               `json:"description,omitempty"`
}

type PolicySectionEntry struct {
	Label     string   `json:"label"`
	Summary   string   `json:"summary,omitempty"`
	Selectors []string `json:"selectors,omitempty"`
	Value     any      `json:"value,omitempty"`
}

type PolicyDraftRequest struct {
	Sources      []string `json:"sources"`
	Destinations []string `json:"destinations"`
	Ports        []string `json:"ports"`
	Protocol     string   `json:"protocol,omitempty"`
}

type PolicyDraftResponse struct {
	Tailnet string   `json:"tailnet"`
	Rule    ACLDraft `json:"rule"`
	HuJSON  string   `json:"hujson"`
}

type ACLDraft struct {
	Action string   `json:"action"`
	Src    []string `json:"src"`
	Dst    []string `json:"dst"`
	Proto  string   `json:"proto,omitempty"`
}

type GrantDraft struct {
	Src []string       `json:"src"`
	Dst []string       `json:"dst"`
	IP  []string       `json:"ip,omitempty"`
	App map[string]any `json:"app,omitempty"`
}

type PolicyMutation struct {
	Type    string          `json:"type"`
	Section string          `json:"section,omitempty"`
	Key     string          `json:"key,omitempty"`
	Index   int             `json:"index,omitempty"`
	Rule    ACLDraft        `json:"rule,omitempty"`
	Grant   GrantDraft      `json:"grant,omitempty"`
	Host    string          `json:"host,omitempty"`
	IPSet   []string        `json:"ipSet,omitempty"`
	Members []string        `json:"members,omitempty"`
	Owners  []string        `json:"owners,omitempty"`
	Value   json.RawMessage `json:"value,omitempty"`
}

type PolicyMutationRequest struct {
	HuJSON   string         `json:"hujson,omitempty"`
	Mutation PolicyMutation `json:"mutation"`
}

type PolicyMutationResponse struct {
	Tailnet string `json:"tailnet"`
	HuJSON  string `json:"hujson"`
	Summary string `json:"summary,omitempty"`
}

type PolicyValidateRequest struct {
	HuJSON string `json:"hujson"`
}

type PolicyValidateResponse struct {
	Valid   bool     `json:"valid"`
	Tailnet string   `json:"tailnet"`
	Errors  []string `json:"errors,omitempty"`
}

type PolicyStageRequest struct {
	HuJSON  string `json:"hujson"`
	Source  string `json:"source,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type PolicySaveRequest struct {
	DraftID   string `json:"draftId"`
	DraftHash string `json:"draftHash"`
}

type PolicySaveResponse struct {
	Saved   bool   `json:"saved"`
	Tailnet string `json:"tailnet"`
	HuJSON  string `json:"hujson"`
}

type StagedDraft struct {
	ID         string                      `json:"id"`
	Source     string                      `json:"source"`
	Tailnet    string                      `json:"tailnet"`
	BaseHash   string                      `json:"baseHash"`
	DraftHash  string                      `json:"draftHash"`
	HuJSON     string                      `json:"hujson,omitempty"`
	Valid      bool                        `json:"valid"`
	Errors     []string                    `json:"errors,omitempty"`
	Evaluation PolicyEvaluateDraftResponse `json:"evaluation"`
	Summary    string                      `json:"summary,omitempty"`
	CreatedAt  string                      `json:"createdAt"`
	UpdatedAt  string                      `json:"updatedAt"`
}

type PolicyStageResponse struct {
	Draft StagedDraft `json:"draft"`
}

type PolicyStagedResponse struct {
	Drafts []StagedDraft `json:"drafts"`
}

type PolicyStagedDraftResponse struct {
	Draft StagedDraft `json:"draft"`
}

type PolicyDiscardStagedResponse struct {
	Discarded bool   `json:"discarded"`
	DraftID   string `json:"draftId"`
}

type PolicyEvaluateDraftRequest struct {
	HuJSON      string `json:"hujson"`
	Perspective string `json:"perspective,omitempty"`
}

type PolicyEvaluateDraftResponse struct {
	Tailnet             string               `json:"tailnet"`
	Added               []PolicyEdgeChange   `json:"added"`
	Removed             []PolicyEdgeChange   `json:"removed"`
	Unchanged           []PolicyEdgeChange   `json:"unchanged"`
	Changed             []PolicyEdgeChange   `json:"changed"`
	BroadAccess         []Edge               `json:"broadAccess"`
	VisibleDeviceIDs    []string             `json:"visibleDeviceIds"`
	UnresolvedSelectors []UnresolvedSelector `json:"unresolvedSelectors"`
	UnsupportedSections []string             `json:"unsupportedSections"`
	ApplicationGrants   []ApplicationGrant   `json:"applicationGrants"`
}

type PolicyEdgeChange struct {
	State string `json:"state"`
	Edge  Edge   `json:"edge"`
	Saved *Edge  `json:"saved,omitempty"`
	Draft *Edge  `json:"draft,omitempty"`
}

type UnresolvedSelector struct {
	Section  string `json:"section"`
	Index    int    `json:"index"`
	Selector string `json:"selector"`
	Role     string `json:"role"`
}

type ApplicationGrant struct {
	Section      string   `json:"section"`
	Index        int      `json:"index"`
	Src          []string `json:"src"`
	Dst          []string `json:"dst"`
	Capabilities []string `json:"capabilities"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// DevSpawnDeviceSpec describes one device in a demo spawn batch (per-device fields override request defaults).
type DevSpawnDeviceSpec struct {
	Name          string   `json:"name"`
	Owner         string   `json:"owner,omitempty"`
	OS            string   `json:"os,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Online        *bool    `json:"online,omitempty"`
	SubnetRouter  bool     `json:"subnetRouter,omitempty"`
	RoutedSubnets []string `json:"routedSubnets,omitempty"`
}

type DevSpawnDevicesRequest struct {
	Count  int                  `json:"count,omitempty"`
	Prefix string               `json:"prefix,omitempty"`
	Names  []string             `json:"names,omitempty"`
	Specs  []DevSpawnDeviceSpec `json:"specs,omitempty"`
	Owner  string               `json:"owner,omitempty"`
	OS     string               `json:"os,omitempty"`
	Tags   []string             `json:"tags,omitempty"`
	Online *bool                `json:"online,omitempty"`
}

type DevSpawnDevicesResponse struct {
	Tailnet string   `json:"tailnet"`
	Spawned []Device `json:"spawned"`
	Devices []Device `json:"devices"`
}

type DevPatchDeviceSpec struct {
	Name   string `json:"name"`
	Online *bool  `json:"online,omitempty"`
}

type DevPatchDevicesRequest struct {
	Devices []DevPatchDeviceSpec `json:"devices"`
}

type DevPatchDevicesResponse struct {
	Tailnet string   `json:"tailnet"`
	Patched []Device `json:"patched"`
	Devices []Device `json:"devices"`
}

type SocketMessage struct {
	Type      string `json:"type"`
	RequestID string `json:"requestId,omitempty"`
	Payload   any    `json:"payload,omitempty"`
}

const (
	SocketMessageTopologySnapshot    = "topology.snapshot"
	SocketMessageLocalAPIUnavailable = "localapi.unavailable"
)
