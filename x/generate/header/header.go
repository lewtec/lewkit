// Package header writes purego bindings for a C header.
// The tree-sitter C grammar supplies the syntax. Layout follows the
// selected ABI, and the generated file dlopens through x/ffi/native.
package header

import (
	"context"
	"fmt"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Config is the lewkit generate header invocation.
type Config struct {
	File    string
	Out     string
	Package string
	ABI     string
	Include []string
	Define  []string
	Libs    []string
}

// Run parses File and writes a Go binding. Out defaults to
// <stem>_purego.go beside the header. Package is inferred from that
// directory when empty.
func Run(ctx context.Context, cfg Config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cfg.File == "" {
		return errHeader
	}
	target, err := ParseABI(cfg.ABI)
	if err != nil {
		return err
	}
	unit, err := Parse(ctx, cfg.File, Options{
		Include: cfg.Include,
		Define:  cfg.Define,
		Target:  target,
	})
	if err != nil {
		return err
	}
	out := cfg.Out
	if out == "" {
		stem := strings.TrimSuffix(filepath.Base(cfg.File), filepath.Ext(cfg.File))
		out = filepath.Join(filepath.Dir(cfg.File), stem+"_purego.go")
	}
	pkg := cfg.Package
	if pkg == "" {
		pkg = inferPackage(filepath.Dir(out))
	}
	if !token.IsIdentifier(pkg) || token.IsKeyword(pkg) {
		return fmt.Errorf("%w: %s", errPackage, pkg)
	}
	buf, err := Write(unit, pkg, cfg.Libs)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	tmp := out + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, out)
}

func inferPackage(dir string) string {
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			if strings.HasSuffix(name, "_purego.go") {
				continue
			}
			pkg, err := packageClause(filepath.Join(dir, name))
			if err == nil && pkg != "" {
				return pkg
			}
		}
	}
	base := filepath.Base(dir)
	if base != "" && base != "." && token.IsIdentifier(base) && !token.IsKeyword(base) {
		r, _ := utf8.DecodeRuneInString(base)
		if unicode.IsLetter(r) {
			return base
		}
	}
	return "bind"
}

func packageClause(file string) (string, error) {
	parsed, err := goparser.ParseFile(token.NewFileSet(), file, nil, goparser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	if parsed.Name == nil {
		return "", errPackage
	}
	return parsed.Name.Name, nil
}
