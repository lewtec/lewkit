package main

import (
	"context"
	"os"

	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/version"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

type releaseCmd struct {
	build *buildCmd
	run   *runCmd
}

func (releaseCmd) Description() string { return "build or run an app" }

type runCmd struct {
	goos    goosArg       `long:"goos" help:"target GOOS"`
	goarch  goarchArg     `long:"goarch" help:"target GOARCH"`
	dir     cmd.StringArg `help:"module directory" default:"."`
	id      cmd.StringArg `long:"id" help:"reverse-domain app id"`
	version cmd.StringArg `long:"version" help:"version stamped into the binary" default:""`
}

func (runCmd) Description() string { return "go run the module for GOOS and GOARCH" }

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
	id := c.id.Value()
	if id == "" {
		stamped, err := release.AppID()
		if err != nil {
			return err
		}
		id = stamped
	}
	if err := release.ValidateAppID(id); err != nil {
		return err
	}
	ldflags := version.Info{Version: c.version.Value(), BuiltBy: "lewkit"}.WithAppID(id)
	return gocmd.Command{
		Verb: "run",
		Dir:  c.dir.Value(),
		Env:  append(os.Environ(), "CGO_ENABLED=0", "GOOS="+c.goos.Value(), "GOARCH="+c.goarch.Value()),
		Args: []string{"-trimpath", "-ldflags", ldflags, "."},
	}.Run(ctx)
}
