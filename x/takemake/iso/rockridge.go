package iso

import (
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"sync"
)

// hostKeepsUnixMode reports whether os.Chmod sticks for a non-default mode.
// Windows reports 0666 for files and 0777 for directories, so Rock Ridge
// would otherwise record those instead of the modes on the entries.
func hostKeepsUnixMode() bool {
	return hostUnixMode()
}

var hostUnixMode = sync.OnceValue(func() bool {
	f, err := os.CreateTemp("", "lewkit-iso-mode-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	defer os.Remove(name)
	if err := os.Chmod(name, 0o640); err != nil {
		return false
	}
	st, err := os.Stat(name)
	if err != nil {
		return false
	}
	return st.Mode().Perm() == 0o640
})

// stampRockRidgeModes rewrites PX permission words to match entries.
// go-diskfs copies the host file mode into Rock Ridge, which drops Unix
// permissions on hosts that cannot store them.
func stampRockRidgeModes(raw []byte, entries []*Entry) error {
	if len(raw) < 17*sectorSize {
		return ErrFormat
	}
	pvd := 16 * sectorSize
	if string(raw[pvd+1:pvd+6]) != "CD001" {
		return ErrFormat
	}
	root := pvd + 156
	loc := binary.LittleEndian.Uint32(raw[root+2 : root+6])
	size := binary.LittleEndian.Uint32(raw[root+10 : root+14])
	want := make(map[string]fs.FileMode, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.Name == "" {
			continue
		}
		want[entry.Name] = entry.Mode
	}
	seen := map[string]struct{}{}
	if err := walkRockRidge(raw, loc, size, "", want, seen, map[uint32]struct{}{}); err != nil {
		return err
	}
	for name := range want {
		if _, ok := seen[name]; !ok {
			return fmt.Errorf("%w: rock ridge entry %s", ErrFormat, name)
		}
	}
	return nil
}

func walkRockRidge(raw []byte, loc, size uint32, prefix string, want map[string]fs.FileMode, seen map[string]struct{}, visiting map[uint32]struct{}) error {
	if _, ok := visiting[loc]; ok {
		return nil
	}
	visiting[loc] = struct{}{}
	if size == 0 || loc == 0 {
		return ErrFormat
	}
	start := int(loc) * sectorSize
	end := start + int(size)
	if start < 0 || end > len(raw) || end < start {
		return ErrFormat
	}
	for at := start; at < end; {
		if at >= len(raw) {
			return ErrFormat
		}
		n := int(raw[at])
		if n == 0 {
			next := (at + sectorSize) &^ (sectorSize - 1)
			if next <= at || next >= end {
				break
			}
			at = next
			continue
		}
		if at+n > end || n < 34 {
			return ErrFormat
		}
		nameLen := int(raw[at+32])
		if 33+nameLen > n {
			return ErrFormat
		}
		flags := raw[at+25]
		childLoc := binary.LittleEndian.Uint32(raw[at+2 : at+6])
		childSize := binary.LittleEndian.Uint32(raw[at+10 : at+14])
		sys := at + 33 + nameLen
		if nameLen%2 == 0 {
			sys++
		}
		name, px := suspName(raw, sys, at+n)
		if nameLen == 1 && (raw[at+33] == 0 || raw[at+33] == 1) {
			at += n
			continue
		}
		if name == "" || name == "." || name == ".." {
			at += n
			continue
		}
		full := name
		if prefix != "" {
			full = prefix + "/" + name
		}
		if mode, ok := want[full]; ok && px >= 0 {
			putRockMode(raw[px:], mode)
			seen[full] = struct{}{}
		}
		if flags&2 != 0 {
			if err := walkRockRidge(raw, childLoc, childSize, full, want, seen, visiting); err != nil {
				return err
			}
		}
		at += n
	}
	return nil
}

// suspName reads the Rock Ridge basename and the offset of its PX signature.
// A CE continuation is followed once.
func suspName(raw []byte, start, end int) (string, int) {
	name, px, cont := suspSpan(raw, start, end)
	if cont == nil {
		return name, px
	}
	more, morePX, _ := suspSpan(raw, cont[0], cont[1])
	if name == "" {
		name = more
	} else if more != "" {
		name += more
	}
	if px < 0 {
		px = morePX
	}
	return name, px
}

func suspSpan(raw []byte, start, end int) (string, int, *[2]int) {
	var name string
	px := -1
	var cont *[2]int
	for at := start; at+4 <= end && at < len(raw); {
		sig := string(raw[at : at+2])
		n := int(raw[at+2])
		if n < 4 || at+n > len(raw) {
			break
		}
		if at+n > end && sig != "CE" {
			break
		}
		switch sig {
		case "ST":
			return name, px, cont
		case "PX":
			if n >= 12 {
				px = at
			}
		case "NM":
			if n >= 5 && raw[at+4]&0x6 == 0 {
				name += string(raw[at+5 : at+n])
			}
		case "CE":
			if n >= 28 {
				block := binary.LittleEndian.Uint32(raw[at+4 : at+8])
				off := binary.LittleEndian.Uint32(raw[at+12 : at+16])
				size := binary.LittleEndian.Uint32(raw[at+20 : at+24])
				from := int(block)*sectorSize + int(off)
				to := from + int(size)
				if from >= 0 && to <= len(raw) && to >= from {
					cont = &[2]int{from, to}
				}
			}
		}
		at += n
	}
	return name, px, cont
}

func putRockMode(px []byte, mode fs.FileMode) {
	if len(px) < 12 {
		return
	}
	word := rockMode(mode)
	binary.LittleEndian.PutUint32(px[4:8], word)
	binary.BigEndian.PutUint32(px[8:12], word)
}

func rockMode(mode fs.FileMode) uint32 {
	word := uint32(mode & 0o777)
	if mode&fs.ModeSetuid != 0 {
		word |= 0o4000
	}
	if mode&fs.ModeSetgid != 0 {
		word |= 0o2000
	}
	if mode&fs.ModeSticky != 0 {
		word |= 0o1000
	}
	regular := true
	switch {
	case mode&fs.ModeSocket != 0:
		word |= 0o140000
		regular = false
	case mode&fs.ModeSymlink != 0:
		word |= 0o120000
		regular = false
	case mode&fs.ModeDevice != 0:
		regular = false
		if mode&fs.ModeCharDevice != 0 {
			word |= 0o20000
		} else {
			word |= 0o60000
		}
	case mode&fs.ModeDir != 0:
		word |= 0o40000
		regular = false
	case mode&fs.ModeNamedPipe != 0:
		word |= 0o10000
		regular = false
	}
	if regular {
		word |= 0o100000
	}
	return word
}
