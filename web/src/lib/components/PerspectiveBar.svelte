<script lang="ts">
	import type { Device, PolicyMapResponse } from '../api/schemas';
	import PerspectiveSelector from './PerspectiveSelector.svelte';

	let {
		perspective = $bindable(''),
		devices = [],
		policyMap,
		graphViewMode = $bindable<'current' | 'draft' | 'diff'>('current'),
		hasDraft = false,
		hasPerspectivePreview = false,
		reachableCount = 0,
		busy = false,
		onApply = () => {},
		onClear = () => {}
	}: {
		perspective?: string;
		devices?: Device[];
		policyMap?: PolicyMapResponse;
		graphViewMode?: 'current' | 'draft' | 'diff';
		hasDraft?: boolean;
		hasPerspectivePreview?: boolean;
		reachableCount?: number;
		busy?: boolean;
		onApply?: () => void;
		onClear?: () => void;
	} = $props();

	const trimmedPerspective = $derived(perspective.trim());
</script>

<div
	class="absolute top-16 left-1/2 z-[3] grid w-[min(42rem,calc(100%-2rem))] -translate-x-1/2 gap-2 rounded-lg border border-graph-border bg-graph-hud-bg p-2 shadow-[0_10px_26px_rgb(23_33_38/8%)] backdrop-blur-sm"
	aria-label="Policy perspective"
>
	<div class="grid grid-cols-[auto_minmax(0,1fr)_auto] items-start gap-2">
		<label
			for="policy-perspective"
			class="pt-[0.45rem] text-[0.72rem] font-extrabold tracking-wider text-label uppercase"
		>
			View as
		</label>
		<div class="min-w-0">
			<PerspectiveSelector
				id="policy-perspective"
				bind:value={perspective}
				{devices}
				{policyMap}
				{hasPerspectivePreview}
				{reachableCount}
				{busy}
				onSimulate={onApply}
			/>
		</div>
		<div class="flex items-center gap-1 pt-[0.15rem]">
			<button
				type="button"
				class="bar-button primary"
				onclick={onApply}
				disabled={busy || !trimmedPerspective}
			>
				Simulate
			</button>
			<button
				type="button"
				class="bar-button"
				onclick={onClear}
				disabled={busy && !trimmedPerspective}
			>
				Clear
			</button>
		</div>
	</div>

	<div class="flex items-center justify-end gap-2">
		<div class="flex shrink-0 rounded-md border border-panel-border bg-panel-input p-[0.12rem]">
			{#each ['current', 'draft', 'diff'] as mode (mode)}
				<button
					type="button"
					class="mode-button"
					data-active={graphViewMode === mode}
					disabled={mode !== 'current' && !hasDraft && !hasPerspectivePreview}
					onclick={() => (graphViewMode = mode as 'current' | 'draft' | 'diff')}
				>
					{mode === 'current' ? 'Current' : mode === 'draft' ? 'Draft' : 'Diff'}
				</button>
			{/each}
		</div>
	</div>
</div>

<style>
	@reference "../../app.css";

	.bar-button {
		@apply min-h-[2.1rem] rounded-md border border-panel-border bg-panel-weak px-2 py-[0.35rem] text-[0.78rem] font-extrabold whitespace-nowrap text-primary transition-[background-color,border-color,color] duration-[140ms] ease-out hover:border-teal hover:bg-hover disabled:cursor-not-allowed disabled:opacity-[0.55];
	}
	.bar-button.primary {
		@apply border-panel-accent bg-panel-accent text-panel-fg hover:bg-panel-accent;
	}
	.mode-button {
		@apply rounded-sm border-0 bg-transparent px-2 py-[0.28rem] text-[0.72rem] font-extrabold text-secondary transition-[background-color,color] duration-[140ms] ease-out disabled:cursor-not-allowed disabled:opacity-[0.45];
	}
	.mode-button[data-active='true'] {
		@apply bg-hover text-primary;
	}
</style>
