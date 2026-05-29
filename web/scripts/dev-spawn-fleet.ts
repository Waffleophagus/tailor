import '../e2e/helpers/load-env.ts';

const apiBase = process.env.TAILOR_API_URL ?? 'http://127.0.0.1:8080';
const devKey = 'tskey-api-tailor-dev';

/** Matches topology websocket poll interval so each wave shows up as a graph tick. */
const WAVE_INTERVAL_MS = 2_500;

type SpawnSpec = {
	name: string;
	owner?: string;
	os?: string;
	tags?: string[];
	online?: boolean;
	subnetRouter?: boolean;
	routedSubnets?: string[];
};

type SpawnWave = {
	label: string;
	delayMs: number;
	specs: SpawnSpec[];
	/** After spawn, flip these names to online (provisioning → joined). */
	bringOnline?: string[];
};

const waves: SpawnWave[] = [
	{
		label: 'K8s prod node pool scale-out',
		delayMs: 0,
		specs: [
			{
				name: 'k8s-prod-worker-04',
				owner: 'platform-ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:k8s-prod'],
				online: true
			},
			{
				name: 'k8s-prod-worker-05',
				owner: 'platform-ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:k8s-prod'],
				online: true
			},
			{
				name: 'k8s-prod-observability-agent',
				owner: 'platform-ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:k8s-prod', 'tag:monitoring'],
				online: true
			}
		]
	},
	{
		label: 'K8s staging burst (CI preview)',
		delayMs: WAVE_INTERVAL_MS,
		specs: [
			{
				name: 'k8s-staging-worker-03',
				owner: 'platform-ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:k8s-staging'],
				online: true
			},
			{
				name: 'k8s-staging-preview-01',
				owner: 'ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:k8s-staging', 'tag:ci'],
				online: true
			},
			{
				name: 'k8s-staging-preview-02',
				owner: 'ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:k8s-staging', 'tag:ci'],
				online: true
			}
		]
	},
	{
		label: 'Data platform batch jobs',
		delayMs: WAVE_INTERVAL_MS,
		specs: [
			{
				name: 'etl-nightly-driver',
				owner: 'maya@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:db'],
				online: true
			},
			{
				name: 'warehouse-sync-agent',
				owner: 'maya@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:db', 'tag:platform'],
				online: true
			},
			{
				name: 'metrics-exporter-sidecar',
				owner: 'maya@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:monitoring'],
				online: true
			}
		]
	},
	{
		label: 'Compliance fleet (provisioned offline)',
		delayMs: WAVE_INTERVAL_MS,
		specs: [
			{
				name: 'compliance-archive-primary',
				owner: 'platform-ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:prod', 'tag:platform'],
				online: false
			},
			{
				name: 'audit-trail-ingest',
				owner: 'platform-ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:prod', 'tag:platform'],
				online: false
			},
			{
				name: 'ledger-reconciliation-east',
				owner: 'platform-ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:prod'],
				online: false
			},
			{
				name: 'policy-enforcement-gateway',
				owner: 'platform-ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:prod', 'tag:web'],
				online: false
			}
		],
		bringOnline: [
			'compliance-archive-primary',
			'audit-trail-ingest',
			'ledger-reconciliation-east',
			'policy-enforcement-gateway'
		]
	},
	{
		label: 'Edge / security appliances',
		delayMs: WAVE_INTERVAL_MS,
		specs: [
			{
				name: 'secrets-rotation-controller',
				owner: 'platform-ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:platform'],
				online: true
			},
			{
				name: 'certificate-transparency-mirror',
				owner: 'platform-ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:platform', 'tag:monitoring'],
				online: true
			},
			{
				name: 'incident-response-bastion',
				owner: 'ops@demo.tailor.ts.net',
				os: 'linux',
				tags: ['tag:prod'],
				online: true
			}
		]
	}
];

async function readJSON<T>(response: Response): Promise<T> {
	const text = await response.text();
	try {
		return JSON.parse(text) as T;
	} catch {
		throw new Error(`${response.url} returned non-JSON (${response.status}): ${text}`);
	}
}

function sleep(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

async function ensureDevAuth(): Promise<void> {
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
	if (status.authenticated && status.devMode) {
		return;
	}

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

async function spawnSpecs(specs: SpawnSpec[]): Promise<{ name: string }[]> {
	const response = await fetch(`${apiBase}/api/dev/spawn-devices`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ specs })
	});
	if (!response.ok) {
		const body = await response.text();
		throw new Error(`Spawn failed (${response.status}): ${body}`);
	}
	const body = await readJSON<{ spawned?: { name: string }[] }>(response);
	return body.spawned ?? [];
}

async function bringDevicesOnline(names: string[]): Promise<void> {
	const response = await fetch(`${apiBase}/api/dev/patch-devices`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({
			devices: names.map((name) => ({ name, online: true }))
		})
	});
	if (!response.ok) {
		const body = await response.text();
		throw new Error(`Patch failed (${response.status}): ${body}`);
	}
}

async function main() {
	await ensureDevAuth();

	let totalSpawned = 0;
	const started = Date.now();

	for (const wave of waves) {
		if (wave.delayMs > 0) {
			console.log(`Waiting ${wave.delayMs}ms — ${wave.label}…`);
			await sleep(wave.delayMs);
		}

		const spawned = await spawnSpecs(wave.specs);
		totalSpawned += spawned.length;
		console.log(`\n▸ ${wave.label} (+${spawned.length})`);
		for (const device of spawned) {
			const spec = wave.specs.find((s) => s.name === device.name);
			const state = spec?.online === false ? 'offline' : 'online';
			console.log(`  • ${device.name} (${state})`);
		}

		if (wave.bringOnline?.length) {
			await sleep(WAVE_INTERVAL_MS);
			await bringDevicesOnline(wave.bringOnline);
			console.log(`  ↳ came online: ${wave.bringOnline.join(', ')}`);
		}
	}

	const fleet = await readJSON<{ devices?: unknown[] }>(await fetch(`${apiBase}/api/topology`));
	const elapsed = ((Date.now() - started) / 1000).toFixed(1);
	console.log(
		`\nDone — spawned ${totalSpawned} devices in ${elapsed}s. Fleet total: ${fleet.devices?.length ?? '?'} (graph updates every ~2s).`
	);
	console.log('Tip: reload with superadmin perspective to see the full ACL mesh.');
}

main().catch((error) => {
	console.error(error instanceof Error ? error.message : error);
	process.exit(1);
});
