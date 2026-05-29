package api

import "encoding/json"

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
	Build   string `json:"build,omitempty"`
}

type LocalAPIStatusResponse struct {
	Available        bool   `json:"available"`
	LocalAPIEndpoint string `json:"localApiEndpoint"`
	Error            string `json:"error,omitempty"`
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
	Devices []Device `json:"devices"`
	Edges   []Edge   `json:"edges"`
	Tailnet string   `json:"tailnet"`
}

type CloudAuthRequest struct {
	Tailnet string `json:"tailnet"`
	APIKey  string `json:"apiKey"`
}

type CloudAuthStatusResponse struct {
	Authenticated bool   `json:"authenticated"`
	Tailnet       string `json:"tailnet,omitempty"`
	HasPolicy     bool   `json:"hasPolicy"`
	DevMode       bool   `json:"devMode,omitempty"`
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

type PolicySaveResponse struct {
	Saved   bool   `json:"saved"`
	Tailnet string `json:"tailnet"`
	HuJSON  string `json:"hujson"`
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
