package native

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
)

func (compiler) Compile(ctx context.Context, src []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "lewkit-glslc-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	in := filepath.Join(dir, "shader.comp")
	if err := os.WriteFile(in, src, 0o600); err != nil {
		return nil, err
	}
	outPath := filepath.Join(dir, "shader.spv")
	cmd := execdriver.MustCommand(ctx, "glslc", "-fshader-stage=compute", "-o", outPath, in)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if _, err := execdriver.Output(ctx, cmd); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("%w: glslc: %s", glsl.ErrCompile, msg)
	}
	out, err := os.ReadFile(outPath)
	if err != nil {
		return nil, err
	}
	if !glsl.IsSPIRV(out) || len(out)%4 != 0 {
		return nil, fmt.Errorf("%w: glslc: output is not SPIR-V", glsl.ErrCompile)
	}
	return out, nil
}
