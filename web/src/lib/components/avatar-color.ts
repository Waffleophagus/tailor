export const osColors: Record<string, string> = {
	windows: '#01A6F0',
	android: '#32DE84',
	linux: '#F4BC00',
	bsd: '#B5010F',
	macOS: '#A2AAAD',
	ios: '#FFFFFF',
	tvos: '#FA6C1B'
};

export function palette(value: string): string {
	if (Object.prototype.hasOwnProperty.call(osColors, value)) return osColors[value];
	const colors = ['#438aa1', '#a5663f', '#7c6fb0', '#b0892f', '#5d7f73', '#b45f74', '#5973b0'];
	let hash = 0;
	for (let i = 0; i < value.length; i += 1) {
		hash = (hash + value.charCodeAt(i) * (i + 1)) % colors.length;
	}
	return colors[hash];
}
