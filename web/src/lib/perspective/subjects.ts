import type { Device, PolicyMapResponse } from '../api/schemas';

function devicesForUser(user: string, devices: Device[]) {
	return devices.filter((d) => d.owner === user);
}

function devicesForUsers(users: string[], devices: Device[]) {
	const set = new Set(users);
	return devices.filter((d) => set.has(d.owner));
}

function devicesForTag(tag: string, devices: Device[]) {
	return devices.filter((d) => d.tags.includes(tag));
}

function devicesWithOwnerUntagged(devices: Device[]) {
	return devices.filter((d) => d.owner && d.tags.length === 0);
}

function devicesWithOwner(devices: Device[]) {
	return devices.filter((d) => d.owner);
}

function devicesWithTags(devices: Device[]) {
	return devices.filter((d) => d.tags.length > 0);
}

function unionDevices(a: Device[], b: Device[]) {
	const seen = new Set<string>();
	const out: Device[] = [];
	for (const device of [...a, ...b]) {
		if (seen.has(device.id)) continue;
		seen.add(device.id);
		out.push(device);
	}
	return out;
}

function groupMembers(policyMap: PolicyMapResponse | undefined, group: string): string[] {
	const section = policyMap?.sections.find((s) => s.name === 'groups');
	const entry = section?.entries?.find((e) => e.label === group);
	if (!entry?.value || !Array.isArray(entry.value)) return [];
	return entry.value.filter((v): v is string => typeof v === 'string');
}

/** Device IDs that act as sources for a policy subject (mirrors backend devicesForPerspective). */
export function subjectDeviceIds(
	selector: string,
	devices: Device[],
	policyMap?: PolicyMapResponse
): Set<string> {
	const trimmed = selector.trim();
	if (!trimmed) return new Set();

	let matched: Device[] = [];
	if (trimmed.startsWith('group:')) {
		matched = devicesForUsers(groupMembers(policyMap, trimmed), devices);
	} else if (trimmed.startsWith('tag:')) {
		matched = devicesForTag(trimmed, devices);
	} else if (trimmed === 'autogroup:member') {
		matched = devicesWithOwnerUntagged(devices);
	} else if (trimmed === 'autogroup:admin') {
		matched = devicesWithOwner(devices);
	} else if (trimmed === 'autogroup:tagged') {
		matched = devicesWithTags(devices);
	} else if (trimmed === 'cohort:member+tagged') {
		matched = unionDevices(devicesWithOwnerUntagged(devices), devicesWithTags(devices));
	} else if (trimmed.includes('@')) {
		matched = devicesForUser(trimmed, devices);
	}

	return new Set(matched.map((d) => d.id));
}
