package main

import (
	"context"
	"fmt"
	"os"

	"runtime"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

type buildCmd struct {
	appConfig `flatten:""`
	out       cmd.StringArg `long:"out" help:"output path or directory" default:"dist"`
	file      cmd.StringArg `long:"archive" help:"archive file name" default:""`
	work      cmd.StringArg `long:"workdir" help:"generated host directory" default:""`
	sdk       cmd.StringArg `long:"sdk" help:"iphonesimulator or iphoneos" default:"iphonesimulator"`
	goOnly    cmd.Flag      `long:"go-only" help:"stop after the Go binary"`
	app       cmd.Flag      `long:"app" help:"package a host app instead of a binary archive"`
	cgo       cmd.Flag      `long:"cgo" help:"android: build with cgo and the NDK clang for GOARCH"`
}

func (buildCmd) Description() string {
	return "build a binary, or a host app with --app"
}

func (c *buildCmd) Run(ctx context.Context) error {
	session, ctx := sessionFrom(ctx)
	var paths []string
	err := progress.Run(session, ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, c.goos.Value()+"/"+c.goarch.Value(), taskgroup.CPU, func(ctx context.Context, status *taskgroup.Status) error {
			done := status.Unit()
			defer done()
			written, err := c.run(ctx)
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

func (c *buildCmd) run(ctx context.Context) ([]string, error) {
	goos := c.goos.Value()
	goarch := c.goarch.Value()
	target := build.Target{GOOS: goos, GOARCH: goarch}
	if !c.app.Value() {
		return c.archive(ctx, target)
	}
	host := build.Host{
		Spec:   c.spec(),
		Out:    c.out.Value(),
		Work:   c.work.Value(),
		GOARCH: goarch,
		SDK:    c.sdk.Value(),
		GoOnly: c.goOnly.Value(),
		CGO:    c.cgo.Value(),
	}
	switch goos {
	case "darwin":
		return c.host(func() (string, error) { return host.Mac(ctx) })
	case "android":
		return c.host(func() (string, error) { return host.Android(ctx) })
	case "ios":
		return c.host(func() (string, error) { return host.IOS(ctx) })
	default:
		return nil, fmt.Errorf("%s has no app package", goos)
	}
}

func (c *buildCmd) archive(ctx context.Context, target build.Target) ([]string, error) {
	cfg, _, err := c.spec().Load()
	if err != nil {
		return nil, err
	}
	id := cfg.PackageID
	if id == "" {
		stamped, err := release.AppID()
		if err != nil {
			return nil, err
		}
		id = stamped
	}
	if err := release.ValidateAppID(id); err != nil {
		return nil, err
	}
	name := c.file.Value()
	if name == "" {
		name = cfg.AppName
	}
	return build.Job{
		Dir:     cfg.GoMain,
		Out:     c.out.Value(),
		Name:    name,
		AppID:   id,
		Version: cfg.VersionName,
		Targets: []build.Target{target},
	}.Run(ctx)
}

func sessionFrom(ctx context.Context) (*taskgroup.Session, context.Context) {
	if session := taskgroup.FromContext(ctx); session != nil {
		return session, ctx
	}
	if arg, ok := cmd.Lookup[taskgroup.Arg](ctx, "taskgroup"); ok {
		return arg.Enter(ctx, taskgroup.DefaultLimits())
	}
	return taskgroup.New(ctx, taskgroup.DefaultLimits())
}

type goosArg struct{ cmd.StringArg }

func (goosArg) ArgDefault() string { return runtime.GOOS }

type goarchArg struct{ cmd.StringArg }

func (goarchArg) ArgDefault() string { return runtime.GOARCH }

func (c *buildCmd) host(run func() (string, error)) ([]string, error) {
	if c.config.Value() == "" {
		return nil, fmt.Errorf("config is required for %s", c.goos.Value())
	}
	path, err := run()
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}
