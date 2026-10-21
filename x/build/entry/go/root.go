// Package gomain is the go: entry provider.
// Importing it registers the provider. x/build/entry/prelude imports it.
package gomain

import (
	"context"

	"github.com/lewtec/lewkit/x/build/entry"
	"github.com/lewtec/lewkit/x/build/gen/common"
)

// Provider resolves ref as a Go main directory.
type Provider struct{}

func init() { entry.Register(Provider{}) }

func (Provider) Name() string { return "go" }

func (Provider) Main(ctx context.Context, request entry.Request) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", context.Cause(ctx)
	}
	return common.ResolveGoMain(request.Ref, request.Base)
}
