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
	appConfig `flatten:""`
	out       cmd.StringArg `long:"out" help:"output path or directory" default:"dist"`
	work      cmd.StringArg `long:"workdir" help:"generated host directory" default:""`
	sdk       cmd.StringArg `long:"sdk" help:"iphonesimulator or iphoneos" default:"iphonesimulator"`
	app       cmd.Flag      `long:"app" help:"build the host app and launch it"`
	cgo       cmd.Flag      `long:"cgo" help:"android: build with cgo and the NDK clang for GOARCH"`
}

func (runCmd) Description() string {
	return "go run the module, or launch the host app with --app"
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
	cfg, _, err := c.spec().Load()
	if err != nil {
		return err
	}
	id := cfg.PackageID
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
	if c.app.Value() {
		path, err := packageHost(ctx, c.spec(), c.goos.Value(), c.goarch.Value(), c.out.Value(), c.work.Value(), c.sdk.Value(), false, c.cgo.Value())
		if err != nil {
			return err
		}
		return launchApp(ctx, c.goos.Value(), path, id)
	}
	ldflags := version.Info{Version: cfg.VersionName, BuiltBy: "lewkit"}.WithAppID(id)
	return gocmd.Command{
		Verb: "run",
		Dir:  cfg.GoMain,
		Env:  append(os.Environ(), "CGO_ENABLED=0", "GOOS="+c.goos.Value(), "GOARCH="+c.goarch.Value()),
		Args: []string{"-trimpath", "-ldflags", ldflags, "."},
	}.Run(ctx)
}
