# tsnet Access Control Plan

## Goal

Use tsnet connection identity to make Tailor view-only by default, while allowing trusted tailnet principals to use ACL policy features and MCP write features.

The first policy model is intentionally simple:

- `autogroup:owner` and `autogroup:admin` get full Tailor access.
- Connections from devices tagged `tag:tailor-acl-editor` can get full Tailor access when the policy explicitly grants the Tailor app capability to that tag.
- Everyone else is view-only.

This replaces bearer-token-only MCP authorization for tailnet exposure and gives the UI, HTTP API, and MCP server the same authorization model.

The first-run experience should still be fast, but the happy path is that the user adds Tailor's suggested app capability grant and lets tailnet policy govern Tailor access:

1. Tailor starts view-only.
2. A tailnet admin enters a Cloud API key with policy read/write access.
3. Tailor stores the Cloud API key in backend memory for the lifetime of the process.
4. Tailor fetches the policy file and checks whether a grant for the resolved Tailor app capability already exists in some form.
5. If a Tailor app capability grant exists, Tailor enters normal mode: the Cloud API key stays in backend memory, and policy features are authorized per request by `WhoIs`.
6. If no Tailor app capability grant exists, Tailor asks whether to apply the Tailor app capability grant shown below. The user can accept, edit, or cancel the suggested policy change.
7. If the user accepts or edits the grant, Tailor appends the grant, validates the full policy through the Cloud API, saves it if validation succeeds, then waits for `WhoIs` to show the capability before unlocking policy editing.
8. If validation or save fails, Tailor falls back to a short browser-only bootstrap session, shows the grant snippet, and explains that fixing the grant in tailnet policy and restarting Tailor will make Tailor work without the temporary session limit.

## Access Model

### Full access

A request has full access when the tsnet `WhoIs` result proves either:

1. The caller is in `autogroup:owner` or `autogroup:admin`, through a Tailor app capability granted by the tailnet policy.
2. The caller's source device is tagged `tag:tailor-acl-editor`, through the same Tailor app capability.

Full access means:

- View topology and status.
- View current ACL policy and structured policy map when Cloud API authentication is configured.
- Draft, mutate, evaluate, validate, stage, and save policy changes through the HTTP API.
- Use all enabled MCP tools, including write tools. Today that means staging drafts; if Tailor later adds an MCP save tool, it must use this same full-access gate.

Cloud API authentication is still required for policy read/write operations. tsnet identity authorizes the caller to use Tailor's policy features; it does not itself provide a Tailscale Cloud API token.

When the Tailor app capability grant is present, policy feature authorization is backend-based and derived from each request's tsnet identity. When the grant is not present, the Cloud API key can only be used for the narrow setup action that appends the Tailor app capability grant. If that setup action fails validation or save, Tailor may issue a short browser-only bootstrap session as an explicit fallback. This fallback is temporary and should not be treated as the normal operating mode.

The per-request `WhoIs` capability map is the authorization source of truth. Inspecting the policy file can help Tailor decide whether to show the setup recommendation, but it must not authorize a caller because a grant can exist without applying to the request's source device.

### View-only access

Every other tailnet caller is view-only.

View-only means:

- View the Tailor UI.
- View topology/status data that Tailor itself can see from its netmap.
- Use read-only MCP tools that do not expose policy HuJSON, such as topology state and embedded ACL reference docs.

View-only callers cannot:

- Fetch raw policy HuJSON.
- Fetch structured policy map entries.
- Draft, mutate, evaluate, validate, stage, or save policy changes.
- Use MCP write tools.

The only exception is an active Tailor Bootstrap Session after automatic setup grant validation or save fails. That fallback is browser-only, lasts 15 minutes, and temporarily allows HTTP ACL editing so the user can repair the policy. It is not a normal full Tailor Access Role and never applies to MCP.

## Tailscale Policy Contract

Tailor should use an app capability because it lets the tailnet policy carry the product authorization decision.

The capability name should be resolved in this order:

1. `TAILOR_APP_CAPABILITY`, when explicitly set.
2. The Tailor tsnet node's MagicDNS name plus `/cap/admin`.

For example, if Tailor is reachable at:

