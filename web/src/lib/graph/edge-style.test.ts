import { describe, expect, it } from 'vitest';

import { edgeClasses } from './edge-classes';
import { resolveEdgeStyle, styleForEdge } from './edge-style';
import { EDGE_CLASS_CASES, EDGE_STYLE_CASES } from './style-cases';
import type { EdgeStylePatch } from './style-catalog';

function expectStyle(actual: ReturnType<typeof resolveEdgeStyle>, expected: EdgeStylePatch) {
	for (const [key, value] of Object.entries(expected)) {
		expect(actual[key as keyof typeof actual]).toBe(value);
	}
}

describe('edgeClasses', () => {
	for (const testCase of EDGE_CLASS_CASES) {
		it(testCase.name, () => {
			expect(edgeClasses(testCase.edge, testCase.options).split(/\s+/).filter(Boolean)).toEqual(
				testCase.expected
			);
		});
	}
});

describe('resolveEdgeStyle catalog', () => {
	for (const testCase of EDGE_STYLE_CASES) {
		it(testCase.name, () => {
			const classes = [
				...edgeClasses(testCase.edge, testCase.options).split(/\s+/).filter(Boolean),
				...(testCase.extraClasses ?? [])
			];
			expectStyle(resolveEdgeStyle(classes), testCase.expected);
		});
	}

	it('styleForEdge matches resolveEdgeStyle for catalog entries', () => {
		for (const testCase of EDGE_STYLE_CASES) {
			if (testCase.extraClasses?.length) continue;
			expectStyle(styleForEdge(testCase.edge, testCase.options), testCase.expected);
		}
	});

	it('diff state overrides ACL scope color', () => {
		const style = styleForEdge({
			id: '1',
			from: 'a',
			to: 'b',
			kind: 'acl',
			accessScope: 'http',
			state: 'added'
		});
		expect(style.lineColor).toBe('#2f9f68');
		expect(style.lineStyle).toBe('dashed');
		expect(style.width).toBe(3.3);
	});
});
