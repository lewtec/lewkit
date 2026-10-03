package filedialog

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lewtec/lewkit/x/driver"
)

var errContent = errors.New("content uri")

var (
	contentMu   sync.Mutex
	contentOpen func([]string) (fs.FS, error)
)

// RegisterContent installs the opener for content URIs.
// The Android document driver calls it from init. A nil opener clears it.
func RegisterContent(open func([]string) (fs.FS, error)) {
	contentMu.Lock()
	contentOpen = open
	contentMu.Unlock()
}

// Open reads paths from [Choose] as a filesystem.
// One directory is the root. One file is a filesystem containing that file.
// Several paths share one root, named by their base names.
// A content URI uses the opener from [RegisterContent].
// No path, or a content URI mixed with a local path, is [ErrRequest].
// A content URI with no opener is [driver.ErrUnavailable].
func Open(names ...string) (fs.FS, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("%w: no path", ErrRequest)
	}
	content, err := contentNames(names)
	if err != nil {
		return nil, err
	}
	if content {
		return openContent(names)
	}
	return openLocal(names)
}

func contentNames(names []string) (bool, error) {
	var n int
	for _, name := range names {
		if strings.HasPrefix(name, "content:") {
			n++
		}
	}
	if n == 0 {
		return false, nil
	}
	if n != len(names) {
		return false, fmt.Errorf("%w: mixed content uri", ErrRequest)
	}
	return true, nil
}

func openContent(names []string) (fs.FS, error) {
	contentMu.Lock()
	open := contentOpen
	contentMu.Unlock()
	if open == nil {
		return nil, fmt.Errorf("%w: %w", driver.ErrUnavailable, errContent)
	}
	return open(names)
}

func openLocal(names []string) (fs.FS, error) {
	if len(names) == 1 {
		st, err := os.Stat(names[0])
		if err != nil {
			return nil, err
		}
		if st.IsDir() {
			return dirFS{path: names[0]}, nil
		}
	}
	seen := map[string]struct{}{}
	ents := make([]localEntry, 0, len(names))
	for _, name := range names {
		st, err := os.Stat(name)
		if err != nil {
			return nil, err
		}
		ents = append(ents, localEntry{
			name: takeName(filepath.Base(name), seen),
			path: name,
			dir:  st.IsDir(),
		})
	}
	slices.SortFunc(ents, func(a, b localEntry) int {
		return cmp.Compare(a.name, b.name)
	})
	return &localFS{entries: ents}, nil
}

type localEntry struct {
	name string
	path string
	dir  bool
}

// dirFS is one real directory. The root is that directory.
// Listings are sorted so a file's ReadDir matches fs.ReadDir.
type dirFS struct{ path string }

func (d dirFS) Open(name string) (fs.File, error) {
	f, err := os.DirFS(d.path).Open(name)
	if err != nil {
		return nil, err
	}
	file, ok := f.(*os.File)
	if !ok {
		return f, nil
	}
	st, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}
	return &localFile{name: path.Base(name), f: file, dir: st.IsDir()}, nil
}

type localFS struct {
	entries []localEntry
}

func (f *localFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if name == "." {
		ents := make([]fs.DirEntry, len(f.entries))
		for i, e := range f.entries {
			st, err := os.Stat(e.path)
			if err != nil {
				return nil, err
			}
			ents[i] = fs.FileInfoToDirEntry(namedInfo{FileInfo: st, name: e.name})
		}
		return &localFile{name: ".", dir: true, ents: ents}, nil
	}
	head, rest, more := strings.Cut(name, "/")
	var ent *localEntry
	for i := range f.entries {
		if f.entries[i].name == head {
			ent = &f.entries[i]
			break
		}
	}
	if ent == nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	if !more {
		file, err := os.Open(ent.path)
		if err != nil {
			return nil, err
		}
		return &localFile{name: ent.name, f: file, dir: ent.dir}, nil
	}
	if !ent.dir {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return dirFS{path: ent.path}.Open(rest)
}

type localFile struct {
	name string
	f    *os.File
	dir  bool
	ents []fs.DirEntry
	off  int
}

func (f *localFile) Stat() (fs.FileInfo, error) {
	if f.f == nil {
		return rootInfo{name: f.name}, nil
	}
	st, err := f.f.Stat()
	if err != nil {
		return nil, err
	}
	return namedInfo{FileInfo: st, name: f.name}, nil
}

func (f *localFile) Read(p []byte) (int, error) {
	if f.dir {
		return 0, &fs.PathError{Op: "read", Path: f.name, Err: fs.ErrInvalid}
	}
	return f.f.Read(p)
}

func (f *localFile) Close() error {
	if f.f == nil {
		return nil
	}
	return f.f.Close()
}

func (f *localFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if !f.dir {
		return nil, &fs.PathError{Op: "readdir", Path: f.name, Err: fs.ErrInvalid}
	}
	if f.ents == nil {
		all, err := f.f.ReadDir(-1)
		if err != nil {
			return nil, err
		}
		slices.SortFunc(all, func(a, b fs.DirEntry) int {
			return cmp.Compare(a.Name(), b.Name())
		})
		f.ents = all
	}
	return pageDir(f.ents, &f.off, n)
}

type namedInfo struct {
	fs.FileInfo
	name string
}

func (n namedInfo) Name() string { return n.name }

type rootInfo struct{ name string }

func (r rootInfo) Name() string       { return r.name }
func (r rootInfo) Size() int64        { return 0 }
func (r rootInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o555 }
func (r rootInfo) ModTime() time.Time { return time.Time{} }
func (r rootInfo) IsDir() bool        { return true }
func (r rootInfo) Sys() any           { return nil }

func pageDir(ents []fs.DirEntry, off *int, n int) ([]fs.DirEntry, error) {
	if *off >= len(ents) {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	if n <= 0 {
		out := ents[*off:]
		*off = len(ents)
		return out, nil
	}
	end := *off + n
	if end > len(ents) {
		end = len(ents)
	}
	out := ents[*off:end]
	*off = end
	return out, nil
}

func takeName(name string, seen map[string]struct{}) string {
	name = cleanName(name)
	if _, ok := seen[name]; !ok {
		seen[name] = struct{}{}
		return name
	}
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for n := 2; ; n++ {
		next := stem + " (" + strconv.Itoa(n) + ")" + ext
		if _, ok := seen[next]; ok {
			continue
		}
		seen[next] = struct{}{}
		return next
	}
}

func cleanName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" || name == "." || name == ".." {
		return "_"
	}
	return name
}
