package ios

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	loadCmdSegment64    = 0x19
	loadCmdBuildVersion = 0x32
	platformIOS         = 2
	platformIOSSim      = 7
	macho64Header       = 32
	buildVersionSize    = 24
	segment64Size       = 72
	section64Size       = 80
)

// stampPlatform writes LC_BUILD_VERSION for the iOS device, or the
// simulator when goarch is amd64. A cgo-free ios link does not emit one.
// An existing command is overwritten. The header pad must fit a new one.
func stampPlatform(path, goarch string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	hdr := make([]byte, macho64Header)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return causePlatform(err)
	}
	if magic := binary.LittleEndian.Uint32(hdr[:4]); magic != 0xFEEDFACF {
		return fmt.Errorf("%w: magic %08x", ErrIOSPlatform, magic)
	}
	ncmds := binary.LittleEndian.Uint32(hdr[16:20])
	sizeofcmds := binary.LittleEndian.Uint32(hdr[20:24])
	cmds := make([]byte, sizeofcmds)
	if _, err := io.ReadFull(f, cmds); err != nil {
		return causePlatform(err)
	}

	platform := uint32(platformIOS)
	if goarch == "amd64" {
		platform = platformIOSSim
	}
	// Go's darwin floor. The bundle's MinimumOSVersion matches.
	version := uint32(13 << 16)

	if at, ok := findBuildVersion(cmds); ok {
		binary.LittleEndian.PutUint32(cmds[at+8:], platform)
		binary.LittleEndian.PutUint32(cmds[at+12:], version)
		binary.LittleEndian.PutUint32(cmds[at+16:], version)
		if _, err := f.WriteAt(cmds[at:at+buildVersionSize], int64(macho64Header+at)); err != nil {
			return err
		}
		return nil
	}

	content, ok := firstSectionOffset(cmds)
	need := uint64(macho64Header) + uint64(sizeofcmds) + buildVersionSize
	if !ok || need > content {
		return fmt.Errorf("%w: header %d section %d ncmds %d", ErrIOSPlatform, need, content, ncmds)
	}
	cmd := make([]byte, buildVersionSize)
	binary.LittleEndian.PutUint32(cmd[0:], loadCmdBuildVersion)
	binary.LittleEndian.PutUint32(cmd[4:], buildVersionSize)
	binary.LittleEndian.PutUint32(cmd[8:], platform)
	binary.LittleEndian.PutUint32(cmd[12:], version)
	binary.LittleEndian.PutUint32(cmd[16:], version)
	if _, err := f.WriteAt(cmd, int64(macho64Header)+int64(sizeofcmds)); err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(hdr[16:20], ncmds+1)
	binary.LittleEndian.PutUint32(hdr[20:24], sizeofcmds+buildVersionSize)
	if _, err := f.WriteAt(hdr, 0); err != nil {
		return err
	}
	return nil
}

func findBuildVersion(cmds []byte) (int, bool) {
	off := 0
	for off+8 <= len(cmds) {
		cmd := binary.LittleEndian.Uint32(cmds[off:])
		size := int(binary.LittleEndian.Uint32(cmds[off+4:]))
		if size < 8 || off+size > len(cmds) {
			return 0, false
		}
		if cmd == loadCmdBuildVersion && size >= buildVersionSize {
			return off, true
		}
		off += size
	}
	return 0, false
}

func firstSectionOffset(cmds []byte) (uint64, bool) {
	var min uint64
	found := false
	off := 0
	for off+8 <= len(cmds) {
		cmd := binary.LittleEndian.Uint32(cmds[off:])
		size := int(binary.LittleEndian.Uint32(cmds[off+4:]))
		if size < 8 || off+size > len(cmds) {
			return 0, false
		}
		if cmd == loadCmdSegment64 && size >= segment64Size {
			nsects := binary.LittleEndian.Uint32(cmds[off+64:])
			sec := off + segment64Size
			for range nsects {
				if sec+section64Size > len(cmds) {
					break
				}
				sz := binary.LittleEndian.Uint64(cmds[sec+40:])
				so := binary.LittleEndian.Uint32(cmds[sec+48:])
				// Zerofill sections have a size and a zero file offset.
				if sz > 0 && so > 0 && (!found || uint64(so) < min) {
					min = uint64(so)
					found = true
				}
				sec += section64Size
			}
		}
		off += size
	}
	return min, found
}

func causePlatform(err error) error {
	return fmt.Errorf("%w: %w", ErrIOSPlatform, err)
}
