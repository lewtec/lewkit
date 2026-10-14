// Package c reads a C header and returns the Go that represents it.
//
// Go parses declarations with x/generate/header and renders them with
// that package's writer. The result is a Go package the compiler accepts
// as a [github.com/lewtec/lewkit/x/toolchain/go.Unit]. Function bodies are
// left out. The symbols are the ones Go calls, through native.Bind.
package c

import (
	"context"
	"errors"
	"fmt"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/generate/header"
)

var (
	// ErrFile means the path is empty.
	ErrFile = errs.Wrap(errs.ErrNotFound, "c: no file")
	// ErrName means the file name is not a Go package name.
	ErrName = errors.New("c: package name")
)

// Go reads file and returns the import path and Go source of its
// declarations. The import path is the file name without its extension.
// The caller writes src into the unit's files and compiles it with the
// other packages of the same target.
func Go(ctx context.Context, file string) (importPath, src string, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(file) == "" {
		return "", "", ErrFile
	}
	name, err := packageName(file)
	if err != nil {
		return "", "", err
	}
	unit, err := header.Parse(ctx, file, header.Options{})
	if err != nil {
		return "", "", err
	}
	body, err := header.Write(unit, name, nil)
	if err != nil {
		return "", "", err
	}
	return name, string(body), nil
}

func packageName(file string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	if base == "" || !token.IsIdentifier(base) || token.IsKeyword(base) {
		return "", fmt.Errorf("%w: %s", ErrName, base)
	}
	return base, nil
}
