package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

type buildAllCmd struct {
	buildFlags `flatten:""`
}

func (buildAllCmd) Description() string {
	return "build every archive and host this machine can package"
}

func (c *buildAllCmd) Run(ctx context.Context) error {
	session, ctx := sessionFrom(ctx)
	var paths []string
	err := progress.Run(session, ctx, func(ctx context.Context) error {
		// Control does not take a pool worker. produce waits on the
		// workflow future for every archive and host step.
		taskgroup.Go(ctx, "release", taskgroup.Control, func(ctx context.Context, _ *taskgroup.Status) error {
			written, err := c.produce(ctx)
			paths = written
			return err
		})
		return nil
	})
	if err != nil {
		return err
	}
	for _, path := range paths {
		fmt.Fprintln(os.Stdout, path)
	}
	return nil
}

func (c *buildAllCmd) produce(ctx context.Context) ([]string, error) {
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
