/** Minimum center-to-center spacing between nodes on the same ring. */
export const LAYOUT_NODE_SPACING = 70;

/** Angular threshold (radians) below which two nodes are treated as radially stacked. */
const RADIAL_COLLISION_ANGLE = (8 * Math.PI) / 180;

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

export interface PlacePeersResult {
	nextRadius: number;
	nextRingIndex: number;
}

/**
 * Place peers on concentric rings with uniform angular spacing.
 * Each ring is rotated so sparse cohorts do not stack on the same radial line.
 */
export function placePeersOnRings(
	peerIds: readonly string[],
	center: Point,
	startRadius: number,
	ringStep: number,
	positions: Map<string, Point>,
	spacing = LAYOUT_NODE_SPACING,
	startRingIndex = 0
): PlacePeersResult {
	if (peerIds.length === 0) {
		return { nextRadius: startRadius, nextRingIndex: startRingIndex };
	}

	const sorted = [...peerIds].sort((a, b) => a.localeCompare(b));
	let index = 0;
	let ringIndex = startRingIndex;

	while (index < sorted.length) {
		const radius = startRadius + (ringIndex - startRingIndex) * ringStep;
		const capacity = maxNodesPerRing(radius, spacing);
		const count = Math.min(capacity, sorted.length - index);
		const startAngle = -Math.PI / 2 + (ringIndex * Math.PI) / Math.max(6, capacity);
		for (let i = 0; i < count; i += 1) {
			const id = sorted[index + i]!;
			const angle = startAngle + (2 * Math.PI * i) / count;
			positions.set(id, {
				x: center.x + Math.cos(angle) * radius,
				y: center.y + Math.sin(angle) * radius
			});
		}
		index += count;
		ringIndex += 1;
	}

	return {
		nextRadius: startRadius + (ringIndex - startRingIndex) * ringStep,
		nextRingIndex: ringIndex
	};
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

function polarFromCenter(center: Point, point: Point): { radius: number; angle: number } {
	const dx = point.x - center.x;
	const dy = point.y - center.y;
	return { radius: Math.hypot(dx, dy), angle: Math.atan2(dy, dx) };
}

function pointFromPolar(center: Point, radius: number, angle: number): Point {
	return {
		x: center.x + Math.cos(angle) * radius,
		y: center.y + Math.sin(angle) * radius
	};
}

function angularDifference(a: number, b: number): number {
	let diff = Math.abs(a - b) % (2 * Math.PI);
	if (diff > Math.PI) diff = 2 * Math.PI - diff;
	return diff;
}

/** Nudge angles when nodes on different rings share a radial line. */
export function resolveRadialCollisions(
	positions: Map<string, Point>,
	center: Point,
	rootId: string | undefined,
	spacing = LAYOUT_NODE_SPACING
): void {
	const peerIds = [...positions.keys()].filter((id) => id !== rootId);
	if (peerIds.length < 2) return;

	const polar = new Map(
		peerIds.map((id) => {
			const point = positions.get(id)!;
			return [id, polarFromCenter(center, point)] as const;
		})
	);

	const sorted = [...peerIds].sort(
		(a, b) => (polar.get(a)?.radius ?? 0) - (polar.get(b)?.radius ?? 0)
	);

	const minSeparation = spacing * 0.85;

	for (let i = 0; i < sorted.length; i += 1) {
		const id = sorted[i]!;
		const current = polar.get(id)!;
		for (let j = 0; j < i; j += 1) {
			const otherId = sorted[j]!;
			const other = polar.get(otherId)!;
			if (angularDifference(current.angle, other.angle) > RADIAL_COLLISION_ANGLE) continue;
			const chord = Math.hypot(
				positions.get(id)!.x - positions.get(otherId)!.x,
				positions.get(id)!.y - positions.get(otherId)!.y
			);
			if (chord >= minSeparation) continue;

			const nudge = (2 * Math.PI) / Math.max(8, maxNodesPerRing(current.radius, spacing));
			const nextAngle = current.angle + nudge;
			polar.set(id, { radius: current.radius, angle: nextAngle });
			positions.set(id, pointFromPolar(center, current.radius, nextAngle));
			break;
		}
	}
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
	let ringIndex = 0;

	const online = partitionPeersByConnection(rootId, onlinePeerIds, edges);
	let placed = placePeersOnRings(
		online.connected,
		center,
		nextRadius,
		ringStep,
		positions,
		LAYOUT_NODE_SPACING,
		ringIndex
	);
	nextRadius = placed.nextRadius;
	ringIndex = placed.nextRingIndex;

	placed = placePeersOnRings(
		online.disconnected,
		center,
		nextRadius,
		ringStep,
		positions,
		LAYOUT_NODE_SPACING,
		ringIndex
	);
	nextRadius = placed.nextRadius;
	ringIndex = placed.nextRingIndex;

	const offline = partitionPeersByConnection(rootId, offlinePeerIds, edges);
	placed = placePeersOnRings(
		offline.connected,
		center,
		nextRadius,
		ringStep,
		positions,
		LAYOUT_NODE_SPACING,
		ringIndex
	);
	nextRadius = placed.nextRadius;
	ringIndex = placed.nextRingIndex;

	placePeersOnRings(
		offline.disconnected,
		center,
		nextRadius,
		ringStep,
		positions,
		LAYOUT_NODE_SPACING,
		ringIndex
	);

	resolveRadialCollisions(positions, center, rootId);

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

/** Smallest angular separation between two peer positions around center (radians). */
export function angularSeparation(
	positions: Map<string, Point>,
	center: Point,
	idA: string,
	idB: string
): number {
	const a = positions.get(idA);
	const b = positions.get(idB);
	if (!a || !b) return 0;
	const polarA = polarFromCenter(center, a);
	const polarB = polarFromCenter(center, b);
	return angularDifference(polarA.angle, polarB.angle);
}

/** True when two points share a radial line through center (within ~2°). */
export function areRadiallyCollinear(
	positions: Map<string, Point>,
	center: Point,
	idA: string,
	idB: string
): boolean {
	return angularSeparation(positions, center, idA, idB) < (2 * Math.PI) / 180;
}
