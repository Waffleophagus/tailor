import type { Device } from '../api/schemas';
import type { RenderEdge } from './engine';

export const AGGREGATE_ID_PREFIX = '__aggregate:';

/** Per-tag collapse rules. Extend this list to collapse other tagged fleets. */
export interface TagCollapseRule {
	tag: string;
	/** Collapse only when at least this many visible devices share the tag. */
	minCount: number;
	/** Graph and sidebar label; defaults to a title derived from the tag. */
	label?: string;
}

export const DEFAULT_TAG_COLLAPSE_RULES: TagCollapseRule[] = [
	{ tag: 'tag:mullvad-exit-node', minCount: 5, label: 'Mullvad exit nodes' }
];

export interface DeviceAggregateMeta {
	aggregateId: string;
	tag: string;
	label: string;
	members: Device[];
}

export interface CollapseDevicesOptions {
	enabled: boolean;
	rules?: TagCollapseRule[];
}

export interface CollapseDevicesResult {
	graphDevices: Device[];
	listDevices: Device[];
	aggregateMeta: Map<string, DeviceAggregateMeta>;
	/** Maps each real device id to the graph node id (self or aggregate). */
	graphIdForDevice: Map<string, string>;
}

export function isAggregateDeviceId(id: string): boolean {
	return id.startsWith(AGGREGATE_ID_PREFIX);
}

export function aggregateIdForTag(tag: string): string {
	return `${AGGREGATE_ID_PREFIX}${tag}`;
}

export function tagFromAggregateId(id: string): string | undefined {
	if (!isAggregateDeviceId(id)) return undefined;
	return id.slice(AGGREGATE_ID_PREFIX.length);
}

function defaultLabelForTag(tag: string): string {
	const base = tag.startsWith('tag:') ? tag.slice(4) : tag;
	return base
		.split(/[-_]+/)
		.filter(Boolean)
		.map((part) => part.charAt(0).toUpperCase() + part.slice(1))
		.join(' ');
}

function buildAggregateDevice(tag: string, label: string, members: Device[]): Device {
	const onlineCount = members.filter((member) => member.online).length;
	return {
		id: aggregateIdForTag(tag),
		name: `${label} (${members.length})`,
		ip: '',
		tailscaleIps: [],
		os: 'aggregate',
		online: onlineCount > 0,
		owner: '',
		tags: [tag],
		subnetRouter: false,
		routedSubnets: [],
		lastSeen: onlineCount < members.length ? `${onlineCount}/${members.length} online` : undefined
	};
}

export function collapseDevicesByTag(
	devices: Device[],
	options: CollapseDevicesOptions
): CollapseDevicesResult {
	const rules = options.rules ?? DEFAULT_TAG_COLLAPSE_RULES;
	const empty: CollapseDevicesResult = {
		graphDevices: devices,
		listDevices: devices,
		aggregateMeta: new Map(),
		graphIdForDevice: new Map(devices.map((device) => [device.id, device.id]))
	};

	if (!options.enabled || devices.length === 0 || rules.length === 0) {
		return empty;
	}

	const collapsedMemberIds = new Set<string>();
	const aggregates: Device[] = [];
	const aggregateMeta = new Map<string, DeviceAggregateMeta>();
	const graphIdForDevice = new Map<string, string>();

	for (const rule of rules) {
		const members = devices.filter(
			(device) => device.tags.includes(rule.tag) && !collapsedMemberIds.has(device.id)
		);
		if (members.length < rule.minCount) {
			continue;
		}

		const label = rule.label ?? defaultLabelForTag(rule.tag);
		const aggregate = buildAggregateDevice(rule.tag, label, members);
		aggregates.push(aggregate);
		aggregateMeta.set(aggregate.id, {
			aggregateId: aggregate.id,
			tag: rule.tag,
			label,
			members
		});
		for (const member of members) {
			collapsedMemberIds.add(member.id);
			graphIdForDevice.set(member.id, aggregate.id);
		}
	}

	if (aggregates.length === 0) {
		return empty;
	}

	const graphDevices: Device[] = [];
	for (const device of devices) {
		if (collapsedMemberIds.has(device.id)) continue;
		graphDevices.push(device);
		if (!graphIdForDevice.has(device.id)) {
			graphIdForDevice.set(device.id, device.id);
		}
	}
	graphDevices.push(...aggregates);
	graphDevices.sort((a, b) => a.name.localeCompare(b.name));

	const listDevices = [...graphDevices];

	return {
		graphDevices,
		listDevices,
		aggregateMeta,
		graphIdForDevice
	};
}

function edgeDedupeKey(edge: RenderEdge): string {
	return [
		edge.from,
		edge.to,
		edge.kind,
		edge.accessScope ?? '',
		edge.ports?.join(',') ?? '',
		edge.protocols?.join(',') ?? ''
	].join('|');
}

export function rewriteEdgesForCollapsedDevices(
	edges: RenderEdge[],
	graphIdForDevice: ReadonlyMap<string, string>
): RenderEdge[] {
	if (graphIdForDevice.size === 0) return edges;

	const seen = new Set<string>();
	const rewritten: RenderEdge[] = [];

	for (const edge of edges) {
		const from = graphIdForDevice.get(edge.from) ?? edge.from;
		const to = graphIdForDevice.get(edge.to) ?? edge.to;
		if (from === to) continue;

		const next: RenderEdge = { ...edge, from, to };
		const key = edgeDedupeKey(next);
		if (seen.has(key)) continue;
		seen.add(key);
		rewritten.push({
			...next,
			id: `collapsed:${key}`
		});
	}

	return rewritten;
}
