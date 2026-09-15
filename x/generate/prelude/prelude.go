// Package prelude writes prelude.go files that register child root.go packages.
package prelude

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	stdpath "path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lewtec/lewkit/x/path"
)

type commandRoot struct {
	directory   path.Path
	packageName string
	importPath  string
}

// Run writes prelude.go under directory for each folder that has child
// root.go files. Each generated init calls Registry.FromGetter on the
// child's GetCommand.
func Run(ctx context.Context, directory string) error {
	if directory == "" {
		return errDirRequired
	}
	filesystem, err := path.Open(directory)
	if err != nil {
		return err
	}
	defer filesystem.Close()
	importPath, err := findImportPath(filesystem)
	if err != nil {
		return err
	}
	packageName, err := packageOfDirectory(filesystem, path.New())
	if err != nil {
		return err
	}
	return walk(ctx, filesystem, commandRoot{
		directory:   path.New(),
		packageName: packageName,
		importPath:  importPath,
	})
}

func walk(ctx context.Context, filesystem *path.Root, current commandRoot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	children, err := childrenOf(filesystem, current)
	if err != nil {
		return err
	}
	if len(children) > 0 {
		if err := write(filesystem, current, children); err != nil {
			return err
		}
	}
	for _, child := range children {
		if err := walk(ctx, filesystem, child); err != nil {
			return err
		}
	}
	return nil
}

func childrenOf(filesystem *path.Root, current commandRoot) ([]commandRoot, error) {
	var out []commandRoot
	for child, err := range current.directory.IterDir(filesystem) {
		if err != nil {
			return nil, err
		}
		isDir, err := child.IsDir(filesystem)
		if err != nil {
			return nil, err
		}
		if !isDir {
			continue
		}
		rootGo := child.Join("root.go")
		isFile, err := rootGo.IsFile(filesystem)
		if err != nil {
			return nil, err
		}
		if !isFile {
			continue
		}
		packageName, err := packageOfFile(filesystem, rootGo)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", rootGo, err)
		}
		out = append(out, commandRoot{
			directory:   child,
			packageName: packageName,
			importPath:  stdpath.Join(current.importPath, child.Name()),
		})
	}
	slices.SortFunc(out, func(a, b commandRoot) int {
		return cmp.Compare(a.importPath, b.importPath)
	})
	return out, nil
}

func packageOfDirectory(filesystem fs.FS, directory path.Path) (string, error) {
	rootGo := directory.Join("root.go")
	ok, err := rootGo.IsFile(filesystem)
	if err != nil {
		return "", err
	}
	if ok {
		return packageOfFile(filesystem, rootGo)
	}
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
