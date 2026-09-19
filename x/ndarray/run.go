package ndarray

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lewtec/lewkit/x/wasm/glsl"
)

// SPIRV compiles the fused GLSL to SPIR-V. That is the shader; a device
// pipeline is the evaluator's job.
func (k *Kernel) SPIRV(ctx context.Context) ([]byte, error) {
	if k == nil {
		return nil, ErrOp
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	src, err := k.GLSL()
	if err != nil {
		return nil, err
	}
	spirv, err := glsl.Load(ctx, []byte(src))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOp, err)
	}
	slog.Debug("ndarray spirv", "bytes", len(spirv))
	return spirv, nil
}
