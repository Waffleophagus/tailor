import '../e2e/helpers/load-env.ts';

const apiBase = process.env.TAILOR_API_URL ?? 'http://127.0.0.1:8080';
const devKey = 'tskey-api-tailor-dev';

const fleetNames = [
	'compliance-archive-primary',
	'audit-trail-ingest',
	'ledger-reconciliation-east',
	'secrets-rotation-controller',
	'policy-enforcement-gateway',
	'identity-federation-hub',
	'observability-collector-core',
	'certificate-transparency-mirror',
	'incident-response-bastion',
	'regulatory-reporting-batch'
];

async function readJSON<T>(response: Response): Promise<T> {
	const text = await response.text();
	try {
		return JSON.parse(text) as T;
	} catch {
		throw new Error(`${response.url} returned non-JSON (${response.status}): ${text}`);
	}
}

async function main() {
	const health = await fetch(`${apiBase}/api/health`);
	if (!health.ok) {
		throw new Error(
			`Tailor backend is not reachable at ${apiBase}. Start it with: pnpm backend:run:dev`
		);
	}

	const meta = await readJSON<{ build?: string }>(health);
	if (meta.build !== 'dev') {
		throw new Error(
			'dev:spawn requires a dev build (pnpm backend:build:dev). Production builds exclude the spawn API.'
		);
	}

	const status = await readJSON<{ authenticated?: boolean; devMode?: boolean }>(
		await fetch(`${apiBase}/api/cloud/status`)
	);
	if (!status.authenticated || !status.devMode) {
		const auth = await fetch(`${apiBase}/api/cloud/auth`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ tailnet: '-', apiKey: devKey })
		});
		if (!auth.ok) {
			const body = await auth.text();
			throw new Error(`ACL auth failed (${auth.status}): ${body}`);
		}
	}

	const spawn = await fetch(`${apiBase}/api/dev/spawn-devices`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			names: fleetNames,
			owner: 'platform-ops@demo.tailor.ts.net',
			os: 'linux',
			tags: ['tag:prod']
		})
	});
	if (!spawn.ok) {
		const body = await spawn.text();
		throw new Error(`Spawn failed (${spawn.status}): ${body}`);
	}

	const body = await readJSON<{ spawned?: { name: string }[]; devices?: unknown[] }>(spawn);
	const spawned = body.spawned ?? [];
	console.log(`Spawned ${spawned.length} devices on demo.tailor.ts.net:`);
	for (const device of spawned) {
		console.log(`  • ${device.name}`);
	}
	console.log(
		`Fleet total: ${body.devices?.length ?? '?'} devices (watch the graph update in ~2s).`
	);
}

main().catch((error) => {
	console.error(error instanceof Error ? error.message : error);
	process.exit(1);
});
