package entry

import (
	"context"

	"github.com/lewtec/lewkit/x/build/gen/common"
)

// goProvider resolves ref as a Go main directory.
type goProvider struct{}

func (goProvider) Name() string { return "go" }

func (goProvider) Main(ctx context.Context, request Request) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	return common.ResolveGoMain(request.Ref, request.Base)
}
