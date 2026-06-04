package mcpserver

import (
	"context"
	"errors"

	"github.com/Waffleophagus/tailor/internal/api"
	"github.com/Waffleophagus/tailor/internal/cloudapi"
	"github.com/Waffleophagus/tailor/internal/mcpserver/policyref"
	"github.com/Waffleophagus/tailor/internal/tailorcore"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type getTailnetStateInput struct{}

type tailnetStateOutput struct {
	Tailnet            string       `json:"tailnet"`
	CloudAuthenticated bool         `json:"cloudAuthenticated"`
	HasPolicy          bool         `json:"hasPolicy"`
	DevMode            bool         `json:"devMode"`
	Devices            []api.Device `json:"devices"`
	Edges              []api.Edge   `json:"edges"`
	DeviceCount        int          `json:"deviceCount"`
	EdgeCount          int          `json:"edgeCount"`
}

type getPolicyInput struct{}

type getPolicyOutput struct {
	Tailnet string `json:"tailnet"`
	HuJSON  string `json:"hujson"`
}

type getPolicyMapInput struct{}

type aclReferenceIndexInput struct{}

type aclReferenceInput struct {
	Topic string `json:"topic" jsonschema:"ACL reference topic ID. Use tailor_acl_reference_index to list valid topics."`
}

type aclReferenceSearchInput struct {
	Query string `json:"query" jsonschema:"Search query for ACL reference topics, for example 'autogroup:self tagged devices ssh'."`
}

type evaluatePolicyDraftInput struct {
	HuJSON      string `json:"hujson" jsonschema:"Policy draft HuJSON to evaluate against current tailnet topology."`
	Perspective string `json:"perspective,omitempty" jsonschema:"Optional source selector perspective for policy evaluation."`
}

type stagePolicyDraftInput struct {
	HuJSON  string `json:"hujson" jsonschema:"Validated policy draft HuJSON to stage for human review in Tailor."`
	Summary string `json:"summary,omitempty" jsonschema:"Short human-readable summary of the intended change."`
}

func registerTools(server *mcp.Server, core *tailorcore.Service, cfg Config) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "tailor_get_tailnet_state",
		Title:       "Get Tailnet State",
		Description: "Return Tailor's current tailnet topology, effective access edges, and Cloud API authentication state.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ getTailnetStateInput) (*mcp.CallToolResult, tailnetStateOutput, error) {
		devices, err := core.TopologyDevices(ctx)
		if err != nil {
			return nil, tailnetStateOutput{}, err
		}
		snapshot := core.TopologySnapshot(ctx, devices)
		status := core.CloudStatus()
		return nil, tailnetStateOutput{
			Tailnet:            snapshot.Tailnet,
			CloudAuthenticated: status.Authenticated,
			HasPolicy:          status.HasPolicy,
			DevMode:            status.DevMode,
			Devices:            snapshot.Devices,
			Edges:              snapshot.Edges,
			DeviceCount:        len(snapshot.Devices),
			EdgeCount:          len(snapshot.Edges),
		}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tailor_get_policy",
		Title:       "Get Policy",
		Description: "Fetch the current ACL policy HuJSON when Cloud API authentication is enabled.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ getPolicyInput) (*mcp.CallToolResult, getPolicyOutput, error) {
		response, err := core.Policy(ctx)
		if err != nil {
			if errors.Is(err, cloudapi.ErrNotAuthenticated) {
				return nil, getPolicyOutput{}, errors.New("Cloud API authentication is required before fetching policy HuJSON.")
			}
			return nil, getPolicyOutput{}, err
		}
		return nil, getPolicyOutput{Tailnet: response.Tailnet, HuJSON: response.HuJSON}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tailor_get_policy_map",
		Title:       "Get Policy Map",
		Description: "Fetch the current ACL policy HuJSON plus structured section inventory, entries, unsupported sections, and raw section values.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ getPolicyMapInput) (*mcp.CallToolResult, api.PolicyMapResponse, error) {
		response, err := core.PolicyMap(ctx)
		if err != nil {
			if errors.Is(err, cloudapi.ErrNotAuthenticated) {
				return nil, api.PolicyMapResponse{}, errors.New("Cloud API authentication is required before fetching the policy map.")
			}
			return nil, api.PolicyMapResponse{}, err
		}
		return nil, response, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tailor_acl_reference_index",
		Title:       "ACL Reference Index",
		Description: "List compact Tailscale ACL reference topics. Read a relevant topic before drafting policy HuJSON changes.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, _ aclReferenceIndexInput) (*mcp.CallToolResult, policyref.Index, error) {
		index, err := policyref.ReferenceIndex()
		return nil, index, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tailor_acl_reference",
		Title:       "ACL Reference Topic",
		Description: "Return one concise Tailscale ACL reference topic by ID. Valid topic IDs are listed by tailor_acl_reference_index.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input aclReferenceInput) (*mcp.CallToolResult, policyref.Topic, error) {
		topic, err := policyref.ReferenceTopic(input.Topic)
		return nil, topic, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tailor_acl_reference_search",
		Title:       "ACL Reference Search",
		Description: "Search ACL reference topic metadata and markdown. Returns up to five topic matches with short snippets.",
	}, func(_ context.Context, _ *mcp.CallToolRequest, input aclReferenceSearchInput) (*mcp.CallToolResult, policyref.SearchResponse, error) {
		response, err := policyref.Search(input.Query)
		return nil, response, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tailor_evaluate_policy_draft",
		Title:       "Evaluate Policy Draft",
		Description: "Compare a HuJSON ACL draft against the current policy and topology without staging or saving it. Use ACL reference topics for syntax questions before evaluating.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input evaluatePolicyDraftInput) (*mcp.CallToolResult, api.PolicyEvaluateDraftResponse, error) {
		output, err := core.EvaluatePolicyDraft(ctx, api.PolicyEvaluateDraftRequest{
			HuJSON:      input.HuJSON,
			Perspective: input.Perspective,
		})
		return nil, output, err
	})

	if cfg.ReadOnly {
		return
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "tailor_stage_policy_draft",
		Title:       "Stage Policy Draft",
		Description: "Validate and evaluate a HuJSON ACL draft, then stage it for explicit human review in the Tailor UI. Evaluate drafts first when possible. This never saves or uploads policy to Tailscale.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input stagePolicyDraftInput) (*mcp.CallToolResult, api.PolicyStageResponse, error) {
		output, err := core.StagePolicyDraft(ctx, api.PolicyStageRequest{
			HuJSON:  input.HuJSON,
			Source:  "mcp",
			Summary: input.Summary,
		})
		return nil, output, err
	})
}
