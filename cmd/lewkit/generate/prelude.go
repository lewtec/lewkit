package generate

import (
	"context"

	"github.com/lewtec/lewkit/x/generate/prelude"
)

type preludeCmd struct{}

func (preludeCmd) Description() string {
	return "blank-import preludes for every root.go in the working directory"
}

func (preludeCmd) Run(ctx context.Context) error {
	return prelude.Run(ctx, ".", "prelude/prelude.go")
}
