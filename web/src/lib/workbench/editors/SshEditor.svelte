<script lang="ts">
	import type { PolicyMapResponse } from '../../api/schemas';
	import type { PolicyMutation } from '../../draft/types';
	import { splitSelectors } from '../../draft/types';
	import { entryCountForRoute, filterSectionsByQuery, sectionsForRoute } from '../nav';

	let {
		policyMap,
		search = $bindable(''),
		scenarioSource = '',
		busy = false,
		onMutate = () => {}
	}: {
		policyMap?: PolicyMapResponse;
		search?: string;
		scenarioSource?: string;
		busy?: boolean;
		onMutate?: (mutation: PolicyMutation, label: string) => void | Promise<void>;
	} = $props();

	let showForm = $state(false);
	let sources = $state('');
	let destinations = $state('');
	let users = $state('');
	let action = $state('accept');
	let check = $state('');

	const sections = $derived(filterSectionsByQuery(sectionsForRoute(policyMap, 'ssh'), search));
	const entryCount = $derived(entryCountForRoute(policyMap, 'ssh'));

	$effect(() => {
		if (scenarioSource && showForm && !sources) sources = scenarioSource;
	});

	const preview = $derived({
		action,
		src: splitSelectors(sources),
		dst: splitSelectors(destinations),
		users: splitSelectors(users),
		...(check ? { check } : {})
	});

	async function saveRule() {
		if (busy || preview.src.length === 0 || preview.dst.length === 0) return;
		await onMutate({ type: 'append-ssh', value: preview }, 'Tailscale SSH — rule added');
		showForm = false;
	}
</script>

<section class="flex min-h-0 flex-1 flex-col">
	<header class="border-b border-panel-strong px-5 py-4">
		<div class="flex items-start justify-between gap-3">
			<div>
				<h2 class="m-0 text-lg font-extrabold text-primary">Tailscale SSH</h2>
				<p class="mt-1 mb-0 text-[0.82rem] text-secondary">
					{entryCount} rules · Graph-partial preview
				</p>
				<p class="mt-1 mb-0 text-[0.76rem] text-code">
					SSH permission is separate from network ACL reachability on the graph.
				</p>
			</div>
			<button type="button" class="add-button" onclick={() => (showForm = true)}>+ Add rule</button>
		</div>
		<input bind:value={search} class="search-input mt-3" placeholder="Search SSH rules" />
	</header>

	{#if showForm}
		<form
			class="form-panel p-4"
			onsubmit={(e) => {
				e.preventDefault();
				void saveRule();
			}}
		>
			<div class="form-grid">
				<label class="field"><span>Sources</span><input bind:value={sources} /></label>
				<label class="field"><span>Destinations</span><input bind:value={destinations} /></label>
				<label class="field"
					><span>SSH users</span><input bind:value={users} placeholder="root, ubuntu" /></label
				>
				<label class="field"
					><span>Action</span><select bind:value={action}
						><option value="accept">accept</option><option value="check">check</option></select
					></label
				>
				<label class="field"
					><span>Check mode</span><input bind:value={check} placeholder="optional" /></label
				>
			</div>
			<pre class="json-preview">{JSON.stringify(preview, null, 2)}</pre>
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
					<strong>{entry.label}</strong>
					{#if entry.summary}<p>{entry.summary}</p>{/if}
					{#if entry.value}<pre class="json-preview">{JSON.stringify(
								entry.value,
								null,
								2
							)}</pre>{/if}
				</article>
			{/each}
		{/each}
	</div>
</section>

<style>
	@reference "../../../app.css";
	.search-input {
		@apply min-h-[2.35rem] w-full rounded-md border border-panel-border bg-panel-input px-3 py-2 text-[0.86rem];
	}
	.add-button {
		@apply rounded-md border border-panel-accent bg-panel-accent px-3 py-2 text-[0.82rem] font-extrabold text-panel-fg;
	}
	.form-panel {
		@apply border-b border-panel-strong bg-panel-weak;
	}
	.form-grid {
		@apply grid grid-cols-2 gap-3;
	}
	.field {
		@apply flex flex-col gap-1 text-[0.75rem] font-extrabold text-label;
	}
	.field input,
	.field select {
		@apply min-h-[2.2rem] rounded-md border border-panel-border bg-panel-input px-2;
	}
	.json-preview {
		@apply mt-2 overflow-auto rounded bg-[oklch(0.96_0.009_158)] p-2 font-mono text-[0.76rem];
	}
	.form-actions {
		@apply mt-3 flex justify-end gap-2;
	}
	.btn {
		@apply rounded-md border border-panel-border bg-panel-weak px-3 py-2 text-[0.82rem] font-extrabold;
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
</style>
