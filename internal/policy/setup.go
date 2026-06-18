package policy

import (
	"encoding/json"
	"strings"

	"github.com/Waffleophagus/tailor/internal/api"
)

const TailorACLServiceTag = "tag:tailor-acl-service"

// HasTailorAppCapabilityGrant reports whether rawPolicy contains any grant whose
// app block grants admin on the resolved Tailor app capability.
func HasTailorAppCapabilityGrant(rawPolicy, capability string) bool {
	capability = strings.TrimSpace(capability)
	if capability == "" {
		return false
	}
	parsed, err := Parse(rawPolicy)
	if err != nil {
		return false
	}
	for _, grant := range parsed.Grants {
		if grantHasAdminAction(grant.App, capability) {
			return true
		}
	}
	return false
}

func grantHasAdminAction(app map[string]any, capability string) bool {
	if len(app) == 0 {
		return false
	}
	raw, ok := app[capability]
	if !ok {
		return false
	}
	if values, ok := raw.([]any); ok {
		return appValuesHaveAdmin(values)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return false
	}
	var decoded []any
	if err := json.Unmarshal(b, &decoded); err != nil {
		return false
	}
	return appValuesHaveAdmin(decoded)
}

func appValuesHaveAdmin(values []any) bool {
	for _, item := range values {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		actionsRaw, ok := obj["actions"]
		if !ok {
			continue
		}
		switch actions := actionsRaw.(type) {
		case []any:
			for _, action := range actions {
				if s, ok := action.(string); ok && s == "admin" {
					return true
				}
			}
		case []string:
			for _, action := range actions {
				if action == "admin" {
					return true
				}
			}
		}
	}
	return false
}

// RecommendedSetupGrant returns the first-run owner/admin grant for the resolved capability.
func RecommendedSetupGrant(capability string) api.GrantDraft {
	capability = strings.TrimSpace(capability)
	return api.GrantDraft{
		Src: []string{"autogroup:owner", "autogroup:admin"},
		Dst: []string{TailorACLServiceTag},
		IP:  []string{"tcp:443"},
		App: map[string]any{
			capability: []map[string]any{
				{"actions": []string{"admin"}},
			},
		},
	}
}

// RecommendedSetupPolicyExtras returns tagOwners entries suggested alongside the setup grant.
func RecommendedSetupPolicyExtras() map[string][]string {
	return map[string][]string{
		TailorACLServiceTag: {"autogroup:admin"},
	}
}

// FormatGrantSnippet renders a grant draft as indented JSON for display.
func FormatGrantSnippet(grant api.GrantDraft) string {
	payload := map[string]any{
		"tagOwners": RecommendedSetupPolicyExtras(),
		"grants":    []api.GrantDraft{grant},
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}
