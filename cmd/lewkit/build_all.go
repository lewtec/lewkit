package main

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/build/gocmd"
)

type buildAllCmd struct {
	buildFlags `flatten:""`
}

func (buildAllCmd) Description() string {
	return "build every archive and host this machine can package"
}

func (c *buildAllCmd) Run(ctx context.Context) error {
	return runRelease(ctx, c.produce)
}

func (c *buildAllCmd) produce(ctx context.Context) ([]string, error) {
	ctx = gocmd.WithCompiler(ctx, c.compiler.Value())
	identity, err := c.identity()
	if err != nil {
		return nil, err
	}
	return build.Uber{
		Spec: c.spec(),
		Out:  c.out.Value(),
		Name: c.file.Value(),
		SDK:  c.sdk.Value(),
		CGO:  c.cgo.Value(),
		Sign: identity,
	}.Run(ctx)
}
