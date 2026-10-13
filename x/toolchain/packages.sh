#!/bin/sh
# Print this module's packages, one import path per line.
# x/toolchain/go/src is the rewritten Go compiler and linker. It has no
# tests of its own. go test and the Go linters must not take those
# packages as roots; loading each one is what killed CI.
set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
list=$(mktemp)
trap 'rm -f "$list"' EXIT
go list -f '{{.ImportPath}}' ./... > "$list"
grep -v '/x/toolchain/go/src' "$list" || true
