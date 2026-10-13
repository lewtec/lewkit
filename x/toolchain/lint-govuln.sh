#!/bin/sh
# govulncheck over this module, skipping the vendored compiler tree.
# Stdout is SARIF.
set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
pkgs=$(sh x/toolchain/packages.sh)
if [ -z "$pkgs" ]; then
	echo "lint: no packages" >&2
	exit 1
fi
# shellcheck disable=SC2086
exec govulncheck --format sarif $pkgs
