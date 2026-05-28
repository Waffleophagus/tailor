import type { Device, PolicyMapResponse } from '../api/schemas';
import { subjectDeviceIds } from './subjects';

export type PerspectiveKind = 'user' | 'group' | 'tag' | 'autogroup';

export interface PerspectiveOption {
	selector: string;
	kind: PerspectiveKind;
	label: string;
	deviceCount: number;
	description?: string;
}

const AUTOGROUPS: PerspectiveOption[] = [
	{
		selector: 'autogroup:member',
		kind: 'autogroup',
		label: 'autogroup:member',
		deviceCount: 0,
		description: 'Any user-owned device'
	},
	{
		selector: 'autogroup:tagged',
		kind: 'autogroup',
		label: 'autogroup:tagged',
		deviceCount: 0,
		description: 'Any tagged device'
	},
	{
		selector: 'autogroup:admin',
		kind: 'autogroup',
		label: 'autogroup:admin',
		deviceCount: 0,
		description: 'Tailscale admin devices (approx)'
	}
];

function groupMemberCount(policyMap: PolicyMapResponse | undefined, group: string): number {
	const members = policyMap?.sections
		.find((s) => s.name === 'groups')
		?.entries?.find((e) => e.label === group)?.value;
	if (!Array.isArray(members)) return 0;
	return members.filter((v) => typeof v === 'string').length;
}

export function buildPerspectiveCatalog(
	devices: Device[],
	policyMap?: PolicyMapResponse
): PerspectiveOption[] {
	const options: PerspectiveOption[] = [];

	const owners = [...new Set(devices.map((d) => d.owner).filter(Boolean))].sort();
	for (const owner of owners) {
		const count = devices.filter((d) => d.owner === owner).length;
		options.push({
			selector: owner,
			kind: 'user',
			label: owner,
			deviceCount: count
		});
	}

	const groupSection = policyMap?.sections.find((s) => s.name === 'groups');
	for (const entry of groupSection?.entries ?? []) {
		options.push({
			selector: entry.label,
			kind: 'group',
			label: entry.label,
			deviceCount: groupMemberCount(policyMap, entry.label),
			description: entry.summary
		});
	}

	const tagSet = new Set<string>();
	for (const device of devices) {
		for (const tag of device.tags) tagSet.add(tag);
	}
	const tagOwners = policyMap?.sections.find((s) => s.name === 'tagOwners');
	for (const entry of tagOwners?.entries ?? []) {
		tagSet.add(entry.label);
	}
	for (const tag of [...tagSet].sort()) {
		const count = devices.filter((d) => d.tags.includes(tag)).length;
		options.push({
			selector: tag,
			kind: 'tag',
			label: tag,
			deviceCount: count
		});
	}

	for (const autogroup of AUTOGROUPS) {
		const count = subjectDeviceIds(autogroup.selector, devices, policyMap).size;
		options.push({ ...autogroup, deviceCount: count });
	}

	return options;
}

export type PerspectiveValidation =
	| { status: 'empty' }
	| {
			status: 'valid';
			selector: string;
			kind: PerspectiveKind;
			deviceCount: number;
			warnings: string[];
	  }
	| { status: 'invalid'; message: string; warnings: string[] };

function classifySelector(
	selector: string,
	catalog: PerspectiveOption[]
): PerspectiveKind | undefined {
	const hit = catalog.find((o) => o.selector === selector);
	if (hit) return hit.kind;
	if (selector.startsWith('group:')) return 'group';
	if (selector.startsWith('tag:')) return 'tag';
	if (selector.startsWith('autogroup:')) return 'autogroup';
	if (selector.includes('@')) return 'user';
	return undefined;
}

function isIPOrHost(selector: string) {
	if (selector === '*' || selector.includes('/')) return true;
	if (/^\d+\.\d+\.\d+\.\d+/.test(selector)) return true;
	if (!selector.includes(':') && !selector.includes('@') && !selector.startsWith('tag:')) {
		return !selector.startsWith('group:') && !selector.startsWith('autogroup:');
	}
	return false;
}

/** Strip destination port suffix from pasted ACL selectors. */
export function parsePerspectiveInput(raw: string): { selector: string; warnings: string[] } {
	const trimmed = raw.trim();
	if (!trimmed) return { selector: '', warnings: [] };

	const warnings: string[] = [];
	if (trimmed === '*') {
		return { selector: trimmed, warnings: ['Wildcard is not a valid simulated subject.'] };
	}

	const lastColon = trimmed.lastIndexOf(':');
	if (lastColon > 3 && !trimmed.startsWith('autogroup:')) {
		const host = trimmed.slice(0, lastColon);
		const portPart = trimmed.slice(lastColon + 1);
		if (/^\d/.test(portPart) || portPart === '*' || portPart.includes(',')) {
			warnings.push(`Stripped port suffix — perspective uses \`${host}\`.`);
			return { selector: host, warnings };
		}
	}

	return { selector: trimmed, warnings };
}

export function validatePerspective(
	raw: string,
	devices: Device[],
	policyMap?: PolicyMapResponse
): PerspectiveValidation {
	const { selector, warnings } = parsePerspectiveInput(raw);
	if (!selector) return { status: 'empty' };

	if (selector === '*' || isIPOrHost(selector)) {
		return {
			status: 'invalid',
			message: 'IPs, hosts, and wildcards cannot be simulated as subjects.',
			warnings
		};
	}

	const catalog = buildPerspectiveCatalog(devices, policyMap);
	const kind = classifySelector(selector, catalog);
	if (!kind) {
		return {
			status: 'invalid',
			message: `Unknown selector \`${selector}\` — pick from the list or check policy.`,
			warnings
		};
	}

	const deviceCount = subjectDeviceIds(selector, devices, policyMap).size;
	const catalogHit = catalog.find((o) => o.selector === selector);
	if (kind === 'group' && !catalogHit) {
		return {
			status: 'invalid',
			message: `Unknown group \`${selector}\` — check policy or pick from the list.`,
			warnings
		};
	}

	return { status: 'valid', selector, kind, deviceCount, warnings };
}

export function kindLabel(kind: PerspectiveKind) {
	switch (kind) {
		case 'user':
			return 'User';
		case 'group':
			return 'Group';
		case 'tag':
			return 'Tag';
		case 'autogroup':
			return 'Autogroup';
	}
}

const RECENT_KEY = 'tailor:perspective-recent';

export function loadRecentPerspectives(): string[] {
	try {
		const raw = localStorage.getItem(RECENT_KEY);
		if (!raw) return [];
		const parsed = JSON.parse(raw);
		return Array.isArray(parsed) ? parsed.filter((v) => typeof v === 'string').slice(0, 5) : [];
	} catch {
		return [];
	}
}

export function saveRecentPerspective(selector: string) {
	const recent = loadRecentPerspectives().filter((s) => s !== selector);
	recent.unshift(selector);
	localStorage.setItem(RECENT_KEY, JSON.stringify(recent.slice(0, 5)));
}

export function filterCatalog(catalog: PerspectiveOption[], query: string): PerspectiveOption[] {
	const q = query.trim().toLowerCase();
	if (!q) return catalog;
	return catalog.filter((o) => {
		const haystack = `${o.selector} ${o.label} ${o.kind} ${o.description ?? ''}`.toLowerCase();
		return haystack.includes(q);
	});
}

export function groupedCatalog(catalog: PerspectiveOption[]) {
	const groups: Record<PerspectiveKind, PerspectiveOption[]> = {
		user: [],
		group: [],
		tag: [],
		autogroup: []
	};
	for (const option of catalog) {
		groups[option.kind].push(option);
	}
	return groups;
}
