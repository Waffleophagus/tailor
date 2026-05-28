<script lang="ts">
	import { onDestroy } from 'svelte';
	import { loadLibs, createEngine } from '../graph/engine';
	import type { ColorBy } from '../graph/engine';
	import type { Device, Edge } from '../api/schemas';
	import type { CloudAuthStatusResponse } from '../api/schemas';

	let {
		devices = [],
		edges = [],
		visibleDevices = [],
		visibleEdges = [],
		graphMode = 'focused',
		selectedDevice = $bindable<Device | undefined>(undefined),
		showLabels = true,
		cloudStatus = { authenticated: false, hasPolicy: false } as CloudAuthStatusResponse,
		colorBy = 'status' as ColorBy,
		rootDevice,
		onNodeSelect = (device: Device) => {
			selectedDevice = device;
		},
		onReady
	}: {
		devices: Device[];
		edges: Edge[];
		visibleDevices: Device[];
		visibleEdges: {
			id: string;
			from: string;
			to: string;
			kind: string;
			accessScope?: Edge['accessScope'];
		}[];
		graphMode: 'focused' | 'all';
		selectedDevice?: Device;
		showLabels: boolean;
		cloudStatus: CloudAuthStatusResponse;
		colorBy: ColorBy;
		rootDevice?: Device;
		onNodeSelect?: (device: Device) => void;
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
				showLabels,
				cloudStatus,
				colorBy,
				rootDevice,
				onNodeSelect
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
