// Package generate writes sqlc packages and a shared Queries interface.
package generate

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func pkgName(dir string) string {
	base := filepath.Base(dir)
	r, _ := utf8.DecodeRuneInString(base)
	if unicode.IsLetter(r) || r == '_' {
		ok := true
		for _, c := range base {
			if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
				ok = false
				break
			}
		}
		if ok {
			return base
		}
	}
	return "db"
}

// Run scans dir for sqlite/ and postgres/, checks query names match,
// runs sqlc, and writes a shared Queries interface plus DBArg.
func Run(ctx context.Context, dir string) error {
	if dir == "" {
		return errDirRequired
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	engines, err := scan(dir)
	if err != nil {
		return err
	}
	if err := compareQueries(engines); err != nil {
		return err
	}
	if err := writeSQLC(dir, engines); err != nil {
		return err
	}
	if err := runSQLC(dir); err != nil {
		return err
	}
	methods, types, err := loadGenerated(dir, engines)
	if err != nil {
		return err
	}
	if err := compareMethods(methods); err != nil {
		return err
	}
	if err := compareStructs(types, methods); err != nil {
		return err
	}
	pkg := pkgName(dir)
	imp, err := importPath(dir)
	if err != nil {
		return err
	}
	return write(out{
		dir:     dir,
		pkg:     pkg,
		imp:     imp,
		engines: engines,
		methods: firstMethods(methods),
		structs: firstStructs(types),
	})
}

func firstMethods(m map[string][]method) []method {
	return m[firstKey(m)]
}

func firstStructs(m map[string][]structType) []structType {
	return m[firstKey(m)]
}

func firstKey[T any](m map[string]T) string {
	var names []string
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names[0]
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
			return filepath.ToSlash(filepath.Join(mod, rel)), nil
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
		if path, ok := strings.CutPrefix(sc.Text(), "module "); ok {
			return strings.TrimSpace(path), nil
		}
	}
	return "", errNoModule
}
