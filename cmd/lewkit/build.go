package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/release"
)

type buildCmd struct {
	goos    cmd.StringArg `long:"goos" help:"target GOOS"`
	goarch  cmd.StringArg `long:"goarch" help:"target GOARCH"`
	dir     cmd.StringArg `help:"module directory" default:"."`
	id      cmd.StringArg `long:"id" help:"reverse-domain app id"`
	version cmd.StringArg `long:"version" help:"version stamped into the binary" default:""`
	out     cmd.StringArg `long:"out" help:"output path or directory" default:"dist"`
	name    cmd.StringArg `long:"name" help:"archive name" default:""`
	config  cmd.StringArg `long:"config" help:"eletrocromo.json for android, darwin, and ios" default:""`
	work    cmd.StringArg `long:"workdir" help:"generated host directory" default:""`
	sdk     cmd.StringArg `long:"sdk" help:"iphonesimulator or iphoneos" default:"iphonesimulator"`
	goOnly  cmd.Flag      `long:"go-only" help:"stop after the Go binary"`
}

func (buildCmd) Description() string {
	return "build the package for GOOS and GOARCH"
}

func (c *buildCmd) Run(ctx context.Context) error {
	goos := c.goos.Value()
	goarch := c.goarch.Value()
	if goos == "" || goarch == "" {
		return fmt.Errorf("goos and goarch are required")
	}
	switch goos {
	case "linux", "windows":
		return c.archive(ctx, build.Target{GOOS: goos, GOARCH: goarch})
	case "darwin":
		if c.config.Value() == "" {
			return c.archive(ctx, build.Target{GOOS: goos, GOARCH: goarch})
		}
		return c.host(func() (string, error) {
			return build.Mac(c.config.Value(), c.out.Value(), c.work.Value(), goarch, c.goOnly.Value())
		})
	case "android":
		return c.host(func() (string, error) {
			return build.Android(c.config.Value(), c.out.Value(), c.work.Value(), goarch, c.goOnly.Value())
		})
	case "ios":
		return c.host(func() (string, error) {
			return build.IOS(c.config.Value(), c.out.Value(), c.work.Value(), c.sdk.Value(), goarch, c.goOnly.Value())
		})
	default:
		return fmt.Errorf("unsupported GOOS %q", goos)
	}
}

func (c *buildCmd) archive(ctx context.Context, target build.Target) error {
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
	written, err := build.Archives(ctx, c.dir.Value(), c.out.Value(), c.name.Value(), id, c.version.Value(), []build.Target{target})
	if err != nil {
		return err
	}
	for _, path := range written {
		fmt.Fprintln(os.Stdout, path)
	}
	return nil
}

func (c *buildCmd) host(run func() (string, error)) error {
	if c.config.Value() == "" {
		return fmt.Errorf("config is required for %s", c.goos.Value())
	}
	path, err := run()
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, path)
	return nil
}
