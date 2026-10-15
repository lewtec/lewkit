package wim

import (
	"encoding/binary"
	"encoding/xml"
	"io/fs"
	"strings"
	"time"
	"unicode/utf16"
)

const (
	supportedHdr = hdrReadOnly | (1 << 7) | (1 << 1) | (1 << 18)
	supportedRes = resMeta | resComp
)

type dentry struct {
	name     string
	attr     uint32
	subdir   int64
	mode     fs.FileMode
	marked   bool
	when     time.Time
	hash     [20]byte
	reparse  uint32
	streams  int
	streamAt int
	size     int
}

func take(raw []byte) (*File, error) {
	if len(raw) < headerSize || string(raw[:8]) != "MSWIM\x00\x00\x00" {
		return nil, ErrFormat
	}
	if binary.LittleEndian.Uint32(raw[12:16]) != version {
		return nil, ErrFormat
	}
	flags := binary.LittleEndian.Uint32(raw[16:20])
	if flags&^supportedHdr != 0 {
		return nil, ErrFormat
	}
	if binary.LittleEndian.Uint32(raw[20:24]) != compressionSize {
		return nil, ErrFormat
	}
	if binary.LittleEndian.Uint16(raw[42:44]) != 1 || binary.LittleEndian.Uint32(raw[44:48]) != 1 {
		return nil, ErrFormat
	}
	tableWord, tableOff, tableOrig := resourceAt(raw[48:72])
	xmlWord, xmlOff, xmlOrig := resourceAt(raw[72:96])
	table, err := sliceResource(raw, tableWord, tableOff, tableOrig)
	if err != nil {
		return nil, err
	}
	xmlBytes, err := sliceResource(raw, xmlWord, xmlOff, xmlOrig)
	if err != nil {
		return nil, err
	}
	if len(table) == 0 || len(table)%streamSize != 0 {
		return nil, ErrFormat
	}
	files := map[[20]byte][]byte{}
	var meta []byte
	metas := 0
	for pos := 0; pos < len(table); pos += streamSize {
		word, off, orig := resourceAt(table[pos : pos+24])
		var hash [20]byte
		copy(hash[:], table[pos+30:pos+50])
		body, err := sliceResource(raw, word, off, orig)
		if err != nil {
			return nil, err
		}
		if byte(word>>56)&resMeta != 0 {
			metas++
			meta = body
			continue
		}
		files[hash] = body
	}
	if metas != 1 {
		return nil, ErrFormat
	}
	name := defaultName
	if len(xmlBytes) > 0 {
		text, err := decodeUTF16(xmlBytes)
		if err != nil {
			return nil, err
		}
		name, err = imageName(text)
		if err != nil {
			return nil, err
		}
	}
	rootAt, err := securityEnd(meta)
	if err != nil {
		return nil, err
	}
	root, err := readListing(meta, rootAt)
	if err != nil {
		return nil, err
	}
	if len(root) != 1 || root[0].attr&attrDir == 0 || root[0].attr&attrReparse != 0 {
		return nil, ErrFormat
	}
	img := &wimImage{meta: meta, files: files}
	var entries []*Entry
	if err := img.walk(root[0].subdir, "", &entries); err != nil {
		return nil, err
	}
	return &File{ImageName: name, Entries: entries, cur: -1}, nil
}

type wimImage struct {
	meta  []byte
	files map[[20]byte][]byte
}

