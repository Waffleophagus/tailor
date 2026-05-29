import { expect, type Page } from '@playwright/test';

import { alternatePerspective, testDestination } from './env';

export async function requireAclEditing(page: Page) {
	await page.goto('/');
	await expect(page.getByRole('button', { name: 'Edit policy' })).toBeVisible();
}

export async function openPolicyEditor(page: Page) {
	await page.getByRole('button', { name: 'Edit policy' }).click();
	await expect(page.getByRole('complementary', { name: 'Policy editor' })).toBeVisible();
}

export async function closePolicyEditor(page: Page) {
	await page.getByRole('button', { name: 'Close policy editor' }).click();
	await expect(page.getByRole('complementary', { name: 'Policy editor' })).toBeHidden();
}

export async function validatePolicyEditor(page: Page) {
	const editor = page.getByRole('complementary', { name: 'Policy editor' });
	await editor.getByRole('button', { name: 'Validate', exact: true }).click();
	await expect(editor.getByText('Validated', { exact: true })).toBeVisible({ timeout: 60_000 });
}

export async function discardPolicyEditor(page: Page) {
	if (await page.getByRole('button', { name: 'Save validated policy' }).isVisible()) {
		await openPolicyEditor(page);
	}
	const editor = page.getByRole('complementary', { name: 'Policy editor' });
	if (await editor.isVisible()) {
		const discard = editor.getByRole('button', { name: 'Discard', exact: true });
		if (await discard.isEnabled()) {
			await discard.click();
		}
	}
}

export async function chooseDevice(page: Page, deviceName: string) {
	await page.getByRole('button', { name: deviceName, exact: true }).first().click();
	await expect(page.getByLabel('Graph summary')).toContainText('focus', { timeout: 15_000 });
}

export interface PlaywrightGraphDebugEdge {
	id: string;
	from: string;
	to: string;
	classes: string[];
	style: {
		lineColor: string;
		lineStyle: string;
		width: number;
	};
}

export async function graphDebugCounts(page: Page) {
	return page.evaluate(() => {
		const api = (
			window as unknown as { __tailorGraphDebug?: { nodeCount: number; edgeCount: number } }
		).__tailorGraphDebug;
		return api ?? { nodeCount: 0, edgeCount: 0 };
	});
}

export async function graphSummaryLinkCount(page: Page) {
	const text = await page.getByLabel('Graph summary').innerText();
	const match = text.match(/(\d+)\s*links/i);
	return match ? Number(match[1]) : 0;
}

export { alternatePerspective, testDestination };
