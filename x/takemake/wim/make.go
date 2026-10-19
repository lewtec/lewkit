package wim

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/xml"
	"io/fs"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

const (
	headerSize      = 212
	version         = 0x10D00
	compressionSize = 0x8000
	streamSize      = 50

	attrDir     = 0x10
	attrArchive = 0x20
	attrReparse = 0x400

	resMeta = 2
	resComp = 4

	hdrReadOnly  = 1 << 2
	reparseTag   = 0xA000000C
	securityNone = 0xffffffff
	modeMark     = 1
)

type node struct {
	name     string
	entry    *Entry
	children []*node
	listing  int
	hash     [20]byte
	data     []byte
}

type blob struct {
	hash [20]byte
	data []byte
	refs uint32
	off  int64
}

func makeImage(f *File) ([]byte, error) {
	root, err := tree(f.Entries)
	if err != nil {
		return nil, err
	}
	blobs := map[[20]byte]*blob{}
	if err := stamp(root, blobs); err != nil {
		return nil, err
	}
	meta, err := metadata(root)
	if err != nil {
		return nil, err
	}
	when := defaultTime
	var walk func(*node)
	walk = func(n *node) {
		if n.entry != nil && n.entry.Time.After(when) {
			when = n.entry.Time
		}
		for _, child := range n.children {
			walk(child)
		}
	}
	walk(root)
	name := f.ImageName
	if name == "" {
		name = defaultName
	}
	xmlBytes, err := xmlDocument(name, when)
	if err != nil {
		return nil, err
	}

	ordered := make([]*blob, 0, len(blobs))
	for _, item := range blobs {
		ordered = append(ordered, item)
	}
	slices.SortFunc(ordered, func(a, b *blob) int {
		return bytes.Compare(a.hash[:], b.hash[:])
	})

	off := int64(headerSize)
	var payload []byte
	for _, item := range ordered {
		if len(item.data) == 0 {
			continue
		}
		item.off = off
		payload = append(payload, item.data...)
		off += int64(len(item.data))
	}
	for _, item := range ordered {
		if len(item.data) == 0 {
			item.off = off
		}
	}

	metaOff := off
	xmlOff := metaOff + int64(len(meta))
	tableOff := xmlOff + int64(len(xmlBytes))
	table := offsetTable(meta, metaOff, ordered)
	image := make([]byte, headerSize+len(payload)+len(meta)+len(xmlBytes)+len(table))
	copy(image[headerSize:], payload)
	copy(image[metaOff:], meta)
	copy(image[xmlOff:], xmlBytes)
	copy(image[tableOff:], table)
	copy(image[:headerSize], header(tableOff, int64(len(table)), xmlOff, int64(len(xmlBytes))))
	return image, nil
}

func tree(entries []*Entry) (*node, error) {
	root := &node{name: "", entry: &Entry{Mode: fs.ModeDir | dirMode, Time: defaultTime}}
	index := map[string]*node{"": root}
	for _, entry := range entries {
		if entry == nil {
			return nil, ErrName
		}
		name, err := cleanName(entry.Name)
		if err != nil {
			return nil, err
		}
		if err := checkEntry(entry); err != nil {
			return nil, err
		}
		parts := strings.Split(name, "/")
		parent := root
		built := ""
		for i, part := range parts {
			if built == "" {
				built = part
			} else {
				built += "/" + part
			}
			child := index[built]
			last := i == len(parts)-1
			if child == nil {
				child = &node{name: part}
				if last {
					child.entry = entry
				} else {
					child.entry = &Entry{Name: built, Mode: fs.ModeDir | dirMode, Time: defaultTime}
				}
				parent.children = append(parent.children, child)
				index[built] = child
			} else if last {
				child.entry = entry
			}
			if !last && !child.entry.isDir() {
				return nil, ErrKind
			}
			parent = child
		}
	}
	for _, n := range index {
		slices.SortFunc(n.children, func(a, b *node) int {
			return strings.Compare(a.name, b.name)
		})
		if len(n.children) > 0 && (n.entry == nil || !n.entry.isDir()) {
			return nil, ErrKind
		}
	}
	return root, nil
}

func checkEntry(entry *Entry) error {
	dir := entry.Mode&fs.ModeDir != 0
	link := entry.Mode&fs.ModeSymlink != 0
	switch {
	case dir && link:
		return ErrKind
	case dir && (len(entry.Data) > 0 || entry.Target != ""):
		return ErrKind
	case link && (entry.Target == "" || len(entry.Data) > 0):
		return ErrKind
	case !dir && !link && entry.Target != "":
		return ErrKind
	}
	return nil
}

