#!/usr/bin/env bash
# Runs the Go server for development together with the dev compose services
# (mailpit, sqlite-web). The services are stopped when the server exits or
# this script is interrupted or killed (SIGINT, SIGTERM, SIGHUP).
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

# Exports KEY=VALUE lines from a dotenv file. Blank lines and # comments are
# skipped, surrounding quotes are removed, and variables that are already set
# in the environment are kept.
load_env() {
	local line key value
	while IFS= read -r line || [[ -n "$line" ]]; do
		[[ "$line" =~ ^[[:space:]]*(#|$) ]] && continue
		key="${line%%=*}"
		key="${key#export }"
		key="${key//[[:space:]]/}"
		value="${line#*=}"
		if [[ "$value" =~ ^\"(.*)\"$ || "$value" =~ ^\'(.*)\'$ ]]; then
			value="${BASH_REMATCH[1]}"
		fi
		if [[ -z "${!key+set}" ]]; then
			export "$key=$value"
		fi
	done <"$1"
}

if [[ ! -f .env ]]; then
	cp .env.example .env
	echo "Created .env from .env.example; adjust it as needed."
fi
load_env .env

export DEV_UID="$(id -u)" DEV_GID="$(id -g)"
compose=(docker compose --profile dev)
# Named explicitly: compose always includes services without a profile (the
# boardly app container), which must not run next to the dev server on :8080.
dev_services=(mailpit sqlite-web)

# The server creates the DB on first start; create the directory first so
# the bind mount isn't created by Docker (as root).
mkdir -p data
go -C backend build -o "$root/bin/boardly-dev" .

server_pid=""
cleanup() {
	trap - EXIT INT TERM HUP
	if [[ -n "$server_pid" ]] && kill -0 "$server_pid" 2>/dev/null; then
		kill -INT "$server_pid" 2>/dev/null || true
		wait "$server_pid" 2>/dev/null || true
	fi
	echo "Stopping dev services..."
	"${compose[@]}" rm --stop --force "${dev_services[@]}"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

"${compose[@]}" up -d "${dev_services[@]}"
echo "Mailpit:    http://localhost:8025"
echo "sqlite-web: http://localhost:8081"

# Run in the background and wait, so signals are handled right away instead of
# after the server exits. The working directory is the repository root, so
# relative paths from .env resolve against it.
start_server() {
	"$root/bin/boardly-dev" &
	server_pid=$!
}

stop_server() {
	if [[ -n "$server_pid" ]] && kill -0 "$server_pid" 2>/dev/null; then
		kill -INT "$server_pid" 2>/dev/null || true
		wait "$server_pid" 2>/dev/null || true
	fi
	server_pid=""
}

start_server

# On a terminal, "r" rebuilds and restarts the server; the dev services keep
# running.
if [[ -t 0 ]]; then
	echo "Press r to rebuild and restart the server, Ctrl+C to stop."
	while kill -0 "$server_pid" 2>/dev/null; do
		# The timeout lets the loop notice when the server exits on its own.
		if read -rsn1 -t 1 key && [[ "$key" == r ]]; then
			echo "Rebuilding..."
			# Build first, so a compile error leaves the running server alone.
			if go -C backend build -o "$root/bin/boardly-dev" .; then
				stop_server
				start_server
				echo "Server restarted."
			else
				echo "Build failed; keeping the current server."
			fi
		fi
	done
fi
wait "$server_pid"
