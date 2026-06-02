<script lang="ts">
	let {
		hasSelection = false,
		cloudAuthenticated = false,
		graphMode = $bindable<'focused' | 'all'>('focused'),
		activeSheet = null,
		onOpenSheet,
		onZoomIn,
		onZoomOut,
		onFit
	}: {
		hasSelection?: boolean;
		cloudAuthenticated?: boolean;
		graphMode?: 'focused' | 'all';
		activeSheet?: 'filters' | 'details' | 'legend' | null;
		onOpenSheet?: (sheet: 'filters' | 'details' | 'legend') => void;
		onZoomIn?: () => void;
		onZoomOut?: () => void;
		onFit?: () => void;
	} = $props();
</script>

<nav class="mobile-bar" aria-label="Graph controls">
	<div class="row row-sheets">
		<button
			type="button"
			class="bar-button"
			data-active={activeSheet === 'filters'}
			onclick={() => onOpenSheet?.('filters')}
		>
			Filters
		</button>
		<button
			type="button"
			class="bar-button"
			data-active={activeSheet === 'details'}
			data-highlight={hasSelection}
			onclick={() => onOpenSheet?.('details')}
		>
			Details
			{#if hasSelection}
				<span class="badge" aria-hidden="true"></span>
			{/if}
		</button>
		<button
			type="button"
			class="bar-button"
			data-active={activeSheet === 'legend'}
			onclick={() => onOpenSheet?.('legend')}
		>
			Legend
		</button>
	</div>

	<div class="row row-graph" class:row-graph-zoom-only={!cloudAuthenticated}>
		{#if cloudAuthenticated}
			<div class="mode-toggle" role="group" aria-label="Graph mode">
				{#each ['focused', 'all'] as mode (mode)}
					<button
						type="button"
						class="mode-button"
						data-active={graphMode === mode}
						onclick={() => (graphMode = mode as 'focused' | 'all')}
					>
						{mode === 'focused' ? 'Focused' : 'All'}
					</button>
				{/each}
			</div>
		{/if}
		<div class="zoom-group" aria-label="Zoom controls">
			<button type="button" class="icon-button" aria-label="Zoom out" onclick={onZoomOut}>−</button>
			<button type="button" class="icon-button" aria-label="Fit to view" onclick={onFit}>⌖</button>
			<button type="button" class="icon-button" aria-label="Zoom in" onclick={onZoomIn}>+</button>
		</div>
	</div>
</nav>

<style>
	@reference "../../app.css";
	.mobile-bar {
		@apply pointer-events-auto fixed right-0 bottom-0 left-0 z-20 flex flex-col gap-2 border-t border-graph-border bg-legend-bg px-3 pt-2 shadow-[0_-8px_24px_rgb(23_33_38/10%)];
		padding-bottom: max(0.5rem, env(safe-area-inset-bottom, 0px));
		touch-action: manipulation;
	}
	.row {
		@apply flex w-full min-w-0 items-stretch gap-2;
	}
	.row-sheets {
		@apply gap-1.5;
	}
	.bar-button {
		@apply relative flex min-h-11 min-w-0 flex-1 cursor-pointer touch-manipulation items-center justify-center rounded-md border border-panel-border bg-panel-weak px-1 text-[0.8rem] font-extrabold text-secondary transition-[background-color,border-color,color] duration-[140ms] ease-out active:bg-hover;
	}
	.bar-button[data-active='true'] {
		@apply border-teal bg-hover text-primary;
	}
	.bar-button[data-highlight='true']:not([data-active='true']) {
		@apply border-teal/50 text-primary;
	}
	.badge {
		@apply absolute top-1.5 right-1.5 h-2 w-2 rounded-full bg-teal;
	}
	.row-graph {
		@apply justify-between gap-2;
	}
	.row-graph-zoom-only {
		@apply justify-end;
	}
	.mode-toggle {
		@apply inline-flex shrink-0 rounded-md border border-panel-border bg-panel-input p-[0.12rem];
	}
	.mode-button {
		@apply min-h-11 min-w-[4.25rem] touch-manipulation rounded-sm border-0 bg-transparent px-2.5 text-[0.75rem] font-extrabold text-secondary transition-[background-color,color] duration-[140ms] ease-out active:bg-hover;
	}
	.mode-button[data-active='true'] {
		@apply bg-hover text-primary;
	}
	.zoom-group {
		@apply flex shrink-0 items-center gap-1.5;
	}
	.icon-button {
		@apply grid h-11 w-11 shrink-0 cursor-pointer touch-manipulation place-items-center rounded-md border border-panel-border bg-panel-weak text-lg leading-none font-extrabold text-primary transition-[background-color,border-color] duration-[160ms] ease-out active:border-teal active:bg-hover;
	}
</style>
