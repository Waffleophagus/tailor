import { defineConfig, devices } from '@playwright/test';

import { e2eWebServers } from './e2e/helpers/dev-server';
import './e2e/helpers/load-env';

const baseURL = process.env.TAILOR_E2E_BASE_URL ?? 'http://127.0.0.1:5173';
const reuseExistingServer = !process.env.CI;

export default defineConfig({
	testDir: './e2e',
	globalSetup: './e2e/global-setup.ts',
	fullyParallel: false,
	forbidOnly: Boolean(process.env.CI),
	retries: process.env.CI ? 1 : 0,
	workers: 1,
	reporter: [['list']],
	timeout: 120_000,
	expect: { timeout: 30_000 },
	webServer: e2eWebServers(baseURL, reuseExistingServer),
	use: {
		baseURL,
		trace: 'on-first-retry',
		screenshot: 'only-on-failure'
	},
	projects: [
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'] }
		}
	]
});
