import { type APIRequestContext } from '@playwright/test';

import { alternatePerspective, baseURL, testDestination } from './env';

export async function isDemoDevMode(request: APIRequestContext): Promise<boolean> {
	const res = await request.get(`${baseURL}/api/cloud/status`);
	if (!res.ok()) {
		return false;
	}
	const status = (await res.json()) as { devMode?: boolean };
	return status.devMode === true;
}

export interface ProbeAclDraftTargets {
	src: string;
	dst: string;
	perspective: string;
}

/** Rule + perspective for mutate/evaluate-draft E2E on demo or a live tailnet. */
export async function probeAclDraftTargets(
	request: APIRequestContext,
	port = 19_999
): Promise<ProbeAclDraftTargets> {
	if (await isDemoDevMode(request)) {
		return {
			src: alternatePerspective,
			dst: `${testDestination}:${port}`,
			perspective: alternatePerspective
		};
	}
	const live = await probeAclTargetsFromTopology(request, port);
	return { ...live, perspective: live.src };
}

/** Live-tailnet probe rule (never uses demo fixtures). */
export async function probeAclTargetsFromTopology(
	request: APIRequestContext,
	port: number
): Promise<Pick<ProbeAclDraftTargets, 'src' | 'dst'>> {
	const topo = await request.get(`${baseURL}/api/topology`);
	if (!topo.ok()) {
		throw new Error(`GET /api/topology failed: ${topo.status()}`);
	}
	const { devices } = (await topo.json()) as {
		devices?: Array<{ owner?: string; tags?: string[] }>;
	};
	const withOwner = devices?.find((d) => d.owner?.includes('@'));
	const withTag = devices?.find((d) => d.tags?.[0]);
	const src = withOwner?.owner?.trim();
	const tag = withTag?.tags?.[0]?.trim();
	if (!src || !tag) {
		throw new Error(
			'Need at least one user-owned device and one tagged device in topology for ACL probe E2E.'
		);
	}
	return { src, dst: `${tag}:${port}` };
}

export function demoDeviceName(): string {
	return 'alice-laptop';
}
