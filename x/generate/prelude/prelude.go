// Package prelude writes a blank-import prelude from root.go files.
package prelude

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	stdpath "path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lewtec/lewkit/x/path"
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
	base, err := findImportPath(filesystem)
	if err != nil {
		return err
	}
	imports, err := rootImports(filesystem, base)
	if err != nil {
		return err
	}
	packageName := "prelude"
	if dest == "" {
		return write(os.Stdout, packageName, imports)
	}
	parent := filepath.Dir(dest)
	outFS, err := path.Open(parent)
	if err != nil {
		return err
	}
	defer outFS.Close()
	if name, err := packageOfDirectory(outFS, path.New()); err == nil {
		packageName = name
	}
	var body bytes.Buffer
	if err := write(&body, packageName, imports); err != nil {
		return err
	}
	return path.New(filepath.Base(dest)).WriteFile(outFS, body.Bytes(), 0o644)
}

func rootImports(filesystem *path.Root, base string) ([]string, error) {
	var out []string
	for name, err := range path.New().Walk(filesystem) {
		if err != nil {
			return nil, err
		}
		if name.Name() != "root.go" {
			continue
		}
		ok, err := name.IsFile(filesystem)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		directory := name.Parent()
		if directory.String() == "." {
			continue
		}
		out = append(out, stdpath.Join(base, directory.String()))
	}
	slices.Sort(out)
	return slices.Compact(out), nil
}

func packageOfDirectory(filesystem fs.FS, directory path.Path) (string, error) {
	entries, err := directory.ReadDir(filesystem)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "prelude.go" {
			continue
		}
		packageName, err := packageOfFile(filesystem, directory.Join(name))
		if err == nil {
			return packageName, nil
		}
	}
	return "", fmt.Errorf("%w: %s", errNoPackage, directory)
}

func packageOfFile(filesystem fs.FS, file path.Path) (string, error) {
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

func findImportPath(start *path.Root) (string, error) {
	current := start.Name()
	var suffix []string
	for {
		ancestor, err := path.Open(current)
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
			module, err := moduleLine(body)
			if err != nil {
				return "", err
			}
			if len(suffix) == 0 {
				return module, nil
			}
			return stdpath.Join(append([]string{module}, suffix...)...), nil
		}
		ancestor.Close()
		parent, name, err := parentDirectory(current)
		if err != nil {
			return "", err
		}
		if parent == current {
			return "", fmt.Errorf("%w: %s", errNoGoMod, start.Name())
		}
		suffix = append([]string{name}, suffix...)
		current = parent
	}
}

func parentDirectory(directory string) (parent, name string, err error) {
	if directory == "." || !filepath.IsAbs(directory) {
		directory, err = filepath.Abs(directory)
		if err != nil {
			return "", "", err
		}
	}
	return filepath.Dir(directory), filepath.Base(directory), nil
}

func moduleLine(body []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	for scanner.Scan() {
		if line, ok := strings.CutPrefix(scanner.Text(), "module "); ok {
			return strings.TrimSpace(line), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errNoModule
}

func write(w io.Writer, packageName string, imports []string) error {
	file := jenFile(packageName, imports)
	return file.Render(w)
}
