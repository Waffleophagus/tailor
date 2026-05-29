import path from 'node:path';
import { fileURLToPath } from 'node:url';

import type { PlaywrightTestConfig } from '@playwright/test';

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../..');

export const tailorPort = process.env.TAILOR_E2E_TAILOR_PORT ?? '8080';
export const tailorHealthURL =
	process.env.TAILOR_E2E_TAILOR_URL ?? `http://127.0.0.1:${tailorPort}/api/health`;

type WebServerConfig = NonNullable<PlaywrightTestConfig['webServer']>;

function asWebServers(config: WebServerConfig): WebServerConfig {
	return config;
}

/** Build and run the Go backend for E2E when nothing is listening yet. */
export function tailorWebServer(reuseExistingServer: boolean): WebServerConfig {
	return {
		command: 'pnpm backend:e2e',
		cwd: webRoot,
		url: tailorHealthURL,
		reuseExistingServer,
		timeout: 120_000,
		env: {
			TAILOR_ADDR: `:${tailorPort}`
		}
	};
}

/** Vite dev server proxying /api to the tailor backend. */
export function viteWebServer(baseURL: string, reuseExistingServer: boolean): WebServerConfig {
	const url = new URL(baseURL);
	const port = url.port || (url.protocol === 'https:' ? '443' : '80');
	const host = url.hostname || '127.0.0.1';

	return {
		command: `pnpm exec vite --host ${host} --port ${port} --strictPort`,
		cwd: webRoot,
		url: baseURL,
		reuseExistingServer,
		timeout: 120_000
	};
}

export function e2eWebServers(baseURL: string, reuseExistingServer: boolean): WebServerConfig {
	return asWebServers([
		tailorWebServer(reuseExistingServer),
		viteWebServer(baseURL, reuseExistingServer)
	]);
}
