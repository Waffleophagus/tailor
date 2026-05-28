import type { RenderEdge } from '../graph/engine';

/** Node IDs visible in focused scenario mode: source cohort plus reachable targets. */
export function focusedScenarioNodeIds(
	edges: RenderEdge[],
	sourceIds: ReadonlySet<string>
): Set<string> {
	const ids = new Set<string>(sourceIds);
	for (const edge of edges) {
		if (sourceIds.has(edge.from)) {
			ids.add(edge.to);
		}
	}
	return ids;
}

/** Distinct reachable targets from the source cohort (excluding sources). */
export function scenarioReachableCount(
	edges: RenderEdge[],
	sourceIds: ReadonlySet<string>
): number {
	const targets = new Set<string>();
	for (const edge of edges) {
		if (sourceIds.has(edge.from) && !sourceIds.has(edge.to)) {
			targets.add(edge.to);
		}
	}
	return targets.size;
}
