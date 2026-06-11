<script lang="ts">
	import type { Device } from '../api/schemas';
	import type { DeviceAggregateMeta } from '../graph/collapse-devices';
	import { isAggregateDeviceId } from '../graph/collapse-devices';
	import type { RenderEdge } from '../graph/engine';
	import { palette } from './avatar-color';
	import { getOwnerColor, getTagColor } from '../tag-color';

	let {
		selectedDevice = $bindable<Device | undefined>(undefined),
		selectedEdge = $bindable<RenderEdge | undefined>(undefined),
		devices = [],
		aggregateMeta = new Map<string, DeviceAggregateMeta>(),
		visibleEdges = [],
		colorBy = $bindable<'status' | 'tag' | 'owner' | 'os'>('status'),
		tagColorMap = new Map<string, string>(),
		ownerColorMap = new Map<string, string>(),
		showCredit = true,
		compact = false
	}: {
		selectedDevice?: Device;
		selectedEdge?: RenderEdge;
		devices?: Device[];
		aggregateMeta?: Map<string, DeviceAggregateMeta>;
		visibleEdges?: RenderEdge[];
		colorBy?: 'status' | 'tag' | 'owner' | 'os';
		tagColorMap?: ReadonlyMap<string, string>;
		ownerColorMap?: ReadonlyMap<string, string>;
		showCredit?: boolean;
		compact?: boolean;
	} = $props();

	const selectedAggregate = $derived(
		selectedDevice && isAggregateDeviceId(selectedDevice.id)
			? aggregateMeta.get(selectedDevice.id)
			: undefined
	);
	const aggregateOnlineCount = $derived(
		selectedAggregate?.members.filter((member) => member.online).length ?? 0
	);

	const edgeSource = $derived(devices.find((device) => device.id === selectedEdge?.from));
	const edgeTarget = $derived(devices.find((device) => device.id === selectedEdge?.to));
	const activeDevice = $derived(selectedDevice ?? edgeSource);
	const deviceInitials = $derived.by(() => {
		if (!activeDevice) return '?';
		if (selectedAggregate) return String(selectedAggregate.members.length);
		return activeDevice.name ? activeDevice.name.split('.')[0].slice(0, 2).toUpperCase() : '?';
	});
	const outgoingEdges = $derived(
		selectedDevice ? visibleEdges.filter((edge) => edge.from === selectedDevice?.id) : []
	);
	const incomingEdges = $derived(
		selectedDevice ? visibleEdges.filter((edge) => edge.to === selectedDevice?.id) : []
	);

	const avatarColor = $derived.by((): string | undefined => {
		if (!activeDevice) return undefined;
		if (colorBy === 'status') {
			return activeDevice.online ? '#41a86f' : '#9aa7a1';
		}
		if (colorBy === 'tag') {
			return getTagColor(activeDevice.tags[0], tagColorMap);
		}
		if (colorBy === 'owner') {
			return getOwnerColor(activeDevice.owner, ownerColorMap);
		}
		return palette(activeDevice.os || 'unknown');
	});

	function edgeTitle(edge: RenderEdge) {
		if (edge.accessScope === 'broad') return 'Broad access';
		if (edge.accessScope === 'ssh') return 'SSH access';
		if (edge.accessScope === 'http') return 'HTTP access';
		return 'Custom access';
	}

	function edgePorts(edge: RenderEdge) {
		return edge.ports?.length
			? edge.ports.join(', ')
			: edge.accessScope === 'broad'
				? 'all ports'
				: 'unspecified';
	}
</script>

