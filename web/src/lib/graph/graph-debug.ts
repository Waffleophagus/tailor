import type { Core } from 'cytoscape';

import { edgeClasses } from './edge-classes';
import { resolveEdgeStyle } from './edge-style';
import type { RenderEdge } from './engine';

export interface GraphDebugEdge {
	id: string;
	from: string;
	to: string;
	classes: string[];
	style: ReturnType<typeof resolveEdgeStyle>;
}

export interface GraphDebugSnapshot {
	edges: GraphDebugEdge[];
}

export function graphDebugSnapshot(
	graph: Core,
	visibleEdges: RenderEdge[],
	selectedEdgeId?: string
): GraphDebugSnapshot {
	return {
		edges: graph.edges().map((element) => {
			const renderEdge = visibleEdges.find((candidate) => candidate.id === element.id());
			const classes = renderEdge
				? edgeClasses(renderEdge, { selectedEdgeId }).split(/\s+/).filter(Boolean)
				: [...element.classes()];
			return {
				id: element.id(),
				from: element.source().id(),
				to: element.target().id(),
				classes,
				style: resolveEdgeStyle(classes)
			};
		})
	};
}

declare global {
	interface Window {
		__tailorGraphDebug?: () => GraphDebugSnapshot;
	}
}

export function installGraphDebug(
	getGraph: () =>
		| { cy: Core | undefined; visibleEdges: RenderEdge[]; selectedEdgeId?: string }
		| undefined
) {
	if (!import.meta.env.DEV) return;
	window.__tailorGraphDebug = () => {
		const ctx = getGraph();
		if (!ctx?.cy) return { edges: [] };
		return graphDebugSnapshot(ctx.cy, ctx.visibleEdges, ctx.selectedEdgeId);
	};
}

export function uninstallGraphDebug() {
	if (!import.meta.env.DEV) return;
	delete window.__tailorGraphDebug;
}
