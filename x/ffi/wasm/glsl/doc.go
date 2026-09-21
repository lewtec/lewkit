// Package glsl compiles Vulkan GLSL to SPIR-V using an embedded
// glslang reactor in [github.com/lewtec/lewkit/x/ffi/wasm].
//
// [Compile] takes GLSL source in memory. [IsSPIRV] reports the SPIR-V magic.
package glsl
