#!/bin/sh
# Regenerate packages outside the vendored Go toolchain.
# x/toolchain/go/src and _upstream carry upstream //go:generate lines
# (stringer, dist) that are not part of this module's build.
set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
# classes.dex is rebuilt by go:generate. build-tools 34 d8 crashes on
# class files from javac 21 and newer. The pinned JDK reproduces the file.
if command -v mise >/dev/null 2>&1; then
	MISE_YES=1 mise install java@temurin-17.0.14+7
fi
pkgs=$(go list -e -f '{{if not .Error}}{{.ImportPath}}{{end}}' ./... | grep -v '/x/toolchain/go' | grep -v '^$' || true)
if [ -z "$pkgs" ]; then
	echo "codegen: no packages" >&2
	exit 1
fi
# Import paths have no spaces.
# shellcheck disable=SC2086
go generate $pkgs
