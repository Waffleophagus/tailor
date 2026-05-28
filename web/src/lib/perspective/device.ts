import type { Device } from '../api/schemas';

export const PERSPECTIVE_DEVICE_OS = 'perspective';

export function perspectiveDeviceID(selector: string) {
	return `perspective:${selector}`;
}

export function isPerspectiveDevice(device: Device | undefined) {
	if (!device) return false;
	return device.id.startsWith('perspective:') || device.os === PERSPECTIVE_DEVICE_OS;
}

export function createPerspectiveDevice(selector: string): Device {
	return {
		id: perspectiveDeviceID(selector),
		name: selector,
		ip: '',
		tailscaleIps: [],
		os: PERSPECTIVE_DEVICE_OS,
		online: true,
		owner: selector,
		tags: [],
		subnetRouter: false,
		routedSubnets: []
	};
}

export function perspectiveSelectorFromDevice(device: Device | undefined) {
	if (!device || !isPerspectiveDevice(device)) return '';
	if (device.id.startsWith('perspective:')) {
		return device.id.slice('perspective:'.length);
	}
	return device.name || device.owner;
}