<div class="flex min-h-full flex-col">
	{#if !compact}
		<div class="mb-3 shrink-0">
			<h2 class="m-0 text-[0.95rem] leading-[1.2]">Details</h2>
			<p class="mt-1 mb-0 text-[0.78rem] font-semibold text-secondary">
				Select a device or access link on the graph.
			</p>
		</div>
	{/if}

	{#if selectedEdge}
		<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
			<p class="m-0 text-[0.72rem] font-extrabold tracking-wider text-label uppercase">
				Access relationship
			</p>
			<h3 class="mt-1 mb-0 text-[1rem] leading-[1.2]">{edgeTitle(selectedEdge)}</h3>
			<div class="edge-route mt-3">
				<strong>{edgeSource?.name ?? selectedEdge.from}</strong>
				<span>can reach</span>
				<strong>{edgeTarget?.name ?? selectedEdge.to}</strong>
			</div>
		</div>

		<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
			<h3 class="section-title">Policy</h3>
			<div class="detail-row">
				<span class="detail-label">Scope</span><span class="detail-value"
					>{selectedEdge.accessScope || 'limited'}</span
				>
			</div>
			<div class="detail-row">
				<span class="detail-label">Ports</span><span class="detail-value"
					>{edgePorts(selectedEdge)}</span
				>
			</div>
			<div class="detail-row">
				<span class="detail-label">Protocols</span><span class="detail-value"
					>{selectedEdge.protocols?.length ? selectedEdge.protocols.join(', ') : 'tcp'}</span
				>
			</div>
			<div class="detail-row">
				<span class="detail-label">Rules</span>
				<span class="detail-value">
					{#if selectedEdge.policyRefs?.length}
						{#each selectedEdge.policyRefs as ref, index (ref.section + ref.index)}
							<span class="ref-chip">{ref.section} #{ref.index + 1}</span
							>{#if index < selectedEdge.policyRefs.length - 1},
							{/if}
						{/each}
					{:else}
						no policy reference
					{/if}
				</span>
			</div>
		</div>
	{:else if selectedDevice}
		<div
			class="border-base-light mb-[0.85rem] flex items-center gap-[0.65rem] border-b pb-[0.85rem]"
		>
			<span
				class="avatar grid h-9 w-9 shrink-0 place-items-center rounded-full text-[0.8rem] font-bold text-white"
				class:aggregate-avatar={Boolean(selectedAggregate)}
				style:background-color={avatarColor}
				data-subnet-router={selectedDevice.subnetRouter}
			>
				{deviceInitials}
			</span>
			<div class="min-w-0">
				<p
					class="m-0 overflow-hidden text-[0.9rem] font-bold text-ellipsis whitespace-nowrap text-primary"
				>
					{selectedAggregate?.label ?? selectedDevice.name}
				</p>
				<div
					class="mt-[0.15rem] flex flex-wrap items-center gap-x-[0.45rem] gap-y-[0.3rem] text-[0.78rem] font-bold text-tertiary"
				>
					{#if selectedAggregate}
						<span>{selectedAggregate.members.length} devices</span>
						<span class="inline-flex items-center gap-[0.35rem]">
							<span class="dot online"></span>
							{aggregateOnlineCount} online
						</span>
					{:else}
						<span class="inline-flex items-center gap-[0.35rem]">
							<span class="dot" class:online={selectedDevice.online}></span>
							{selectedDevice.online ? 'online' : 'offline'}
						</span>
					{/if}
					{#if selectedDevice.tags.length > 0}
						<span class="tag-pill">{selectedDevice.tags[0]}</span>
					{/if}
				</div>
			</div>
		</div>

		{#if selectedAggregate}
			<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
				<h3 class="section-title">Collapsed fleet</h3>
				<p class="m-0 text-[0.85rem] leading-[1.45] text-secondary">
					This node represents <strong class="text-primary"
						>{selectedAggregate.members.length}</strong
					>
					devices tagged
					<span class="tag-pill">{selectedAggregate.tag}</span>. Any member can serve the same role
					in your tailnet (for example as an exit node).
				</p>
				<div class="detail-row mt-3">
					<span class="detail-label">Online</span><span class="detail-value"
						>{aggregateOnlineCount} / {selectedAggregate.members.length}</span
					>
				</div>
			</div>
		{/if}

		<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
			<h3 class="section-title">Reachability</h3>
			<div class="detail-row">
				<span class="detail-label">Can reach</span><span class="detail-value"
					>{outgoingEdges.length} visible target{outgoingEdges.length === 1 ? '' : 's'}</span
				>
			</div>
			<div class="detail-row">
				<span class="detail-label">Reachable by</span><span class="detail-value"
					>{incomingEdges.length} visible source{incomingEdges.length === 1 ? '' : 's'}</span
				>
			</div>
		</div>

		<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
			<h3 class="section-title">Identity</h3>
			{#if selectedAggregate}
				<div class="detail-row">
					<span class="detail-label">Type</span><span class="detail-value">Tagged fleet</span>
				</div>
			{:else}
				<div class="detail-row">
					<span class="detail-label">Owner</span><span class="detail-value"
						>{selectedDevice.owner || 'unknown'}</span
					>
				</div>
				<div class="detail-row">
					<span class="detail-label">OS</span><span class="detail-value"
						>{selectedDevice.os || 'unknown'}</span
					>
				</div>
			{/if}
		</div>

		{#if !selectedAggregate}
			<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
				<h3 class="section-title">Network</h3>
				<div class="detail-row">
					<span class="detail-label">IP</span><span class="detail-value"
						>{selectedDevice.ip || 'unknown'}</span
					>
				</div>
				<div class="detail-row">
					<span class="detail-label">Tailscale IPs</span><span class="detail-value"
						>{selectedDevice.tailscaleIps.length
							? selectedDevice.tailscaleIps.join(', ')
							: 'unknown'}</span
					>
				</div>
			</div>
		{/if}

		{#if selectedDevice.tags.length > 0}
			<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
				<h3 class="section-title">Tags</h3>
				<div class="flex flex-wrap gap-[0.35rem]">
					{#each selectedDevice.tags as tag (tag)}
						<span class="tag-pill">{tag}</span>
					{/each}
				</div>
			</div>
		{/if}

		{#if selectedAggregate}
			<div class="border-base-light mb-[0.85rem] border-b">
				<h3 class="section-title">Members ({selectedAggregate.members.length})</h3>
				<ul class="member-list m-0 max-h-[12rem] list-none overflow-y-auto p-0">
					{#each selectedAggregate.members as member (member.id)}
						<li class="member-item">
							<span class="dot" class:online={member.online}></span>
							<span class="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap"
								>{member.name}</span
							>
						</li>
					{/each}
				</ul>
			</div>
		{/if}
	{:else}
		<div class="flex flex-1 items-center justify-center py-8">
			<p class="m-0 text-center text-[0.85rem] leading-[1.5] text-muted">
				Select a graph node or edge to inspect it here.
			</p>
		</div>
	{/if}

	{#if showCredit}
		<div class="d6-credit">
			<span>Made with ❤️ by </span>
			<a
				href="https://d6software.com"
				target="_blank"
				rel="noreferrer"
				aria-label="Made by D6software.com"
			>
				<img src="/d6-logo.svg" alt="" aria-hidden="true" />
				<span>D6software.com</span>
			</a>
		</div>
	{/if}
</div>

<style>
	@reference "../../app.css";
	.avatar[data-subnet-router='true'] {
		@apply rounded-lg;
	}
	.aggregate-avatar {
		@apply rounded-lg text-[0.72rem];
	}
	.section-title {
		@apply m-0 mb-2 p-0 text-[0.72rem] font-bold tracking-wider text-label uppercase;
	}
	.detail-row {
		@apply mb-2 grid grid-cols-[6.5rem_minmax(0,1fr)] items-start gap-x-[0.5rem] text-[0.85rem];
	}
	.detail-label {
		@apply font-bold text-secondary;
	}
	.detail-value {
		@apply min-w-0 leading-[1.35] font-bold break-words whitespace-normal text-primary;
	}
	.edge-route {
		@apply grid gap-1 rounded-lg border border-panel-border bg-panel-weak p-2 text-[0.82rem] font-bold text-secondary;
	}
	.edge-route strong {
		@apply text-primary;
	}
	.ref-chip {
		@apply text-[0.78rem] font-extrabold text-teal;
	}
	.tag-pill {
		@apply inline-flex items-center rounded-full bg-light px-[0.55rem] py-[0.25rem] text-[0.8rem] font-bold text-primary;
	}
	.dot {
		@apply h-[0.6rem] w-[0.6rem] shrink-0 rounded-full bg-gray;
	}
	.dot.online {
		@apply bg-green;
	}
	.member-list {
		@apply flex flex-col gap-[0.2rem];
	}
	.member-item {
		@apply flex items-center gap-[0.45rem] text-[0.82rem] font-semibold text-secondary;
	}
	.d6-credit {
		@apply mt-auto flex min-h-8 shrink-0 items-center gap-[0.45rem] border-t border-light pt-3 text-[0.68rem] leading-[1.2] font-extrabold text-muted;
		font-family: 'Space Grotesk', 'Red Hat Text', 'Helvetica Neue', sans-serif;
	}
	.d6-credit a {
		@apply flex items-center gap-[0.45rem] text-muted no-underline transition-[color,opacity] duration-[140ms] ease-out hover:text-primary;
	}
	.d6-credit img {
		@apply h-[1.15rem] w-[1.15rem] shrink-0;
	}
	.d6-credit a span {
		@apply min-w-0 overflow-hidden text-ellipsis whitespace-nowrap;
		letter-spacing: 0;
	}
</style>
