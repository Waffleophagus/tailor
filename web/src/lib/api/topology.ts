import { Result } from "better-result";

import {
  errorResponseSchema,
  topologyResponseSchema,
  type TopologyResponse,
} from "./schemas";

export async function fetchTopology(): Promise<Result<TopologyResponse, Error>> {
  const response = await Result.tryPromise(() => fetch("/api/topology"));
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
      : `/api/topology failed with ${response.value.status}`;
    return Result.err(new Error(message));
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
