package main

import (
	"bytes"
	"context"

	"github.com/dave/jennifer/jen"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
)

func jit(ctx context.Context) (string, error) {
	file := jen.NewFile("main")
	file.Func().Id("Value").Params().String().Block(
		jen.Return(jen.Lit("jit-ok\n")),
	)
	file.Func().Id("main").Params().Block()
	var src bytes.Buffer
	if err := file.Render(&src); err != nil {
		return "", err
	}
	prog, err := gotool.CompileSource(ctx, src.String())
	if err != nil {
		return "", err
	}
	fn, err := gotool.Bind[func() string](prog, "Value")
	if err != nil {
		return "", err
	}
	return fn(), nil
}
