package main

import (
	"context"

	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

type releaseCmd struct {
	build *buildCmd
	run   *runCmd
}

func (releaseCmd) Description() string { return "build or run an app" }

type runCmd struct {
	buildFlags `flatten:""`
}

func (runCmd) Description() string {
	return "build, then run that artifact"
}

func (c *runCmd) Run(ctx context.Context) error {
	session, ctx := sessionFrom(ctx)
	return progress.Run(session, ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, c.goos.Value()+"/"+c.goarch.Value(), taskgroup.CPU, func(ctx context.Context, status *taskgroup.Status) error {
			done := status.Unit()
			defer done()
			return c.execute(ctx)
		})
		return nil
	})
}

func (c *runCmd) execute(ctx context.Context) error {
	paths, err := c.produce(ctx)
	if err != nil {
		return err
	}
	if c.goOnly.Value() || len(paths) == 0 {
		return nil
	}
	if c.app.Value() {
		cfg, _, err := c.spec().Load()
		if err != nil {
			return err
		}
		id := cfg.PackageID
		return launchApp(ctx, c.goos.Value(), paths[0], id)
	}
	return runBuilt(ctx, c.goos.Value(), c.goarch.Value(), paths[0])
}
