#!/bin/sh
set -eu

go generate ./resources
go build -o snailproxy .
