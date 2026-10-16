package iso

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/errs"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

var (
	errEFIBootImage         = errors.New("iso: EFIBOOT.IMG is the boot image")
	errEmptyFilesystem      = errs.Wrap(errs.ErrEmpty, "iso: empty filesystem")
	errFilesystemNotISO9660 = errors.New("iso: filesystem is not iso9660")
	errISO                  = errors.New("iso")
	errImageShort           = errors.New("iso: image is short")
	errNilFilesystem        = errors.New("iso: nil filesystem")
)

const bootImage = "EFIBOOT.IMG"

// Write returns an El Torito UEFI CD of fsys.
// The tree is copied with go-diskfs into a FAT16 boot image and into the
// ISO9660 directory. The boot catalog points at that FAT image.
// EFIBOOT.IMG is the boot image, not a member of the source tree.
func Write(fsys iofs.FS) ([]byte, error) {
	if fsys == nil {
		return nil, errNilFilesystem
	}
	payload, count, err := inspect(fsys)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, errEmptyFilesystem
	}
	dir, err := os.MkdirTemp("", "lewkit-iso")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	fat, err := writeFAT(dir, fsys, payload)
	if err != nil {
		return nil, err
	}
	return writeISO(dir, fsys, fat, payload)
}

// ReadFile reads one file out of an image written by [Write].
// name is an io/fs path. Lookup is case-insensitive.
func ReadFile(image []byte, name string) ([]byte, error) {
	if len(image) < 17*2048 {
		return nil, errImageShort
	}
	fsys, err := iso9660.Read(imageStorage(image), int64(len(image)), 0, 2048)
	if err != nil {
		return nil, err
	}
	for _, candidate := range []string{name, strings.ToUpper(name)} {
		body, err := fsys.ReadFile(candidate)
		if err == nil {
			return body, nil
		}
	}
	return nil, fmt.Errorf("%w: missing %s", errISO, name)
}

func inspect(fsys iofs.FS) (int64, int, error) {
	var payload int64
	count := 0
	err := iofs.WalkDir(fsys, ".", func(name string, entry iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		count++
		if strings.EqualFold(path.Base(name), bootImage) {
			return errEFIBootImage
		}
		if entry.Type()&iofs.ModeSymlink != 0 {
			return fmt.Errorf("%w: %s is a symlink", errISO, name)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("%w: %s is not a regular file", errISO, name)
		}
		file, err := fsys.Open(name)
		if err != nil {
			return err
		}
		info, err := file.Stat()
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		payload += info.Size()
		return nil
	})
	return payload, count, err
}

func writeFAT(dir string, fsys iofs.FS, payload int64) ([]byte, error) {
	size := fatVolumeSize(payload)
	img := filepath.Join(dir, "efiboot.img")
	volume, err := diskfs.Create(img, size, diskfs.SectorSize512)
	if err != nil {
		return nil, err
	}
	defer volume.Close()
	fat, err := volume.CreateFilesystem(disk.FilesystemSpec{
		FSType:       filesystem.TypeFat16,
		VolumeLabel:  "LEWKIT",
		Reproducible: true,
	})
	if err != nil {
		return nil, err
	}
	if err := copyTree(fsys, fat); err != nil {
		return nil, err
	}
	return os.ReadFile(img)
}

func writeISO(dir string, fsys iofs.FS, fat []byte, payload int64) ([]byte, error) {
	size := isoVolumeSize(int64(len(fat)) + payload)
	img := filepath.Join(dir, "cd.iso")
	volume, err := diskfs.Create(img, size, diskfs.SectorSizeDefault)
	if err != nil {
		return nil, err
	}
	defer volume.Close()
	volume.LogicalBlocksize = 2048
	created, err := volume.CreateFilesystem(disk.FilesystemSpec{
		FSType:      filesystem.TypeISO9660,
		VolumeLabel: "LEWKIT",
	})
	if err != nil {
		return nil, err
	}
	if err := copyTree(fsys, created); err != nil {
		return nil, err
	}
	boot, err := created.OpenFile(bootImage, os.O_CREATE|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	if _, err := boot.Write(fat); err != nil {
		boot.Close()
		return nil, err
	}
	if err := boot.Close(); err != nil {
		return nil, err
	}
	iso, ok := created.(*iso9660.FileSystem)
	if !ok {
		return nil, errFilesystemNotISO9660
	}
	err = iso.Finalize(iso9660.FinalizeOptions{
		RockRidge:        true,
		VolumeIdentifier: "LEWKIT",
		ElTorito: &iso9660.ElTorito{
			Entries: []*iso9660.ElToritoEntry{{
				Platform:  iso9660.EFI,
				Emulation: iso9660.NoEmulation,
				BootFile:  bootImage,
			}},
		},
	})
	if err != nil {
		return nil, err
	}
	return os.ReadFile(img)
}

func copyTree(src iofs.FS, dst filesystem.FileSystem) error {
	return iofs.WalkDir(src, ".", func(name string, entry iofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if name == "." {
			return nil
		}
		if entry.IsDir() {
			return dst.Mkdir(name)
		}
		parent := path.Dir(name)
		if parent != "." {
			if err := dst.Mkdir(parent); err != nil {
				return err
			}
		}
		in, err := src.Open(name)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := dst.OpenFile(name, os.O_CREATE|os.O_RDWR)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

// fatVolumeSize is a FAT16 volume large enough for payload.
// go-diskfs rejects a FAT16 with fewer than 4085 clusters, so a small
// tree still gets an 8 MiB image. 32 MiB is 65536 sectors of 512 bytes,
// which does not fit the El Torito sector count, so the small image stays
// under that.
func fatVolumeSize(payload int64) int64 {
	const mb = 1 << 20
	need := payload + 2*mb
	if need < 8*mb {
		need = 8 * mb
	}
	if rem := need % mb; rem != 0 {
		need += mb - rem
	}
	return need
}

func isoVolumeSize(payload int64) int64 {
	size := payload + 2<<20
	const block = 2048
	if rem := size % block; rem != 0 {
		size += block - rem
	}
	return size
}

func imageStorage(image []byte) backend.Storage {
	return &storage{r: bytes.NewReader(image), n: int64(len(image))}
}

type storage struct {
	r   *bytes.Reader
	n   int64
	pos int64
}

func (s *storage) Read(p []byte) (int, error) {
	n, err := s.r.ReadAt(p, s.pos)
	s.pos += int64(n)
	return n, err
}

func (s *storage) ReadAt(p []byte, off int64) (int, error) {
	return s.r.ReadAt(p, off)
}

func (s *storage) Seek(offset int64, whence int) (int64, error) {
	next, err := s.r.Seek(offset, whence)
	if err != nil {
		return 0, err
	}
	s.pos = next
	return next, nil
}

func (s *storage) Stat() (iofs.FileInfo, error) { return storageInfo{n: s.n}, nil }

func (s *storage) Close() error { return nil }

func (s *storage) Sys() (*os.File, error) { return nil, backend.ErrNotSuitable }

func (s *storage) Writable() (backend.WritableFile, error) {
	return nil, backend.ErrIncorrectOpenMode
}

func (s *storage) Path() string { return "" }

type storageInfo struct{ n int64 }

func (storageInfo) Name() string        { return "" }
func (i storageInfo) Size() int64       { return i.n }
func (storageInfo) Mode() iofs.FileMode { return 0 }
func (storageInfo) ModTime() time.Time  { return time.Time{} }
func (storageInfo) IsDir() bool         { return false }
func (storageInfo) Sys() any            { return nil }
