#!/bin/sh
# golangci-lint over this module, skipping the vendored compiler tree.
# Stdout is SARIF. The package list stays on the script's own stdout
# capture so it is not mixed into the report.
set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
pkgs=$(sh x/toolchain/packages.sh)
if [ -z "$pkgs" ]; then
	echo "lint: no packages" >&2
	exit 1
fi
# shellcheck disable=SC2086
exec golangci-lint run \
	--output.sarif.path=stdout \
	--show-stats=false \
	--issues-exit-code=0 \
	$pkgs
