package glsl

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/release"
)

// compileGLSLC runs the host glslc when it is on PATH.
// found is false when glslc is absent. The caller then uses the embedded glslang.
func compileGLSLC(ctx context.Context, stage Stage, src []byte) (spv []byte, found bool, err error) {
	if _, err := execdriver.Which(ctx, "glslc"); err != nil {
		return nil, false, err
	}
	dir, err := os.MkdirTemp("", release.Name()+"-glslc-")
	if err != nil {
		return nil, true, err
	}
	defer os.RemoveAll(dir)
	in := filepath.Join(dir, "shader"+stageExt(stage))
	out := filepath.Join(dir, "shader.spv")
	if err := os.WriteFile(in, src, 0o600); err != nil {
		return nil, true, err
	}
	var stderr bytes.Buffer
	cmd := execdriver.MustCommand(
		"glslc",
		"--target-env=vulkan1.1",
		"--target-spv=spv1.3",
		"-fshader-stage="+stageFlag(stage),
		"-o", out,
		in,
	)
	cmd.Stderr = &stderr
	if err := execdriver.Run(ctx, cmd); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, true, err
		}
		return nil, true, fmt.Errorf("%w: %s", err, msg)
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return nil, true, err
	}
	if !validSPIRV(raw) {
		return nil, true, fmt.Errorf("%w: glslc output", ErrCompile)
	}
	return raw, true, nil
}

func stageFlag(stage Stage) string {
	switch stage {
	case StageVertex:
		return "vertex"
	case StageFragment:
		return "fragment"
	default:
		return "compute"
	}
}

func stageExt(stage Stage) string {
	switch stage {
	case StageVertex:
		return ".vert"
	case StageFragment:
		return ".frag"
	default:
		return ".comp"
	}
}
