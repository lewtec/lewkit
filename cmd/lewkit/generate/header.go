package generate

import (
	"context"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/generate/header"
)

type headerCmd struct {
	file    cmd.StringArg   `help:"C header"`
	out     cmd.StringArg   `long:"out" default:"" help:"Go file to write; several ABIs get one file each"`
	pkg     cmd.StringArg   `long:"package" default:"" help:"Go package name"`
	include []cmd.StringArg `long:"include" short:"I" help:"include directory"`
	define  []cmd.StringArg `long:"define" short:"D" help:"macro NAME or NAME=value"`
	lib     []cmd.StringArg `long:"lib" help:"soname passed to Open"`
	abi     cmd.StringArg   `long:"abi" default:"" help:"all by default; host, sysv64, win64, darwinarm64, 386, win386, arm, or goos/goarch"`
	bind    cmd.StringArg   `long:"bind" default:"" help:"struct type holding the C functions; Symbols when empty"`
	soft    cmd.Flag        `long:"soft" help:"leave missing symbols nil"`
}

func (headerCmd) Description() string {
	return "purego bindings for a C header"
}

func (c *headerCmd) Run(ctx context.Context) error {
	return header.Run(ctx, header.Config{
		File:    c.file.Value(),
		Out:     c.out.Value(),
		Package: c.pkg.Value(),
		ABI:     c.abi.Value(),
		Include: cmd.Values(c.include),
		Define:  cmd.Values(c.define),
		Libs:    cmd.Values(c.lib),
		Bind:    c.bind.Value(),
		Soft:    c.soft.Value(),
	})
}
