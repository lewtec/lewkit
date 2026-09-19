// Package glsl compiles Vulkan GLSL to SPIR-V using an embedded
// glslang reactor running in wazero.
//
// [Compile] takes GLSL source in memory. [IsSPIRV] reports the SPIR-V magic.
package glsl
