package compose

import (
	"bytes"
	"io"
	iofs "io/fs"
	"time"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

// FS is the encoded destination. Open returns the combined file.
type FS struct {
	root *lewfs.PathNode[File]
	base iofs.FS
}

var (
	_ iofs.FS         = (*FS)(nil)
	_ iofs.ReadDirFS  = (*FS)(nil)
	_ iofs.ReadFileFS = (*FS)(nil)
	_ iofs.ReadLinkFS = (*FS)(nil)
	_ iofs.StatFS     = (*FS)(nil)
)

// FS encodes the tree. base opens ref slots. A nil base rejects them at Open.
func (tree *Tree) FS(base iofs.FS) (*FS, error) {
	if tree == nil {
		return nil, pathError("open", ".", iofs.ErrInvalid)
	}
	root := lewfs.NewPathDir[File](".")
	for name, file := range tree.files {
		stored := file
		if err := root.Add(name, &stored, false); err != nil {
			return nil, err
		}
	}
	return &FS{root: root, base: base}, nil
}

// Open implements [io/fs.FS].
func (filesystem *FS) Open(name string) (iofs.File, error) {
	node, err := filesystem.lookup(name)
	if err != nil {
		return nil, err
	}
	if node.IsDir() {
		return node.OpenDir(filesystem.info), nil
	}
	file := node.Payload()
	if file == nil {
		return nil, pathError("open", name, iofs.ErrNotExist)
	}
	body, err := Encode(*file, filesystem.base)
	if err != nil {
		return nil, pathError("open", name, err)
	}
	info := node.Info(int64(len(body)), presentedMode(file), time.Time{})
	return &memFile{info: info, Reader: bytes.NewReader(body)}, nil
}

// ReadDir implements [io/fs.ReadDirFS].
func (filesystem *FS) ReadDir(name string) ([]iofs.DirEntry, error) {
	node, err := filesystem.lookup(name)
	if err != nil {
		return nil, err
	}
	if !node.IsDir() {
		return nil, pathError("readdir", name, iofs.ErrInvalid)
	}
	return node.Entries(filesystem.info), nil
}

// ReadFile implements [io/fs.ReadFileFS].
func (filesystem *FS) ReadFile(name string) ([]byte, error) {
	opened, err := filesystem.Open(name)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	body, err := io.ReadAll(opened)
	if err != nil {
		return nil, pathError("read", name, err)
	}
	return body, nil
}

// ReadLink implements [io/fs.ReadLinkFS].
// The result is the link target stored in the tree.
func (filesystem *FS) ReadLink(name string) (string, error) {
	node, err := filesystem.lookup(name)
	if err != nil {
		return "", err
	}
	if node.IsDir() {
		return "", pathError("readlink", name, iofs.ErrInvalid)
	}
	file := node.Payload()
	if file == nil || file.Type != TypeLink {
		return "", pathError("readlink", name, iofs.ErrInvalid)
	}
	for _, slot := range file.Values {
		target, ok := slot.Link()
		if ok {
			return target, nil
		}
	}
	return "", pathError("readlink", name, ErrSlot)
}

// Lstat implements [io/fs.ReadLinkFS].
// A link is not followed. The info matches [FS.Stat].
func (filesystem *FS) Lstat(name string) (iofs.FileInfo, error) {
	return filesystem.Stat(name)
}

// Stat implements [io/fs.StatFS].
func (filesystem *FS) Stat(name string) (iofs.FileInfo, error) {
	opened, err := filesystem.Open(name)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	return opened.Stat()
}

func (filesystem *FS) lookup(name string) (*lewfs.PathNode[File], error) {
	if filesystem == nil || filesystem.root == nil {
		return nil, pathError("open", name, iofs.ErrInvalid)
	}
	return filesystem.root.Lookup(name)
}

func (filesystem *FS) info(node *lewfs.PathNode[File]) iofs.FileInfo {
	if node.IsDir() {
		return node.Info(0, iofs.ModeDir|0o755, time.Time{})
	}
	return node.Info(0, presentedMode(node.Payload()), time.Time{})
}

func presentedMode(file *File) iofs.FileMode {
	mode := iofs.FileMode(0o644)
	if file != nil && file.Mode != 0 {
		mode = file.Mode
	}
	if file != nil && file.Type == TypeLink {
		mode |= iofs.ModeSymlink
	}
	return mode
}

type memFile struct {
	info iofs.FileInfo
	*bytes.Reader
}

func (file *memFile) Stat() (iofs.FileInfo, error) { return file.info, nil }

func (file *memFile) Close() error { return nil }
