package compose

import (
	"io/fs"

	"github.com/lewtec/lewkit/x/path"
)

const refSlotKey = "src"

// Squash lowers a source filesystem into a tree.
// A directory whose last two extensions are .d and .tmpl becomes one [TypeLines] file.
// The destination path is that directory with those extensions removed.
// Each direct child file is a text slot keyed by the child name.
// A child directory is [ErrPath]. An empty directory adds no file.
// Any other file becomes a [TypeRef] slot with key src.
// A file whose [path.Path.Suffix] is .tmpl is skipped. A non-directory
// with the .d.tmpl extensions is [ErrPath].
func Squash(fsys fs.FS) (*Tree, error) {
	if fsys == nil {
		return nil, errNilFilesystem
	}
	tree := New()
	err := path.New(".").WalkDir(fsys, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		current := path.New(name)
		if current.String() == "." {
			return nil
		}
		if isDotDirectory(current) {
			if !entry.IsDir() {
				return pathError("squash", current.String(), ErrPath)
			}
			if err := addDotDirectory(tree, fsys, current); err != nil {
				return err
			}
			return fs.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		if current.Suffix() == ".tmpl" {
			return nil
		}
		return addRef(tree, current, entry)
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
	entries, err := directory.ReadDir(fsys)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		child := directory.Join(entry.Name())
		if entry.IsDir() {
			return pathError("squash", child.String(), ErrPath)
		}
		if child.Suffix() == ".tmpl" {
			continue
		}
		body, err := child.ReadFile(fsys)
		if err != nil {
			return err
		}
		err = tree.Add(destination, File{
			Type:   TypeLines,
			Values: map[string]Slot{child.Name(): Text(string(body))},
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func isDotDirectory(name path.Path) bool {
	suffixes := name.Suffixes()
	count := len(suffixes)
	return count >= 2 && suffixes[count-2] == ".d" && suffixes[count-1] == ".tmpl"
}

func linesPath(directory path.Path) (path.Path, error) {
	if !isDotDirectory(directory) {
		return path.Path{}, ErrPath
	}
	trimmed := directory.WithSuffix("").WithSuffix("")
	base := trimmed.Name()
	if base == "" || base == "." || base == ".." {
		return path.Path{}, ErrPath
	}
	return trimmed, nil
}

func addRef(tree *Tree, name path.Path, entry fs.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return err
	}
	mode := info.Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}
	return tree.Add(name, File{
		Type:   TypeRef,
		Mode:   mode,
		Values: map[string]Slot{refSlotKey: Ref(name.String())},
	})
}
