#!/bin/sh
set -eu

env -u GOOS -u GOARCH go generate ./resources
go build -o snailproxy .
