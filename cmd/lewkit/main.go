package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/cmd/lewkit/decomp"
	"github.com/lewtec/lewkit/report/sentry"
	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/db/generate"
	"github.com/lewtec/lewkit/x/entry"
	"github.com/lewtec/lewkit/x/generate/header"
	"github.com/lewtec/lewkit/x/generate/prelude"
	"github.com/lewtec/lewkit/x/generate/protobuf"
	"github.com/lewtec/lewkit/x/generate/shader"
	"github.com/lewtec/lewkit/x/taskgroup"

	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
	_ "github.com/lewtec/lewkit/x/tool/conda"
)

func main() {
	if done, err := toolchainChild(); done {
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	app, err := cmd.Parse[cmd.App[root]](normalizeReleaseGoFlags(os.Args[1:])...)
	if err != nil {
		entry.Main(context.Background(), func(context.Context) error { return err })
		return
	}
	parent := taskgroup.WithLimits(context.Background(), app.Args.Apply(taskgroup.DefaultLimits()))
	entry.MainFrom(parent, app.Run)
}

type root struct {
	taskgroup.Arg `flatten:"" ctx:"taskgroup"`
	sentry        sentry.Arg `long:"sentry-dsn" env:"SENTRY_DSN" help:"Sentry DSN" default:"https://26fa6b84edbc334b77bf7f6e1d7d69bc@o4508616651505664.ingest.us.sentry.io/4512090764607488"`
	generate      *generateCmd
	release       *releaseCmd
	exec          *execCmd
	disasm        *disasmCmd
	decomp        *decomp.Cmd
	doctor        *doctorCmd
	workflow      *workflowCmd
	completion    *completionCmd
}

func (r *root) Setup() error {
	return r.sentry.Setup()
}

type generateCmd struct {
	db       *dbCmd
	prelude  *preludeCmd
	protobuf *protobufCmd
	shader   *shaderCmd
	header   *headerCmd
	clib     *clibCmd
}

func (generateCmd) Description() string {
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

func argValues(args []cmd.StringArg) []string {
	out := make([]string, len(args))
	for i := range args {
		out[i] = args[i].Value()
	}
	return out
}

type completionCmd struct{}

func (completionCmd) Description() string {
	return "print the bash complete -C line"
}

func (*completionCmd) Run(context.Context) error {
	_, err := fmt.Println(cmd.BashCompleteLine(filepath.Base(os.Args[0])))
	return err
}

func (root) Description() string {
	return "Well planned primitives to be used in other projects."
}
