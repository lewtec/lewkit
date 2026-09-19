// Package wasm embeds the glslang reactor.
//
// glslang.wasm is built from Khronos glslang plus compile.c via
// generate.sh (mise: emscripten + cmake). workspaced places the
// artifact into this directory.
package wasm

import _ "embed"

//go:embed glslang.wasm
var Lib []byte
