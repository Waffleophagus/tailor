//go:build dev

package tailorcore

import (
	"context"
	"errors"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
)

func TestSaveStagedPolicyRejectsStaleBasePolicyInDevMode(t *testing.T) {
	ctx := context.Background()
	service := New(Options{})
	defer service.Close()

	if _, err := service.AuthenticateCloud(ctx, cloudapi.AuthRequest{
		Tailnet: "-",
		APIKey:  "tskey-api-tailor-dev",
	}); err != nil {
		t.Fatal(err)
	}

	current, err := service.Policy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	staged, err := service.StagePolicyDraft(ctx, api.PolicyStageRequest{
		HuJSON: current.HuJSON,
		Source: "ui",
	})
	if err != nil {
		t.Fatal(err)
	}

	newLivePolicy := `{"acls":[{"action":"accept","src":["autogroup:member"],"dst":["autogroup:self:*"]}]}`
	if _, err := service.cloudAPI.SavePolicy(ctx, newLivePolicy); err != nil {
		t.Fatal(err)
	}

	_, err = service.SaveStagedPolicy(ctx, api.PolicySaveRequest{
		DraftID:   staged.Draft.ID,
		DraftHash: staged.Draft.DraftHash,
	})
	if !errors.Is(err, ErrStagedDraftBaseMismatch) {
		t.Fatalf("SaveStagedPolicy error = %v, want %v", err, ErrStagedDraftBaseMismatch)
	}
}
