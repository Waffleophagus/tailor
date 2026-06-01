import { expect, test } from '@playwright/test';

import { baseURL } from './helpers/env';

test.describe('mobile graph UI', () => {
	test.use({ viewport: { width: 390, height: 844 } });

	test.beforeAll(async ({ request }) => {
		const health = await request.get(`${baseURL}/api/health`);
		expect(health.ok()).toBeTruthy();
	});

	test('shows graph explorer without desktop ACL controls', async ({ page }) => {
		await page.goto('/');

		await expect(page.getByRole('region', { name: 'Topology graph' })).toBeVisible();
		await expect(page.getByRole('navigation', { name: 'Graph controls' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Filters' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Details' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Legend' })).toBeVisible();

		await expect(page.getByRole('button', { name: 'Edit policy' })).toBeHidden();
		await expect(page.getByRole('button', { name: 'Enable ACL Editing' })).toBeHidden();
		await expect(page.getByText('Policy editing is available on desktop.')).toBeVisible();
	});

	test('opens filters sheet from bottom bar', async ({ page }) => {
		await page.goto('/');
		await expect(page.getByRole('navigation', { name: 'Graph controls' })).toBeVisible();

		await page.getByRole('button', { name: 'Filters' }).click();
		await expect(page.getByRole('dialog', { name: 'Filters' })).toBeVisible();
		await expect(page.getByRole('heading', { name: 'View' })).toBeVisible();
	});
});
