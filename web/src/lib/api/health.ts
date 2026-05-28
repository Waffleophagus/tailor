import { Result } from 'better-result';

import { healthResponseSchema, type HealthResponse } from './schemas';

export async function fetchHealth(): Promise<Result<HealthResponse, Error>> {
	const response = await Result.tryPromise(() => fetch('/api/health'));
	if (Result.isError(response)) {
		return Result.err(toError(response.error));
	}

	if (!response.value.ok) {
		return Result.err(new Error(`Health check failed with ${response.value.status}`));
	}

	const body = await Result.tryPromise(() => response.value.json());
	if (Result.isError(body)) {
		return Result.err(toError(body.error));
	}

	const parsed = healthResponseSchema.safeParse(body.value);
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
