import { converter, differenceEuclidean, formatHex, parse } from 'culori';

const oklchConverter = converter('oklch');
const oklchDistance = differenceEuclidean('oklch');

/** Fixed color for devices with no tag — reserved and excluded from tag hash collisions. */
export const UNTAGGED_DEVICE_COLOR = '#788896';

const GOLDEN_ANGLE = 137.508;
const HUE_CYCLE_LEN = Math.ceil(360 / GOLDEN_ANGLE);
const MIN_OKLCH_DISTANCE = 0.025;
const MAX_ATTEMPTS = 500;

const L_MIN = 0.48;
const L_RANGE = 0.1;
const C_MIN = 0.1;
const C_RANGE = 0.06;

interface OklchColor {
	l: number;
	c: number;
	h: number;
}

interface AssignedColor {
	oklch: OklchColor;
	hex: string;
}

function fnv1a(value: string): number {
	let hash = 0x811c9dc5;
	for (let i = 0; i < value.length; i += 1) {
		hash ^= value.charCodeAt(i);
		hash = Math.imul(hash, 0x01000193);
	}
	return hash >>> 0;
}

function clamp(value: number, min: number, max: number): number {
	return Math.min(max, Math.max(min, value));
}

function hashToOklchAnchor(tag: string): OklchColor {
	const h = fnv1a(tag);
	const h2 = fnv1a(`${tag}\u0000`);
	const h3 = fnv1a(`${tag}\u0001`);
	return {
		l: L_MIN + ((h2 % 1000) / 1000) * L_RANGE,
		c: C_MIN + ((h3 % 1000) / 1000) * C_RANGE,
		h: h % 360
	};
}

function toOklch(color: OklchColor) {
	return oklchConverter({ mode: 'oklch', l: color.l, c: color.c, h: color.h });
}

function oklchToHex(color: OklchColor): string {
	return formatHex(toOklch(color)) ?? UNTAGGED_DEVICE_COLOR;
}

function parsedOklch(hex: string) {
	return oklchConverter(parse(hex));
}

function isDistinct(_candidate: OklchColor, hex: string, assigned: AssignedColor[]): boolean {
	if (assigned.some((prior) => prior.hex === hex)) return false;
	const cand = parsedOklch(hex);
	if (!cand) return false;
	for (const prior of assigned) {
		const priorColor = parsedOklch(prior.hex);
		if (!priorColor) continue;
		if (oklchDistance(cand, priorColor) < MIN_OKLCH_DISTANCE) {
			return false;
		}
	}
	return true;
}

function spiralCandidate(anchor: OklchColor, attempt: number): OklchColor {
	if (attempt === 0) return anchor;

	const hueCycle = Math.floor(attempt / HUE_CYCLE_LEN);
	const chromaRing = Math.floor(hueCycle / 3);
	const lightRing = Math.floor(hueCycle / 9);
	const chromaSign = chromaRing % 2 === 0 ? 1 : -1;
	const lightSign = lightRing % 2 === 0 ? 1 : -1;

	return {
		h: (anchor.h + attempt * GOLDEN_ANGLE) % 360,
		c: clamp(anchor.c + chromaRing * 0.02 * chromaSign, 0.06, 0.2),
		l: clamp(anchor.l + lightRing * 0.04 * lightSign, 0.4, 0.65)
	};
}

function reservedAssignedColors(): AssignedColor[] {
	const parsed = oklchConverter(parse(UNTAGGED_DEVICE_COLOR));
	if (!parsed || parsed.mode !== 'oklch') return [];
	const oklch: OklchColor = { l: parsed.l, c: parsed.c, h: parsed.h ?? 0 };
	return [{ oklch, hex: UNTAGGED_DEVICE_COLOR }];
}

function* searchCandidates(anchor: OklchColor): Generator<OklchColor> {
	for (let attempt = 0; attempt < MAX_ATTEMPTS; attempt += 1) {
		yield spiralCandidate(anchor, attempt);
	}

	for (let step = 0; step < 720; step += 1) {
		yield {
			h: (anchor.h + step * 0.5) % 360,
			c: clamp(C_MIN + (step % 6) * 0.02, 0.06, 0.2),
			l: clamp(L_MIN + (Math.floor(step / 6) % 5) * 0.025, 0.4, 0.65)
		};
	}
}

function resolveTagColor(tag: string, assigned: AssignedColor[]): string {
	const anchor = hashToOklchAnchor(tag);

	for (const candidate of searchCandidates(anchor)) {
		const hex = oklchToHex(candidate);
		if (isDistinct(candidate, hex, assigned)) {
			assigned.push({ oklch: candidate, hex });
			return hex;
		}
	}

	const fallback = {
		h: (anchor.h + assigned.length * GOLDEN_ANGLE) % 360,
		c: anchor.c,
		l: anchor.l
	};
	const hex = oklchToHex(fallback);
	assigned.push({ oklch: fallback, hex });
	return hex;
}

/** Build a stable tag→hex map with perceptual collision resolution. */
export function buildTagColorMap(tags: readonly string[]): ReadonlyMap<string, string> {
	const sorted = [...new Set(tags)].sort((a, b) => a.localeCompare(b));
	const assigned = reservedAssignedColors();
	const map = new Map<string, string>();

	for (const tag of sorted) {
		map.set(tag, resolveTagColor(tag, assigned));
	}

	return map;
}

export function getTagColor(
	firstTag: string | undefined,
	tagColorMap: ReadonlyMap<string, string>
): string {
	if (!firstTag) return UNTAGGED_DEVICE_COLOR;
	return tagColorMap.get(firstTag) ?? UNTAGGED_DEVICE_COLOR;
}

/** Minimum OKLCH distance between two hex colors (for tests). */
export function oklchHexDistance(a: string, b: string): number {
	const left = oklchConverter(parse(a));
	const right = oklchConverter(parse(b));
	if (!left || !right) return 0;
	return oklchDistance(left, right);
}
