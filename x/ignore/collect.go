package ignore

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/git-pkgs/gitignore"
	lewpath "github.com/lewtec/lewkit/x/path"
)

// Collect reads ignore rules for the tree at root.
//
// root is a directory. When it sits inside a git checkout, Root is the
// checkout toplevel: .git/info/exclude and parent .gitignore files from
// that toplevel apply, then .gitignore files nested under root. Without a
// .git directory, only root and its descendants are read. Global
// core.excludesfile is not loaded.
func Collect(ctx context.Context, root string, cfg Config) (*Set, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if root == "" {
		return nil, &fs.PathError{Op: "collect", Path: root, Err: lewpath.ErrEmptyPath}
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	absolute = filepath.Clean(absolute)
	info, err := os.Stat(absolute)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &fs.PathError{Op: "collect", Path: absolute, Err: lewpath.ErrNotDir}
	}

	toplevel := findToplevel(absolute)
	tree, err := lewpath.Open(toplevel)
	if err != nil {
		return nil, err
	}
	defer tree.Close()

	scope, ok := slashRelative(toplevel, absolute)
	if !ok {
		scope = ""
	}
	ancestors := ancestorDirs(toplevel, absolute)
	set := &Set{Root: toplevel, matcher: gitignore.New("")}
	set.addPatterns(skipDirectoryPatterns(cfg.SkipDirs), scope)

	if exclude, ok := excludeFile(toplevel); ok {
		data, err := os.ReadFile(exclude)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		set.addPatterns(data, "")
	}

	if err := set.addAncestorFiles(ctx, tree, ancestors, ".gitignore", nil); err != nil {
		return nil, err
	}
	if err := set.addNestedFiles(ctx, tree, scope, ".gitignore", cfg.SkipDirs, nil); err != nil {
		return nil, err
	}
	if cfg.Attributes {
		if err := set.addAncestorFiles(ctx, tree, ancestors, ".gitattributes", attributePatterns); err != nil {
			return nil, err
		}
		if err := set.addNestedFiles(ctx, tree, scope, ".gitattributes", cfg.SkipDirs, attributePatterns); err != nil {
			return nil, err
		}
	}
	return set, nil
}

func skipDirectoryPatterns(names []string) []byte {
	seen := map[string]struct{}{}
	var out strings.Builder
	for _, name := range names {
		if name == "" || strings.ContainsAny(name, `/\`) {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out.WriteString(name)
		out.WriteString("/\n")
	}
	return []byte(out.String())
}

func findToplevel(dir string) string {
	current := dir
	for {
		gitPath := filepath.Join(current, ".git")
		info, err := os.Lstat(gitPath)
		if err == nil && (info.IsDir() || info.Mode().IsRegular()) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return dir
		}
		current = parent
	}
}

func ancestorDirs(toplevel, dir string) []string {
	var dirs []string
	for current := dir; ; current = filepath.Dir(current) {
		dirs = append(dirs, current)
		if current == toplevel {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	slices.Reverse(dirs)
	return dirs
}

// excludeFile is $GIT_DIR/info/exclude. A worktree .git file is followed
// to the common git directory, which is where that file lives.
func excludeFile(toplevel string) (string, bool) {
	gitDir, ok := resolveGitDir(toplevel)
	if !ok {
		return "", false
	}
	if common, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil {
		name := strings.TrimSpace(string(common))
		if name != "" {
			if !filepath.IsAbs(name) {
				name = filepath.Join(gitDir, name)
			}
			gitDir = filepath.Clean(name)
		}
	}
	return filepath.Join(gitDir, "info", "exclude"), true
}

func resolveGitDir(toplevel string) (string, bool) {
	gitPath := filepath.Join(toplevel, ".git")
	info, err := os.Lstat(gitPath)
	if err != nil {
		return "", false
	}
	if info.IsDir() {
		return gitPath, true
	}
	if !info.Mode().IsRegular() {
		return "", false
	}
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return "", false
	}
	line := strings.TrimSpace(string(data))
	rest, ok := strings.CutPrefix(line, "gitdir:")
	if !ok {
		return "", false
	}
	gitDir := strings.TrimSpace(rest)
	if gitDir == "" {
		return "", false
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(toplevel, gitDir)
	}
	return filepath.Clean(gitDir), true
}

type transform func([]byte) []byte

func (s *Set) addAncestorFiles(ctx context.Context, tree fs.FS, dirs []string, name string, rewrite transform) error {
	toplevel := s.Root
	for _, dir := range dirs {
		if err := ctx.Err(); err != nil {
			return err
		}
		scope, ok := slashRelative(toplevel, dir)
		if !ok {
			continue
		}
		if err := s.addTreeFile(tree, joinSlash(scope, name), scope, rewrite); err != nil {
			return err
		}
	}
	return nil
}

func (s *Set) addNestedFiles(ctx context.Context, tree fs.FS, scope, name string, skipDirs []string, rewrite transform) error {
	files, err := nestedFiles(ctx, tree, scope, name, skipSet(skipDirs))
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		dir := strings.TrimSuffix(file, "/"+name)
		if dir == file {
			dir = ""
		}
		if err := s.addTreeFile(tree, file, dir, rewrite); err != nil {
			return err
		}
	}
	return nil
}

func (s *Set) addTreeFile(tree fs.FS, slashPath, scope string, rewrite transform) error {
	if slashPath == "" || slashPath == "." {
		return nil
	}
	data, err := fs.ReadFile(tree, slashPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}
	data = bytes.TrimPrefix(data, []byte("\uFEFF"))
	if rewrite != nil {
		data = rewrite(data)
	}
	s.addPatterns(data, scope)
	return nil
}

func nestedFiles(ctx context.Context, tree fs.FS, scope, name string, skip map[string]struct{}) ([]string, error) {
	start := scope
	if start == "" {
		start = "."
	}
	var files []string
	err := fs.WalkDir(tree, start, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if path != start && (entry.Name() == ".git" || skipHas(skip, entry.Name())) {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Name() != name || ownFile(start, path, name) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.SortStableFunc(files, func(a, b string) int {
		depthA := strings.Count(a, "/")
		depthB := strings.Count(b, "/")
		if depthA != depthB {
			return depthA - depthB
		}
		return strings.Compare(a, b)
	})
	return files, nil
}

func ownFile(start, path, name string) bool {
	if start == "." {
		return path == name
	}
	return path == start+"/"+name
}

func skipSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name != "" {
			set[name] = struct{}{}
		}
	}
	return set
}

func skipHas(skip map[string]struct{}, name string) bool {
	_, ok := skip[name]
	return ok
}

func slashRelative(root, path string) (string, bool) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return "", false
	}
	if relative == "." {
		return "", true
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(relative), true
}
