package localapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Waffleophagus/tailor/internal/api"
	"tailscale.com/client/local"
	"tailscale.com/ipn/ipnstate"
	"tailscale.com/tailcfg"
	"tailscale.com/types/views"
)

const PlatformDefaultEndpoint = "platform default"

var ErrUnavailable = errors.New("tailscale LocalAPI unavailable")

type Client struct {
	socketOverride string
	localClient    *local.Client
	logger         *slog.Logger
}

type ClientOption func(*Client)

func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		if logger != nil {
			c.logger = logger
		}
	}
}

func New(socketPath string, options ...ClientOption) *Client {
	localClient := &local.Client{Socket: socketPath}
	c := &Client{
		socketOverride: socketPath,
		localClient:    localClient,
		logger:         slog.New(slog.DiscardHandler),
	}
	for _, option := range options {
		option(c)
	}
	return c
}

func (c *Client) Endpoint() string {
	if c.socketOverride != "" {
		return c.socketOverride
	}
	return defaultEndpointDescription()
}

func (c *Client) Status(ctx context.Context) ([]api.Device, error) {
	status, err := c.localClient.Status(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}

	return DevicesFromIPNStatus(status), nil
}

// StatusLogged is like Status but logs LocalAPI unavailability for HTTP handlers.
func (c *Client) StatusLogged(ctx context.Context, operation string) ([]api.Device, error) {
	devices, err := c.Status(ctx)
	if err != nil && errors.Is(err, ErrUnavailable) {
		c.logger.Warn("localapi unavailable",
			"operation", operation,
			"endpoint", c.Endpoint(),
			"error", err.Error(),
		)
	}
	return devices, err
}

func (c *Client) TailnetName(ctx context.Context) (string, error) {
	status, err := c.localClient.StatusWithoutPeers(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	// Prefer the DNS suffix (e.g. "triceratops-gecko.ts.net") — this is what
	// Tailscale's Cloud API expects for /api/v2/tailnet/{name}/acl calls.
	if status.CurrentTailnet != nil && status.CurrentTailnet.MagicDNSSuffix != "" {
		return status.CurrentTailnet.MagicDNSSuffix, nil
	}
	if status.MagicDNSSuffix != "" {
		return status.MagicDNSSuffix, nil
	}
	if status.CurrentTailnet != nil && status.CurrentTailnet.Name != "" {
		return status.CurrentTailnet.Name, nil
	}
	return "", nil
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

func DevicesFromIPNStatus(status *ipnstate.Status) []api.Device {
	if status == nil {
		return nil
	}

	devices := make([]api.Device, 0, len(status.Peer)+1)
	if status.Self != nil {
		devices = append(devices, deviceFromPeerStatus(status.Self, status.User))
	}

	keys := make([]string, 0, len(status.Peer))
	peers := make(map[string]*ipnstate.PeerStatus, len(status.Peer))
	for key, peer := range status.Peer {
		keyString := key.String()
		keys = append(keys, keyString)
		peers[keyString] = peer
	}
	sort.Strings(keys)

	for _, key := range keys {
		devices = append(devices, deviceFromPeerStatus(peers[key], status.User))
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

func deviceFromPeerStatus(peer *ipnstate.PeerStatus, users map[tailcfg.UserID]tailcfg.UserProfile) api.Device {
	if peer == nil {
		return api.Device{Tags: []string{}, RoutedSubnets: []string{}}
	}

	tailscaleIPs := addrsToStrings(peer.TailscaleIPs)
	id := firstNonEmpty(string(peer.ID), peer.PublicKey.String(), peer.DNSName, peer.HostName)
	name := strings.TrimSuffix(firstNonEmpty(peer.DNSName, peer.HostName, id), ".")
	ip := ""
	if len(tailscaleIPs) > 0 {
		ip = tailscaleIPs[0]
	}

	lastSeen := ""
	if !peer.LastSeen.IsZero() {
		lastSeen = peer.LastSeen.Format(time.RFC3339)
	}

	routedSubnets := routedSubnetsFromPeerStatus(peer, tailscaleIPs)

	return api.Device{
		ID:            id,
		Name:          name,
		IP:            ip,
		TailscaleIPs:  tailscaleIPs,
		OS:            peer.OS,
		Online:        peer.Online,
		Owner:         ownerNameFromUserProfiles(peer.UserID, users),
		Tags:          viewStrings(peer.Tags),
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

func routedSubnetsFromPeerStatus(peer *ipnstate.PeerStatus, tailscaleIPs []string) []string {
	routes := prefixesToStrings(peer.PrimaryRoutes)
	if len(routes) == 0 {
		routes = prefixesToStrings(peer.AllowedIPs)
	}

	subnets := make([]string, 0, len(routes))
	for _, route := range routes {
		if !isTailscaleHostRoute(route, tailscaleIPs) {
			subnets = append(subnets, route)
		}
	}
	sort.Strings(subnets)
	return subnets
}

func addrsToStrings(addrs []netip.Addr) []string {
	values := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		values = append(values, addr.String())
	}
	return values
}

func prefixesToStrings(prefixes *views.Slice[netip.Prefix]) []string {
	if prefixes == nil {
		return nil
	}
	values := make([]string, 0, prefixes.Len())
	for i := 0; i < prefixes.Len(); i++ {
		values = append(values, prefixes.At(i).String())
	}
	return values
}

func viewStrings(values *views.Slice[string]) []string {
	if values == nil {
		return []string{}
	}
	strings := make([]string, 0, values.Len())
	for i := 0; i < values.Len(); i++ {
		strings = append(strings, values.At(i))
	}
	return strings
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

func ownerNameFromUserProfiles(userID tailcfg.UserID, users map[tailcfg.UserID]tailcfg.UserProfile) string {
	user, ok := users[userID]
	if !ok {
		return ""
	}
	return firstNonEmpty(user.LoginName, user.DisplayName)
}

func defaultEndpointDescription() string {
	switch runtime.GOOS {
	case "linux":
		return "default Linux tailscaled socket"
	case "darwin":
		return "default macOS Tailscale LocalAPI endpoint"
	case "windows":
		return "default Windows tailscaled named pipe"
	default:
		return PlatformDefaultEndpoint
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
