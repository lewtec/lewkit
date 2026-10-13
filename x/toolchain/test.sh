#!/bin/sh
# go test this module, skipping the vendored compiler tree.
set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
pkgs=$(sh x/toolchain/packages.sh)
if [ -z "$pkgs" ]; then
	echo "test: no packages" >&2
	exit 1
fi
# Import paths have no spaces.
# shellcheck disable=SC2086
exec go test $pkgs
