import { expect, test, type Page } from '@playwright/test';

const capability = 'tailor.example.ts.net/cap/admin';
const snippet = `{
  "src": ["autogroup:owner", "autogroup:admin"],
  "dst": ["tag:tailor-acl-service"],
  "ip": ["tcp:443"],
  "app": {"${capability}": [{"actions": ["admin"]}]}
}`;

async function mockCommonAPI(page: Page, cloudStatus: Record<string, unknown>) {
	await page.route('**/api/health', (route) => route.fulfill({ json: { status: 'ok' } }));
	await page.route('**/api/cloud/status', (route) => route.fulfill({ json: cloudStatus }));
	await page.route('**/api/topology', (route) =>
		route.fulfill({ json: { tailnet: 'example.ts.net', devices: [], edges: [], stagedDrafts: [] } })
	);
}

test('viewer sees explicit view-only state and no editing control', async ({ page }) => {
	await mockCommonAPI(page, {
		authenticated: true,
		tailnet: 'example.ts.net',
		hasPolicy: true,
		callerRole: 'viewer',
		canEditPolicy: false,
		hasAppCapabilityGrant: true,
		appCapability: capability,
		statusMessage: 'API key accepted, but your current device or user is view-only.'
	});

	await page.goto('/');
	await expect(page.getByText('View-only', { exact: true })).toBeVisible();
	await expect(page.getByRole('status')).toContainText('API key accepted');
	await expect(page.getByRole('button', { name: 'Edit policy' })).toBeHidden();
});

test('auth dialog is a keyboard-accessible modal', async ({ page }) => {
	await mockCommonAPI(page, { authenticated: false, canEditPolicy: false, callerRole: 'full' });

	await page.goto('/');
	const opener = page.getByRole('button', { name: 'Enable ACL Editing' });
	await opener.click();
	const dialog = page.getByRole('dialog', { name: 'Enable ACL Editing' });
	await expect(dialog).toBeVisible();

	// Focus moves into the dialog and Tab never escapes it.
	await expect(dialog.locator(':focus')).toHaveCount(1);
	for (let i = 0; i < 8; i++) {
		await page.keyboard.press('Tab');
		await expect(dialog.locator(':focus')).toHaveCount(1);
	}

	// Clicking the backdrop drops focus out of the dialog; Escape must still close it
	// and return focus to the button that opened it.
	await page.mouse.click(5, 5);
	await expect(dialog.locator(':focus')).toHaveCount(0);
	await page.keyboard.press('Escape');
	await expect(dialog).toBeHidden();
	await expect(opener).toBeFocused();
});

test('setup recommendation supports cancel, recommended, and edited grants', async ({ page }) => {
	await mockCommonAPI(page, { authenticated: false, canEditPolicy: false, callerRole: 'full' });
	await page.route('**/api/cloud/auth', (route) =>
		route.fulfill({
			json: {
				authenticated: true,
				tailnet: 'example.ts.net',
				hasPolicy: true,
				callerRole: 'viewer',
				canEditPolicy: false,
				needsSetupGrant: true,
				appCapability: capability,
				setupGrantSnippet: snippet,
				statusMessage:
					'Tailor should add an app capability grant so ACL editing access is controlled by your tailnet policy.'
			}
		})
	);

	let setupBody: unknown;
	await page.route('**/api/cloud/setup-grant', async (route) => {
		setupBody = route.request().postDataJSON();
		await route.fulfill({
			json: {
				tailnet: 'example.ts.net',
				appCapability: capability,
				hasAppCapabilityGrant: true,
				callerRole: 'viewer',
				canEditPolicy: false,
				statusMessage: 'Tailor access was configured, but your current device or user is view-only.'
			}
		});
	});

	await page.goto('/');
	await openSetupDialog(page);
	const dialog = page.getByRole('dialog', { name: 'Configure Tailor Access' });
	await expect(dialog).toContainText(capability);
	await dialog.getByRole('button', { name: 'Not now' }).click();
	await expect(dialog).toBeHidden();

	await page.reload();
	await page.getByRole('button', { name: 'Enable ACL Editing' }).click();
	await page.getByLabel('Tailscale API Key').fill('tskey-api-test');
	await page.getByRole('button', { name: 'Fetch Policy' }).click();
	await dialog.getByRole('button', { name: 'Add recommended grant' }).click();
	await expect(dialog).toBeHidden();
	expect(setupBody).toBeNull();
	await expect(page.getByText('View-only', { exact: true })).toBeVisible();

	await page.reload();
	await openSetupDialog(page);
	await dialog.getByRole('button', { name: 'Edit grant' }).click();
	const edited = snippet.replace('["autogroup:owner", "autogroup:admin"]', '["autogroup:owner"]');
	await dialog.getByLabel('App capability grant').fill(edited);
	await dialog.getByRole('button', { name: 'Save edited grant' }).click();
	expect(setupBody).toEqual({ grant: JSON.parse(edited) });
});

