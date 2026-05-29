/** Minimum center-to-center spacing between nodes on the same ring. */
export const LAYOUT_NODE_SPACING = 70;

export interface Point {
	x: number;
	y: number;
}

export interface LayoutEdge {
	from: string;
	to: string;
}

export function maxNodesPerRing(radius: number, spacing = LAYOUT_NODE_SPACING): number {
	if (radius <= 0) return 1;
	return Math.max(1, Math.floor((2 * Math.PI * radius) / spacing));
}

/** Place peers on concentric rings with uniform angular spacing. Returns the next free radius. */
export function placePeersOnRings(
	peerIds: readonly string[],
	center: Point,
	startRadius: number,
	ringStep: number,
	positions: Map<string, Point>,
	spacing = LAYOUT_NODE_SPACING
): number {
	if (peerIds.length === 0) return startRadius;

	const sorted = [...peerIds].sort((a, b) => a.localeCompare(b));
	let index = 0;
	let ringIndex = 0;

	while (index < sorted.length) {
		const radius = startRadius + ringIndex * ringStep;
		const capacity = maxNodesPerRing(radius, spacing);
		const count = Math.min(capacity, sorted.length - index);
		for (let i = 0; i < count; i += 1) {
			const id = sorted[index + i]!;
			const angle = -Math.PI / 2 + (2 * Math.PI * i) / count;
			positions.set(id, {
				x: center.x + Math.cos(angle) * radius,
				y: center.y + Math.sin(angle) * radius
			});
		}
		index += count;
		ringIndex += 1;
	}

	return startRadius + ringIndex * ringStep;
}

export function partitionPeersByConnection(
	rootId: string | undefined,
	peerIds: readonly string[],
	edges: readonly LayoutEdge[]
): { connected: string[]; disconnected: string[] } {
	const sorted = [...peerIds].sort((a, b) => a.localeCompare(b));
	if (!rootId) {
		return { connected: [], disconnected: sorted };
	}
	const connected = new Set<string>();
	for (const edge of edges) {
		if (edge.from === rootId) connected.add(edge.to);
		if (edge.to === rootId) connected.add(edge.from);
	}
	return {
		connected: sorted.filter((id) => connected.has(id)),
		disconnected: sorted.filter((id) => !connected.has(id))
	};
}

export function computeGraphLayout(input: {
	width: number;
	height: number;
	rootId?: string;
	onlinePeerIds: string[];
	offlinePeerIds: string[];
	edges?: readonly LayoutEdge[];
}): Map<string, Point> {
	const { width, height, rootId, onlinePeerIds, offlinePeerIds, edges = [] } = input;
	const center = { x: width / 2, y: height / 2 };
	const positions = new Map<string, Point>();
	const minDim = Math.min(width, height);
	const baseRadius = Math.max(140, minDim * 0.2);
	const ringStep = LAYOUT_NODE_SPACING;

	if (rootId) positions.set(rootId, { ...center });

	let nextRadius = baseRadius;
	const online = partitionPeersByConnection(rootId, onlinePeerIds, edges);
	nextRadius = placePeersOnRings(online.connected, center, nextRadius, ringStep, positions);
	nextRadius = placePeersOnRings(online.disconnected, center, nextRadius, ringStep, positions);

	const offline = partitionPeersByConnection(rootId, offlinePeerIds, edges);
	nextRadius = placePeersOnRings(offline.connected, center, nextRadius, ringStep, positions);
	placePeersOnRings(offline.disconnected, center, nextRadius, ringStep, positions);

	return positions;
}

export function minCenterDistance(positions: Map<string, Point>, ids: readonly string[]): number {
	let min = Number.POSITIVE_INFINITY;
	for (let i = 0; i < ids.length; i += 1) {
		const a = positions.get(ids[i]!);
		if (!a) continue;
		for (let j = i + 1; j < ids.length; j += 1) {
			const b = positions.get(ids[j]!);
			if (!b) continue;
			min = Math.min(min, Math.hypot(a.x - b.x, a.y - b.y));
		}
	}
	return min;
}
