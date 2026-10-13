package gotool

import (
	"context"
	"os"
	"path/filepath"
)

// SharedLibrary compiles src into a linux/amd64 shared object at out.
// src is one package main. The compiler reads it under the name
// _cgo_export.go, which is what makes //go:cgo_export_dynamic take effect.
// The temporary source is removed before return. out is the library.
func SharedLibrary(ctx context.Context, out, src string) error {
	if ctx == nil {
		return ErrNilContext
	}
	dir, err := os.MkdirTemp("", "lewkit-share-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "_cgo_export.go")
	if err := os.WriteFile(file, []byte(src), 0o644); err != nil {
		return err
	}
	compileCfg, linkDeps, err := runtimeDeps(dir)
	if err != nil {
		return err
	}
	archive := filepath.Join(dir, "main.a")
	if err := Compile(ctx, "main", archive, compileCfg, "", "go1.27", true, []string{file}); err != nil {
		return err
	}
	linkCfg := filepath.Join(dir, "link.importcfg")
	body := "# import config\npackagefile main=" + archive + "\n" + linkDeps
	if err := os.WriteFile(linkCfg, []byte(body), 0o644); err != nil {
		return err
	}
	if err := Link(ctx, out, linkCfg, "pie", archive); err != nil {
		return err
	}
	return MakeShared(out)
}
