package vulkan

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/glslc"
	_ "github.com/lewtec/lewkit/x/driver/glslc/prelude"
	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
)

// compileCompute returns SPIR-V for one compute shader.
// A registered hash is that SPIR-V. A miss uses the glslc driver.
// Native glslc wins when it is on PATH. Draw shaders do not use this path.
func compileCompute(ctx context.Context, src []byte) ([]byte, error) {
	spv, err := glsl.Load(ctx, src)
	if err == nil || !errors.Is(err, glsl.ErrCompile) {
		return spv, err
	}
	compiler, err := driver.Get[glslc.Compiler](ctx)
	if err != nil {
		return nil, err
	}
	return compiler.Compile(ctx, src)
}
