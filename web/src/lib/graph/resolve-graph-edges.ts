import type { Edge, PolicyEvaluateDraftResponse } from '../api/schemas';
import type { RenderEdge } from './engine';
import { evaluationEdges, renderPolicyEdge } from './policy-graph-edges';

export type GraphEdgeSource = 'preview' | 'saved-evaluation' | 'topology' | 'local';

export interface ResolveGraphEdgesInput {
	cloudAuthenticated: boolean;
	topologyEdges: Edge[];
	previewEvaluation?: PolicyEvaluateDraftResponse;
	policyEvaluation?: PolicyEvaluateDraftResponse;
	editorOpen: boolean;
	editorDirty: boolean;
	hasValidatedPending: boolean;
	stagedPreviewActive?: boolean;
}

/** Which edge list `resolveBaseGraphEdges` will use (for tests and debugging). */
export function resolveGraphEdgeSource(input: ResolveGraphEdgesInput): GraphEdgeSource {
	if (!input.cloudAuthenticated) {
		return 'local';
	}
	// Live topology wins unless the user is actively editing / previewing in the policy panel.
	const editingGraph =
		input.stagedPreviewActive ||
		(input.editorOpen && (input.editorDirty || input.hasValidatedPending));
	if (editingGraph && input.previewEvaluation) {
		return 'preview';
	}
	if (editingGraph && input.policyEvaluation) {
		return 'saved-evaluation';
	}
	if (input.topologyEdges.length > 0) {
		return 'topology';
	}
	if (input.policyEvaluation) {
		return 'saved-evaluation';
	}
	return 'local';
}

/**
 * Policy/tailnet edges before focus filtering. Returns null when the caller should
 * synthesize local mesh links (unauthenticated or no policy data yet).
 */
export function resolveBaseGraphEdges(input: ResolveGraphEdgesInput): RenderEdge[] | null {
	if (!input.cloudAuthenticated) {
		return null;
	}

	const source = resolveGraphEdgeSource(input);

	if (source === 'preview' && input.previewEvaluation) {
		return evaluationEdges(input.previewEvaluation, 'draft', true);
	}

	if (source === 'saved-evaluation' && input.policyEvaluation) {
		return evaluationEdges(input.policyEvaluation, 'current', input.editorDirty);
	}

	if (source === 'topology') {
		return input.topologyEdges.map((edge) => renderPolicyEdge(edge));
	}

	if (input.policyEvaluation) {
		return evaluationEdges(input.policyEvaluation, 'current', false);
	}

	return null;
}

export function filterEdgesForGraph(
	rendered: RenderEdge[],
	visibleDeviceIDs: ReadonlySet<string>,
	graphMode: 'focused' | 'all',
	focusDeviceID: string | undefined
): RenderEdge[] {
	const filtered = rendered.filter(
		(edge) => visibleDeviceIDs.has(edge.from) || visibleDeviceIDs.has(edge.to)
	);
	if (graphMode === 'all') {
		return filtered;
	}
	if (!focusDeviceID) {
		return [];
	}
	return filtered.filter((edge) => edge.from === focusDeviceID || edge.to === focusDeviceID);
}
