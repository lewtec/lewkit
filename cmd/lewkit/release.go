package main

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

type releaseCmd struct {
	build *buildCmd
	run   *runCmd
}

func (releaseCmd) Description() string { return "build or run an app" }

var errHostArgs = errors.New("program arguments are not passed to a host app")

type runCmd struct {
	buildFlags `flatten:""`
	sep        *cmd.Dash
	args       []cmd.StringArg `help:"arguments for the built program"`
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
		if len(c.args) > 0 {
			return errHostArgs
		}
		cfg, _, err := c.spec().Load()
		if err != nil {
			return err
		}
		id := cfg.PackageID
		return launchApp(ctx, c.goos.Value(), paths[0], id)
	}
	return runBuilt(ctx, builtProgram{
		goos:    c.goos.Value(),
		goarch:  c.goarch.Value(),
		archive: paths[0],
		args:    cmd.Values(c.args),
	})
}
