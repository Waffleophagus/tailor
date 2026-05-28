<script lang="ts">
	import type { Device, PolicyMapResponse } from '../api/schemas';
	import {
		buildPerspectiveCatalog,
		filterCatalog,
		groupedCatalog,
		kindLabel,
		loadRecentPerspectives,
		parsePerspectiveInput,
		type PerspectiveKind,
		type PerspectiveValidation,
		validatePerspective
	} from '../perspective/catalog';

	let {
		id = 'policy-perspective',
		value = $bindable(''),
		devices = [],
		policyMap,
		hasPerspectivePreview = false,
		reachableCount = 0,
		busy = false,
		onSimulate = () => {}
	}: {
		id?: string;
		value?: string;
		devices?: Device[];
		policyMap?: PolicyMapResponse;
		hasPerspectivePreview?: boolean;
		reachableCount?: number;
		busy?: boolean;
		onSimulate?: () => void;
	} = $props();

	let open = $state(false);
	let highlightIndex = $state(0);
	let inputEl: HTMLInputElement | undefined;

	const catalog = $derived(buildPerspectiveCatalog(devices, policyMap));
	const validation = $derived(validatePerspective(value, devices, policyMap));
	const parsed = $derived(parsePerspectiveInput(value));

	const filtered = $derived(filterCatalog(catalog, parsed.selector || value));
	const grouped = $derived(groupedCatalog(filtered));
	const recent = $derived(loadRecentPerspectives());

	const flatOptions = $derived([
		...(value.trim()
			? []
			: recent.flatMap((selector) => catalog.filter((o) => o.selector === selector))),
		...grouped.user,
		...grouped.group,
		...grouped.tag,
		...grouped.autogroup
	]);

	const statusMessage = $derived(statusCopy(validation, hasPerspectivePreview, reachableCount));

	function statusCopy(
		v: PerspectiveValidation,
		active: boolean,
		reachable: number
	): { tone: 'neutral' | 'ok' | 'warn' | 'error'; text: string } {
		if (v.status === 'empty') {
			return { tone: 'neutral', text: 'Whole tailnet — saved effective access for all devices.' };
		}
		if (v.status === 'invalid') {
			return { tone: 'error', text: v.message };
		}
		const kind = kindLabel(v.kind);
		const warn = v.warnings[0];
		if (active) {
			const reach =
				reachable > 0 ? ` · ${reachable} reachable target${reachable === 1 ? '' : 's'}` : '';
			return {
				tone: 'ok',
				text: warn
					? `${warn} Simulated as ${kind} · ${v.deviceCount} device${v.deviceCount === 1 ? '' : 's'}${reach}.`
					: `Simulated as ${kind} · ${v.deviceCount} device${v.deviceCount === 1 ? '' : 's'}${reach}.`
			};
		}
		return {
			tone: 'warn',
			text: warn
				? `${warn} ${kind} · ${v.deviceCount} device${v.deviceCount === 1 ? '' : 's'} · Simulate to preview.`
				: `${kind} · ${v.deviceCount} device${v.deviceCount === 1 ? '' : 's'} · Simulate to preview reachability.`
		};
	}

	function pick(option: { selector: string }) {
		value = option.selector;
		open = false;
		highlightIndex = 0;
		onSimulate();
	}

	function commitSimulate() {
		if (validation.status === 'valid') {
			value = validation.selector;
			open = false;
			onSimulate();
		}
	}

	function onInputKeydown(event: KeyboardEvent) {
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			open = true;
			highlightIndex = Math.min(highlightIndex + 1, Math.max(flatOptions.length - 1, 0));
		} else if (event.key === 'ArrowUp') {
			event.preventDefault();
			highlightIndex = Math.max(highlightIndex - 1, 0);
		} else if (event.key === 'Enter') {
			event.preventDefault();
			if (open && flatOptions[highlightIndex]) {
				pick(flatOptions[highlightIndex]);
			} else {
				commitSimulate();
			}
		} else if (event.key === 'Escape') {
			open = false;
		}
	}

	function pillClass(kind: PerspectiveKind) {
		switch (kind) {
			case 'user':
				return 'pill-user';
			case 'group':
				return 'pill-group';
			case 'tag':
				return 'pill-tag';
			case 'autogroup':
				return 'pill-autogroup';
		}
	}