func stamp(n *node, blobs map[[20]byte]*blob) error {
	if n.entry != nil && n.entry.isSymlink() {
		raw, err := reparseSymlink(n.entry.Target)
		if err != nil {
			return err
		}
		n.data = raw
		n.hash = sha1.Sum(raw)
		touch(blobs, n.hash, raw)
	} else if n.entry != nil && n.entry.isFile() {
		n.data = n.entry.Data
		n.hash = sha1.Sum(n.data)
		touch(blobs, n.hash, n.data)
	}
	for _, child := range n.children {
		if err := stamp(child, blobs); err != nil {
			return err
		}
	}
	return nil
}

func touch(blobs map[[20]byte]*blob, hash [20]byte, data []byte) {
	item := blobs[hash]
	if item == nil {
		item = &blob{hash: hash, data: append([]byte(nil), data...)}
		blobs[hash] = item
	}
	item.refs++
}

func metadata(root *node) ([]byte, error) {
	dirs := directories(root)
	pos := 8 + recordSize("") + 8
	for _, dir := range dirs {
		dir.listing = pos
		for _, child := range dir.children {
			pos += recordSize(child.name)
		}
		pos += 8
	}
	buf := make([]byte, pos)
	binary.LittleEndian.PutUint32(buf[0:4], 8)
	if err := writeRecord(buf[8:], "", fs.ModeDir|dirMode, defaultTime, int64(root.listing), 0, [20]byte{}, true); err != nil {
		return nil, err
	}
	for _, dir := range dirs {
		at := dir.listing
		for _, child := range dir.children {
			mode := fs.FileMode(defaultMode)
			when := defaultTime
			if child.entry != nil {
				mode = child.entry.Mode
				when = wall(child.entry.Time)
			}
			sub := int64(0)
			if child.entry.isDir() {
				sub = int64(child.listing)
			}
			if err := writeRecord(buf[at:], child.name, mode, when, sub, reparseOf(child), child.hash, child.entry.isDir()); err != nil {
				return nil, err
			}
			at += recordSize(child.name)
		}
	}
	return buf, nil
}

func directories(root *node) []*node {
	var out []*node
	var walk func(*node)
	walk = func(n *node) {
		if n.entry.isDir() {
			out = append(out, n)
		}
		for _, child := range n.children {
			walk(child)
		}
	}
	walk(root)
	return out
}

func reparseOf(n *node) int64 {
	if n.entry.isSymlink() {
		return reparseTag
	}
	return 0
}

func recordSize(name string) int {
	raw := 8 + 94 + utf16Bytes(name) + 2
	if raw%8 != 0 {
		raw += 8 - raw%8
	}
	return raw
}

func utf16Bytes(name string) int {
	return len(utf16.Encode([]rune(name))) * 2
}

func writeRecord(dst []byte, name string, mode fs.FileMode, when time.Time, subdir, reparse int64, hash [20]byte, dir bool) error {
	size := recordSize(name)
	if size > len(dst) {
		return ErrFormat
	}
	nameLen := utf16Bytes(name)
	if nameLen > 0xffff {
		return ErrSize
	}
	binary.LittleEndian.PutUint64(dst[0:8], uint64(size))
	attr := uint32(attrArchive)
	stored := hash
	link := int64(0)
	if dir {
		attr = attrDir
		stored = [20]byte{}
	} else if mode&fs.ModeSymlink != 0 {
		attr = attrReparse
		link = reparse
	}
	binary.LittleEndian.PutUint32(dst[8:12], attr)
	binary.LittleEndian.PutUint32(dst[12:16], securityNone)
	binary.LittleEndian.PutUint64(dst[16:24], uint64(subdir))
	binary.LittleEndian.PutUint64(dst[24:32], uint64(uint32(mode)))
	binary.LittleEndian.PutUint64(dst[32:40], modeMark)
	putFiletime(dst[40:48], when)
	putFiletime(dst[48:56], when)
	putFiletime(dst[56:64], when)
	copy(dst[64:84], stored[:])
	binary.LittleEndian.PutUint64(dst[88:96], uint64(link))
	binary.LittleEndian.PutUint16(dst[100:102], uint16(nameLen))
	units := utf16.Encode([]rune(name))
	for i, unit := range units {
		binary.LittleEndian.PutUint16(dst[102+i*2:], unit)
	}
	return nil
}

func putFiletime(dst []byte, when time.Time) {
	ticks := wall(when).Unix()*10_000_000 + 116444736000000000
	binary.LittleEndian.PutUint32(dst[0:4], uint32(ticks))
	binary.LittleEndian.PutUint32(dst[4:8], uint32(uint64(ticks)>>32))
}

