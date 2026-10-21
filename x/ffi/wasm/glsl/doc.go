// Package glsl returns registered Vulkan SPIR-V for GLSL source.
//
// [CompileStage] returns SPIR-V already registered for the SHA-256 of the
// stage and the source. The lewkit generate shader command writes that
// registry for name.vertex.glsl, name.fragment.glsl, and name.compute.glsl
// files. The second-to-last extension is the stage. A missing hash is
// [ErrCompile]. [github.com/lewtec/lewkit/x/ffi/wasm/glsl/reactor] compiles
// with the embedded glslang module.
//
// [Compile] takes GLSL compute source in memory. [IsSPIRV] reports the
// SPIR-V magic.
package glsl
