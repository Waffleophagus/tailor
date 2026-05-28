<script lang="ts">
	import type { PolicyMapResponse } from '../../api/schemas';
	import type { PolicyMutation } from '../../draft/types';
	import { splitSelectors } from '../../draft/types';
	import {
		entryCountForRoute,
		filterSectionsByQuery,
		sectionsForRoute,
		workbenchNavItem,
		canSimulateEntry,
		type WorkbenchRoute
	} from '../nav';

	let {
		route,
		policyMap,
		search = $bindable(''),
		busy = false,
		onViewAs = () => {},
		onMutate = () => {}
	}: {
		route: WorkbenchRoute;
		policyMap?: PolicyMapResponse;
		search?: string;
		busy?: boolean;
		onViewAs?: (selector: string) => void;
		onMutate?: (mutation: PolicyMutation, label: string) => void | Promise<void>;
	} = $props();

	let showForm = $state(false);
	let key = $state('');
	let values = $state('');
	let hostTarget = $state('');

	const navItem = $derived(workbenchNavItem(route));
	const sections = $derived(filterSectionsByQuery(sectionsForRoute(policyMap, route), search));
	const entryCount = $derived(entryCountForRoute(policyMap, route));
	const isHost = $derived(route === 'hosts');
	const valueLabel = $derived(
		route === 'groups'
			? 'Members'
			: route === 'tags'
				? 'Owners'
				: route === 'ip-sets'
					? 'Targets'
					: isHost
						? 'IP / CIDR'
						: 'Targets'
	);

	async function saveDefinition() {
		if (busy || !key.trim()) return;
		const label = navItem?.label ?? route;
		if (route === 'groups') {
			await onMutate(
				{ type: 'upsert-group', key: key.trim(), members: splitSelectors(values) },
				`${label} — updated ${key.trim()}`
			);
		} else if (route === 'tags') {
			const tag = key.trim().startsWith('tag:') ? key.trim() : `tag:${key.trim()}`;
			await onMutate(
				{ type: 'upsert-tag', key: tag, owners: splitSelectors(values) },
				`${label} — updated ${tag}`
			);
		} else if (route === 'hosts') {
			await onMutate(
				{ type: 'upsert-host', key: key.trim(), host: hostTarget.trim() },
				`${label} — updated ${key.trim()}`
			);
		} else if (route === 'ip-sets') {
			await onMutate(
				{ type: 'upsert-ipset', key: key.trim(), ipSet: splitSelectors(values) },
				`${label} — updated ${key.trim()}`
			);
		}
		showForm = false;
		key = '';
		values = '';
		hostTarget = '';
	}

	function editEntry(entryKey: string, entryValue: unknown) {
		key = entryKey;
		if (isHost && typeof entryValue === 'string') hostTarget = entryValue;
		else if (Array.isArray(entryValue)) values = entryValue.join(', ');
		else if (entryValue !== undefined) values = JSON.stringify(entryValue);
		showForm = true;
	}
</script>

<section class="flex min-h-0 flex-1 flex-col">
	<header class="border-b border-panel-strong px-5 py-4">
		<div class="flex items-start justify-between gap-3">
			<div>
				<h2 class="m-0 text-lg font-extrabold text-primary">{navItem?.label ?? route}</h2>
				<p class="mt-1 mb-0 text-[0.82rem] text-secondary">{entryCount} entries</p>
			</div>
			<button type="button" class="add-button" onclick={() => (showForm = true)}>+ Add entry</button
			>
		</div>
		<div class="mt-3">
			<input
				bind:value={search}
				placeholder={navItem?.searchPlaceholder ?? 'Search'}
				class="search-input"
			/>
		</div>
	</header>

	{#if showForm}
		<form
			class="form-panel border-b border-panel-strong bg-panel-weak p-4"
			onsubmit={(e) => {
				e.preventDefault();
				void saveDefinition();
			}}
		>
			<label class="field"
				><span>Name</span><input
					bind:value={key}
					placeholder="group:eng, tag:server, db-host"
				/></label
			>
			{#if isHost}
				<label class="field"
					><span>{valueLabel}</span><input
						bind:value={hostTarget}
						placeholder="100.64.0.10 or 10.0.0.0/24"
					/></label
				>
			{:else}
				<label class="field"
					><span>{valueLabel}</span><textarea
						bind:value={values}
						rows="3"
						placeholder="Comma or newline separated"
					></textarea></label
				>
			{/if}
			<div class="form-actions">
				<button type="button" class="btn" onclick={() => (showForm = false)}>Cancel</button>
				<button type="submit" class="btn primary" disabled={busy}>Save to draft</button>
			</div>
		</form>
	{/if}

	<div class="entry-list">
		{#each sections as section (section.name)}
			{#each section.entries ?? [] as entry (entry.label)}
				<article class="entry-card">
					<div class="entry-header">
						<div class="min-w-0">
							<strong class="entry-title">{entry.label}</strong>
							{#if entry.summary}<p class="entry-summary">{entry.summary}</p>{/if}
						</div>
						<div class="actions">
							<button
								type="button"
								class="link-button"
								onclick={() => editEntry(entry.label, entry.value)}>Edit</button
							>
							{#if canSimulateEntry(section.name, entry.label)}
								<button type="button" class="link-button" onclick={() => onViewAs(entry.label)}
									>View as</button
								>
							{/if}
						</div>
					</div>
				</article>
			{/each}
		{/each}
	</div>
</section>

<style>
	@reference "../../../app.css";
	.search-input {
		@apply min-h-[2.35rem] w-full rounded-md border border-panel-border bg-panel-input px-3 py-2 text-[0.86rem] text-primary outline-none focus:border-teal;
	}
	.add-button {
		@apply shrink-0 rounded-md border border-panel-accent bg-panel-accent px-3 py-2 text-[0.82rem] font-extrabold text-panel-fg;
	}
	.form-panel {
		@apply grid gap-3;
	}
	.field {
		@apply flex flex-col gap-1 text-[0.75rem] font-extrabold text-label;
	}
	.field input,
	.field textarea {
		@apply rounded-md border border-panel-border bg-panel-input px-2 py-2 text-[0.84rem] font-normal text-primary outline-none focus:border-teal;
	}
	.form-actions {
		@apply flex justify-end gap-2;
	}
	.btn {
		@apply rounded-md border border-panel-border bg-panel-weak px-3 py-2 text-[0.82rem] font-extrabold text-primary;
	}
	.btn.primary {
		@apply border-panel-accent bg-panel-accent text-panel-fg;
	}
	.entry-list {
		@apply grid gap-3 overflow-auto p-4;
	}
	.entry-card {
		@apply rounded-md border border-panel-border bg-panel-input p-3;
	}
	.entry-header {
		@apply flex items-start justify-between gap-2;
	}
	.entry-title {
		@apply block text-[0.86rem] text-primary;
	}
	.entry-summary {
		@apply mt-1 mb-0 text-[0.82rem] text-code;
	}
	.link-button {
		@apply cursor-pointer rounded border border-panel-border bg-panel-weak px-2 py-1 text-[0.72rem] font-extrabold text-teal;
	}
	.actions {
		@apply flex shrink-0 gap-1;
	}
</style>
