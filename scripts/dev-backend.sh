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

# The server creates the DB on first start; create the directory first so
# the bind mount isn't created by Docker (as root).
mkdir -p data
go -C backend build -o "$root/bin/famodash-dev" .

server_pid=""
cleanup() {
	trap - EXIT INT TERM HUP
	if [[ -n "$server_pid" ]] && kill -0 "$server_pid" 2>/dev/null; then
		kill -INT "$server_pid" 2>/dev/null || true
		wait "$server_pid" 2>/dev/null || true
	fi
	echo "Stopping dev services..."
	"${compose[@]}" down
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

"${compose[@]}" up -d
echo "Mailpit:    http://localhost:8025"
echo "sqlite-web: http://localhost:8081"

# Run in the background and wait, so signals are handled right away instead of
# after the server exits. The working directory is the repository root, so
# relative paths from .env resolve against it.
"$root/bin/famodash-dev" &
server_pid=$!
wait "$server_pid"
