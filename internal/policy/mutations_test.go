package policy

import (
	"strings"
	"testing"

	"github.com/Waffleophagus/tailor/internal/api"
)

const samplePolicy = `{
	"acls": [
		{"action": "accept", "src": ["group:eng"], "dst": ["tag:server:22"]}
	],
	"groups": {
		"group:eng": ["alice@example.com"]
	},
	"tagOwners": {
		"tag:server": ["group:eng"]
	}
}`

func TestApplyMutationAppendACL(t *testing.T) {
	out, err := ApplyMutation(samplePolicy, api.PolicyMutation{
		Type: "append-acl",
		Rule: api.ACLDraft{
			Action: "accept",
			Src:    []string{"group:eng"},
			Dst:    []string{"tag:web:443"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyMutation: %v", err)
	}
	if !strings.Contains(out, `"tag:web:443"`) {
		t.Fatalf("expected appended ACL destination in output: %s", out)
	}
}

func TestApplyMutationUpsertGroup(t *testing.T) {
	out, err := ApplyMutation(samplePolicy, api.PolicyMutation{
		Type:    "upsert-group",
		Key:     "group:ops",
		Members: []string{"bob@example.com"},
	})
	if err != nil {
		t.Fatalf("ApplyMutation: %v", err)
	}
	if !strings.Contains(out, `"group:ops"`) || !strings.Contains(out, `"bob@example.com"`) {
		t.Fatalf("expected upserted group in output: %s", out)
	}
}

func TestApplyMutationRemoveACL(t *testing.T) {
	out, err := ApplyMutation(samplePolicy, api.PolicyMutation{
		Type:  "remove-acl",
		Index: 0,
	})
	if err != nil {
		t.Fatalf("ApplyMutation: %v", err)
	}
	if strings.Contains(out, `"tag:server:22"`) {
		t.Fatalf("expected ACL removed from output: %s", out)
	}
}
