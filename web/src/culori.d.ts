declare module 'culori' {
	export function converter(
		mode: string
	): (color: unknown) => { mode: string; l: number; c: number; h?: number } | undefined;
	export function differenceEuclidean(mode: string): (a: unknown, b: unknown) => number;
	export function formatHex(color: unknown): string | undefined;
	export function parse(color: string): unknown;
}
