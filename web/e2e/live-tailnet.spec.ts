import { expect, test } from '@playwright/test';

import { baseURL } from './helpers/env';
import {
	alternatePerspective,
	chooseDevice,
	closePolicyEditor,
	discardPolicyEditor,
	graphSummaryLinkCount,
	openPolicyEditor,
	requireAclEditing,
	testDestination,
	validatePolicyEditor
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
	await discardPolicyEditor(page);
});

test.afterEach(async ({ page }) => {
	await discardPolicyEditor(page);
	await closePolicyEditor(page).catch(() => {});
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
	await expect(page.getByRole('button', { name: 'Edit policy' })).toBeVisible();
});

test('policy editor validates HuJSON and shows graph preview', async ({ page }) => {
	await openPolicyEditor(page);
	const editor = page.getByRole('complementary', { name: 'Policy editor' });
	const textarea = editor.getByRole('textbox', { name: 'Policy HuJSON' });
	const current = await textarea.inputValue();
	const marker = `"tailor-e2e-marker-${Date.now()}"`;
	await textarea.fill(`${current}\n// ${marker}\n`);
	await validatePolicyEditor(page);
	await expect(page.getByLabel('Graph summary')).toContainText('Preview');
	await closePolicyEditor(page);
	await expect(page.getByRole('button', { name: 'Save validated policy' })).toBeVisible();
	await expect(page.getByLabel('Graph summary')).toContainText('Preview');
	await openPolicyEditor(page);
	await editor.getByRole('button', { name: 'Discard', exact: true }).click();
	await expect(page.getByRole('button', { name: 'Save validated policy' })).toBeHidden();
	await expect(page.getByLabel('Graph summary')).not.toContainText('Preview');
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

test('focused device view shows policy links for the selected node', async ({ page }) => {
	await chooseDevice(page, 'alice-laptop');
	const focusedLinks = await graphSummaryLinkCount(page);
	expect(focusedLinks).toBeGreaterThan(0);

	await page.getByLabel('Graph summary').getByRole('button', { name: 'All', exact: true }).click();
	const allLinks = await graphSummaryLinkCount(page);
	expect(allLinks).toBeGreaterThanOrEqual(focusedLinks);
});

test('spawned device shows policy links without page reload', async ({ page, request }) => {
	const health = await request.get(`${baseURL}/api/health`);
	expect(health.ok()).toBeTruthy();
	const meta = (await health.json()) as { build?: string };
	test.skip(meta.build !== 'dev', 'requires tailor built with -tags dev');

	await chooseDevice(page, 'k8s-staging-worker-01');

	const spawnName = `playwright-spawn-${Date.now()}`;
	const spawn = await request.post(`${baseURL}/api/dev/spawn-devices`, {
		data: {
			specs: [
				{
					name: spawnName,
					owner: 'ops@demo.tailor.ts.net',
					os: 'linux',
					tags: ['tag:k8s-staging', 'tag:ci'],
					online: true
				}
			]
		}
	});
	expect(spawn.ok()).toBeTruthy();

	await expect
		.poll(
			async () => {
				const button = page.getByRole('button', { name: spawnName, exact: true }).first();
				if (!(await button.isVisible())) {
					return false;
				}
				await button.click();
				return (await graphSummaryLinkCount(page)) > 0;
			},
			{ timeout: 15_000, message: 'spawned node should appear with policy links' }
		)
		.toBe(true);
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
