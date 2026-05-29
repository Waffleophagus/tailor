import { expect, test } from '@playwright/test';

import { alternatePerspective, isProductionApiKey, tailscaleApiKey } from './helpers/env';
import {
	appendProbeAclRule,
	fetchPolicyHujson,
	policiesEquivalent,
	policyContainsProbe
} from './helpers/policy';
import {
	enableAclEditingViaUI,
	resolveTailnetName,
	saveValidatedPolicy,
	setPolicyEditorText,
	validatePolicyEditor
} from './helpers/tailor';

test.describe.configure({ mode: 'serial', timeout: 180_000 });

test.beforeAll(() => {
	test.skip(
		!isProductionApiKey(),
		'Set a real TAILSCALE_API_KEY in web/.env (not tskey-api-tailor-dev) to run production ACL save E2E'
	);
});

test('round-trips a real ACL change against Tailscale Cloud', async ({ page, request }) => {
	const tailnet = await resolveTailnetName(request);
	const marker = `tailor-e2e-production-${Date.now()}`;
	const probePort = 37_000 + (Date.now() % 1_000);

	let initialHujson = '';

	try {
		await enableAclEditingViaUI(page, { apiKey: tailscaleApiKey, tailnet });

		const cloudStatus = await request.get('/api/cloud/status');
		expect(cloudStatus.ok()).toBeTruthy();
		const status = (await cloudStatus.json()) as { authenticated?: boolean; devMode?: boolean };
		expect(status.authenticated).toBe(true);
		expect(status.devMode ?? false).toBe(false);

		const initialPolicy = await fetchPolicyHujson(request);
		initialHujson = initialPolicy.hujson;
		expect(initialHujson.length).toBeGreaterThan(10);

		const mutatedHujson = await appendProbeAclRule(request, initialHujson, probePort);
		const editedHujson = `${mutatedHujson}\n// ${marker}\n`;
		expect(editedHujson).toContain(alternatePerspective);
		expect(editedHujson).toContain(`:${probePort}`);

		await setPolicyEditorText(page, editedHujson);
		await validatePolicyEditor(page);
		await saveValidatedPolicy(page);

		const afterSave = await fetchPolicyHujson(request);
		expect(
			policyContainsProbe(afterSave.hujson, probePort, marker),
			'saved policy should include the probe ACL rule'
		).toBe(true);

		await setPolicyEditorText(page, initialHujson);
		await validatePolicyEditor(page);
		await saveValidatedPolicy(page);

		const reverted = await fetchPolicyHujson(request);
		expect(
			policyContainsProbe(reverted.hujson, probePort, marker),
			'reverted policy should not include the probe ACL rule'
		).toBe(false);
		expect(
			policiesEquivalent(initialHujson, reverted.hujson),
			'reverted policy should match the initial snapshot'
		).toBe(true);
	} catch (error) {
		if (initialHujson) {
			const current = await fetchPolicyHujson(request).catch(() => null);
			const needsCleanup =
				current &&
				!policiesEquivalent(initialHujson, current.hujson) &&
				policyContainsProbe(current.hujson, probePort, marker);
			if (needsCleanup) {
				await setPolicyEditorText(page, initialHujson).catch(() => {});
				await validatePolicyEditor(page).catch(() => {});
				await saveValidatedPolicy(page).catch(() => {});
			}
		}
		throw error;
	}
});
