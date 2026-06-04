import { describe, expect, it } from 'vitest';

import { edgeClasses } from './edge-classes';
import { resolveEdgeStyle, styleForEdge } from './edge-style';
import { EDGE_CLASS_CASES, EDGE_STYLE_CASES } from './style-cases';
import type { EdgeStylePatch } from './style-catalog';
import type { RenderEdge } from './engine';

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

	it('allowed ACL draft and interaction states preserve legend color and line style', () => {
		const legendStyles: Array<{
			name: string;
			accessScope?: RenderEdge['accessScope'];
			lineColor: string;
			lineStyle: ReturnType<typeof resolveEdgeStyle>['lineStyle'];
		}> = [
			{ name: 'generic ACL', lineColor: '#438aa1', lineStyle: 'solid' },
			{ name: 'SSH', accessScope: 'ssh', lineColor: '#2f9f68', lineStyle: 'solid' },
			{ name: 'HTTP/S', accessScope: 'http', lineColor: '#438aa1', lineStyle: 'solid' },
			{ name: 'broad', accessScope: 'broad', lineColor: '#b0892f', lineStyle: 'solid' },
			{ name: 'custom', accessScope: 'custom', lineColor: '#7c6fb0', lineStyle: 'dashed' },
			{ name: 'limited', accessScope: 'limited', lineColor: '#7c6fb0', lineStyle: 'dashed' }
		];
		const allowedStates: Array<RenderEdge['state']> = [undefined, 'added', 'changed', 'unchanged'];

		for (const legend of legendStyles) {
			for (const state of allowedStates) {
				for (const selected of [false, true]) {
					for (const focused of [false, true]) {
						const edge: RenderEdge = {
							id: 'edge-1',
							from: 'a',
							to: 'b',
							kind: 'acl',
							accessScope: legend.accessScope,
							state
						};
						const classes = edgeClasses(edge, {
							selectedEdgeId: selected ? edge.id : undefined
						})
							.split(/\s+/)
							.filter(Boolean);
						if (focused) classes.push('focused');

						const style = resolveEdgeStyle(classes);
						expect(
							{ lineColor: style.lineColor, lineStyle: style.lineStyle },
							`${legend.name} state=${state ?? 'none'} selected=${selected} focused=${focused}`
						).toEqual({
							lineColor: legend.lineColor,
							lineStyle: legend.lineStyle
						});
					}
				}
			}
		}
	});
});
