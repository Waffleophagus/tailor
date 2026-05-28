<script lang="ts">
	import { onDestroy } from 'svelte';
	import { loadLibs, createEngine } from '../graph/engine';
	import type { ColorBy, RenderEdge } from '../graph/engine';
	import type { Device, Edge } from '../api/schemas';
	import type { CloudAuthStatusResponse } from '../api/schemas';

	let {
		devices = [],
		edges = [],
		visibleDevices = [],
		visibleEdges = [],
		graphMode = 'focused',
		selectedDevice = $bindable<Device | undefined>(undefined),
		selectedEdge = $bindable<RenderEdge | undefined>(undefined),
		showLabels = true,
		cloudStatus = { authenticated: false, hasPolicy: false } as CloudAuthStatusResponse,
		colorBy = 'status' as ColorBy,
		rootDevice,
		onNodeSelect = (device: Device) => {
			selectedDevice = device;
		},
		onEdgeSelect = (edge?: RenderEdge) => {
			selectedEdge = edge;
		},
		onReady
	}: {
		devices: Device[];
		edges: Edge[];
		visibleDevices: Device[];
		visibleEdges: RenderEdge[];
		graphMode: 'focused' | 'all';
		selectedDevice?: Device;
		selectedEdge?: RenderEdge;
		showLabels: boolean;
		cloudStatus: CloudAuthStatusResponse;
		colorBy: ColorBy;
		rootDevice?: Device;
		onNodeSelect?: (device: Device) => void;
		onEdgeSelect?: (edge?: RenderEdge) => void;
		onReady?: (api: {
			fit: () => void;
			zoom: (delta: number) => void;
			reflow: () => void;
			selectDevice: (device: Device) => void;
		}) => void;
	} = $props();

	let graphEl: HTMLDivElement;
	let engine = $state<ReturnType<typeof createEngine> | undefined>(undefined);
	let libsLoaded = $state(false);

	$effect(() => {
		if (!graphEl || devices.length === 0 || libsLoaded) return;
		void (async () => {
			await loadLibs();
			libsLoaded = true;
			engine = createEngine({
				container: graphEl,
				devices,
				edges,
				visibleDevices,
				visibleEdges,
				graphMode,
				selectedDevice,
				selectedEdge,
				showLabels,
				cloudStatus,
				colorBy,
				rootDevice,
				onNodeSelect,
				onEdgeSelect
			});
			onReady?.({
				fit: () => engine!.fit(),
				zoom: (delta: number) => engine!.zoom(delta),
				reflow: () => engine!.reflow(),
				selectDevice: (device: Device) => engine!.selectDevice(device)
			});
		})();
	});

	$effect(() => {
		if (!libsLoaded || !engine) return;
		engine.sync({
			devices,
			edges,
			visibleDevices,
			visibleEdges,
			graphMode,
			selectedDevice,
			selectedEdge,
			showLabels,
			cloudStatus,
			colorBy,
			rootDevice
		});
	});

	onDestroy(() => {
		engine?.destroy();
	});
</script>

<div bind:this={graphEl} class="graph-canvas"></div>
