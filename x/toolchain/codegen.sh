#!/bin/sh
# Regenerate packages outside the vendored Go toolchain.
# x/toolchain/go/src and _upstream carry upstream //go:generate lines
# (stringer, dist) that are not part of this module's build.
set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
pkgs=$(go list -e -f '{{if not .Error}}{{.ImportPath}}{{end}}' ./... | grep -v '/x/toolchain/go' | grep -v '^$' || true)
if [ -z "$pkgs" ]; then
	echo "codegen: no packages" >&2
	exit 1
fi
# Import paths have no spaces.
# shellcheck disable=SC2086
go generate $pkgs
