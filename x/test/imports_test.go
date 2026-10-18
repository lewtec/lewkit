package test

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestModuleImports requires every import under this module to name a
// package in the tree. go mod tidy reads Go files behind build tags, so
// a copied toolchain file can fail a consumer that never builds that GOOS.
func TestModuleImports(t *testing.T) {
	root := moduleRoot(t)
	mod := modulePath(root)
	require.NotEmpty(t, mod)

	checked := map[string]struct{}{}
	var problems []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if entry.IsDir() {
			if skipImportDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if buildIgnored(src) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, rel, src, parser.ImportsOnly|parser.SkipObjectResolution)
		if err != nil {
			problems = append(problems, rel+": "+err.Error())
			return nil
		}
		for _, spec := range file.Imports {
			imp, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				problems = append(problems, rel+": "+err.Error())
				continue
			}
			if imp != mod && !strings.HasPrefix(imp, mod+"/") {
				continue
			}
			if _, ok := checked[imp]; ok {
				continue
			}
			checked[imp] = struct{}{}
			if modulePackageMissing(root, mod, imp) {
				problems = append(problems, rel+": missing package "+imp)
			}
		}
		return nil
	})
	require.NoError(t, err)
	require.Empty(t, problems)
}

func skipImportDir(name string) bool {
	switch name {
	case ".git", "vendor", "testdata", "_rewrite", "_upstream":
		return true
	default:
		return false
	}
}

func buildIgnored(src []byte) bool {
	for _, line := range bytes.Split(src, []byte("\n")) {
		trim := bytes.TrimSpace(line)
		if len(trim) == 0 {
			continue
		}
		if !bytes.HasPrefix(trim, []byte("//")) {
			return false
		}
		if bytes.HasPrefix(trim, []byte("//go:build")) || bytes.HasPrefix(trim, []byte("// +build")) {
			for _, field := range bytes.Fields(trim) {
				if bytes.Equal(field, []byte("ignore")) {
					return true
				}
			}
		}
	}
	return false
}

func modulePackageMissing(root, mod, imp string) bool {
	rel := "."
	if imp != mod {
		rel = strings.TrimPrefix(imp, mod+"/")
	}
	dir := filepath.Join(root, filepath.FromSlash(rel))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || buildIgnored(body) {
			continue
		}
		return false
	}
	return true
}
