<script lang="ts">
	let {
		perspective = $bindable(''),
		selectorOptions = [],
		graphViewMode = $bindable<'current' | 'draft' | 'diff'>('current'),
		hasDraft = false,
		hasPerspectivePreview = false,
		busy = false,
		onApply = () => {},
		onClear = () => {}
	}: {
		perspective?: string;
		selectorOptions?: string[];
		graphViewMode?: 'current' | 'draft' | 'diff';
		hasDraft?: boolean;
		hasPerspectivePreview?: boolean;
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
	<div class="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-2">
		<label
			for="policy-perspective"
			class="text-[0.72rem] font-extrabold tracking-wider text-label uppercase"
		>
			View as
		</label>
		<input
			id="policy-perspective"
			bind:value={perspective}
			list="policy-perspective-options"
			placeholder="Whole tailnet, user@example.com, group:ops, tag:ci, autogroup:member"
			class="min-h-[2.1rem] rounded-md border border-panel-border bg-panel-input px-2 py-[0.4rem] text-[0.83rem] text-primary transition-[border-color,box-shadow] duration-[140ms] ease-out outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)]"
		/>
		<datalist id="policy-perspective-options">
			{#each selectorOptions as selector (selector)}
				<option value={selector}></option>
			{/each}
		</datalist>
		<div class="flex items-center gap-1">
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

	<div class="flex items-center justify-between gap-2">
		<p class="m-0 min-w-0 text-[0.76rem] font-bold text-secondary">
			{#if trimmedPerspective}
				Simulated policy subject:
				<strong class="text-primary">{trimmedPerspective}</strong>
				{hasPerspectivePreview ? ' is active on the graph.' : ' needs simulation.'}
			{:else}
				Saved effective access for the whole tailnet.
			{/if}
		</p>
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
