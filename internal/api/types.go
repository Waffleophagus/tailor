package api

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type LocalAPIStatusResponse struct {
	Available  bool   `json:"available"`
	SocketPath string `json:"socketPath"`
	Error      string `json:"error,omitempty"`
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

type Edge struct {
	ID     string   `json:"id"`
	From   string   `json:"from"`
	To     string   `json:"to"`
	Kind   EdgeKind `json:"kind"`
	Labels []string `json:"labels,omitempty"`
}

type TopologyResponse struct {
	Devices []Device `json:"devices"`
	Edges   []Edge   `json:"edges"`
}
