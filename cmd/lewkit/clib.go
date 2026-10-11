package main

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/generate/clib"
)

type clibCmd struct {
	file    cmd.StringArg   `help:"C header"`
	out     cmd.StringArg   `long:"out" default:"" help:"Go file to write"`
	pkg     cmd.StringArg   `long:"package" default:"" help:"Go package name"`
	include []cmd.StringArg `long:"include" short:"I" help:"include directory"`
	define  []cmd.StringArg `long:"define" short:"D" help:"macro NAME or NAME=value"`
	bind    cmd.StringArg   `long:"bind" default:"" help:"struct holding the C functions; API when empty"`
	build   cmd.StringArg   `long:"build" default:"" help:"also link this shared library without cgo"`
}

func (clibCmd) Description() string {
	return "C shared library from a header, built without cgo"
}

func (c *clibCmd) Run(ctx context.Context) error {
	cfg := clib.Config{
		File:    c.file.Value(),
		Out:     c.out.Value(),
		Package: c.pkg.Value(),
		Include: argValues(c.include),
		Define:  argValues(c.define),
		Bind:    c.bind.Value(),
	}
	if err := clib.Run(ctx, cfg); err != nil {
		return err
	}
	if c.build.Value() == "" {
		return nil
	}
	dir := filepath.Dir(cfg.File)
	if out := cfg.Out; out != "" {
		if strings.HasSuffix(out, ".go") {
			dir = filepath.Dir(out)
		} else {
			dir = out
		}
	}
	return clib.Build(ctx, dir, c.build.Value())
}
