//go:build dev

package devtailnet

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
)

// Enabled reports whether this binary was built with the dev tag.
const Enabled = true

// APIKey unlocks an in-memory demo tailnet — no Tailscale Cloud API calls.
const APIKey = "tskey-api-tailor-dev"

// Name is the tailnet identifier returned to the frontend.
const Name = "demo.tailor.ts.net"

// SuperUserEmail owns a device with *:* ACL access for graph/simulation debugging.
const SuperUserEmail = "superadmin@demo.tailor.ts.net"

const maxSpawnBatch = 25

var (
	storeMu sync.RWMutex
	store   = newStore(seedDevices())
)

func IsDevAPIKey(key string) bool {
	return strings.TrimSpace(key) == APIKey
}

func Devices() []api.Device {
	storeMu.RLock()
	defer storeMu.RUnlock()
	return cloneDevices(store.devices)
}

func Policy() string {
	return policyHuJSON
}

// ResetStore reinitializes the in-memory demo device store (for tests).
func ResetStore() {
	storeMu.Lock()
	defer storeMu.Unlock()
	store = newStore(seedDevices())
}

type spawnPlanEntry struct {
	name          string
	owner         string
	osName        string
	tags          []string
	online        bool
	subnetRouter  bool
	routedSubnets []string
}

func SpawnDevices(request api.DevSpawnDevicesRequest) ([]api.Device, error) {
	plan, err := buildSpawnPlan(request)
	if err != nil {
		return nil, err
	}
	return spawnFromPlan(plan)
}

func PatchDevices(request api.DevPatchDevicesRequest) ([]api.Device, error) {
	patches := request.Devices
	if len(patches) == 0 {
		return nil, fmt.Errorf("patch at least one device")
	}
	if len(patches) > maxSpawnBatch {
		return nil, fmt.Errorf("patch at most %d devices at a time", maxSpawnBatch)
	}

	storeMu.Lock()
	defer storeMu.Unlock()

	patched := make([]api.Device, 0, len(patches))
	for _, patch := range patches {
		name := strings.TrimSpace(patch.Name)
		if name == "" {
			return nil, fmt.Errorf("patch device name is required")
		}
		idx := store.deviceIndexByName(name)
		if idx < 0 {
			return nil, fmt.Errorf("device %q not found", name)
		}
		device := store.devices[idx]
		if patch.Online != nil {
			device.Online = *patch.Online
			if device.Online {
				device.LastSeen = ""
			} else {
				device.LastSeen = time.Now().Add(-20 * time.Minute).UTC().Format(time.RFC3339)
			}
		}
		store.devices[idx] = device
		patched = append(patched, device)
	}
	return cloneDevices(patched), nil
}

func buildSpawnPlan(request api.DevSpawnDevicesRequest) ([]spawnPlanEntry, error) {
	defaultOwner := strings.TrimSpace(request.Owner)
	if defaultOwner == "" {
		defaultOwner = "spawn@demo.tailor.ts.net"
	}
	defaultOS := strings.TrimSpace(request.OS)
	if defaultOS == "" {
		defaultOS = "linux"
	}
	defaultOnline := true
	if request.Online != nil {
		defaultOnline = *request.Online
	}
	defaultTags := compactStrings(request.Tags)

	if len(request.Specs) > 0 {
		if len(request.Specs) > maxSpawnBatch {
			return nil, fmt.Errorf("spawn at most %d devices at a time", maxSpawnBatch)
		}
		plan := make([]spawnPlanEntry, 0, len(request.Specs))
		for _, spec := range request.Specs {
			name := strings.TrimSpace(spec.Name)
			if name == "" {
				return nil, fmt.Errorf("each spawn spec requires a name")
			}
			owner := strings.TrimSpace(spec.Owner)
			if owner == "" {
				owner = defaultOwner
			}
			osName := strings.TrimSpace(spec.OS)
			if osName == "" {
				osName = defaultOS
			}
			online := defaultOnline
			if spec.Online != nil {
				online = *spec.Online
			}
			tags := compactStrings(spec.Tags)
			if len(tags) == 0 {
				tags = defaultTags
			}
			plan = append(plan, spawnPlanEntry{
				name:          name,
				owner:         owner,
				osName:        osName,
				tags:          tags,
				online:        online,
				subnetRouter:  spec.SubnetRouter,
				routedSubnets: compactStrings(spec.RoutedSubnets),
			})
		}
		return plan, nil
	}

	names := compactStrings(request.Names)
	count := request.Count
	if len(names) > 0 {
		count = len(names)
	}
	if count <= 0 {
		count = 1
	}
	if count > maxSpawnBatch {
		return nil, fmt.Errorf("spawn at most %d devices at a time", maxSpawnBatch)
	}

	prefix := strings.TrimSpace(request.Prefix)
	if prefix == "" && len(names) == 0 {
		prefix = "worker"
	}

	plan := make([]spawnPlanEntry, 0, count)
	for i := range count {
		name := fmt.Sprintf("%s-%d", prefix, i+1)
		if len(names) > 0 {
			name = names[i]
		}
		plan = append(plan, spawnPlanEntry{
			name:   name,
			owner:  defaultOwner,
			osName: defaultOS,
			tags:   defaultTags,
			online: defaultOnline,
		})
	}
	return plan, nil
}

