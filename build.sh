#!/bin/sh
set -eu

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)
cd "$script_dir"

: "${GOCACHE:=$script_dir/.cache/go-build}"
export GOCACHE
mkdir -p "$GOCACHE"

env -u GOOS -u GOARCH go generate ./resources
sh "$script_dir/build-only.sh"
