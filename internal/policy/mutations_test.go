package policy

import (
	"encoding/json"
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

func TestApplyMutationAppendACLPreservesSrcPosture(t *testing.T) {
	out, err := ApplyMutation(samplePolicy, api.PolicyMutation{
		Type: "append-acl",
		Rule: api.ACLDraft{
			Action:     "accept",
			Src:        []string{"group:eng"},
			Dst:        []string{"tag:web:443"},
			SrcPosture: []string{"posture:trusted"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyMutation: %v", err)
	}
	if !strings.Contains(out, `"srcPosture":["posture:trusted"]`) {
		t.Fatalf("expected ACL posture in output: %s", out)
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

func TestApplyMutationAppendGrantPreservesPostureAndVia(t *testing.T) {
	out, err := ApplyMutation(samplePolicy, api.PolicyMutation{
		Type: "append-grant",
		Grant: api.GrantDraft{
			Src:        []string{"group:eng"},
			Dst:        []string{"10.10.0.0/24"},
			IP:         []string{"tcp:443"},
			SrcPosture: []string{"posture:trusted"},
			Via:        []string{"tag:router"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyMutation: %v", err)
	}
	if !strings.Contains(out, `"srcPosture":["posture:trusted"]`) || !strings.Contains(out, `"via":["tag:router"]`) {
		t.Fatalf("expected grant posture and via in output: %s", out)
	}
}

func TestApplyMutationUpsertPosture(t *testing.T) {
	out, err := ApplyMutation(samplePolicy, api.PolicyMutation{
		Type:    "upsert-posture",
		Key:     "posture:trusted",
		Posture: []string{"node:os == 'macos'"},
	})
	if err != nil {
		t.Fatalf("ApplyMutation: %v", err)
	}
	if !strings.Contains(out, `"posture:trusted"`) || !strings.Contains(out, `"node:os == 'macos'"`) {
		t.Fatalf("expected upserted posture in output: %s", out)
	}
}

func TestApplyMutationUpsertPostureCreatesAndUpdatesSection(t *testing.T) {
	created, err := ApplyMutation(`{"acls":[]}`, api.PolicyMutation{
		Type:    "upsert-posture",
		Key:     "posture:trusted",
		Posture: []string{"node:os == 'macos'"},
	})
	if err != nil {
		t.Fatalf("create posture: %v", err)
	}
	if !strings.Contains(created, `"postures"`) || !strings.Contains(created, `"posture:trusted"`) {
		t.Fatalf("expected created posture section: %s", created)
	}

	updated, err := ApplyMutation(`{
		"postures": {
			"posture:trusted": ["node:os == 'macos'"],
			"posture:other": ["node:os == 'linux'"]
		}
	}`, api.PolicyMutation{
		Type:    "upsert-posture",
		Key:     "posture:trusted",
		Posture: []string{"node:tsVersion >= '1.90.0'"},
	})
	if err != nil {
		t.Fatalf("update posture: %v", err)
	}
	var parsed struct {
		Postures map[string][]string `json:"postures"`
	}
	if err := json.Unmarshal([]byte(updated), &parsed); err != nil {
		t.Fatalf("decode updated policy: %v\n%s", err, updated)
	}
	if got := parsed.Postures["posture:trusted"]; len(got) != 1 || got[0] != "node:tsVersion >= '1.90.0'" {
		t.Fatalf("trusted posture = %#v, want updated tsVersion assertion", got)
	}
	if got := parsed.Postures["posture:other"]; len(got) != 1 || got[0] != "node:os == 'linux'" {
		t.Fatalf("other posture = %#v, want preserved sibling", got)
	}
}

func TestApplyMutationUpsertPostureRejectsInvalidInput(t *testing.T) {
	tests := []api.PolicyMutation{
		{Type: "upsert-posture", Key: "", Posture: []string{"node:os == 'macos'"}},
		{Type: "upsert-posture", Key: "trusted", Posture: []string{"node:os == 'macos'"}},
		{Type: "upsert-posture", Key: "posture:trusted", Posture: nil},
	}
	for _, mutation := range tests {
		if _, err := ApplyMutation(samplePolicy, mutation); err == nil {
			t.Fatalf("expected mutation %#v to fail", mutation)
		}
	}
}
