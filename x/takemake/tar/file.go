package tar

import (
	"io"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrNilReader = errs.Wrap(errs.ErrInvalid, "tar: nil reader")
	ErrNilWriter = errs.Wrap(errs.ErrInvalid, "tar: nil writer")
	ErrNilFile   = errs.Wrap(errs.ErrInvalid, "tar: nil file")
	ErrName      = errs.Wrap(errs.ErrInvalid, "tar: invalid name")
	ErrKind      = errs.Wrap(errs.ErrInvalid, "tar: invalid entry")
	ErrEntry     = errs.Wrap(errs.ErrUnsupported, "tar: unsupported entry")
	ErrFormat    = errs.Wrap(errs.ErrSyntax, "tar: invalid archive")
)

const (
	defaultMode = 0o644
	dirMode     = 0o755
	linkMode    = 0o777
)

// defaultTime is the modification time of an entry whose Time was not set.
var defaultTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// Entry is one member of the archive.
// A directory has [fs.ModeDir] set. A symlink has [fs.ModeSymlink] set
// and [Entry.Target] holds the link text. Other members hold [Entry.Data].
type Entry struct {
	Name   string
	Mode   fs.FileMode
	Time   time.Time
	Data   []byte
	Target string
}

// File is one tar archive.
// Entries are the members in archive order.
type File struct {
	Entries []*Entry

	cur int
	bad error
}

// New returns an empty archive.
func New() *File {
	return &File{cur: -1}
}

// Entry appends a member, or selects one that is already present, and returns the archive.
// A missing parent is stored as a directory. Mode, Time, Dir, Bytes, and Symlink apply to this member.
func (f *File) Entry(name string) *File {
	if f == nil {
		return nil
	}
	cleaned, err := cleanName(name)
	if err != nil {
		f.bad = err
		return f
	}
	parts := strings.Split(cleaned, "/")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[:i], "/")
		if f.find(parent) >= 0 {
			continue
		}
		f.Entries = append(f.Entries, &Entry{
			Name: parent,
			Mode: fs.ModeDir | dirMode,
			Time: defaultTime,
		})
	}
	if i := f.find(cleaned); i >= 0 {
		f.cur = i
		return f
	}
	f.Entries = append(f.Entries, &Entry{
		Name: cleaned,
		Mode: defaultMode,
		Time: defaultTime,
	})
	f.cur = len(f.Entries) - 1
	return f
}

// Dir marks the latest entry as a directory and returns the archive.
func (f *File) Dir() *File {
	entry := f.current()
	if entry == nil {
		return f
	}
	perm := permission(entry.Mode)
	if perm == 0 || perm == defaultMode {
		perm = dirMode
	}
	entry.Mode = fs.ModeDir | perm
	entry.Data = nil
	entry.Target = ""
	return f
}

// Mode sets the latest entry's permissions and returns the archive.
// A type bit already stored on the entry is kept when mode has none.
func (f *File) Mode(mode fs.FileMode) *File {
	entry := f.current()
	if entry == nil {
		return f
	}
	kind := entry.Mode & fs.ModeType
	if mode&fs.ModeType != 0 {
		kind = mode & fs.ModeType
	}
	entry.Mode = kind | permission(mode)
	if kind&fs.ModeDir != 0 {
		entry.Data = nil
		entry.Target = ""
	}
	if kind&fs.ModeSymlink == 0 {
		entry.Target = ""
	}
	return f
}

// Time sets the latest entry's modification time and returns the archive.
func (f *File) Time(t time.Time) *File {
	entry := f.current()
	if entry == nil {
		return f
	}
	entry.Time = wall(t)
	return f
}

// Bytes sets the latest entry's contents and returns the archive.
// The entry is a regular file.
func (f *File) Bytes(data []byte) *File {
	entry := f.current()
	if entry == nil {
		return f
	}
	perm := permission(entry.Mode)
	if perm&fs.ModePerm == 0 {
		perm |= defaultMode
	}
	entry.Mode = perm
	entry.Target = ""
	if len(data) == 0 {
		entry.Data = nil
	} else {
		entry.Data = append([]byte(nil), data...)
	}
	return f
}

// Symlink sets the latest entry's link target and returns the archive.
// The entry is a symlink.
func (f *File) Symlink(target string) *File {
	entry := f.current()
	if entry == nil {
		return f
	}
	if target == "" || strings.ContainsRune(target, 0) {
		f.bad = ErrName
		return f
	}
	perm := permission(entry.Mode)
	if perm == 0 || perm == defaultMode {
		perm = linkMode
	}
	entry.Mode = fs.ModeSymlink | perm
	entry.Target = target
	entry.Data = nil
	return f
}

// Take reads a tar archive into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, ErrNilReader
	}
	return readArchive(r)
}

// Make writes the tar archive.
func (f *File) Make(w io.Writer) error {
	if f == nil {
		return ErrNilFile
	}
	if w == nil {
		return ErrNilWriter
	}
	if f.bad != nil {
		return f.bad
	}
	return writeArchive(f, w)
}

func (f *File) current() *Entry {
	if f == nil || f.cur < 0 || f.cur >= len(f.Entries) {
		return nil
	}
	return f.Entries[f.cur]
}

func (f *File) find(name string) int {
	for i, entry := range f.Entries {
		if entry != nil && entry.Name == name {
			return i
		}
	}
	return -1
}

func cleanName(name string) (string, error) {
	if name == "" || strings.ContainsRune(name, 0) || strings.Contains(name, "\\") {
		return "", ErrName
	}
	if len(name) > 1<<20 {
		return "", ErrName
	}
	if strings.HasPrefix(name, "/") {
		return "", ErrName
	}
	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", ErrName
	}
	for part := range strings.SplitSeq(cleaned, "/") {
		if part == "" || part == "." || part == ".." {
			return "", ErrName
		}
	}
	return cleaned, nil
}

func wall(t time.Time) time.Time {
	if t.IsZero() {
		return defaultTime
	}
	return t.UTC()
}

func permission(mode fs.FileMode) fs.FileMode {
	return mode & (fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky)
}
