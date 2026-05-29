import { edgeClassList, type EdgeClassOptions } from './edge-classes';
import type { RenderEdge } from './engine';
import {
	EDGE_STYLE_RULES,
	type EdgeStylePatch,
	type LineStyle,
	type ResolvedEdgeStyle
} from './style-catalog';

const DEFAULT_STYLE: ResolvedEdgeStyle = {
	lineColor: '#74857e',
	targetArrowColor: '#74857e',
	lineStyle: 'solid',
	curveStyle: 'unbundled-bezier',
	width: 1.8,
	opacity: 0.6
};

function matchesSelector(selector: string, classes: ReadonlySet<string>): boolean {
	return selector.split(',').some((part) => {
		const trimmed = part.trim();
		if (trimmed === 'edge') return true;
		if (trimmed.startsWith('edge.')) {
			return classes.has(trimmed.slice('edge.'.length));
		}
		return false;
	});
}

function applyPatch(style: ResolvedEdgeStyle, patch: EdgeStylePatch): ResolvedEdgeStyle {
	return {
		lineColor: patch.lineColor ?? style.lineColor,
		targetArrowColor: patch.targetArrowColor ?? style.targetArrowColor,
		lineStyle: (patch.lineStyle ?? style.lineStyle) as LineStyle,
		curveStyle: patch.curveStyle ?? style.curveStyle,
		width: patch.width ?? style.width,
		opacity: patch.opacity ?? style.opacity,
		targetArrowShape: patch.targetArrowShape ?? style.targetArrowShape
	};
}

export function resolveEdgeStyle(classes: string | readonly string[]): ResolvedEdgeStyle {
	const classSet = new Set(
		(typeof classes === 'string' ? classes.split(/\s+/) : [...classes]).filter(Boolean)
	);
	let style = { ...DEFAULT_STYLE };
	for (const rule of EDGE_STYLE_RULES) {
		if (matchesSelector(rule.selector, classSet)) {
			style = applyPatch(style, rule.style);
		}
	}
	return style;
}

export function styleForEdge(edge: RenderEdge, options: EdgeClassOptions = {}): ResolvedEdgeStyle {
	return resolveEdgeStyle(edgeClassList(edge, options));
}
