// Package shader writes a SPIR-V registry for GLSL files in a tree.
// A shader is name.<stage>.glsl. The second-to-last extension is the
// stage: vertex, fragment, or compute. Each file is compiled with the
// embedded glslang and recorded in spirv_gen.go beside the Go package
// that owns it.
package shader

import (
	"context"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/lewtec/lewkit/x/generate"
)

const generatedName = "spirv_gen.go"

var errNoPackage = errors.New("no Go package for shader")

type shaderFile struct {
	rel   string
	abs   string
	stage glsl.Stage
}

// Run scans directory for Vulkan GLSL and writes one spirv_gen.go per
// Go package that owns a shader. The file registers the SHA-256 of the
// stage and the source. An empty directory is a no-op.
func Run(ctx context.Context, directory string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if directory == "" {
		return generate.ErrDirRequired
	}
	root, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	root = filepath.Clean(root)
	grouped, err := findShaders(ctx, root)
	if err != nil {
		return err
	}
	dirs := make([]string, 0, len(grouped))
	for dir := range grouped {
		dirs = append(dirs, dir)
	}
	slices.Sort(dirs)
	for _, dir := range dirs {
		if err := ctx.Err(); err != nil {
			return err
		}
		files := grouped[dir]
		slices.SortFunc(files, func(a, b shaderFile) int {
			return strings.Compare(a.rel, b.rel)
		})
		if err := writePackage(ctx, dir, files); err != nil {
			return err
		}
	}
	return nil
}

func findShaders(ctx context.Context, root string) (map[string][]shaderFile, error) {
	grouped := map[string][]shaderFile{}
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		base := entry.Name()
		if entry.IsDir() {
			if filepath.Clean(name) != root && skipDir(base) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(base, ".") {
			return nil
		}
		stage, ok := stageOf(base)
		if !ok {
			return nil
		}
		pkgDir, err := packageDir(name, root)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(pkgDir, name)
		if err != nil {
			return err
		}
		grouped[pkgDir] = append(grouped[pkgDir], shaderFile{
			rel:   filepath.ToSlash(rel),
			abs:   name,
			stage: stage,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return grouped, nil
}

func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor"
}

func stageOf(name string) (glsl.Stage, bool) {
	_, stage, ok := splitShader(name)
	return stage, ok
}

// splitShader reads a name.<stage>.glsl path.
// The second-to-last extension is the stage. stem is the path without that suffix.
func splitShader(name string) (stem string, stage glsl.Stage, ok bool) {
	if filepath.Ext(name) != ".glsl" {
		return "", 0, false
	}
	without := strings.TrimSuffix(name, ".glsl")
	switch filepath.Ext(without) {
	case ".vertex":
		stage = glsl.StageVertex
	case ".fragment":
		stage = glsl.StageFragment
	case ".compute":
		stage = glsl.StageCompute
	default:
		return "", 0, false
	}
	return strings.TrimSuffix(without, filepath.Ext(without)), stage, true
}

func packageDir(file, root string) (string, error) {
	dir := filepath.Dir(file)
	for {
		_, err := goPackage(dir)
		if err == nil {
			return dir, nil
		}
		if !errors.Is(err, errNoPackage) {
			return "", err
		}
		if filepath.Clean(dir) == root {
			return "", fmt.Errorf("%w: %s", errNoPackage, file)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("%w: %s", errNoPackage, file)
		}
		dir = parent
	}
}

func goPackage(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	fallback := ""
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		pkg, err := packageClause(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if name == generatedName {
			fallback = pkg
			continue
		}
		return pkg, nil
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", errNoPackage
}

func packageClause(file string) (string, error) {
	parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.PackageClauseOnly)
	if err != nil {
		return "", err
	}
	if parsed.Name == nil || parsed.Name.Name == "" {
		return "", errNoPackage
	}
	return parsed.Name.Name, nil
}
