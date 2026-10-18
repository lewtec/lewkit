package main

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/build/gen/ios"
	"github.com/lewtec/lewkit/x/build/gen/uki"
	"github.com/lewtec/lewkit/x/build/gen/unikernel"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/taskgroup/progress"
)

type releaseCmd struct {
	key      *keyCmd
	build    *buildCmd
	buildAll *buildAllCmd `cmd:"build-all"`
	run      *runCmd
	gobuild  *goCmd `cmd:"go"`
}

func (releaseCmd) Description() string { return "key, build, run, or compile an app" }

var (
	errHostArgs             = errs.Wrap(errs.ErrInvalid, "program arguments are not passed to a host app")
	errIOSRunNeedsSimulator = errs.Wrap(errs.ErrInvalid, "release run installs on the booted iOS simulator")
	errGuestArgs            = errs.Wrap(errs.ErrInvalid, "program arguments are not passed to a linux guest")
	errUnikernelArgs        = errs.Wrap(errs.ErrInvalid, "program arguments are not passed to a unikernel")
)

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
		taskgroup.Go(ctx, "release", taskgroup.Control, func(ctx context.Context, _ *taskgroup.Status) error {
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
	if err := c.prepareIOSRun(); err != nil {
		return nil, err
	}
	paths, err := c.produce(ctx)
	if err != nil {
		return nil, err
	}
	return c.program(ctx, paths)
}

// prepareIOSRun selects the simulator slice that simctl can launch.
// An omitted --sdk becomes iphonesimulator. --sdk iphoneos stays a device build.
func (c *runCmd) prepareIOSRun() error {
	if !c.app.Value() || c.goos.Value() != build.GOOSIOS {
		return nil
	}
	if c.sdk.ArgSet() {
		parsed, err := ios.ParseSDK(c.sdk.Value())
		if err != nil {
			return err
		}
		if parsed == ios.SDKDevice {
			return errIOSRunNeedsSimulator
		}
		return nil
	}
	if err := c.sdk.Parse(string(ios.SDKSimulator)); err != nil {
		return err
	}
	c.sdk.MarkSet()
	return nil
}

// program starts a host app in this task.
// A built binary is returned so it can run after the progress view stops.
func (c *runCmd) program(ctx context.Context, paths []string) (func(context.Context) error, error) {
	if c.goOnly.Value() || len(paths) == 0 {
		return nil, nil
	}
	goos := string(c.goos.Value())
	goarch := string(c.goarch.Value())
	if c.app.Value() {
		if len(c.args) > 0 {
			return nil, errHostArgs
		}
		cfg, _, err := c.spec().Load()
		if err != nil {
			return nil, err
		}
		return nil, launchApp(ctx, goos, paths[0], cfg.PackageID)
	}
	if uki.Guest(goos) {
		if len(c.args) > 0 {
			return nil, errGuestArgs
		}
		dir := paths[0]
		return func(ctx context.Context) error {
			return uki.Run(ctx, dir)
		}, nil
	}
	if unikernel.Firmware(goos) {
		if len(c.args) > 0 {
			return nil, errUnikernelArgs
		}
		image := paths[0]
		return func(ctx context.Context) error {
			return unikernel.Run(ctx, goos, image)
		}, nil
	}
	prog := builtProgram{
		goos:    goos,
		goarch:  goarch,
		archive: paths[0],
		args:    cmd.Values(c.args),
	}
	return func(ctx context.Context) error {
		return runBuilt(ctx, prog)
	}, nil
}
