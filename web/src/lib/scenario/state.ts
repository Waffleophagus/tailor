export type PolicyMode = 'current' | 'draft' | 'diff';
export type GraphMode = 'focused' | 'all';

export interface PolicyScenario {
	id: string;
	sourceSelector: string;
	policyMode: PolicyMode;
	graphMode: GraphMode;
	simulatedAt?: number;
	label?: string;
}

const STORAGE_KEY = 'tailor:policy-scenario';

export function createScenario(sourceSelector: string): PolicyScenario {
	return {
		id: crypto.randomUUID(),
		sourceSelector,
		policyMode: 'current',
		graphMode: 'focused',
		simulatedAt: Date.now()
	};
}

export function loadScenario(): PolicyScenario | null {
	try {
		const raw = sessionStorage.getItem(STORAGE_KEY);
		if (!raw) return null;
		const parsed = JSON.parse(raw) as PolicyScenario;
		if (!parsed?.sourceSelector) return null;
		return parsed;
	} catch {
		return null;
	}
}

export function saveScenario(scenario: PolicyScenario | null) {
	try {
		if (!scenario) {
			sessionStorage.removeItem(STORAGE_KEY);
			return;
		}
		sessionStorage.setItem(STORAGE_KEY, JSON.stringify(scenario));
	} catch {
		// ignore storage errors
	}
}

export function scenarioLabel(scenario: PolicyScenario, sourceCount: number): string {
	return `Viewing as ${scenario.sourceSelector} · ${sourceCount} source${sourceCount === 1 ? '' : 's'}`;
}

export function isScenarioActive(
	scenario: PolicyScenario | null,
	inputSelector: string,
	simulatedSelector: string
): boolean {
	const trimmed = inputSelector.trim();
	return Boolean(scenario && trimmed && trimmed === simulatedSelector);
}
