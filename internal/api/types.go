package api

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
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
}

type CloudAuthRequest struct {
	Tailnet string `json:"tailnet"`
	APIKey  string `json:"apiKey"`
}

type CloudAuthStatusResponse struct {
	Authenticated bool   `json:"authenticated"`
	Tailnet       string `json:"tailnet,omitempty"`
	HasPolicy     bool   `json:"hasPolicy"`
}

type PolicyResponse struct {
	Tailnet string `json:"tailnet"`
	HuJSON  string `json:"hujson"`
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

type ErrorResponse struct {
	Error string `json:"error"`
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
