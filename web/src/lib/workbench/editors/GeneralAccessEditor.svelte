<script lang="ts">
	import type { PolicyMapResponse, PolicySection } from '../../api/schemas';
	import type { PolicyMutation } from '../../draft/types';
	import {
		buildACLRule,
		buildGrantRule,
		portPresetValues,
		splitSelectors
	} from '../../draft/types';
	import {
		canSimulateEntry,
		filterSectionsByQuery,
		sectionsForRoute,
		type WorkbenchRoute
	} from '../nav';

	let {
		policyMap,
		search = $bindable(''),
		draftHuJSON = '',
		scenarioSource = '',
		editSeed = $bindable({ sources: '', destinations: '', ports: '443' }),
		busy = false,
		onViewAs = () => {},
		onMutate = () => {}
	}: {
		policyMap?: PolicyMapResponse;
		search?: string;
		draftHuJSON?: string;
		scenarioSource?: string;
		editSeed?: { sources: string; destinations: string; ports: string };
		busy?: boolean;
		onViewAs?: (selector: string) => void;
		onMutate?: (mutation: PolicyMutation, label: string) => void | Promise<void>;
	} = $props();

	let showForm = $state(false);
	let ruleKind = $state<'acl' | 'grant'>('acl');
	let sources = $state('');
	let destinations = $state('');
	let portPreset = $state('443');
	let customPorts = $state('');
	let note = $state('');
	let showAdvanced = $state(false);
	let viaTag = $state('');
	let posture = $state('');

	const route: WorkbenchRoute = 'general-access';
	const sections = $derived(filterSectionsByQuery(sectionsForRoute(policyMap, route), search));
	const entryCount = $derived(sections.reduce((sum, section) => sum + section.count, 0));

	const previewRule = $derived.by(() => {
		const src = splitSelectors(sources);
		const dst = splitSelectors(destinations);
		const ports = portPresetValues(portPreset, customPorts);
		if (src.length === 0 || dst.length === 0 || ports.length === 0) return null;
		if (ruleKind === 'grant') return buildGrantRule({ sources: src, destinations: dst, ports });
		return buildACLRule({ sources: src, destinations: dst, ports, protocol: 'tcp' });
	});

	$effect(() => {
		if (editSeed.sources || editSeed.destinations) {
			sources = editSeed.sources || scenarioSource || sources;
			destinations = editSeed.destinations || destinations;
			if (editSeed.ports) portPreset = editSeed.ports;
			showForm = true;
		} else if (scenarioSource && !sources && showForm) {
			sources = scenarioSource;
		}
	});

	function ruleColumns(entry: NonNullable<PolicySection['entries']>[number], sectionName: string) {
		const value = entry.value as Record<string, unknown> | undefined;
		if (!value || typeof value !== 'object') {
			return {
				sources: entry.summary?.split(' -> ')[0] ?? entry.label,
				destinations: entry.summary?.split(' -> ')[1]?.split(' (')[0] ?? '—',
				ports: '—'
			};
		}
		const src = Array.isArray(value.src) ? (value.src as string[]).join(', ') : '*';
		const dst = Array.isArray(value.dst) ? (value.dst as string[]).join(', ') : '*';
		const ip = Array.isArray(value.ip) ? (value.ip as string[]).join(', ') : '';
		const proto = typeof value.proto === 'string' ? value.proto : '';
		const ports = ip || proto || '*';
		if (sectionName === 'grants') return { sources: src, destinations: dst, ports: ip || '*' };
		return { sources: src, destinations: dst, ports };
	}

	function openAddForm() {
		showForm = true;
		if (scenarioSource && !sources) sources = scenarioSource;
	}

	function cancelForm() {
		showForm = false;
		note = '';
		showAdvanced = false;
		viaTag = '';
		posture = '';
	}

	async function saveToDraft() {
		if (!previewRule || busy) return;
		const src = splitSelectors(sources);
		const dst = splitSelectors(destinations);
		const ports = portPresetValues(portPreset, customPorts);
		if (ruleKind === 'grant') {
			await onMutate(
				{
					type: 'append-grant',
					grant: buildGrantRule({ sources: src, destinations: dst, ports })
				},
				`General access rules — grant added`
			);
		} else {
			await onMutate(
				{ type: 'append-acl', rule: buildACLRule({ sources: src, destinations: dst, ports }) },
				`General access rules — ACL rule added`
			);
		}
		cancelForm();
	}

	async function removeRule(sectionName: string, index: number) {
		if (sectionName !== 'acls' || busy) return;
		await onMutate(
			{ type: 'remove-acl', index },
			`General access rules — removed ACL #${index + 1}`
		);
	}
