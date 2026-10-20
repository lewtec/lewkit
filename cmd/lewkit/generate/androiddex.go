package generate

import (
	"context"

	"github.com/lewtec/lewkit/x/generate/androiddex"
)

type androiddexCmd struct{}

func (androiddexCmd) Description() string {
	return "Android host classes.dex in the working directory"
}

func (androiddexCmd) Run(ctx context.Context) error {
	return androiddex.Run(ctx, ".")
}
