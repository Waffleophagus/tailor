import { Result } from "better-result";

import {
  localApiStatusResponseSchema,
  topologyResponseSchema,
  type LocalAPIStatusResponse,
  type TopologyResponse,
} from "./schemas";

export async function fetchTailnet(): Promise<
  Result<TopologyResponse, LocalAPIStatusResponse | Error>
> {
  const response = await Result.tryPromise(() => fetch("/api/tailnet"));
  if (Result.isError(response)) {
    return Result.err(toError(response.error));
  }

  const body = await Result.tryPromise(() => response.value.json());
  if (Result.isError(body)) {
    return Result.err(toError(body.error));
  }

  if (response.value.status === 503) {
    const parsed = localApiStatusResponseSchema.safeParse(body.value);
    return parsed.success ? Result.err(parsed.data) : Result.err(parsed.error);
  }

  if (!response.value.ok) {
    return Result.err(new Error(`Tailnet request failed with ${response.value.status}`));
  }

  const parsed = topologyResponseSchema.safeParse(body.value);
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
