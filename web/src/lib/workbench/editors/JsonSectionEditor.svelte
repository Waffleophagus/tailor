<script lang="ts">
	import type { PolicyMapResponse } from '../../api/schemas';
	import type { PolicyMutation } from '../../draft/types';
	import {
		entryCountForRoute,
		filterSectionsByQuery,
		sectionsForRoute,
		workbenchNavItem,
		type WorkbenchRoute
	} from '../nav';

	let {
		route,
		policyMap,
		search = $bindable(''),
		busy = false,
		onMutate = () => {}
	}: {
		route: WorkbenchRoute;
		policyMap?: PolicyMapResponse;
		search?: string;
		busy?: boolean;
		onMutate?: (mutation: PolicyMutation, label: string) => void | Promise<void>;
	} = $props();

	let editorValue = $state('');

	const navItem = $derived(workbenchNavItem(route));
	const sections = $derived(filterSectionsByQuery(sectionsForRoute(policyMap, route), search));
	const entryCount = $derived(entryCountForRoute(policyMap, route));
	const sectionName = $derived(sections[0]?.name ?? '');

	async function saveSection() {
		if (!sectionName || busy) return;
		let parsed: unknown;
		try {
			parsed = JSON.parse(editorValue);
		} catch {
			return;
		}
		await onMutate(
			{ type: 'upsert-section-json', section: sectionName, value: parsed },
			`${navItem?.label ?? route} — section updated`
		);
	}

	function loadSectionRaw() {
		const section = sections[0];
		if (!section?.raw) {
			editorValue = sections[0]?.entries?.length
				? JSON.stringify(sections[0].entries, null, 2)
				: '{}';
			return;
		}
		editorValue = JSON.stringify(section.raw, null, 2);
	}

	$effect(() => {
		if (sections.length > 0 && !editorValue) loadSectionRaw();
	});
</script>

<section class="flex min-h-0 flex-1 flex-col">
	<header class="border-b border-panel-strong px-5 py-4">
		<h2 class="m-0 text-lg font-extrabold text-primary">{navItem?.label ?? route}</h2>
		<p class="mt-1 mb-0 text-[0.82rem] text-secondary">{entryCount} entries · Edit & validate</p>
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
				</article>
			{/each}
		{/each}
	</div>

	<div class="editor-panel border-t border-panel-strong p-4">
		<p class="m-0 text-[0.72rem] font-extrabold tracking-wide text-label uppercase">
			Section JSON editor
		</p>
		<textarea bind:value={editorValue} class="json-editor mt-2" rows="12"></textarea>
		<div class="form-actions mt-2">
			<button type="button" class="btn" onclick={loadSectionRaw}>Reload</button>
			<button type="button" class="btn primary" disabled={busy} onclick={() => void saveSection()}
				>Save to draft</button
			>
		</div>
	</div>
</section>

<style>
	@reference "../../../app.css";
	.search-input {
		@apply min-h-[2.35rem] w-full rounded-md border border-panel-border bg-panel-input px-3 py-2 text-[0.86rem];
	}
	.entry-list {
		@apply max-h-[40%] overflow-auto p-4;
	}
	.entry-card {
		@apply mb-2 rounded-md border border-panel-border bg-panel-input p-3;
	}
	.json-editor {
		@apply w-full rounded-md border border-panel-border bg-panel-input p-2 font-mono text-[0.78rem];
	}
	.form-actions {
		@apply flex justify-end gap-2;
	}
	.btn {
		@apply rounded-md border border-panel-border bg-panel-weak px-3 py-2 text-[0.82rem] font-extrabold;
	}
	.btn.primary {
		@apply border-panel-accent bg-panel-accent text-panel-fg;
	}
</style>
