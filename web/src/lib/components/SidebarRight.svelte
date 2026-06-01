<script lang="ts">
	import type { Device } from '../api/schemas';
	import type { DeviceAggregateMeta } from '../graph/collapse-devices';
	import { isAggregateDeviceId } from '../graph/collapse-devices';
	import type { RenderEdge } from '../graph/engine';
	import DeviceDetailsPanel from './DeviceDetailsPanel.svelte';
	import ResizableSidebar from './ResizableSidebar.svelte';

	let {
		open = $bindable(true),
		selectedDevice = $bindable<Device | undefined>(undefined),
		selectedEdge = $bindable<RenderEdge | undefined>(undefined),
		devices = [],
		aggregateMeta = new Map<string, DeviceAggregateMeta>(),
		visibleEdges = [],
		colorBy = $bindable<'status' | 'tag' | 'owner' | 'os'>('status')
	}: {
		open?: boolean;
		selectedDevice?: Device;
		selectedEdge?: RenderEdge;
		devices?: Device[];
		aggregateMeta?: Map<string, DeviceAggregateMeta>;
		visibleEdges?: RenderEdge[];
		colorBy?: 'status' | 'tag' | 'owner' | 'os';
	} = $props();

	const selectedAggregate = $derived(
		selectedDevice && isAggregateDeviceId(selectedDevice.id)
			? aggregateMeta.get(selectedDevice.id)
			: undefined
	);

	const edgeSource = $derived(devices.find((device) => device.id === selectedEdge?.from));
	const activeDevice = $derived(selectedDevice ?? edgeSource);
	const deviceInitials = $derived.by(() => {
		if (!activeDevice) return '?';
		if (selectedAggregate) return String(selectedAggregate.members.length);
		return activeDevice.name ? activeDevice.name.split('.')[0].slice(0, 2).toUpperCase() : '?';
	});

	const osColors: Record<string, string> = {
		windows: '#01A6F0',
		android: '#32DE84',
		linux: '#F4BC00',
		bsd: '#B5010F',
		macOS: '#A2AAAD',
		ios: '#FFFFFF',
		tvos: '#FA6C1B'
	};

	function palette(value: string): string {
		const osColor = osColors[value];
		if (osColor) return osColor;
		const colors = ['#438aa1', '#a5663f', '#7c6fb0', '#b0892f', '#5d7f73', '#b45f74', '#5973b0'];
		let hash = 0;
		for (let i = 0; i < value.length; i += 1) {
			hash = (hash + value.charCodeAt(i) * (i + 1)) % colors.length;
		}
		return colors[hash];
	}

	const avatarColor = $derived.by((): string | undefined => {
		if (!activeDevice) return undefined;
		if (colorBy === 'status') {
			return activeDevice.online ? '#41a86f' : '#9aa7a1';
		}
		const value =
			colorBy === 'tag'
				? (activeDevice.tags[0] ?? 'untagged')
				: colorBy === 'owner'
					? activeDevice.owner
					: activeDevice.os;
		return palette(value || 'unknown');
	});
</script>

<ResizableSidebar position="right" defaultWidth={18 * 16} {open}>
	<DeviceDetailsPanel
		bind:selectedDevice
		bind:selectedEdge
		{devices}
		{aggregateMeta}
		{visibleEdges}
		bind:colorBy
	/>
	{#snippet collapsed()}
		<button class="sidebar-icon" title="Details panel" type="button" onclick={() => (open = true)}>
			<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
				<path
					d="M10 1L1 5V10C1 15.25 4.75 19.35 10 20C15.25 19.35 19 15.25 19 10V5L10 1Z"
					stroke="currentColor"
					stroke-width="1.6"
					stroke-linecap="round"
					stroke-linejoin="round"
					fill="none"
				/>
			</svg>
		</button>
		<div class="bg-border h-px w-[1.2rem]"></div>
		{#if selectedEdge}
			<button
				class="sidebar-icon"
				title="Selected access relationship"
				type="button"
				onclick={() => (open = true)}
			>
				<span class="text-[0.65rem] font-extrabold">→</span>
			</button>
		{:else if selectedDevice}
			<button
				class="sidebar-icon"
				title={selectedDevice.name}
				type="button"
				onclick={() => (open = true)}
			>
				<span
					class="mini-avatar grid h-7 w-7 shrink-0 place-items-center rounded-full text-[0.65rem] font-bold text-white"
					style:background-color={avatarColor}
					data-subnet-router={selectedDevice.subnetRouter}>{deviceInitials}</span
				>
			</button>
		{:else}
			<span class="text-[0.75rem] font-bold text-muted" title="Nothing selected">—</span>
		{/if}
	{/snippet}
</ResizableSidebar>

<style>
	@reference "../../app.css";
	.mini-avatar[data-subnet-router='true'] {
		@apply rounded-lg;
	}
	.sidebar-icon {
		@apply grid h-8 w-8 cursor-pointer place-items-center rounded-md border border-strong bg-transparent p-0 text-secondary transition-[background-color,border-color,color] duration-[140ms] ease-out hover:border-teal hover:bg-hover hover:text-primary;
	}
	.sidebar-icon svg {
		@apply h-4 w-4;
	}
</style>
