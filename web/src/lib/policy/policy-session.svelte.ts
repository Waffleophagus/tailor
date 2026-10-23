import {
	discardStagedPolicyDraft,
	evaluatePolicyDraft,
	fetchPolicy,
	fetchStagedPolicyDraft,
	fetchStagedPolicyDrafts,
	saveValidatedPolicyDraft,
	stagePolicyDraft,
	validatePolicyDraft
} from '../api/cloud';
import type { PolicyEvaluateDraftResponse, PolicyResponse, StagedDraft } from '../api/schemas';

export interface PolicySessionOptions {
	/** Whether the current caller may read and edit the policy. */
	canEdit: () => boolean;
	/** Surfaces an error to the user; an empty string clears it. */
	onError: (message: string) => void;
}

const TOPOLOGY_REFRESH_DEBOUNCE_MS = 300;

/**
 * The policy-editing session: the saved policy and its evaluated graph edges, the
 * editor draft and its validation/preview, and drafts staged by the UI or MCP agents.
 */
export class PolicySession {
	policy = $state<PolicyResponse | undefined>();
	policyEvaluation = $state<PolicyEvaluateDraftResponse | undefined>();
	previewEvaluation = $state<PolicyEvaluateDraftResponse | undefined>();
	editorHuJSON = $state('');
	validatedHuJSON = $state('');
	editorOpen = $state(false);
	editorValid = $state<boolean | null>(null);
	editorBusy = $state(false);
	editorStatus = $state('');
	editorErrors = $state<string[]>([]);
	stagedDrafts = $state<StagedDraft[]>([]);
	selectedStagedDraft = $state<StagedDraft | undefined>();
	stagedBusy = $state(false);

	editorDirty = $derived(Boolean(this.policy && this.editorHuJSON !== this.policy.hujson));
	validationStale = $derived(
		this.validatedHuJSON !== '' && this.editorHuJSON !== this.validatedHuJSON
	);
	effectiveValid = $derived(this.validationStale ? null : this.editorValid);
	effectiveErrors = $derived(this.validationStale ? [] : this.editorErrors);
	effectivePreviewEvaluation = $derived(this.validationStale ? undefined : this.previewEvaluation);
	stagedPreviewActive = $derived(
		Boolean(this.selectedStagedDraft?.valid && this.previewEvaluation)
	);
	hasValidatedPending = $derived(
		!this.validationStale && this.editorValid === true && this.validatedHuJSON !== ''
	);
	hasPendingDraft = $derived(this.editorDirty || this.hasValidatedPending);
	mcpStagedDrafts = $derived(this.stagedDrafts.filter((draft) => draft.source === 'mcp'));

	#canEdit: () => boolean;
	#reportError: (message: string) => void;
	#refreshTimer: ReturnType<typeof setTimeout> | undefined;
	// Evaluations race with the 2s topology refresh, so only the latest request of each
	// kind may apply its result. Bumping #previewEvalSeq also cancels in-flight previews.
	#savedEvalSeq = 0;
	#previewEvalSeq = 0;

	constructor(options: PolicySessionOptions) {
		this.#canEdit = options.canEdit;
		this.#reportError = options.onError;
	}

	/** Re-evaluate the graph shortly after a topology change (debounced). */
	scheduleTopologyRefresh = () => {
		if (!this.#canEdit()) return;
		clearTimeout(this.#refreshTimer);
		this.#refreshTimer = setTimeout(() => {
			this.#refreshTimer = undefined;
			void this.#runTopologyRefresh();
		}, TOPOLOGY_REFRESH_DEBOUNCE_MS);
	};

	dispose = () => {
		clearTimeout(this.#refreshTimer);
		this.#refreshTimer = undefined;
	};

	async #runTopologyRefresh() {
		if (!this.#canEdit()) return;

		if (this.hasValidatedPending && !this.validationStale && this.validatedHuJSON) {
			await this.#evaluate(this.validatedHuJSON, true);
			return;
		}
		if (this.stagedPreviewActive && this.editorHuJSON && !this.validationStale) {
			await this.#evaluate(this.editorHuJSON, true);
			return;
		}
		const savedPolicy = this.policy;
		if (!savedPolicy || this.editorDirty || this.hasValidatedPending) return;
		await this.#evaluate(savedPolicy.hujson);
	}

	#setPreviewEvaluation(value: PolicyEvaluateDraftResponse | undefined) {
		this.#previewEvalSeq++;
		this.previewEvaluation = value;
	}

	async #evaluate(hujson: string, preview = false) {
		const seq = preview ? ++this.#previewEvalSeq : ++this.#savedEvalSeq;
		const result = await evaluatePolicyDraft({ hujson });
		if (seq !== (preview ? this.#previewEvalSeq : this.#savedEvalSeq)) return;
		result.match({
			ok: (value) => {
				if (preview) {
					this.previewEvaluation = value;
					return;
				}
				this.policyEvaluation = value;
			},
			err: (error) => {
				if (preview) {
					this.previewEvaluation = undefined;
					this.editorStatus = 'Validated, but graph preview is unavailable.';
				}
				this.#reportError(error.message);
			}
		});
	}

	#resetEditorFromPolicy() {
		if (!this.policy) return;
		this.editorHuJSON = this.policy.hujson;
		this.validatedHuJSON = '';
		this.editorValid = null;
		this.editorStatus = '';
		this.editorErrors = [];
		this.#setPreviewEvaluation(undefined);
		this.selectedStagedDraft = undefined;
	}

