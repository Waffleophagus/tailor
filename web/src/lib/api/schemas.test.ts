import { describe, expect, test } from 'vitest';

import {
	aclDraftSchema,
	cloudAuthStatusResponseSchema,
	deviceSchema,
	policyMutationSchema
} from './schemas';

describe('api schemas', () => {
	test('device schema accepts metadata-backed topology fields', () => {
		const parsed = deviceSchema.parse({
			id: 'svc:web',
			kind: 'service',
			name: 'svc:web',
			ip: '100.100.0.1',
			tailscaleIps: ['100.100.0.1'],
			os: '',
			online: true,
			owner: '',
			roles: ['admin'],
			tags: ['tag:web'],
			shared: true,
			subnetRouter: false,
			routedSubnets: [],
			postureAttrs: {
				'custom:tier': 'prod',
				'custom:score': 90,
				'custom:encrypted': true
			}
		});

		expect(parsed.kind).toBe('service');
		expect(parsed.shared).toBe(true);
		expect(parsed.postureAttrs?.['custom:tier']).toBe('prod');
	});

	test('policy mutation schema accepts posture-aware ACLs, grants, and posture upserts', () => {
		expect(
			policyMutationSchema.parse({
				type: 'append-acl',
				rule: {
					action: 'accept',
					src: ['group:eng'],
					dst: ['tag:web:443'],
					srcPosture: ['posture:trusted']
				}
			}).rule?.srcPosture
		).toEqual(['posture:trusted']);

		expect(
			policyMutationSchema.parse({
				type: 'append-grant',
				grant: {
					src: ['group:eng'],
					dst: ['10.10.0.0/24'],
					ip: ['tcp:443'],
					srcPosture: ['posture:trusted'],
					via: ['tag:router']
				}
			}).grant?.via
		).toEqual(['tag:router']);

		expect(
			policyMutationSchema.parse({
				type: 'upsert-posture',
				key: 'posture:trusted',
				posture: ["node:os == 'macos'"]
			}).posture
		).toEqual(["node:os == 'macos'"]);
	});

	test('acl draft and cloud status schemas accept new branch fields', () => {
		expect(
			aclDraftSchema.parse({
				action: 'accept',
				src: ['group:eng'],
				dst: ['tag:web:443'],
				srcPosture: ['posture:trusted']
			}).srcPosture
		).toEqual(['posture:trusted']);

		const status = cloudAuthStatusResponseSchema.parse({
			authenticated: true,
			hasPolicy: true,
			canEditPolicy: false,
			hasAppCapabilityGrant: false,
			appCapability: 'tailor.example.ts.net/cap/admin',
			needsSetupGrant: true,
			bootstrapActive: true,
			bootstrapExpiresAt: '2030-01-01T00:00:00Z',
			statusMessage: 'setup required',
			setupGrantSnippet: '{"grants":[]}'
		});

		expect(status.needsSetupGrant).toBe(true);
		expect(status.setupGrantSnippet).toContain('grants');
	});
});
