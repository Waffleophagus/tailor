import { describe, expect, it } from 'vitest';

import type { Device } from '../api/schemas';
import {
	aggregateIdForTag,
	collapseDevicesByTag,
	DEFAULT_TAG_COLLAPSE_RULES,
	isAggregateDeviceId,
	rewriteEdgesForCollapsedDevices
} from './collapse-devices';
import type { RenderEdge } from './engine';

const device = (overrides: Partial<Device> & Pick<Device, 'id'>): Device => ({
	name: overrides.id,
	ip: '100.64.0.1',
	tailscaleIps: ['100.64.0.1'],
	os: 'linux',
	online: true,
	owner: 'alice@example.com',
	tags: [],
	subnetRouter: false,
	routedSubnets: [],
	...overrides
});

describe('collapseDevicesByTag', () => {
	it('leaves devices unchanged when collapse is disabled', () => {
		const devices = [
			device({ id: 'm1', tags: ['tag:mullvad-exit-node'] }),
			device({ id: 'm2', tags: ['tag:mullvad-exit-node'] })
		];
		const result = collapseDevicesByTag(devices, { enabled: false });
		expect(result.graphDevices).toEqual(devices);
		expect(result.aggregateMeta.size).toBe(0);
	});

	it('collapses when member count meets the tag threshold', () => {
		const devices = Array.from({ length: 5 }, (_, index) =>
			device({ id: `m${index}`, tags: ['tag:mullvad-exit-node'], online: index < 3 })
		);
		const result = collapseDevicesByTag(devices, {
			enabled: true,
			rules: [{ tag: 'tag:mullvad-exit-node', minCount: 5, label: 'Mullvad exit nodes' }]
		});

		expect(result.graphDevices).toHaveLength(1);
		expect(isAggregateDeviceId(result.graphDevices[0]!.id)).toBe(true);
		expect(result.graphDevices[0]!.name).toBe('Mullvad exit nodes (5)');
		expect(
			result.aggregateMeta.get(aggregateIdForTag('tag:mullvad-exit-node'))?.members
		).toHaveLength(5);
	});

	it('respects per-tag minCount thresholds independently', () => {
		const devices = [
			...Array.from({ length: 3 }, (_, index) =>
				device({ id: `m${index}`, tags: ['tag:mullvad-exit-node'] })
			),
			...Array.from({ length: 4 }, (_, index) =>
				device({ id: `c${index}`, tags: ['tag:ci-runner'] })
			)
		];
		const result = collapseDevicesByTag(devices, {
			enabled: true,
			rules: [
				{ tag: 'tag:mullvad-exit-node', minCount: 5, label: 'Mullvad exit nodes' },
				{ tag: 'tag:ci-runner', minCount: 4, label: 'CI runners' }
			]
		});

		expect(result.graphDevices).toHaveLength(4);
		expect(result.aggregateMeta.size).toBe(1);
		expect(result.aggregateMeta.has(aggregateIdForTag('tag:ci-runner'))).toBe(true);
	});

	it('uses default rules for Mullvad exit nodes', () => {
		const devices = Array.from({ length: 4 }, (_, index) =>
			device({ id: `m${index}`, tags: ['tag:mullvad-exit-node'] })
		);
		const below = collapseDevicesByTag(devices, {
			enabled: true,
			rules: DEFAULT_TAG_COLLAPSE_RULES
		});
		expect(below.graphDevices).toHaveLength(4);

		const atThreshold = collapseDevicesByTag(
			[...devices, device({ id: 'm4', tags: ['tag:mullvad-exit-node'] })],
			{ enabled: true, rules: DEFAULT_TAG_COLLAPSE_RULES }
		);
		expect(atThreshold.graphDevices).toHaveLength(1);
	});
});

describe('rewriteEdgesForCollapsedDevices', () => {
	it('maps edges to aggregate nodes and deduplicates parallel links', () => {
		const graphIdForDevice = new Map([
			['m1', 'agg'],
			['m2', 'agg'],
			['alice', 'alice']
		]);
		const edges: RenderEdge[] = [
			{ id: 'e1', from: 'alice', to: 'm1', kind: 'acl' },
			{ id: 'e2', from: 'alice', to: 'm2', kind: 'acl' },
			{ id: 'e3', from: 'm1', to: 'm2', kind: 'tag' }
		];

		const rewritten = rewriteEdgesForCollapsedDevices(edges, graphIdForDevice);
		expect(rewritten).toHaveLength(1);
		expect(rewritten[0]).toMatchObject({ from: 'alice', to: 'agg', kind: 'acl' });
	});
});
