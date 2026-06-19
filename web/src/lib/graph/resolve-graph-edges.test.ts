import { describe, expect, it } from 'vitest';

import type { Edge, PolicyEvaluateDraftResponse } from '../api/schemas';
import {
	filterEdgesForGraph,
	resolveBaseGraphEdges,
	resolveGraphEdgeSource
} from './resolve-graph-edges';

const sampleEdge = (overrides: Partial<Edge> = {}): Edge => ({
	id: 'alice:web',
	from: 'alice',
	to: 'web',
	kind: 'acl',
	accessScope: 'http',
	...overrides
});

const emptyEvaluation = (): PolicyEvaluateDraftResponse => ({
	tailnet: 'demo.tailor.ts.net',
	added: [],
	removed: [],
	changed: [],
	unchanged: [{ edge: sampleEdge({ id: 'eval-only' }), state: 'unchanged' }],
	broadAccess: [],
	visibleDeviceIds: [],
	unresolvedSelectors: [],
	unsupportedSections: [],
	applicationGrants: []
});

const evaluationWithAdded = (): PolicyEvaluateDraftResponse => ({
	...emptyEvaluation(),
	unchanged: [],
	added: [
		{ edge: sampleEdge({ id: 'spawn:link', from: 'dev-spawn-1', to: 'web' }), state: 'added' }
	]
});

describe('resolveGraphEdgeSource', () => {
	it('prefers topology over stale policy evaluation in steady state', () => {
		expect(
			resolveGraphEdgeSource({
				cloudAuthenticated: true,
				topologyEdges: [sampleEdge({ id: 'live', from: 'a', to: 'b' })],
				policyEvaluation: emptyEvaluation(),
				editorOpen: false,
				editorDirty: false,
				hasValidatedPending: false
			})
		).toBe('topology');
	});

	it('prefers topology over stale preview when the editor is closed', () => {
		expect(
			resolveGraphEdgeSource({
				cloudAuthenticated: true,
				topologyEdges: [sampleEdge()],
				previewEvaluation: emptyEvaluation(),
				policyEvaluation: emptyEvaluation(),
				editorOpen: false,
				editorDirty: false,
				hasValidatedPending: true
			})
		).toBe('topology');
	});

	it('uses preview evaluation only while the policy editor is open', () => {
		expect(
			resolveGraphEdgeSource({
				cloudAuthenticated: true,
				topologyEdges: [sampleEdge()],
				previewEvaluation: emptyEvaluation(),
				policyEvaluation: emptyEvaluation(),
				editorOpen: true,
				editorDirty: true,
				hasValidatedPending: false
			})
		).toBe('preview');
	});

	it('uses staged draft preview even when live topology edges are present', () => {
		expect(
			resolveGraphEdgeSource({
				cloudAuthenticated: true,
				topologyEdges: [sampleEdge({ id: 'live' })],
				previewEvaluation: evaluationWithAdded(),
				policyEvaluation: emptyEvaluation(),
				editorOpen: false,
				editorDirty: false,
				hasValidatedPending: false,
				stagedPreviewActive: true
			})
		).toBe('preview');

		const rendered = resolveBaseGraphEdges({
			cloudAuthenticated: true,
			topologyEdges: [sampleEdge({ id: 'live' })],
			previewEvaluation: evaluationWithAdded(),
			policyEvaluation: emptyEvaluation(),
			editorOpen: false,
			editorDirty: false,
			hasValidatedPending: false,
			stagedPreviewActive: true
		});

		expect(rendered?.map((edge) => [edge.id, edge.state])).toEqual([['spawn:link', 'added']]);
	});

	it('uses saved evaluation while editing without preview', () => {
		expect(
			resolveGraphEdgeSource({
				cloudAuthenticated: true,
				topologyEdges: [sampleEdge()],
				policyEvaluation: emptyEvaluation(),
				editorOpen: true,
				editorDirty: true,
				hasValidatedPending: false
			})
		).toBe('saved-evaluation');
	});

	it('falls back to saved evaluation before first topology snapshot', () => {
		expect(
			resolveGraphEdgeSource({
				cloudAuthenticated: true,
				topologyEdges: [],
				policyEvaluation: emptyEvaluation(),
				editorOpen: false,
				editorDirty: false,
				hasValidatedPending: false
			})
		).toBe('saved-evaluation');
	});
});

