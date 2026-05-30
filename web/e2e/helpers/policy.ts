import { expect, type APIRequestContext } from '@playwright/test';

import { probeAclDraftTargets, probeAclTargetsFromTopology } from './demo';
import { baseURL } from './env';

export interface PolicyResponseBody {
	hujson: string;
	tailnet?: string;
}

export async function fetchPolicyHujson(request: APIRequestContext): Promise<PolicyResponseBody> {
	const response = await request.get(`${baseURL}/api/policy`);
	expect(response.ok(), `GET /api/policy failed: ${response.status()}`).toBeTruthy();
	return (await response.json()) as PolicyResponseBody;
}

/** Loose HuJSON → JSON parse for semantic policy comparison in E2E. */
export function parsePolicyDocument(hujson: string): unknown {
	const withoutComments = hujson.replace(/\/\/[^\n]*/g, '').replace(/\/\*[\s\S]*?\*\//g, '');
	const withoutTrailingCommas = withoutComments.replace(/,\s*([}\]])/g, '$1');
	return JSON.parse(withoutTrailingCommas);
}

export function policiesEquivalent(a: string, b: string): boolean {
	return JSON.stringify(parsePolicyDocument(a)) === JSON.stringify(parsePolicyDocument(b));
}

export async function appendProbeAclRule(
	request: APIRequestContext,
	hujson: string,
	port: number,
	options?: { liveTailnetOnly?: boolean }
): Promise<{ hujson: string; src: string; dst: string }> {
	const probe = options?.liveTailnetOnly
		? await probeAclTargetsFromTopology(request, port)
		: await probeAclDraftTargets(request, port);
	const response = await request.post(`${baseURL}/api/policy/mutate`, {
		data: {
			hujson,
			mutation: {
				type: 'append-acl',
				rule: {
					action: 'accept',
					src: [probe.src],
					dst: [probe.dst]
				}
			}
		}
	});
	expect(response.ok(), `POST /api/policy/mutate failed: ${response.status()}`).toBeTruthy();
	const body = (await response.json()) as { hujson: string };
	return { hujson: body.hujson, src: probe.src, dst: probe.dst };
}

export function policyContainsProbe(hujson: string, port: number, marker: string): boolean {
	return hujson.includes(marker) || hujson.includes(`:${port}`);
}
