<script lang="ts">
	import type { Device, PolicyEvaluateDraftResponse, PolicyMapResponse } from '../api/schemas';
	import type { RenderEdge } from '../graph/engine';
	import { kindLabel, validatePerspective } from '../perspective/catalog';
	import { isPerspectiveDevice, perspectiveSelectorFromDevice } from '../perspective/device';
	import ResizableSidebar from './ResizableSidebar.svelte';

	let {
		open = $bindable(true),
		selectedDevice = $bindable<Device | undefined>(undefined),
		selectedEdge = $bindable<RenderEdge | undefined>(undefined),
		devices = [],
		visibleEdges = [],
		colorBy = $bindable<'status' | 'tag' | 'owner' | 'os'>('status'),
		activePerspective = '',
		policyMap,
		graphViewMode = 'current',
		draftEvaluation = undefined,
		onSeedSource = () => {},
		onSeedDestination = () => {},
		onOpenPolicy = () => {},
		onViewAsOwner = () => {},
		onViewAsTag = () => {}
	}: {
		open?: boolean;
		selectedDevice?: Device;
		selectedEdge?: RenderEdge;
		devices?: Device[];
		visibleEdges?: RenderEdge[];
		colorBy?: 'status' | 'tag' | 'owner' | 'os';
		activePerspective?: string;
		policyMap?: PolicyMapResponse;
		graphViewMode?: 'current' | 'draft' | 'diff';
		draftEvaluation?: PolicyEvaluateDraftResponse;
		onSeedSource?: () => void;
		onSeedDestination?: () => void;
		onOpenPolicy?: () => void;
		onViewAsOwner?: (owner: string) => void;
		onViewAsTag?: (tag: string) => void;
	} = $props();

	const isPerspectiveSelection = $derived(
		selectedDevice ? isPerspectiveDevice(selectedDevice) : false
	);
	const perspectiveSelector = $derived(perspectiveSelectorFromDevice(selectedDevice));
	const perspectiveValidation = $derived(
		perspectiveSelector
			? validatePerspective(perspectiveSelector, devices, policyMap)
			: { status: 'empty' as const }
	);

	const edgeSource = $derived(devices.find((device) => device.id === selectedEdge?.from));
	const edgeTarget = $derived(devices.find((device) => device.id === selectedEdge?.to));
	const activeDevice = $derived(selectedDevice ?? edgeSource);
	const deviceInitials = $derived(
		activeDevice?.name ? activeDevice.name.split('.')[0].slice(0, 2).toUpperCase() : '?'
	);
	const outgoingEdges = $derived(
		selectedDevice ? visibleEdges.filter((edge) => edge.from === selectedDevice?.id) : []
	);
	const incomingEdges = $derived(
		selectedDevice ? visibleEdges.filter((edge) => edge.to === selectedDevice?.id) : []
	);

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

	function stateLabel(edge: RenderEdge) {
		return edge.state ? edge.state : graphViewMode === 'current' ? 'saved' : 'preview';
	}
</script>

