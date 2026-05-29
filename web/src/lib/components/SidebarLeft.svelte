<script lang="ts">
	import type { Device } from '../api/schemas';
	import ResizableSidebar from './ResizableSidebar.svelte';
	import SearchInput from './SearchInput.svelte';

	let {
		open = $bindable(true),
		devices = [],
		visibleDevices = [],
		selectedDevice = $bindable<Device | undefined>(undefined),
		showLabels = $bindable(true),
		showOffline = $bindable(true),
		showSubnetRouters = $bindable(true),
		showTailnet = $bindable(false),
		selectedTag = $bindable('all'),
		selectedOwner = $bindable('all'),
		selectedOS = $bindable('all'),
		colorBy = $bindable<'status' | 'tag' | 'owner' | 'os'>('status'),
		tagOptions = [],
		ownerOptions = [],
		osOptions = [],
		visibleOnlineCount = 0,
		chooseDevice
	}: {
		open?: boolean;
		devices: Device[];
		visibleDevices: Device[];
		selectedDevice?: Device;
		showLabels?: boolean;
		showOffline?: boolean;
		showSubnetRouters?: boolean;
		showTailnet?: boolean;
		selectedTag?: string;
		selectedOwner?: string;
		selectedOS?: string;
		colorBy?: 'status' | 'tag' | 'owner' | 'os';
		tagOptions: string[];
		ownerOptions: string[];
		osOptions: string[];
		visibleOnlineCount?: number;
		chooseDevice: (device: Device) => void;
	} = $props();

	let searchQuery = $state('');

	const filteredDevices = $derived(
		visibleDevices.filter((d) => {
			if (!searchQuery.trim()) return true;
			return d.name.toLowerCase().includes(searchQuery.toLowerCase().trim());
		})
	);

	const visibleOfflineCount = $derived(visibleDevices.filter((d) => !d.online).length);

	function displayName(name: string) {
		return showTailnet ? name : name.split('.')[0];
	}
</script>