func reparseSymlink(target string) ([]byte, error) {
	units := utf16.Encode([]rune(target))
	nbytes := len(units) * 2
	dataLen := 12 + nbytes*2
	if dataLen > 0xffff {
		return nil, ErrSize
	}
	buf := make([]byte, 8+dataLen)
	binary.LittleEndian.PutUint32(buf[0:4], reparseTag)
	binary.LittleEndian.PutUint16(buf[4:6], uint16(dataLen))
	binary.LittleEndian.PutUint16(buf[10:12], uint16(nbytes))
	binary.LittleEndian.PutUint16(buf[12:14], uint16(nbytes))
	binary.LittleEndian.PutUint16(buf[14:16], uint16(nbytes))
	binary.LittleEndian.PutUint32(buf[16:20], 1)
	writeUnits(buf[20:], units)
	writeUnits(buf[20+nbytes:], units)
	return buf, nil
}

func writeUnits(dst []byte, units []uint16) {
	for i, unit := range units {
		binary.LittleEndian.PutUint16(dst[i*2:], unit)
	}
}

func offsetTable(meta []byte, metaOff int64, blobs []*blob) []byte {
	table := make([]byte, 0, streamSize*(1+len(blobs)))
	sum := sha1.Sum(meta)
	table = appendStream(table, resMeta, metaOff, int64(len(meta)), 1, sum)
	for _, item := range blobs {
		table = appendStream(table, 0, item.off, int64(len(item.data)), item.refs, item.hash)
	}
	return table
}

func appendStream(dst []byte, flags byte, off, size int64, refs uint32, hash [20]byte) []byte {
	var word uint64
	if size > 0 {
		word = uint64(size)
	}
	word |= uint64(flags) << 56
	var buf [streamSize]byte
	binary.LittleEndian.PutUint64(buf[0:8], word)
	binary.LittleEndian.PutUint64(buf[8:16], uint64(off))
	binary.LittleEndian.PutUint64(buf[16:24], uint64(size))
	binary.LittleEndian.PutUint16(buf[24:26], 1)
	binary.LittleEndian.PutUint32(buf[26:30], refs)
	copy(buf[30:], hash[:])
	return append(dst, buf[:]...)
}

func header(tableOff, tableLen, xmlOff, xmlLen int64) []byte {
	buf := make([]byte, headerSize)
	copy(buf[:8], []byte{'M', 'S', 'W', 'I', 'M', 0, 0, 0})
	binary.LittleEndian.PutUint32(buf[8:12], headerSize)
	binary.LittleEndian.PutUint32(buf[12:16], version)
	binary.LittleEndian.PutUint32(buf[16:20], hdrReadOnly)
	binary.LittleEndian.PutUint32(buf[20:24], compressionSize)
	binary.LittleEndian.PutUint16(buf[40:42], 1)
	binary.LittleEndian.PutUint16(buf[42:44], 1)
	binary.LittleEndian.PutUint32(buf[44:48], 1)
	putResource(buf[48:72], 0, tableOff, tableLen)
	putResource(buf[72:96], 0, xmlOff, xmlLen)
	return buf
}

func putResource(dst []byte, flags byte, off, size int64) {
	word := uint64(size) | uint64(flags)<<56
	binary.LittleEndian.PutUint64(dst[0:8], word)
	binary.LittleEndian.PutUint64(dst[8:16], uint64(off))
	binary.LittleEndian.PutUint64(dst[16:24], uint64(size))
}

func xmlDocument(name string, when time.Time) ([]byte, error) {
	lo, hi := filetimeParts(when)
	stamp := "<LOWPART>" + strconv.FormatUint(uint64(lo), 10) + "</LOWPART><HIGHPART>" + strconv.FormatUint(uint64(hi), 10) + "</HIGHPART>"
	var text bytes.Buffer
	if err := xml.EscapeText(&text, []byte(name)); err != nil {
		return nil, ErrName
	}
	body := "<WIM><IMAGE INDEX=\"1\"><NAME>" + text.String() + "</NAME><CREATIONTIME>" + stamp + "</CREATIONTIME><LASTMODIFICATIONTIME>" + stamp + "</LASTMODIFICATIONTIME></IMAGE></WIM>"
	units := utf16.Encode([]rune(body))
	out := make([]byte, 2+len(units)*2)
	binary.LittleEndian.PutUint16(out[0:2], 0xFEFF)
	writeUnits(out[2:], units)
	return out, nil
}

func filetimeParts(when time.Time) (uint32, uint32) {
	ticks := wall(when).Unix()*10_000_000 + 116444736000000000
	return uint32(ticks), uint32(uint64(ticks) >> 32)
}

func (e *Entry) isDir() bool {
	return e != nil && e.Mode&fs.ModeDir != 0
}

func (e *Entry) isSymlink() bool {
	return e != nil && e.Mode&fs.ModeSymlink != 0 && !e.isDir()
}

func (e *Entry) isFile() bool {
	return e != nil && !e.isDir() && !e.isSymlink()
}
