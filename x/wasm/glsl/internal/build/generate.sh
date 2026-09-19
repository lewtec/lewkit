#!/usr/bin/env bash
# Build glslang.wasm.
# Tools: mise conda:emscripten + conda:cmake; Python via uv.
# workspaced places the resulting wasm into this directory.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
out="$(cd "$here/../wasm" && pwd)/glslang.wasm"
src="$here/compile.c"
ver=15.1.0
work=${TMPDIR:-/tmp}/lewkit-glslang-wasm
mkdir -p "$work"

PY=$(uv python find 3.13)
# conda emscripten's wasm-opt is too old for emcc 4.0.9; skip it.
mkdir -p "$work/bin"
cat >"$work/bin/wasm-opt" <<'EOF'
#!/bin/sh
out="" src=""
while [ $# -gt 0 ]; do
	if [ "$1" = "-o" ]; then out=$2; shift 2; continue; fi
	case $1 in
	--*) shift ;;
	*) src=$1; shift ;;
	esac
done
if [ -n "$out" ] && [ -n "$src" ] && [ "$out" != "$src" ]; then cp "$src" "$out"; fi
exit 0
EOF
chmod +x "$work/bin/wasm-opt"
export PATH="$work/bin:$(dirname "$PY"):$PATH"

run() {
	mise exec conda:emscripten@4.0.9 conda:cmake@4.4.3 conda:ninja@1.13.2 -- "$@"
}

tarball="$work/glslang-$ver.tar.gz"
if [[ ! -d "$work/glslang-$ver" ]]; then
	curl -fsSL -o "$tarball" "https://github.com/KhronosGroup/glslang/archive/refs/tags/$ver.tar.gz"
	tar -xzf "$tarball" -C "$work"
fi
srcroot="$work/glslang-$ver"
build="$work/build"
mkdir -p "$build"

if [[ ! -f "$build/CMakeCache.txt" ]]; then
	run emcmake cmake -S "$srcroot" -B "$build" \
		-G Ninja \
		-DCMAKE_BUILD_TYPE=Release \
		-DENABLE_OPT=OFF \
		-DENABLE_HLSL=OFF \
		-DENABLE_GLSLANG_BINARIES=OFF \
		-DENABLE_SPVREMAPPER=OFF \
		-DBUILD_TESTING=OFF \
		-DENABLE_GLSLANG_JS=OFF
fi
run emmake cmake --build "$build" --parallel

libs=()
for name in glslang SPIRV MachineIndependent GenericCodeGen \
	glslang-default-resource-limits OSDependent SPIRV-Tools-static SPIRV-Tools; do
	f=$(find "$build" -name "lib${name}.a" | head -1 || true)
	if [[ -n "${f:-}" ]]; then
		libs+=("$f")
	fi
done
if [[ ${#libs[@]} -eq 0 ]]; then
	echo "no glslang static libs in $build" >&2
	find "$build" -name '*.a' | head
	exit 1
fi

run emcc "$src" -O2 \
	-I"$srcroot" \
	"${libs[@]}" \
	--no-entry \
	--no-wasm-opt \
	-sSTANDALONE_WASM=1 \
	-sALLOW_MEMORY_GROWTH=1 \
	-sINITIAL_MEMORY=67108864 \
	-sEXPORTED_FUNCTIONS=_compile_compute,_last_error,_malloc,_free \
	-o "$out"

ls -l "$out"
