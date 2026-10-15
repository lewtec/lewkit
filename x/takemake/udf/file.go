package udf

import (
	"io"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrNilReader = errs.Wrap(errs.ErrInvalid, "udf: nil reader")
	ErrNilWriter = errs.Wrap(errs.ErrInvalid, "udf: nil writer")
	ErrNilFile   = errs.Wrap(errs.ErrInvalid, "udf: nil file")
	ErrName      = errs.Wrap(errs.ErrInvalid, "udf: invalid name")
	ErrKind      = errs.Wrap(errs.ErrInvalid, "udf: invalid entry")
	ErrSymlink   = errs.Wrap(errs.ErrInvalid, "udf: symlink has no target")
	ErrFormat    = errs.Wrap(errs.ErrSyntax, "udf: invalid volume")
)

const defaultLabel = "LEWKIT"

// defaultTime is the modification time of an entry whose Time was not set.
var defaultTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// Entry is one record in the volume.
// A directory has [fs.ModeDir] set. A symlink has [fs.ModeSymlink] set and
// Target holds the path. Any other entry holds bytes.
type Entry struct {
	Name   string
	Mode   fs.FileMode
	Time   time.Time
	Data   []byte
	Target string
}

// File is one UDF volume.
type File struct {
	Label   string
	Entries []*Entry

	cur int
	bad error
}

// New returns an empty volume.
func New() *File {
	return &File{Label: defaultLabel, cur: -1}
}

// Volume sets the volume identifier and returns the file.
func (f *File) Volume(name string) *File {
	if f == nil {
		return nil
	}
	f.Label = name
	return f
}

// Entry appends a file, or selects one that is already present, and returns the file.
// A missing parent is stored as a directory. Mode, Time, Dir, Bytes, and Symlink
// apply to this entry.
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
			Mode: fs.ModeDir | 0o755,
			Time: defaultTime,
		})
	}
	if i := f.find(cleaned); i >= 0 {
		f.cur = i
		return f
	}
	f.Entries = append(f.Entries, &Entry{
		Name: cleaned,
		Mode: 0o644,
		Time: defaultTime,
	})
	f.cur = len(f.Entries) - 1
	return f
}

// Dir marks the latest entry as a directory and returns the file.
func (f *File) Dir() *File {
	entry := f.current()
	if entry == nil {
		return f
	}
	perm := entry.Mode.Perm()
	if perm == 0 || (entry.Mode&fs.ModeType == 0 && perm == 0o644) {
		perm = 0o755
	}
	entry.Mode = fs.ModeDir | perm
	entry.Data = nil
	entry.Target = ""
	return f
}

// Mode sets the latest entry's permissions and returns the file.
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
	perm := mode & (fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky)
	entry.Mode = kind | perm
	return f
}

// Time sets the latest entry's modification time and returns the file.
// The recorded value is whole UTC seconds.
func (f *File) Time(t time.Time) *File {
	entry := f.current()
	if entry == nil {
		return f
	}
	entry.Time = wall(t)
	return f
}

// Bytes sets the latest entry's contents and returns the file.
// The entry is a regular file.
func (f *File) Bytes(data []byte) *File {
	entry := f.current()
	if entry == nil {
		return f
	}
	perm := entry.Mode & (fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky)
	if perm&fs.ModePerm == 0 {
		perm |= 0o644
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

// Symlink sets the latest entry's target and returns the file.
func (f *File) Symlink(target string) *File {
	entry := f.current()
	if entry == nil {
		return f
	}
	if target == "" || strings.ContainsRune(target, 0) {
		f.bad = ErrSymlink
		return f
	}
	perm := entry.Mode.Perm()
	if perm == 0 || (entry.Mode&fs.ModeType == 0 && perm == 0o644) {
		perm = 0o777
	}
	entry.Mode = fs.ModeSymlink | perm
	entry.Target = target
	entry.Data = nil
	return f
}

// Take reads a UDF volume into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, ErrNilReader
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return take(raw)
}

// Make writes the UDF volume.
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
	raw, err := makeVolume(f)
	if err != nil {
		return err
	}
	_, err = w.Write(raw)
	return err
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
	if strings.HasPrefix(name, "/") {
		return "", ErrName
	}
	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", ErrName
	}
	for _, part := range strings.Split(cleaned, "/") {
		if part == "" || part == "." || part == ".." || len(part) > 254 {
			return "", ErrName
		}
	}
	return cleaned, nil
}

func wall(t time.Time) time.Time {
	t = t.UTC().Truncate(time.Second)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
}
