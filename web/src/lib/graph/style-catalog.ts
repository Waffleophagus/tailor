import type { StylesheetJson } from 'cytoscape';

export type LineStyle = 'solid' | 'dashed' | 'dotted';

export type CurveStyle = 'bezier' | 'unbundled-bezier' | 'straight';

export interface ResolvedEdgeStyle {
	lineColor: string;
	targetArrowColor: string;
	lineStyle: LineStyle;
	curveStyle: CurveStyle;
	width: number;
	opacity: number;
	targetArrowShape?: string;
}

export type EdgeStylePatch = Partial<ResolvedEdgeStyle>;

export interface EdgeStyleRule {
	selector: string;
	style: EdgeStylePatch;
}

/** Edge stylesheet rules in Cytoscape application order (later rules override). */
export const EDGE_STYLE_RULES: EdgeStyleRule[] = [
	{
		selector: 'edge',
		style: {
			lineColor: '#74857e',
			opacity: 0.6,
			width: 1.8,
			lineStyle: 'solid',
			curveStyle: 'unbundled-bezier',
			targetArrowColor: '#74857e'
		}
	},
	{
		selector: 'edge.owner',
		style: { lineColor: '#5d7f73', targetArrowColor: '#5d7f73', width: 2.4 }
	},
	{
		selector: 'edge.tag',
		style: {
			lineColor: '#7c6fb0',
			targetArrowColor: '#7c6fb0',
			lineStyle: 'dashed',
			width: 1.7
		}
	},
	{
		selector: 'edge.subnet',
		style: {
			lineColor: '#a5663f',
			targetArrowColor: '#a5663f',
			lineStyle: 'dotted'
		}
	},
	{
		selector: 'edge.acl',
		style: {
			lineColor: '#438aa1',
			targetArrowColor: '#438aa1',
			targetArrowShape: 'triangle',
			width: 2.2
		}
	},
	{
		selector: 'edge.scope-ssh',
		style: { lineColor: '#2f9f68', targetArrowColor: '#2f9f68', width: 2.8 }
	},
	{
		selector: 'edge.scope-http',
		style: { lineColor: '#438aa1', targetArrowColor: '#438aa1', width: 2.4 }
	},
	{
		selector: 'edge.scope-broad',
		style: { lineColor: '#b0892f', targetArrowColor: '#b0892f', width: 3.1 }
	},
	{
		selector: 'edge.scope-custom, edge.scope-limited',
		style: {
			lineColor: '#7c6fb0',
			targetArrowColor: '#7c6fb0',
			lineStyle: 'dashed',
			width: 2.3
		}
	},
	{
		selector: 'edge.local',
		style: {
			curveStyle: 'straight',
			lineColor: '#2f9f68',
			targetArrowColor: '#2f9f68',
			opacity: 0.66,
			width: 2.2
		}
	},
	{
		selector: 'edge.state-added',
		style: {
			opacity: 0.94,
			width: 3.3
		}
	},
	{
		selector: 'edge.state-removed',
		style: {
			lineColor: '#b94c4c',
			targetArrowColor: '#b94c4c',
			lineStyle: 'dotted',
			opacity: 0.78,
			width: 2.8
		}
	},
	{
		selector: 'edge.state-changed',
		style: {
			opacity: 0.9,
			width: 3
		}
	},
	{
		selector: 'edge.state-ghost-denied',
		style: {
			lineColor: '#9aa7a1',
			targetArrowColor: '#9aa7a1',
			lineStyle: 'dotted',
			opacity: 0.42,
			width: 1.8
		}
	},
	{ selector: 'edge.focused', style: { opacity: 0.96, width: 3.3 } },
	{
		selector: 'edge.selected',
		style: {
			opacity: 1,
			width: 4.4
		}
	}
];

function toCytoscapeStyle(style: EdgeStylePatch): Record<string, string | number> {
	const mapped: Record<string, string | number> = {};
	if (style.lineColor !== undefined) mapped['line-color'] = style.lineColor;
	if (style.targetArrowColor !== undefined) mapped['target-arrow-color'] = style.targetArrowColor;
	if (style.lineStyle !== undefined) mapped['line-style'] = style.lineStyle;
	if (style.curveStyle !== undefined) mapped['curve-style'] = style.curveStyle;
	if (style.width !== undefined) mapped.width = style.width;
	if (style.opacity !== undefined) mapped.opacity = style.opacity;
	if (style.targetArrowShape !== undefined) mapped['target-arrow-shape'] = style.targetArrowShape;
	return mapped;
}

export function graphEdgeStylesheet(): StylesheetJson {
	return EDGE_STYLE_RULES.map((rule) => {
		const style = toCytoscapeStyle(rule.style);
		if (rule.selector === 'edge') {
			Object.assign(style, {
				'control-point-distances': (ele: { data: (key: string) => unknown }) => {
					const distances = (ele.data('cpDistances') as number[] | undefined) ?? [40, 40];
					return distances.join(' ');
				},
				'control-point-weights': (ele: { data: (key: string) => unknown }) => {
					const weights = (ele.data('cpWeights') as number[] | undefined) ?? [0.25, 0.75];
					return weights.join(' ');
				}
			});
		}
		return { selector: rule.selector, style };
	});
}
