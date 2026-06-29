#!/bin/sh
set -eu

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)
repo_root=$(CDPATH= cd "$script_dir/.." && pwd)
cd "$repo_root"

: "${GOCACHE:=$repo_root/.cache/go-build}"
export GOCACHE
mkdir -p "$GOCACHE"

generate_resources=1
case "${1:-}" in
	"")
		;;
	--skip-generate)
		generate_resources=0
		shift
		;;
	*)
		echo "Usage: scripts/build.sh [--skip-generate]" >&2
		exit 2
		;;
esac

if [ "$generate_resources" -eq 1 ]; then
	env -u GOOS -u GOARCH go generate ./resources
fi

env -u GOOS -u GOARCH go build -o snailproxy .
