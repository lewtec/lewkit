package compose

import (
	"bytes"
	"cmp"
	"io"
	iofs "io/fs"
	"slices"
	"strings"
	"time"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/path"
)

// FS is the encoded destination. Open returns the combined file.
type FS struct {
	files map[string]File
	dirs  map[string]struct{}
	base  iofs.FS
}

var (
	_ iofs.FS         = (*FS)(nil)
	_ iofs.ReadDirFS  = (*FS)(nil)
	_ iofs.ReadFileFS = (*FS)(nil)
	_ iofs.StatFS     = (*FS)(nil)
)

// FS encodes the tree. base opens ref slots. A nil base rejects them at Open.
func (tree *Tree) FS(base iofs.FS) (*FS, error) {
	if tree == nil {
		return nil, pathError("open", ".", iofs.ErrInvalid)
	}
	files := make(map[string]File, len(tree.files))
	dirs := map[string]struct{}{".": {}}
	for name, file := range tree.files {
		if err := file.check(); err != nil {
			return nil, pathError("open", name, err)
		}
		for existing := range files {
			if pathClash(existing, name) {
				return nil, pathError("open", name, ErrPath)
			}
		}
		copied, err := file.normalized()
		if err != nil {
			return nil, pathError("open", name, err)
		}
		files[name] = copied
		directory := path.New(name).Parent()
		for directory.String() != "." {
			dirs[directory.String()] = struct{}{}
			directory = directory.Parent()
		}
	}
	return &FS{files: files, dirs: dirs, base: base}, nil
}

// Open implements [io/fs.FS].
func (filesystem *FS) Open(name string) (iofs.File, error) {
	if filesystem == nil {
		return nil, pathError("open", name, iofs.ErrInvalid)
	}
	if name == "." {
		return filesystem.directory(name)
	}
	if !iofs.ValidPath(name) {
		return nil, pathError("open", name, iofs.ErrInvalid)
	}
	if _, isDir := filesystem.dirs[name]; isDir {
		return filesystem.directory(name)
	}
	file, ok := filesystem.files[name]
	if !ok {
		return nil, pathError("open", name, iofs.ErrNotExist)
	}
	body, err := Encode(file, filesystem.base)
	if err != nil {
		return nil, pathError("open", name, err)
	}
	info := fileInfo{
		name: path.New(name).Name(),
		size: int64(len(body)),
		mode: presentedMode(file.Mode),
	}
	return &memFile{info: info, Reader: bytes.NewReader(body)}, nil
}

// ReadDir implements [io/fs.ReadDirFS].
func (filesystem *FS) ReadDir(name string) ([]iofs.DirEntry, error) {
	if filesystem == nil {
		return nil, pathError("readdir", name, iofs.ErrInvalid)
	}
	if name != "." && !iofs.ValidPath(name) {
		return nil, pathError("readdir", name, iofs.ErrInvalid)
	}
	if _, isDir := filesystem.dirs[name]; !isDir {
		if _, isFile := filesystem.files[name]; isFile {
			return nil, pathError("readdir", name, iofs.ErrInvalid)
		}
		return nil, pathError("readdir", name, iofs.ErrNotExist)
	}
	return filesystem.children(name), nil
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

// Stat implements [io/fs.StatFS].
func (filesystem *FS) Stat(name string) (iofs.FileInfo, error) {
	opened, err := filesystem.Open(name)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	return opened.Stat()
}

func (filesystem *FS) directory(name string) (iofs.File, error) {
	entries := filesystem.children(name)
	base := "."
	if name != "." {
		base = path.New(name).Name()
	}
	return &dirFile{
		name:    name,
		info:    fileInfo{name: base, mode: iofs.ModeDir | 0o755},
		entries: entries,
	}, nil
}

func (filesystem *FS) children(name string) []iofs.DirEntry {
	prefix := ""
	if name != "." {
		prefix = name + "/"
	}
	seen := map[string]bool{}
	entries := make([]iofs.DirEntry, 0)
	for filePath, file := range filesystem.files {
		rest := filePath
		if prefix != "" {
			if !strings.HasPrefix(filePath, prefix) {
				continue
			}
			rest = filePath[len(prefix):]
		}
		element, _, nested := strings.Cut(rest, "/")
		if element == "" || seen[element] {
			continue
		}
		seen[element] = true
		if nested {
			entries = append(entries, dirEntry{name: element, mode: iofs.ModeDir | 0o755})
			continue
		}
		entries = append(entries, dirEntry{name: element, mode: presentedMode(file.Mode)})
	}
	slices.SortFunc(entries, func(left, right iofs.DirEntry) int {
		return cmp.Compare(left.Name(), right.Name())
	})
	return entries
}

func presentedMode(mode iofs.FileMode) iofs.FileMode {
	if mode == 0 {
		return 0o644
	}
	return mode
}

type memFile struct {
	info fileInfo
	*bytes.Reader
}

func (file *memFile) Stat() (iofs.FileInfo, error) { return file.info, nil }

func (file *memFile) Close() error { return nil }

type dirFile struct {
	name    string
	info    fileInfo
	entries []iofs.DirEntry
	off     int
}

func (directory *dirFile) Stat() (iofs.FileInfo, error) { return directory.info, nil }

func (directory *dirFile) Read([]byte) (int, error) {
	return 0, pathError("read", directory.name, iofs.ErrInvalid)
}

func (directory *dirFile) Close() error { return nil }

func (directory *dirFile) ReadDir(count int) ([]iofs.DirEntry, error) {
	return lewfs.DirEntries(directory.entries, &directory.off, count)
}

type fileInfo struct {
	name string
	size int64
	mode iofs.FileMode
}

func (info fileInfo) Name() string        { return info.name }
func (info fileInfo) Size() int64         { return info.size }
func (info fileInfo) Mode() iofs.FileMode { return info.mode }
func (info fileInfo) ModTime() time.Time  { return time.Time{} }
func (info fileInfo) IsDir() bool         { return info.mode.IsDir() }
func (info fileInfo) Sys() any            { return nil }

type dirEntry struct {
	name string
	mode iofs.FileMode
}

func (entry dirEntry) Name() string        { return entry.name }
func (entry dirEntry) IsDir() bool         { return entry.mode.IsDir() }
func (entry dirEntry) Type() iofs.FileMode { return entry.mode.Type() }
func (entry dirEntry) Info() (iofs.FileInfo, error) {
	return fileInfo{name: entry.name, mode: entry.mode}, nil
}
