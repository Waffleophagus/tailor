import type { RenderEdge } from './engine';

/** Control-point spread (px) between parallel edges in the same directed bundle. */
export const PARALLEL_EDGE_SPREAD = 40;

export interface ParallelEdgeControlPoints {
	cpDistances: [number, number];
	cpWeights: [number, number];
	bundleIndex: number;
	bundleSize: number;
}

export interface ParallelEdgeBundleInput {
	id: string;
	from: string;
	to: string;
}

function parallelEdgeKey(from: string, to: string): string {
	return `${from}\0${to}`;
}

function controlPointsForBundleIndex(
	bundleIndex: number,
	bundleSize: number,
	spread = PARALLEL_EDGE_SPREAD
): ParallelEdgeControlPoints {
	if (bundleSize <= 1) {
		return {
			cpDistances: [spread, spread],
			cpWeights: [0.25, 0.75],
			bundleIndex: 0,
			bundleSize: 1
		};
	}
	const offset = (bundleIndex - (bundleSize - 1) / 2) * spread;
	const distance = Math.abs(offset) || spread * 0.25;
	const sign = offset === 0 ? 1 : Math.sign(offset);
	return {
		cpDistances: [distance, distance],
		cpWeights: sign < 0 ? [0.2, 0.8] : [0.8, 0.2],
		bundleIndex,
		bundleSize
	};
}

/**
 * Groups directed edges by (from, to) and assigns symmetric control-point offsets
 * so parallel curves fan instead of overlapping.
 */
export function computeParallelEdgeBundles<T extends ParallelEdgeBundleInput>(
	edges: readonly T[],
	spread = PARALLEL_EDGE_SPREAD
): Map<string, ParallelEdgeControlPoints> {
	const groups = new Map<string, T[]>();
	for (const edge of edges) {
		const key = parallelEdgeKey(edge.from, edge.to);
		const list = groups.get(key);
		if (list) list.push(edge);
		else groups.set(key, [edge]);
	}

	const result = new Map<string, ParallelEdgeControlPoints>();
	for (const group of groups.values()) {
		const sorted = [...group].sort((a, b) => a.id.localeCompare(b.id));
		const bundleSize = sorted.length;
		sorted.forEach((edge, bundleIndex) => {
			result.set(edge.id, controlPointsForBundleIndex(bundleIndex, bundleSize, spread));
		});
	}
	return result;
}

export function parallelEdgeData(
	edge: ParallelEdgeBundleInput,
	bundles: Map<string, ParallelEdgeControlPoints>
): ParallelEdgeControlPoints {
	return (
		bundles.get(edge.id) ?? {
			cpDistances: [PARALLEL_EDGE_SPREAD, PARALLEL_EDGE_SPREAD],
			cpWeights: [0.25, 0.75],
			bundleIndex: 0,
			bundleSize: 1
		}
	);
}

export function edgeElementData(
	edge: RenderEdge,
	bundles: Map<string, ParallelEdgeControlPoints>,
	label: string
) {
	const bundle = parallelEdgeData(edge, bundles);
	return {
		id: edge.id,
		source: edge.from,
		target: edge.to,
		label,
		cpDistances: bundle.cpDistances,
		cpWeights: bundle.cpWeights,
		bundleIndex: bundle.bundleIndex,
		bundleSize: bundle.bundleSize
	};
}
