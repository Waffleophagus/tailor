import type { RenderEdge } from './engine';

export interface EdgeClassOptions {
	selectedEdgeId?: string;
}

export function edgeClasses(edge: RenderEdge, options: EdgeClassOptions = {}): string {
	return [
		edge.kind,
		edge.accessScope ? `scope-${edge.accessScope}` : '',
		edge.state ? `state-${edge.state}` : '',
		options.selectedEdgeId === edge.id ? 'selected' : ''
	]
		.filter(Boolean)
		.join(' ');
}

export function edgeClassList(edge: RenderEdge, options: EdgeClassOptions = {}): string[] {
	return edgeClasses(edge, options).split(/\s+/).filter(Boolean);
}
