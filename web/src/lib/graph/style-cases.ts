import type { RenderEdge } from './engine';
import type { EdgeStylePatch } from './style-catalog';

export interface EdgeStyleCase {
	name: string;
	edge: RenderEdge;
	options?: { selectedEdgeId?: string };
	extraClasses?: string[];
	expected: EdgeStylePatch;
}

function edge(overrides: Partial<RenderEdge> & Pick<RenderEdge, 'kind'>): RenderEdge {
	return {
		id: 'edge-1',
		from: 'alice',
		to: 'web',
		...overrides
	};
}

/** Table-driven style expectations for every graph edge visual variant. */
export const EDGE_STYLE_CASES: EdgeStyleCase[] = [
	{
		name: 'default edge base',
		edge: edge({ kind: 'owner' }),
		expected: { lineColor: '#5d7f73', width: 2.4 }
	},
	{
		name: 'inferred tag link is purple dashed',
		edge: edge({ kind: 'tag' }),
		expected: { lineColor: '#7c6fb0', lineStyle: 'dashed', width: 1.7 }
	},
	{
		name: 'inferred subnet link is orange dotted',
		edge: edge({ kind: 'subnet' }),
		expected: { lineColor: '#a5663f', lineStyle: 'dotted' }
	},
	{
		name: 'generic ACL link is teal',
		edge: edge({ kind: 'acl' }),
		expected: { lineColor: '#438aa1', width: 2.2, targetArrowShape: 'triangle' }
	},
	{
		name: 'ACL SSH scope is green',
		edge: edge({ kind: 'acl', accessScope: 'ssh' }),
		expected: { lineColor: '#2f9f68', width: 2.8 }
	},
	{
		name: 'ACL HTTP scope is blue',
		edge: edge({ kind: 'acl', accessScope: 'http' }),
		expected: { lineColor: '#438aa1', width: 2.4 }
	},
	{
		name: 'ACL broad scope is gold',
		edge: edge({ kind: 'acl', accessScope: 'broad' }),
		expected: { lineColor: '#b0892f', width: 3.1 }
	},
	{
		name: 'ACL custom scope is purple dashed',
		edge: edge({ kind: 'acl', accessScope: 'custom' }),
		expected: { lineColor: '#7c6fb0', lineStyle: 'dashed', width: 2.3 }
	},
	{
		name: 'ACL limited scope is purple dashed',
		edge: edge({ kind: 'acl', accessScope: 'limited' }),
		expected: { lineColor: '#7c6fb0', lineStyle: 'dashed', width: 2.3 }
	},
	{
		name: 'local tailnet link is straight green',
		edge: edge({ kind: 'local' }),
		expected: { lineColor: '#2f9f68', curveStyle: 'straight', opacity: 0.66, width: 2.2 }
	},
	{
		name: 'draft added edge is green dashed',
		edge: edge({ kind: 'acl', accessScope: 'http', state: 'added' }),
		expected: { lineColor: '#2f9f68', lineStyle: 'dashed', width: 3.3, opacity: 0.94 }
	},
	{
		name: 'draft removed edge is red dotted',
		edge: edge({ kind: 'acl', accessScope: 'ssh', state: 'removed' }),
		expected: { lineColor: '#b94c4c', lineStyle: 'dotted', width: 2.8, opacity: 0.78 }
	},
	{
		name: 'draft changed edge is gold dashed',
		edge: edge({ kind: 'acl', accessScope: 'http', state: 'changed' }),
		expected: { lineColor: '#b0892f', lineStyle: 'dashed', width: 3, opacity: 0.9 }
	},
	{
		name: 'ghost denied edge is gray dotted',
		edge: edge({ kind: 'acl', state: 'ghost-denied' }),
		expected: { lineColor: '#9aa7a1', lineStyle: 'dotted', width: 1.8, opacity: 0.42 }
	},
	{
		name: 'selected edge is wider',
		edge: edge({ id: 'pick-me', kind: 'acl', accessScope: 'http' }),
		options: { selectedEdgeId: 'pick-me' },
		expected: { lineColor: '#438aa1', width: 4.4, opacity: 1 }
	},
	{
		name: 'focused edge boosts opacity',
		edge: edge({ kind: 'acl' }),
		extraClasses: ['focused'],
		expected: { lineColor: '#438aa1', opacity: 0.96, width: 3.3 }
	}
];

export const EDGE_CLASS_CASES: Array<{
	name: string;
	edge: RenderEdge;
	options?: { selectedEdgeId?: string };
	expected: string[];
}> = [
	{
		name: 'ACL edge with HTTP scope',
		edge: edge({ kind: 'acl', accessScope: 'http' }),
		expected: ['acl', 'scope-http']
	},
	{
		name: 'ghost edge classes',
		edge: edge({ kind: 'acl', state: 'ghost-denied' }),
		expected: ['acl', 'state-ghost-denied']
	},
	{
		name: 'selected edge adds class',
		edge: edge({ id: 'sel', kind: 'tag' }),
		options: { selectedEdgeId: 'sel' },
		expected: ['tag', 'selected']
	}
];