	loadPolicy = async () => {
		const result = await fetchPolicy();
		await result.match({
			ok: async (value) => {
				this.policy = value;
				this.#resetEditorFromPolicy();
				this.#reportError('');
				await this.#evaluate(value.hujson);
				await this.loadStagedDrafts();
			},
			err: async (error) => {
				this.#reportError(error.message);
			}
		});
	};

	loadStagedDrafts = async () => {
		if (!this.#canEdit()) return;
		const result = await fetchStagedPolicyDrafts();
		result.match({
			ok: (value) => this.setStagedDrafts(value.drafts),
			err: (error) => this.#reportError(error.message)
		});
	};

	setStagedDrafts = (drafts: StagedDraft[]) => {
		this.stagedDrafts = drafts;
		const selected = this.selectedStagedDraft;
		if (!selected) return;
		const found = drafts.find((draft) => draft.id === selected.id);
		if (selected.hujson && found && !found.hujson) {
			found.hujson = selected.hujson;
		}
		this.selectedStagedDraft = found;
	};

	openEditor = () => {
		const open = () => {
			if (this.policy && !this.hasPendingDraft) {
				this.#resetEditorFromPolicy();
			}
			this.editorOpen = true;
		};
		if (this.policy) {
			open();
			return;
		}
		void this.loadPolicy().then(open);
	};

	closeEditor = () => {
		this.editorOpen = false;
		if (!this.hasValidatedPending) {
			this.#setPreviewEvaluation(undefined);
		}
	};

	validate = async () => {
		if (this.editorBusy || !this.editorDirty) return;
		this.editorBusy = true;
		this.#reportError('');
		this.editorErrors = [];
		// The textarea stays editable while validating; only mark the submitted text as validated.
		const hujson = this.editorHuJSON;
		const result = await validatePolicyDraft(hujson);
		await result.match({
			ok: async (value) => {
				this.editorValid = value.valid;
				if (value.valid) {
					this.validatedHuJSON = hujson;
					this.editorStatus = 'Policy validated. Preview updated on the graph.';
					await this.#evaluate(hujson, true);
				} else {
					this.validatedHuJSON = '';
					this.editorStatus = 'Fix validation errors before saving.';
					this.editorErrors = value.errors ?? ['Policy failed validation.'];
					this.#setPreviewEvaluation(undefined);
				}
			},
			err: async (error) => {
				this.editorValid = false;
				this.validatedHuJSON = '';
				this.editorErrors = [error.message];
				this.editorStatus = 'Validation failed.';
				this.#setPreviewEvaluation(undefined);
			}
		});
		this.editorBusy = false;
	};

	discardChanges = () => {
		this.#resetEditorFromPolicy();
	};

	loadStagedDraft = async (draft: StagedDraft) => {
		if (!this.policy || this.stagedBusy) return;
		this.stagedBusy = true;
		const result = await fetchStagedPolicyDraft(draft.id);
		result.match({
			ok: (value) => {
				this.selectedStagedDraft = value.draft;
				this.editorHuJSON = value.draft.hujson ?? '';
				this.validatedHuJSON = value.draft.hujson ?? '';
				this.editorValid = value.draft.valid;
				this.editorErrors = value.draft.errors ?? [];
				this.#setPreviewEvaluation(value.draft.evaluation);
				this.editorStatus = `${value.draft.source.toUpperCase()} staged draft loaded for review.`;
				this.editorOpen = true;
			},
			err: (error) => this.#reportError(error.message)
		});
		this.stagedBusy = false;
	};

	discardStagedDraft = async (draft: StagedDraft) => {
		if (this.stagedBusy) return;
		this.stagedBusy = true;
		const result = await discardStagedPolicyDraft(draft.id);
		result.match({
			ok: () => {
				this.stagedDrafts = this.stagedDrafts.filter((item) => item.id !== draft.id);
				if (this.selectedStagedDraft?.id === draft.id) {
					this.#resetEditorFromPolicy();
				}
			},
			err: (error) => this.#reportError(error.message)
		});
		this.stagedBusy = false;
	};

	save = async () => {
		if (this.editorBusy || !this.hasValidatedPending) return;
		this.editorBusy = true;
		this.#reportError('');
		let result: Awaited<ReturnType<typeof saveValidatedPolicyDraft>> | undefined;
		const selected = this.selectedStagedDraft;
		if (selected?.hujson === this.validatedHuJSON) {
			result = await saveValidatedPolicyDraft({
				draftId: selected.id,
				draftHash: selected.draftHash
			});
		} else {
			const staged = await stagePolicyDraft({
				hujson: this.validatedHuJSON,
				source: 'ui',
				summary: 'Staged from policy editor'
			});
			await staged.match({
				ok: async (value) => {
					result = await saveValidatedPolicyDraft({
						draftId: value.draft.id,
						draftHash: value.draft.draftHash
					});
				},
				err: async (error) => this.#reportError(error.message)
			});
		}
		if (!result) {
			this.editorBusy = false;
			return;
		}
		await result.match({
			ok: async (value) => {
				const savedDraftID = this.selectedStagedDraft?.id;
				this.policy = { tailnet: value.tailnet, hujson: value.hujson };
				this.#resetEditorFromPolicy();
				if (savedDraftID) {
					this.stagedDrafts = this.stagedDrafts.filter((draft) => draft.id !== savedDraftID);
				}
				this.editorOpen = false;
				this.editorStatus = 'Policy saved.';
				await this.#evaluate(value.hujson);
				await this.loadStagedDrafts();
			},
			err: async (error) => this.#reportError(error.message)
		});
		this.editorBusy = false;
	};
}
