import type {
	Core,
	EdgeSingular,
	ElementDefinition,
	NodeSingular,
	StylesheetJson,
	EventObject as CyEventObject
} from 'cytoscape';
import type { SimulationNodeDatum, SimulationLinkDatum } from 'd3-force';
import type { Device, Edge } from '../api/schemas';
import type { CloudAuthStatusResponse } from '../api/schemas';
import { edgeClasses } from './edge-classes';
import { installGraphDebug, uninstallGraphDebug } from './graph-debug';
import { graphEdgeStylesheet } from './style-catalog';

export type ColorBy = 'status' | 'tag' | 'owner' | 'os';

export interface SyncOptions {
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
	scenarioSourceIds?: ReadonlySet<string>;
}

export interface GraphInitOptions extends SyncOptions {
	container: HTMLElement;
	onNodeSelect: (device: Device) => void;
	onEdgeSelect: (edge?: RenderEdge) => void;
}

export type RenderEdgeState = 'added' | 'removed' | 'changed' | 'unchanged' | 'ghost-denied';

export interface RenderEdge {
	id: string;
	from: string;
	to: string;
	kind: string;
	labels?: Edge['labels'];
	protocols?: Edge['protocols'];
	ports?: Edge['ports'];
	accessScope?: Edge['accessScope'];
	policyRefs?: Edge['policyRefs'];
	perspectives?: Edge['perspectives'];
	state?: RenderEdgeState;
}

let cytoscapeMod: typeof import('cytoscape') | undefined;
let d3Mod: typeof import('d3-force') | undefined;

export async function loadLibs() {
	if (!cytoscapeMod) {
		cytoscapeMod =
			(await import('cytoscape')).default ||
			((await import('cytoscape')) as unknown as typeof import('cytoscape'));
	}
	if (!d3Mod) {
		d3Mod = await import('d3-force');
	}
}

interface PhysicsNode extends SimulationNodeDatum {
	id: string;
}

