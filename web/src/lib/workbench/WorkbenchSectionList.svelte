<script lang="ts">
	import type { PolicyMapResponse } from '../api/schemas';
	import {
		canSimulateEntry,
		entryCountForRoute,
		filterSectionsByQuery,
		sectionsForRoute,
		simulationTierLabel,
		workbenchNavItem,
		type WorkbenchRoute
	} from './nav';

	let {
		route,
		policyMap,
		search = $bindable(''),
		onViewAs = () => {}
	}: {
		route: WorkbenchRoute;
		policyMap?: PolicyMapResponse;
		search?: string;
		onViewAs?: (selector: string) => void;
	} = $props();

	const navItem = $derived(workbenchNavItem(route));
	const sections = $derived(filterSectionsByQuery(sectionsForRoute(policyMap, route), search));
	const entryCount = $derived(entryCountForRoute(policyMap, route));

	function formatValue(value: unknown) {
		if (value === undefined || value === null) return '';
		if (typeof value === 'string') return value;
		return JSON.stringify(value, null, 2);
	}
</script>

<section class="flex min-h-0 flex-1 flex-col" aria-label={navItem?.label ?? 'Policy section'}>
	<header class="border-b border-panel-strong px-5 py-4">
		<h2 class="m-0 text-lg font-extrabold text-primary">{navItem?.label ?? route}</h2>
		<p class="mt-1 mb-0 text-[0.82rem] text-secondary">
			{entryCount} entries
			{#if navItem && navItem.simulationTier !== 'graph-simulated'}
				· {simulationTierLabel(navItem.simulationTier)}
			{/if}
		</p>
		<input
			bind:value={search}
			class="search-input mt-3"
			placeholder={navItem?.searchPlaceholder ?? 'Search'}
		/>
	</header>
	<div class="entry-list">
		{#each sections as section (section.name)}
			{#each section.entries ?? [] as entry (entry.label)}
				<article class="entry-card">
					<strong>{entry.label}</strong>
					{#if entry.summary}<p>{entry.summary}</p>{/if}
					{#if canSimulateEntry(section.name, entry.label)}
						<button type="button" class="link-button" onclick={() => onViewAs(entry.label)}
							>View as</button
						>
					{/if}
					{#if entry.value !== undefined}<pre class="json-preview">{formatValue(
								entry.value
							)}</pre>{/if}
				</article>
			{/each}
		{/each}
	</div>
</section>

<style>
	@reference "../../app.css";
	.search-input {
		@apply min-h-[2.35rem] w-full rounded-md border border-panel-border bg-panel-input px-3 py-2 text-[0.86rem];
	}
	.entry-list {
		@apply grid gap-3 overflow-auto p-4;
	}
	.entry-card {
		@apply rounded-md border border-panel-border bg-panel-input p-3;
	}
	.link-button {
		@apply mt-2 cursor-pointer rounded border border-panel-border bg-panel-weak px-2 py-1 text-[0.72rem] font-extrabold text-teal;
	}
	.json-preview {
		@apply mt-2 overflow-auto rounded bg-[oklch(0.96_0.009_158)] p-2 font-mono text-[0.76rem];
	}
</style>
