import { Result } from "better-result";

import {
  cloudAuthStatusResponseSchema,
  errorResponseSchema,
  policyResponseSchema,
  type CloudAuthRequest,
  type CloudAuthStatusResponse,
  type PolicyResponse,
} from "./schemas";

export async function fetchCloudStatus(): Promise<Result<CloudAuthStatusResponse, Error>> {
  return fetchJSON("/api/cloud/status", cloudAuthStatusResponseSchema);
}

export async function authenticateCloud(
  request: CloudAuthRequest,
): Promise<Result<CloudAuthStatusResponse, Error>> {
  return fetchJSON("/api/cloud/auth", cloudAuthStatusResponseSchema, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
}

export async function fetchPolicy(): Promise<Result<PolicyResponse, Error>> {
  return fetchJSON("/api/policy", policyResponseSchema);
}

async function fetchJSON<T>(
  url: string,
  schema: { safeParse: (value: unknown) => { success: true; data: T } | { success: false; error: Error } },
  init?: RequestInit,
): Promise<Result<T, Error>> {
  const response = await Result.tryPromise(() => fetch(url, init));
  if (Result.isError(response)) {
    return Result.err(toError(response.error));
  }

  const body = await Result.tryPromise(() => response.value.json());
  if (Result.isError(body)) {
    return Result.err(toError(body.error));
  }

  if (!response.value.ok) {
    const parsedError = errorResponseSchema.safeParse(body.value);
    const message = parsedError.success
      ? parsedError.data.error
      : `${url} failed with ${response.value.status}`;
    return Result.err(new Error(message));
  }

  const parsed = schema.safeParse(body.value);
  if (!parsed.success) {
    return Result.err(parsed.error);
  }

  return Result.ok(parsed.data);
}

function toError(value: unknown): Error {
  if (value instanceof Error) {
    return value;
  }

  return new Error(String(value));
}
