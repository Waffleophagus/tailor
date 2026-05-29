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

func SpawnDevices(request api.DevSpawnDevicesRequest) ([]api.Device, error) {
	names := compactStrings(request.Names)
	count := request.Count
	if len(names) > 0 {
		count = len(names)
	}
	if count <= 0 {
		count = 1
	}
	if count > 20 {
		return nil, fmt.Errorf("spawn at most 20 devices at a time")
	}

	prefix := strings.TrimSpace(request.Prefix)
	if prefix == "" && len(names) == 0 {
		prefix = "worker"
	}
	owner := strings.TrimSpace(request.Owner)
	if owner == "" {
		owner = "spawn@demo.tailor.ts.net"
	}
	osName := strings.TrimSpace(request.OS)
	if osName == "" {
		osName = "linux"
	}
	online := true
	if request.Online != nil {
		online = *request.Online
	}
	tags := compactStrings(request.Tags)

	storeMu.Lock()
	defer storeMu.Unlock()

	available := store.availableSpawnIPs()
	if available < count {
		return nil, fmt.Errorf("cannot spawn %d devices: only %d demo IPs available in 100.100.0.100-250", count, available)
	}

	spawned := make([]api.Device, 0, count)
	for i := range count {
		ip, err := store.allocateIP()
		if err != nil {
			return nil, err
		}
		id, seq := store.allocateSpawnID()

		name := fmt.Sprintf("%s-%d", prefix, seq)
		if len(names) > 0 {
			name = names[i]
		}

		device := api.Device{
			ID:            id,
			Name:          name,
			IP:            ip,
			TailscaleIPs:  []string{ip},
			OS:            osName,
			Online:        online,
			Owner:         owner,
			Tags:          tags,
			SubnetRouter:  false,
			RoutedSubnets: []string{},
		}
		if !online {
			device.LastSeen = time.Now().Add(-30 * time.Minute).UTC().Format(time.RFC3339)
		}
		store.devices = append(store.devices, device)
		spawned = append(spawned, device)
	}

	return cloneDevices(spawned), nil
}

type deviceStore struct {
	devices        []api.Device
	nextSeq        int
	nextHostOctet  int
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

	return []api.Device{
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
		{
			ID: "dev-edge-router", Name: "edge-router", IP: "100.100.0.50",
			TailscaleIPs: []string{"100.100.0.50"}, OS: "linux", Online: true,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:prod"}, SubnetRouter: true,
			RoutedSubnets: []string{"10.20.0.0/24"},
		},
		{
			ID: "dev-bastion", Name: "bastion", IP: "100.100.0.51",
			TailscaleIPs: []string{"100.100.0.51"}, OS: "linux", Online: true,
			Owner: "ops@demo.tailor.ts.net", Tags: []string{"tag:prod"}, SubnetRouter: false, RoutedSubnets: []string{},
		},
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
	// Tailor demo tailnet — varied ACL scopes for graph styling and draft simulation.
	"groups": {
		"group:eng": [
			"alice@demo.tailor.ts.net",
			"bob@demo.tailor.ts.net",
		],
		"group:ops": [
			"ops@demo.tailor.ts.net",
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
		"tag:prod": ["group:ops"],
		"tag:contractor": ["group:ops"],
	},
	"hosts": {
		"prod-app": "10.20.0.10",
		"prod-cache": "10.20.0.20",
	},
	"acls": [
		{"action": "accept", "src": ["group:superuser"], "dst": ["*:*"]},
		{"action": "accept", "src": ["group:eng"], "dst": ["tag:web:443"]},
		{"action": "accept", "src": ["bob@demo.tailor.ts.net"], "dst": ["tag:db:22"]},
		{"action": "accept", "src": ["tag:ci"], "dst": ["tag:web:443"]},
		{"action": "accept", "src": ["group:ops"], "dst": ["prod-app:80,443"]},
		{"action": "accept", "src": ["group:ops"], "dst": ["prod-cache:6379"]},
		{"action": "accept", "src": ["alice@demo.tailor.ts.net"], "dst": ["tag:monitoring:3000"]},
		{"action": "accept", "src": ["tag:web"], "dst": ["tag:db:5432"]},
		{"action": "accept", "src": ["group:eng"], "dst": ["autogroup:self:22"]},
		{"action": "accept", "src": ["tag:contractor"], "dst": ["tag:web:443"]},
	],
	"ssh": [
		{
			"action": "accept",
			"src": ["group:ops"],
			"dst": ["tag:prod"],
			"users": ["root", "deploy"],
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
	],
}`
