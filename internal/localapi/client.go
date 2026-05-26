package localapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
)

const DefaultSocketPath = "/var/run/tailscale/tailscaled.sock"

var ErrUnavailable = errors.New("tailscale LocalAPI unavailable")

type Client struct {
	socketPath string
	httpClient *http.Client
}

func New(socketPath string) *Client {
	if socketPath == "" {
		socketPath = DefaultSocketPath
	}

	dialer := &net.Dialer{Timeout: 2 * time.Second}
	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					return dialer.DialContext(ctx, "unix", socketPath)
				},
			},
		},
	}
}

func (c *Client) SocketPath() string {
	return c.socketPath
}

func (c *Client) Status(ctx context.Context) ([]api.Device, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://local-tailscaled.sock/localapi/v0/status", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status endpoint returned %s", ErrUnavailable, resp.Status)
	}

	var status Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("decode tailscale status: %w", err)
	}

	return DevicesFromStatus(status), nil
}

type Status struct {
	Self *Peer           `json:"Self"`
	Peer map[string]Peer `json:"Peer"`
	User map[string]User `json:"User"`
}

type Peer struct {
	ID            string    `json:"ID"`
	PublicKey     string    `json:"PublicKey"`
	HostName      string    `json:"HostName"`
	DNSName       string    `json:"DNSName"`
	TailscaleIPs  []string  `json:"TailscaleIPs"`
	AllowedIPs    []string  `json:"AllowedIPs"`
	PrimaryRoutes []string  `json:"PrimaryRoutes"`
	OS            string    `json:"OS"`
	UserID        int64     `json:"UserID"`
	Tags          []string  `json:"Tags"`
	Online        bool      `json:"Online"`
	LastSeen      time.Time `json:"LastSeen"`
}

type User struct {
	ID          int64  `json:"ID"`
	LoginName   string `json:"LoginName"`
	DisplayName string `json:"DisplayName"`
}

func DevicesFromStatus(status Status) []api.Device {
	devices := make([]api.Device, 0, len(status.Peer)+1)
	if status.Self != nil {
		devices = append(devices, deviceFromPeer(*status.Self, status.User))
	}

	keys := make([]string, 0, len(status.Peer))
	for key := range status.Peer {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		devices = append(devices, deviceFromPeer(status.Peer[key], status.User))
	}

	return devices
}

func deviceFromPeer(peer Peer, users map[string]User) api.Device {
	id := firstNonEmpty(peer.ID, peer.PublicKey, peer.DNSName, peer.HostName)
	name := strings.TrimSuffix(firstNonEmpty(peer.DNSName, peer.HostName, id), ".")
	ip := ""
	if len(peer.TailscaleIPs) > 0 {
		ip = peer.TailscaleIPs[0]
	}

	lastSeen := ""
	if !peer.LastSeen.IsZero() {
		lastSeen = peer.LastSeen.Format(time.RFC3339)
	}

	tags := peer.Tags
	if tags == nil {
		tags = []string{}
	} else {
		tags = append([]string(nil), tags...)
	}
	tailscaleIPs := append([]string(nil), peer.TailscaleIPs...)
	routedSubnets := routedSubnets(peer)

	return api.Device{
		ID:            id,
		Name:          name,
		IP:            ip,
		TailscaleIPs:  tailscaleIPs,
		OS:            peer.OS,
		Online:        peer.Online,
		Owner:         ownerName(peer.UserID, users),
		Tags:          tags,
		SubnetRouter:  len(routedSubnets) > 0,
		RoutedSubnets: routedSubnets,
		LastSeen:      lastSeen,
	}
}

func routedSubnets(peer Peer) []string {
	routes := peer.PrimaryRoutes
	if len(routes) == 0 {
		routes = peer.AllowedIPs
	}

	subnets := make([]string, 0, len(routes))
	for _, route := range routes {
		if !isTailscaleHostRoute(route, peer.TailscaleIPs) {
			subnets = append(subnets, route)
		}
	}
	sort.Strings(subnets)
	return subnets
}

func isTailscaleHostRoute(route string, tailscaleIPs []string) bool {
	for _, ip := range tailscaleIPs {
		if route == ip || route == ip+"/32" || route == ip+"/128" {
			return true
		}
	}
	return false
}

func ownerName(userID int64, users map[string]User) string {
	user, ok := users[fmt.Sprint(userID)]
	if !ok {
		return ""
	}
	return firstNonEmpty(user.LoginName, user.DisplayName)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
