import type { PolicyMapResponse, PolicySection } from '../api/schemas';

export type WorkbenchRoute =
	| 'general-access'
	| 'ssh'
	| 'tests'
	| 'auto-approvers'
	| 'groups'
	| 'tags'
	| 'ip-sets'
	| 'hosts'
	| 'device-posture'
	| 'node-attributes'
	| 'advanced';

export type SimulationTier = 'graph-simulated' | 'graph-partial' | 'edit-validate' | 'non-graph';

export interface WorkbenchNavItem {
	id: WorkbenchRoute;
	label: string;
	group: 'policy' | 'definitions';
	sectionNames: string[];
	simulationTier: SimulationTier;
	searchPlaceholder: string;
}

const KNOWN_SECTIONS = new Set([
	'acls',
	'grants',
	'ssh',
	'tests',
	'autoApprovers',
	'groups',
	'tagOwners',
	'ipsets',
	'hosts',
	'postures',
	'nodeAttrs'
]);

export const WORKBENCH_NAV: WorkbenchNavItem[] = [
	{
		id: 'general-access',
		label: 'General access rules',
		group: 'policy',
		sectionNames: ['acls', 'grants'],
		simulationTier: 'graph-simulated',
		searchPlaceholder: 'Search by user, group, device, tag, port, or IP address'
	},
	{
		id: 'ssh',
		label: 'Tailscale SSH',
		group: 'policy',
		sectionNames: ['ssh'],
		simulationTier: 'graph-partial',
		searchPlaceholder: 'Search SSH rules'
	},
	{
		id: 'tests',
		label: 'Tests',
		group: 'policy',
		sectionNames: ['tests'],
		simulationTier: 'non-graph',
		searchPlaceholder: 'Search tests'
	},
	{
		id: 'auto-approvers',
		label: 'Auto-approvers',
		group: 'policy',
		sectionNames: ['autoApprovers'],
		simulationTier: 'non-graph',
		searchPlaceholder: 'Search auto-approvers'
	},
	{
		id: 'groups',
		label: 'Groups',
		group: 'definitions',
		sectionNames: ['groups'],
		simulationTier: 'graph-simulated',
		searchPlaceholder: 'Search groups'
	},
	{
		id: 'tags',
		label: 'Tags',
		group: 'definitions',
		sectionNames: ['tagOwners'],
		simulationTier: 'graph-simulated',
		searchPlaceholder: 'Search tags'
	},
	{
		id: 'ip-sets',
		label: 'IP sets',
		group: 'definitions',
		sectionNames: ['ipsets'],
		simulationTier: 'graph-simulated',
		searchPlaceholder: 'Search IP sets'
	},
	{
		id: 'hosts',
		label: 'Hosts',
		group: 'definitions',
		sectionNames: ['hosts'],
		simulationTier: 'graph-simulated',
		searchPlaceholder: 'Search hosts'
	},
	{
		id: 'device-posture',
		label: 'Device posture',
		group: 'definitions',
		sectionNames: ['postures'],
		simulationTier: 'edit-validate',
		searchPlaceholder: 'Search device posture'
	},
	{
		id: 'node-attributes',
		label: 'Node attributes',
		group: 'definitions',
		sectionNames: ['nodeAttrs'],
		simulationTier: 'edit-validate',
		searchPlaceholder: 'Search node attributes'
	}
];

export function defaultWorkbenchRoute(): WorkbenchRoute {
	return 'general-access';
}

export function workbenchNavItem(route: WorkbenchRoute): WorkbenchNavItem | undefined {
	return WORKBENCH_NAV.find((item) => item.id === route);
}

export function sectionsForRoute(
	policyMap: PolicyMapResponse | undefined,
	route: WorkbenchRoute
): PolicySection[] {
	if (!policyMap?.sections) return [];
	if (route === 'advanced') {
		return policyMap.sections.filter((section) => !KNOWN_SECTIONS.has(section.name));
	}
	const item = workbenchNavItem(route);
	if (!item) return [];
	const names = new Set(item.sectionNames);
	return policyMap.sections.filter((section) => names.has(section.name));
}

export function entryCountForRoute(
	policyMap: PolicyMapResponse | undefined,
	route: WorkbenchRoute
): number {
	return sectionsForRoute(policyMap, route).reduce((sum, section) => sum + section.count, 0);
}

export function hasAdvancedSections(policyMap: PolicyMapResponse | undefined): boolean {
	return sectionsForRoute(policyMap, 'advanced').length > 0;
}

export function simulationTierLabel(tier: SimulationTier): string {
	switch (tier) {
		case 'graph-simulated':
			return 'Graph preview';
		case 'graph-partial':
			return 'Partial graph preview';
		case 'edit-validate':
			return 'Edit & validate';
		case 'non-graph':
			return 'Not on graph';
	}
}

export function filterSectionsByQuery(sections: PolicySection[], query: string): PolicySection[] {
	const q = query.trim().toLowerCase();
	if (!q) return sections;
	return sections
		.map((section) => {
			const sectionHaystack = [section.name, section.description ?? ''].join(' ').toLowerCase();
			if (sectionHaystack.includes(q)) return section;
			const entries = (section.entries ?? []).filter((entry) => {
				const haystack = [entry.label, entry.summary ?? '', ...(entry.selectors ?? [])]
					.join(' ')
					.toLowerCase();
				return haystack.includes(q);
			});
			if (entries.length === 0) return null;
			return { ...section, entries };
		})
		.filter((section): section is PolicySection => section !== null);
}

export function canSimulateEntry(sectionName: string, label: string) {
	return (
		(sectionName === 'groups' && label.startsWith('group:')) ||
		(sectionName === 'tagOwners' && label.startsWith('tag:'))
	);
}
