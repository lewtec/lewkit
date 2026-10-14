package generate

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/generate/enum"
)

type enumCmd struct {
	file []cmd.StringArg `help:"Go file whose marked types are enums"`
}

func (enumCmd) Description() string {
	return "String, Parse, and text marshaling for enums"
}

func (c *enumCmd) Run(ctx context.Context) error {
	return enum.Run(ctx, cmd.Values(c.file))
}
