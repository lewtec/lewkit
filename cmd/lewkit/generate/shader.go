package generate

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/generate/shader"
)

type shaderCmd struct {
	dir cmd.WorkDirArg `help:"directory to scan for name.<stage>.glsl"`
}

func (shaderCmd) Description() string {
	return "SPIR-V registry for GLSL shaders under DIR"
}

func (c *shaderCmd) Run(ctx context.Context) error {
	return shader.Run(ctx, c.dir.Value())
}
