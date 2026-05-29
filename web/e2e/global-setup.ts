import { request as playwrightRequest } from '@playwright/test';

import { baseURL } from './helpers/env';
import './helpers/load-env';

const apiKey = process.env.TAILSCALE_API_KEY?.trim();
const configuredTailnet = process.env.TAILOR_TAILNET?.trim();

export default async function globalSetup() {
	if (!apiKey) {
		throw new Error(
			'Missing TAILSCALE_API_KEY. Copy web/.env.example to web/.env and add your tskey-api-… key.'
		);
	}

	const ctx = await playwrightRequest.newContext({ baseURL });

	try {
		const health = await ctx.get('/api/health');
		if (!health.ok()) {
			throw new Error(
				`Tailor backend is not reachable at ${baseURL}. Playwright should start it via webServer — check Go build output.`
			);
		}

		if (process.env.TAILOR_E2E_SKIP_GLOBAL_AUTH === '1') {
			return;
		}

		const statusRes = await ctx.get('/api/cloud/status');
		if (!statusRes.ok()) {
			throw new Error(
				`GET /api/cloud/status failed: ${statusRes.status()} ${await statusRes.text()}`
			);
		}

		const status = (await statusRes.json()) as { authenticated?: boolean; tailnet?: string };
		if (status.authenticated) {
			return;
		}

		let tailnet = configuredTailnet;
		if (!tailnet) {
			if (apiKey === 'tskey-api-tailor-dev') {
				tailnet = 'demo.tailor.ts.net';
			} else {
				const topology = await ctx.get('/api/topology');
				if (!topology.ok()) {
					throw new Error(
						'Could not resolve tailnet from /api/topology. Set TAILOR_TAILNET in web/.env.'
					);
				}
				const topo = (await topology.json()) as { tailnet?: string };
				tailnet = topo.tailnet?.trim();
			}
		}

		if (!tailnet) {
			throw new Error('Tailnet is required. Set TAILOR_TAILNET in web/.env.');
		}

		const auth = await ctx.post('/api/cloud/auth', {
			data: { tailnet, apiKey }
		});
		if (!auth.ok()) {
			throw new Error(`POST /api/cloud/auth failed: ${auth.status()} ${await auth.text()}`);
		}

		const body = (await auth.json()) as { authenticated?: boolean };
		if (!body.authenticated) {
			throw new Error('ACL editing was not enabled after auth. Check your API key and tailnet.');
		}
	} finally {
		await ctx.dispose();
	}
}
