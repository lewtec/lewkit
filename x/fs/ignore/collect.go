package ignore

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/git-pkgs/gitignore"
)

// Collect reads ignore rules from fsys and returns an [FS] that hides them.
//
// Rules come only from fsys. Shallower .gitignore files are the parents of
// deeper ones. .git/info/exclude is read when .git is a directory in fsys,
// or when a .git gitdir file points at a directory that stays inside fsys.
// Global core.excludesfile is not loaded.
func Collect(ctx context.Context, fsys fs.FS, cfg Config) (*FS, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if fsys == nil {
		return nil, errNilFS
	}
	info, err := fs.Stat(fsys, ".")
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, notDirectory("collect", ".")
	}

	collected := &FS{base: fsys, matcher: gitignore.New("")}
	collected.addPatterns(skipDirectoryPatterns(cfg.SkipDirs), "")
	if exclude := excludeName(fsys); exclude != "" {
		if err := collected.addFile(fsys, exclude, "", nil); err != nil {
			return nil, err
		}
	}
	skip := skipSet(cfg.SkipDirs)
	if err := collected.addNamed(ctx, fsys, ".gitignore", skip, nil); err != nil {
		return nil, err
	}
	if cfg.Attributes {
		if err := collected.addNamed(ctx, fsys, ".gitattributes", skip, attributePatterns); err != nil {
			return nil, err
		}
	}
	return collected, nil
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

// excludeName is the fs path of info/exclude. A gitdir or commondir that
// leaves fsys is ignored.
func excludeName(fsys fs.FS) string {
	info, err := fs.Stat(fsys, ".git")
	if err != nil {
		return ""
	}
	gitDir := ".git"
	if !info.IsDir() {
		data, err := fs.ReadFile(fsys, ".git")
		if err != nil {
			return ""
		}
		line, _, _ := strings.Cut(string(data), "\n")
		rest, ok := strings.CutPrefix(strings.TrimSpace(line), "gitdir:")
		if !ok {
			return ""
		}
		gitDir = strings.TrimSpace(rest)
		if gitDir == "" || filepath.IsAbs(gitDir) {
			return ""
		}
		gitDir = path.Clean(gitDir)
		if !fs.ValidPath(gitDir) {
			return ""
		}
	}
	if common, err := fs.ReadFile(fsys, path.Join(gitDir, "commondir")); err == nil {
		name := strings.TrimSpace(string(common))
		if name != "" && !filepath.IsAbs(name) {
			joined := path.Clean(path.Join(gitDir, filepath.ToSlash(name)))
			if fs.ValidPath(joined) {
				gitDir = joined
			}
		}
	}
	exclude := path.Join(gitDir, "info", "exclude")
	if !fs.ValidPath(exclude) {
		return ""
	}
	if _, err := fs.Stat(fsys, exclude); err != nil {
		return ""
	}
	return exclude
}

type transform func([]byte) []byte

func (f *FS) addNamed(ctx context.Context, fsys fs.FS, name string, skip map[string]struct{}, rewrite transform) error {
	files, err := namedFiles(ctx, fsys, name, skip)
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		scope := ""
		if file != name {
			scope = strings.TrimSuffix(file, "/"+name)
		}
		if err := f.addFile(fsys, file, scope, rewrite); err != nil {
			return err
		}
	}
	return nil
}

func (f *FS) addFile(fsys fs.FS, slashPath, scope string, rewrite transform) error {
	if slashPath == "" || slashPath == "." {
		return nil
	}
	data, err := fs.ReadFile(fsys, slashPath)
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
	f.addPatterns(data, scope)
	return nil
}

func namedFiles(ctx context.Context, fsys fs.FS, name string, skip map[string]struct{}) ([]string, error) {
	var files []string
	err := fs.WalkDir(fsys, ".", func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			if filePath != "." && (entry.Name() == ".git" || skipHas(skip, entry.Name())) {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Name() == name {
			files = append(files, filePath)
		}
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
