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
});

export const topologyResponseSchema = z.object({
  devices: z.array(deviceSchema),
  edges: z.array(edgeSchema),
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
export type SocketMessage = z.infer<typeof socketMessageSchema>;
