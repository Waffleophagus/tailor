<script lang="ts">
	import { palette } from './avatar-color';
	import { UNTAGGED_DEVICE_COLOR } from '../tag-color';

	let {
		colorBy = $bindable<'status' | 'tag' | 'owner' | 'os'>('status'),
		authenticated = false,
		graphMode = $bindable<'focused' | 'all'>('all'),
		tagOptions = [] as string[],
		tagColorMap = new Map<string, string>(),
		hasUntaggedDevices = false,
		ownerOptions = [] as string[],
		osOptions = [] as string[],
		embedded = false
	}: {
		colorBy?: 'status' | 'tag' | 'owner' | 'os';
		authenticated?: boolean;
		graphMode?: 'focused' | 'all';
		tagOptions?: string[];
		tagColorMap?: ReadonlyMap<string, string>;
		hasUntaggedDevices?: boolean;
		ownerOptions?: string[];
		osOptions?: string[];
		embedded?: boolean;
	} = $props();

	interface ColorEntry {
		color: string;
		label: string;
	}

	const nodeEntries = $derived.by((): ColorEntry[] => {
		if (colorBy === 'status') {
			return [
				{ color: '#41a86f', label: 'Online' },
				{ color: '#9aa7a1', label: 'Offline' }
			];
		}
		if (colorBy === 'tag') {
			const sortedTags = [...tagOptions].sort((a, b) => a.localeCompare(b));
			const entries: ColorEntry[] = [];
			if (hasUntaggedDevices) {
				entries.push({ color: UNTAGGED_DEVICE_COLOR, label: 'No tag' });
			}
			for (const tag of sortedTags) {
				const color = tagColorMap.get(tag);
				if (!color) continue;
				entries.push({ color, label: tag });
			}
			return entries;
		}
		const options = colorBy === 'owner' ? ownerOptions : osOptions;
		const maxVisible = 8;
		const visible = options.slice(0, maxVisible);
		return visible.map((value) => ({
			color: palette(value || 'unknown'),
			label: value || 'unknown'
		}));
	});

	const nodeLegendTitle = $derived.by((): string => {
		if (colorBy === 'status') return 'Status';
		if (colorBy === 'tag') return 'Tag';
		if (colorBy === 'owner') return 'Owner';
		return 'OS';
	});

	const lineTitle = $derived.by((): string => {
		if (!authenticated) return 'Inferred relationships';
		if (graphMode === 'focused') return `ACL focus\u00a0\u2014\u00a0focused`;
		return 'ACL access scope';
	});

	const tagLegendCount = $derived(tagOptions.length);
</script>

<div
	class={embedded
		? 'text-[0.675rem] font-bold text-secondary'
		: 'pointer-events-auto absolute bottom-3 left-3 z-[3] hidden max-h-[calc(100%-1.5rem)] w-48 overflow-y-auto rounded-lg border border-graph-border bg-legend-bg/95 p-2 text-[0.675rem] font-bold text-secondary shadow-[0_8px_22px_rgb(23_33_38/8%)] md:block'}
	role="region"
	aria-label="Graph legend"
>
	<div
		class="border-base-light mb-2 border-b pb-1 text-[0.6rem] font-extrabold tracking-widest text-legend-title uppercase"
	>
		{lineTitle}
	</div>

	{#if !authenticated}
		<div class="flex flex-col gap-1">
			<div class="legend-row">
				<span
					class="mt-px inline-block h-0 w-5 min-w-5 rounded-[0.0625rem] border-t-2 border-solid border-[#5d7f73]"
				></span>
				<span>Owner</span>
			</div>
			<div class="legend-row">
				<span
					class="mt-px inline-block h-0 w-5 min-w-5 rounded-[0.0625rem] border-t-[1.7px] border-dashed border-[#7c6fb0]"
				></span>
				<span>Tag</span>
			</div>
			<div class="legend-row">
				<span
					class="mt-px inline-block h-0 w-5 min-w-5 rounded-[0.0625rem] border-t-[1.8px] border-dotted border-[#a5663f]"
				></span>
				<span>Subnet</span>
			</div>
		</div>
	{:else}
		<div class="flex flex-col gap-1">
			<div class="legend-row">
				<span
					class="mt-px inline-block h-0 w-5 min-w-5 rounded-[0.0625rem] border-t-[2.2px] border-solid border-[#438aa1]"
				></span>
				<span>ACL (generic)</span>
			</div>
			<div class="legend-row">
				<span
					class="mt-px inline-block h-0 w-5 min-w-5 rounded-[0.0625rem] border-t-[2.8px] border-solid border-[#2f9f68]"
				></span>
				<span>SSH (port 22)</span>
			</div>
			<div class="legend-row">
				<span
					class="mt-px inline-block h-0 w-5 min-w-5 rounded-[0.0625rem] border-t-[2.4px] border-solid border-[#438aa1]"
				></span>
				<span>HTTP/S (80, 443)</span>
			</div>
			<div class="legend-row">
				<span
					class="mt-px inline-block h-0 w-5 min-w-5 rounded-[0.0625rem] border-t-[3.1px] border-solid border-[#b0892f]"
				></span>
				<span>Broad (all ports)</span>
			</div>
			<div class="legend-row">
				<span
					class="mt-px inline-block h-0 w-5 min-w-5 rounded-[0.0625rem] border-t-[2.3px] border-dashed border-[#7c6fb0]"
				></span>
				<span>Limited / Custom</span>
			</div>
		</div>
	{/if}

	<div class="my-[0.35rem] h-px bg-light"></div>

	<div class="flex flex-col gap-1">
		<div class="mb-0.5 text-[0.6rem] font-extrabold tracking-widest text-legend-title uppercase">
			{nodeLegendTitle}
		</div>
		{#each nodeEntries as entry (entry.label)}
			<div class="legend-row">
				<span
					class="inline-block h-2 w-2 min-w-2 rounded-full"
					style="background-color: {entry.color}"
				></span>
				<span title={entry.label}>{entry.label}</span>
			</div>
		{/each}
		{#if colorBy === 'tag' && tagLegendCount > 10}
			<div class="mt-1 text-[0.55rem] font-semibold tracking-wide text-muted uppercase">
				{tagLegendCount} tags
			</div>
		{/if}
	</div>
</div>

<style>
	@reference "../../app.css";
	.legend-row {
		@apply flex items-center gap-[0.4rem] overflow-hidden leading-[1.2];
	}

	.legend-row > span:last-child {
		@apply min-w-0 overflow-hidden text-ellipsis whitespace-nowrap;
	}
</style>
