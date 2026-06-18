package policy

import (
	"fmt"

	"github.com/Waffleophagus/tailor/internal/api"
)

// AppendSetupGrant appends the recommended setup grant and ensures tagOwners for the service tag.
func AppendSetupGrant(raw string, grant api.GrantDraft) (string, error) {
	if HasTailorAppCapabilityGrant(raw, capabilityFromGrant(grant)) {
		return "", fmt.Errorf("tailor app capability grant already exists")
	}
	updated := raw
	var err error
	for tag, owners := range RecommendedSetupPolicyExtras() {
		updated, err = upsertObjectEntry(updated, "tagOwners", tag, owners)
		if err != nil {
			return "", err
		}
	}
	return appendGrantRule(updated, grant)
}

func capabilityFromGrant(grant api.GrantDraft) string {
	for key := range grant.App {
		return key
	}
	return ""
}
