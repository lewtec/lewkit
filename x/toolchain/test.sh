#!/bin/sh
# go test this module, skipping the vendored compiler tree.
set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
pkgs=$(sh x/toolchain/packages.sh)
if [ -z "$pkgs" ]; then
	echo "test: no packages" >&2
	exit 1
fi
# cmd/lewkit's TestReleaseGoLewkit builds this module with the embedded
# compiler. The link was still running when the default 10m binary timeout fired.
self=github.com/lewtec/lewkit/cmd/lewkit
rest=$(printf '%s\n' "$pkgs" | grep -v "^${self}$" || true)
if [ -n "$rest" ]; then
	# Import paths have no spaces.
	# shellcheck disable=SC2086
	go test $rest
fi
go test -timeout 40m "$self"
