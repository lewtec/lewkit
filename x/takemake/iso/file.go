package iso

import (
	"io"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrNilReader = errs.Wrap(errs.ErrInvalid, "iso: nil reader")
	ErrNilWriter = errs.Wrap(errs.ErrInvalid, "iso: nil writer")
	ErrNilFile   = errs.Wrap(errs.ErrInvalid, "iso: nil file")
	ErrName      = errs.Wrap(errs.ErrInvalid, "iso: invalid name")
	ErrKind      = errs.Wrap(errs.ErrInvalid, "iso: not a regular file")
	ErrFormat    = errs.Wrap(errs.ErrSyntax, "iso: invalid image")
)

const (
	defaultLabel = "LEWKIT"
	defaultBoot  = "EFIBOOT.IMG"
)

// defaultTime is the modification time of an entry whose Time was not set.
// ISO 9660 records whole seconds.
var defaultTime = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// Entry is one ISO 9660 directory record other than the boot image.
// A directory has [fs.ModeDir] set and no bytes.
type Entry struct {
	Name string
	Mode fs.FileMode
	Time time.Time
	Data []byte
}

// Catalog is the El Torito boot catalog.
type Catalog struct {
	Platform  Platform
	Emulation Emulation
	BootFile  string
}

// File is one El Torito image.
// Entries are the ISO 9660 directory. Catalog is the boot catalog.
// Make writes those entries into the directory and into a FAT16 boot image.
type File struct {
	Label   string
	Catalog Catalog
	Entries []*Entry

	cur int
	bad error
}

// New returns an empty El Torito image.
// The catalog boots EFI with no emulation from EFIBOOT.IMG.
func New() *File {
	return &File{
		Label: defaultLabel,
		Catalog: Catalog{
			Platform:  PlatformEFI,
			Emulation: EmulationNone,
			BootFile:  defaultBoot,
		},
		cur: -1,
	}
}

// Volume sets the volume identifier and returns the file.
func (f *File) Volume(name string) *File {
	if f == nil {
		return nil
	}
	f.Label = name
	return f
}

// Platform sets the boot catalog platform and returns the file.
func (f *File) Platform(platform Platform) *File {
	if f == nil {
		return nil
	}
	f.Catalog.Platform = platform
	return f
}

// Emulation sets the boot catalog medium and returns the file.
func (f *File) Emulation(emulation Emulation) *File {
	if f == nil {
		return nil
	}
	f.Catalog.Emulation = emulation
	return f
}

// Boot sets the boot-image name and returns the file.
func (f *File) Boot(name string) *File {
	if f == nil {
		return nil
	}
	f.Catalog.BootFile = name
	return f
}

// Entry appends a file, or selects one that is already present, and returns the file.
// A missing parent is stored as a directory. Mode, Time, Dir, and Bytes apply to this entry.
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
	if len(data) == 0 {
		entry.Data = nil
	} else {
		entry.Data = append([]byte(nil), data...)
	}
	return f
}

// Take reads an El Torito image into a [File].
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

// Make writes the El Torito image.
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
	raw, err := makeImage(f)
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
		if entry.Name == name {
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
	for part := range strings.SplitSeq(cleaned, "/") {
		if part == "" || part == "." || part == ".." || len(part) > 255 {
			return "", ErrName
		}
	}
	return cleaned, nil
}

func wall(t time.Time) time.Time {
	t = t.UTC().Truncate(time.Second)
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
}

func reservedName(name, boot string) bool {
	base := path.Base(name)
	if strings.EqualFold(base, "boot.catalog") || strings.EqualFold(base, "BOOT.CAT") {
		return true
	}
	return boot != "" && strings.EqualFold(name, boot)
}
