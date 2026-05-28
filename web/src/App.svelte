<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import {
		authenticateCloud,
		draftPolicyRule,
		evaluatePolicyDraft,
		fetchCloudStatus,
		fetchPolicy,
		fetchPolicyMap,
		saveValidatedPolicyDraft,
		validatePolicyDraft
	} from './lib/api/cloud';
	import { fetchHealth } from './lib/api/health';
	import type {
		CloudAuthStatusResponse,
		Device,
		Edge,
		LocalAPIStatusResponse,
		PolicyEvaluateDraftResponse,
		PolicyMapResponse,
		PolicyResponse
	} from './lib/api/schemas';
	import { fetchTopology } from './lib/api/topology';
	import { connectTopologySocket } from './lib/api/topologySocket';
	import type { RenderEdge } from './lib/graph/engine';
	import AuthDialog from './lib/components/AuthDialog.svelte';
	import DraftTray from './lib/components/DraftTray.svelte';
	import GraphCanvas from './lib/components/GraphCanvas.svelte';
	import GraphLegend from './lib/components/GraphLegend.svelte';
	import PerspectiveBar from './lib/components/PerspectiveBar.svelte';
	import PolicyPanel from './lib/components/PolicyPanel.svelte';
	import SidebarLeft from './lib/components/SidebarLeft.svelte';
	import SidebarRight from './lib/components/SidebarRight.svelte';
	import SidebarToggleButton from './lib/components/SidebarToggleButton.svelte';

	let apiStatus = $state('checking');
	let devices = $state<Device[]>([]);
	let edges = $state<Edge[]>([]);
	let tailnetName = $state('');
	let selectedDevice = $state<Device | undefined>();
	let selectedEdge = $state<RenderEdge | undefined>();
	let cloudStatus = $state<CloudAuthStatusResponse>({
		authenticated: false,
		hasPolicy: false
	});
	let cloudError = $state('');
	let policy = $state<PolicyResponse | undefined>();
	let policyMap = $state<PolicyMapResponse | undefined>();
	let policySearch = $state('');
	let draftHuJSON = $state('');
	let draftRuleText = $state('');
	let draftEvaluation = $state<PolicyEvaluateDraftResponse | undefined>();
	let draftEvaluationPerspective = $state('');
	let editSource = $state('');
	let editDestination = $state('');
	let editPortPreset = $state('443');
	let editCustomPorts = $state('');
	let editStatus = $state('');
	let editBusy = $state(false);
	let draftValid = $state(false);
	let policyPerspective = $state('');
	let simulatedPerspective = $state('');
	let perspectiveEvaluation = $state<PolicyEvaluateDraftResponse | undefined>();
	let policyGraphViewMode = $state<'current' | 'draft' | 'diff'>('current');
	let phase2Open = $state(false);
	let policyOpen = $state(false);
	let localApiError = $state<LocalAPIStatusResponse | Error | undefined>();
	let cloudBusy = $state(false);
	let showOffline = $state(true);
	let showSubnetRouters = $state(true);
	let showTailnet = $state(false);
	let showLabels = $state(false);
	let graphMode = $state<'focused' | 'all'>('focused');
	let selectedTag = $state('all');
	let selectedOwner = $state('all');
	let selectedOS = $state('all');
	let colorBy = $state<'status' | 'tag' | 'owner' | 'os'>('status');
	let leftOpen = $state(true);
	let rightOpen = $state(true);

	let graphAPI:
		| {
				fit: () => void;
				zoom: (delta: number) => void;
				reflow: () => void;
				selectDevice: (device: Device) => void;
		  }
		| undefined;
	let disconnectTopologySocket: (() => void) | undefined;

	const visibleDevices = $derived(
		devices.filter((device) => {
			if (!showOffline && !device.online) return false;
			if (!showSubnetRouters && device.subnetRouter) return false;
			if (selectedTag !== 'all' && !device.tags.includes(selectedTag)) return false;
			if (selectedOwner !== 'all' && device.owner !== selectedOwner) return false;
			if (selectedOS !== 'all' && device.os !== selectedOS) return false;
			return true;
		})
	);
	const tagOptions = $derived(unique(devices.flatMap((device) => device.tags)));
	const ownerOptions = $derived(unique(devices.map((device) => device.owner).filter(Boolean)));
	const osOptions = $derived(unique(devices.map((device) => device.os).filter(Boolean)));
	const rootDevice = $derived(devices[0]);
	const graphRootDevice = $derived(
		cloudStatus.authenticated && graphMode === 'focused'
			? (selectedDevice ?? rootDevice)
			: rootDevice
	);
	const visibleDeviceIDs = $derived(new Set(visibleDevices.map((device) => device.id)));
	const visibleEdges = $derived(graphEdges());
	const graphDevices = $derived(devicesForGraph());
	const visibleOnlineCount = $derived(visibleDevices.filter((device) => device.online).length);
	const graphOnlineCount = $derived(graphDevices.filter((device) => device.online).length);
	const activePerspective = $derived(policyPerspective.trim());
	const activePerspectiveEvaluation = $derived(
		activePerspective && activePerspective === simulatedPerspective
			? perspectiveEvaluation
			: undefined
	);
	const activeDraftEvaluation = $derived(
		draftEvaluation && activePerspective === draftEvaluationPerspective
			? draftEvaluation
			: undefined
	);
	const activePolicyEvaluation = $derived(activeDraftEvaluation ?? activePerspectiveEvaluation);
	const selectorOptions = $derived(policySelectorOptions());

	function unique(values: string[]) {
		return [...new Set(values)].sort((a, b) => a.localeCompare(b));
	}

	function graphEdges(): RenderEdge[] {
		if (cloudStatus.authenticated && edges.length > 0) {
			const rendered = policyEdgesForGraph().filter(
				(edge) => visibleDeviceIDs.has(edge.from) && visibleDeviceIDs.has(edge.to)
			);
			if (graphMode === 'all') return rendered;
			const focusID = graphRootDevice?.id;
			if (!focusID) return [];
			return rendered.filter((edge) => edge.from === focusID || edge.to === focusID);
		}
		const root = rootDevice;
		if (!root || !visibleDeviceIDs.has(root.id) || !root.online) return [];
		return visibleDevices
			.filter((device) => device.id !== root.id && device.online)
			.map((device) => ({
				id: `local:${root.id}:${device.id}`,
				from: root.id,
				to: device.id,
				kind: 'local'
			}));
	}

	function policyEdgesForGraph(): RenderEdge[] {
		if (!activePolicyEvaluation || (!draftHuJSON && !activePerspectiveEvaluation)) {
			return edges.map((edge) => renderEdge(edge));
		}
		return evaluationEdges(activePolicyEvaluation, policyGraphViewMode, Boolean(draftHuJSON));
	}

	function evaluationEdges(
		evaluation: PolicyEvaluateDraftResponse,
		mode: 'current' | 'draft' | 'diff',
		hasDraft: boolean
	): RenderEdge[] {
		if (mode === 'diff' && hasDraft) {
			return [
				...evaluation.added.map((change) => renderEdge(change.edge, 'added')),
				...evaluation.removed.map((change) => renderEdge(change.edge, 'removed')),
				...evaluation.changed.map((change) => renderEdge(change.draft ?? change.edge, 'changed'))
			];
		}
		if (mode === 'draft') {
			return [
				...evaluation.unchanged.map((change) => renderEdge(change.edge, 'unchanged')),
				...evaluation.added.map((change) => renderEdge(change.edge, 'added')),
				...evaluation.changed.map((change) => renderEdge(change.draft ?? change.edge, 'changed'))
			];
		}
		return [
			...evaluation.unchanged.map((change) => renderEdge(change.edge, 'unchanged')),
			...evaluation.removed.map((change) =>
				renderEdge(change.edge, hasDraft ? 'removed' : 'unchanged')
			),
			...evaluation.changed.map((change) =>
				renderEdge(change.saved ?? change.edge, hasDraft ? 'changed' : 'unchanged')
			)
		];
	}

	function renderEdge(edge: Edge, state?: RenderEdge['state']): RenderEdge {
		return { ...edge, state };
	}

	function devicesForGraph(): Device[] {
		if (!cloudStatus.authenticated || graphMode === 'all' || edges.length === 0) {
			return visibleDevices;
		}
		const ids = new Set<string>(); // eslint-disable-line svelte/prefer-svelte-reactivity
		if (graphRootDevice?.id && visibleDeviceIDs.has(graphRootDevice.id)) {
			ids.add(graphRootDevice.id);
		}
		for (const edge of visibleEdges) {
			ids.add(edge.from);
			ids.add(edge.to);
		}
		return visibleDevices.filter((device) => ids.has(device.id));
	}

	function chooseDevice(device: Device) {
		selectedEdge = undefined;
		selectedDevice = device;
		graphAPI?.selectDevice(device);
	}

	function localApiErrorMessage(error: LocalAPIStatusResponse | Error | undefined) {
		if (!error) return '';
		if ('available' in error) {
			return error.error ?? `Unable to reach ${error.localApiEndpoint}`;
		}
		return error.message;
	}

	function splitSelectors(value: string) {
		return value
			.split(',')
			.map((part) => part.trim())
			.filter(Boolean);
	}

	function getPorts() {
		if (editPortPreset === 'custom') {
			return splitSelectors(editCustomPorts);
		}
		return splitSelectors(editPortPreset);
	}

	function policySelectorOptions() {
		const groupSelectors =
			policyMap?.sections
				.find((section) => section.name === 'groups')
				?.entries?.map((entry) => entry.label) ?? [];
		return unique(
			[...ownerOptions, ...tagOptions, ...groupSelectors, 'autogroup:member'].filter(Boolean)
		);
	}

	async function enableACLEditing(data: { tailnet: string; apiKey: string }) {
		if (cloudBusy) return;
		cloudBusy = true;
		cloudError = '';
		const result = await authenticateCloud({
			tailnet: data.tailnet,
			apiKey: data.apiKey
		});
		await result.match({
			ok: async (value) => {
				const topology = await fetchTopology();
				if (topology.isErr()) {
					cloudError = topology.error.message;
					return;
				}
				devices = topology.value.devices;
				edges = topology.value.edges;
				tailnetName = topology.value.tailnet;
				selectedDevice = selectedDevice
					? (topology.value.devices.find((device) => device.id === selectedDevice?.id) ??
						topology.value.devices[0])
					: topology.value.devices[0];
				cloudStatus = value;
				phase2Open = false;
				await loadPolicy();
			},
			err: async (error) => {
				cloudError = error.message;
			}
		});
		cloudBusy = false;
	}

	async function loadPolicy(options: { open?: boolean } = {}) {
		const [rawResult, mapResult] = await Promise.all([fetchPolicy(), fetchPolicyMap()]);
		rawResult.match({
			ok: (value) => {
				policy = value;
				draftHuJSON = '';
				draftRuleText = '';
				draftEvaluation = undefined;
				draftEvaluationPerspective = '';
				draftValid = false;
				perspectiveEvaluation = undefined;
				simulatedPerspective = '';
				policyOpen = options.open ?? false;
				cloudError = '';
			},
			err: (error) => {
				cloudError = error.message;
			}
		});
		mapResult.match({
			ok: (value) => {
				policyMap = value;
			},
			err: (error) => {
				cloudError = error.message;
			}
		});
	}

	async function createPolicyDraft() {
		if (editBusy) return;
		editBusy = true;
		cloudError = '';
		editStatus = '';
		draftValid = false;
		draftEvaluation = undefined;
		draftEvaluationPerspective = '';
		const result = await draftPolicyRule({
			sources: splitSelectors(editSource),
			destinations: splitSelectors(editDestination),
			ports: getPorts(),
			protocol: 'tcp'
		});
		await result.match({
			ok: async (value) => {
				draftHuJSON = value.hujson;
				draftRuleText = JSON.stringify(value.rule, null, 2);
				policyGraphViewMode = 'draft';
				await evaluateDraftImpact(value.hujson);
			},
			err: async (error) => {
				cloudError = error.message;
			}
		});
		editBusy = false;
	}

	async function evaluateDraftImpact(hujson: string) {
		const result = await evaluatePolicyDraft({
			hujson,
			perspective: activePerspective || undefined
		});
		result.match({
			ok: (value) => {
				draftEvaluation = value;
				draftEvaluationPerspective = activePerspective;
				editStatus = draftEvaluationSummary(value);
			},
			err: (error) => {
				draftEvaluation = undefined;
				draftEvaluationPerspective = '';
				editStatus = 'Draft ready. Impact preview unavailable; validate before saving.';
				cloudError = error.message;
			}
		});
	}

	async function simulatePerspective() {
		if (editBusy || !activePerspective || !policy?.hujson) return;
		editBusy = true;
		cloudError = '';
		if (draftHuJSON) {
			await evaluateDraftImpact(draftHuJSON);
			simulatedPerspective = activePerspective;
			policyGraphViewMode = 'draft';
			graphMode = 'all';
			editBusy = false;
			return;
		}
		const result = await evaluatePolicyDraft({
			hujson: policy.hujson,
			perspective: activePerspective
		});
		result.match({
			ok: (value) => {
				perspectiveEvaluation = value;
				simulatedPerspective = activePerspective;
				policyGraphViewMode = draftHuJSON ? 'draft' : 'current';
				graphMode = 'all';
				editStatus = `${activePerspective} can reach ${value.unchanged.length} saved edges.`;
			},
			err: (error) => {
				perspectiveEvaluation = undefined;
				simulatedPerspective = '';
				cloudError = error.message;
			}
		});
		editBusy = false;
	}

	function clearPerspective() {
		policyPerspective = '';
		simulatedPerspective = '';
		perspectiveEvaluation = undefined;
		policyGraphViewMode = draftHuJSON ? 'draft' : 'current';
	}

	function draftEvaluationSummary(value: PolicyEvaluateDraftResponse) {
		const parts = [
			`${value.added.length} added`,
			`${value.changed.length} changed`,
			`${value.removed.length} removed`
		];
		if (value.broadAccess.length > 0) {
			parts.push(`${value.broadAccess.length} broad`);
		}
		if (value.unresolvedSelectors.length > 0) {
			parts.push(`${value.unresolvedSelectors.length} unresolved`);
		}
		if (value.applicationGrants.length > 0) {
			parts.push(`${value.applicationGrants.length} app grant`);
		}
		return `Draft impact: ${parts.join(', ')}. Validate before saving.`;
	}

	async function validateDraft() {
		if (editBusy || !draftHuJSON) return;
		editBusy = true;
		cloudError = '';
		const result = await validatePolicyDraft(draftHuJSON);
		result.match({
			ok: (value) => {
				draftValid = value.valid;
				editStatus = value.valid
					? 'Draft validated. Save is enabled.'
					: (value.errors ?? ['Draft failed validation.']).join(' ');
			},
			err: (error) => {
				draftValid = false;
				cloudError = error.message;
			}
		});
		editBusy = false;
	}

	function discardDraft() {
		draftHuJSON = '';
		draftRuleText = '';
		draftEvaluation = undefined;
		draftEvaluationPerspective = '';
		draftValid = false;
		editStatus = '';
		policyGraphViewMode = 'current';
	}

	async function saveDraft() {
		if (editBusy || !draftValid) return;
		editBusy = true;
		cloudError = '';
		const result = await saveValidatedPolicyDraft();
		result.match({
			ok: (value) => {
				policy = { tailnet: value.tailnet, hujson: value.hujson };
				void refreshPolicyMap();
				draftHuJSON = '';
				draftRuleText = '';
				draftEvaluation = undefined;
				draftEvaluationPerspective = '';
				draftValid = false;
				perspectiveEvaluation = undefined;
				simulatedPerspective = '';
				policyGraphViewMode = 'current';
				editStatus = 'Saved. Topology will refresh from the updated policy.';
			},
			err: (error) => {
				cloudError = error.message;
			}
		});
		editBusy = false;
	}

	async function refreshPolicyMap() {
		const result = await fetchPolicyMap();
		result.match({
			ok: (value) => {
				policyMap = value;
			},
			err: (error) => {
				cloudError = error.message;
			}
		});
	}

	function closePhase2Dialog() {
		if (cloudBusy) return;
		phase2Open = false;
	}

	function deriveTailnet(): string {
		if (cloudStatus.tailnet) return cloudStatus.tailnet;
		if (tailnetName) return tailnetName;
		return '-';
	}

	function selectorForDevice(device: Device | undefined, role: 'source' | 'destination') {
		if (!device) return '';
		if (role === 'source' && device.owner) return device.owner;
		if (device.tags[0]) return device.tags[0];
		return device.tailscaleIps[0] ?? device.ip ?? device.name;
	}

	function stripDestinationPorts(selector: string) {
		if (!selector || selector === '*') return selector;
		const lastColon = selector.lastIndexOf(':');
		if (lastColon <= 3) return selector;
		return selector.slice(0, lastColon);
	}

	function isWildcardSelector(selector: string | undefined) {
		if (!selector || selector === '*') return true;
		const host = stripDestinationPorts(selector);
		return host === '*' || host === '*:*';
	}

	async function openPolicyEditor() {
		if (policy) {
			policyOpen = true;
			return;
		}
		await loadPolicy({ open: true });
	}

	function seedBuilder(role: 'source' | 'destination') {
		const edge = selectedEdge;
		const edgeRef = edge?.policyRefs?.[0];
		const device = edge
			? devices.find((item) => item.id === (role === 'source' ? edge.from : edge.to))
			: selectedDevice;
		const deviceSelector = selectorForDevice(device, role);

		if (role === 'source') {
			editSource = edgeRef?.src && !isWildcardSelector(edgeRef.src) ? edgeRef.src : deviceSelector;
		} else {
			const refDestination = stripDestinationPorts(edgeRef?.dst ?? '');
			editDestination =
				refDestination && !isWildcardSelector(refDestination) ? refDestination : deviceSelector;
		}
		void openPolicyEditor();
	}

	onMount(async () => {
		const health = await fetchHealth();
		health.match({
			ok: (value) => {
				apiStatus = value.status;
			},
			err: (error) => {
				apiStatus = error.message;
			}
		});

		const cloud = await fetchCloudStatus();
		cloud.match({
			ok: (value) => {
				cloudStatus = value;
			},
			err: (error) => {
				cloudError = error.message;
			}
		});

		disconnectTopologySocket = connectTopologySocket({
			onSnapshot: (value) => {
				apiStatus = 'connected';
				localApiError = undefined;
				devices = value.devices;
				edges = value.edges;
				tailnetName = value.tailnet;
				selectedDevice = selectedDevice
					? (value.devices.find((device) => device.id === selectedDevice?.id) ?? value.devices[0])
					: value.devices[0];
			},
			onUnavailable: (status) => {
				apiStatus = 'LocalAPI unavailable';
				localApiError = status;
				devices = [];
				edges = [];
				tailnetName = '';
				selectedDevice = undefined;
			},
			onConnectionState: (state) => {
				if (state === 'connected' && devices.length > 0) {
					apiStatus = 'connected';
					return;
				}
				apiStatus = state;
			},
			onError: (error) => {
				if (devices.length === 0) {
					apiStatus = 'socket error';
					localApiError = error;
				}
			}
		});
	});

	onDestroy(() => {
		disconnectTopologySocket?.();
	});