<ResizableSidebar position="left" defaultWidth={16 * 16} {open}>
	<div class="flex min-h-0 flex-col overflow-y-hidden">
		<div class="mb-3 shrink-0">
			<h2 class="m-0 text-[0.95rem] leading-[1.2]">Devices</h2>
		</div>

		<div class="border-base-light mb-[0.85rem] shrink-0 border-b pb-[0.85rem]">
			<h3 class="section-title">View</h3>
			<div class="flex flex-col gap-[0.35rem]">
				<label class="label">
					<input type="checkbox" bind:checked={showLabels} class="m-0 h-4 w-4" />
					<span>Show labels</span>
				</label>
				<label class="label">
					<input type="checkbox" bind:checked={showOffline} class="m-0 h-4 w-4" />
					<span>Show offline</span>
				</label>
				<label class="label">
					<input type="checkbox" bind:checked={showSubnetRouters} class="m-0 h-4 w-4" />
					<span>Subnet routers</span>
				</label>
			</div>
		</div>

		<div class="border-base-light mb-[0.85rem] shrink-0 border-b pb-[0.85rem]">
			<h3 class="section-title">Filter</h3>
			<div class="flex flex-col gap-[0.45rem]">
				<label class="filter-label">
					<span>Tag</span>
					<select bind:value={selectedTag} class="select">
						<option value="all">All tags</option>
						{#each tagOptions as tag (tag)}
							<option value={tag}>{tag}</option>
						{/each}
					</select>
				</label>
				<label class="filter-label">
					<span>Owner</span>
					<select bind:value={selectedOwner} class="select">
						<option value="all">All owners</option>
						{#each ownerOptions as owner (owner)}
							<option value={owner}>{owner}</option>
						{/each}
					</select>
				</label>
				<label class="filter-label">
					<span>OS</span>
					<select bind:value={selectedOS} class="select">
						<option value="all">All OSes</option>
						{#each osOptions as os (os)}
							<option value={os}>{os}</option>
						{/each}
					</select>
				</label>
			</div>
		</div>

		<div class="border-base-light mb-[0.85rem] shrink-0 border-b pb-[0.85rem]">
			<h3 class="section-title">Colorize</h3>
			<div class="flex gap-0">
				{#each ['status', 'tag', 'owner', 'os'] as const as mode (mode)}
					<button
						type="button"
						class="segment"
						data-active={colorBy === mode}
						onclick={() => (colorBy = mode)}
					>
						{mode === 'os' ? 'OS' : mode[0].toUpperCase() + mode.slice(1)}
					</button>
				{/each}
			</div>
		</div>

		<div class="mb-0 flex min-h-0 flex-1 shrink flex-col border-b-0 pb-0">
			<h3 class="section-title">
				<span>List</span>
				<span
					class="pill"
					class:online={visibleOnlineCount > 0}
					class:offline={visibleOfflineCount > 0}
				>
					{visibleOnlineCount}/{visibleDevices.length}
				</span>
			</h3>
			{#if devices.length === 0}
				<p class="mt-2 text-[0.85rem] text-muted">No devices loaded.</p>
			{:else}
				<SearchInput
					bind:value={searchQuery}
					placeholder="Search devices..."
					count={filteredDevices.length}
					total={visibleDevices.length}
				/>
				<label class="label mt-1 mb-[0.35rem] text-[0.78rem] font-semibold text-label">
					<input type="checkbox" bind:checked={showTailnet} class="m-0 h-4 w-4" />
					<span>Show tailnet names</span>
				</label>
				<ul class="m-0 flex min-h-0 flex-1 list-none flex-col gap-[0.25rem] overflow-y-auto p-0">
					{#each filteredDevices as device (device.id)}
						<li>
							<button
								class={['device-item', selectedDevice?.id === device.id && 'active']}
								type="button"
								onclick={() => chooseDevice(device)}
							>
								<span class={['dot', device.online && 'online']}></span>
								<span class="overflow-hidden text-ellipsis whitespace-nowrap">
									{displayName(device.name)}
								</span>
							</button>
						</li>
					{/each}
				</ul>
			{/if}
		</div>
	</div>
	{#snippet collapsed()}
		<button class="sidebar-icon" title="Devices panel" type="button" onclick={() => (open = true)}>
			<svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
				<path d="M10 2.5L2.5 7.5V17.5H8.75V12.5H11.25V17.5H17.5V7.5L10 2.5Z" fill="currentColor" />
			</svg>
		</button>
		<div class="bg-border h-px w-[1.2rem]"></div>
		<div class="mt-[0.2rem] flex flex-col items-center gap-[0.3rem]">
			<span class="mini-count" title={`${visibleOnlineCount} online`}>
				<span class="dot mini online"></span>
				{visibleOnlineCount}
			</span>
			<span class="mini-count">{devices.length}</span>
		</div>
	{/snippet}
</ResizableSidebar>

<style>
	@reference "../../app.css";
	.section-title {
		@apply m-0 mb-2 flex items-center justify-between p-0 text-[0.72rem] font-bold tracking-wider text-label uppercase;
	}
	.label {
		@apply flex cursor-pointer items-center gap-[0.45rem] text-[0.85rem] font-bold text-primary;
	}
	.filter-label {
		@apply grid items-center gap-[0.4rem] text-[0.85rem] font-bold text-primary;
		grid-template-columns: 3.5rem minmax(0, 1fr);
	}
	.select {
		@apply w-full min-w-0 rounded-md border border-medium bg-input px-[0.45rem] py-[0.35rem] text-[0.85rem] text-primary transition-[border-color,box-shadow] duration-[140ms] ease-out outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)];
	}
	.segment {
		@apply flex-1 cursor-pointer rounded-none border border-strong bg-page px-[0.2rem] py-[0.35rem] text-[0.78rem] font-bold whitespace-nowrap text-secondary transition-[background-color,border-color,color] duration-[140ms] ease-out hover:bg-hover-weak hover:text-primary;
	}
	.segment:first-child {
		@apply rounded-l-md;
	}
	.segment:last-child {
		@apply rounded-r-md;
	}
	.segment[data-active='true'] {
		@apply border-teal bg-hover text-primary;
	}
	.device-item {
		@apply flex w-full min-w-0 cursor-pointer items-center gap-2 rounded-md border border-transparent bg-transparent px-2 py-[0.45rem] text-left text-[0.85rem] text-primary transition-[background-color,border-color] duration-[140ms] ease-out hover:border-strong hover:bg-hover;
	}
	.device-item.active {
		@apply border-strong bg-hover;
	}
	.dot {
		@apply h-[0.6rem] w-[0.6rem] shrink-0 rounded-full bg-gray;
	}
	.dot.online {
		@apply bg-green;
	}
	.dot.mini {
		@apply h-[0.45rem] w-[0.45rem];
	}
	.pill {
		@apply inline-flex items-center gap-[0.3rem] rounded-full bg-pill px-[0.4rem] py-[0.15rem] text-[0.68rem] font-bold text-teal transition-[background-color,color] duration-[160ms] ease-out;
	}
	.pill.online {
		@apply bg-online-pill text-online-pill-text;
	}
	.pill.offline {
		@apply bg-offline-pill text-offline-pill-text;
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
