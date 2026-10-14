package generate

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/generate/enum"
)

type enumCmd struct {
	file []cmd.StringArg `help:"Go file to read; with none, every marked file in the package"`
}

func (enumCmd) Description() string {
	return "String, Parse, and text marshaling for enums"
}

func (c *enumCmd) Run(ctx context.Context) error {
	return enum.Run(ctx, cmd.Values(c.file))
}