func spawnFromPlan(plan []spawnPlanEntry) ([]api.Device, error) {
	storeMu.Lock()
	defer storeMu.Unlock()

	available := store.availableSpawnIPs()
	if available < len(plan) {
		return nil, fmt.Errorf("cannot spawn %d devices: only %d demo IPs available in 100.100.0.100-250", len(plan), available)
	}

	spawned := make([]api.Device, 0, len(plan))
	for _, entry := range plan {
		if store.deviceIndexByName(entry.name) >= 0 {
			return nil, fmt.Errorf("device %q already exists", entry.name)
		}
		ip, err := store.allocateIP()
		if err != nil {
			return nil, err
		}
		id, _ := store.allocateSpawnID()

		device := api.Device{
			ID:            id,
			Name:          entry.name,
			IP:            ip,
			TailscaleIPs:  []string{ip},
			OS:            entry.osName,
			Online:        entry.online,
			Owner:         entry.owner,
			Tags:          entry.tags,
			SubnetRouter:  entry.subnetRouter,
			RoutedSubnets: entry.routedSubnets,
		}
		if !entry.online {
			device.LastSeen = time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339)
		}
		store.devices = append(store.devices, device)
		spawned = append(spawned, device)
	}
	return cloneDevices(spawned), nil
}

type deviceStore struct {
	devices       []api.Device
	nextSeq       int
	nextHostOctet int
}

func newStore(devices []api.Device) *deviceStore {
	return &deviceStore{
		devices:       cloneDevices(devices),
		nextHostOctet: 100,
	}
}

func cloneDevices(devices []api.Device) []api.Device {
	out := make([]api.Device, len(devices))
	copy(out, devices)
	return out
}

func (s *deviceStore) deviceIndexByName(name string) int {
	for i, d := range s.devices {
		if d.Name == name {
			return i
		}
	}
	return -1
}

func (s *deviceStore) availableSpawnIPs() int {
	used := make(map[string]bool, len(s.devices))
	for _, d := range s.devices {
		used[d.IP] = true
	}
	const minOctet, maxOctet = 100, 250
	n := 0
	for octet := minOctet; octet <= maxOctet; octet++ {
		if !used[fmt.Sprintf("100.100.0.%d", octet)] {
			n++
		}
	}
	return n
}

