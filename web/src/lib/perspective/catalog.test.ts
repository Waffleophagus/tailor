import { describe, expect, it } from 'vitest';

import type { Device } from '../api/schemas';
import { parsePerspectiveInput, validatePerspective } from './catalog';

const devices: Device[] = [
	{
		id: 'dev-superadmin-console',
		name: 'superadmin-console',
		ip: '100.100.0.70',
		tailscaleIps: ['100.100.0.70'],
		os: 'linux',
		online: true,
		owner: 'superadmin@demo.tailor.ts.net',
		tags: [],
		subnetRouter: false,
		routedSubnets: []
	}
];

describe('validatePerspective', () => {
	it('accepts the demo super-user email', () => {
		const parsed = parsePerspectiveInput('superadmin@demo.tailor.ts.net');
		expect(parsed.selector).toBe('superadmin@demo.tailor.ts.net');

		const validation = validatePerspective('superadmin@demo.tailor.ts.net', devices);
		expect(validation.status).toBe('valid');
		if (validation.status === 'valid') {
			expect(validation.deviceCount).toBe(1);
		}
	});

	it('accepts the demo super-user group', () => {
		const validation = validatePerspective('group:superuser', devices, {
			tailnet: 'demo.tailor.ts.net',
			hujson: '',
			sections: [
				{
					name: 'groups',
					type: 'object',
					supported: true,
					count: 1,
					entries: [{ label: 'group:superuser', value: ['superadmin@demo.tailor.ts.net'] }]
				}
			]
		});
		expect(validation.status).toBe('valid');
	});
});