```text
tailor.triceratops-gecko.ts.net
```

then the default app capability should be:

```text
tailor.triceratops-gecko.ts.net/cap/admin
```

This makes the capability name self-hosted, tailnet-local, and tied to the Tailor instance the user is actually configuring. If Tailor is not running in tsnet mode, or the MagicDNS name is not available yet, Tailor should ask for `TAILOR_APP_CAPABILITY` before generating the policy snippet.

The `/cap/admin` suffix is not a web route. It is part of the capability identifier. Tailscale app capabilities follow a `{domain}/{path}` naming shape, such as:

```text
<domain-or-magicdns-name>/cap/<feature>
```

Tailor uses `/cap/admin` to name the coarse first-pass permission: full administration of Tailor's ACL editing surface. Tailscale reserves `tailscale.com` and `tailscale.io`; Tailor should not use those unless the project becomes an official Tailscale product.

Capability values should start with a small action list:

```json
{ "actions": ["admin"] }
```

`admin` is deliberately coarse for the first pass. It maps to full access. If later needed, it can expand into `read-policy`, `draft`, `stage`, and `save`.

Example policy snippet:

```json
{
  "tagOwners": {
    "tag:tailor-acl-service": ["autogroup:admin"]
  },
  "grants": [
    {
      "src": ["autogroup:owner", "autogroup:admin"],
      "dst": ["tag:tailor-acl-service"],
      "ip": ["tcp:443"],
      "app": {
        "tailor.triceratops-gecko.ts.net/cap/admin": [
          { "actions": ["admin"] }
        ]
      }
    },
    {
      "src": ["autogroup:member"],
      "dst": ["tag:tailor-acl-service"],
      "ip": ["tcp:443"]
    }
  ]
}
```

Notes:

- `tag:tailor-acl-service` is the destination tag on the Tailor tsnet node for the ACL editing service surface.
- Do not use a generic `tag:tailor` in the first-run snippet; reserve that shape for future Tailor behaviors such as visibility/discovery if needed.
- `tag:tailor-acl-editor` is an optional source-device override for trusted workstations or MCP runners. It should be documented as an advanced pattern, not included in the first-run generated snippet.
- `tag:tailor-acl-editor` does not mean "this user is an ACL editor"; it means "requests from this tagged device are allowed to administer Tailor."
- `tailor.triceratops-gecko.ts.net/cap/admin` is an example generated from the Tailor node's MagicDNS name; each install should use its own resolved capability name.
- The view-only grant is optional if Tailor should not be reachable by all members.

## First-Run Recommendation UI

When a Cloud API key is accepted and Tailor does not find an existing grant for the resolved Tailor app capability, show an editable recommendation before enabling policy editing.

Suggested copy:

```text
Tailor should add an app capability grant so ACL editing access is controlled by your tailnet policy.
```

Actions:

- `Add recommended grant`: append the generated owner/admin grant, validate the policy, and save it through the Cloud API.
- `Edit grant`: open the generated grant in an editable HuJSON view, then append it, validate the policy, and save it through the Cloud API.
- `Not now`: leave the caller view-only and do not unlock policy editing.

The edited grant can use stricter or broader source selectors, such as only `autogroup:owner`, only `autogroup:admin`, or an existing group. Tailor's setup detection should only require that the policy contains a grant with the resolved Tailor app capability and `actions` containing `admin`; it should not require the generated source selectors to remain unchanged.

Do not persist dismissal. If the user chooses `Not now`, they remain view-only. If they enter the Cloud API key again and the Tailor app capability grant still does not exist, show the recommendation again.

The setup write is intentionally a direct Cloud API validate-and-save, not a staged draft. It is a narrow setup action whose only job is to move Tailor into the normal grant-governed authorization model. Use the existing HuJSON round-trip machinery so comments and formatting are preserved as much as possible when appending the grant.

If setup validation or save fails, show an explicit fallback message:

```text
Tailor could not apply the app capability grant automatically. ACL editing is temporarily available in this browser session. Add the grant below to your tailnet policy and restart Tailor to use grant-based access without a time limit.
```

Include the generated or edited grant snippet in the message. The fallback bootstrap session should be browser-only, last 15 minutes, and never apply to MCP.

