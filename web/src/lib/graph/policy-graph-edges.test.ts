import { describe, expect, it } from 'vitest';

import type { Edge, PolicyEvaluateDraftResponse } from '../api/schemas';
import { evaluationEdges, ghostDeniedEdges, renderPolicyEdge } from './policy-graph-edges';

const sampleEdge = (overrides: Partial<Edge> = {}): Edge => ({
	id: 'alice:web',
	from: 'alice',
	to: 'web',
	kind: 'acl',
	accessScope: 'http',
	...overrides
});

describe('renderPolicyEdge', () => {
	it('copies edge fields and attaches state', () => {
		expect(renderPolicyEdge(sampleEdge(), 'added')).toEqual({
			...sampleEdge(),
			state: 'added'
		});
	});
});

describe('evaluationEdges', () => {
	const evaluation = {
		tailnet: 'example.com',
		added: [{ edge: sampleEdge({ id: 'added' }), state: 'added' as const }],
		removed: [
			{ edge: sampleEdge({ id: 'removed', accessScope: 'ssh' }), state: 'removed' as const }
		],
		changed: [
			{
				edge: sampleEdge({ id: 'changed' }),
				saved: sampleEdge({ id: 'changed', accessScope: 'http' }),
				draft: sampleEdge({ id: 'changed', accessScope: 'ssh' }),
				state: 'changed' as const
			}
		],
		unchanged: [{ edge: sampleEdge({ id: 'same' }), state: 'unchanged' as const }],
		broadAccess: [],
		visibleDeviceIds: [],
		unresolvedSelectors: [],
		unsupportedSections: [],
		applicationGrants: []
	} satisfies PolicyEvaluateDraftResponse;

	it('diff mode with draft shows added, removed, and changed draft edges', () => {
		const edges = evaluationEdges(evaluation, 'diff', true);
		expect(edges.map((edge) => [edge.id, edge.state])).toEqual([
			['added', 'added'],
			['removed', 'removed'],
			['changed', 'changed']
		]);
		expect(edges.find((edge) => edge.id === 'changed')?.accessScope).toBe('ssh');
	});

	it('draft mode keeps unchanged and merges added/changed', () => {
		const edges = evaluationEdges(evaluation, 'draft', true);
		expect(edges.map((edge) => [edge.id, edge.state])).toEqual([
			['same', 'unchanged'],
			['added', 'added'],
			['changed', 'changed']
		]);
	});

	it('current mode with draft marks removed/changed appropriately', () => {
		const edges = evaluationEdges(evaluation, 'current', true);
		expect(edges.find((edge) => edge.id === 'removed')?.state).toBe('removed');
		expect(edges.find((edge) => edge.id === 'changed')?.state).toBe('changed');
		expect(edges.find((edge) => edge.id === 'changed')?.accessScope).toBe('http');
	});
});

describe('ghostDeniedEdges', () => {
	const allowed = [
		{ id: 'allow', from: 'src', to: 'allowed-target', kind: 'acl' as const },
		{ id: 'allow2', from: 'src', to: 'reachable-via-target', kind: 'acl' as const }
	];
	const devices = [
		{ id: 'src' },
		{ id: 'allowed-target' },
		{ id: 'reachable-via-target' },
		{ id: 'blocked' }
	];

	it('creates ghost edges from sources to unreachable devices', () => {
		const ghosts = ghostDeniedEdges(allowed, new Set(['src']), devices);
		expect(ghosts).toEqual([
			{
				id: 'ghost:src:blocked',
				from: 'src',
				to: 'blocked',
				kind: 'acl',
				state: 'ghost-denied'
			}
		]);
	});

	it('skips cohort members and already reachable targets', () => {
		const ghosts = ghostDeniedEdges(allowed, new Set(['src']), devices);
		expect(ghosts.some((edge) => edge.to === 'allowed-target')).toBe(false);
		expect(ghosts.some((edge) => edge.to === 'reachable-via-target')).toBe(false);
	});

	it('respects max ghost cap', () => {
		const manyDevices = Array.from({ length: 30 }, (_, index) => ({ id: `d${index}` }));
		const ghosts = ghostDeniedEdges([], new Set(['solo']), manyDevices, 5);
		expect(ghosts).toHaveLength(5);
	});
});