</script>

<div class="selector-root">
	<div class="input-wrap">
		<input
			{id}
			bind:this={inputEl}
			type="text"
			bind:value
			placeholder="user@example.com, group:ops, tag:ci, autogroup:member"
			class="selector-input"
			role="combobox"
			aria-expanded={open}
			aria-autocomplete="list"
			aria-controls="perspective-options"
			onfocus={() => (open = true)}
			onblur={() => setTimeout(() => (open = false), 150)}
			oninput={() => {
				open = true;
				highlightIndex = 0;
			}}
			onkeydown={onInputKeydown}
			disabled={busy}
		/>
		{#if validation.status === 'valid'}
			<span class="type-pill {pillClass(validation.kind)}">{kindLabel(validation.kind)}</span>
		{/if}
	</div>

	{#if open && flatOptions.length > 0}
		<ul id="perspective-options" class="options-panel" role="listbox">
			{#if !value.trim() && recent.length > 0}
				<li class="section-label" role="presentation">Recent</li>
			{/if}
			{#each flatOptions as option, index (option.selector)}
				<li role="presentation">
					<button
						type="button"
						role="option"
						aria-selected={index === highlightIndex}
						class="option-row"
						class:highlighted={index === highlightIndex}
						onmousedown={(e) => e.preventDefault()}
						onclick={() => pick(option)}
					>
						<span class="pill {pillClass(option.kind)}">{kindLabel(option.kind)}</span>
						<span class="option-label">{option.selector}</span>
						<span class="option-meta"
							>{option.deviceCount} device{option.deviceCount === 1 ? '' : 's'}</span
						>
					</button>
				</li>
			{/each}
		</ul>
	{/if}

	<p class="status" data-tone={statusMessage.tone}>{statusMessage.text}</p>
</div>

<style>
	@reference "../../app.css";

	.selector-root {
		@apply relative min-w-0;
	}
	.input-wrap {
		@apply relative flex items-center;
	}
	.selector-input {
		@apply min-h-[2.1rem] w-full rounded-md border border-panel-border bg-panel-input py-[0.4rem] pr-[5.5rem] pl-2 text-[0.83rem] text-primary transition-[border-color,box-shadow] duration-[140ms] ease-out outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)];
	}
	.type-pill {
		@apply pointer-events-none absolute right-2 rounded px-1.5 py-0.5 text-[0.65rem] font-extrabold tracking-wide uppercase;
	}
	.pill-user {
		@apply bg-teal/15 text-teal;
	}
	.pill-group {
		@apply bg-violet-500/15 text-violet-700;
	}
	.pill-tag {
		@apply bg-amber-500/15 text-amber-800;
	}
	.pill-autogroup {
		@apply bg-slate-500/15 text-slate-700;
	}
	.options-panel {
		@apply absolute top-[calc(100%+0.25rem)] right-0 left-0 z-10 m-0 max-h-[14rem] list-none overflow-y-auto rounded-md border border-panel-border bg-panel-bg p-1 shadow-[0_10px_26px_rgb(23_33_38/12%)];
	}
	.section-label {
		@apply px-2 py-1 text-[0.65rem] font-extrabold tracking-wider text-label uppercase;
	}
	.option-row {
		@apply flex w-full cursor-pointer items-center gap-2 rounded border-0 bg-transparent px-2 py-1.5 text-left text-[0.8rem];
	}
	.option-row.highlighted,
	.option-row:hover {
		@apply bg-hover;
	}
	.pill {
		@apply shrink-0 rounded px-1 py-0.5 text-[0.62rem] font-extrabold tracking-wide uppercase;
	}
	.option-label {
		@apply min-w-0 flex-1 truncate font-semibold text-primary;
	}
	.option-meta {
		@apply shrink-0 text-[0.72rem] font-bold text-secondary;
	}
	.status {
		@apply m-0 mt-1 min-w-0 text-[0.74rem] leading-snug font-bold text-secondary;
	}
	.status[data-tone='ok'] {
		@apply text-primary;
	}
	.status[data-tone='error'] {
		@apply text-red-700;
	}
	.status[data-tone='warn'] {
		@apply text-secondary;
	}
</style>
