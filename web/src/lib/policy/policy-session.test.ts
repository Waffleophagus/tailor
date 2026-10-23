import { Result } from 'better-result';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import * as cloud from '../api/cloud';
import type { PolicyEvaluateDraftResponse, StagedDraft } from '../api/schemas';
import { PolicySession } from './policy-session.svelte';

vi.mock('../api/cloud', () => ({
	discardStagedPolicyDraft: vi.fn(),
	evaluatePolicyDraft: vi.fn(),
	fetchPolicy: vi.fn(),
	fetchStagedPolicyDraft: vi.fn(),
	fetchStagedPolicyDrafts: vi.fn(),
	saveValidatedPolicyDraft: vi.fn(),
	stagePolicyDraft: vi.fn(),
	validatePolicyDraft: vi.fn()
}));

const api = vi.mocked(cloud);
const SAVED = '{"acls":[]}';

function evaluation(label: string) {
	return { label } as unknown as PolicyEvaluateDraftResponse;
}

function deferred<T>() {
	let resolve!: (value: T) => void;
	const promise = new Promise<T>((r) => (resolve = r));
	return { promise, resolve };
}

async function loadedSession() {
	api.fetchPolicy.mockResolvedValue(Result.ok({ tailnet: 'example.com', hujson: SAVED }) as never);
	api.evaluatePolicyDraft.mockResolvedValue(Result.ok(evaluation('saved')) as never);
	api.fetchStagedPolicyDrafts.mockResolvedValue(Result.ok({ drafts: [] }) as never);
	const errors: string[] = [];
	const session = new PolicySession({ canEdit: () => true, onError: (m) => errors.push(m) });
	await session.loadPolicy();
	return { session, errors };
}

describe('PolicySession', () => {
	beforeEach(() => {
		vi.resetAllMocks();
	});

	it('finishes evaluating and loading staged drafts before loadPolicy resolves', async () => {
		api.fetchPolicy.mockResolvedValue(
			Result.ok({ tailnet: 'example.com', hujson: SAVED }) as never
		);
		api.evaluatePolicyDraft.mockResolvedValue(Result.ok(evaluation('saved')) as never);
		const stagedList = deferred<unknown>();
		api.fetchStagedPolicyDrafts.mockReturnValue(stagedList.promise as never);
		const session = new PolicySession({ canEdit: () => true, onError: () => {} });

		let resolved = false;
		const loading = session.loadPolicy().then(() => (resolved = true));
		await vi.waitFor(() => expect(api.fetchStagedPolicyDrafts).toHaveBeenCalled());
		expect(resolved).toBe(false);

		const draft = { id: 'draft-1', source: 'mcp' } as unknown as StagedDraft;
		stagedList.resolve(Result.ok({ drafts: [draft] }));
		await loading;
		expect(session.policyEvaluation).toEqual(evaluation('saved'));
		expect(session.stagedDrafts).toEqual([draft]);
	});

	it('only marks the text that was submitted for validation as validated', async () => {
		const { session } = await loadedSession();
		session.editorHuJSON = '{"acls":[{"action":"accept"}]}';
		const validation = deferred<unknown>();
		api.validatePolicyDraft.mockReturnValue(validation.promise as never);

		const validating = session.validate();
		session.editorHuJSON = '{"acls":[{"action":"accept"}]} // typed while validating';
		validation.resolve(Result.ok({ valid: true }));
		await validating;

		expect(session.validatedHuJSON).toBe('{"acls":[{"action":"accept"}]}');
		expect(session.validationStale).toBe(true);
		expect(session.hasValidatedPending).toBe(false);
	});

	it('ignores a preview that finishes after the edits were discarded', async () => {
		const { session } = await loadedSession();
		session.editorHuJSON = '{"acls":[{"action":"accept"}]}';
		api.validatePolicyDraft.mockResolvedValue(Result.ok({ valid: true }) as never);
		const preview = deferred<unknown>();
		api.evaluatePolicyDraft.mockReturnValue(preview.promise as never);

		const validating = session.validate();
		await vi.waitFor(() => expect(api.evaluatePolicyDraft).toHaveBeenCalled());
		session.discardChanges();
		preview.resolve(Result.ok(evaluation('stale preview')));
		await validating;

		expect(session.previewEvaluation).toBeUndefined();
	});

	it('keeps a loaded staged-draft preview when an older saved-policy evaluation lands', async () => {
		vi.useFakeTimers();
		try {
			const { session } = await loadedSession();
			const savedEval = deferred<unknown>();
			api.evaluatePolicyDraft.mockReturnValue(savedEval.promise as never);
			session.scheduleTopologyRefresh();
			await vi.advanceTimersByTimeAsync(300);
			expect(api.evaluatePolicyDraft).toHaveBeenCalled();

			const draft = {
				id: 'draft-1',
				source: 'mcp',
				valid: true,
				hujson: '{"acls":[{"action":"accept"}]}',
				evaluation: evaluation('draft preview')
			} as unknown as StagedDraft;
			api.fetchStagedPolicyDraft.mockResolvedValue(Result.ok({ draft }) as never);
			await session.loadStagedDraft(draft);

			savedEval.resolve(Result.ok(evaluation('refreshed saved')));
			await vi.runAllTimersAsync();

			expect(session.previewEvaluation).toEqual(evaluation('draft preview'));
			expect(session.policyEvaluation).toEqual(evaluation('refreshed saved'));
		} finally {
			vi.useRealTimers();
		}
	});
});
