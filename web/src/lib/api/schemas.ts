import { z } from "zod";

export const healthResponseSchema = z.object({
  status: z.string(),
  version: z.string(),
});

export const localApiStatusResponseSchema = z.object({
  available: z.boolean(),
  localApiEndpoint: z.string(),
  error: z.string().optional(),
});

export const deviceSchema = z.object({
  id: z.string(),
  name: z.string(),
  ip: z.string(),
  tailscaleIps: z.array(z.string()),
  os: z.string(),
  online: z.boolean(),
  owner: z.string(),
  tags: z.array(z.string()),
  subnetRouter: z.boolean(),
  routedSubnets: z.array(z.string()),
  lastSeen: z.string().optional(),
});

export const edgeSchema = z.object({
  id: z.string(),
  from: z.string(),
  to: z.string(),
  kind: z.enum(["owner", "tag", "subnet", "acl"]),
  labels: z.array(z.string()).optional(),
  protocols: z.array(z.string()).optional(),
  ports: z.array(z.string()).optional(),
  accessScope: z.enum(["ssh", "http", "broad", "custom", "limited", "none"]).optional(),
  policyRefs: z
    .array(
      z.object({
        section: z.string(),
        index: z.number(),
        src: z.string().optional(),
        dst: z.string().optional(),
      }),
    )
    .optional(),
  perspectives: z.array(z.string()).optional(),
});

export const topologyResponseSchema = z.object({
  devices: z.array(deviceSchema),
  edges: z.array(edgeSchema),
});

export const cloudAuthStatusResponseSchema = z.object({
  authenticated: z.boolean(),
  tailnet: z.string().optional(),
  hasPolicy: z.boolean(),
});

export const cloudAuthRequestSchema = z.object({
  tailnet: z.string(),
  apiKey: z.string(),
});

export const policyResponseSchema = z.object({
  tailnet: z.string(),
  hujson: z.string(),
});

export const policySectionEntrySchema = z.object({
  label: z.string(),
  summary: z.string().optional(),
  selectors: z.array(z.string()).optional(),
  value: z.unknown().optional(),
});

export const policySectionSchema = z.object({
  name: z.string(),
  type: z.string(),
  supported: z.boolean(),
  count: z.number(),
  entries: z.array(policySectionEntrySchema).optional(),
  raw: z.unknown().optional(),
  description: z.string().optional(),
});

export const policyMapResponseSchema = z.object({
  tailnet: z.string(),
  hujson: z.string(),
  sections: z.array(policySectionSchema),
  parseError: z.string().optional(),
});

export const policyDraftRequestSchema = z.object({
  sources: z.array(z.string()),
  destinations: z.array(z.string()),
  ports: z.array(z.string()),
  protocol: z.string().optional(),
});

export const aclDraftSchema = z.object({
  action: z.string(),
  src: z.array(z.string()),
  dst: z.array(z.string()),
  proto: z.string().optional(),
});

export const policyDraftResponseSchema = z.object({
  tailnet: z.string(),
  rule: aclDraftSchema,
  hujson: z.string(),
});

export const policyValidateResponseSchema = z.object({
  valid: z.boolean(),
  tailnet: z.string(),
  errors: z.array(z.string()).optional(),
});

export const policySaveResponseSchema = z.object({
  saved: z.boolean(),
  tailnet: z.string(),
  hujson: z.string(),
});

export const errorResponseSchema = z.object({
  error: z.string(),
});

export const topologySnapshotMessageSchema = z.object({
  type: z.literal("topology.snapshot"),
  requestId: z.string().optional(),
  payload: topologyResponseSchema,
});

export const localApiUnavailableMessageSchema = z.object({
  type: z.literal("localapi.unavailable"),
  requestId: z.string().optional(),
  payload: localApiStatusResponseSchema,
});

export const socketMessageSchema = z.discriminatedUnion("type", [
  topologySnapshotMessageSchema,
  localApiUnavailableMessageSchema,
]);

export type HealthResponse = z.infer<typeof healthResponseSchema>;
export type LocalAPIStatusResponse = z.infer<typeof localApiStatusResponseSchema>;
export type Device = z.infer<typeof deviceSchema>;
export type Edge = z.infer<typeof edgeSchema>;
export type TopologyResponse = z.infer<typeof topologyResponseSchema>;
export type CloudAuthStatusResponse = z.infer<typeof cloudAuthStatusResponseSchema>;
export type CloudAuthRequest = z.infer<typeof cloudAuthRequestSchema>;
export type PolicyResponse = z.infer<typeof policyResponseSchema>;
export type PolicyMapResponse = z.infer<typeof policyMapResponseSchema>;
export type PolicySection = z.infer<typeof policySectionSchema>;
export type PolicyDraftRequest = z.infer<typeof policyDraftRequestSchema>;
export type PolicyDraftResponse = z.infer<typeof policyDraftResponseSchema>;
export type PolicyValidateResponse = z.infer<typeof policyValidateResponseSchema>;
export type PolicySaveResponse = z.infer<typeof policySaveResponseSchema>;
export type SocketMessage = z.infer<typeof socketMessageSchema>;
