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

test('bootstrap fallback shows temporary access warning and grant', async ({ page }) => {
	await mockCommonAPI(page, {
		authenticated: true,
		tailnet: 'example.ts.net',
		hasPolicy: true,
		callerRole: 'viewer',
		canEditPolicy: true,
		bootstrapActive: true,
		bootstrapExpiresAt: '2030-01-01T00:00:00Z',
		setupGrantSnippet: snippet,
		statusMessage:
			'Tailor could not apply the app capability grant automatically. ACL editing is temporarily available in this browser session.'
	});
	await page.route('**/api/policy', (route) =>
		route.fulfill({ json: { tailnet: 'example.ts.net', hujson: '{"grants":[]}' } })
	);
	await page.route('**/api/policy/**', (route) => route.fulfill({ json: { drafts: [] } }));

	await page.goto('/');
	await expect(page.getByRole('status')).toContainText('temporarily available');
	await expect(page.getByText(capability)).toBeVisible();
	await expect(page.getByRole('button', { name: 'Edit policy' })).toBeVisible();
});

async function openSetupDialog(page: Page) {
	await page.getByRole('button', { name: 'Enable ACL Editing' }).click();
	await page.getByLabel('Tailscale API Key').fill('tskey-api-test');
	await page.getByRole('button', { name: 'Fetch Policy' }).click();
	await expect(page.getByRole('dialog', { name: 'Configure Tailor Access' })).toBeVisible();
}
