package iso

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

func makeImage(f *File) ([]byte, error) {
	label := f.Label
	if label == "" {
		label = defaultLabel
	}
	if len(label) > 32 {
		return nil, ErrName
	}
	catalog, bootable := resolveCatalog(f.Catalog)
	boot := ""
	if bootable {
		boot = catalog.BootFile
		if err := cleanBoot(boot); err != nil {
			return nil, err
		}
		if !catalog.Platform.IsValid() || !catalog.Emulation.IsValid() {
			return nil, ErrFormat
		}
	}
	seen := make(map[string]struct{}, len(f.Entries))
	for _, entry := range f.Entries {
		if entry == nil || (bootable && reservedName(entry.Name, boot)) {
			return nil, ErrName
		}
		if _, ok := seen[entry.Name]; ok {
			return nil, ErrName
		}
		seen[entry.Name] = struct{}{}
		if entry.Mode&fs.ModeType&^fs.ModeDir != 0 {
			return nil, ErrKind
		}
	}

	dir, err := os.MkdirTemp("", "lewkit-iso")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	tree := filepath.Join(dir, "tree")
	if err := os.Mkdir(tree, 0o755); err != nil {
		return nil, err
	}
	if err := writeTree(tree, f.Entries); err != nil {
		return nil, err
	}
	if bootable {
		fat, err := writeFAT(dir, tree, payloadSize(f.Entries))
		if err != nil {
			return nil, err
		}
		bootPath := filepath.Join(tree, filepath.FromSlash(boot))
		if err := os.MkdirAll(filepath.Dir(bootPath), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(bootPath, fat, 0o644); err != nil {
			return nil, err
		}
	}
	raw, err := writeISO(dir, tree, label, boot, catalog)
	if err != nil {
		return nil, err
	}
	// Windows stat cannot return the Unix mode os.Chmod was asked for.
	if !hostKeepsUnixMode() {
		if err := stampRockRidgeModes(raw, f.Entries); err != nil {
			return nil, err
		}
	}
	return raw, nil
}

func cleanBoot(name string) error {
	cleaned, err := cleanName(name)
	if err != nil {
		return err
	}
	if cleaned != name || strings.Contains(name, "/") {
		return ErrName
	}
	return nil
}

func payloadSize(entries []*Entry) int64 {
	var n int64
	for _, entry := range entries {
		if entry.Mode&fs.ModeDir == 0 {
			n += int64(len(entry.Data))
		}
	}
	return n
}

func writeTree(dir string, entries []*Entry) error {
	for _, entry := range entries {
		full := filepath.Join(dir, filepath.FromSlash(entry.Name))
		if entry.Mode&fs.ModeDir != 0 {
			if err := os.MkdirAll(full, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(full, entry.Data, 0o644); err != nil {
			return err
		}
	}
	for _, entry := range entries {
		full := filepath.Join(dir, filepath.FromSlash(entry.Name))
		perm := entry.Mode.Perm()
		if perm == 0 {
			if entry.Mode&fs.ModeDir != 0 {
				perm = 0o755
			} else {
				perm = 0o644
			}
		}
		if err := os.Chmod(full, perm); err != nil {
			return err
		}
		when := entry.Time
		if when.IsZero() {
			when = defaultTime
		}
		if err := os.Chtimes(full, when, when); err != nil {
			return err
		}
	}
	return nil
}

func writeFAT(dir, tree string, payload int64) ([]byte, error) {
	img := filepath.Join(dir, "efiboot.img")
	volume, err := diskfs.Create(img, fatVolumeSize(payload), diskfs.SectorSize512)
	if err != nil {
		return nil, err
	}
	fat, err := volume.CreateFilesystem(disk.FilesystemSpec{
		FSType:       filesystem.TypeFat16,
		VolumeLabel:  defaultLabel,
		Reproducible: true,
	})
	if err != nil {
		volume.Close()
		return nil, err
	}
	copyErr := copyTree(tree, fat)
	closeErr := volume.Close()
	if copyErr != nil {
		return nil, copyErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return os.ReadFile(img)
}

func copyTree(root string, dst filesystem.FileSystem) error {
	return filepath.WalkDir(root, func(full string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if full == root {
			return nil
		}
		rel, err := filepath.Rel(root, full)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(rel)
		if entry.IsDir() {
			return dst.Mkdir(name)
		}
		parent := path.Dir(name)
		if parent != "." {
			if err := dst.Mkdir(parent); err != nil && !os.IsExist(err) {
				return err
			}
		}
		in, err := os.Open(full)
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

func writeISO(dir, tree, label, boot string, catalog Catalog) ([]byte, error) {
	img := filepath.Join(dir, "cd.iso")
	out, err := os.Create(img)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	backend := file.New(out, false)
	fsys, err := iso9660.Create(backend, 0, 0, 2048, tree)
	if err != nil {
		return nil, err
	}
	options := iso9660.FinalizeOptions{
		RockRidge:        true,
		VolumeIdentifier: label,
	}
	if boot != "" {
		options.ElTorito = &iso9660.ElTorito{
			Entries: []*iso9660.ElToritoEntry{{
				Platform:  diskPlatform(catalog.Platform),
				Emulation: diskEmulation(catalog.Emulation),
				BootFile:  boot,
			}},
		}
	}
	err = fsys.Finalize(options)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	if err := out.Close(); err != nil {
		return nil, err
	}
	return os.ReadFile(img)
}

func diskPlatform(platform Platform) iso9660.Platform {
	switch platform {
	case PlatformBIOS:
		return iso9660.BIOS
	case PlatformPPC:
		return iso9660.PPC
	case PlatformMac:
		return iso9660.Mac
	default:
		return iso9660.EFI
	}
}

func diskEmulation(emulation Emulation) iso9660.Emulation {
	switch emulation {
	case EmulationFloppy12:
		return iso9660.Floppy12Emulation
	case EmulationFloppy144:
		return iso9660.Floppy144Emulation
	case EmulationFloppy288:
		return iso9660.Floppy288Emulation
	case EmulationHardDisk:
		return iso9660.HardDiskEmulation
	default:
		return iso9660.NoEmulation
	}
}

// fatVolumeSize is a FAT16 volume large enough for payload.
// go-diskfs rejects a FAT16 with fewer than 4085 clusters, so a small
// tree still gets an 8 MiB image. 32 MiB is 65536 sectors of 512 bytes,
// which does not fit the El Torito sector count.
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
