// Package prelude writes prelude.go files that register child root.go packages.
package prelude

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

type root struct {
	dir, pkg, imp string
}

// Run writes prelude.go under dir for each folder that has child root.go
// files. Each generated init calls Registry.FromGetter on the child's GetCommand.
func Run(ctx context.Context, dir string) error {
	if dir == "" {
		return errDirRequired
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	imp, err := importPath(dir)
	if err != nil {
		return err
	}
	pkg, err := packageOfDir(dir)
	if err != nil {
		return err
	}
	return walk(ctx, root{dir: dir, pkg: pkg, imp: imp})
}

func walk(ctx context.Context, r root) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kids, err := children(r.dir, r.imp)
	if err != nil {
		return err
	}
	if len(kids) > 0 {
		if err := write(r.dir, r.pkg, kids); err != nil {
			return err
		}
	}
	for _, kid := range kids {
		if err := walk(ctx, kid); err != nil {
			return err
		}
	}
	return nil
}

func children(dir, imp string) ([]root, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []root
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		file := filepath.Join(dir, name, "root.go")
		st, err := os.Stat(file)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		if st.IsDir() {
			continue
		}
		pkg, err := packageOfFile(file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		out = append(out, root{
			dir: filepath.Join(dir, name),
			pkg: pkg,
			imp: path.Join(imp, name),
		})
	}
	slices.SortFunc(out, func(a, b root) int {
		return cmp.Compare(a.imp, b.imp)
	})
	return out, nil
}

func packageOfDir(dir string) (string, error) {
	rootFile := filepath.Join(dir, "root.go")
	if st, err := os.Stat(rootFile); err == nil && !st.IsDir() {
		return packageOfFile(rootFile)
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "prelude.go" {
			continue
		}
		pkg, err := packageOfFile(filepath.Join(dir, name))
		if err == nil {
			return pkg, nil
		}
	}
	return "", fmt.Errorf("%w: %s", errNoPackage, dir)
}

func packageOfFile(file string) (string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	if f.Name == nil || f.Name.Name == "" {
		return "", errNoPackage
	}
	return f.Name.Name, nil
}

func importPath(dir string) (string, error) {
	d := dir
	for {
		b, err := os.ReadFile(filepath.Join(d, "go.mod"))
		if err == nil {
			mod, err := moduleLine(b)
			if err != nil {
				return "", err
			}
			rel, err := filepath.Rel(d, dir)
			if err != nil {
				return "", err
			}
			if rel == "." {
				return mod, nil
			}
			return path.Join(mod, filepath.ToSlash(rel)), nil
		}
		parent := filepath.Dir(d)
		if parent == d {
			return "", fmt.Errorf("%w: %s", errNoGoMod, dir)
		}
		d = parent
	}
}

func moduleLine(b []byte) (string, error) {
	sc := bufio.NewScanner(bytes.NewReader(b))
	for sc.Scan() {
		if line, ok := strings.CutPrefix(sc.Text(), "module "); ok {
			return strings.TrimSpace(line), nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return "", errNoModule
}