## Runtime Design

### Request identity

In tsnet mode, Tailor should attach a request identity before routing API or MCP handlers:

```go
type TailnetIdentity struct {
    LoginName string
    NodeName  string
    NodeTags  []string
    CapMap    tailcfg.PeerCapMap
}
```

The middleware should:

1. Parse `r.RemoteAddr`.
2. Call `local.Client.WhoIs(ctx, r.RemoteAddr)`.
3. Store the identity in request context.
4. Classify the request as `full` or `viewer`.

If `WhoIs` fails in tsnet tailnet mode, reject with `403`. If Tailor is running in local host-socket mode, fall back to today's local behavior until the local authorization model is designed.

### Authorization checks

Centralize authorization in one package rather than scattering checks through handlers:

```go
type Permission string

const (
    PermissionViewTopology Permission = "view-topology"
    PermissionReadPolicy   Permission = "read-policy"
    PermissionWritePolicy  Permission = "write-policy"
    PermissionUseMCPWrite  Permission = "use-mcp-write"
)
```

Policy:

- `view-topology`: allowed for everyone who can connect.
- `read-policy`: full access only.
- `write-policy`: full access only.
- `use-mcp-write`: full access only.

This should gate both HTTP handlers and MCP tool registration/execution.
For HTTP handlers only, an active Tailor Bootstrap Session may satisfy `read-policy` and `write-policy` during its 15-minute fallback window. It must never satisfy `use-mcp-write`.

The authorization layer must keep two separate facts:

- Cloud API authentication: whether Tailor has a usable Cloud API credential for policy operations.
- Caller permission: whether this request has full access through the Tailor app capability.

Process-wide Cloud API authentication must not by itself grant every connected tailnet caller policy access.
When both a Cloud API credential and a tailnet policy file are available, Tailor should inspect the policy to detect whether the Tailor app capability grant is already configured in some form. That detection controls setup UX only: if the grant exists, do not show the suggested-grant setup prompt. Request authorization still comes from `WhoIs`.

After a caller submits a valid Cloud API key, Tailor should fetch the current policy and look for any grant whose `app` block includes the resolved Tailor app capability with `actions` containing `admin`. The grant does not need to exactly match Tailor's suggested snippet; stricter or broader source selectors are valid. If such a grant exists, Tailor enters normal mode: the Cloud API key remains in backend memory, and policy features are authorized per request by `WhoIs`. A caller who supplied the key but does not receive the capability in `WhoIs` remains view-only.

In that state, the UI should say: "API key accepted, but your current device or user is view-only." Do not show the suggested-grant setup prompt if a Tailor app capability grant already exists in the policy.

### Setup grant save

The setup grant save is the only policy write allowed before the current caller has the Tailor app capability in `WhoIs`. It should:

- Require a freshly accepted Cloud API key.
- Fetch the current policy.
- Verify that the resolved Tailor app capability grant is still absent.
- Append the generated or edited grant.
- Validate through the Cloud API.
- Save through the Cloud API only if validation succeeds.
- Poll or re-check `WhoIs` until the capability is effective for the caller, then unlock normal policy editing if the caller receives `actions: ["admin"]`.

If the save succeeds but `WhoIs` does not grant the capability to the current caller, keep the caller view-only and show: "Tailor access was configured, but your current device or user is view-only." A browser refresh should trigger a new `WhoIs` check; the UI can mention refresh/retry in case Tailscale policy propagation is still catching up.

Do not create a bootstrap fallback after a successful setup grant save. If the saved grant does not appear in the current caller's `WhoIs`, presume the caller is intentionally outside the grant or propagation has not completed yet. The caller can refresh or retry, but Tailor should not bypass the saved grant.

If validation or save fails, create a 15-minute browser-only bootstrap session for the current caller and show the fallback message with the grant snippet. The bootstrap session should be tied to the current browser and current `WhoIs` node/user, and it must not authorize MCP.

## Implementation Plan

