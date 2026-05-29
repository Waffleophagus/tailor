import { describe, expect, it } from 'vitest';

import {
	PARALLEL_EDGE_SPREAD,
	computeParallelEdgeBundles,
	parallelEdgeData
} from './parallel-edges';

describe('computeParallelEdgeBundles', () => {
	it('assigns zero spread for a single edge in a bundle', () => {
		const bundles = computeParallelEdgeBundles([{ id: 'e1', from: 'a', to: 'b' }]);
		const bundle = bundles.get('e1')!;
		expect(bundle.bundleSize).toBe(1);
		expect(bundle.bundleIndex).toBe(0);
		expect(bundle.cpDistances).toEqual([PARALLEL_EDGE_SPREAD, PARALLEL_EDGE_SPREAD]);
	});

	it('fans three parallel edges symmetrically', () => {
		const edges = [
			{ id: 'e-a', from: 'a', to: 'b' },
			{ id: 'e-b', from: 'a', to: 'b' },
			{ id: 'e-c', from: 'a', to: 'b' }
		];
		const bundles = computeParallelEdgeBundles(edges);
		const offsets = edges.map((edge) => {
			const { cpDistances, cpWeights } = bundles.get(edge.id)!;
			const sign = cpWeights[0] < cpWeights[1] ? -1 : 1;
			return sign * cpDistances[0];
		});
		offsets.sort((a, b) => a - b);
		expect(offsets).toEqual([-40, 10, 40]);
	});

	it('treats opposite directions as separate bundles', () => {
		const bundles = computeParallelEdgeBundles([
			{ id: 'ab', from: 'a', to: 'b' },
			{ id: 'ba', from: 'b', to: 'a' }
		]);
		expect(bundles.get('ab')?.bundleSize).toBe(1);
		expect(bundles.get('ba')?.bundleSize).toBe(1);
	});

	it('sorts bundle members by id for stable indices', () => {
		const bundles = computeParallelEdgeBundles([
			{ id: 'z-edge', from: 'a', to: 'b' },
			{ id: 'a-edge', from: 'a', to: 'b' }
		]);
		expect(bundles.get('a-edge')?.bundleIndex).toBe(0);
		expect(bundles.get('z-edge')?.bundleIndex).toBe(1);
	});
});

describe('parallelEdgeData', () => {
	it('falls back to default control points when edge is missing from map', () => {
		const data = parallelEdgeData({ id: 'x', from: 'a', to: 'b' }, new Map());
		expect(data.bundleSize).toBe(1);
		expect(data.cpDistances).toEqual([PARALLEL_EDGE_SPREAD, PARALLEL_EDGE_SPREAD]);
	});
});
