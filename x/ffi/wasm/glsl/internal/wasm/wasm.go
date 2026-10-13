// Package wasm embeds the glslang reactor.
//
// glslang.wasm is built from Khronos glslang plus
// ../build/compile.c. go run ../build/generate runs that workflow:
// glslang.wasm depends on libs, libs depends on configure, and configure
// depends on source and tools. The wasm file is committed next to this package.
package wasm

import _ "embed"

//go:embed glslang.wasm
var Lib []byte
