import { Result } from 'better-result';

import {
	cloudAuthStatusResponseSchema,
	errorResponseSchema,
	policyEvaluateDraftResponseSchema,
	policyDiscardStagedResponseSchema,
	policyMapResponseSchema,
	policyResponseSchema,
	policyDraftResponseSchema,
	policyStagedDraftResponseSchema,
	policyStagedResponseSchema,
	policyStageResponseSchema,
	policySaveResponseSchema,
	policyValidateResponseSchema,
	setupGrantResponseSchema,
	type CloudAuthRequest,
	type CloudAuthStatusResponse,
	type SetupGrantResponse,
	type PolicyDraftRequest,
	type PolicyDraftResponse,
	type PolicyDiscardStagedResponse,
	type PolicyEvaluateDraftRequest,
	type PolicyEvaluateDraftResponse,
	policyMutationResponseSchema,
	type PolicyMutationRequest,
	type PolicyMutationResponse,
	type PolicyMapResponse,
	type PolicyResponse,
	type PolicySaveRequest,
	type PolicySaveResponse,
	type PolicyStagedDraftResponse,
	type PolicyStagedResponse,
	type PolicyStageRequest,
	type PolicyStageResponse,
	type PolicyValidateResponse
} from './schemas';

export async function fetchCloudStatus(): Promise<Result<CloudAuthStatusResponse, Error>> {
	return fetchJSON('/api/cloud/status', cloudAuthStatusResponseSchema);
}

export async function authenticateCloud(
	request: CloudAuthRequest
): Promise<Result<CloudAuthStatusResponse, Error>> {
	return fetchJSON('/api/cloud/auth', cloudAuthStatusResponseSchema, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(request)
	});
}

export async function saveSetupGrant(options?: {
	editedSnippet?: string;
}): Promise<Result<SetupGrantResponse, Error>> {
	let body: string | undefined;
	if (options?.editedSnippet) {
		const parsed = JSON.parse(options.editedSnippet) as {
			grants?: unknown[];
			src?: unknown;
		};
		const grant = Array.isArray(parsed.grants) ? parsed.grants[0] : parsed;
		body = JSON.stringify({ grant });
	}
	return fetchJSON('/api/cloud/setup-grant', setupGrantResponseSchema, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body
	});
}

export async function fetchPolicy(): Promise<Result<PolicyResponse, Error>> {
	return fetchJSON('/api/policy', policyResponseSchema);
}

export async function fetchPolicyMap(): Promise<Result<PolicyMapResponse, Error>> {
	return fetchJSON('/api/policy/map', policyMapResponseSchema);
}

export async function draftPolicyRule(
	request: PolicyDraftRequest
): Promise<Result<PolicyDraftResponse, Error>> {
	return fetchJSON('/api/policy/draft', policyDraftResponseSchema, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(request)
	});
}

export async function evaluatePolicyDraft(
	request: PolicyEvaluateDraftRequest
): Promise<Result<PolicyEvaluateDraftResponse, Error>> {
	return fetchJSON('/api/policy/evaluate-draft', policyEvaluateDraftResponseSchema, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(request)
	});
}

export async function validatePolicyDraft(
	hujson: string
): Promise<Result<PolicyValidateResponse, Error>> {
	return fetchJSON('/api/policy/validate', policyValidateResponseSchema, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ hujson })
	});
}

export async function mutatePolicyDraft(
	request: PolicyMutationRequest
): Promise<Result<PolicyMutationResponse, Error>> {
	return fetchJSON('/api/policy/mutate', policyMutationResponseSchema, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(request)
	});
}

export async function stagePolicyDraft(
	request: PolicyStageRequest
): Promise<Result<PolicyStageResponse, Error>> {
	return fetchJSON('/api/policy/stage', policyStageResponseSchema, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(request)
	});
}

export async function fetchStagedPolicyDrafts(): Promise<Result<PolicyStagedResponse, Error>> {
	return fetchJSON('/api/policy/staged', policyStagedResponseSchema);
}

export async function fetchStagedPolicyDraft(
	id: string
): Promise<Result<PolicyStagedDraftResponse, Error>> {
	return fetchJSON(`/api/policy/staged/${encodeURIComponent(id)}`, policyStagedDraftResponseSchema);
}

export async function discardStagedPolicyDraft(
	id: string
): Promise<Result<PolicyDiscardStagedResponse, Error>> {
	return fetchJSON(
		`/api/policy/staged/${encodeURIComponent(id)}`,
		policyDiscardStagedResponseSchema,
		{
			method: 'DELETE'
		}
	);
}

export async function saveValidatedPolicyDraft(
	request: PolicySaveRequest
): Promise<Result<PolicySaveResponse, Error>> {
	return fetchJSON('/api/policy/save', policySaveResponseSchema, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(request)
	});
}

async function fetchJSON<T>(
	url: string,
	schema: {
		safeParse: (value: unknown) => { success: true; data: T } | { success: false; error: Error };
	},
	init?: RequestInit
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