</script>

<section class="flex min-h-0 flex-1 flex-col">
	<header class="border-b border-panel-strong px-5 py-4">
		<div class="flex items-start justify-between gap-3">
			<div>
				<h2 class="m-0 text-lg font-extrabold text-primary">General access rules</h2>
				<p class="mt-1 mb-0 text-[0.82rem] text-secondary">{entryCount} entries</p>
			</div>
			<button type="button" class="add-button" onclick={openAddForm}>+ Add rule</button>
		</div>
		<div class="mt-3">
			<input
				bind:value={search}
				placeholder="Search by user, group, device, tag, port, or IP address"
				class="search-input"
			/>
		</div>
	</header>

	<div class="min-h-0 flex-1 overflow-auto">
		{#if showForm}
			<form
				class="form-panel border-b border-panel-strong bg-panel-weak p-4"
				onsubmit={(event) => {
					event.preventDefault();
					void saveToDraft();
				}}
			>
				<div class="flex items-center justify-between gap-2">
					<h3 class="m-0 text-[0.9rem] font-extrabold text-primary">Add rule</h3>
					<div class="flex gap-1">
						<button
							type="button"
							class="kind-button"
							data-active={ruleKind === 'acl'}
							onclick={() => (ruleKind = 'acl')}>ACL</button
						>
						<button
							type="button"
							class="kind-button"
							data-active={ruleKind === 'grant'}
							onclick={() => (ruleKind = 'grant')}>Grant</button
						>
					</div>
				</div>
				<div class="form-grid">
					<label class="field"
						><span>Sources</span><input
							bind:value={sources}
							placeholder="alice@example.com, group:eng"
						/></label
					>
					<label class="field"
						><span>Destinations</span><input
							bind:value={destinations}
							placeholder="tag:web, db-host"
						/></label
					>
					<label class="field">
						<span>Port and protocol</span>
						<select bind:value={portPreset}>
							<option value="443">HTTPS 443</option>
							<option value="80,443">HTTP/S 80,443</option>
							<option value="22">SSH 22</option>
							<option value="*">All ports</option>
							<option value="custom">Custom</option>
						</select>
					</label>
					{#if portPreset === 'custom'}
						<label class="field"
							><span>Custom ports</span><input
								bind:value={customPorts}
								placeholder="8080,8443"
							/></label
						>
					{/if}
					<label class="field span-2"
						><span>Note</span><input
							bind:value={note}
							placeholder="Optional note (stored as comment in a future pass)"
							disabled
						/></label
					>
				</div>
				<details class="advanced" bind:open={showAdvanced}>
					<summary>Advanced options</summary>
					<div class="form-grid mt-2">
						<label class="field"
							><span>Device posture</span><input
								bind:value={posture}
								placeholder="posture:..."
								disabled
								title="Graph eval for posture rules coming soon"
							/></label
						>
						<label class="field"
							><span>Via</span><input
								bind:value={viaTag}
								placeholder="tag:router"
								disabled
							/></label
						>
					</div>
					<p class="hint">
						Posture and via fields are shown for Tailscale parity; serialization lands in a
						follow-up.
					</p>
				</details>
				<div class="preview-grid">
					<div>
						<p class="preview-label">Live JSON preview</p>
						<pre class="json-preview">{previewRule
								? JSON.stringify(previewRule, null, 2)
								: 'Fill in sources, destinations, and ports.'}</pre>
					</div>
				</div>
				<div class="form-actions">
					<button type="button" class="btn" onclick={cancelForm}>Cancel</button>
					<button type="submit" class="btn primary" disabled={busy || !previewRule}
						>Save to draft</button
					>
				</div>
			</form>
		{/if}

		<div class="table-wrap">
			<table class="rules-table">
				<thead>
					<tr>
						<th>Sources</th>
						<th>can access destinations</th>
						<th>on port and protocol</th>
						<th aria-label="Actions"></th>
					</tr>
				</thead>
				<tbody>
					{#each sections as section (section.name)}
						{#each section.entries ?? [] as entry, index (entry.label)}
							{@const cols = ruleColumns(entry, section.name)}
							<tr>
								<td
									><div class="entry-label">{entry.label}</div>
									<div class="entry-value">{cols.sources}</div></td
								>
								<td><div class="entry-value">{cols.destinations}</div></td>
								<td><div class="entry-value">{cols.ports}</div></td>
								<td class="actions">
									{#if canSimulateEntry(section.name, entry.label)}
										<button type="button" class="link-button" onclick={() => onViewAs(entry.label)}
											>View as</button
										>
									{/if}
									{#if section.name === 'acls' && draftHuJSON}
										<button
											type="button"
											class="link-button danger"
											onclick={() => void removeRule(section.name, index)}>Remove</button
										>
									{/if}
								</td>
							</tr>
						{/each}
					{/each}
				</tbody>
			</table>
		</div>
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
		@apply shrink-0;
	}
	.form-grid {
		@apply mt-3 grid grid-cols-2 gap-3;
	}
	.field {
		@apply flex flex-col gap-1 text-[0.75rem] font-extrabold text-label;
	}
	.field input,
	.field select {
		@apply min-h-[2.2rem] rounded-md border border-panel-border bg-panel-input px-2 py-2 text-[0.84rem] font-normal text-primary outline-none focus:border-teal;
	}
	.span-2 {
		@apply col-span-2;
	}
	.kind-button {
		@apply rounded-md border border-panel-border bg-panel-weak px-2 py-1 text-[0.72rem] font-extrabold text-secondary;
	}
	.kind-button[data-active='true'] {
		@apply border-teal bg-hover text-primary;
	}
	.advanced summary {
		@apply mt-3 cursor-pointer text-[0.78rem] font-extrabold text-teal;
	}
	.hint {
		@apply mt-2 text-[0.74rem] text-code;
	}
	.preview-label {
		@apply m-0 text-[0.72rem] font-extrabold tracking-wide text-label uppercase;
	}
	.json-preview {
		@apply mt-2 max-h-40 overflow-auto rounded bg-[oklch(0.96_0.009_158)] p-2 font-mono text-[0.76rem] leading-[1.45] text-[#1c2c26];
	}
	.form-actions {
		@apply mt-3 flex justify-end gap-2;
	}
	.btn {
		@apply rounded-md border border-panel-border bg-panel-weak px-3 py-2 text-[0.82rem] font-extrabold text-primary disabled:opacity-50;
	}
	.btn.primary {
		@apply border-panel-accent bg-panel-accent text-panel-fg;
	}
	.table-wrap {
		@apply min-w-0 overflow-auto px-3 py-3;
	}
	.rules-table {
		@apply w-full min-w-[36rem] border-collapse text-left text-[0.84rem];
	}
	.rules-table th {
		@apply border-b border-panel-strong px-3 py-2 text-[0.72rem] font-extrabold tracking-wide text-label uppercase;
	}
	.rules-table td {
		@apply border-b border-panel-border px-3 py-3 align-top text-primary;
	}
	.entry-label {
		@apply mb-1 text-[0.72rem] font-extrabold text-secondary uppercase;
	}
	.entry-value {
		@apply text-[0.84rem] wrap-anywhere text-code;
	}
	.actions {
		@apply w-[8rem];
	}
	.link-button {
		@apply mr-1 cursor-pointer rounded border border-panel-border bg-panel-weak px-2 py-1 text-[0.72rem] font-extrabold text-teal hover:bg-hover;
	}
	.link-button.danger {
		@apply text-danger;
	}
</style>
