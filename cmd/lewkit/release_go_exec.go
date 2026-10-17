package main

import (
	"context"
	"os"

	"github.com/lewtec/lewkit/x/cmd"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/lewtec/lewkit/x/toolchain/load"
)

func (c *goBuildCmd) Run(ctx context.Context) error {
	return c.execute(ctx, false, nil)
}

func (c *goRunCmd) Run(ctx context.Context) error {
	return c.execute(ctx, true, cmd.Values(c.args))
}

func (c *goBuildCmd) execute(ctx context.Context, run bool, args []string) error {
	ldflags, err := load.SplitFlags(c.ldflags.Value())
	if err != nil {
		return err
	}
	dir := c.dir.Value()
	if dir == "" {
		dir = "."
	}
	g, err := load.Load(ctx, dir, cmd.Values(c.pkgs), load.SplitTags(c.tags.Value()))
	if err != nil {
		return err
	}
	std, err := load.Archives(c.pkgdir.Value())
	if err != nil {
		return err
	}
	work, err := os.MkdirTemp("", "lewkit-go-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	out, err := gotool.BuildLoaded(ctx, gotool.Loaded{
		Graph:       g,
		Std:         std,
		Work:        work,
		Out:         c.output.Value(),
		Ldflags:     ldflags,
		RequireMain: run || c.output.Value() != "",
	})
	if err != nil || !run {
		return err
	}
	command := execdriver.MustCommand(ctx, out, args...)
	command.Dir = g.Dir
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return execdriver.Run(ctx, command)
}
