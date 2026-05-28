export interface DraftChange {
	id: string;
	label: string;
	route?: string;
	at: number;
}

export interface PolicyMutation {
	type: string;
	section?: string;
	key?: string;
	index?: number;
	rule?: {
		action?: string;
		src?: string[];
		dst?: string[];
		proto?: string;
	};
	grant?: {
		src?: string[];
		dst?: string[];
		ip?: string[];
		app?: Record<string, unknown>;
	};
	host?: string;
	ipSet?: string[];
	members?: string[];
	owners?: string[];
	value?: unknown;
}

export function createChange(label: string, route?: string): DraftChange {
	return {
		id: crypto.randomUUID(),
		label,
		route,
		at: Date.now()
	};
}

export function splitSelectors(value: string): string[] {
	return value
		.split(/[,\n]/)
		.map((part) => part.trim())
		.filter(Boolean);
}

export function portPresetValues(preset: string, custom: string): string[] {
	if (preset === 'custom') return splitSelectors(custom);
	if (preset === '*') return ['*'];
	return splitSelectors(preset);
}

export function buildACLRule(input: {
	sources: string[];
	destinations: string[];
	ports: string[];
	protocol?: string;
}) {
	const dst = input.destinations.map((destination) => {
		if (destination.includes(':') && !destination.startsWith('tag:')) return destination;
		const portSet = input.ports.join(',');
		return `${destination}:${portSet}`;
	});
	return {
		action: 'accept',
		src: input.sources,
		dst,
		proto: input.protocol === 'tcp' ? '' : input.protocol
	};
}

export function buildGrantRule(input: {
	sources: string[];
	destinations: string[];
	ports: string[];
}) {
	return {
		src: input.sources,
		dst: input.destinations,
		ip: input.ports.map((port) => (port === '*' ? '*:*' : `tcp:${port}`))
	};
}

/** Simple line diff for HuJSON review in the draft tray. */
export function diffLines(before: string, after: string): string[] {
	const left = before.split('\n');
	const right = after.split('\n');
	const max = Math.max(left.length, right.length);
	const lines: string[] = [];
	for (let i = 0; i < max; i += 1) {
		const a = left[i];
		const b = right[i];
		if (a === b) {
			if (a !== undefined) lines.push(`  ${a}`);
		} else {
			if (a !== undefined) lines.push(`- ${a}`);
			if (b !== undefined) lines.push(`+ ${b}`);
		}
	}
	return lines;
}
