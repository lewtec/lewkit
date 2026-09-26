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
	desktop *desktopBuildCmd
	android *androidBuildCmd
	macos   *macosBuildCmd
	ios     *iosBuildCmd
}

func (buildCmd) Description() string { return "cross-compile a module or package a host" }

type desktopBuildCmd struct {
	dir     cmd.StringArg `help:"module directory" default:"."`
	id      cmd.StringArg `long:"id" help:"reverse-domain app id"`
	version cmd.StringArg `long:"version" help:"version stamped into the binary" default:""`
	out     cmd.StringArg `long:"out" help:"archive directory" default:"dist"`
	name    cmd.StringArg `long:"name" help:"archive name" default:""`
}

func (desktopBuildCmd) Description() string {
	return "CGO-free archives for Linux, macOS, and Windows"
}

func (c *desktopBuildCmd) Run(ctx context.Context) error {
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
	written, err := build.Desktop(ctx, c.dir.Value(), c.out.Value(), c.name.Value(), id, c.version.Value())
	if err != nil {
		return err
	}
	for _, path := range written {
		fmt.Fprintln(os.Stdout, path)
	}
	return nil
}

type androidBuildCmd struct {
	config cmd.StringArg `help:"eletrocromo.json file or directory"`
	out    cmd.StringArg `long:"out" help:"apk path" default:""`
	work   cmd.StringArg `long:"workdir" help:"gradle project directory" default:""`
	goOnly cmd.Flag      `long:"go-only" help:"stop after the Go library"`
}

func (androidBuildCmd) Description() string { return "Android debug APK" }

func (c *androidBuildCmd) Run(context.Context) error {
	path, err := build.Android(c.config.Value(), c.out.Value(), c.work.Value(), c.goOnly.Value())
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, path)
	return nil
}

type macosBuildCmd struct {
	config cmd.StringArg `help:"eletrocromo.json file or directory"`
	out    cmd.StringArg `long:"out" help:".app path" default:""`
	work   cmd.StringArg `long:"workdir" help:"xcode project directory" default:""`
	goOnly cmd.Flag      `long:"go-only" help:"stop after the Go helper"`
}

func (macosBuildCmd) Description() string { return "unsigned macOS .app" }

func (c *macosBuildCmd) Run(context.Context) error {
	path, err := build.Mac(c.config.Value(), c.out.Value(), c.work.Value(), c.goOnly.Value())
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, path)
	return nil
}

type iosBuildCmd struct {
	config cmd.StringArg `help:"eletrocromo.json file or directory"`
	out    cmd.StringArg `long:"out" help:".app path" default:""`
	work   cmd.StringArg `long:"workdir" help:"xcode project directory" default:""`
	sdk    cmd.StringArg `long:"sdk" help:"iphonesimulator or iphoneos" default:"iphonesimulator"`
	goOnly cmd.Flag      `long:"go-only" help:"stop after the c-archive"`
}

func (iosBuildCmd) Description() string { return "iOS .app" }

func (c *iosBuildCmd) Run(context.Context) error {
	path, err := build.IOS(c.config.Value(), c.out.Value(), c.work.Value(), c.sdk.Value(), c.goOnly.Value())
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, path)
	return nil
}
