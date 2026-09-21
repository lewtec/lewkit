package compose

import (
	"io/fs"
	"strings"

	"github.com/lewtec/lewkit/x/path"
)

const (
	dotDirectorySuffix = ".d.tmpl"
	templateSuffix     = ".tmpl"
	refSlotKey         = "src"
)

// Squash lowers a source filesystem into a tree.
// A directory whose name ends in .d.tmpl becomes one [TypeLines] file.
// The destination path is that directory with the suffix removed.
// Each direct child file is a text slot keyed by the child name.
// A child directory is [ErrPath]. An empty directory adds no file.
// Any other file becomes a [TypeRef] slot with key src.
// A file whose name ends in .tmpl is skipped. A non-directory whose name
// ends in .d.tmpl is [ErrPath].
func Squash(fsys fs.FS) (*Tree, error) {
	if fsys == nil {
		return nil, ErrNilFilesystem
	}
	tree := New()
	err := fs.WalkDir(fsys, ".", func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == "." {
			return nil
		}
		if strings.HasSuffix(entry.Name(), dotDirectorySuffix) {
			if !entry.IsDir() {
				return pathError("squash", name, ErrPath)
			}
			if err := addDotDirectory(tree, fsys, path.New(name)); err != nil {
				return err
			}
			return fs.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(entry.Name(), templateSuffix) {
			return nil
		}
		return addRef(tree, name, entry)
	})
	if err != nil {
		return nil, err
	}
	return tree, nil
}

func addDotDirectory(tree *Tree, fsys fs.FS, directory path.Path) error {
	destination, err := linesPath(directory)
	if err != nil {
		return pathError("squash", directory.String(), err)
	}
	entries, err := fs.ReadDir(fsys, directory.String())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		childName := directory.Join(entry.Name()).String()
		if entry.IsDir() {
			return pathError("squash", childName, ErrPath)
		}
		if strings.HasSuffix(entry.Name(), templateSuffix) {
			continue
		}
		body, err := fs.ReadFile(fsys, childName)
		if err != nil {
			return err
		}
		err = tree.Add(destination, File{
			Type:   TypeLines,
			Values: map[string]Slot{entry.Name(): Text(string(body))},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func linesPath(directory path.Path) (path.Path, error) {
	base := directory.Name()
	if !strings.HasSuffix(base, dotDirectorySuffix) {
		return path.Path{}, ErrPath
	}
	trimmed := strings.TrimSuffix(base, dotDirectorySuffix)
	if trimmed == "" || trimmed == "." || trimmed == ".." {
		return path.Path{}, ErrPath
	}
	parent := directory.Parent()
	if parent.String() == "." {
		return path.New(trimmed), nil
	}
	return parent.Join(trimmed), nil
}

func addRef(tree *Tree, name string, entry fs.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}
	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}
	return tree.Add(path.New(name), File{
		Type:   TypeRef,
		Mode:   mode,
		Values: map[string]Slot{refSlotKey: Ref(name)},
	})
}