<ResizableSidebar position="right" defaultWidth={18 * 16} {open}>
	<div class="mb-3 shrink-0">
		<h2 class="m-0 text-[0.95rem] leading-[1.2]">Policy Lens</h2>
		<p class="mt-1 mb-0 text-[0.78rem] leading-[1.4] font-semibold text-secondary">
			{#if activePerspective}
				Viewing access as <strong class="text-primary">{activePerspective}</strong>.
			{:else}
				Select a node or edge to explain and edit policy.
			{/if}
		</p>
	</div>

	{#if selectedEdge}
		<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
			<div class="flex items-center justify-between gap-2">
				<div>
					<p class="m-0 text-[0.72rem] font-extrabold tracking-wider text-label uppercase">
						Access relationship
					</p>
					<h3 class="mt-1 mb-0 text-[1rem] leading-[1.2]">{edgeTitle(selectedEdge)}</h3>
				</div>
				<span class="state-pill" data-state={selectedEdge.state ?? 'saved'}
					>{stateLabel(selectedEdge)}</span
				>
			</div>

			<div class="edge-route mt-3">
				<strong>{edgeSource?.name ?? selectedEdge.from}</strong>
				<span>can reach</span>
				<strong>{edgeTarget?.name ?? selectedEdge.to}</strong>
			</div>
		</div>

		<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
			<h3 class="section-title">Policy source</h3>
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
						{selectedEdge.policyRefs.map((ref) => `${ref.section} #${ref.index + 1}`).join(', ')}
					{:else}
						no policy reference
					{/if}
				</span>
			</div>
		</div>

		<div class="grid gap-2">
			<button class="lens-button primary" type="button" onclick={onSeedSource}>
				Use source in builder
			</button>
			<button class="lens-button" type="button" onclick={onSeedDestination}>
				Use destination in builder
			</button>
			<button class="lens-button" type="button" onclick={onOpenPolicy}>Open HuJSON detail</button>
		</div>
	{:else if selectedDevice && isPerspectiveSelection}
		<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
			<p class="m-0 text-[0.72rem] font-extrabold tracking-wider text-label uppercase">
				Simulated policy subject
			</p>
			<h3 class="mt-1 mb-0 text-[1rem] leading-[1.2] wrap-anywhere">{perspectiveSelector}</h3>
			<p class="mt-2 mb-0 text-[0.78rem] leading-[1.45] font-semibold text-secondary">
				Simulation only — not real login or device impersonation. The graph shows effective access
				this subject would have under saved policy.
			</p>
			{#if perspectiveValidation.status === 'valid'}
				<p class="mt-2 mb-0 text-[0.78rem] font-bold text-primary">
					{kindLabel(perspectiveValidation.kind)} · {perspectiveValidation.deviceCount} source device{perspectiveValidation.deviceCount ===
					1
						? ''
						: 's'}
				</p>
			{/if}
		</div>

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
	{:else if selectedDevice}
		<div
			class="border-base-light mb-[0.85rem] flex items-center gap-[0.65rem] border-b pb-[0.85rem]"
		>
			<span
				class="avatar grid h-9 w-9 shrink-0 place-items-center rounded-full text-[0.8rem] font-bold text-white transition-colors duration-[160ms] ease-out"
				style:background-color={avatarColor}
				data-subnet-router={selectedDevice.subnetRouter}
			>
				{deviceInitials}
			</span>
			<div class="min-w-0">
				<p
					class="m-0 overflow-hidden text-[0.9rem] font-bold text-ellipsis whitespace-nowrap text-primary"
				>
					{selectedDevice.name}
				</p>
				<div
					class="mt-[0.15rem] flex flex-wrap items-center gap-x-[0.45rem] gap-y-[0.3rem] text-[0.78rem] font-bold text-tertiary"
				>
					{#if colorBy !== 'status'}
						<span class="inline-flex items-center gap-[0.35rem]">
							<span class="dot" class:online={selectedDevice.online}></span>
							{selectedDevice.online ? 'online' : 'offline'}
						</span>
					{/if}
					{#if selectedDevice.tags.length > 0}
						<span
							class="bg-border-light inline-flex items-center rounded-full px-[0.45rem] py-[0.15rem] text-[0.75rem] font-bold text-primary"
							>{selectedDevice.tags[0]}</span
						>
						{#if selectedDevice.tags.length > 1}
							<span
								class="text-[0.75rem] font-bold text-muted"
								title={selectedDevice.tags.slice(1).join(', ')}
								>+{selectedDevice.tags.length - 1}</span
							>
						{/if}
					{/if}
				</div>
			</div>
		</div>

		<div class="flex min-h-0 flex-1 flex-col">
			<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
				<h3 class="section-title">Perspective</h3>
				<div class="grid gap-2">
					{#if selectedDevice.owner}
						<button
							class="lens-button"
							type="button"
							onclick={() => onViewAsOwner(selectedDevice.owner)}
						>
							View as {selectedDevice.owner}
						</button>
					{/if}
					{#each selectedDevice.tags as tag (tag)}
						<button class="lens-button" type="button" onclick={() => onViewAsTag(tag)}>
							View as {tag}
						</button>
					{/each}
				</div>
			</div>

			<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
				<h3 class="section-title">Policy actions</h3>
				<div class="grid gap-2">
					<button class="lens-button primary" type="button" onclick={onSeedSource}>
						Allow from this owner/tag
					</button>
					<button class="lens-button" type="button" onclick={onSeedDestination}>
						Allow to this device/tag
					</button>
				</div>
			</div>

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
				<div class="detail-row">
					<span class="detail-label">Preview</span><span class="detail-value"
						>{draftEvaluation
							? `${draftEvaluation.added.length} added, ${draftEvaluation.removed.length} removed`
							: 'saved policy'}</span
					>
				</div>
			</div>

			<div class="border-base-light mb-[0.85rem] border-b pb-[0.85rem]">
				<h3 class="section-title">Identity</h3>
				<div class="detail-row">
					<span class="detail-label">Name</span><span class="detail-value"
						>{selectedDevice.name}</span
					>
				</div>
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
				<div class="detail-row">
					<span class="detail-label">Status</span><span class="detail-value"
						><span class="dot" class:online={selectedDevice.online}></span>{selectedDevice.online
							? 'online'
							: 'offline'}</span
					>
				</div>
			</div>

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
				<div class="detail-row">
					<span class="detail-label">Subnet routes</span><span class="detail-value"
						>{selectedDevice.routedSubnets.length
							? selectedDevice.routedSubnets.join(', ')
							: 'none'}</span
					>
				</div>
			</div>

			<div class="border-base-light mb-[0.85rem] border-b">
				<h3 class="section-title">Tags</h3>
				{#if selectedDevice.tags.length > 0}
					<div class="flex flex-wrap gap-[0.35rem]">
						{#each selectedDevice.tags as tag (tag)}
							<span
								class="bg-border-light inline-flex items-center rounded-full px-[0.55rem] py-[0.25rem] text-[0.8rem] font-bold text-primary"
								>{tag}</span
							>
						{/each}
					</div>
				{:else}
					<p class="text-[0.85rem] text-muted">none</p>
				{/if}
			</div>
		</div>
	{:else}
		<div class="flex flex-1 items-center justify-center py-8">
			<p class="m-0 text-center text-[0.85rem] leading-[1.5] text-muted">
				Select a graph node for device policy, or an edge to see the rule that grants access.
			</p>
		</div>
	{/if}
	{#snippet collapsed()}
		<button
			class="sidebar-icon"
			title="Policy Lens panel"
			type="button"
			onclick={() => (open = true)}
		>
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
					class="mini-avatar rounded-full text-[0.65rem] font-bold text-white transition-colors duration-[160ms]"
					style:background-color={avatarColor}
					data-subnet-router={selectedDevice.subnetRouter}>{deviceInitials}</span
				>
			</button>
		{:else}
			<span class="text-[0.75rem] font-bold text-muted" title="No device selected">—</span>
		{/if}
	{/snippet}
</ResizableSidebar>

<style>
	@reference "../../app.css";
	.avatar[data-subnet-router='true'] {
		@apply rounded-lg;
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
	.state-pill {
		@apply shrink-0 rounded-full border border-panel-border bg-panel-weak px-2 py-[0.2rem] text-[0.68rem] font-extrabold tracking-wide text-secondary uppercase;
	}
	.state-pill[data-state='added'] {
		@apply border-ok text-ok;
	}
	.state-pill[data-state='removed'] {
		@apply border-danger text-danger;
	}
	.state-pill[data-state='changed'] {
		@apply border-warn text-warn;
	}
	.lens-button {
		@apply rounded-md border border-panel-border bg-panel-weak px-3 py-[0.55rem] text-left text-[0.8rem] font-extrabold text-primary transition-[background-color,border-color,color] duration-[140ms] ease-out hover:border-teal hover:bg-hover;
	}
	.lens-button.primary {
		@apply border-panel-accent bg-panel-accent text-panel-fg hover:bg-panel-accent;
	}
	.dot {
		@apply h-[0.6rem] w-[0.6rem] shrink-0 rounded-full bg-gray;
	}
	.dot.online {
		@apply bg-green;
	}
	.mini-avatar {
		@apply grid h-7 w-7 shrink-0 place-items-center;
	}
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
