package iso

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/diskfs/go-diskfs/backend/file"
	"github.com/diskfs/go-diskfs/filesystem/fat16"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

const sectorSize = 2048

func take(raw []byte) (*File, error) {
	if len(raw) < 18*sectorSize {
		return nil, ErrFormat
	}
	if string(raw[16*sectorSize+1:16*sectorSize+6]) != "CD001" {
		return nil, ErrFormat
	}
	platform, emulation, bootLBA, err := bootCatalog(raw)
	if err != nil {
		return nil, err
	}

	dir, err := os.MkdirTemp("", "lewkit-iso-read")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	isoPath := dir + "/cd.iso"
	if err := os.WriteFile(isoPath, raw, 0o644); err != nil {
		return nil, err
	}
	isoFile, err := os.Open(isoPath)
	if err != nil {
		return nil, err
	}
	defer isoFile.Close()
	fsys, err := iso9660.Read(file.New(isoFile, true), int64(len(raw)), 0, sectorSize)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}

	found, err := walkISO(fsys, ".")
	if err != nil {
		return nil, err
	}
	var bootName string
	var boot []byte
	entries := make([]*Entry, 0, len(found))
	for _, item := range found {
		if item.loc == bootLBA && !item.info.IsDir() {
			bootName = item.name
			boot = item.data
			continue
		}
		if reservedName(item.name, "") {
			continue
		}
		mode := item.info.Mode() & (fs.ModeDir | fs.ModePerm | fs.ModeSetuid | fs.ModeSetgid | fs.ModeSticky)
		entry := &Entry{
			Name: item.name,
			Mode: mode,
			Time: wall(item.info.ModTime()),
		}
		if !item.info.IsDir() {
			entry.Data = item.data
			if len(entry.Data) == 0 {
				entry.Data = nil
			}
		}
		entries = append(entries, entry)
	}
	if bootName == "" {
		return nil, ErrFormat
	}
	if err := fatMatches(boot, entries); err != nil {
		return nil, err
	}
	slices.SortFunc(entries, func(a, b *Entry) int {
		return strings.Compare(a.Name, b.Name)
	})
	label := strings.TrimRight(fsys.Label(), " \x00")
	return &File{
		Label: label,
		Catalog: Catalog{
			Platform:  platform,
			Emulation: emulation,
			BootFile:  path.Base(bootName),
		},
		Entries: entries,
		cur:     -1,
	}, nil
}

type foundEntry struct {
	name string
	info fs.FileInfo
	data []byte
	loc  uint32
}

func walkISO(fsys *iso9660.FileSystem, name string) ([]foundEntry, error) {
	list, err := fsys.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrFormat, err)
	}
	var out []foundEntry
	for _, item := range list {
		info, err := item.Info()
		if err != nil {
			return nil, err
		}
		child := info.Name()
		if name != "." {
			child = name + "/" + child
		}
		var data []byte
		var loc uint32
		if st, ok := info.Sys().(*iso9660.StatT); ok {
			loc = st.Location
		}
		if !info.IsDir() {
			data, err = readISOFile(fsys, child)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, foundEntry{name: child, info: info, data: data, loc: loc})
		if info.IsDir() {
			nested, err := walkISO(fsys, child)
			if err != nil {
				return nil, err
			}
			out = append(out, nested...)
		}
	}
	return out, nil
}

func readISOFile(fsys *iso9660.FileSystem, name string) ([]byte, error) {
	body, err := fsys.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrFormat, name)
	}
	if len(body) == 0 {
		return nil, nil
	}
	return bytes.Clone(body), nil
}

func bootCatalog(raw []byte) (Platform, Emulation, uint32, error) {
	record := raw[17*sectorSize : 18*sectorSize]
	if !bytes.Contains(record[:64], []byte("EL TORITO SPECIFICATION")) {
		return 0, 0, 0, ErrFormat
	}
	catLBA := binary.LittleEndian.Uint32(record[71:])
	off := int(catLBA) * sectorSize
	if catLBA == 0 || off < 0 || off+64 > len(raw) {
		return 0, 0, 0, ErrFormat
	}
	cat := raw[off:]
	if cat[0] != 0x01 || cat[30] != 0x55 || cat[31] != 0xAA || cat[32] != 0x88 {
		return 0, 0, 0, ErrFormat
	}
	platform, err := platformFrom(cat[1])
	if err != nil {
		return 0, 0, 0, err
	}
	emulation, err := emulationFrom(cat[33])
	if err != nil {
		return 0, 0, 0, err
	}
	bootLBA := binary.LittleEndian.Uint32(cat[40:])
	if bootLBA == 0 {
		return 0, 0, 0, ErrFormat
	}
	return platform, emulation, bootLBA, nil
}

func platformFrom(b byte) (Platform, error) {
	switch b {
	case 0x00:
		return PlatformBIOS, nil
	case 0x01:
		return PlatformPPC, nil
	case 0x02:
		return PlatformMac, nil
	case 0xEF:
		return PlatformEFI, nil
	default:
		return 0, ErrPlatform
	}
}

func emulationFrom(b byte) (Emulation, error) {
	switch b {
	case 0:
		return EmulationNone, nil
	case 1:
		return EmulationFloppy12, nil
	case 2:
		return EmulationFloppy144, nil
	case 3:
		return EmulationFloppy288, nil
	case 4:
		return EmulationHardDisk, nil
	default:
		return 0, ErrEmulation
	}
}

func fatMatches(image []byte, entries []*Entry) error {
	if len(image) < 512 {
		return ErrFormat
	}
	dir, err := os.MkdirTemp("", "lewkit-iso-fat")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	pathName := dir + "/efiboot.img"
	if err := os.WriteFile(pathName, image, 0o644); err != nil {
		return err
	}
	fatFile, err := os.Open(pathName)
	if err != nil {
		return err
	}
	defer fatFile.Close()
	fat, err := fat16.Read(file.New(fatFile, true), int64(len(image)), 0, 512)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrFormat, err)
	}
	for _, entry := range entries {
		if entry.Mode&fs.ModeDir != 0 {
			continue
		}
		body, err := readFAT(fat, entry.Name)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFormat, entry.Name)
		}
		if !bytes.Equal(body, entry.Data) {
			return fmt.Errorf("%w: %s", ErrFormat, entry.Name)
		}
	}
	return nil
}

// readFAT reads one file at its recorded size.
// fat12.File.Read past the first cluster fills the rest of the cluster,
// so a sized buffer is required.
func readFAT(fat *fat16.FileSystem, name string) ([]byte, error) {
	opened, err := fat.Open(name)
	if err != nil {
		return nil, err
	}
	defer opened.Close()
	info, err := opened.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() == 0 {
		return nil, nil
	}
	buf := make([]byte, info.Size())
	n, err := opened.Read(buf)
	if int64(n) != info.Size() {
		if err == nil {
			return nil, fmt.Errorf("short fat read")
		}
		return nil, err
	}
	return buf, nil
}
