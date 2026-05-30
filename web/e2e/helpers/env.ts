export const baseURL = process.env.TAILOR_E2E_BASE_URL ?? 'http://127.0.0.1:5173';

/** Built-in demo key — in-memory tailnet only, no Cloud API writes. */
export const demoApiKey = 'tskey-api-tailor-dev';

/** Tailscale API key from web/.env (see load-env). */
export const tailscaleApiKey = process.env.TAILSCALE_API_KEY?.trim() ?? '';

/** Optional tailnet override from web/.env. */
export const configuredTailnet = process.env.TAILOR_TAILNET?.trim() ?? '';

/** True when web/.env uses the built-in demo API key. */
export function isDemoApiKey(): boolean {
	return tailscaleApiKey === demoApiKey;
}

/** True when web/.env has a real Cloud API key (not the demo key). */
export function isProductionApiKey(): boolean {
	return tailscaleApiKey.length > 0 && !isDemoApiKey();
}

/** Primary simulation subject. Override with TAILOR_E2E_PERSPECTIVE. */
export const defaultPerspective = process.env.TAILOR_E2E_PERSPECTIVE ?? 'alice@demo.tailor.ts.net';

/** Secondary user for cross-perspective checks. */
export const alternatePerspective =
	process.env.TAILOR_E2E_ALT_PERSPECTIVE ?? 'bob@demo.tailor.ts.net';

/** Broad *:* ACL subject for graph connectivity debugging. */
export const superUserPerspective = process.env.TAILOR_E2E_SUPER_USER ?? 'group:superuser';

/** Device name for the super-user owner in the demo fleet. */
export const superUserDevice = process.env.TAILOR_E2E_SUPER_DEVICE ?? 'superadmin-console';

export const testDestination = process.env.TAILOR_E2E_DESTINATION ?? 'tag:web';
