// Package prelude writes one blank-import prelude per directory that
// contains a descendant root.go.
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

// Run scans directory for root.go files and writes one prelude per
// directory that contains a descendant root.go. A prelude blank-imports
// each child package that has root.go and each child prelude when that
// child has a nested root.go. dest is the prelude file for directory.
// An empty dest prints that file to stdout and does not write the others.
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
	files, err := preludeFiles(ctx, filesystem, base)
	if err != nil {
		return err
	}
	if dest == "" {
		return write(lewfs.ContextWriter(ctx, os.Stdout), "prelude", topImports(files))
	}
	outFS, top, err := outputPlace(dest)
	if err != nil {
		return err
	}
	defer outFS.Close()
	for _, file := range files {
		target := top
		if file.rel != "" {
			target = path.New(file.rel).Join("prelude", "prelude.go")
		}
		packageName := "prelude"
		if name, err := packageOfDirectory(ctx, outFS, target.Parent()); err == nil {
			packageName = name
		}
		var body bytes.Buffer
		if err := write(&body, packageName, file.imports); err != nil {
			return err
		}
		if parent := target.Parent(); parent.String() != "." {
			if err := parent.MkdirAll(outFS, 0o755); err != nil {
				return err
			}
		}
		if err := target.WriteFile(outFS, body.Bytes(), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func topImports(files []preludeFile) []string {
	for _, file := range files {
		if file.rel == "" {
			return file.imports
		}
	}
	return nil
}

// outputPlace returns the filesystem rooted at the directory that contains
// every prelude, plus the path of the top prelude inside that filesystem.
// A dest whose parent directory is named prelude uses the grandparent as
// the root. Any other dest keeps that file in the root.
func outputPlace(dest string) (*path.Root, path.Path, error) {
	name := path.New(dest)
	if name.Parent().Name() == "prelude" {
		root, err := path.Open(name.Parent().Parent().String())
		return root, path.New("prelude", name.Name()), err
	}
	root, err := path.Open(name.Parent().String())
	return root, path.New(name.Name()), err
}

type preludeFile struct {
	rel     string
	imports []string
}

type dirNode struct {
	hasRoot bool
	kids    map[string]*dirNode
}

func (n *dirNode) child(name string) *dirNode {
	if n.kids == nil {
		n.kids = map[string]*dirNode{}
	}
	kid, ok := n.kids[name]
	if !ok {
		kid = &dirNode{}
		n.kids[name] = kid
	}
	return kid
}

func (n *dirNode) add(dir path.Path) {
	if dir.String() == "." {
		n.hasRoot = true
		return
	}
	parts := strings.Split(dir.String(), "/")
	cur := n
	for i, part := range parts {
		cur = cur.child(part)
		if i == len(parts)-1 {
			cur.hasRoot = true
		}
	}
}

func (n *dirNode) descendantRoot() bool {
	for _, kid := range n.kids {
		if kid.hasRoot || kid.descendantRoot() {
			return true
		}
	}
	return false
}

type preludePlan struct {
	base string
	out  []preludeFile
}

func preludeFiles(ctx context.Context, filesystem *path.Root, base string) ([]preludeFile, error) {
	root := &dirNode{}
	for file, err := range lewfs.Walk(ctx, filesystem, pick.Glob("**/root.go")) {
		if err != nil {
			return nil, err
		}
		root.add(file.Name.Parent())
	}
	plan := preludePlan{base: base}
	plan.add(root, "")
	return plan.out, nil
}

func (p *preludePlan) add(n *dirNode, rel string) {
	if !n.descendantRoot() {
		if rel == "" {
			p.out = append(p.out, preludeFile{})
		}
		return
	}
	names := make([]string, 0, len(n.kids))
	for name := range n.kids {
		names = append(names, name)
	}
	slices.Sort(names)
	imports := make([]string, 0, len(names))
	for _, name := range names {
		kid := n.kids[name]
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		if kid.hasRoot {
			imports = append(imports, stdpath.Join(p.base, childRel))
		}
		if kid.descendantRoot() {
			imports = append(imports, stdpath.Join(p.base, childRel, "prelude"))
		}
	}
	slices.Sort(imports)
	p.out = append(p.out, preludeFile{rel: rel, imports: slices.Compact(imports)})
	for _, name := range names {
		childRel := name
		if rel != "" {
			childRel = rel + "/" + name
		}
		p.add(n.kids[name], childRel)
	}
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
