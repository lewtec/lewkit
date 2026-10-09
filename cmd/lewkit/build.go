package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/build/gen/common"
	"github.com/lewtec/lewkit/x/build/sign"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/release"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

type buildFlags struct {
	appConfig `flatten:""`
	out       cmd.StringArg `long:"out" help:"output path or directory" default:"dist"`
	file      cmd.StringArg `long:"archive" help:"archive file name" default:""`
	work      cmd.StringArg `long:"workdir" help:"generated host directory" default:""`
	sdk       cmd.StringArg `long:"sdk" help:"iphonesimulator or iphoneos" default:"iphonesimulator"`
	goOnly    cmd.Flag      `long:"go-only" help:"stop after the Go binary"`
	app       cmd.Flag      `long:"app" help:"package a host app instead of a binary archive"`
	cgo       cmd.Flag      `long:"cgo" help:"android: build with cgo and the NDK clang for GOARCH"`
}

type buildCmd struct {
	buildFlags `flatten:""`
}

func (buildCmd) Description() string {
	return "build a binary, or a host app with --app"
}

func (c *buildCmd) Run(ctx context.Context) error {
	session, ctx := sessionFrom(ctx)
	var paths []string
	err := progress.Run(session, ctx, func(ctx context.Context) error {
		// Control does not take a pool worker. produce waits for the
		// icons step and the goos/goarch step on this goroutine.
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

func (c *buildFlags) produce(ctx context.Context) ([]string, error) {
	goos := c.goos.Value()
	goarch := c.goarch.Value()
	target := build.Target{GOOS: goos, GOARCH: goarch}
	identity, err := c.identity()
	if err != nil {
		return nil, err
	}
	if !c.app.Value() {
		return c.archive(ctx, target, identity)
	}
	out, err := artifactPath(goos, goarch, c.out.Value(), c.spec())
	if err != nil {
		return nil, err
	}
	work := c.work.Value()
	if c.goOnly.Value() && strings.TrimSpace(work) == "" {
		work, err = os.MkdirTemp("", release.Name()+"-app-")
		if err != nil {
			return nil, err
		}
	}
	host := build.Host{
		Spec:   c.spec(),
		Out:    out,
		Work:   work,
		GOARCH: goarch,
		SDK:    c.sdk.Value(),
		GoOnly: c.goOnly.Value(),
		CGO:    c.cgo.Value(),
		Sign:   identity,
	}
	path, err := host.Build(ctx, goos)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func (c *buildFlags) identity() (*sign.Identity, error) {
	raw := c.p12.Value()
	if raw == nil {
		return nil, nil
	}
	return sign.LoadPKCS12(raw, "")
}

// artifactPath turns a directory such as dist into one host file whose
// name contains GOOS and GOARCH. A path that already names a file is kept.
func artifactPath(goos, goarch, out string, spec build.Spec) (string, error) {
	out = strings.TrimSpace(out)
	if out == "" {
		out = "dist"
	}
	info, err := os.Stat(out)
	asDir := err == nil && info.IsDir()
	if err != nil && errors.Is(err, fs.ErrNotExist) && filepath.Ext(out) == "" {
		asDir = true
	}
	if !asDir {
		return out, nil
	}
	cfg, _, err := spec.Load()
	if err != nil {
		return "", err
	}
	return filepath.Join(out, build.AppFile(common.ProductName(cfg.PackageID, cfg.AppName), cfg.PackageID, goos, goarch)), nil
}

func (c *buildFlags) archive(ctx context.Context, target build.Target, identity *sign.Identity) ([]string, error) {
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
		Sign:    identity,
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
