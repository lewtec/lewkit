package gotool

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/toolchain/go/src/compile/amd64"
	"github.com/lewtec/lewkit/x/toolchain/go/src/compile/arm64"
	"github.com/lewtec/lewkit/x/toolchain/go/src/compile/base"
	"github.com/lewtec/lewkit/x/toolchain/go/src/compile/gc"
	"github.com/lewtec/lewkit/x/toolchain/go/src/compile/ssagen"
	"github.com/lewtec/lewkit/x/toolchain/go/src/std/buildcfg"
)

// Compile compiles files as one package. shared sets -shared, which a PIE
// object needs. lang is the -lang value, such as go1.27. embedcfg is the
// //go:embed configuration, or empty. The compile runs in its own process.
func Compile(ctx context.Context, pkgpath, out, importcfg, embedcfg, lang string, shared bool, files []string) error {
	return spawn(ctx, "compile", childRequest{
		ImportPath: pkgpath,
		Out:        out,
		ImportCfg:  importcfg,
		EmbedCfg:   embedcfg,
		Lang:       lang,
		Shared:     shared,
		Files:      files,
	})
}

// compileHere is the compile inside the process spawn started.
// One process compiles one package, then exits.
func compileHere(pkgpath, out, importcfg, embedcfg, lang string, shared bool, trimpath string, files []string) (err error) {
	if pkgpath == "" {
		pkgpath = "main"
	}
	if lang == "" {
		lang = "go1.27"
	}
	archInit, errArch := compileArch(buildcfg.GOARCH)
	if errArch != nil {
		return errArch
	}
	args := []string{
		"-o", out,
		"-p", pkgpath,
		"-lang=" + lang,
		"-complete",
		"-nolocalimports",
		"-importcfg", importcfg,
		"-pack",
		"-c=1",
	}
	if trimpath != "" {
		args = append(args, "-trimpath", trimpath)
	}
	if embedcfg != "" {
		args = append(args, "-embedcfg", embedcfg)
	}
	if shared {
		args = append(args, "-shared")
	}
	args = append(args, files...)

	var buf bytes.Buffer
	base.SetDiag(&buf)
	defer base.SetDiag(nil)
	defer func() {
		msg := buf.String()
		rec := recover()
		if rec == nil {
			if _, statErr := os.Stat(out); statErr != nil {
				err = fmt.Errorf("compile wrote no object: %w\n%s", statErr, msg)
			}
			return
		}
		sig, ok := rec.(base.ExitSignal)
		if !ok {
			panic(rec)
		}
		if sig != 0 {
			err = fmt.Errorf("compile exited %d\n%s", int(sig), msg)
			return
		}
		if _, statErr := os.Stat(out); statErr != nil {
			err = fmt.Errorf("compile wrote no object: %w\n%s", statErr, msg)
		}
	}()

	buildcfg.Check()
	gc.MainArgs(archInit, args)
	return nil
}

func compileArch(goarch string) (func(*ssagen.ArchInfo), error) {
	switch goarch {
	case "amd64":
		return amd64.Init, nil
	case "arm64":
		return arm64.Init, nil
	default:
		return nil, fmt.Errorf("compile supports GOARCH=amd64 or arm64, have %s", goarch)
	}
}
