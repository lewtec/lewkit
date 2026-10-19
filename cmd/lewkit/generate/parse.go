package generate

import (
	"context"

	"github.com/lewtec/lewkit/x/generate/parse"
)

type parseCmd struct{}

func (parseCmd) Description() string {
	return "LR(1) table for the language in the working directory"
}

func (parseCmd) Run(ctx context.Context) error {
	return parse.Run(ctx)
}
