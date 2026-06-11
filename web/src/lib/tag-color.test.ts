import { describe, expect, it } from 'vitest';
import {
	buildTagColorMap,
	getTagColor,
	oklchHexDistance,
	UNTAGGED_DEVICE_COLOR
} from './tag-color';

describe('buildTagColorMap', () => {
	it('returns stable colors for the same tag set', () => {
		const tags = ['tag:alpha', 'tag:beta', 'tag:gamma'];
		const first = buildTagColorMap(tags);
		const second = buildTagColorMap(tags);
		for (const tag of tags) {
			expect(first.get(tag)).toBe(second.get(tag));
		}
	});

	it('is independent of input order', () => {
		const ordered = buildTagColorMap(['tag:b', 'tag:a', 'tag:c']);
		const shuffled = buildTagColorMap(['tag:c', 'tag:a', 'tag:b']);
		expect(ordered.get('tag:a')).toBe(shuffled.get('tag:a'));
		expect(ordered.get('tag:b')).toBe(shuffled.get('tag:b'));
		expect(ordered.get('tag:c')).toBe(shuffled.get('tag:c'));
	});

	it('assigns distinct colors for 200 synthetic tags vs prior assignments', () => {
		const tags = Array.from({ length: 200 }, (_, i) => `tag:test-${i}`);
		const map = buildTagColorMap(tags);
		const colors = [...map.values()];
		expect(new Set(colors).size).toBe(200);

		const sorted = [...tags].sort((a, b) => a.localeCompare(b));
		for (let i = 1; i < sorted.length; i += 1) {
			const current = map.get(sorted[i])!;
			for (let j = 0; j < i; j += 1) {
				const prior = map.get(sorted[j])!;
				expect(oklchHexDistance(current, prior)).toBeGreaterThanOrEqual(0.025);
			}
		}
	});

	it('keeps beszel and tsdproxy-racknerd distinct from each other and untagged', () => {
		const map = buildTagColorMap(['tag:beszel', 'tag:tsdproxy-racknerd']);
		const beszel = map.get('tag:beszel')!;
		const racknerd = map.get('tag:tsdproxy-racknerd')!;
		expect(beszel).not.toBe(racknerd);
		expect(beszel).not.toBe(UNTAGGED_DEVICE_COLOR);
		expect(racknerd).not.toBe(UNTAGGED_DEVICE_COLOR);
	});
});

describe('getTagColor', () => {
	it('returns the fixed untagged color when no tag is present', () => {
		const map = buildTagColorMap(['tag:beszel']);
		expect(getTagColor(undefined, map)).toBe(UNTAGGED_DEVICE_COLOR);
	});
});
