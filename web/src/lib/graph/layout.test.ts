import { describe, expect, it } from 'vitest';

import {
	LAYOUT_NODE_SPACING,
	angularSeparation,
	areRadiallyCollinear,
	computeGraphLayout,
	maxNodesPerRing,
	minCenterDistance,
	partitionPeersByConnection,
	placePeersOnRings
} from './layout';

describe('maxNodesPerRing', () => {
	it('scales capacity with circumference', () => {
		expect(maxNodesPerRing(140)).toBe(Math.floor((2 * Math.PI * 140) / LAYOUT_NODE_SPACING));
		expect(maxNodesPerRing(140)).toBeGreaterThan(maxNodesPerRing(70));
	});

	it('always allows at least one node', () => {
		expect(maxNodesPerRing(1)).toBe(1);
	});
});

describe('partitionPeersByConnection', () => {
	it('splits peers by visible edges to the root', () => {
		const parts = partitionPeersByConnection(
			'root',
			['a', 'b', 'c', 'd'],
			[
				{ from: 'root', to: 'a' },
				{ from: 'b', to: 'root' }
			]
		);
		expect(parts.connected).toEqual(['a', 'b']);
		expect(parts.disconnected).toEqual(['c', 'd']);
	});
});

describe('placePeersOnRings', () => {
	it('uses multiple rings when one ring would overcrowd', () => {
		const positions = new Map<string, { x: number; y: number }>();
		const peerIds = Array.from({ length: 40 }, (_, i) => `device-${String(i).padStart(2, '0')}`);
		placePeersOnRings(peerIds, { x: 450, y: 310 }, 140, LAYOUT_NODE_SPACING, positions);

		expect(positions.size).toBe(40);
		const radii = [...positions.values()].map((point) => Math.hypot(point.x - 450, point.y - 310));
		expect(new Set(radii.map((r) => Math.round(r))).size).toBeGreaterThan(1);
	});

	it('keeps same-ring neighbors at least LAYOUT_NODE_SPACING apart on the arc', () => {
		const positions = new Map<string, { x: number; y: number }>();
		const peerIds = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'];
		placePeersOnRings(peerIds, { x: 0, y: 0 }, 160, LAYOUT_NODE_SPACING, positions);
		expect(minCenterDistance(positions, peerIds)).toBeGreaterThanOrEqual(LAYOUT_NODE_SPACING - 0.5);
	});

	it('offsets successive rings so the first node is not always at 12 o clock', () => {
		const positions = new Map<string, { x: number; y: number }>();
		const center = { x: 450, y: 310 };
		placePeersOnRings(
			['inner'],
			center,
			140,
			LAYOUT_NODE_SPACING,
			positions,
			LAYOUT_NODE_SPACING,
			0
		);
		placePeersOnRings(
			['outer'],
			center,
			210,
			LAYOUT_NODE_SPACING,
			positions,
			LAYOUT_NODE_SPACING,
			1
		);
		expect(areRadiallyCollinear(positions, center, 'inner', 'outer')).toBe(false);
	});
});

describe('computeGraphLayout', () => {
	it('places the root at the viewport center', () => {
		const layout = computeGraphLayout({
			width: 900,
			height: 620,
			rootId: 'root',
			onlinePeerIds: ['a', 'b'],
			offlinePeerIds: ['c']
		});
		expect(layout.get('root')).toEqual({ x: 450, y: 310 });
	});

	it('separates a large online cohort across rings without crowding', () => {
		const peerIds = Array.from({ length: 64 }, (_, i) => `device-${String(i).padStart(2, '0')}`);
		const layout = computeGraphLayout({
			width: 900,
			height: 620,
			rootId: 'root',
			onlinePeerIds: peerIds,
			offlinePeerIds: []
		});
		expect(layout.size).toBe(65);
		expect(minCenterDistance(layout, peerIds)).toBeGreaterThanOrEqual(LAYOUT_NODE_SPACING - 0.5);
	});

	it('places connected peers on inner rings before disconnected peers', () => {
		const layout = computeGraphLayout({
			width: 900,
			height: 620,
			rootId: 'root',
			onlinePeerIds: ['far-a', 'far-b', 'near-a', 'near-b'],
			offlinePeerIds: [],
			edges: [
				{ from: 'root', to: 'near-a' },
				{ from: 'root', to: 'near-b' }
			]
		});
		const center = { x: 450, y: 310 };
		const radius = (id: string) => {
			const point = layout.get(id)!;
			return Math.hypot(point.x - center.x, point.y - center.y);
		};
		expect(Math.max(radius('near-a'), radius('near-b'))).toBeLessThan(
			Math.min(radius('far-a'), radius('far-b'))
		);
	});

	it('separates one connected and one disconnected peer on different rings', () => {
		const layout = computeGraphLayout({
			width: 900,
			height: 620,
			rootId: 'root',
			onlinePeerIds: ['connected', 'lonely'],
			offlinePeerIds: [],
			edges: [{ from: 'root', to: 'connected' }]
		});
		const center = { x: 450, y: 310 };
		expect(areRadiallyCollinear(layout, center, 'connected', 'lonely')).toBe(false);
		expect(angularSeparation(layout, center, 'connected', 'lonely')).toBeGreaterThan(
			(8 * Math.PI) / 180
		);
	});

	it('never places two nodes at the exact center', () => {
		const layout = computeGraphLayout({
			width: 900,
			height: 620,
			rootId: 'root',
			onlinePeerIds: ['a', 'b', 'c'],
			offlinePeerIds: ['d'],
			edges: [{ from: 'root', to: 'a' }]
		});
		const center = { x: 450, y: 310 };
		for (const [id, point] of layout) {
			if (id === 'root') continue;
			expect(Math.hypot(point.x - center.x, point.y - center.y)).toBeGreaterThan(100);
		}
	});
});
