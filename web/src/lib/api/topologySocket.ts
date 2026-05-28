import { socketMessageSchema, type LocalAPIStatusResponse, type TopologyResponse } from './schemas';

type ConnectionState = 'connecting' | 'connected' | 'reconnecting' | 'disconnected';

interface TopologySocketHandlers {
	onSnapshot: (topology: TopologyResponse) => void;
	onUnavailable: (status: LocalAPIStatusResponse) => void;
	onConnectionState: (state: ConnectionState) => void;
	onError: (error: Error) => void;
}

export function connectTopologySocket(handlers: TopologySocketHandlers) {
	let socket: WebSocket | undefined;
	let reconnectTimer: number | undefined;
	let reconnectAttempt = 0;
	let closed = false;

	function connect() {
		handlers.onConnectionState(reconnectAttempt === 0 ? 'connecting' : 'reconnecting');
		socket = new WebSocket(topologySocketURL());

		socket.addEventListener('open', () => {
			reconnectAttempt = 0;
			handlers.onConnectionState('connected');
		});

		socket.addEventListener('message', (event) => {
			const parsedJSON = parseJSON(event.data);
			if (parsedJSON instanceof Error) {
				handlers.onError(parsedJSON);
				return;
			}

			const parsedMessage = socketMessageSchema.safeParse(parsedJSON);
			if (!parsedMessage.success) {
				handlers.onError(parsedMessage.error);
				return;
			}

			switch (parsedMessage.data.type) {
				case 'topology.snapshot':
					handlers.onSnapshot(parsedMessage.data.payload);
					break;
				case 'localapi.unavailable':
					handlers.onUnavailable(parsedMessage.data.payload);
					break;
			}
		});

		socket.addEventListener('close', () => {
			if (closed) {
				handlers.onConnectionState('disconnected');
				return;
			}
			scheduleReconnect();
		});

		socket.addEventListener('error', () => {
			handlers.onError(new Error('Topology socket failed'));
		});
	}

	function scheduleReconnect() {
		reconnectAttempt += 1;
		handlers.onConnectionState('reconnecting');
		const delay = Math.min(1000 * 2 ** (reconnectAttempt - 1), 15000);
		reconnectTimer = window.setTimeout(connect, delay);
	}

	connect();

	return () => {
		closed = true;
		if (reconnectTimer !== undefined) {
			window.clearTimeout(reconnectTimer);
		}
		socket?.close(1000, 'component destroyed');
	};
}

function topologySocketURL() {
	const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
	return `${protocol}//${window.location.host}/api/topology/socket`;
}

function parseJSON(value: unknown) {
	if (typeof value !== 'string') {
		return new Error('Topology socket received a non-text message');
	}

	try {
		return JSON.parse(value) as unknown;
	} catch (error) {
		return error instanceof Error ? error : new Error(String(error));
	}
}
