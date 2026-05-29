package devtailnet

import (
	"strings"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
)

// APIKey unlocks an in-memory demo tailnet — no Tailscale Cloud API calls.
const APIKey = "tskey-api-tailor-dev"

// Name is the tailnet identifier returned to the frontend.
const Name = "demo.tailor.ts.net"

func IsDevAPIKey(key string) bool {
	return strings.TrimSpace(key) == APIKey
}

func Devices() []api.Device {
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
	}
}

func Policy() string {
	return policyHuJSON
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