func (m *wimImage) walk(at int64, prefix string, out *[]*Entry) error {
	items, err := readListing(m.meta, int(at))
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.name == "" || strings.Contains(item.name, "/") || strings.Contains(item.name, `\`) {
			return ErrFormat
		}
		full := item.name
		if prefix != "" {
			full = prefix + "/" + item.name
		}
		hash := item.hash
		if item.streams > 0 {
			streams, err := readStreams(m.meta, item.streamAt, item.streams)
			if err != nil {
				return err
			}
			if len(streams) == 0 || streams[0].name != "" {
				return ErrInode
			}
			hash = streams[0].hash
		}
		entry := &Entry{Name: full, Time: item.when}
		switch {
		case item.attr&attrReparse != 0:
			if item.reparse != reparseTag {
				return ErrInode
			}
			target, err := m.symlink(hash)
			if err != nil {
				return err
			}
			entry.Target = target
			entry.Mode = fs.ModeSymlink | linkMode
		case item.attr&attrDir != 0:
			entry.Mode = fs.ModeDir | dirMode
		default:
			data, err := m.fileBytes(hash)
			if err != nil {
				return err
			}
			entry.Data = data
			entry.Mode = defaultMode
		}
		if item.marked {
			entry.Mode = item.mode
		}
		*out = append(*out, entry)
		if item.attr&attrDir != 0 && item.attr&attrReparse == 0 {
			if err := m.walk(item.subdir, full, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *wimImage) fileBytes(hash [20]byte) ([]byte, error) {
	if hash == ([20]byte{}) {
		return nil, nil
	}
	data, ok := m.files[hash]
	if !ok {
		return nil, ErrFormat
	}
	if len(data) == 0 {
		return nil, nil
	}
	return append([]byte(nil), data...), nil
}

func (m *wimImage) symlink(hash [20]byte) (string, error) {
	data, err := m.fileBytes(hash)
	if err != nil {
		return "", err
	}
	if len(data) < 20 {
		return "", ErrFormat
	}
	if binary.LittleEndian.Uint32(data[0:4]) != reparseTag {
		return "", ErrInode
	}
	dataLen := int(binary.LittleEndian.Uint16(data[4:6]))
	if dataLen < 12 || 8+dataLen > len(data) {
		return "", ErrFormat
	}
	substOff := int(binary.LittleEndian.Uint16(data[8:10]))
	substLen := int(binary.LittleEndian.Uint16(data[10:12]))
	printOff := int(binary.LittleEndian.Uint16(data[12:14]))
	printLen := int(binary.LittleEndian.Uint16(data[14:16]))
	path := data[20 : 8+dataLen]
	if printLen > 0 {
		if printOff < 0 || printLen < 0 || printOff+printLen > len(path) {
			return "", ErrFormat
		}
		return decodeUTF16(path[printOff : printOff+printLen])
	}
	if substOff < 0 || substLen < 0 || substOff+substLen > len(path) {
		return "", ErrFormat
	}
	target, err := decodeUTF16(path[substOff : substOff+substLen])
	if err != nil {
		return "", err
	}
	target = strings.TrimPrefix(target, `\??\`)
	target = strings.TrimPrefix(target, `\\?\`)
	return target, nil
}

func readListing(meta []byte, at int) ([]dentry, error) {
	if at < 0 || at > len(meta) {
		return nil, ErrFormat
	}
	var out []dentry
	for at < len(meta) {
		if at+8 > len(meta) {
			return nil, ErrFormat
		}
		size := int(binary.LittleEndian.Uint64(meta[at : at+8]))
		if size == 0 {
			return out, nil
		}
		if size < 102 || at+size > len(meta) {
			return nil, ErrFormat
		}
		item, err := parseDentry(meta[at : at+size])
		if err != nil {
			return nil, err
		}
		item.streamAt = at + size
		out = append(out, item)
		at += size
		if item.streams > 0 {
			end, err := skipStreams(meta, item.streamAt, item.streams)
			if err != nil {
				return nil, err
			}
			at = end
		}
	}
	return nil, ErrFormat
}

func parseDentry(rec []byte) (dentry, error) {
	nameLen := int(binary.LittleEndian.Uint16(rec[100:102]))
	shortLen := int(binary.LittleEndian.Uint16(rec[98:100]))
	names := nameLen + 2 + shortLen
	if names < 2 || 102+names > len(rec) || nameLen%2 != 0 || shortLen%2 != 0 {
		return dentry{}, ErrFormat
	}
	item := dentry{
		attr:    binary.LittleEndian.Uint32(rec[8:12]),
		subdir:  int64(binary.LittleEndian.Uint64(rec[16:24])),
		when:    fromFiletime(rec[56:64]),
		reparse: uint32(binary.LittleEndian.Uint64(rec[88:96])),
		streams: int(binary.LittleEndian.Uint16(rec[96:98])),
		size:    len(rec),
	}
	copy(item.hash[:], rec[64:84])
	if binary.LittleEndian.Uint64(rec[32:40]) == modeMark {
		item.marked = true
		item.mode = fs.FileMode(uint32(binary.LittleEndian.Uint64(rec[24:32])))
	}
	if nameLen > 0 {
		var err error
		item.name, err = decodeUTF16(rec[102 : 102+nameLen])
		if err != nil {
			return dentry{}, err
		}
	}
	if item.attr&attrDir != 0 && item.attr&attrReparse == 0 && item.subdir == 0 {
		return dentry{}, ErrFormat
	}
	if (item.attr&attrDir == 0 || item.attr&attrReparse != 0) && item.subdir != 0 {
		return dentry{}, ErrFormat
	}
	return item, nil
}

type stream struct {
	name string
	hash [20]byte
}

func readStreams(meta []byte, at, count int) ([]stream, error) {
	var out []stream
	for range count {
		item, next, err := parseStream(meta, at)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
		at = next
	}
	return out, nil
}

func skipStreams(meta []byte, at, count int) (int, error) {
	for range count {
		_, next, err := parseStream(meta, at)
		if err != nil {
			return 0, err
		}
		at = next
	}
	return at, nil
}

func parseStream(meta []byte, at int) (stream, int, error) {
	if at < 0 || at+8 > len(meta) {
		return stream{}, 0, ErrFormat
	}
	size := int(binary.LittleEndian.Uint64(meta[at : at+8]))
	if size < 38 || at+size > len(meta) {
		return stream{}, 0, ErrFormat
	}
	rec := meta[at : at+size]
	nameLen := int(int16(binary.LittleEndian.Uint16(rec[36:38])))
	if nameLen < 0 || nameLen%2 != 0 || 38+nameLen > len(rec) {
		return stream{}, 0, ErrFormat
	}
	var item stream
	copy(item.hash[:], rec[16:36])
	if nameLen > 0 {
		name, err := decodeUTF16(rec[38 : 38+nameLen])
		if err != nil {
			return stream{}, 0, err
		}
		item.name = name
	}
	return item, at + size, nil
}

func securityEnd(meta []byte) (int, error) {
	if len(meta) < 8 {
		return 0, ErrFormat
	}
	total := int(binary.LittleEndian.Uint32(meta[0:4]))
	num := int(binary.LittleEndian.Uint32(meta[4:8]))
	if total < 8 || num < 0 {
		return 0, ErrFormat
	}
	n := 8
	if n+num*8 > len(meta) {
		return 0, ErrFormat
	}
	sizes := make([]int, num)
	for i := range sizes {
		sizes[i] = int(binary.LittleEndian.Uint64(meta[n : n+8]))
		n += 8
	}
	for _, size := range sizes {
		if size < 0 || n+size > len(meta) {
			return 0, ErrFormat
		}
		n += size
	}
	aligned := (total + 7) &^ 7
	if n > aligned || aligned > len(meta) {
		return 0, ErrFormat
	}
	return aligned, nil
}

func resourceAt(b []byte) (uint64, int64, int64) {
	return binary.LittleEndian.Uint64(b[0:8]), int64(binary.LittleEndian.Uint64(b[8:16])), int64(binary.LittleEndian.Uint64(b[16:24]))
}

func sliceResource(raw []byte, word uint64, off, orig int64) ([]byte, error) {
	flags := byte(word >> 56)
	if flags&^supportedRes != 0 {
		return nil, ErrFormat
	}
	if flags&resComp != 0 {
		return nil, ErrCompression
	}
	size := int64(word & 0x00ffffffffffffff)
	if size != orig || off < 0 || size < 0 {
		return nil, ErrFormat
	}
	if off > int64(len(raw)) || size > int64(len(raw))-off {
		return nil, ErrFormat
	}
	return raw[off : off+size], nil
}

func imageName(text string) (string, error) {
	type image struct {
		Name string `xml:"NAME"`
	}
	type doc struct {
		Image []image `xml:"IMAGE"`
	}
	var parsed doc
	if err := xml.Unmarshal([]byte(text), &parsed); err != nil || len(parsed.Image) != 1 {
		return "", ErrFormat
	}
	if parsed.Image[0].Name == "" {
		return "", ErrFormat
	}
	return parsed.Image[0].Name, nil
}

func decodeUTF16(b []byte) (string, error) {
	if len(b)%2 != 0 {
		return "", ErrFormat
	}
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	if len(units) > 0 && units[0] == 0xFEFF {
		units = units[1:]
	}
	return string(utf16.Decode(units)), nil
}

func fromFiletime(b []byte) time.Time {
	ticks := int64(binary.LittleEndian.Uint32(b[4:8]))<<32 | int64(binary.LittleEndian.Uint32(b[0:4]))
	ticks -= 116444736000000000
	sec := ticks / 10_000_000
	nsec := (ticks % 10_000_000) * 100
	return time.Unix(sec, nsec).UTC()
}
