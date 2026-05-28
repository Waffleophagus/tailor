# Tailnet Topology Visualizer & ACL Editor

## Canonical Names

| Term | Definition |
|------|------------|
| **Device** | A specific machine (physical or virtual) connected to a Tailscale tailnet. Has a name, Tailscale IP, OS, online status, tags, and an owning user. The primary node type in the graph. |
| **User** | A human identity (e.g., `alice@company.com`) who owns one or more devices. Users are not graph nodes but are metadata attached to devices. |
| **Tag** | A label applied to devices in Tailscale (e.g., `tag:server`, `tag:database`). Tags are the primary targets in ACL rules and appear in the graph as device metadata and color-coding. |
| **Group** | A named collection of users defined in the tailnet policy file (e.g., `group:eng`). Groups are policy abstractions, not graph nodes. |
| **Autogroup** | Built-in Tailscale groups (`autogroup:member`, `autogroup:admin`, `autogroup:self`). Used in ACL rules as sources or destinations. |
| **Policy Subject** | A user, group, tag, autogroup, host, or IP range referenced by policy rules. A device is matched by policy subjects through its owner, tags, IPs, and other policy-defined selectors; it is not directly a member of a group. |
| **ACL Rule (ACE)** | An access control entry in the tailnet policy file: `{ "action": "accept", "src": [...], "dst": [...], "proto": ... }`. Defines which sources can reach which destinations on which ports. |
| **Grant** | A newer Tailscale access control primitive (replaces some ACL use cases) that provides application-layer permissions. Distinguished from ACLs in the policy file. |
| **HuJSON** | Human JSON — Tailscale's dialect of JSON that permits C-style comments (`//` and `/* */`) and trailing commas. The wire format for the tailnet policy file. |
| **Tailnet** | A private network managed by Tailscale, consisting of devices, users, ACLs, and DNS settings. |
| **LocalAPI** | The Unix-socket (Linux/macOS) or TCP-port (Windows) API exposed by the local `tailscaled` daemon. Requires no authentication. Provides device status, peer lists, and local identity. |
| **Cloud API** | The HTTPS REST API at `api.tailscale.com`. Requires an OAuth Client or API Access Token. Provides ACL policy files, device management, DNS settings, etc. |
| **Tailscale Status** | The output of `tailscale status --json`: a list of all devices in the tailnet with names, IPs, tags, online status, and owner information. |
| **Policy File** | The HuJSON file containing `acls`, `grants`, `groups`, `tagOwners`, `ssh`, and other tailnet configuration. Lives in the Tailscale admin console and is accessible via Cloud API. |
| **Effective Access** | The resolved, concrete reachability between two devices after evaluating all ACL rules, groups, tags, autogroups, destination ports, and protocols. Not the same as visibility (all devices are visible to all members). |
| **Access Scope** | The port/protocol subset allowed by an effective access path, such as HTTPS-only (`tcp:443`) versus SSH (`tcp:22`) or broader/custom access. Used by graph styling to distinguish full, partial, and protocol-specific reachability. |
| **Phase 1** | Read-only topology discovery mode. Uses LocalAPI only. Renders all devices with inferred relationships (owner clusters, shared tags). No ACL resolution. |
| **Phase 2** | Authenticated ACL editing mode. Uses Cloud API with a `tskey-api-...` API key. Resolves effective access edges and allows policy editing with staged commit. |
| **Policy Workbench** | The Tailscale-shaped policy editing surface: POLICY nav (general access rules, SSH, tests, auto-approvers) and DEFINITIONS nav (groups, tags, IP sets, hosts, device posture, node attributes). Primary place to configure ACLs; graph is the preview. See `.scratch/tailnet-topology-visualizer/018-policy-scenario-roadmap.md`. |
| **Policy Scenario** | A persistent simulation setup: who initiates (`sourceSelector`), policy mode (`current` / `draft` / `diff`), and graph mode (`focused` / `all`). The scenario bar above the graph answers "view as whom?" while the workbench answers "edit what?" |
| **Perspective** | The simulated policy subject (`sourceSelector`) within a Policy Scenario — a user, group, tag, autogroup, or composite cohort. Backend filters evaluation edges by this subject; the graph highlights matching real source devices (no synthetic center node). |
| **Simulation tier** | How faithfully a policy section affects the graph preview: graph-simulated (ACLs, grants, selector definitions), graph-partial (SSH vs network path), edit+validate only (posture, node attrs until device data available), or non-graph (tests, auto-approvers). |
| **Node (graph)** | A rendered element in the Cytoscape.js graph. Currently always represents a Device. |
| **Edge (graph)** | A rendered connection in the Cytoscape.js graph. In Phase 1: inferred relationship (shared owner or shared tag). In Phase 2: effective access path (allowed by ACL rules). |
| **Policy Lens** | The graph-adjacent panel where selecting a device or edge reveals provenance and safe edit actions. Jumps to the Policy Workbench section that owns the rule or definition; does not edit graph nodes directly. |
| **Staged Commit** | The editing model where workbench and lens mutations batch into a draft tray, review as HuJSON diff, validate against the Cloud API, and save only after validation. |