</script>

<main class="min-h-screen">
	<section class="grid h-screen grid-rows-[auto_minmax(0,1fr)] overflow-hidden">
		<div class="flex items-center justify-between gap-4 border-b border-base bg-surface px-5 py-4">
			<div>
				<p class="m-0 text-[0.8rem] font-bold tracking-normal text-secondary uppercase">
					Tailnet topology
				</p>
				<h1 class="m-0 text-2xl leading-[1.1]">Tailor</h1>
			</div>
			<div class="flex min-w-0 items-center gap-[0.6rem]">
				{#if cloudStatus.authenticated}
					<div class="hidden" aria-label="Policy Lens graph mode">
						<button
							type="button"
							class="min-h-[1.95rem] rounded-md border-0 bg-transparent px-[0.55rem] py-[0.35rem] text-[0.78rem] font-extrabold whitespace-nowrap text-secondary"
							class:bg-panel-accent={graphMode === 'focused'}
							class:text-panel-fg={graphMode === 'focused'}
							onclick={() => (graphMode = 'focused')}
						>
							Focused
						</button>
						<button
							type="button"
							class="min-h-[1.95rem] rounded-md border-0 bg-transparent px-[0.55rem] py-[0.35rem] text-[0.78rem] font-extrabold whitespace-nowrap text-secondary"
							class:bg-panel-accent={graphMode === 'all'}
							class:text-panel-fg={graphMode === 'all'}
							onclick={() => (graphMode = 'all')}
						>
							All connections
						</button>
					</div>
				{/if}
				{#if cloudStatus.authenticated}
					<button class="btn-secondary" type="button" onclick={() => loadPolicy({ open: true })}>
						Raw HuJSON
					</button>
				{:else}
					<button class="btn-primary" type="button" onclick={() => (phase2Open = true)}>
						Enable ACL Editing
					</button>
				{/if}
				<div
					class="flex min-w-[5rem] items-center gap-[0.3rem] rounded-full border border-status-border bg-status-bg p-[0.45rem_0.7rem] text-center text-[0.85rem] font-bold text-status-text"
					data-state={apiStatus}
				>
					<span class="h-2 w-2 rounded-full"></span>{apiStatus}
				</div>
			</div>
		</div>

		<div class="flex h-full min-h-0">
			<SidebarLeft
				bind:open={leftOpen}
				{devices}
				{visibleDevices}
				bind:selectedDevice
				bind:showLabels
				bind:showOffline
				bind:showSubnetRouters
				bind:showTailnet
				bind:selectedTag
				bind:selectedOwner
				bind:selectedOS
				bind:colorBy
				{tagOptions}
				{ownerOptions}
				{osOptions}
				{visibleOnlineCount}
				chooseDevice={(device) => chooseDevice(device)}
			/>

			<section
				class="graph relative min-h-[32rem] min-w-0 flex-1 overflow-hidden"
				aria-label="Topology graph"
			>
				<SidebarToggleButton
					position="left"
					open={leftOpen}
					ontoggle={() => (leftOpen = !leftOpen)}
				/>
				<SidebarToggleButton
					position="right"
					open={rightOpen}
					ontoggle={() => (rightOpen = !rightOpen)}
				/>

				<div
					class="absolute top-3 left-3 z-[2] flex items-center gap-[0.4rem] rounded-lg border border-graph-border bg-graph-hud-bg p-[0.35rem] shadow-[0_10px_26px_rgb(23_33_38/8%)]"
					aria-label="Graph summary"
				>
					<span
						class="inline-flex min-h-8 items-baseline gap-[0.3rem] rounded-md bg-graph-dot p-[0.35rem_0.55rem] text-[0.78rem] font-bold whitespace-nowrap text-secondary"
						><strong class="text-[0.98rem] leading-none text-graph-hud-strong"
							>{graphOnlineCount}</strong
						> online</span
					>
					<span
						class="inline-flex min-h-8 items-baseline gap-[0.3rem] rounded-md bg-graph-dot p-[0.35rem_0.55rem] text-[0.78rem] font-bold whitespace-nowrap text-secondary"
						><strong class="text-[0.98rem] leading-none text-graph-hud-strong"
							>{visibleEdges.length}</strong
						> links</span
					>
					{#if cloudStatus.authenticated}
						<span
							class="inline-flex min-h-8 items-baseline gap-[0.3rem] rounded-md bg-graph-dot p-[0.35rem_0.55rem] text-[0.78rem] font-bold whitespace-nowrap text-secondary capitalize"
							>{policyGraphViewMode} view</span
						>
					{/if}
					{#if cloudStatus.authenticated && graphMode === 'focused' && graphRootDevice}
						<span
							class="inline-flex min-h-8 items-baseline gap-[0.3rem] rounded-md bg-graph-dot p-[0.35rem_0.55rem] text-[0.78rem] font-bold whitespace-nowrap text-secondary"
							><strong class="text-[0.98rem] leading-none text-graph-hud-strong"
								>{graphRootDevice.name || graphRootDevice.ip}</strong
							> focus</span
						>
					{/if}
				</div>
				{#if cloudStatus.authenticated}
					<PerspectiveBar
						bind:perspective={policyPerspective}
						{selectorOptions}
						bind:graphViewMode={policyGraphViewMode}
						hasDraft={Boolean(draftHuJSON)}
						hasPerspectivePreview={Boolean(
							activePerspectiveEvaluation || (activePerspective && activeDraftEvaluation)
						)}
						busy={editBusy}
						onApply={simulatePerspective}
						onClear={clearPerspective}
					/>
				{/if}
				<div
					class="absolute top-3 right-3 z-[2] flex gap-[0.35rem] rounded-lg border border-graph-border bg-graph-hud-bg p-[0.35rem] shadow-[0_8px_22px_rgb(23_33_38/8%)]"
					aria-label="Graph controls"
				>
					<button
						type="button"
						title="Zoom in"
						onclick={() => graphAPI?.zoom(1.2)}
						class="grid h-[2.1rem] w-[2.1rem] min-w-[2.1rem] place-items-center rounded-md border border-panel-border bg-panel-weak leading-none font-extrabold text-primary transition-[background-color,border-color,transform] duration-[160ms] ease-out hover:-translate-y-px hover:border-teal hover:bg-hover motion-reduce:transition-none motion-reduce:hover:transform-none"
						>+</button
					>
					<button
						type="button"
						title="Zoom out"
						onclick={() => graphAPI?.zoom(0.8)}
						class="grid h-[2.1rem] w-[2.1rem] min-w-[2.1rem] place-items-center rounded-md border border-panel-border bg-panel-weak leading-none font-extrabold text-primary transition-[background-color,border-color,transform] duration-[160ms] ease-out hover:-translate-y-px hover:border-teal hover:bg-hover motion-reduce:transition-none motion-reduce:hover:transform-none"
						>-</button
					>
					<button
						type="button"
						title="Fit to view"
						onclick={() => graphAPI?.fit()}
						class="grid h-[2.1rem] w-[2.1rem] min-w-[2.1rem] place-items-center rounded-md border border-panel-border bg-panel-weak leading-none font-extrabold text-primary transition-[background-color,border-color,transform] duration-[160ms] ease-out hover:-translate-y-px hover:border-teal hover:bg-hover motion-reduce:transition-none motion-reduce:hover:transform-none"
						>⌖</button
					>
					<button
						type="button"
						title="Reflow layout"
						onclick={() => graphAPI?.reflow()}
						class="grid h-[2.1rem] w-[2.1rem] min-w-[2.1rem] place-items-center rounded-md border border-panel-border bg-panel-weak leading-none font-extrabold text-primary transition-[background-color,border-color,transform] duration-[160ms] ease-out hover:-translate-y-px hover:border-teal hover:bg-hover motion-reduce:transition-none motion-reduce:hover:transform-none"
						>↻</button
					>
				</div>

				<GraphCanvas
					{devices}
					{edges}
					{visibleDevices}
					{visibleEdges}
					{graphMode}
					bind:selectedDevice
					bind:selectedEdge
					{showLabels}
					{cloudStatus}
					{colorBy}
					{rootDevice}
					onReady={(api) => (graphAPI = api)}
				/>

				<GraphLegend
					{colorBy}
					authenticated={cloudStatus.authenticated}
					{graphMode}
					{tagOptions}
					{ownerOptions}
					{osOptions}
				/>

				{#if localApiError}
					<div
						class="absolute top-1/2 left-1/2 w-[min(28rem,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-base bg-surface p-4 shadow-[0_10px_30px_rgb(23_33_38/8%)]"
					>
						<h2 class="mb-[0.4rem] text-base">Connect to Tailscale</h2>
						<p class="mb-0 wrap-anywhere">{localApiErrorMessage(localApiError)}</p>
					</div>
				{/if}

				<PolicyPanel
					bind:open={policyOpen}
					{policy}
					{policyMap}
					bind:search={policySearch}
					{draftHuJSON}
					{draftRuleText}
					{draftEvaluation}
					{draftValid}
					{editBusy}
					{editStatus}
					bind:editSource
					bind:editDestination
					bind:editPortPreset
					bind:editCustomPorts
					{cloudError}
					onClose={() => (policyOpen = false)}
					onDraft={createPolicyDraft}
					onValidate={validateDraft}
					onSave={saveDraft}
				/>
				<DraftTray
					{draftEvaluation}
					{draftRuleText}
					draftValid={draftValid ? true : null}
					{editBusy}
					{editStatus}
					onValidate={validateDraft}
					onSave={saveDraft}
					onDiscard={discardDraft}
					onOpenAdvanced={openPolicyEditor}
				/>
			</section>

			<SidebarRight
				bind:open={rightOpen}
				bind:selectedDevice
				bind:selectedEdge
				{devices}
				{visibleEdges}
				{colorBy}
				{activePerspective}
				graphViewMode={policyGraphViewMode}
				draftEvaluation={activePolicyEvaluation}
				onSeedSource={() => seedBuilder('source')}
				onSeedDestination={() => seedBuilder('destination')}
				onOpenPolicy={openPolicyEditor}
			/>
		</div>
	</section>

	<AuthDialog
		bind:open={phase2Open}
		initialTailnet={deriveTailnet()}
		{cloudBusy}
		{cloudError}
		onClose={closePhase2Dialog}
		onSubmit={enableACLEditing}
	/>
</main>

<style>
	@reference "./app.css";
	.btn-primary {
		@apply min-h-[2.35rem] rounded-md border border-panel-accent bg-panel-accent px-3 py-[0.45rem] text-sm font-extrabold text-panel-fg transition-[background-color,border-color,color,transform] duration-[160ms] ease-out hover:-translate-y-px disabled:transform-none disabled:cursor-not-allowed disabled:opacity-[0.58];
	}
	.btn-secondary {
		@apply min-h-[2.35rem] rounded-md border border-panel-border bg-panel-weak px-3 py-[0.45rem] text-sm font-extrabold text-primary transition-[background-color,border-color,color,transform] duration-[160ms] ease-out hover:-translate-y-px disabled:transform-none disabled:cursor-not-allowed disabled:opacity-[0.58];
	}
</style>
