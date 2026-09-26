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
	goos    goosArg       `long:"goos" help:"target GOOS"`
	goarch  goarchArg     `long:"goarch" help:"target GOARCH"`
	dir     cmd.StringArg `help:"module directory" default:"."`
	id      cmd.StringArg `long:"id" help:"reverse-domain app id"`
	version cmd.StringArg `long:"version" help:"version stamped into the binary" default:""`
	out     cmd.StringArg `long:"out" help:"output path or directory" default:"dist"`
	name    cmd.StringArg `long:"name" help:"archive name" default:""`
	config  cmd.StringArg `long:"config" help:"eletrocromo.json for android, darwin, and ios" default:""`
	work    cmd.StringArg `long:"workdir" help:"generated host directory" default:""`
	sdk     cmd.StringArg `long:"sdk" help:"iphonesimulator or iphoneos" default:"iphonesimulator"`
	goOnly  cmd.Flag      `long:"go-only" help:"stop after the Go binary"`
	app     cmd.Flag      `long:"app" help:"package a host app instead of a binary archive"`
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
		Context: ctx,
		Config:  c.config.Value(),
		Out:     c.out.Value(),
		Work:    c.work.Value(),
		GOARCH:  goarch,
		SDK:     c.sdk.Value(),
		GoOnly:  c.goOnly.Value(),
	}
	switch goos {
	case "darwin":
		return c.host(func() (string, error) { return build.Mac(host) })
	case "android":
		return c.host(func() (string, error) { return build.Android(host) })
	case "ios":
		return c.host(func() (string, error) { return build.IOS(host) })
	default:
		return nil, fmt.Errorf("%s has no app package", goos)
	}
}

func (c *buildCmd) archive(ctx context.Context, target build.Target) ([]string, error) {
	id := c.id.Value()
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
	return build.Archives(build.Job{
		Context: ctx,
		Dir:     c.dir.Value(),
		Out:     c.out.Value(),
		Name:    c.name.Value(),
		AppID:   id,
		Version: c.version.Value(),
		Targets: []build.Target{target},
	})
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
