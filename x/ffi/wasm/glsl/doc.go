// Package glsl compiles Vulkan GLSL to SPIR-V.
//
// [CompileStage] returns SPIR-V already registered for the SHA-256 of the
// stage and the source. The lewkit generate shader command writes that
// registry for .vert, .frag, and .comp files in a tree. A missing hash runs
// glslc when it is on PATH, and otherwise the embedded glslang reactor in
// [github.com/lewtec/lewkit/x/ffi/wasm].
//
// [Compile] takes GLSL compute source in memory. [IsSPIRV] reports the
// SPIR-V magic.
package glsl