func (s *deviceStore) allocateIP() (string, error) {
	used := make(map[string]bool, len(s.devices))
	for _, d := range s.devices {
		used[d.IP] = true
	}
	const minOctet, maxOctet = 100, 250
	for range maxOctet - minOctet + 1 {
		candidate := fmt.Sprintf("100.100.0.%d", s.nextHostOctet)
		s.nextHostOctet++
		if s.nextHostOctet > maxOctet {
			s.nextHostOctet = minOctet
		}
		if !used[candidate] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no available IPs in 100.100.0.100-250 range")
}

func (s *deviceStore) allocateSpawnID() (id string, seq int) {
	used := make(map[string]bool, len(s.devices))
	maxSeq := 0
	for _, d := range s.devices {
		used[d.ID] = true
		var n int
		if _, err := fmt.Sscanf(d.ID, "dev-spawn-%d", &n); err == nil && n > maxSeq {
			maxSeq = n
		}
	}
	for {
		maxSeq++
		candidate := fmt.Sprintf("dev-spawn-%d", maxSeq)
		if !used[candidate] {
			s.nextSeq = maxSeq
			return candidate, maxSeq
		}
	}
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

func seedDevices() []api.Device {
	lastWeek := time.Now().Add(-7 * 24 * time.Hour).UTC().Format(time.RFC3339)
	lastHour := time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	lastDay := time.Now().Add(-26 * time.Hour).UTC().Format(time.RFC3339)

	return []api.Device{
		// Engineering — laptops and phones
		{
			ID: "dev-alice-laptop", Name: "alice-laptop", IP: "100.100.0.1",
			TailscaleIPs: []string{"100.100.0.1"}, OS: "macOS", Online: true,
			Owner: "alice@demo.tailor.ts.net", Tags: []string{}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-alice-phone", Name: "alice-phone", IP: "100.100.0.2",
			TailscaleIPs: []string{"100.100.0.2"}, OS: "iOS", Online: true,
			Owner: "alice@demo.tailor.ts.net", Tags: []string{}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-bob-laptop", Name: "bob-laptop", IP: "100.100.0.3",
			TailscaleIPs: []string{"100.100.0.3"}, OS: "linux", Online: true,
			Owner: "bob@demo.tailor.ts.net", Tags: []string{}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-bob-workstation", Name: "bob-workstation", IP: "100.100.0.4",
			TailscaleIPs: []string{"100.100.0.4"}, OS: "windows", Online: false, LastSeen: lastWeek,
			Owner: "bob@demo.tailor.ts.net", Tags: []string{}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-carol-macbook", Name: "carol-macbook", IP: "100.100.0.5",
			TailscaleIPs: []string{"100.100.0.5"}, OS: "macOS", Online: true,
			Owner: "carol@demo.tailor.ts.net", Tags: []string{}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-dave-dev-laptop", Name: "dave-dev-laptop", IP: "100.100.0.6",
			TailscaleIPs: []string{"100.100.0.6"}, OS: "linux", Online: true,
			Owner: "dave@demo.tailor.ts.net", Tags: []string{}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-maya-laptop", Name: "maya-laptop", IP: "100.100.0.7",
			TailscaleIPs: []string{"100.100.0.7"}, OS: "macOS", Online: false, LastSeen: lastHour,
			Owner: "maya@demo.tailor.ts.net", Tags: []string{}, SubnetRouter: false, RoutedSubnets: []string{},
		},

		// SaaS tier
		{
			ID: "dev-web-frontend", Name: "web-frontend", IP: "100.100.0.10",
			TailscaleIPs: []string{"100.100.0.10"}, OS: "linux", Online: true,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:web"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-web-api", Name: "web-api", IP: "100.100.0.11",
			TailscaleIPs: []string{"100.100.0.11"}, OS: "linux", Online: true,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:web"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-db-primary", Name: "db-primary", IP: "100.100.0.20",
			TailscaleIPs: []string{"100.100.0.20"}, OS: "linux", Online: true,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:db"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-db-replica", Name: "db-replica", IP: "100.100.0.21",
			TailscaleIPs: []string{"100.100.0.21"}, OS: "linux", Online: false, LastSeen: lastHour,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:db"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-ci-runner", Name: "ci-runner", IP: "100.100.0.30",
			TailscaleIPs: []string{"100.100.0.30"}, OS: "linux", Online: true,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:ci"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-ci-cache", Name: "ci-cache", IP: "100.100.0.31",
			TailscaleIPs: []string{"100.100.0.31"}, OS: "linux", Online: true,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:ci"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-grafana", Name: "grafana", IP: "100.100.0.40",
			TailscaleIPs: []string{"100.100.0.40"}, OS: "linux", Online: true,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:monitoring"}, SubnetRouter: false, RoutedSubnets: []string{},
		},

		// HQ / legacy VPC
		{
			ID: "dev-edge-router", Name: "hq-vpn-connector", IP: "100.100.0.50",
			TailscaleIPs: []string{"100.100.0.50"}, OS: "linux", Online: true,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:prod"}, SubnetRouter: true,
			RoutedSubnets: []string{"10.20.0.0/24"},
		},
		{
			ID: "dev-bastion", Name: "bastion", IP: "100.100.0.51",
			TailscaleIPs: []string{"100.100.0.51"}, OS: "linux", Online: true,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:prod"}, SubnetRouter: false, RoutedSubnets: []string{},
		},

		// Kubernetes — production cluster (EKS-style)
		{
			ID: "dev-k8s-prod-connector", Name: "k8s-prod-connector", IP: "100.100.0.80",
			TailscaleIPs: []string{"100.100.0.80"}, OS: "linux", Online: true,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:k8s-prod"}, SubnetRouter: true,
			RoutedSubnets: []string{"10.30.0.0/16"},
		},
		{
			ID: "dev-k8s-prod-cp", Name: "k8s-prod-cp-01", IP: "100.100.0.81",
			TailscaleIPs: []string{"100.100.0.81"}, OS: "linux", Online: true,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:k8s-prod"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-k8s-prod-worker-1", Name: "k8s-prod-worker-01", IP: "100.100.0.82",
			TailscaleIPs: []string{"100.100.0.82"}, OS: "linux", Online: true,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:k8s-prod"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-k8s-prod-worker-2", Name: "k8s-prod-worker-02", IP: "100.100.0.83",
			TailscaleIPs: []string{"100.100.0.83"}, OS: "linux", Online: true,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:k8s-prod"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-k8s-prod-worker-3", Name: "k8s-prod-worker-03", IP: "100.100.0.84",
			TailscaleIPs: []string{"100.100.0.84"}, OS: "linux", Online: false, LastSeen: lastDay,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:k8s-prod"}, SubnetRouter: false, RoutedSubnets: []string{},
		},

		// Kubernetes — staging cluster (GKE-style)
		{
			ID: "dev-k8s-staging-connector", Name: "k8s-staging-connector", IP: "100.100.0.90",
			TailscaleIPs: []string{"100.100.0.90"}, OS: "linux", Online: true,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:k8s-staging"}, SubnetRouter: true,
			RoutedSubnets: []string{"10.31.0.0/16"},
		},
		{
			ID: "dev-k8s-staging-cp", Name: "k8s-staging-cp-01", IP: "100.100.0.91",
			TailscaleIPs: []string{"100.100.0.91"}, OS: "linux", Online: true,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:k8s-staging"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-k8s-staging-worker-1", Name: "k8s-staging-worker-01", IP: "100.100.0.92",
			TailscaleIPs: []string{"100.100.0.92"}, OS: "linux", Online: true,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:k8s-staging"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-k8s-staging-worker-2", Name: "k8s-staging-worker-02", IP: "100.100.0.93",
			TailscaleIPs: []string{"100.100.0.93"}, OS: "linux", Online: true,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:k8s-staging"}, SubnetRouter: false, RoutedSubnets: []string{},
		},

		// Platform services
		{
			ID: "dev-artifact-registry", Name: "artifact-registry", IP: "100.100.0.94",
			TailscaleIPs: []string{"100.100.0.94"}, OS: "linux", Online: true,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:platform"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-secrets-agent", Name: "secrets-vault-agent", IP: "100.100.0.95",
			TailscaleIPs: []string{"100.100.0.95"}, OS: "linux", Online: true,
			Owner: "platform-ops@demo.tailor.ts.net", Tags: []string{"tag:platform"}, SubnetRouter: false, RoutedSubnets: []string{},
		},

		// External / admin
		{
			ID: "dev-contractor", Name: "contractor-laptop", IP: "100.100.0.60",
			TailscaleIPs: []string{"100.100.0.60"}, OS: "macOS", Online: true,
			Owner: "contractor@demo.tailor.ts.net", Tags: []string{"tag:contractor"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
		{
			ID: "dev-superadmin-console", Name: "superadmin-console", IP: "100.100.0.70",
			TailscaleIPs: []string{"100.100.0.70"}, OS: "linux", Online: true,
			Owner: SuperUserEmail, Tags: []string{}, SubnetRouter: false, RoutedSubnets: []string{},
		},
	}
}

const policyHuJSON = `{
	// Tailor demo tailnet — enterprise-shaped ACLs for graph styling and draft simulation.
	"groups": {
		"group:eng": [
			"alice@demo.tailor.ts.net",
			"bob@demo.tailor.ts.net",
			"carol@demo.tailor.ts.net",
			"dave@demo.tailor.ts.net",
		],
		"group:ops": [
			"ops@demo.tailor.ts.net",
		],
		"group:platform": [
			"platform-ops@demo.tailor.ts.net",
			"ops@demo.tailor.ts.net",
		],
		"group:data": [
			"maya@demo.tailor.ts.net",
		],
		"group:superuser": [
			"superadmin@demo.tailor.ts.net",
		],
	},
	"tagOwners": {
		"tag:web": ["group:ops"],
		"tag:db": ["group:ops"],
		"tag:ci": ["group:ops"],
		"tag:monitoring": ["group:ops"],
		"tag:prod": ["group:platform"],
		"tag:platform": ["group:platform"],
		"tag:k8s-prod": ["group:platform"],
		"tag:k8s-staging": ["group:platform"],
		"tag:contractor": ["group:ops"],
	},
	"hosts": {
		"prod-app": "10.20.0.10",
		"prod-cache": "10.20.0.20",
		"k8s-prod-api": "10.30.0.1",
		"k8s-prod-ingress": "10.30.0.50",
		"k8s-staging-api": "10.31.0.1",
	},
	"acls": [
		{"action": "accept", "src": ["group:superuser"], "dst": ["*:*"]},
		{"action": "accept", "src": ["group:eng"], "dst": ["tag:web:443"]},
		{"action": "accept", "src": ["group:eng"], "dst": ["tag:k8s-staging:443,6443"]},
		{"action": "accept", "src": ["carol@demo.tailor.ts.net"], "dst": ["tag:ci:22"]},
		{"action": "accept", "src": ["dave@demo.tailor.ts.net"], "dst": ["tag:db:22"]},
		{"action": "accept", "src": ["bob@demo.tailor.ts.net"], "dst": ["tag:db:22"]},
		{"action": "accept", "src": ["tag:ci"], "dst": ["tag:web:443"]},
		{"action": "accept", "src": ["tag:ci"], "dst": ["tag:k8s-staging:443"]},
		{"action": "accept", "src": ["group:platform"], "dst": ["tag:k8s-prod:443,6443"]},
		{"action": "accept", "src": ["group:platform"], "dst": ["tag:k8s-prod:10250"]},
		{"action": "accept", "src": ["group:platform"], "dst": ["tag:prod:22"]},
		{"action": "accept", "src": ["group:platform"], "dst": ["tag:platform:443"]},
		{"action": "accept", "src": ["tag:k8s-prod"], "dst": ["tag:db:5432"]},
		{"action": "accept", "src": ["tag:k8s-staging"], "dst": ["tag:web:443"]},
		{"action": "accept", "src": ["tag:k8s-staging"], "dst": ["tag:db:5432"]},
		{"action": "accept", "src": ["group:ops"], "dst": ["prod-app:80,443"]},
		{"action": "accept", "src": ["group:ops"], "dst": ["prod-cache:6379"]},
		{"action": "accept", "src": ["group:ops"], "dst": ["k8s-prod-api:443"]},
		{"action": "accept", "src": ["group:data"], "dst": ["tag:db:5432"]},
		{"action": "accept", "src": ["group:data"], "dst": ["tag:monitoring:3000"]},
		{"action": "accept", "src": ["alice@demo.tailor.ts.net"], "dst": ["tag:monitoring:3000"]},
		{"action": "accept", "src": ["tag:web"], "dst": ["tag:db:5432"]},
		{"action": "accept", "src": ["tag:web"], "dst": ["k8s-prod-ingress:443"]},
		{"action": "accept", "src": ["group:eng"], "dst": ["autogroup:self:22"]},
		{"action": "accept", "src": ["tag:contractor"], "dst": ["tag:web:443"]},
		{"action": "accept", "src": ["tag:prod"], "dst": ["tag:monitoring:9090"]},
		{"action": "accept", "src": ["platform-ops@demo.tailor.ts.net"], "dst": ["tag:prod:22,443"]},
	],
	"ssh": [
		{
			"action": "accept",
			"src": ["group:platform"],
			"dst": ["tag:prod", "tag:k8s-prod"],
			"users": ["root", "deploy"],
		},
		{
			"action": "accept",
			"src": ["group:eng"],
			"dst": ["tag:k8s-staging"],
			"users": ["ubuntu", "deploy"],
		},
	],
	"grants": [
		{
			"src": ["alice@demo.tailor.ts.net"],
			"dst": ["tag:web"],
			"app": {
				"tailscale.com/cap/file-sharing": [{"shares": ["eng"]}],
			},
		},
		{
			"src": ["group:platform"],
			"dst": ["tag:platform"],
			"app": {
				"tailscale.com/cap/kubernetes": [{"roles": ["view", "exec"]}],
			},
		},
	],
}`
