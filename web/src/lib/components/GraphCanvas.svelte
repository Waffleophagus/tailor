<script lang="ts">
	import { onDestroy } from 'svelte';
	import type { Attachment } from 'svelte/attachments';
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
		tagColorMap = new Map<string, string>(),
		ownerColorMap = new Map<string, string>(),
		rootDevice,
		scenarioSourceIds,
		onNodeSelect = (device: Device) => {
			selectedDevice = device;
		},
		onEdgeSelect = (edge?: RenderEdge) => {
			selectedEdge = edge;
		},
		onSelectionSettled,
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
		tagColorMap: ReadonlyMap<string, string>;
		ownerColorMap: ReadonlyMap<string, string>;
		rootDevice?: Device;
		scenarioSourceIds?: ReadonlySet<string>;
		onNodeSelect?: (device: Device) => void;
		onEdgeSelect?: (edge?: RenderEdge) => void;
		onSelectionSettled?: () => void;
		onReady?: (api: {
			fit: () => void;
			zoom: (delta: number) => void;
			reflow: () => void;
			selectDevice: (device: Device) => void;
		}) => void;
	} = $props();

	let graphEl = $state<HTMLDivElement | undefined>(undefined);
	let engine = $state<ReturnType<typeof createEngine> | undefined>(undefined);
	let libsLoaded = $state(false);

	const graphContainer: Attachment<HTMLDivElement> = (element) => {
		graphEl = element;
		return () => {
			graphEl = undefined;
		};
	};

	$effect(() => {
		if (!graphEl || devices.length === 0 || libsLoaded) return;
		void (async () => {
			await loadLibs();
			libsLoaded = true;
			engine = createEngine({
				container: graphEl!,
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
				tagColorMap,
				ownerColorMap,
				rootDevice,
				scenarioSourceIds,
				onNodeSelect,
				onEdgeSelect,
				onSelectionSettled
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
			tagColorMap,
			ownerColorMap,
			rootDevice,
			scenarioSourceIds
		});
	});

	$effect(() => {
		if (!graphEl || !engine) return;
		let skipInitial = true;
		const observer = new ResizeObserver(() => {
			if (skipInitial) {
				skipInitial = false;
				return;
			}
			engine?.resize();
		});
		observer.observe(graphEl);
		return () => observer.disconnect();
	});

	onDestroy(() => {
		engine?.destroy();
	});
</script>

<div {@attach graphContainer} class="graph-canvas"></div>
