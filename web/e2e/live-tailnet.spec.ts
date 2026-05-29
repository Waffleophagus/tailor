import { expect, test } from '@playwright/test';

import { baseURL, superUserPerspective } from './helpers/env';
import {
	addGeneralAccessRule,
	alternatePerspective,
	clearScenario,
	closeAccessControls,
	discardDraft,
	graphDebugCounts,
	graphSummaryLinkCount,
	requireAclEditing,
	scenarioBar,
	scenarioDiffButton,
	scenarioDraftButton,
	simulateAsDeviceOwner,
	simulatePerspective,
	testDestination,
	validateDraft
} from './helpers/tailor';

test.describe.configure({ mode: 'serial' });

test.beforeAll(async ({ request }) => {
	const health = await request.get(`${baseURL}/api/health`);
	expect(health.ok()).toBeTruthy();

	const cloud = await request.get(`${baseURL}/api/cloud/status`);
	expect(cloud.ok()).toBeTruthy();
	const status = (await cloud.json()) as { authenticated?: boolean };
	expect(
		status.authenticated,
		'ACL editing should be enabled by globalSetup via TAILSCALE_API_KEY in web/.env'
	).toBe(true);
});

test.beforeEach(async ({ page }) => {
	await requireAclEditing(page);
	await discardDraft(page);
	await clearScenario(page);
});

test.afterEach(async ({ page }) => {
	await discardDraft(page);
	await clearScenario(page);
});

test('loads topology and policy for an authenticated tailnet', async ({ request, page }) => {
	const topology = await request.get(`${baseURL}/api/topology`);
	expect(topology.ok()).toBeTruthy();
	const topo = (await topology.json()) as { devices?: unknown[]; tailnet?: string };
	expect(topo.devices?.length ?? 0).toBeGreaterThan(0);
	expect(topo.tailnet).toBeTruthy();

	const policy = await request.get(`${baseURL}/api/policy`);
	expect(policy.ok()).toBeTruthy();
	const raw = (await policy.json()) as { hujson?: string };
	expect(raw.hujson?.length ?? 0).toBeGreaterThan(10);

	await page.goto('/');
	await expect(page.getByRole('region', { name: 'Topology graph' })).toBeVisible();
	await expect(page.getByRole('button', { name: 'Access controls' })).toBeVisible();
});

test('policy workbench stages an ACL mutation and evaluates draft impact', async ({ page }) => {
	await addGeneralAccessRule(page, {
		sources: alternatePerspective,
		destinations: testDestination,
		port: '443'
	});

	await expect(page.getByText('General access rules — ACL rule added')).toBeVisible();
	await expect(scenarioDraftButton(page)).toBeEnabled();

	await simulatePerspective(page, alternatePerspective);
	await closeAccessControls(page);

	const bar = scenarioBar(page);
	await bar.getByRole('button', { name: 'Focused', exact: true }).click();
	const ghostToggle = bar.getByRole('checkbox', { name: 'Show denied links' });
	await expect(ghostToggle).toBeVisible();
	await ghostToggle.uncheck();
	await ghostToggle.check();
	await bar.getByRole('button', { name: 'All connections', exact: true }).click();
	await expect(ghostToggle).toBeHidden();

	await scenarioDraftButton(page).click();
	await scenarioDiffButton(page).click();

	await expect(
		page
			.getByRole('region', { name: 'Staged policy change' })
			.getByText(/Draft impact:|Impact preview unavailable/)
	).toBeVisible({ timeout: 60_000 });
	await validateDraft(page);
});

test('mutate API appends and evaluates a reversible ACL draft', async ({ request }) => {
	const policyRes = await request.get(`${baseURL}/api/policy`);
	expect(policyRes.ok()).toBeTruthy();
	const saved = (await policyRes.json()) as { hujson: string };

	const mutate = await request.post(`${baseURL}/api/policy/mutate`, {
		data: {
			hujson: saved.hujson,
			mutation: {
				type: 'append-acl',
				rule: {
					action: 'accept',
					src: [alternatePerspective],
					dst: [`${testDestination}:19999`]
				}
			}
		}
	});
	expect(mutate.ok()).toBeTruthy();
	const draft = (await mutate.json()) as { hujson: string };
	expect(draft.hujson).toContain('19999');
	expect(draft.hujson).toContain(alternatePerspective);

	const evaluate = await request.post(`${baseURL}/api/policy/evaluate-draft`, {
		data: {
			hujson: draft.hujson,
			perspective: alternatePerspective
		}
	});
	expect(evaluate.ok()).toBeTruthy();
	const impact = (await evaluate.json()) as { added?: unknown[]; unchanged?: unknown[] };
	expect((impact.added?.length ?? 0) + (impact.unchanged?.length ?? 0)).toBeGreaterThan(0);

	const validate = await request.post(`${baseURL}/api/policy/validate`, {
		data: { hujson: draft.hujson }
	});
	expect(validate.ok()).toBeTruthy();
	const validation = (await validate.json()) as { valid?: boolean };
	expect(validation.valid).toBe(true);
});

