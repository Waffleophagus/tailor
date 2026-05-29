import { expect, type Locator, type Page } from '@playwright/test';

import {
	alternatePerspective,
	defaultPerspective,
	superUserPerspective,
	testDestination
} from './env';

function scenarioBar(page: Page | Locator): Locator {
	return page.getByLabel('Policy scenario');
}

export function scenarioDraftButton(page: Page): Locator {
	return scenarioBar(page).getByRole('button', { name: 'Draft', exact: true });
}

export function scenarioDiffButton(page: Page): Locator {
	return scenarioBar(page).getByRole('button', { name: 'Diff', exact: true });
}

export { scenarioBar };

export async function requireAclEditing(page: Page) {
	await page.goto('/');
	await expect(page.getByRole('button', { name: 'Access controls' })).toBeVisible();
}

export async function openAccessControls(page: Page) {
	await page.getByRole('button', { name: 'Access controls' }).click();
	await expect(page.getByRole('complementary', { name: 'Access controls' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'General access rules' })).toBeVisible();
}

export async function closeAccessControls(page: Page) {
	await page.getByRole('button', { name: 'Close access controls' }).click();
	await expect(page.getByRole('complementary', { name: 'Access controls' })).toBeHidden();
}

export async function addGeneralAccessRule(
	page: Page,
	options: {
		sources?: string;
		destinations?: string;
		port?: '443' | '22' | '*';
	} = {}
) {
	const sources = options.sources ?? defaultPerspective;
	const destinations = options.destinations ?? testDestination;
	const port = options.port ?? '443';

	await openAccessControls(page);
	await page.getByRole('button', { name: '+ Add rule' }).click();
	await page.getByRole('textbox', { name: 'Sources' }).fill(sources);
	await page.getByRole('textbox', { name: 'Destinations' }).fill(destinations);
	await page.getByRole('combobox', { name: /Port and protocol/ }).selectOption(port);
	await expect(page.getByRole('button', { name: 'Save to draft' })).toBeEnabled();
	await page.getByRole('button', { name: 'Save to draft' }).click();
	await expect(page.getByRole('region', { name: 'Staged policy change' })).toBeVisible({
		timeout: 60_000
	});
	await expect(scenarioDraftButton(page)).toBeEnabled({ timeout: 60_000 });
}

export async function simulatePerspective(page: Page, perspective: string = defaultPerspective) {
	const bar = scenarioBar(page);
	const input = bar.getByRole('combobox', { name: 'Viewing as' });
	const simulate = bar.getByRole('button', { name: 'Simulate' });
	await input.click();
	await input.fill(perspective);
	await expect(simulate).toBeEnabled({ timeout: 15_000 });
	await simulate.click();
	await expect
		.poll(async () => (await page.getByLabel('Graph summary').innerText()).toLowerCase(), {
			timeout: 60_000
		})
		.toContain('simulated');
}

export async function simulateAsDeviceOwner(page: Page, deviceName: string) {
	await page
		.getByRole('listitem')
		.filter({ has: page.getByRole('button', { name: deviceName, exact: true }) })
		.getByRole('button', { name: 'View as' })
		.click();
	await expect
		.poll(async () => (await page.getByLabel('Graph summary').innerText()).toLowerCase(), {
			timeout: 60_000
		})
		.toContain('simulated');
}

export async function discardDraft(page: Page) {
	const tray = page.getByRole('region', { name: 'Staged policy change' });
	if (await tray.isVisible()) {
		await page.getByRole('button', { name: 'Discard' }).click();
		await expect(tray).toBeHidden();
	}
}

export async function validateDraft(page: Page) {
	const tray = page.getByRole('region', { name: 'Staged policy change' });
	await tray.getByRole('button', { name: 'Validate', exact: true }).click();
	await expect(tray.getByText('Validated', { exact: true })).toBeVisible({ timeout: 60_000 });
}

export async function clearScenario(page: Page) {
	const clear = page.getByRole('button', { name: 'Clear' });
	if (await clear.isEnabled()) {
		await clear.click();
	}
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
		opacity: number;
	};
}

export async function graphDebugSnapshot(page: Page): Promise<PlaywrightGraphDebugEdge[]> {
	return page.evaluate(() => {
		const snapshot = window.__tailorGraphDebug?.();
		return snapshot?.edges ?? [];
	});
}

export async function graphSummaryLinkCount(page: Page): Promise<number> {
	const text = await page.getByLabel('Graph summary').innerText();
	const match = text.match(/(\d+)\s+links/i);
	return match ? Number.parseInt(match[1], 10) : 0;
}

export async function graphDebugCounts(page: Page): Promise<{ nodes: number; edges: number }> {
	return page.evaluate(() => {
		const snapshot = window.__tailorGraphDebug?.();
		return {
			nodes: snapshot?.nodes?.length ?? 0,
			edges: snapshot?.edges?.length ?? 0
		};
	});
}

export { alternatePerspective, defaultPerspective, superUserPerspective, testDestination };
