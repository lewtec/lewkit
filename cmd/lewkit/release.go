package main

import (
	"context"
	"errors"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

type releaseCmd struct {
	key   *keyCmd
	build *buildCmd
	run   *runCmd
}

func (releaseCmd) Description() string { return "key, build, or run an app" }

var errHostArgs = errors.New("program arguments are not passed to a host app")

type runCmd struct {
	buildFlags `flatten:""`
	sep        *cmd.Dash
	args       []cmd.StringArg `help:"arguments for the built program"`
}

func (runCmd) Description() string {
	return "build, then run that artifact"
}

type pendingRun struct {
	err error
	run func(context.Context) error
}

func (c *runCmd) Run(ctx context.Context) error {
	session, ctx := sessionFrom(ctx)
	ch := make(chan pendingRun, 1)
	var pending pendingRun
	err := progress.Run(session, ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, c.goos.Value()+"/"+c.goarch.Value(), taskgroup.CPU, func(ctx context.Context, status *taskgroup.Status) error {
			done := status.Unit()
			defer done()
			run, err := c.stage(ctx)
			ch <- pendingRun{err: err, run: run}
			return err
		})
		// Nested progress.Run does not wait. Receive so the build is done
		// before the outer view stops and the program starts.
		pending = <-ch
		return pending.err
	})
	if err != nil {
		return err
	}
	if pending.run != nil {
		entry.After(pending.run)
	}
	return nil
}

func (c *runCmd) stage(ctx context.Context) (func(context.Context) error, error) {
	paths, err := c.produce(ctx)
	if err != nil {
		return nil, err
	}
	return c.program(ctx, paths)
}

// program starts a host app in this task.
// A built binary is returned so it can run after the progress view stops.
func (c *runCmd) program(ctx context.Context, paths []string) (func(context.Context) error, error) {
	if c.goOnly.Value() || len(paths) == 0 {
		return nil, nil
	}
	if c.app.Value() {
		if len(c.args) > 0 {
			return nil, errHostArgs
		}
		cfg, _, err := c.spec().Load()
		if err != nil {
			return nil, err
		}
		return nil, launchApp(ctx, c.goos.Value(), paths[0], cfg.PackageID)
	}
	prog := builtProgram{
		goos:    c.goos.Value(),
		goarch:  c.goarch.Value(),
		archive: paths[0],
		args:    cmd.Values(c.args),
	}
	return func(ctx context.Context) error {
		return runBuilt(ctx, prog)
	}, nil
}