test('super user simulation shows full topology nodes and broad connectivity', async ({
	request,
	page
}) => {
	const health = await request.get(`${baseURL}/api/health`);
	expect(health.ok()).toBeTruthy();
	const meta = (await health.json()) as { build?: string };
	test.skip(meta.build !== 'dev', 'requires tailor built with -tags dev');

	const topology = await request.get(`${baseURL}/api/topology`);
	expect(topology.ok()).toBeTruthy();
	const topo = (await topology.json()) as { devices?: { id: string }[] };
	const deviceCount = topo.devices?.length ?? 0;
	expect(deviceCount).toBeGreaterThan(10);

	const policyRes = await request.get(`${baseURL}/api/policy`);
	expect(policyRes.ok()).toBeTruthy();
	const saved = (await policyRes.json()) as { hujson: string };
	test.skip(
		!saved.hujson.includes('group:superuser'),
		'requires demo tailnet seeded with group:superuser (*:*) ACL'
	);

	const evaluate = await request.post(`${baseURL}/api/policy/evaluate-draft`, {
		data: {
			hujson: saved.hujson,
			perspective: superUserPerspective
		}
	});
	expect(evaluate.ok()).toBeTruthy();
	const impact = (await evaluate.json()) as {
		added?: unknown[];
		unchanged?: unknown[];
		visibleDeviceIds?: string[];
		unresolvedSelectors?: unknown[];
		unsupportedSections?: unknown[];
	};
	expect(Array.isArray(impact.unresolvedSelectors)).toBe(true);
	expect(Array.isArray(impact.unsupportedSections)).toBe(true);
	const superEdgeCount = (impact.added?.length ?? 0) + (impact.unchanged?.length ?? 0);
	const superVisibleCount = impact.visibleDeviceIds?.length ?? 0;
	expect(superEdgeCount).toBeGreaterThan(deviceCount - 4);
	expect(superVisibleCount).toBeGreaterThan(deviceCount - 2);

	await simulateAsDeviceOwner(page, 'superadmin-console');

	await expect
		.poll(async () => graphDebugCounts(page), { timeout: 60_000 })
		.toMatchObject({ nodes: superVisibleCount });

	const focusedLinks = await graphSummaryLinkCount(page);
	expect(focusedLinks).toBeGreaterThan(8);

	await scenarioBar(page).getByRole('button', { name: 'All connections', exact: true }).click();
	await expect
		.poll(async () => graphSummaryLinkCount(page), { timeout: 15_000 })
		.toBeGreaterThanOrEqual(focusedLinks);

	const aliceEvaluate = await request.post(`${baseURL}/api/policy/evaluate-draft`, {
		data: {
			hujson: saved.hujson,
			perspective: alternatePerspective
		}
	});
	expect(aliceEvaluate.ok()).toBeTruthy();
	const aliceImpact = (await aliceEvaluate.json()) as {
		added?: unknown[];
		unchanged?: unknown[];
		visibleDeviceIds?: string[];
	};
	const aliceEdgeCount = (aliceImpact.added?.length ?? 0) + (aliceImpact.unchanged?.length ?? 0);
	const aliceVisibleCount = aliceImpact.visibleDeviceIds?.length ?? 0;
	expect(superEdgeCount).toBeGreaterThan(aliceEdgeCount);
	expect(superVisibleCount).toBeGreaterThan(aliceVisibleCount);
});

test('dev spawn endpoint adds devices to the demo tailnet', async ({ request }) => {
	const health = await request.get(`${baseURL}/api/health`);
	expect(health.ok()).toBeTruthy();
	const meta = (await health.json()) as { build?: string };
	test.skip(meta.build !== 'dev', 'requires tailor built with -tags dev');

	const before = await request.get(`${baseURL}/api/topology`);
	expect(before.ok()).toBeTruthy();
	const initial = (await before.json()) as { devices?: unknown[] };
	const initialCount = initial.devices?.length ?? 0;

	const spawn = await request.post(`${baseURL}/api/dev/spawn-devices`, {
		data: { count: 4, prefix: 'playwright' }
	});
	expect(spawn.ok()).toBeTruthy();
	const body = (await spawn.json()) as { spawned?: unknown[]; devices?: unknown[] };
	expect(body.spawned?.length).toBe(4);
	expect(body.devices?.length).toBe(initialCount + 4);
});
