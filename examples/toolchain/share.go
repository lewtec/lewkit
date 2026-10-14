package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dave/jennifer/jen"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/lewtec/lewkit/x/toolchain/purego"
)

var (
	ErrShare = errors.New("")
)

// hostPingSrc renders the library. HostPing is the C symbol.
// The function is nosplit so the FFI call reads the returned byte.
func hostPingSrc() (string, error) {
	file := jen.NewFile("main")
	file.Func().Id("main").Params().Block()
	file.Comment("//go:cgo_export_dynamic main.HostPing HostPing")
	file.Comment("//go:nosplit")
	file.Func().Id("HostPing").Params().Byte().Block(
		jen.Return(jen.Lit(7)),
	)
	var src bytes.Buffer
	if err := file.Render(&src); err != nil {
		return "", err
	}
	return src.String(), nil
}

func share(ctx context.Context) (string, error) {
	src, err := hostPingSrc()
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "lewkit-share-example-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(dir)
	so := filepath.Join(dir, "lib.so")
	if err := gotool.SharedLibrary(ctx, so, src); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, os.Args[0])
	cmd.Env = append(os.Environ(), "LEWKIT_SHARE_SO="+so)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w%w\n%s", ErrShare, err, stderr.String())
	}
	return stdout.String(), nil
}

func callShared(path string) error {
	handle, err := purego.Open(path)
	if err != nil {
		return err
	}
	var ping func() byte
	purego.Register(&ping, handle, "HostPing")
	fmt.Println(ping())
	return nil
}
