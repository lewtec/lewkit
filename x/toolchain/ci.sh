#!/bin/sh
# Test, then lint. mise would otherwise run them together, and the
# toolchain package pulls the compiler into both at once.
set -eu
cd "$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"
mise run test
mise run lint
