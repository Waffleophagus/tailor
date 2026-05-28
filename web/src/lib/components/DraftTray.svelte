<script lang="ts">
	import type { PolicyEvaluateDraftResponse } from '../api/schemas';

	let {
		draftEvaluation = undefined,
		draftRuleText = '',
		draftValid = null,
		editBusy = false,
		editStatus = '',
		onValidate = () => {},
		onSave = () => {},
		onDiscard = () => {},
		onOpenAdvanced = () => {}
	}: {
		draftEvaluation?: PolicyEvaluateDraftResponse;
		draftRuleText?: string;
		draftValid?: boolean | null;
		editBusy?: boolean;
		editStatus?: string;
		onValidate?: () => void;
		onSave?: () => void;
		onDiscard?: () => void;
		onOpenAdvanced?: () => void;
	} = $props();

	const addedCount = $derived(draftEvaluation?.added.length ?? 0);
	const removedCount = $derived(draftEvaluation?.removed.length ?? 0);
	const changedCount = $derived(draftEvaluation?.changed.length ?? 0);
	const broadCount = $derived(draftEvaluation?.broadAccess.length ?? 0);
	const unresolvedCount = $derived(draftEvaluation?.unresolvedSelectors.length ?? 0);
	const hasDraft = $derived(Boolean(draftRuleText || draftEvaluation));
</script>

{#if hasDraft}
	<section
		class="absolute right-4 bottom-4 left-4 z-[3] grid gap-3 rounded-xl border border-panel-border bg-panel-bg p-3 shadow-[0_18px_48px_rgb(23_33_38/12%)] backdrop-blur-md lg:left-auto lg:w-[44rem]"
		aria-label="Staged policy change"
	>
		<div class="flex items-start justify-between gap-3">
			<div>
				<p class="m-0 text-[0.72rem] font-extrabold tracking-wider text-label uppercase">
					Staged policy change
				</p>
				<p class="mt-1 mb-0 text-[0.83rem] font-semibold text-secondary">
					Review impact, validate with Tailscale, then save when the graph looks right.
				</p>
			</div>
			{#if draftValid !== null}
				<span
					class="rounded-full border px-2 py-[0.25rem] text-[0.72rem] font-extrabold {draftValid
						? 'border-ok text-ok'
						: 'border-danger text-danger'}"
				>
					{draftValid ? 'Validated' : 'Invalid'}
				</span>
			{/if}
		</div>

		{#if draftRuleText}
			<code
				class="block max-h-16 overflow-auto rounded-lg border border-panel-border bg-panel-weak p-2 text-[0.72rem] leading-relaxed text-primary"
			>
				{draftRuleText}
			</code>
		{/if}

		<div class="grid grid-cols-2 gap-2 sm:grid-cols-5">
			<span class="metric"><strong>{addedCount}</strong> added</span>
			<span class="metric"><strong>{removedCount}</strong> removed</span>
			<span class="metric"><strong>{changedCount}</strong> changed</span>
			<span class="metric {broadCount > 0 ? 'warn' : ''}"><strong>{broadCount}</strong> broad</span>
			<span class="metric {unresolvedCount > 0 ? 'warn' : ''}"
				><strong>{unresolvedCount}</strong> unresolved</span
			>
		</div>

		<div class="flex flex-wrap items-center justify-between gap-2">
			<p class="m-0 min-w-0 text-[0.78rem] font-bold text-secondary">{editStatus}</p>
			<div class="flex flex-wrap gap-2">
				<button type="button" class="tray-button" onclick={onOpenAdvanced}>HuJSON</button>
				<button type="button" class="tray-button" onclick={onDiscard} disabled={editBusy}
					>Discard</button
				>
				<button type="button" class="tray-button" onclick={onValidate} disabled={editBusy}
					>Validate</button
				>
				<button
					type="button"
					class="tray-button primary"
					onclick={onSave}
					disabled={editBusy || draftValid !== true}
				>
					Save policy
				</button>
			</div>
		</div>
	</section>
{/if}

<style>
	@reference "../../app.css";

	.metric {
		@apply rounded-lg border border-panel-border bg-panel-weak px-2 py-[0.45rem] text-[0.76rem] font-extrabold text-secondary;
	}
	.metric strong {
		@apply mr-1 text-primary;
	}
	.metric.warn {
		@apply border-warn text-warn;
	}
	.metric.warn strong {
		@apply text-warn;
	}
	.tray-button {
		@apply rounded-md border border-panel-border bg-panel-weak px-3 py-[0.45rem] text-[0.78rem] font-extrabold text-primary transition-[background-color,border-color,color] duration-[140ms] ease-out hover:border-teal hover:bg-hover disabled:cursor-not-allowed disabled:opacity-[0.55];
	}
	.tray-button.primary {
		@apply border-panel-accent bg-panel-accent text-panel-fg hover:bg-panel-accent;
	}
</style>
