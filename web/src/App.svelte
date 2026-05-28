<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import { SvelteSet } from 'svelte/reactivity';
	import {
		authenticateCloud,
		evaluatePolicyDraft,
		fetchCloudStatus,
		fetchPolicy,
		fetchPolicyMap,
		mutatePolicyDraft,
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
	import { saveRecentPerspective, validatePerspective } from './lib/perspective/catalog';
	import { subjectDeviceIds } from './lib/perspective/subjects';
	import { focusedScenarioNodeIds, scenarioReachableCount } from './lib/scenario/graph';
	import {
		createScenario,
		loadScenario,
		saveScenario,
		scenarioLabel,
		type PolicyScenario
	} from './lib/scenario/state';
	import {
		createChange,
		diffLines,
		type DraftChange,
		type PolicyMutation
	} from './lib/draft/types';
	import AuthDialog from './lib/components/AuthDialog.svelte';
	import DraftTray from './lib/components/DraftTray.svelte';
	import GraphCanvas from './lib/components/GraphCanvas.svelte';
	import GraphLegend from './lib/components/GraphLegend.svelte';
	import PerspectiveBar from './lib/components/PerspectiveBar.svelte';
	import PolicyWorkbench from './lib/workbench/PolicyWorkbench.svelte';
	import { defaultWorkbenchRoute, type WorkbenchRoute } from './lib/workbench/nav';
	import RawPolicyPanel from './lib/components/RawPolicyPanel.svelte';
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
	let draftChanges = $state<DraftChange[]>([]);
	let draftEvaluation = $state<PolicyEvaluateDraftResponse | undefined>();
	let draftEvaluationPerspective = $state('');
	let editSeed = $state({ sources: '', destinations: '', ports: '443' });
	let editStatus = $state('');
	let editBusy = $state(false);
	let draftValid = $state<boolean | null>(null);
	let scenario = $state<PolicyScenario | null>(null);
	let showGhostEdges = $state(true);
	let policyPerspective = $state('');
	let simulatedPerspective = $state('');
	let perspectiveEvaluation = $state<PolicyEvaluateDraftResponse | undefined>();
	let policyGraphViewMode = $state<'current' | 'draft' | 'diff'>('current');
	let phase2Open = $state(false);
	let workbenchOpen = $state(false);
	let workbenchRoute = $state<WorkbenchRoute>(defaultWorkbenchRoute());
	let rawPolicyOpen = $state(false);
	let sidebarSnapshot = $state<{ left: boolean; right: boolean } | null>(null);
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
	const activeScenario = $derived(
		Boolean(scenario && activePerspective && activePerspective === simulatedPerspective)
	);
	const subjectIDs = $derived.by(() => {
		if (!activePerspective) return new SvelteSet<string>();
		return new SvelteSet(subjectDeviceIds(activePerspective, devices, policyMap));
	});
	const scenarioSourceIDs = $derived(activeScenario ? subjectIDs : new SvelteSet<string>());
	const graphRootDevice = $derived.by(() => {
		if (activeScenario && subjectIDs.size > 0) {
			if (selectedDevice && subjectIDs.has(selectedDevice.id)) return selectedDevice;
			return visibleDevices.find((device) => subjectIDs.has(device.id)) ?? rootDevice;
		}
		if (cloudStatus.authenticated && graphMode === 'focused') {
			return selectedDevice ?? rootDevice;
		}
		return rootDevice;
	});
	const graphVisibleDeviceIDs = $derived(new SvelteSet(visibleDevices.map((device) => device.id)));
	const visibleEdges = $derived(graphEdges());
	const graphDevices = $derived(devicesForGraph());
	const scenarioSourceCount = $derived(activeScenario ? subjectIDs.size : 0);
	const perspectiveReachableCount = $derived(
		activeScenario ? scenarioReachableCount(visibleEdges, subjectIDs) : 0
	);
	const activeScenarioLabel = $derived(
		activeScenario && scenario ? scenarioLabel(scenario, scenarioSourceCount) : ''
	);
	const draftDiffLines = $derived(
		policy?.hujson && draftHuJSON ? diffLines(policy.hujson, draftHuJSON) : []
	);
	const visibleOnlineCount = $derived(visibleDevices.filter((device) => device.online).length);
	const graphOnlineCount = $derived(graphDevices.filter((device) => device.online).length);

	function ghostDeniedEdges(allowed: RenderEdge[], sourceIds: ReadonlySet<string>): RenderEdge[] {
		const allowedPairs = new Set(allowed.map((edge) => `${edge.from}\0${edge.to}`));
		const reachable = new Set<string>(); // eslint-disable-line svelte/prefer-svelte-reactivity
		for (const edge of allowed) {
			if (sourceIds.has(edge.from)) reachable.add(edge.to);
		}
		const ghosts: RenderEdge[] = [];
		let count = 0;
		for (const sourceId of sourceIds) {
			for (const device of visibleDevices) {
				if (sourceIds.has(device.id) || reachable.has(device.id)) continue;
				const key = `${sourceId}\0${device.id}`;
				if (allowedPairs.has(key)) continue;
				ghosts.push({
					id: `ghost:${sourceId}:${device.id}`,
					from: sourceId,
					to: device.id,
					kind: 'acl',
					state: 'ghost-denied'
				});
				count += 1;
				if (count >= 24) return ghosts;
			}
		}
		return ghosts;
	}

	function syncScenarioModes() {
		if (!scenario) return;
		scenario = {
			...scenario,
			policyMode: policyGraphViewMode,
			graphMode
		};
		saveScenario(scenario);
	}

	async function applyDraftMutation(mutation: PolicyMutation, label: string) {
		if (editBusy) return;
		editBusy = true;
		cloudError = '';
		draftValid = null;
		editStatus = '';
		const result = await mutatePolicyDraft({
			hujson: draftHuJSON || undefined,
			mutation
		});
		await result.match({
			ok: async (value) => {
				draftHuJSON = value.hujson;
				draftRuleText = label;
				draftChanges = [...draftChanges, createChange(label)];
				policyGraphViewMode = 'draft';
				syncScenarioModes();
				await evaluateDraftImpact(value.hujson);
			},
			err: async (error) => {
				cloudError = error.message;
			}
		});
		editBusy = false;
	}

	function unique(values: string[]) {
		return [...new Set(values)].sort((a, b) => a.localeCompare(b));
	}

	function graphEdges(): RenderEdge[] {
		if (cloudStatus.authenticated && edges.length > 0) {
			let rendered = policyEdgesForGraph();
			rendered = rendered.filter(
				(edge) => graphVisibleDeviceIDs.has(edge.from) || graphVisibleDeviceIDs.has(edge.to)
			);
			if (graphMode === 'all') return rendered;
			if (activeScenario && subjectIDs.size > 0) {
				const nodeIds = focusedScenarioNodeIds(rendered, subjectIDs);
				rendered = rendered.filter((edge) => nodeIds.has(edge.from) && nodeIds.has(edge.to));
				if (showGhostEdges && graphMode === 'focused') {
					rendered = [...rendered, ...ghostDeniedEdges(rendered, subjectIDs)];
				}
				return rendered;
			}
			const focusID = graphRootDevice?.id;
			if (!focusID) return [];
			return rendered.filter((edge) => edge.from === focusID || edge.to === focusID);
		}
		const root = rootDevice;
		if (!root || !graphVisibleDeviceIDs.has(root.id) || !root.online) return [];
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
		const pool = visibleDevices;
		if (!cloudStatus.authenticated || graphMode === 'all' || edges.length === 0) {
			return pool;
		}
		const ids = new Set<string>(); // eslint-disable-line svelte/prefer-svelte-reactivity
		if (activeScenario && subjectIDs.size > 0) {
			for (const id of subjectIDs) ids.add(id);
			for (const edge of visibleEdges) {
				ids.add(edge.from);
				ids.add(edge.to);
			}
			return pool.filter((device) => ids.has(device.id));
		}
		if (graphRootDevice?.id) ids.add(graphRootDevice.id);
		for (const edge of visibleEdges) {
			ids.add(edge.from);
			ids.add(edge.to);
		}
		return pool.filter((device) => ids.has(device.id));
	}

	function chooseDevice(device: Device) {
		selectedEdge = undefined;
		selectedDevice = device;
		graphAPI?.selectDevice(device);
	}

	async function applyPerspectiveFromSelector(selector: string) {
		policyPerspective = selector;
		await simulatePerspective();
	}

	function localApiErrorMessage(error: LocalAPIStatusResponse | Error | undefined) {
		if (!error) return '';
		if ('available' in error) {
			return error.error ?? `Unable to reach ${error.localApiEndpoint}`;
		}
		return error.message;
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

	async function loadPolicy() {
		const [rawResult, mapResult] = await Promise.all([fetchPolicy(), fetchPolicyMap()]);
		rawResult.match({
			ok: (value) => {
				policy = value;
				draftHuJSON = '';
				draftRuleText = '';
				draftChanges = [];
				draftEvaluation = undefined;
				draftEvaluationPerspective = '';
				draftValid = null;
				perspectiveEvaluation = undefined;
				simulatedPerspective = '';
				scenario = null;
				saveScenario(null);
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

	function finishPerspectiveSimulation(selector: string) {
		simulatedPerspective = selector;
		graphMode = 'focused';
		scenario = createScenario(selector);
		scenario.policyMode = policyGraphViewMode;
		scenario.graphMode = graphMode;
		saveScenario(scenario);
		const sources = subjectDeviceIds(selector, devices, policyMap);
		const firstSource = visibleDevices.find((device) => sources.has(device.id));
		selectedDevice = firstSource ?? rootDevice;
		selectedEdge = undefined;
		saveRecentPerspective(selector);
		if (selectedDevice) graphAPI?.selectDevice(selectedDevice);
	}

	function openWorkbench(options: { ensurePolicy?: boolean; route?: WorkbenchRoute } = {}) {
		const openPanel = () => {
			if (options.route) workbenchRoute = options.route;
			if (!workbenchOpen) {
				sidebarSnapshot = { left: leftOpen, right: rightOpen };
				leftOpen = false;
				rightOpen = false;
			}
			workbenchOpen = true;
			queueMicrotask(() => graphAPI?.reflow());
		};
		if (policy) {
			openPanel();
			return;
		}
		if (options.ensurePolicy === false) {
			openPanel();
			return;
		}
		void loadPolicy().then(openPanel);
	}

	function closeWorkbench() {
		workbenchOpen = false;
		if (sidebarSnapshot) {
			leftOpen = sidebarSnapshot.left;
			rightOpen = sidebarSnapshot.right;
			sidebarSnapshot = null;
		}
		queueMicrotask(() => graphAPI?.reflow());
	}

	function openRawPolicy() {
		if (policy) {
			rawPolicyOpen = true;
			return;
		}
		void loadPolicy().then(() => {
			rawPolicyOpen = true;
		});
	}

	async function simulatePerspective() {
		const validation = validatePerspective(policyPerspective, devices, policyMap);
		if (editBusy || validation.status !== 'valid' || !policy?.hujson) return;
		const selector = validation.selector;
		policyPerspective = selector;
		editBusy = true;
		cloudError = '';
		if (draftHuJSON) {
			await evaluateDraftImpact(draftHuJSON);
			finishPerspectiveSimulation(selector);
			policyGraphViewMode = 'draft';
			editBusy = false;
			return;
		}
		const result = await evaluatePolicyDraft({
			hujson: policy.hujson,
			perspective: selector
		});
		result.match({
			ok: (value) => {
				perspectiveEvaluation = value;
				finishPerspectiveSimulation(selector);
				policyGraphViewMode = draftHuJSON ? 'draft' : 'current';
				const edgeCount = value.unchanged.length + value.added.length + value.changed.length;
				editStatus = `${selector} can reach ${edgeCount} saved edge${edgeCount === 1 ? '' : 's'}.`;
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
		scenario = null;
		saveScenario(null);
		policyGraphViewMode = draftHuJSON ? 'draft' : 'current';
		graphMode = 'focused';
		selectedDevice = rootDevice;
		selectedEdge = undefined;
		if (rootDevice) graphAPI?.selectDevice(rootDevice);
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
		draftChanges = [];
		draftEvaluation = undefined;
		draftEvaluationPerspective = '';
		draftValid = null;
		editStatus = '';
		policyGraphViewMode = 'current';
		syncScenarioModes();
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
				draftChanges = [];
				draftEvaluation = undefined;
				draftEvaluationPerspective = '';
				draftValid = null;
				perspectiveEvaluation = undefined;
				simulatedPerspective = '';
				scenario = null;
				saveScenario(null);
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

	function seedBuilder(role: 'source' | 'destination') {
		const edge = selectedEdge;
		const edgeRef = edge?.policyRefs?.[0];
		const device = edge
			? devices.find((item) => item.id === (role === 'source' ? edge.from : edge.to))
			: selectedDevice;
		const deviceSelector = selectorForDevice(device, role);

		const sources =
			role === 'source'
				? edgeRef?.src && !isWildcardSelector(edgeRef.src)
					? edgeRef.src
					: deviceSelector
				: editSeed.sources;
		const refDestination = stripDestinationPorts(edgeRef?.dst ?? '');
		const destinations =
			role === 'destination'
				? refDestination && !isWildcardSelector(refDestination)
					? refDestination
					: deviceSelector
				: editSeed.destinations;

		editSeed = {
			sources,
			destinations,
			ports: edge?.ports?.join(',') || editSeed.ports || '443'
		};
		openWorkbench({ route: edge?.accessScope === 'ssh' ? 'ssh' : 'general-access' });
	}

	function openPolicyFromLens(section?: string) {
		const route: WorkbenchRoute = section === 'ssh' ? 'ssh' : 'general-access';
		openWorkbench({ route });
	}

	$effect(() => {
		if (!scenario) return;
		const mode = policyGraphViewMode;
		const graph = graphMode;
		if (scenario.policyMode === mode && scenario.graphMode === graph) return;
		scenario = { ...scenario, policyMode: mode, graphMode: graph };
		saveScenario(scenario);
	});

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

		const restored = loadScenario();
		if (cloudStatus.authenticated) {
			await loadPolicy();
			if (restored) {
				scenario = restored;
				policyPerspective = restored.sourceSelector;
				policyGraphViewMode = restored.policyMode;
				graphMode = restored.graphMode;
				await simulatePerspective();
			}
		}

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
					<button class="btn-primary" type="button" onclick={() => openWorkbench()}>
						Access controls
					</button>
					<button class="btn-secondary" type="button" onclick={openRawPolicy}>Raw HuJSON</button>
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
				onViewAsOwner={(owner) => void applyPerspectiveFromSelector(owner)}
			/>

			<div class="flex min-h-0 min-w-0 flex-1">
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
								>{activeScenario ? ' simulated' : ' focus'}</span
							>
						{/if}
					</div>
					{#if cloudStatus.authenticated}
						<PerspectiveBar
							bind:perspective={policyPerspective}
							{devices}
							{policyMap}
							bind:graphViewMode={policyGraphViewMode}
							bind:graphMode
							bind:showGhostEdges
							hasDraft={Boolean(draftHuJSON)}
							hasPerspectivePreview={Boolean(
								activePerspectiveEvaluation || (activePerspective && activeDraftEvaluation)
							)}
							sourceCount={scenarioSourceCount}
							reachableCount={perspectiveReachableCount}
							scenarioActive={activeScenario}
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
						devices={graphDevices}
						{edges}
						visibleDevices={graphDevices}
						{visibleEdges}
						{graphMode}
						bind:selectedDevice
						bind:selectedEdge
						{showLabels}
						{cloudStatus}
						{colorBy}
						rootDevice={graphRootDevice}
						scenarioSourceIds={scenarioSourceIDs}
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

					<RawPolicyPanel
						bind:open={rawPolicyOpen}
						{policy}
						{draftHuJSON}
						onClose={() => (rawPolicyOpen = false)}
					/>
					<DraftTray
						{draftEvaluation}
						{draftRuleText}
						{draftChanges}
						{draftDiffLines}
						{draftValid}
						{editBusy}
						{editStatus}
						onValidate={validateDraft}
						onSave={saveDraft}
						onDiscard={discardDraft}
						onOpenAdvanced={openRawPolicy}
						onOpenWorkbench={() => openWorkbench()}
					/>
				</section>

				<PolicyWorkbench
					bind:open={workbenchOpen}
					bind:route={workbenchRoute}
					{policy}
					{policyMap}
					bind:search={policySearch}
					{draftHuJSON}
					scenarioSource={activePerspective}
					bind:editSeed
					{editBusy}
					{activeScenarioLabel}
					onClose={closeWorkbench}
					onMutate={applyDraftMutation}
					onViewAs={(selector) => void applyPerspectiveFromSelector(selector)}
				/>
			</div>

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
				{policyMap}
				onSeedSource={() => seedBuilder('source')}
				onSeedDestination={() => seedBuilder('destination')}
				onOpenPolicy={openPolicyFromLens}
				onOpenPolicySection={(section) => openPolicyFromLens(section)}
				scenarioSourceIds={scenarioSourceIDs}
				onViewAsOwner={(owner) => void applyPerspectiveFromSelector(owner)}
				onViewAsTag={(tag) => void applyPerspectiveFromSelector(tag)}
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
