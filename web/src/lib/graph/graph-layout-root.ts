import type { Device } from '../api/schemas';

/** Pick the device that should sit at the center of the radial layout. */
export function resolveGraphLayoutRoot(
	selectedDevice: Device | undefined,
	fallbackDevice: Device | undefined,
	visibleDeviceIds: ReadonlySet<string>
): Device | undefined {
	if (selectedDevice && visibleDeviceIds.has(selectedDevice.id)) {
		return selectedDevice;
	}
	return fallbackDevice;
}
