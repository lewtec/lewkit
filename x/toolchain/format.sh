#!/bin/sh
# gofmt this module. The Go pin under x/toolchain/go/_upstream keeps
# upstream testdata that is invalid syntax on purpose. gofmt -w . exits 2
# on those files, so that tree is pruned.
set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
find . \
	\( -path './.git' -o -path './x/toolchain/go/_upstream' \) -prune \
	-o -type f -name '*.go' -print0 |
	xargs -0 gofmt -w
