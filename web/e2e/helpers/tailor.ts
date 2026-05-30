import { expect, type APIRequestContext, type Page } from '@playwright/test';

import { alternatePerspective, baseURL, configuredTailnet, testDestination } from './env';

export async function requireAclEditing(page: Page) {
	await page.goto('/');
	await expect(page.getByRole('button', { name: 'Edit policy' })).toBeVisible();
}

export async function resolveTailnetName(request: APIRequestContext): Promise<string> {
	if (configuredTailnet) {
		return configuredTailnet;
	}
	const topology = await request.get(`${baseURL}/api/topology`);
	expect(topology.ok()).toBeTruthy();
	const topo = (await topology.json()) as { tailnet?: string };
	const tailnet = topo.tailnet?.trim();
	expect(
		tailnet,
		'Could not resolve tailnet from /api/topology. Set TAILOR_TAILNET in web/.env.'
	).toBeTruthy();
	return tailnet!;
}

export async function enableAclEditingViaUI(
	page: Page,
	options: { apiKey: string; tailnet: string }
) {
	await page.goto('/');
	if (await page.getByText(/Demo tailnet/).isVisible()) {
		throw new Error(
			'Tailor backend is authenticated in demo mode. Stop any running tailor process on the E2E port and re-run pnpm test:e2e:production.'
		);
	}

	const enableButton = page.getByRole('button', { name: 'Enable ACL Editing' });
	const editPolicy = page.getByRole('button', { name: 'Edit policy' });
	if (await editPolicy.isVisible()) {
		await expect(enableButton).toBeHidden();
		return;
	}

	await expect(enableButton).toBeVisible({ timeout: 30_000 });
	await enableButton.click();
	const dialog = page.getByRole('dialog', { name: 'Enable ACL Editing' });
	await expect(dialog).toBeVisible();
	await dialog.getByPlaceholder('tskey-api-...').fill(options.apiKey);
	await dialog.getByPlaceholder('example.com or -').fill(options.tailnet);
	await dialog.getByRole('button', { name: 'Fetch Policy' }).click();

	await expect(editPolicy).toBeVisible({ timeout: 90_000 });
	await expect(enableButton).toBeHidden();
	await expect(page.getByText(/Demo tailnet/)).toBeHidden();
}

export async function setPolicyEditorText(page: Page, hujson: string) {
	await openPolicyEditor(page);
	const editor = page.getByRole('complementary', { name: 'Policy editor' });
	await editor.getByRole('textbox', { name: 'Policy HuJSON' }).fill(hujson);
}

export async function saveValidatedPolicy(page: Page) {
	const headerSave = page.getByRole('button', { name: 'Save validated policy' });
	if (await headerSave.isVisible()) {
		await headerSave.click();
	} else {
		const editor = page.getByRole('complementary', { name: 'Policy editor' });
		await editor.getByRole('button', { name: 'Save policy' }).click();
	}
	await expect(page.getByRole('button', { name: 'Save validated policy' })).toBeHidden({
		timeout: 90_000
	});
}

export async function openPolicyEditor(page: Page) {
	await page.getByRole('button', { name: 'Edit policy' }).click();
	await expect(page.getByRole('complementary', { name: 'Policy editor' })).toBeVisible();
}

export async function closePolicyEditor(page: Page) {
	const editor = page.getByRole('complementary', { name: 'Policy editor' });
	if (!(await editor.isVisible())) {
		return;
	}
	await page.getByRole('button', { name: 'Close policy editor' }).click();
	await expect(editor).toBeHidden();
}

export async function validatePolicyEditor(page: Page) {
	const editor = page.getByRole('complementary', { name: 'Policy editor' });
	await expect(editor.getByText('Unsaved changes')).toBeVisible({ timeout: 15_000 });
	await editor.getByRole('button', { name: 'Validate', exact: true }).click();
	await expect(editor.getByText('Validated', { exact: true })).toBeVisible({ timeout: 60_000 });
	await expect(page.getByLabel('Graph summary')).toContainText('Preview', { timeout: 60_000 });
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
