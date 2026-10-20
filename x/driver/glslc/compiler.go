// Package glslc compiles compute GLSL to SPIR-V.
//
// [Compiler] is the driver. Native glslc is weight 80 and wins when that
// binary is on PATH. The lewkit driver is weight 40 and compiles with the
// embedded glslang reactor. Draw shaders do not use this package.
package glslc

import "context"

// Compiler turns compute GLSL into SPIR-V.
type Compiler interface {
	Compile(ctx context.Context, src []byte) ([]byte, error)
}
