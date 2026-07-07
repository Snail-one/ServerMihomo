#!/bin/sh
set -eu

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)
cd "$script_dir"

: "${GOCACHE:=$script_dir/.cache/go-build}"
export GOCACHE
mkdir -p "$GOCACHE"

case "${1:-}" in
	"")
		;;
	*)
		echo "Usage: sh build-only.sh" >&2
		exit 2
		;;
esac

version=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || printf dev)}
commit=$(git rev-parse --short HEAD 2>/dev/null || printf unknown)
build_date=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
ldflags="-X snailproxy/internal/version.Version=$version -X snailproxy/internal/version.Commit=$commit -X snailproxy/internal/version.BuildDate=$build_date"

GOOS=linux go build -ldflags "$ldflags" -o snailproxy ./cmd/snailproxy