1. Keep tailnet runtime wiring at the process boundary.
   - `cmd/tailor` selects host LocalAPI or embedded tsnet mode and owns the
     `tsnet.Server`, listeners, and lifecycle.
   - In embedded mode, `cmd/tailor` obtains tsnet's `LocalClient` and passes it
     to the existing LocalAPI adapter used by `tailorcore.Service`.
   - The HTTP server receives the narrower `WhoIs` and tailnet-status
     dependencies needed for request identity and capability resolution.
   - Do not introduce one shared runtime interface spanning LocalAPI access,
     request identity, listeners, and lifecycle. Those responsibilities have
     different consumers, and keeping them separate makes the boundaries more
     explicit. Add narrower interfaces only when a consumer or test requires
     one.

2. Serve Docker deployments directly through tsnet.
   - Use `TAILSCALE_AUTHKEY` or `TS_AUTHKEY`.
   - Advertise `tag:tailor-acl-service` by default when configured.
   - Replace Tailscale Serve with a direct tsnet TLS listener.
   - Keep `:8080` for local container access when configured.

3. Add request identity middleware.
   - No identity requirement for static frontend assets.
   - API and MCP routes get identity when the listener is tsnet-backed.
   - Log user/node metadata without logging capability payloads.

4. Add authorization gates.
   - Gate `/api/policy`, `/api/policy/map`, draft, mutate, evaluate, validate, stage, staged draft reads, discard, and save.
   - Gate MCP write tools.
   - Keep topology/status read-only.

5. Add the setup grant recommendation flow.
   - When a caller successfully adds a Cloud API key and no Tailor app capability is detected, show a suggested grant snippet.
   - The first-run snippet should include `autogroup:owner` and `autogroup:admin`.
   - Let the user accept the generated grant, edit it before saving, or cancel.
   - Document `tag:tailor-acl-editor` separately as an advanced source-device grant for trusted workstations and MCP runners.
   - The snippet should use the resolved Tailor app capability name.
   - Validate and save this setup grant directly through the Cloud API rather than routing it through Tailor's staged commit flow.
   - If validation or save fails, create a 15-minute browser-only bootstrap session and show the grant snippet with instructions to add it manually and restart Tailor.

6. Update UI behavior.
   - Add an auth status field that distinguishes Cloud API authentication from caller permission.
   - Hide or disable ACL editing controls for view-only callers.
   - Show an explicit "view-only" state instead of prompting every caller for an API key.
   - If a Cloud API key is accepted but the current caller lacks the Tailor app capability, show: "API key accepted, but your current device or user is view-only."
   - If the setup grant save succeeds but the current caller still lacks the Tailor app capability, show: "Tailor access was configured, but your current device or user is view-only."
   - If setup validation or save fails and Tailor falls back to a bootstrap session, show: "Tailor could not apply the app capability grant automatically. ACL editing is temporarily available in this browser session. Add the grant below to your tailnet policy and restart Tailor to use grant-based access without a time limit."

7. Update MCP behavior.
   - In tailnet mode, prefer tsnet identity and app capability authorization.
   - Do not accept bootstrap fallback sessions for MCP.
   - Keep bearer tokens for public exposure.
   - Keep localhost mode loopback-only.

8. Update docs and examples.
   - Document `tag:tailor-acl-service` and `tag:tailor-acl-editor`.
   - Document the app capability snippet.
   - Keep the first-run generated snippet owner/admin focused; show `tag:tailor-acl-editor` primarily in MCP and automation documentation.
   - Update Docker docs away from Tailscale Serve once tsnet serving is implemented.

## Open Questions

1. Should `autogroup:network-admin` also get full access, or should the first pass remain strictly `autogroup:owner`, `autogroup:admin`, and optional `tag:tailor-acl-editor` grants?
2. Should the UI allow a full-access caller to paste a Cloud API key for the process, or should Docker/tsnet deployments prefer a server-side environment variable?
3. Should MCP ever get a direct save tool, or should it remain stage-only even for full-access callers?
4. In host LocalAPI mode, should Tailor remain trust-localhost, or should we add a separate local auth mechanism?

## Non-Goals

- Replacing Cloud API authentication with Tailscale identity.
- Inferring Tailscale admin role without an app capability.
- Supporting fine-grained per-section policy permissions in the first pass.
- Making public exposure safe without bearer-token or external auth.
