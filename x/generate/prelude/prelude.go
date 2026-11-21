// Package prelude writes a blank-import prelude from root.go files.
package prelude

import (
	"bytes"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	stdpath "path"
	"slices"
	"strings"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/generate"
	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/path/pick"
)

// Run scans directory for root.go files and writes a prelude of blank
// imports. dest is the file to write. An empty dest prints to stdout.
func Run(ctx context.Context, directory, dest string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if directory == "" {
		return errDirRequired
	}
	filesystem, err := path.Open(directory)
	if err != nil {
		return err
	}
	defer filesystem.Close()
	base, err := findImportPath(ctx, filesystem)
	if err != nil {
		return err
	}
	imports, err := rootImports(ctx, filesystem, base)
	if err != nil {
		return err
	}
	packageName := "prelude"
	if dest == "" {
		return write(lewfs.ContextWriter(ctx, os.Stdout), packageName, imports)
	}
	outFS, file, err := openDest(dest)
	if err != nil {
		return err
	}
	defer outFS.Close()
	if name, err := packageOfDirectory(ctx, outFS, file.Parent()); err == nil {
		packageName = name
	}
	var body bytes.Buffer
	if err := write(&body, packageName, imports); err != nil {
		return err
	}
	if parent := file.Parent(); parent.String() != "." {
		if err := parent.MkdirAll(outFS, 0o755); err != nil {
			return err
		}
	}
	return file.WriteFile(outFS, body.Bytes(), 0o644)
}

func openDest(dest string) (*path.Root, path.Path, error) {
	name := path.New(dest)
	if name.IsAbs() {
		root, err := path.Open(name.Parent().String())
		return root, path.New(name.Name()), err
	}
	root, err := path.Open(".")
	return root, name, err
}

func rootImports(ctx context.Context, filesystem *path.Root, base string) ([]string, error) {
	var out []string
	for file, err := range lewfs.Walk(ctx, filesystem, pick.Glob("**/root.go")) {
		if err != nil {
			return nil, err
		}
		directory := file.Name.Parent()
		if directory.String() == "." {
			continue
		}
		out = append(out, stdpath.Join(base, directory.String()))
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

func packageOfDirectory(ctx context.Context, filesystem *path.Root, directory path.Path) (string, error) {
	for child, err := range directory.IterDir(filesystem) {
		if err != nil {
			return "", err
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if child.Suffix() != ".go" || child.Name() == "prelude.go" || strings.HasSuffix(child.Stem(), "_test") {
			continue
		}
		ok, err := child.IsFile(filesystem)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}
		packageName, err := packageOfFile(filesystem, child)
		if err == nil {
			return packageName, nil
		}
	}
	return "", fmt.Errorf("%w: %s", errNoPackage, directory)
}

func packageOfFile(filesystem *path.Root, file path.Path) (string, error) {
	body, err := file.ReadFile(filesystem)
	if err != nil {
		return "", err
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), file.String(), body, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	if parsed.Name == nil || parsed.Name.Name == "" {
		return "", errNoPackage
	}
	return parsed.Name.Name, nil
}

func findImportPath(ctx context.Context, start *path.Root) (string, error) {
	current, err := climbStart(start)
	if err != nil {
		return "", err
	}
	var suffix []string
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		ancestor, err := path.Open(current.String())
		if err != nil {
			return "", err
		}
		goMod := path.New("go.mod")
		ok, err := goMod.Exists(ancestor)
		if err != nil {
			ancestor.Close()
			return "", err
		}
		if ok {
			body, err := goMod.ReadFile(ancestor)
			ancestor.Close()
			if err != nil {
				return "", err
			}
			module, err := generate.ModuleLine(body)
			if err != nil {
				return "", err
			}
			if len(suffix) == 0 {
				return module, nil
			}
			return stdpath.Join(append([]string{module}, suffix...)...), nil
		}
		ancestor.Close()
		parent := current.Parent()
		if parent == current {
			return "", fmt.Errorf("%w: %s", errNoGoMod, start.Name())
		}
		suffix = append([]string{current.Name()}, suffix...)
		current = parent
	}
}

func climbStart(start *path.Root) (path.Path, error) {
	current := path.New(start.Name())
	if current.String() != "." {
		return current, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return path.Path{}, err
	}
	return path.New(wd), nil
}

func write(w io.Writer, packageName string, imports []string) error {
	return jenFile(packageName, imports).Render(w)
}
