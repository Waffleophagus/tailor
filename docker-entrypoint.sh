#!/bin/sh
set -eu

mode="${TAILOR_TAILSCALE_MODE:-auto}"
if [ "$mode" = "auto" ]; then
	if [ -n "${TAILSCALE_AUTHKEY:-}" ]; then
		mode="embedded"
	elif [ -n "${TAILOR_LOCALAPI_ENDPOINT:-}" ] || [ -n "${TAILOR_LOCALAPI_SOCKET:-}" ]; then
		mode="external"
	else
		mode="embedded"
	fi
fi

if [ "$mode" = "embedded" ]; then
	: "${TAILOR_LOCALAPI_ENDPOINT:=${TAILOR_LOCALAPI_SOCKET:-/var/run/tailscale/tailscaled.sock}}"
	: "${TAILSCALE_STATE_DIR:=/var/lib/tailscale}"

	mkdir -p "$(dirname "$TAILOR_LOCALAPI_ENDPOINT")" "$TAILSCALE_STATE_DIR"
	export TAILOR_LOCALAPI_ENDPOINT

	tailscaled \
		--socket="$TAILOR_LOCALAPI_ENDPOINT" \
		--state="$TAILSCALE_STATE_DIR/tailscaled.state" \
		--tun="${TAILSCALE_TUN:-userspace-networking}" \
		${TAILSCALED_EXTRA_ARGS:-} &
	tailscaled_pid="$!"

	trap 'kill "$tailscaled_pid" >/dev/null 2>&1 || true' INT TERM EXIT

	for _ in 1 2 3 4 5 6 7 8 9 10; do
		[ -S "$TAILOR_LOCALAPI_ENDPOINT" ] && break
		sleep 0.2
	done

	if [ -n "${TAILSCALE_AUTHKEY:-}" ]; then
		tailscale --socket="$TAILOR_LOCALAPI_ENDPOINT" up \
			--authkey="$TAILSCALE_AUTHKEY" \
			--hostname="${TAILSCALE_HOSTNAME:-tailor}" \
			${TAILSCALE_UP_EXTRA_ARGS:-}
	fi
elif [ "$mode" != "external" ]; then
	echo "unknown TAILOR_TAILSCALE_MODE: $mode" >&2
	exit 2
fi

exec /usr/local/bin/tailor
