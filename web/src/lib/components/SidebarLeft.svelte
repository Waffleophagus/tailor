<script lang="ts">
	import type { Device } from '../api/schemas';
	import DeviceFiltersPanel from './DeviceFiltersPanel.svelte';
	import ResizableSidebar from './ResizableSidebar.svelte';

	let {
		open = $bindable(true),
		devices = [],
		listDevices = [],
		selectedDevice = $bindable<Device | undefined>(undefined),
		showLabels = $bindable(true),
		showOffline = $bindable(true),
		showSubnetRouters = $bindable(true),
		collapseTaggedFleets = $bindable(true),
		showTailnet = $bindable(false),
		selectedTag = $bindable('all'),
		selectedOwner = $bindable('all'),
		selectedOS = $bindable('all'),
		colorBy = $bindable<'status' | 'tag' | 'owner' | 'os'>('status'),
		tagOptions = [],
		ownerOptions = [],
		osOptions = [],
		listOnlineCount = 0,
		chooseDevice
	}: {
		open?: boolean;
		devices: Device[];
		listDevices: Device[];
		selectedDevice?: Device;
		showLabels?: boolean;
		showOffline?: boolean;
		showSubnetRouters?: boolean;
		collapseTaggedFleets?: boolean;
		showTailnet?: boolean;
		selectedTag?: string;
		selectedOwner?: string;
		selectedOS?: string;
		colorBy?: 'status' | 'tag' | 'owner' | 'os';
		tagOptions: string[];
		ownerOptions: string[];
		osOptions: string[];
		listOnlineCount?: number;
		chooseDevice: (device: Device) => void;
	} = $props();
</script>

<ResizableSidebar position="left" defaultWidth={16 * 16} {open}>
	<DeviceFiltersPanel
		{devices}
		{listDevices}
		bind:selectedDevice
		bind:showLabels
		bind:showOffline
		bind:showSubnetRouters
		bind:collapseTaggedFleets
		bind:showTailnet
		bind:selectedTag
		bind:selectedOwner
		bind:selectedOS
		bind:colorBy
		{tagOptions}
		{ownerOptions}
		{osOptions}
		{listOnlineCount}
		{chooseDevice}
	/>
	{#snippet collapsed()}
		<button class="sidebar-icon" title="Devices panel" type="button" onclick={() => (open = true)}>
			<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
				<path d="M10 2.5L2.5 7.5V17.5H8.75V12.5H11.25V17.5H17.5V7.5L10 2.5Z" fill="currentColor" />
			</svg>
		</button>
		<div class="bg-border h-px w-[1.2rem]"></div>
		<div class="mt-[0.2rem] flex flex-col items-center gap-[0.3rem]">
			<span class="mini-count" title={`${listOnlineCount} online`}>
				<span class="dot mini online"></span>
				{listOnlineCount}
			</span>
			<span class="mini-count">{listDevices.length}</span>
		</div>
	{/snippet}
</ResizableSidebar>

<style>
	@reference "../../app.css";
	.dot {
		@apply h-[0.6rem] w-[0.6rem] shrink-0 rounded-full bg-gray;
	}
	.dot.online {
		@apply bg-green;
	}
	.dot.mini {
		@apply h-[0.45rem] w-[0.45rem];
	}
	.sidebar-icon {
		@apply grid h-8 w-8 cursor-pointer place-items-center rounded-md border border-strong bg-transparent p-0 text-secondary transition-[background-color,border-color,color] duration-[140ms] ease-out hover:border-teal hover:bg-hover hover:text-primary;
	}
	.sidebar-icon svg {
		@apply h-4 w-4;
	}
	.mini-count {
		@apply flex items-center gap-[0.2rem] text-[0.7rem] font-bold whitespace-nowrap text-secondary;
	}
</style>
