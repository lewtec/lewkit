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
		taskgroup.Go(ctx, c.goos.Value()+"/"+c.goarch.Value(), taskgroup.CPU, func(ctx context.Context, status *taskgroup.Status) error {
			done := status.Unit()
			defer done()
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
	return c.host(func() (string, error) {
		return packageHost(ctx, c.spec(), goos, goarch, c.out.Value(), c.work.Value(), c.sdk.Value(), c.goOnly.Value(), c.cgo.Value(), identity)
	})
}

func (c *buildFlags) identity() (*sign.Identity, error) {
	raw := c.p12.Value()
	if raw == nil {
		return nil, nil
	}
	return sign.LoadPKCS12(raw, "")
}

func packageHost(ctx context.Context, spec build.Spec, goos, goarch, out, work, sdk string, goOnly, cgo bool, identity *sign.Identity) (string, error) {
	out, err := artifactPath(goos, out, spec)
	if err != nil {
		return "", err
	}
	host := build.Host{
		Spec:   spec,
		Out:    out,
		Work:   work,
		GOARCH: goarch,
		SDK:    sdk,
		GoOnly: goOnly,
		CGO:    cgo,
		Sign:   identity,
	}
	switch goos {
	case "darwin":
		return host.Mac(ctx)
	case "android":
		return host.Android(ctx)
	case "ios":
		return host.IOS(ctx)
	case "windows":
		return host.Windows(ctx)
	case "linux":
		return host.Linux(ctx)
	default:
		return "", fmt.Errorf("%s has no app package", goos)
	}
}

// artifactPath turns a directory such as dist into the file eletrocromo wrote:
// dist/<label>-debug.apk or dist/<App>.app. A path that already names a file is kept.
func artifactPath(goos, out string, spec build.Spec) (string, error) {
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
	switch goos {
	case "android":
		label := cfg.PackageID
		if i := strings.LastIndex(label, "."); i >= 0 {
			label = label[i+1:]
		}
		if label == "" {
			label = "app"
		}
		return filepath.Join(out, label+"-debug.apk"), nil
	case "windows":
		return filepath.Join(out, common.ProductName(cfg.PackageID, cfg.AppName)+".exe"), nil
	case "linux":
		return filepath.Join(out, common.ProductName(cfg.PackageID, cfg.AppName)+".app"), nil
	default:
		return filepath.Join(out, common.ProductName(cfg.PackageID, cfg.AppName)+".app"), nil
	}
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

func (c *buildFlags) host(run func() (string, error)) ([]string, error) {
	path, err := run()
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}
