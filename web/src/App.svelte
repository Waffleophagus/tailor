<script lang="ts">
	import { onDestroy, onMount } from 'svelte';
	import {
		authenticateCloud,
		discardStagedPolicyDraft,
		evaluatePolicyDraft,
		fetchCloudStatus,
		fetchPolicy,
		fetchStagedPolicyDraft,
		fetchStagedPolicyDrafts,
		saveValidatedPolicyDraft,
		stagePolicyDraft,
		validatePolicyDraft
	} from './lib/api/cloud';
	import { fetchHealth } from './lib/api/health';
	import type {
		CloudAuthStatusResponse,
		Device,
		Edge,
		LocalAPIStatusResponse,
		TailscaleSetupInfo,
		PolicyEvaluateDraftResponse,
		PolicyResponse,
		StagedDraft
	} from './lib/api/schemas';
	import { fetchTopology } from './lib/api/topology';
	import { connectTopologySocket } from './lib/api/topologySocket';
	import type { RenderEdge } from './lib/graph/engine';
	import {
		collapseDevicesByTag,
		DEFAULT_TAG_COLLAPSE_RULES,
		isAggregateDeviceId,
		rewriteEdgesForCollapsedDevices,
		tagFromAggregateId
	} from './lib/graph/collapse-devices';
	import { resolveGraphLayoutRoot } from './lib/graph/graph-layout-root';
	import { resolveBaseGraphEdges } from './lib/graph/resolve-graph-edges';
	import AuthDialog from './lib/components/AuthDialog.svelte';
	import DeviceDetailsPanel from './lib/components/DeviceDetailsPanel.svelte';
	import DeviceFiltersPanel from './lib/components/DeviceFiltersPanel.svelte';
	import GraphCanvas from './lib/components/GraphCanvas.svelte';
	import GraphLegend from './lib/components/GraphLegend.svelte';
	import MobileGraphBar from './lib/components/MobileGraphBar.svelte';
	import MobileSheet from './lib/components/MobileSheet.svelte';
	import PolicyEditorPanel from './lib/components/PolicyEditorPanel.svelte';
	import SidebarLeft from './lib/components/SidebarLeft.svelte';
	import SidebarRight from './lib/components/SidebarRight.svelte';
	import SidebarToggleButton from './lib/components/SidebarToggleButton.svelte';
	import { viewport } from './lib/ui/viewport.svelte';

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
	let policyEvaluation = $state<PolicyEvaluateDraftResponse | undefined>();
	let previewEvaluation = $state<PolicyEvaluateDraftResponse | undefined>();
	let editorHuJSON = $state('');
	let validatedHuJSON = $state('');
	let editorOpen = $state(false);
	let editorValid = $state<boolean | null>(null);
	let editorBusy = $state(false);
	let editorStatus = $state('');
	let editorErrors = $state<string[]>([]);
	let stagedDrafts = $state<StagedDraft[]>([]);
	let selectedStagedDraft = $state<StagedDraft | undefined>();
	let stagedBusy = $state(false);
	let phase2Open = $state(false);
	let localApiError = $state<LocalAPIStatusResponse | Error | undefined>();
	let tailscaleSetup = $state<TailscaleSetupInfo | undefined>();
	let cloudBusy = $state(false);
	let showOffline = $state(true);
	let showSubnetRouters = $state(true);
	let collapseTaggedFleets = $state(true);
	let showTailnet = $state(false);
	let showLabels = $state(false);
	let graphMode = $state<'focused' | 'all'>('focused');
	let selectedTag = $state('all');
	let selectedOwner = $state('all');
	let selectedOS = $state('all');
	let colorBy = $state<'status' | 'tag' | 'owner' | 'os'>('status');
	let leftOpen = $state(true);
	let rightOpen = $state(true);
	let mobileSheet = $state<'filters' | 'details' | 'legend' | null>(null);

	let graphAPI:
		| {
				fit: () => void;
				zoom: (delta: number) => void;
				reflow: () => void;
				selectDevice: (device: Device) => void;
		  }
		| undefined;
	let disconnectTopologySocket: (() => void) | undefined;
	let topologyEvalTimer: number | undefined;

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
	const editorDirty = $derived(Boolean(policy && editorHuJSON !== policy.hujson));
	const validationStale = $derived(validatedHuJSON !== '' && editorHuJSON !== validatedHuJSON);
	const effectiveValid = $derived(validationStale ? null : editorValid);
	const effectiveErrors = $derived(validationStale ? [] : editorErrors);
	const effectivePreviewEvaluation = $derived(validationStale ? undefined : previewEvaluation);
	const stagedPreviewActive = $derived(Boolean(selectedStagedDraft?.valid && previewEvaluation));
	const hasValidatedPending = $derived(
		!validationStale && editorValid === true && validatedHuJSON !== ''
	);
	const hasPendingDraft = $derived(editorDirty || hasValidatedPending);
	const mcpStagedDrafts = $derived(stagedDrafts.filter((draft) => draft.source === 'mcp'));
	const deviceCollapse = $derived(
		collapseDevicesByTag(visibleDevices, {
			enabled: collapseTaggedFleets,
			rules: DEFAULT_TAG_COLLAPSE_RULES
		})
	);
	const graphDevices = $derived(deviceCollapse.graphDevices);
	const listDevices = $derived(deviceCollapse.listDevices);
	const aggregateMeta = $derived(deviceCollapse.aggregateMeta);
	const graphVisibleDeviceIDs = $derived(new Set(graphDevices.map((device) => device.id)));
	const graphRootDevice = $derived(
		resolveGraphLayoutRoot(selectedDevice, rootDevice, graphVisibleDeviceIDs)
	);
	const visibleEdges = $derived(graphEdges());
	const visibleDeviceIDs = $derived(new Set(visibleDevices.map((device) => device.id)));
	const listOnlineCount = $derived(listDevices.filter((device) => device.online).length);
	const graphOnlineCount = $derived(graphDevices.filter((device) => device.online).length);

	function graphEdges(): RenderEdge[] {
		const policyRendered = resolveBaseGraphEdges({
			cloudAuthenticated: cloudStatus.authenticated,
			topologyEdges: edges,
			previewEvaluation: effectivePreviewEvaluation,
			policyEvaluation,
			editorOpen,
			editorDirty,
			hasValidatedPending,
			stagedPreviewActive
		});

		let rendered: RenderEdge[];
		if (policyRendered) {
			rendered = policyRendered.filter(
				(edge) => visibleDeviceIDs.has(edge.from) || visibleDeviceIDs.has(edge.to)
			);
		} else {
			const root = graphRootDevice;
			if (!root || !graphVisibleDeviceIDs.has(root.id) || !root.online) {
				rendered = [];
			} else {
				rendered = graphDevices
					.filter((device) => device.id !== root.id && device.online)
					.map((device) => ({
						id: `local:${root.id}:${device.id}`,
						from: root.id,
						to: device.id,
						kind: 'local'
					}));
			}
		}

		if (collapseTaggedFleets && deviceCollapse.aggregateMeta.size > 0) {
			rendered = rewriteEdgesForCollapsedDevices(rendered, deviceCollapse.graphIdForDevice);
		}

		if (graphMode === 'all') {
			return rendered;
		}
		const focus = graphRootDevice?.id;
		if (!focus) {
			return [];
		}
		return rendered.filter((edge) => edge.from === focus || edge.to === focus);
	}

	$effect(() => {
		const selected = selectedDevice;
		if (!selected) return;

		if (collapseTaggedFleets) {
			if (isAggregateDeviceId(selected.id)) return;
			const mapped = deviceCollapse.graphIdForDevice.get(selected.id);
			if (!mapped || mapped === selected.id) return;
			const aggregate = graphDevices.find((device) => device.id === mapped);
			if (aggregate) selectedDevice = aggregate;
			return;
		}

		if (!isAggregateDeviceId(selected.id)) return;
		const tag = tagFromAggregateId(selected.id);
		const member = tag ? devices.find((device) => device.tags.includes(tag)) : undefined;
		selectedDevice = member ?? devices[0];
	});

	function scheduleTopologyPolicySync() {
		const savedPolicy = policy;
		if (!cloudStatus.authenticated || !savedPolicy || editorDirty || hasValidatedPending) {
			return;
		}
		if (topologyEvalTimer !== undefined) {
			window.clearTimeout(topologyEvalTimer);
		}
		topologyEvalTimer = window.setTimeout(() => {
			topologyEvalTimer = undefined;
			void evaluatePolicy(savedPolicy.hujson);
		}, 300);
	}

	function unique(values: string[]) {
		return [...new Set(values)].sort((a, b) => a.localeCompare(b));
	}

	function chooseDevice(device: Device) {
		selectedEdge = undefined;
		selectedDevice = device;
		graphAPI?.selectDevice(device);
	}

	function openMobileSheet(sheet: 'filters' | 'details' | 'legend') {
		mobileSheet = mobileSheet === sheet ? null : sheet;
	}

	const hasMobileSelection = $derived(Boolean(selectedDevice || selectedEdge));

	$effect(() => {
		if (!viewport.isMobile) return;
		leftOpen = false;
		rightOpen = false;
		if (editorOpen) editorOpen = false;
	});

	function localApiErrorMessage(error: LocalAPIStatusResponse | Error | undefined) {
		if (!error) return '';
		if ('available' in error) {
			return error.error ?? `Unable to reach ${error.localApiEndpoint}`;
		}
		return error.message;
	}

	function resetEditorFromPolicy() {
		if (!policy) return;
		editorHuJSON = policy.hujson;
		validatedHuJSON = '';
		editorValid = null;
		editorStatus = '';
		editorErrors = [];
		previewEvaluation = undefined;
		selectedStagedDraft = undefined;
	}

	async function evaluatePolicy(hujson: string, preview = false) {
		const result = await evaluatePolicyDraft({ hujson });
		result.match({
			ok: (value) => {
				if (preview) {
					previewEvaluation = value;
					return;
				}
				policyEvaluation = value;
				previewEvaluation = undefined;
			},
			err: (error) => {
				if (preview) {
					previewEvaluation = undefined;
					editorStatus = 'Validated, but graph preview is unavailable.';
				}
				cloudError = error.message;
			}
		});
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
				stagedDrafts = topology.value.stagedDrafts ?? stagedDrafts;
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
		const result = await fetchPolicy();
		result.match({
			ok: async (value) => {
				policy = value;
				resetEditorFromPolicy();
				cloudError = '';
				await evaluatePolicy(value.hujson);
				await loadStagedDrafts();
			},
			err: async (error) => {
				cloudError = error.message;
			}
		});
	}

	async function loadStagedDrafts() {
		if (!cloudStatus.authenticated) return;
		const result = await fetchStagedPolicyDrafts();
		result.match({
			ok: (value) => {
				setStagedDrafts(value.drafts);
			},
			err: (error) => {
				cloudError = error.message;
			}
		});
	}

	function setStagedDrafts(drafts: StagedDraft[]) {
		stagedDrafts = drafts;
		if (!selectedStagedDraft) return;
		const found = drafts.find((draft) => draft.id === selectedStagedDraft?.id);
		if (selectedStagedDraft.hujson && found && !found.hujson) {
			found.hujson = selectedStagedDraft.hujson;
		}
		selectedStagedDraft = found;
	}

	function openPolicyEditor() {
		const open = () => {
			if (policy && !hasPendingDraft) {
				resetEditorFromPolicy();
			}
			editorOpen = true;
		};
		if (policy) {
			open();
			return;
		}
		void loadPolicy().then(open);
	}

	function closePolicyEditor() {
		editorOpen = false;
		if (!hasValidatedPending) {
			previewEvaluation = undefined;
		}
	}

	async function validateEditor() {
		if (editorBusy || !editorDirty) return;
		editorBusy = true;
		cloudError = '';
		editorErrors = [];
		const result = await validatePolicyDraft(editorHuJSON);
		await result.match({
			ok: async (value) => {
				editorValid = value.valid;
				if (value.valid) {
					validatedHuJSON = editorHuJSON;
					editorStatus = 'Policy validated. Preview updated on the graph.';
					await evaluatePolicy(editorHuJSON, true);
				} else {
					validatedHuJSON = '';
					editorStatus = 'Fix validation errors before saving.';
					editorErrors = value.errors ?? ['Policy failed validation.'];
					previewEvaluation = undefined;
				}
			},
			err: async (error) => {
				editorValid = false;
				validatedHuJSON = '';
				editorErrors = [error.message];
				editorStatus = 'Validation failed.';
				previewEvaluation = undefined;
			}
		});
		editorBusy = false;
	}

	function discardEditorChanges() {
		resetEditorFromPolicy();
	}

	async function loadStagedDraft(draft: StagedDraft) {
		if (!policy || stagedBusy) return;
		stagedBusy = true;
		const result = await fetchStagedPolicyDraft(draft.id);
		result.match({
			ok: (value) => {
				selectedStagedDraft = value.draft;
				editorHuJSON = value.draft.hujson ?? '';
				validatedHuJSON = value.draft.hujson ?? '';
				editorValid = value.draft.valid;
				editorErrors = value.draft.errors ?? [];
				previewEvaluation = value.draft.evaluation;
				editorStatus = `${value.draft.source.toUpperCase()} staged draft loaded for review.`;
				editorOpen = true;
			},
			err: (error) => {
				cloudError = error.message;
			}
		});
		stagedBusy = false;
	}

	async function discardStagedDraft(draft: StagedDraft) {
		if (stagedBusy) return;
		stagedBusy = true;
		const result = await discardStagedPolicyDraft(draft.id);
		result.match({
			ok: () => {
				stagedDrafts = stagedDrafts.filter((item) => item.id !== draft.id);
				if (selectedStagedDraft?.id === draft.id) {
					resetEditorFromPolicy();
				}
			},
			err: (error) => {
				cloudError = error.message;
			}
		});
		stagedBusy = false;
	}

	async function saveEditorPolicy() {
		if (editorBusy || !hasValidatedPending) return;
		editorBusy = true;
		cloudError = '';
		let result: Awaited<ReturnType<typeof saveValidatedPolicyDraft>> | undefined;
		if (selectedStagedDraft?.hujson === validatedHuJSON) {
			result = await saveValidatedPolicyDraft({
				draftId: selectedStagedDraft.id,
				draftHash: selectedStagedDraft.draftHash
			});
		} else {
			const staged = await stagePolicyDraft({
				hujson: validatedHuJSON,
				source: 'ui',
				summary: 'Staged from policy editor'
			});
			await staged.match({
				ok: async (value) => {
					result = await saveValidatedPolicyDraft({
						draftId: value.draft.id,
						draftHash: value.draft.draftHash
					});
				},
				err: async (error) => {
					cloudError = error.message;
				}
			});
		}
		if (!result) {
			editorBusy = false;
			return;
		}
		await result.match({
			ok: async (value) => {
				const savedDraftID = selectedStagedDraft?.id;
				policy = { tailnet: value.tailnet, hujson: value.hujson };
				resetEditorFromPolicy();
				if (savedDraftID) {
					stagedDrafts = stagedDrafts.filter((draft) => draft.id !== savedDraftID);
				}
				editorOpen = false;
				editorStatus = 'Policy saved.';
				await evaluatePolicy(value.hujson);
				await loadStagedDrafts();
			},
			err: async (error) => {
				cloudError = error.message;
			}
		});
		editorBusy = false;
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

	function deviceShortLabel(device: Device | undefined): string {
		if (!device) return '';
		const raw = device.name || device.ip || '';
		const short = raw.includes('.') ? raw.split('.')[0] : raw;
		if (short.length <= 20) return short;
		return `${short.slice(0, 18)}…`;
	}

	onMount(() => {
		const unbindViewport = viewport.bind();

		void (async () => {
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

			if (cloudStatus.authenticated) {
				await loadPolicy();
			}

			disconnectTopologySocket = connectTopologySocket({
				onSnapshot: (value) => {
					tailscaleSetup = value.setup ?? undefined;
					if (value.setup?.required) {
						apiStatus = 'setup required';
						localApiError = undefined;
					} else {
						apiStatus = 'connected';
						localApiError = undefined;
					}
					devices = value.devices;
					edges = value.edges;
					tailnetName = value.tailnet;
					if (value.stagedDrafts) {
						setStagedDrafts(value.stagedDrafts);
					}
					selectedDevice = selectedDevice
						? (value.devices.find((device) => device.id === selectedDevice?.id) ?? value.devices[0])
						: value.devices[0];
					scheduleTopologyPolicySync();
				},
				onUnavailable: (status) => {
					tailscaleSetup = status.setup ?? undefined;
					if (status.setup?.required) {
						apiStatus = 'setup required';
						localApiError = undefined;
					} else {
						apiStatus = 'LocalAPI unavailable';
						localApiError = status;
					}
					devices = [];
					edges = [];
					tailnetName = '';
					selectedDevice = undefined;
				},
				onConnectionState: (state) => {
					if (tailscaleSetup?.required) {
						apiStatus = 'setup required';
						return;
					}
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
		})();

		return () => unbindViewport();
	});

	onDestroy(() => {
		if (topologyEvalTimer !== undefined) {
			window.clearTimeout(topologyEvalTimer);
		}
		disconnectTopologySocket?.();
	});
</script>

<main class="min-h-screen overflow-hidden">
	<section class="grid h-screen grid-rows-[auto_minmax(0,1fr)] overflow-hidden">
		<div
			class="app-header shrink-0 border-b border-base bg-surface px-4 py-3 md:gap-4 md:px-5 md:py-4"
			class:app-header-mobile={viewport.isMobile}
			class:app-header-desktop={!viewport.isMobile}
		>
			<div class="min-w-0 flex-1">
				<p
					class="m-0 text-[0.72rem] font-bold tracking-normal text-secondary uppercase md:text-[0.8rem]"
				>
					Tailnet topology
				</p>
				<h1 class="m-0 text-xl leading-[1.1] md:text-2xl">Tailor</h1>
				{#if viewport.isMobile}
					<p
						class="m-0 mt-0.5 line-clamp-2 text-[0.72rem] leading-snug font-semibold text-secondary"
					>
						Policy editing is available on desktop.
					</p>
				{/if}
				{#if cloudStatus.devMode && !viewport.isMobile}
					<p class="m-0 mt-1 text-[0.78rem] font-bold text-teal">
						Demo tailnet — {cloudStatus.tailnet ?? 'demo.tailor.ts.net'}
					</p>
				{/if}
			</div>
			<div
				class="app-header-actions flex min-w-0 shrink-0 items-center gap-[0.6rem]"
				class:app-header-actions-mobile={viewport.isMobile}
			>
				{#if !viewport.isMobile}
					{#if cloudStatus.authenticated}
						{#if hasValidatedPending}
							<button
								class="btn-save"
								type="button"
								disabled={editorBusy}
								onclick={saveEditorPolicy}
							>
								Save validated policy
							</button>
						{/if}
						<button class="btn-primary" type="button" onclick={openPolicyEditor}>Edit policy</button
						>
					{:else}
						<button class="btn-primary" type="button" onclick={() => (phase2Open = true)}>
							Enable ACL Editing
						</button>
					{/if}
				{/if}
				<div
					class="status-pill flex max-w-full min-w-0 items-center gap-[0.3rem] rounded-full border border-status-border bg-status-bg p-[0.45rem_0.7rem] text-[0.85rem] font-bold text-status-text"
					data-state={apiStatus}
					title={apiStatus}
				>
					<span class="h-2 w-2 shrink-0 rounded-full"></span>
					<span class="truncate">{apiStatus}</span>
				</div>
			</div>
		</div>

		<div class="flex h-full min-h-0">
			{#if !viewport.isMobile}
				<SidebarLeft
					bind:open={leftOpen}
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
					chooseDevice={(device) => chooseDevice(device)}
				/>
			{/if}

			<div class="relative flex min-h-0 min-w-0 flex-1 flex-col">
				<section
					class="graph relative min-h-0 min-w-0 flex-1 overflow-hidden md:min-h-[32rem]"
					class:graph-mobile={viewport.isMobile}
					aria-label="Topology graph"
				>
					{#if !viewport.isMobile}
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
					{/if}

					{#if viewport.isMobile}
						<div class="graph-hud graph-hud-mobile" aria-label="Graph summary">
							<div class="hud-row">
								<span class="hud-chip"
									><strong class="hud-chip-strong">{graphOnlineCount}</strong> online</span
								>
								<span class="hud-chip"
									><strong class="hud-chip-strong">{visibleEdges.length}</strong> links</span
								>
								{#if effectivePreviewEvaluation}
									<span class="hud-chip hud-chip-warn">Preview</span>
								{/if}
							</div>
							{#if cloudStatus.authenticated && graphMode === 'focused' && graphRootDevice}
								<div class="hud-row">
									<span
										class="hud-chip hud-chip-focus"
										title="{graphRootDevice.name || graphRootDevice.ip} focus"
									>
										<strong class="hud-chip-strong">{deviceShortLabel(graphRootDevice)}</strong>
										focus
									</span>
								</div>
							{/if}
						</div>
					{:else}
						<div class="graph-hud" aria-label="Graph summary">
							<span class="hud-chip"
								><strong class="hud-chip-strong">{graphOnlineCount}</strong> online</span
							>
							<span class="hud-chip"
								><strong class="hud-chip-strong">{visibleEdges.length}</strong> links</span
							>
							{#if effectivePreviewEvaluation}
								<span class="hud-chip hud-chip-warn">Preview</span>
							{/if}
							{#if cloudStatus.authenticated && graphMode === 'focused' && graphRootDevice}
								<span class="hud-chip"
									><strong class="hud-chip-strong"
										>{graphRootDevice.name || graphRootDevice.ip}</strong
									> focus</span
								>
							{/if}
							{#if cloudStatus.authenticated}
								<div class="mode-toggle">
									{#each ['focused', 'all'] as mode (mode)}
										<button
											type="button"
											class="mode-button"
											data-active={graphMode === mode}
											onclick={() => (graphMode = mode as 'focused' | 'all')}
										>
											{mode === 'focused' ? 'Focused' : 'All'}
										</button>
									{/each}
								</div>
							{/if}
							<button
								type="button"
								title="Zoom in"
								onclick={() => graphAPI?.zoom(1.2)}
								class="graph-control">+</button
							>
							<button
								type="button"
								title="Zoom out"
								onclick={() => graphAPI?.zoom(0.8)}
								class="graph-control">-</button
							>
							<button
								type="button"
								title="Fit to view"
								onclick={() => graphAPI?.fit()}
								class="graph-control">⌖</button
							>
							<button
								type="button"
								title="Reflow layout"
								onclick={() => graphAPI?.reflow()}
								class="graph-control">↻</button
							>
						</div>
					{/if}

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
						onReady={(api) => (graphAPI = api)}
					/>

					{#if !viewport.isMobile}
						<GraphLegend
							{colorBy}
							authenticated={cloudStatus.authenticated}
							bind:graphMode
							{tagOptions}
							{ownerOptions}
							{osOptions}
						/>
					{/if}

					{#if tailscaleSetup?.required}
						<div class="tailscale-setup" role="status">
							<h2 class="mb-[0.35rem] font-extrabold text-base text-primary">
								Connect Tailscale in Docker
							</h2>
							<p class="mb-[0.65rem] text-[0.88rem] text-secondary">
								Tailor needs Tailscale configured in the container before it can show your tailnet.
								Use one of these options:
							</p>
							<ul class="m-0 list-none space-y-[0.55rem] p-0">
								{#each tailscaleSetup.hints ?? [] as hint (hint.id)}
									<li class="setup-hint">
										<span class="setup-hint-label"
											>{hint.id === 'auth-key' ? 'Recommended' : 'Alternative'}</span
										>
										<p class="mb-0 text-[0.88rem] leading-snug text-primary">{hint.message}</p>
									</li>
								{/each}
							</ul>
						</div>
					{:else if localApiError}
						<div class="local-api-error">
							<h2 class="mb-[0.4rem] text-base">Connect to Tailscale</h2>
							<p class="mb-0 wrap-anywhere">{localApiErrorMessage(localApiError)}</p>
						</div>
					{/if}

					{#if cloudError}
						<div class="cloud-error" class:cloud-error-mobile={viewport.isMobile} role="alert">
							{cloudError}
						</div>
					{/if}

					{#if !viewport.isMobile && mcpStagedDrafts.length > 0}
						<div class="staged-drafts" aria-label="MCP staged drafts">
							<div class="staged-drafts-header">
								<div>
									<p class="staged-eyebrow">MCP staged draft available</p>
									<h2 class="staged-title">{mcpStagedDrafts.length} pending review</h2>
								</div>
								<button
									type="button"
									class="staged-refresh"
									onclick={loadStagedDrafts}
									disabled={stagedBusy}
								>
									Refresh
								</button>
							</div>
							<div class="staged-list">
								{#each mcpStagedDrafts as draft (draft.id)}
									<article class="staged-item">
										<div class="staged-item-main">
											<div class="staged-row">
												<span class="staged-source">{draft.source}</span>
												<span class:staged-valid={draft.valid} class:staged-invalid={!draft.valid}>
													{draft.valid ? 'Validated' : 'Invalid'}
												</span>
											</div>
											<p class="staged-summary">
												{draft.summary || 'Policy draft staged for review.'}
											</p>
											<p class="staged-meta">
												{draft.evaluation.added.length} added · {draft.evaluation.removed.length} removed
												·
												{draft.evaluation.changed.length} changed · {draft.evaluation.broadAccess
													.length}
												broad
											</p>
										</div>
										<div class="staged-actions">
											<button
												type="button"
												class="staged-load"
												onclick={() => loadStagedDraft(draft)}
												disabled={stagedBusy || !draft.valid}
											>
												Load
											</button>
											<button
												type="button"
												class="staged-discard"
												onclick={() => discardStagedDraft(draft)}
												disabled={stagedBusy}
											>
												Discard
											</button>
										</div>
									</article>
								{/each}
							</div>
						</div>
					{/if}

					{#if !viewport.isMobile}
						<PolicyEditorPanel
							bind:open={editorOpen}
							{policy}
							bind:editorText={editorHuJSON}
							isDirty={editorDirty}
							valid={effectiveValid}
							busy={editorBusy}
							status={editorStatus}
							errors={effectiveErrors}
							stagedDraft={selectedStagedDraft}
							onValidate={validateEditor}
							onSave={saveEditorPolicy}
							onDiscard={discardEditorChanges}
							onClose={closePolicyEditor}
						/>
					{/if}
				</section>

				{#if viewport.isMobile}
					<MobileGraphBar
						hasSelection={hasMobileSelection}
						cloudAuthenticated={cloudStatus.authenticated}
						bind:graphMode
						activeSheet={mobileSheet}
						onOpenSheet={openMobileSheet}
						onZoomIn={() => graphAPI?.zoom(1.2)}
						onZoomOut={() => graphAPI?.zoom(0.8)}
						onFit={() => graphAPI?.fit()}
					/>
				{/if}
			</div>

			{#if !viewport.isMobile}
				<SidebarRight
					bind:open={rightOpen}
					bind:selectedDevice
					bind:selectedEdge
					devices={graphDevices}
					{aggregateMeta}
					{visibleEdges}
					{colorBy}
				/>
			{/if}
		</div>
	</section>

	{#if viewport.isMobile}
		<MobileSheet
			open={mobileSheet === 'filters'}
			onclose={() => (mobileSheet = null)}
			title="Filters"
		>
			{#if mobileSheet === 'filters'}
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
					chooseDevice={(device) => chooseDevice(device)}
					compact
				/>
			{/if}
		</MobileSheet>

		<MobileSheet
			open={mobileSheet === 'details'}
			onclose={() => (mobileSheet = null)}
			title="Details"
		>
			{#if mobileSheet === 'details'}
				<DeviceDetailsPanel
					bind:selectedDevice
					bind:selectedEdge
					devices={graphDevices}
					{aggregateMeta}
					{visibleEdges}
					bind:colorBy
					showCredit={false}
					compact
				/>
			{/if}
		</MobileSheet>

		<MobileSheet
			open={mobileSheet === 'legend'}
			onclose={() => (mobileSheet = null)}
			title="Legend"
		>
			{#if mobileSheet === 'legend'}
				<GraphLegend
					{colorBy}
					authenticated={cloudStatus.authenticated}
					bind:graphMode
					{tagOptions}
					{ownerOptions}
					{osOptions}
					embedded
				/>
			{/if}
		</MobileSheet>
	{/if}

	{#if !viewport.isMobile}
		<AuthDialog
			bind:open={phase2Open}
			initialTailnet={deriveTailnet()}
			{cloudBusy}
			{cloudError}
			onClose={closePhase2Dialog}
			onSubmit={enableACLEditing}
		/>
	{/if}
</main>

<style>
	@reference "./app.css";
	.btn-primary {
		@apply min-h-[2.35rem] rounded-md border border-panel-accent bg-panel-accent px-3 py-[0.45rem] text-sm font-extrabold text-panel-fg transition-[background-color,border-color,color,transform] duration-[160ms] ease-out hover:-translate-y-px disabled:transform-none disabled:cursor-not-allowed disabled:opacity-[0.58];
	}
	.btn-save {
		@apply min-h-[2.35rem] rounded-md border border-ok bg-ok/10 px-3 py-[0.45rem] text-sm font-extrabold text-ok transition-[background-color,border-color,color,transform] duration-[160ms] ease-out hover:-translate-y-px hover:bg-ok/15 disabled:transform-none disabled:cursor-not-allowed disabled:opacity-[0.58];
	}
	.graph-hud {
		@apply absolute top-3 left-3 z-[2] flex w-fit max-w-[calc(100%-1.5rem)] flex-wrap items-center gap-[0.4rem] rounded-lg border border-graph-border bg-graph-hud-bg p-[0.35rem] shadow-[0_10px_26px_rgb(23_33_38/8%)];
	}
	.hud-chip {
		@apply inline-flex min-h-8 items-baseline gap-[0.3rem] rounded-md bg-graph-dot px-[0.55rem] py-[0.35rem] text-[0.78rem] font-bold whitespace-nowrap text-secondary;
	}
	.hud-chip-strong {
		@apply text-[0.98rem] leading-none text-graph-hud-strong;
	}
	.hud-chip-warn {
		@apply items-center text-warn;
	}
	.mode-toggle {
		@apply inline-flex rounded-md border border-panel-border bg-panel-input p-[0.12rem];
	}
	.mode-button {
		@apply rounded-sm border-0 bg-transparent px-2 py-[0.28rem] text-[0.72rem] font-extrabold text-secondary transition-[background-color,color] duration-[140ms] ease-out;
	}
	.mode-button[data-active='true'] {
		@apply bg-hover text-primary;
	}
	.graph-control {
		@apply grid h-[2.1rem] w-[2.1rem] min-w-[2.1rem] place-items-center rounded-md border border-panel-border bg-panel-weak leading-none font-extrabold text-primary transition-[background-color,border-color,transform] duration-[160ms] ease-out hover:-translate-y-px hover:border-teal hover:bg-hover motion-reduce:transition-none motion-reduce:hover:transform-none;
	}
	.tailscale-setup {
		@apply absolute top-1/2 left-1/2 z-[3] w-[min(32rem,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-teal/35 bg-surface p-4 shadow-[0_10px_30px_rgb(23_33_38/10%)];
	}
	.setup-hint {
		@apply rounded-md border border-panel-border bg-panel-weak p-[0.65rem_0.75rem];
	}
	.setup-hint-label {
		@apply mb-[0.25rem] block text-[0.68rem] font-extrabold tracking-wide text-teal uppercase;
	}
	.local-api-error {
		@apply absolute top-1/2 left-1/2 w-[min(28rem,calc(100%-2rem))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-base bg-surface p-4 shadow-[0_10px_30px_rgb(23_33_38/8%)];
	}
	.cloud-error {
		@apply absolute bottom-3 left-1/2 z-[4] max-w-[min(36rem,calc(100%-2rem))] -translate-x-1/2 rounded-lg border border-danger/30 bg-panel-bg px-3 py-2 text-[0.78rem] font-semibold text-danger shadow-[0_10px_26px_rgb(23_33_38/8%)];
	}
	.cloud-error-mobile {
		bottom: calc(8.75rem + env(safe-area-inset-bottom, 0px));
	}
	.staged-drafts {
		@apply absolute bottom-3 left-[13.5rem] z-[4] grid max-h-[min(24rem,calc(100%-6rem))] w-[min(28rem,calc(100%-15rem))] grid-rows-[auto_minmax(0,1fr)] overflow-hidden rounded-lg border border-panel-border bg-panel-bg shadow-[0_14px_34px_rgb(23_33_38/14%)];
	}
	.staged-drafts-header {
		@apply flex items-center justify-between gap-3 border-b border-panel-strong px-3 py-2.5;
	}
	.staged-eyebrow {
		@apply m-0 text-[0.68rem] font-extrabold tracking-wide text-teal uppercase;
	}
	.staged-title {
		@apply m-0 text-sm font-extrabold text-primary;
	}
	.staged-refresh,
	.staged-load,
	.staged-discard {
		@apply min-h-8 rounded-md border px-2.5 text-[0.76rem] font-extrabold disabled:cursor-not-allowed disabled:opacity-50;
	}
	.staged-refresh {
		@apply border-panel-border bg-panel-weak text-primary;
	}
	.staged-list {
		@apply min-h-0 overflow-auto p-2;
	}
	.staged-item {
		@apply flex items-start justify-between gap-3 border-b border-panel-strong px-1 py-2 last:border-b-0;
	}
	.staged-item-main {
		@apply min-w-0 flex-1;
	}
	.staged-row {
		@apply mb-1 flex flex-wrap items-center gap-1.5;
	}
	.staged-source,
	.staged-valid,
	.staged-invalid {
		@apply rounded-full border px-1.5 py-[0.12rem] text-[0.64rem] font-extrabold uppercase;
	}
	.staged-source {
		@apply border-teal text-teal;
	}
	.staged-valid {
		@apply border-ok text-ok;
	}
	.staged-invalid {
		@apply border-danger text-danger;
	}
	.staged-summary {
		@apply m-0 line-clamp-2 text-[0.8rem] leading-snug font-bold text-primary;
	}
	.staged-meta {
		@apply m-0 mt-1 text-[0.72rem] font-semibold text-secondary;
	}
	.staged-actions {
		@apply flex shrink-0 flex-col gap-1;
	}
	.staged-load {
		@apply border-panel-accent bg-panel-accent text-panel-fg;
	}
	.staged-discard {
		@apply border-danger/35 bg-panel-weak text-danger;
	}
	.graph-mobile {
		padding-bottom: calc(8.75rem + env(safe-area-inset-bottom, 0px));
	}
	.graph-hud-mobile {
		@apply right-3 left-3 z-[4] flex w-auto max-w-none flex-col items-stretch gap-[0.35rem] p-[0.4rem];
	}
	.graph-hud-mobile .hud-row {
		@apply flex min-w-0 flex-wrap items-center gap-[0.35rem];
	}
	.hud-chip-focus {
		@apply max-w-full min-w-0;
	}
	.hud-chip-focus .hud-chip-strong {
		@apply inline-block max-w-[12rem] overflow-hidden align-bottom text-ellipsis whitespace-nowrap;
	}
	.app-header {
		@apply relative z-10;
	}
	.app-header-mobile {
		@apply flex flex-col gap-2;
	}
	.app-header-desktop {
		@apply flex flex-row items-center justify-between gap-3;
	}
	.app-header-actions-mobile {
		@apply w-full;
	}
	.app-header-actions-mobile .status-pill {
		@apply w-full max-w-none justify-center;
	}
</style>
