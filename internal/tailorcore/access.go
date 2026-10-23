package tailorcore

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/Waffleophagus/tailor/internal/cloudapi"
)

// policyAdminRoles are the Tailscale user roles allowed to edit the tailnet policy
// file. Tailor mirrors that: these users may edit through Tailor without a separate
// app capability grant.
var policyAdminRoles = map[string]bool{
	"owner":         true,
	"admin":         true,
	"network-admin": true,
}

var (
	// ErrCredentialsOtherTailnet means the API key cannot see this Tailor node, so it
	// belongs to some other tailnet.
	ErrCredentialsOtherTailnet = errors.New("this API key belongs to a different tailnet than the one Tailor is running in")
	// ErrCredentialsNotAdmin means the person submitting the key is not an Owner,
	// Admin or Network admin of the key's tailnet.
	ErrCredentialsNotAdmin = errors.New("only tailnet Owners, Admins and Network admins can connect Tailor to the Tailscale API")
)

// CredentialCheck identifies this Tailor node and, optionally, the person whose
// tailnet role must be verified before Cloud API credentials are accepted.
type CredentialCheck struct {
	SelfNodeID string
	SelfIPs    []string
	// AdminLogin, when set, must be an Owner, Admin or Network admin according to
	// the submitted key's user list.
	AdminLogin string
}

// VerifyCloudCredentials vets an API key before it is stored: the key must see
// this Tailor node (so it is for Tailor's own tailnet), and check.AdminLogin, if
// set, must hold a policy-admin role there.
func (s *Service) VerifyCloudCredentials(ctx context.Context, request cloudapi.AuthRequest, check CredentialCheck) error {
	info, err := s.cloudAPI.InspectCredentials(ctx, request)
	if err != nil {
		return err
	}
	if !containsNode(info.Devices, check.SelfNodeID, check.SelfIPs) {
		return ErrCredentialsOtherTailnet
	}
	if check.AdminLogin != "" && !isPolicyAdmin(info.Users, check.AdminLogin) {
		return ErrCredentialsNotAdmin
	}
	return nil
}

// IsTailnetPolicyAdmin reports whether login is an Owner, Admin or Network admin
// of the authenticated tailnet. It fails closed when Cloud API data is unavailable.
func (s *Service) IsTailnetPolicyAdmin(ctx context.Context, login string) bool {
	status := s.cloudAPI.Status()
	if !status.Authenticated || status.DevMode || strings.TrimSpace(login) == "" {
		return false
	}
	users, err := s.cloudMeta.Users(ctx)
	if err != nil {
		s.logger.Warn("tailnet admin lookup failed; treating caller as view-only", "error", err.Error())
		return false
	}
	return isPolicyAdmin(users, login)
}

func isPolicyAdmin(users []cloudapi.User, login string) bool {
	login = strings.TrimSpace(login)
	if login == "" {
		return false
	}
	for _, user := range users {
		if !strings.EqualFold(strings.TrimSpace(user.LoginName), login) {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(user.Status)) {
		case "", "active", "idle":
		default: // suspended, needs-approval, over-billing-limit, ...
			return false
		}
		return policyAdminRoles[strings.ToLower(strings.TrimSpace(user.Role))]
	}
	return false
}

func containsNode(devices []cloudapi.Device, nodeID string, ips []string) bool {
	for _, device := range devices {
		if nodeID != "" && device.NodeID == nodeID {
			return true
		}
		for _, address := range device.Addresses {
			if address != "" && slices.Contains(ips, address) {
				return true
			}
		}
	}
	return false
}
