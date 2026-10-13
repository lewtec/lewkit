// Package generate is the lewkit generate command.
// cmd/lewkit stores Cmd and does not declare the subcommands.
package generate

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/db/generate"
	"github.com/lewtec/lewkit/x/generate/clib"
	"github.com/lewtec/lewkit/x/generate/header"
	"github.com/lewtec/lewkit/x/generate/prelude"
	"github.com/lewtec/lewkit/x/generate/protobuf"
	"github.com/lewtec/lewkit/x/generate/shader"
)

// Cmd is lewkit generate.
type Cmd struct {
	db       *dbCmd
	prelude  *preludeCmd
	protobuf *protobufCmd
	shader   *shaderCmd
	header   *headerCmd
	clib     *clibCmd
}

// Description is the command summary.
func (Cmd) Description() string {
	return "generate code"
}

type dbCmd struct {
	dir cmd.WorkDirArg `help:"directory with sqlite/ and postgres/"`
}

func (dbCmd) Description() string {
	return "generate sqlc packages and a shared Queries interface"
}

func (c *dbCmd) Run(ctx context.Context) error {
	return generate.Run(ctx, c.dir.Value())
}

type preludeCmd struct{}

func (preludeCmd) Description() string {
	return "blank-import preludes for every root.go in the working directory"
}

func (preludeCmd) Run(ctx context.Context) error {
	return prelude.Run(ctx, ".", "prelude/prelude.go")
}

type protobufCmd struct {
	proto     cmd.StringArg `help:".proto file"`
	goPackage cmd.StringArg `long:"package" default:"" help:"Go import path when the proto has no go_package"`
}

func (protobufCmd) Description() string {
	return "generate Go from a .proto file"
}

func (c *protobufCmd) Run(ctx context.Context) error {
	return protobuf.Run(ctx, c.proto.Value(), c.goPackage.Value())
}

type shaderCmd struct {
	dir cmd.WorkDirArg `help:"directory to scan for name.<stage>.glsl"`
}

func (shaderCmd) Description() string {
	return "SPIR-V registry for GLSL shaders under DIR"
}

func (c *shaderCmd) Run(ctx context.Context) error {
	return shader.Run(ctx, c.dir.Value())
}

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
		Include: argValues(c.include),
		Define:  argValues(c.define),
		Libs:    argValues(c.lib),
		Bind:    c.bind.Value(),
		Soft:    c.soft.Value(),
	})
}

type clibCmd struct {
	file    cmd.StringArg   `help:"C header"`
	out     cmd.StringArg   `long:"out" default:"" help:"Go file to write"`
	pkg     cmd.StringArg   `long:"package" default:"" help:"Go package name"`
	include []cmd.StringArg `long:"include" short:"I" help:"include directory"`
	define  []cmd.StringArg `long:"define" short:"D" help:"macro NAME or NAME=value"`
	bind    cmd.StringArg   `long:"bind" default:"" help:"struct holding the C functions; API when empty"`
	abi     cmd.StringArg   `long:"abi" default:"" help:"linux and android arches, windows, darwin, host, an ABI id, or goos/goarch"`
	build   cmd.StringArg   `long:"build" default:"" help:"also link this shared library without cgo"`
	target  cmd.StringArg   `long:"target" default:"" help:"GOOS/GOARCH for --build; this process when empty"`
}

func (clibCmd) Description() string {
	return "C shared library from a header, built without cgo"
}

func (c *clibCmd) Run(ctx context.Context) error {
	if c.target.Value() != "" && c.build.Value() == "" {
		return fmt.Errorf("clib: --target requires --build")
	}
	cfg := clib.Config{
		File:    c.file.Value(),
		Out:     c.out.Value(),
		Package: c.pkg.Value(),
		Include: argValues(c.include),
		Define:  argValues(c.define),
		Bind:    c.bind.Value(),
		ABI:     c.abi.Value(),
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
	goos, goarch := runtime.GOOS, runtime.GOARCH
	if spec := c.target.Value(); spec != "" {
		goos, goarch, _ = strings.Cut(spec, "/")
	}
	return clib.BuildFor(ctx, dir, c.build.Value(), goos, goarch)
}

func argValues(args []cmd.StringArg) []string {
	out := make([]string, len(args))
	for i := range args {
		out[i] = args[i].Value()
	}
	return out
}
