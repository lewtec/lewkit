//go:build linux

package native

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var nixStoreBin = regexp.MustCompile(`/nix/store/[^/\s'"]+/bin/(?:\.[^/\s'"]+|[A-Za-z0-9._+-]+)`)

// runpathsOf reads DT_RUNPATH from path and from the ELF a Nix wrapper execs.
func runpathsOf(path string) []string {
	var dirs []string
	seen := map[string]bool{}
	for _, bin := range elfCandidates(path) {
		for _, dir := range elfRunpath(bin) {
			if dir == "" || seen[dir] {
				continue
			}
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// elfCandidates is path, its symlink target, a sibling .name-wrapped binary,
// and absolute /nix/store bin paths named by a wrapper script.
func elfCandidates(path string) []string {
	var out []string
	seen := map[string]bool{}
	var add func(string)
	add = func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		info, err := os.Stat(name)
		if err != nil || info.IsDir() {
			return
		}
		out = append(out, name)
		dir := filepath.Dir(name)
		base := filepath.Base(name)
		wrapped := filepath.Join(dir, "."+base+"-wrapped")
		if !seen[wrapped] {
			if info, err := os.Stat(wrapped); err == nil && !info.IsDir() {
				add(wrapped)
			}
		}
		data, err := os.ReadFile(name)
		if err != nil || len(data) < 4 || bytes.Equal(data[:4], []byte{0x7f, 'E', 'L', 'F'}) {
			return
		}
		if len(data) > 1<<20 {
			data = data[:1<<20]
		}
		for _, bin := range nixStoreBin.FindAll(data, -1) {
			add(string(bin))
		}
	}
	add(path)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		add(resolved)
	}
	return out
}

// elfRunpath reads DT_RUNPATH, or DT_RPATH when RUNPATH is absent.
// $ORIGIN is the directory that contains path.
func elfRunpath(path string) []string {
	dyn, ok := openElfDyn(path)
	if !ok {
		return nil
	}
	defer dyn.file.Close()
	which, ok := dyn.runOff, dyn.hasRun
	if !ok && dyn.hasR {
		which, ok = dyn.rpathOff, true
	}
	if !ok {
		return nil
	}
	return elfSearchPath(dyn, which, filepath.Dir(path))
}

// elfNeeded reads DT_NEEDED sonames. The names are not paths.
func elfNeeded(path string) []string {
	dyn, ok := openElfDyn(path)
	if !ok || len(dyn.needed) == 0 {
		if ok {
			dyn.file.Close()
		}
		return nil
	}
	defer dyn.file.Close()
	strFile, ok := elfVaddr(dyn.loads, dyn.strtab)
	if !ok {
		return nil
	}
	var names []string
	for _, off := range dyn.needed {
		text, ok := elfCString(dyn.file, strFile+off)
		if !ok || text == "" {
			continue
		}
		names = append(names, text)
	}
	return names
}

type elfDyn struct {
	file     *os.File
	loads    []elfSegment
	strtab   uint64
	needed   []uint64
	runOff   uint64
	rpathOff uint64
	hasRun   bool
	hasR     bool
}

func openElfDyn(path string) (*elfDyn, bool) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	dyn := &elfDyn{file: file}
	var hdr [64]byte
	if _, err := io.ReadFull(file, hdr[:]); err != nil {
		file.Close()
		return nil, false
	}
	if !bytes.Equal(hdr[:4], []byte{0x7f, 'E', 'L', 'F'}) || hdr[4] != 2 || hdr[5] != 1 {
		file.Close()
		return nil, false
	}
	phoff := binary.LittleEndian.Uint64(hdr[32:])
	phentsize := binary.LittleEndian.Uint16(hdr[54:])
	phnum := binary.LittleEndian.Uint16(hdr[56:])
	if phentsize < 56 || phnum == 0 || phnum > 64 {
		file.Close()
		return nil, false
	}
	table := make([]byte, int(phentsize)*int(phnum))
	if _, err := file.ReadAt(table, int64(phoff)); err != nil {
		file.Close()
		return nil, false
	}
	var dynOff, dynSize uint64
	var hasDyn bool
	for i := range int(phnum) {
		ent := table[i*int(phentsize):]
		switch binary.LittleEndian.Uint32(ent[0:]) {
		case 1: // PT_LOAD
			dyn.loads = append(dyn.loads, elfSegment{
				off:   binary.LittleEndian.Uint64(ent[8:]),
				vaddr: binary.LittleEndian.Uint64(ent[16:]),
				size:  binary.LittleEndian.Uint64(ent[32:]),
			})
		case 2: // PT_DYNAMIC
			dynOff = binary.LittleEndian.Uint64(ent[8:])
			dynSize = binary.LittleEndian.Uint64(ent[32:])
			hasDyn = true
		}
	}
	if !hasDyn || dynSize < 16 || dynSize > 1<<20 {
		file.Close()
		return nil, false
	}
	raw := make([]byte, dynSize)
	if _, err := file.ReadAt(raw, int64(dynOff)); err != nil {
		file.Close()
		return nil, false
	}
	for off := 0; off+16 <= len(raw); off += 16 {
		tag := binary.LittleEndian.Uint64(raw[off:])
		val := binary.LittleEndian.Uint64(raw[off+8:])
		switch tag {
		case 0: // DT_NULL
			off = len(raw)
		case 1: // DT_NEEDED
			dyn.needed = append(dyn.needed, val)
		case 5: // DT_STRTAB
			dyn.strtab = val
		case 15: // DT_RPATH
			dyn.rpathOff, dyn.hasR = val, true
		case 29: // DT_RUNPATH
			dyn.runOff, dyn.hasRun = val, true
		}
	}
	return dyn, true
}

func elfSearchPath(dyn *elfDyn, which uint64, origin string) []string {
	strFile, ok := elfVaddr(dyn.loads, dyn.strtab)
	if !ok {
		return nil
	}
	text, ok := elfCString(dyn.file, strFile+which)
	if !ok {
		return nil
	}
	var out []string
	for _, part := range strings.Split(text, ":") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		part = strings.ReplaceAll(part, "${ORIGIN}", origin)
		part = strings.ReplaceAll(part, "$ORIGIN", origin)
		part = filepath.Clean(part)
		if part != "" && part != "." {
			out = append(out, part)
		}
	}
	return out
}

type elfSegment struct {
	off, vaddr, size uint64
}

func elfVaddr(loads []elfSegment, vaddr uint64) (uint64, bool) {
	for _, seg := range loads {
		if seg.size == 0 || vaddr < seg.vaddr || vaddr >= seg.vaddr+seg.size {
			continue
		}
		return seg.off + (vaddr - seg.vaddr), true
	}
	return 0, false
}

func elfCString(file *os.File, off uint64) (string, bool) {
	buf := make([]byte, 4096)
	n, err := file.ReadAt(buf, int64(off))
	if n == 0 || (err != nil && err != io.EOF) {
		return "", false
	}
	buf = buf[:n]
	i := bytes.IndexByte(buf, 0)
	if i < 0 {
		return "", false
	}
	return string(buf[:i]), true
}
