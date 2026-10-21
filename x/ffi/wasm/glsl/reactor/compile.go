package reactor

import (
	"context"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
)

// Compile turns Vulkan GLSL compute source into SPIR-V.
// It does not read the registry.
func Compile(ctx context.Context, src []byte) ([]byte, error) {
	return CompileGlslang(ctx, glsl.StageCompute, src)
}

// CompileStage turns Vulkan GLSL for stage into SPIR-V.
// It does not read the registry.
func CompileStage(ctx context.Context, stage glsl.Stage, src []byte) ([]byte, error) {
	return CompileGlslang(ctx, stage, src)
}

// CompileGlslang compiles src with the embedded glslang reactor.
// It does not read the registry. Codegen uses it so the committed
// SPIR-V comes from this reactor.
func CompileGlslang(ctx context.Context, stage glsl.Stage, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, glsl.ErrEmpty
	}
	if !stage.IsValid() {
		return nil, glsl.ErrCompile
	}
	if glsl.IsSPIRV(src) {
		if len(src)%4 != 0 {
			return nil, glsl.ErrCompile
		}
		return append([]byte(nil), src...), nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return compileSPIRV(ctx, stage, src)
}