test('editor who declines the optional grant still gets their policy loaded', async ({ page }) => {
	await mockCommonAPI(page, { authenticated: false, canEditPolicy: false, callerRole: 'full' });
	await page.route('**/api/cloud/auth', (route) =>
		route.fulfill({
			json: {
				authenticated: true,
				tailnet: 'example.ts.net',
				hasPolicy: true,
				callerRole: 'full',
				canEditPolicy: true,
				needsSetupGrant: true,
				appCapability: capability,
				setupGrantSnippet: snippet,
				statusMessage: 'As a tailnet admin you can already edit.'
			}
		})
	);
	let policyFetches = 0;
	await page.route('**/api/policy', (route) => {
		policyFetches++;
		return route.fulfill({
			json: { tailnet: 'example.ts.net', hujson: '{"grants":[]}', revision: 'rev-1' }
		});
	});
	await page.route('**/api/policy/**', (route) => route.fulfill({ json: { drafts: [] } }));

	await page.goto('/');
	await page.getByRole('button', { name: 'Enable ACL Editing' }).click();
	await page.getByLabel('Tailscale API Key').fill('tskey-api-test');
	await page.getByRole('button', { name: 'Fetch Policy' }).click();
	const dialog = page.getByRole('dialog', { name: 'Configure Tailor Access' });
	await dialog.getByRole('button', { name: 'Not now' }).click();
	await expect(dialog).toBeHidden();

	await expect.poll(() => policyFetches).toBeGreaterThan(0);
	await expect(page.getByRole('button', { name: 'Edit policy' })).toBeVisible();
});

test('non-admin API key submission is refused with the reason', async ({ page }) => {
	await mockCommonAPI(page, { authenticated: false, canEditPolicy: false, callerRole: 'viewer' });
	await page.route('**/api/cloud/auth', (route) =>
		route.fulfill({
			status: 403,
			json: {
				error:
					'Only tailnet Owners, Admins and Network admins can connect Tailor to the Tailscale API.'
			}
		})
	);

	await page.goto('/');
	await page.getByRole('button', { name: 'Enable ACL Editing' }).click();
	const dialog = page.getByRole('dialog', { name: 'Enable ACL Editing' });
	await dialog.getByLabel('Tailscale API Key').fill('tskey-api-test');
	await dialog.getByRole('button', { name: 'Fetch Policy' }).click();

	await expect(dialog).toContainText('Owners, Admins and Network admins');
	await expect(page.getByRole('button', { name: 'Edit policy' })).toBeHidden();
});

async function openSetupDialog(page: Page) {
	await page.getByRole('button', { name: 'Enable ACL Editing' }).click();
	await page.getByLabel('Tailscale API Key').fill('tskey-api-test');
	await page.getByRole('button', { name: 'Fetch Policy' }).click();
	await expect(page.getByRole('dialog', { name: 'Configure Tailor Access' })).toBeVisible();
}
