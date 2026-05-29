import { describe, expect, it } from 'vitest';

import { resolveGraphLayoutRoot } from './graph-layout-root';

const device = (id: string): import('../api/schemas').Device => ({
	id,
	name: id,
	online: true,
	owner: '',
	os: '',
	ip: '',
	tailscaleIps: [],
	tags: [],
	subnetRouter: false,
	routedSubnets: []
});

describe('resolveGraphLayoutRoot', () => {
	it('prefers the selected device when it is visible', () => {
		const visible = new Set(['a', 'b']);
		expect(resolveGraphLayoutRoot(device('b'), device('a'), visible)?.id).toBe('b');
	});

	it('falls back when selection is hidden or missing', () => {
		const visible = new Set(['a']);
		expect(resolveGraphLayoutRoot(device('b'), device('a'), visible)?.id).toBe('a');
		expect(resolveGraphLayoutRoot(undefined, device('a'), visible)?.id).toBe('a');
	});
});
