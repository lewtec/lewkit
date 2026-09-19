// Package wasm embeds the glslang reactor.
//
// glslang.wasm is built from Khronos glslang plus
// ../build/compile.c via ../build/generate.sh (mise emscripten/cmake,
// uv python). workspaced requires the artifact here.
package wasm

import _ "embed"

//go:embed glslang.wasm
var Lib []byte
