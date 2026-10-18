#!/bin/sh
# gofmt this module. The Go pin under x/toolchain/go/_upstream keeps
# upstream testdata that is invalid syntax on purpose. The purego pin is
# upstream text too. gofmt -w . exits 2 on the Go pin, so both trees are
# pruned.
set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
find . \
	\( -path './.git' -o -path './x/toolchain/go/_upstream' -o -path './third_party/purego' \) -prune \
	-o -type f -name '*.go' -print0 |
	xargs -0 gofmt -w