export function createEngine(opts: GraphInitOptions) {
	const { container, onNodeSelect, onEdgeSelect } = opts;

	if (!cytoscapeMod || !d3Mod) {
		throw new Error('loadLibs() must be called first');
	}
	const cytoscape = cytoscapeMod;
	const { forceSimulation, forceLink, forceManyBody } = d3Mod;

	let graph: Core | undefined;
	let cleanupPhysics: (() => void) | undefined;
	let physicsSignature = '';
	let layoutSignature = '';
	let graphDragActive = false;
	let graphDragMoved = false;
	let lastGraphDragEndAt = 0;
	let pendingLayoutSync = false;
	let hoveredNodeID: string | undefined;

	const deviceAngles = new Map<string, number>();
	const lastOnlineState = new Map<string, boolean>();

	let current = opts;

	installGraphDebug(() => ({
		cy: graph,
		visibleEdges: current.visibleEdges,
		selectedEdgeId: current.selectedEdge?.id
	}));

	const osColors: Record<string, string> = {
		windows: '#01A6F0',
		android: '#32DE84',
		linux: '#F4BC00',
		bsd: '#B5010F',
		macOS: '#A2AAAD',
		ios: '#FFFFFF',
		tvos: '#FA6C1B'
	};

	function palette(value: string) {
		const osColor = osColors[value];
		if (osColor) return osColor;
		const colors = ['#438aa1', '#a5663f', '#7c6fb0', '#b0892f', '#5d7f73', '#b45f74', '#5973b0'];
		let hash = 0;
		for (let i = 0; i < value.length; i += 1) {
			hash = (hash + value.charCodeAt(i) * (i + 1)) % colors.length;
		}
		return colors[hash];
	}

	function deviceColor(device: Device) {
		if (current.colorBy === 'status') {
			return device.online ? '#41a86f' : '#9aa7a1';
		}
		const value =
			current.colorBy === 'tag'
				? (device.tags[0] ?? 'untagged')
				: current.colorBy === 'owner'
					? device.owner
					: device.os;
		return palette(value || 'unknown');
	}

	function isScenarioSource(device: Device) {
		return current.scenarioSourceIds?.has(device.id) ?? false;
	}

	function deviceClasses(device: Device) {
		return [
			device.online ? 'online' : 'offline',
			graphRootDevice()?.id === device.id ? 'root' : '',
			isScenarioSource(device) ? 'scenario-source' : '',
			current.selectedDevice?.id === device.id ? 'selected' : '',
			device.subnetRouter ? 'subnet-router' : '',
			current.showLabels ? 'with-labels' : 'hide-labels'
		]
			.filter(Boolean)
			.join(' ');
	}

	function deviceData(device: Device) {
		const scenarioSource = isScenarioSource(device);
		return {
			id: device.id,
			label: device.name || device.ip || device.id,
			color: scenarioSource ? '#5d7f73' : deviceColor(device),
			ringColor: scenarioSource
				? '#2f5f4a'
				: graphRootDevice()?.id === device.id
					? '#163f31'
					: device.online
						? '#1f7a52'
						: '#74857e'
		};
	}

	function graphRootDevice() {
		if (current.cloudStatus.authenticated && current.graphMode === 'focused') {
			return current.selectedDevice ?? current.rootDevice;
		}
		return current.rootDevice;
	}

	function graphDevices() {
		return current.visibleDevices;
	}

	function edgeClassesFor(edge: RenderEdge) {
		return edgeClasses(edge, { selectedEdgeId: current.selectedEdge?.id });
	}

	function clamp(value: number, min: number, max: number) {
		return Math.min(Math.max(value, min), max);
	}

	function normalizeAngle(angle: number) {
		const fullCircle = Math.PI * 2;
		return ((angle % fullCircle) + fullCircle) % fullCircle;
	}

	function wheelAngle(id: string) {
		const existing = deviceAngles.get(id);
		if (existing !== undefined) return existing;
		const angles = Array.from(deviceAngles.values())
			.map(normalizeAngle)
			.sort((a, b) => a - b);
		let angle = -Math.PI / 2;
		if (angles.length === 1) {
			angle = angles[0] + Math.PI;
		} else if (angles.length > 1) {
			let bestStart = angles[angles.length - 1];
			let bestGap = angles[0] + Math.PI * 2 - bestStart;
			for (let i = 0; i < angles.length - 1; i += 1) {
				const gap = angles[i + 1] - angles[i];
				if (gap > bestGap) {
					bestGap = gap;
					bestStart = angles[i];
				}
			}
			angle = bestStart + bestGap / 2;
		}
		deviceAngles.set(id, angle);
		return angle;
	}

	function placeOnRing(
		positions: Map<string, { x: number; y: number }>,
		ringDevices: Device[],
		center: { x: number; y: number },
		radius: number
	) {
		ringDevices.forEach((device) => {
			const angle = wheelAngle(device.id);
			positions.set(device.id, {
				x: center.x + Math.cos(angle) * radius,
				y: center.y + Math.sin(angle) * radius
			});
		});
	}

	function graphPositions() {
		const width = container.clientWidth || 900;
		const height = container.clientHeight || 620;
		const center = { x: width / 2, y: height / 2 };
		const positions = new Map<string, { x: number; y: number }>();
		const devices = graphDevices();
		const rootID =
			graphRootDevice() && devices.some((d) => d.id === graphRootDevice()!.id)
				? graphRootDevice()!.id
				: undefined;
		const onlinePeers = devices.filter((d) => d.id !== rootID && d.online);
		const offlinePeers = devices.filter((d) => d.id !== rootID && !d.online);
		const minDim = Math.min(width, height);
		const onlineRadius = clamp(onlinePeers.length * 18, 150, Math.max(170, minDim * 0.34));
		const offlineRadius = clamp(
			onlineRadius + 92,
			onlineRadius + 72,
			Math.max(onlineRadius + 96, minDim * 0.47)
		);
		if (rootID) positions.set(rootID, center);
		placeOnRing(positions, onlinePeers, center, onlineRadius);
		placeOnRing(positions, offlinePeers, center, offlineRadius);
		return positions;
	}

	function graphSpreadPositionsAfterDrag(draggedID: string) {
		const positions = graphPositions();
		const rootID =
			graphRootDevice() && graphDevices().some((d) => d.id === graphRootDevice()!.id)
				? graphRootDevice()!.id
				: undefined;
		if (draggedID === rootID && rootID) {
			const dropped = graph?.getElementById(rootID).position();
			const planned = positions.get(rootID);
			if (dropped && planned) {
				const offset = { x: dropped.x - planned.x, y: dropped.y - planned.y };
				positions.forEach((pos, id) => {
					if (id !== draggedID) {
						positions.set(id, { x: pos.x + offset.x, y: pos.y + offset.y });
					}
				});
			}
		}
		const draggedPos = graph?.getElementById(draggedID).position();
		if (!draggedPos) return positions;
		positions.forEach((pos, id) => {
			if (id !== draggedID) {
				positions.set(id, separatePosition(pos, draggedPos, minNodeSpacing(id, draggedID)));
			}
		});
		return positions;
	}

	function separatePosition(
		position: { x: number; y: number },
		avoided: { x: number; y: number },
		minDistance: number
	) {
		const dx = position.x - avoided.x;
		const dy = position.y - avoided.y;
		const distance = Math.hypot(dx, dy);
		if (distance >= minDistance) return position;
		const angle = distance > 0.1 ? Math.atan2(dy, dx) : -Math.PI / 2;
		return {
			x: avoided.x + Math.cos(angle) * minDistance,
			y: avoided.y + Math.sin(angle) * minDistance
		};
	}

	function nodeRadius(node: NodeSingular | undefined) {
		if (!node?.length) return 28;
		const w = Number(node.style('width'));
		const h = Number(node.style('height'));
		return Math.max(Number.isFinite(w) ? w : 56, Number.isFinite(h) ? h : 56) / 2;
	}

	function minNodeSpacing(nodeID: string, avoidedID: string) {
		return (
			nodeRadius(graph?.getElementById(nodeID)) + nodeRadius(graph?.getElementById(avoidedID)) + 24
		);
	}

	function graphLayoutSignature() {
		return [
			graphRootDevice()?.id ?? '',
			graphDevices()
				.map((d) => d.id)
				.sort()
				.join(','),
			current.visibleEdges
				.map((e) => e.id)
				.sort()
				.join(','),
			container.clientWidth ?? 0,
			container.clientHeight ?? 0
		].join('|');
	}

	function prefersReducedMotion() {
		return (
			typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
		);
	}

	function graphElements(positions: Map<string, { x: number; y: number }>) {
		const elements: ElementDefinition[] = [
			...graphDevices().map((device) => ({
				classes: deviceClasses(device),
				data: deviceData(device),
				position: positions.get(device.id)
			})),
			...current.visibleEdges.map((edge) => ({
				classes: edgeClassesFor(edge),
				data: { id: edge.id, source: edge.from, target: edge.to, label: edgeLabel(edge) }
			}))
		];
		return elements;
	}

	function edgeLabel(edge: RenderEdge) {
		const ports = edge.ports?.length ? edge.ports.join(',') : '';
		if (edge.accessScope === 'broad') return 'all ports';
		if (edge.accessScope === 'ssh') return 'ssh';
		if (edge.accessScope === 'http') return 'http';
		return ports;
	}

	function graphLayoutOptions() {
		const animate = !prefersReducedMotion();
		return {
			name: 'preset' as const,
			animate,
			animationDuration: animate ? 520 : 0,
			animationEasing: 'ease-out-cubic',
			fit: false,
			padding: 56
		};
	}

	function rememberOnlineState() {
		for (const device of current.devices) {
			lastOnlineState.set(device.id, device.online);
		}
	}

	function updateGraphSelection() {
		if (!graph) return;
		const selectedID = current.selectedDevice?.id;
		graph.nodes().forEach((node) => {
			node.toggleClass('selected', selectedID === node.id());
		});
		const selectedEdgeID = current.selectedEdge?.id;
		graph.edges().forEach((edge) => {
			const renderEdge = current.visibleEdges.find((candidate) => candidate.id === edge.id());
			if (renderEdge) edge.classes(edgeClassesFor(renderEdge));
			edge.toggleClass('selected', selectedEdgeID === edge.id());
		});
	}

	function applyCurrentGraphFocus() {
		if (!graph) return;
		if (current.selectedEdge?.id) {
			const edge = graph.getElementById(current.selectedEdge.id);
			if (edge.length) {
				applyEdgeFocus(edge as EdgeSingular);
				return;
			}
		}
		const focusID = hoveredNodeID ?? current.selectedDevice?.id;
		if (!focusID) {
			clearGraphFocus();
			return;
		}
		const node = graph.getElementById(focusID);
		if (!node.length) {
			clearGraphFocus();
			return;
		}
		applyGraphFocus(node);
	}

	function applyGraphFocus(node: NodeSingular) {
		if (!graph) return;
		const neighborhood = node.closedNeighborhood();
		graph.elements().removeClass('dim focused');
		graph.elements().difference(neighborhood).addClass('dim');
		neighborhood.addClass('focused');
	}

	function applyEdgeFocus(edge: EdgeSingular) {
		if (!graph) return;
		const neighborhood = edge.connectedNodes().union(edge);
		graph.elements().removeClass('dim focused');
		graph.elements().difference(neighborhood).addClass('dim');
		neighborhood.addClass('focused');
	}

	function clearGraphFocus() {
		graph?.elements().removeClass('dim focused');
	}

	function pulseNode(node: NodeSingular) {
		if (prefersReducedMotion()) return;
		node
			.animate({
				style: { 'underlay-opacity': 0.42, 'underlay-padding': 20 },
				duration: 120,
				easing: 'ease-out-cubic'
			})
			.animate({
				style: { 'underlay-opacity': 0.28, 'underlay-padding': 14 },
				duration: 240,
				easing: 'ease-out-cubic',
				complete: () => node.removeStyle('underlay-opacity underlay-padding')
			});
	}

	function animateNodeTo(
		node: NodeSingular,
		targetPosition: { x: number; y: number },
		becameOnline: boolean
	) {
		const currentPosition = node.position();
		const moved =
			Math.abs(currentPosition.x - targetPosition.x) > 1 ||
			Math.abs(currentPosition.y - targetPosition.y) > 1;
		node.stop(true, false);
		if (moved && !prefersReducedMotion()) {
			node.animate(
				{ position: targetPosition },
				{ duration: becameOnline ? 420 : 280, easing: 'ease-out-cubic' }
			);
		} else {
			node.position(targetPosition);
		}
		if (becameOnline) pulseNode(node);
	}

	function animateGraphToPositions(
		positions: Map<string, { x: number; y: number }>,
		duration: number
	) {
		graph?.nodes().forEach((node) => {
			const position = positions.get(node.id());
			if (!position) return;
			node.stop(true, false);
			if (prefersReducedMotion()) {
				node.position(position);
				return;
			}
			node.animate({ position }, { duration, easing: 'ease-out-cubic' });
		});
	}

	function removeNode(node: NodeSingular) {
		if (prefersReducedMotion()) {
			node.remove();
			return;
		}
		node.animate(
			{ style: { opacity: 0, 'underlay-opacity': 0 } },
			{ duration: 180, easing: 'ease-out-cubic', complete: () => node.remove() }
		);
	}

	function removeEdge(edge: EdgeSingular) {
		if (prefersReducedMotion()) {
			edge.remove();
			return;
		}
		edge.animate(
			{ style: { opacity: 0, width: 0.2 } },
			{ duration: 160, easing: 'ease-out-cubic', complete: () => edge.remove() }
		);
	}

	function addNode(device: Device, targetPosition: { x: number; y: number }) {
		if (!graph) return;
		const rootPosition = graphRootDevice()
			? graph.getElementById(graphRootDevice()!.id).position()
			: undefined;
		const startPosition =
			device.online && graphRootDevice()?.id !== device.id && rootPosition?.x !== undefined
				? rootPosition
				: targetPosition;
		const node = graph.add({
			group: 'nodes',
			classes: `${deviceClasses(device)} entering`,
			data: deviceData(device),
			position: startPosition
		});
		if (prefersReducedMotion() || graphDragActive) {
			node.removeClass('entering');
			node.position(targetPosition);
			return;
		}
		node.animate(
			{
				position: targetPosition,
				style: { opacity: device.online ? 1 : 0.68, 'underlay-padding': 8 }
			},
			{
				duration: 360,
				easing: 'ease-out-cubic',
				complete: () => {
					node.removeClass('entering');
					node.removeStyle('opacity underlay-padding');
				}
			}
		);
	}

	function addEdge(edge: RenderEdge) {
		if (!graph) return;
		const added = graph.add({
			group: 'edges',
			classes: edgeClassesFor(edge),
			data: { id: edge.id, source: edge.from, target: edge.to, label: edgeLabel(edge) }
		});
		if (prefersReducedMotion()) return;
		added.style({ opacity: 0, width: 0.2 });
		added.animate(
			{ style: { opacity: 0.66, width: 2.2 } },
			{
				duration: 260,
				easing: 'ease-out-cubic',
				complete: () => added.removeStyle('opacity width')
			}
		);
	}

	function syncGraph() {
		if (!graph) return;
		const devices = graphDevices();
		const edges = current.visibleEdges;
		const desiredNodeIDs = new Set(devices.map((d) => d.id));
		const desiredEdgeIDs = new Set(edges.map((e) => e.id));
		const positions = graphPositions();
		const nextLayoutSig = graphLayoutSignature();
		const layoutChanged = nextLayoutSig !== layoutSignature;
		if (layoutChanged && graphDragActive) {
			pendingLayoutSync = true;
		}

		graph.edges().forEach((edge) => {
			if (!desiredEdgeIDs.has(edge.id())) removeEdge(edge);
		});
		graph.nodes().forEach((node) => {
			if (!desiredNodeIDs.has(node.id())) removeNode(node);
		});

		for (const device of devices) {
			const targetPosition = positions.get(device.id);
			if (!targetPosition) continue;
			const node = graph.getElementById(device.id);
			const wasOnline = lastOnlineState.get(device.id);
			if (!node.length) {
				addNode(device, targetPosition);
				continue;
			}
			node.data(deviceData(device));
			node.classes(deviceClasses(device));
			if (layoutChanged && !graphDragActive && !node.grabbed()) {
				animateNodeTo(node, targetPosition, wasOnline === false && device.online);
			}
		}

		for (const edge of edges) {
			const existing = graph.getElementById(edge.id);
			if (existing.length) {
				existing.data({ id: edge.id, source: edge.from, target: edge.to, label: edgeLabel(edge) });
				existing.classes(edgeClassesFor(edge));
				continue;
			}
			addEdge(edge);
		}

		if (!graphDragActive) {
			layoutSignature = nextLayoutSig;
		}
		rememberOnlineState();
		refreshPhysics(graph);
		updateGraphSelection();
		applyCurrentGraphFocus();
	}

	function renderGraph() {
		const elements = graphElements(graphPositions());
		const style: StylesheetJson = [
			{
				selector: 'node',
				style: {
					'background-color': 'data(color)',
					'background-opacity': 1,
					'border-color': 'data(ringColor)',
					'border-opacity': 0.82,
					'border-width': 2.5,
					color: '#21352e',
					content: 'data(label)',
					'font-size': 12,
					'font-weight': 700,
					height: 42,
					'min-zoomed-font-size': 7,
					'overlay-opacity': 0,
					'text-margin-y': -12,
					'text-outline-color': '#edf4f0',
					'text-outline-width': 2,
					'text-valign': 'top',
					'transition-duration': 180,
					'transition-property':
						'background-color, border-color, border-width, height, opacity, underlay-opacity, underlay-padding, width',
					'underlay-color': '#2f9f68',
					'underlay-opacity': 0,
					'underlay-padding': 8,
					'underlay-shape': 'ellipse',
					width: 42
				}
			},
			{
				selector: 'node.entering',
				style: { opacity: 0, 'underlay-opacity': 0.26, 'underlay-padding': 18 }
			},
			{ selector: 'node.hide-labels', style: { content: '' } },
			{ selector: 'node.online', style: { 'underlay-opacity': 0.08 } },
			{ selector: 'node.offline', style: { opacity: 0.68 } },
			{
				selector: 'node.subnet-router',
				style: { shape: 'round-rectangle', width: 56 }
			},
			{
				selector: 'node.root',
				style: {
					'border-width': 4,
					height: 58,
					'underlay-opacity': 0.2,
					'underlay-padding': 14,
					width: 58
				}
			},
			{
				selector: 'node.scenario-source',
				style: {
					'background-color': '#e8f2ec',
					'border-color': '#2f5f4a',
					'border-width': 3,
					height: 58,
					width: 58
				}
			},
			{
				selector: 'node.scenario-source.root',
				style: {
					'border-width': 4,
					'underlay-opacity': 0.22,
					'underlay-padding': 14
				}
			},
			...graphEdgeStylesheet(),
			{
				selector: 'edge',
				style: {
					'overlay-opacity': 0,
					'transition-duration': 180,
					'transition-property': 'line-color, opacity, width'
				}
			},
			{
				selector: 'edge.selected',
				style: {
					'underlay-color': '#163f31',
					'underlay-opacity': 0.18,
					'underlay-padding': 6
				}
			},
			{
				selector: 'edge[label]',
				style: {
					color: '#31443d',
					content: 'data(label)',
					'font-size': 9,
					'font-weight': 800,
					'min-zoomed-font-size': 8,
					'text-background-color': '#f6faf7',
					'text-background-opacity': 0.86,
					'text-background-padding': '2px',
					'text-rotation': 'autorotate'
				}
			},
			{ selector: '.dim', style: { opacity: 0.16 } },
			{
				selector: 'node.focused',
				style: { opacity: 1, 'underlay-opacity': 0.16, 'underlay-padding': 10 }
			},
			{
				selector: 'node.selected',
				style: {
					'border-color': '#163f31',
					'border-width': 4,
					height: 54,
					'underlay-opacity': 0.28,
					'underlay-padding': 14,
					width: 54
				}
			}
		];

		graph = cytoscape({
			container,
			elements,
			layout: graphLayoutOptions(),
			boxSelectionEnabled: false,
			selectionType: 'single',
			style
		});
		layoutSignature = graphLayoutSignature();

		graph.autoungrabify(false);
		graph.autounselectify(true);
		graph.userPanningEnabled(true);
		graph.userZoomingEnabled(true);
		refreshPhysics(graph);

		graph.on('tap', 'node', (event: CyEventObject) => {
			if (graphDragMoved || Date.now() - lastGraphDragEndAt < 120) return;
			const node = event.target as NodeSingular;
			const device = current.devices.find((d) => d.id === node.id());
			if (device) {
				onEdgeSelect(undefined);
				onNodeSelect(device);
				updateGraphSelection();
				applyGraphFocus(node);
				pulseNode(node);
			}
		});

		graph.on('tap', 'edge', (event: CyEventObject) => {
			const edgeID = event.target.id();
			const edge = current.visibleEdges.find((candidate) => candidate.id === edgeID);
			if (!edge) return;
			onEdgeSelect(edge);
			updateGraphSelection();
			applyEdgeFocus(event.target as EdgeSingular);
		});

		graph.on('mouseover', 'node', (event: CyEventObject) => {
			hoveredNodeID = event.target.id();
			applyGraphFocus(event.target as NodeSingular);
		});

		graph.on('mouseout', 'node', (event: CyEventObject) => {
			if (hoveredNodeID === event.target.id()) {
				hoveredNodeID = undefined;
			}
			applyCurrentGraphFocus();
		});

		graph.on('tap', (event: CyEventObject) => {
			if (event.target === graph) {
				hoveredNodeID = undefined;
				onEdgeSelect(undefined);
				clearGraphFocus();
			}
		});

		updateGraphSelection();
		applyCurrentGraphFocus();
		rememberOnlineState();
		window.requestAnimationFrame(() => fitGraph());
	}

	function refreshPhysics(cy: Core) {
		const signature = [
			cy
				.nodes()
				.map((n) => n.id())
				.sort()
				.join(','),
			cy
				.edges()
				.map((e) => e.id())
				.sort()
				.join(',')
		].join('|');
		if (signature === physicsSignature) return;
		cleanupPhysics?.();
		cleanupPhysics = setupPhysics(cy);
		physicsSignature = signature;
	}

	function setupPhysics(cy: Core) {
		const whitelistedNodes = new Set<string>();
		cy.edges().forEach((edge) => {
			whitelistedNodes.add(edge.source().id());
			whitelistedNodes.add(edge.target().id());
		});

		const physicsNodesMap = new Map<string, PhysicsNode>();
		cy.nodes().forEach((node) => {
			if (!whitelistedNodes.has(node.id())) return;
			const pos = node.position();
			const pn: PhysicsNode = { id: node.id(), x: pos.x, y: pos.y };
			physicsNodesMap.set(node.id(), pn);
		});

		const nodes = Array.from(physicsNodesMap.values());
		const links: SimulationLinkDatum<PhysicsNode>[] = [];
		cy.edges().forEach((edge) => {
			links.push({ source: edge.source().id(), target: edge.target().id() });
		});

		const sim = forceSimulation(nodes)
			.force(
				'link',
				forceLink(links)
					.id((d) => (d as PhysicsNode).id)
					.distance(180)
					.strength(0.5)
			)
			.force('charge', forceManyBody().strength(-300))
			.alphaDecay(0.02)
			.velocityDecay(0.4);

		let isDragging = false;

		const syncSimulationNodesFromGraph = () => {
			for (const [id, simNode] of physicsNodesMap) {
				const node = cy.getElementById(id);
				if (!node.length) continue;
				const pos = node.position();
				simNode.x = pos.x;
				simNode.y = pos.y;
				simNode.vx = 0;
				simNode.vy = 0;
			}
		};

		const onGrab = (event: CyEventObject) => {
			const node = event.target as NodeSingular;
			const simNode = physicsNodesMap.get(node.id());
			cy.elements().stop(true, false);
			graphDragActive = true;
			graphDragMoved = false;
			syncSimulationNodesFromGraph();
			if (simNode) {
				const pos = node.position();
				simNode.fx = pos.x;
				simNode.fy = pos.y;
			}
			isDragging = true;
			sim.alphaDecay(0.02);
			sim.alpha(0.8).restart();
		};

		const onFree = (event: CyEventObject) => {
			const node = event.target as NodeSingular;
			const simNode = physicsNodesMap.get(node.id());
			const didMove = graphDragMoved;
			if (simNode) {
				const pos = node.position();
				simNode.x = pos.x;
				simNode.y = pos.y;
				simNode.fx = undefined;
				simNode.fy = undefined;
			}
			isDragging = false;
			graphDragActive = false;
			if (didMove) lastGraphDragEndAt = Date.now();
			sim.alphaDecay(0.1);
			sim.stop();
			window.setTimeout(() => {
				graphDragMoved = false;
			}, 120);
			if (pendingLayoutSync) {
				pendingLayoutSync = false;
				syncGraph();
			} else if (didMove) {
				spreadGraphAfterDrag(node.id(), syncSimulationNodesFromGraph);
			}
		};

		const onDrag = (event: CyEventObject) => {
			const node = event.target as NodeSingular;
			const simNode = physicsNodesMap.get(node.id());
			if (!simNode) return;
			graphDragMoved = true;
			const pos = node.position();
			simNode.x = pos.x;
			simNode.y = pos.y;
			simNode.fx = pos.x;
			simNode.fy = pos.y;
			sim.nodes(nodes).alpha(0.8).restart();
		};

		const tickCallback = () => {
			if (!isDragging) return;
			for (const n of nodes) {
				const node = cy.getElementById(n.id);
				if (node.grabbed()) continue;
				if (n.x !== undefined && n.y !== undefined) {
					node.position({ x: n.x, y: n.y });
				}
			}
		};

		sim.on('tick', tickCallback);
		cy.on('grab', 'node', onGrab);
		cy.on('free', 'node', onFree);
		cy.on('drag', 'node', onDrag);

		return () => {
			cy.off('grab', 'node', onGrab);
			cy.off('free', 'node', onFree);
			cy.off('drag', 'node', onDrag);
			graphDragActive = false;
			sim.stop();
		};
	}

	function spreadGraphAfterDrag(draggedID: string, onComplete?: () => void) {
		if (!graph) {
			onComplete?.();
			return;
		}
		const positions = graphSpreadPositionsAfterDrag(draggedID);
		let pendingAnimations = 0;
		const finishAnimation = () => {
			pendingAnimations -= 1;
			if (pendingAnimations === 0) onComplete?.();
		};

		graph.nodes().forEach((node) => {
			if (node.id() === draggedID) return;
			const position = positions.get(node.id());
			if (!position) return;
			node.stop(true, false);
			if (prefersReducedMotion()) {
				node.position(position);
				return;
			}
			pendingAnimations += 1;
			node.animate(
				{ position },
				{ duration: 380, easing: 'ease-out-cubic', complete: finishAnimation }
			);
		});

		layoutSignature = graphLayoutSignature();
		if (pendingAnimations === 0) onComplete?.();
	}

	function fitGraph() {
		graph?.animate({
			fit: { eles: graph.elements(), padding: 64 },
			duration: 260,
			easing: 'ease-out-cubic'
		});
	}

	function zoomGraph(delta: number) {
		if (!graph) return;
		graph.zoom({
			level: graph.zoom() * delta,
			renderedPosition: { x: graph.width() / 2, y: graph.height() / 2 }
		});
	}

	function reflowGraph() {
		animateGraphToPositions(graphPositions(), 420);
		layoutSignature = graphLayoutSignature();
	}

	function selectDevice(device: Device) {
		if (!graph) return;
		updateGraphSelection();
		window.requestAnimationFrame(() => {
			const node = graph?.getElementById(device.id);
			if (!node?.length) return;
			updateGraphSelection();
			applyGraphFocus(node);
			graph?.animate({
				center: { eles: node },
				duration: 320,
				easing: 'ease-out-cubic'
			});
		});
	}

	function sync(opts: SyncOptions) {
		current = { ...current, ...opts };
		if (!graph) {
			renderGraph();
			return;
		}
		syncGraph();
	}

	function destroy() {
		uninstallGraphDebug();
		cleanupPhysics?.();
		graph?.destroy();
		graph = undefined;
	}

	return {
		sync,
		fit: fitGraph,
		zoom: zoomGraph,
		reflow: reflowGraph,
		selectDevice,
		destroy
	};
}
