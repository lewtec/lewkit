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
	if k.spirv != nil {
		return k.spirv, nil
	}
	spirv, err := glsl.Load(ctx, []byte(k.glsl))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOp, err)
	}
	k.spirv = spirv
	slog.Debug("ndarray spirv", "bytes", len(spirv))
	return spirv, nil
}
