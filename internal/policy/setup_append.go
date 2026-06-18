package policy

import (
	"fmt"

	"github.com/Waffleophagus/tailor/internal/api"
)

// AppendSetupGrant appends the recommended setup grant and ensures tagOwners for the service tag.
func AppendSetupGrant(raw string, grant api.GrantDraft) (string, error) {
	capability, err := capabilityFromGrant(grant)
	if err != nil {
		return "", err
	}
	if HasTailorAppCapabilityGrant(raw, capability) {
		return "", fmt.Errorf("tailor app capability grant already exists")
	}
	updated := raw
	for tag, owners := range RecommendedSetupPolicyExtras() {
		var upsertErr error
		updated, upsertErr = upsertObjectEntry(updated, "tagOwners", tag, owners)
		if upsertErr != nil {
			return "", upsertErr
		}
	}
	return appendGrantRule(updated, grant)
}

func capabilityFromGrant(grant api.GrantDraft) (string, error) {
	if len(grant.App) != 1 {
		return "", fmt.Errorf("setup grant must include exactly one app capability")
	}
	for key := range grant.App {
		return key, nil
	}
	return "", fmt.Errorf("setup grant must include exactly one app capability")
}
