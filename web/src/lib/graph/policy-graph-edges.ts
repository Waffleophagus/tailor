import type { Edge, PolicyEvaluateDraftResponse } from '../api/schemas';
import type { RenderEdge, RenderEdgeState } from './engine';

export function renderPolicyEdge(edge: Edge, state?: RenderEdgeState): RenderEdge {
	return { ...edge, state };
}

export function evaluationEdges(
	evaluation: PolicyEvaluateDraftResponse,
	mode: 'current' | 'draft' | 'diff',
	hasDraft: boolean
): RenderEdge[] {
	if (mode === 'diff' && hasDraft) {
		return [
			...evaluation.added.map((change) => renderPolicyEdge(change.edge, 'added')),
			...evaluation.removed.map((change) => renderPolicyEdge(change.edge, 'removed')),
			...evaluation.changed.map((change) =>
				renderPolicyEdge(change.draft ?? change.edge, 'changed')
			)
		];
	}
	if (mode === 'draft') {
		return [
			...evaluation.unchanged.map((change) => renderPolicyEdge(change.edge, 'unchanged')),
			...evaluation.added.map((change) => renderPolicyEdge(change.edge, 'added')),
			...evaluation.changed.map((change) =>
				renderPolicyEdge(change.draft ?? change.edge, 'changed')
			)
		];
	}
	const added = evaluation.added.map((change) =>
		renderPolicyEdge(change.edge, hasDraft ? 'added' : 'unchanged')
	);
	return [
		...evaluation.unchanged.map((change) => renderPolicyEdge(change.edge, 'unchanged')),
		...added,
		...evaluation.removed.map((change) =>
			renderPolicyEdge(change.edge, hasDraft ? 'removed' : 'unchanged')
		),
		...evaluation.changed.map((change) =>
			renderPolicyEdge(change.saved ?? change.edge, hasDraft ? 'changed' : 'unchanged')
		)
	];
}

export function ghostDeniedEdges(
	allowed: RenderEdge[],
	sourceIds: ReadonlySet<string>,
	visibleDevices: ReadonlyArray<{ id: string }>,
	maxGhosts = 24
): RenderEdge[] {
	const allowedPairs = new Set(allowed.map((edge) => `${edge.from}\0${edge.to}`));
	const reachable = new Set<string>();
	for (const edge of allowed) {
		if (sourceIds.has(edge.from)) reachable.add(edge.to);
	}
	const ghosts: RenderEdge[] = [];
	for (const sourceId of sourceIds) {
		for (const device of visibleDevices) {
			if (sourceIds.has(device.id) || reachable.has(device.id)) continue;
			const key = `${sourceId}\0${device.id}`;
			if (allowedPairs.has(key)) continue;
			ghosts.push({
				id: `ghost:${sourceId}:${device.id}`,
				from: sourceId,
				to: device.id,
				kind: 'acl',
				state: 'ghost-denied'
			});
			if (ghosts.length >= maxGhosts) return ghosts;
		}
	}
	return ghosts;
}
