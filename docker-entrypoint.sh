#!/bin/sh
set -eu

if [ "${TAILOR_TAILSCALE_MODE:-embedded}" = "embedded" ]; then
	: "${TAILOR_LOCALAPI_SOCKET:=/var/run/tailscale/tailscaled.sock}"
	: "${TAILSCALE_STATE_DIR:=/var/lib/tailscale}"

	mkdir -p "$(dirname "$TAILOR_LOCALAPI_SOCKET")" "$TAILSCALE_STATE_DIR"
	export TAILOR_LOCALAPI_SOCKET

	tailscaled \
		--socket="$TAILOR_LOCALAPI_SOCKET" \
		--state="$TAILSCALE_STATE_DIR/tailscaled.state" \
		--tun="${TAILSCALE_TUN:-userspace-networking}" \
		${TAILSCALED_EXTRA_ARGS:-} &
	tailscaled_pid="$!"

	trap 'kill "$tailscaled_pid" >/dev/null 2>&1 || true' INT TERM EXIT

	for _ in 1 2 3 4 5 6 7 8 9 10; do
		[ -S "$TAILOR_LOCALAPI_SOCKET" ] && break
		sleep 0.2
	done

	if [ -n "${TAILSCALE_AUTHKEY:-}" ]; then
		tailscale --socket="$TAILOR_LOCALAPI_SOCKET" up \
			--authkey="$TAILSCALE_AUTHKEY" \
			--hostname="${TAILSCALE_HOSTNAME:-tailor}" \
			${TAILSCALE_UP_EXTRA_ARGS:-}
	fi
fi

exec /usr/local/bin/tailor