describe('resolveBaseGraphEdges', () => {
	it('returns topology edges in steady state even when policyEvaluation exists', () => {
		const topology = [
			sampleEdge({ id: 'fresh', from: 'spawn', to: 'web' }),
			sampleEdge({ id: 'fresh2', from: 'spawn', to: 'db' })
		];
		const rendered = resolveBaseGraphEdges({
			cloudAuthenticated: true,
			topologyEdges: topology,
			policyEvaluation: emptyEvaluation(),
			editorOpen: false,
			editorDirty: false,
			hasValidatedPending: false
		});
		expect(rendered?.map((e) => e.id)).toEqual(['fresh', 'fresh2']);
	});

	it('includes added edges from evaluation when topology is empty', () => {
		const rendered = resolveBaseGraphEdges({
			cloudAuthenticated: true,
			topologyEdges: [],
			policyEvaluation: evaluationWithAdded(),
			editorOpen: false,
			editorDirty: false,
			hasValidatedPending: false
		});
		expect(rendered?.some((e) => e.id === 'spawn:link')).toBe(true);
	});

	it('keeps live service and shared-node policy edges from topology snapshots', () => {
		const topology = [
			sampleEdge({
				id: 'admin:svc:web',
				from: 'admin-laptop',
				to: 'svc:web',
				accessScope: 'http',
				ports: ['443'],
				policyRefs: [{ section: 'grants', index: 0, src: 'autogroup:admin', dst: 'svc:web' }]
			}),
			sampleEdge({
				id: 'shared:prod',
				from: 'shared-laptop',
				to: 'prod',
				accessScope: 'limited',
				ports: ['8443'],
				policyRefs: [{ section: 'acls', index: 1, src: 'autogroup:shared', dst: 'tag:prod:8443' }]
			})
		];

		const rendered = resolveBaseGraphEdges({
			cloudAuthenticated: true,
			topologyEdges: topology,
			policyEvaluation: emptyEvaluation(),
			editorOpen: false,
			editorDirty: false,
			hasValidatedPending: false
		});

		expect(rendered).toEqual(topology);
		expect(rendered?.find((edge) => edge.to === 'svc:web')?.policyRefs?.[0]?.src).toBe(
			'autogroup:admin'
		);
		expect(rendered?.find((edge) => edge.from === 'shared-laptop')?.accessScope).toBe('limited');
	});
});

describe('filterEdgesForGraph', () => {
	const edges = [
		sampleEdge({ id: '1', from: 'focus', to: 'other' }),
		sampleEdge({ id: '2', from: 'x', to: 'y' })
	].map((e) => ({ ...e }));

	it('keeps only focus spokes in focused mode', () => {
		const visible = new Set(['focus', 'other', 'x', 'y']);
		const filtered = filterEdgesForGraph(edges, visible, 'focused', 'focus');
		expect(filtered.map((e) => e.id)).toEqual(['1']);
	});

	it('keeps service destination edges when the service node is focused', () => {
		const rendered = [
			sampleEdge({ id: 'service', from: 'alice', to: 'svc:web' }),
			sampleEdge({ id: 'unrelated', from: 'alice', to: 'db' })
		].map((edge) => ({ ...edge }));

		const filtered = filterEdgesForGraph(
			rendered,
			new Set(['alice', 'svc:web', 'db']),
			'focused',
			'svc:web'
		);

		expect(filtered.map((edge) => edge.id)).toEqual(['service']);
	});

	it('keeps shared-node source edges when the shared node is visible in all mode', () => {
		const rendered = [
			sampleEdge({ id: 'shared-policy', from: 'shared-laptop', to: 'prod' }),
			sampleEdge({ id: 'hidden', from: 'hidden-source', to: 'hidden-target' })
		].map((edge) => ({ ...edge }));

		const filtered = filterEdgesForGraph(
			rendered,
			new Set(['shared-laptop', 'prod']),
			'all',
			undefined
		);

		expect(filtered.map((edge) => edge.id)).toEqual(['shared-policy']);
	});
});
