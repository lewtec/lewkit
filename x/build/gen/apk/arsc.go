package apk

import (
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

// resources.arsc. One package, id 0x7f. The package name is the
// application id. The system resolves android:icon and android:label
// against that name. String values live in the table string pool.
// Entry names live in the package key pool.
const (
	chunkTable   = 0x0002
	chunkPackage = 0x0200
	chunkType    = 0x0201
	chunkSpec    = 0x0202
	arscNoEntry  = 0xffffffff

	arscFlagComplex = 0x0001
)

type arscBag struct {
	name uint32
	kind byte
	data uint32
}

type arscValue struct {
	bag    bool
	parent uint32
	items  []arscBag
	kind   byte
	data   uint32
}

type arscConfig struct {
	density uint16
	sdk     uint16
	values  []arscValue // parallel to the type's entries; absent is a zero kind with missing=true
	missing []bool
}

type arscEntry struct {
	name string
}

type arscType struct {
	id      byte
	name    string
	entries []arscEntry
	configs []arscConfig
}

type arscTable struct {
	name    string
	strings []string
	types   []arscType
}

func (t arscTable) encode() ([]byte, error) {
	global := encodeStringPool(t.strings)
	pkg, err := t.packageChunk()
	if err != nil {
		return nil, err
	}
	body := append(global, pkg...)
	out := make([]byte, 12+len(body))
	binary.LittleEndian.PutUint16(out[0:], chunkTable)
	binary.LittleEndian.PutUint16(out[2:], 12)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(out)))
	binary.LittleEndian.PutUint32(out[8:], 1)
	copy(out[12:], body)
	return out, nil
}

func (t arscTable) packageChunk() ([]byte, error) {
	typeNames := make([]string, len(t.types))
	var keyNames []string
	keyIndex := map[string]uint32{}
	for i, typ := range t.types {
		typeNames[i] = typ.name
		for _, ent := range typ.entries {
			if _, ok := keyIndex[ent.name]; ok {
				continue
			}
			keyIndex[ent.name] = uint32(len(keyNames))
			keyNames = append(keyNames, ent.name)
		}
	}
	typePool := encodeStringPool(typeNames)
	keyPool := encodeStringPool(keyNames)
	var kids []byte
	for _, typ := range t.types {
		spec, err := typ.specChunk()
		if err != nil {
			return nil, err
		}
		kids = append(kids, spec...)
		for _, cfg := range typ.configs {
			chunk, err := typ.typeChunk(cfg, keyIndex)
			if err != nil {
				return nil, err
			}
			kids = append(kids, chunk...)
		}
	}
	const headerSize = 288
	body := make([]byte, 0, headerSize+len(typePool)+len(keyPool)+len(kids))
	header := make([]byte, headerSize)
	size := headerSize + len(typePool) + len(keyPool) + len(kids)
	binary.LittleEndian.PutUint16(header[0:], chunkPackage)
	binary.LittleEndian.PutUint16(header[2:], headerSize)
	binary.LittleEndian.PutUint32(header[4:], uint32(size))
	binary.LittleEndian.PutUint32(header[8:], 0x7f)
	name := utf16.Encode([]rune(t.name))
	for i, c := range name {
		if 12+i*2+2 > 268 {
			break
		}
		binary.LittleEndian.PutUint16(header[12+i*2:], c)
	}
	binary.LittleEndian.PutUint32(header[268:], headerSize)
	binary.LittleEndian.PutUint32(header[272:], publicLast(len(t.types)))
	binary.LittleEndian.PutUint32(header[276:], uint32(headerSize+len(typePool)))
	binary.LittleEndian.PutUint32(header[280:], publicLast(len(keyNames)))
	body = append(body, header...)
	body = append(body, typePool...)
	body = append(body, keyPool...)
	body = append(body, kids...)
	return body, nil
}

func (t arscType) specChunk() ([]byte, error) {
	n := len(t.entries)
	size := 16 + 4*n
	buf := make([]byte, size)
	binary.LittleEndian.PutUint16(buf[0:], chunkSpec)
	binary.LittleEndian.PutUint16(buf[2:], 16)
	binary.LittleEndian.PutUint32(buf[4:], uint32(size))
	buf[8] = t.id
	binary.LittleEndian.PutUint32(buf[12:], uint32(n))
	return buf, nil
}

func (t arscType) typeChunk(cfg arscConfig, keys map[string]uint32) ([]byte, error) {
	if len(cfg.values) != len(t.entries) || len(cfg.missing) != len(t.entries) {
		return nil, errArscEntries(t.name)
	}
	n := len(t.entries)
	config := configBytes(cfg.density, cfg.sdk)
	headerSize := 20 + len(config)
	entriesStart := headerSize + 4*n
	var entries []byte
	offs := make([]uint32, n)
	for i := range t.entries {
		if cfg.missing[i] {
			offs[i] = arscNoEntry
			continue
		}
		offs[i] = uint32(len(entries))
		key := keys[t.entries[i].name]
		entries = append(entries, encodeValue(key, cfg.values[i])...)
	}
	if pad := (4 - len(entries)%4) % 4; pad != 0 {
		entries = append(entries, make([]byte, pad)...)
	}
	size := entriesStart + len(entries)
	buf := make([]byte, size)
	binary.LittleEndian.PutUint16(buf[0:], chunkType)
	binary.LittleEndian.PutUint16(buf[2:], uint16(headerSize))
	binary.LittleEndian.PutUint32(buf[4:], uint32(size))
	buf[8] = t.id
	binary.LittleEndian.PutUint32(buf[12:], uint32(n))
	binary.LittleEndian.PutUint32(buf[16:], uint32(entriesStart))
	copy(buf[20:], config)
	for i, off := range offs {
		binary.LittleEndian.PutUint32(buf[headerSize+4*i:], off)
	}
	copy(buf[entriesStart:], entries)
	return buf, nil
}

// publicLast is the last public type or key index stored in the package header.
func publicLast(n int) uint32 {
	if n == 0 {
		return 0
	}
	return uint32(n - 1)
}

func errArscEntries(name string) error {
	return fmt.Errorf("%w: %s entries", ErrAssembleAPK, name)
}

func encodeValue(key uint32, v arscValue) []byte {
	if !v.bag {
		buf := make([]byte, 16)
		binary.LittleEndian.PutUint16(buf[0:], 8)
		binary.LittleEndian.PutUint32(buf[4:], key)
		binary.LittleEndian.PutUint16(buf[8:], 8)
		buf[11] = v.kind
		binary.LittleEndian.PutUint32(buf[12:], v.data)
		return buf
	}
	buf := make([]byte, 16+12*len(v.items))
	binary.LittleEndian.PutUint16(buf[0:], 16)
	binary.LittleEndian.PutUint16(buf[2:], arscFlagComplex)
	binary.LittleEndian.PutUint32(buf[4:], key)
	binary.LittleEndian.PutUint32(buf[8:], v.parent)
	binary.LittleEndian.PutUint32(buf[12:], uint32(len(v.items)))
	off := 16
	for _, item := range v.items {
		binary.LittleEndian.PutUint32(buf[off:], item.name)
		binary.LittleEndian.PutUint16(buf[off+4:], 8)
		buf[off+7] = item.kind
		binary.LittleEndian.PutUint32(buf[off+8:], item.data)
		off += 12
	}
	return buf
}

func configBytes(density, sdk uint16) []byte {
	buf := make([]byte, 36)
	binary.LittleEndian.PutUint32(buf[0:], 36)
	binary.LittleEndian.PutUint16(buf[14:], density)
	binary.LittleEndian.PutUint16(buf[24:], sdk)
	return buf
}

func encodeStringPool(list []string) []byte {
	var data []byte
	offs := make([]uint32, len(list))
	for i, s := range list {
		offs[i] = uint32(len(data))
		data = appendUTF16(data, s)
	}
	headerSize := 28
	stringsStart := headerSize + 4*len(list)
	size := stringsStart + len(data)
	if pad := (4 - size%4) % 4; pad != 0 {
		data = append(data, make([]byte, pad)...)
		size += pad
	}
	buf := make([]byte, size)
	binary.LittleEndian.PutUint16(buf[0:], xmlChunkString)
	binary.LittleEndian.PutUint16(buf[2:], uint16(headerSize))
	binary.LittleEndian.PutUint32(buf[4:], uint32(size))
	binary.LittleEndian.PutUint32(buf[8:], uint32(len(list)))
	binary.LittleEndian.PutUint32(buf[20:], uint32(stringsStart))
	for i, off := range offs {
		binary.LittleEndian.PutUint32(buf[headerSize+4*i:], off)
	}
	copy(buf[stringsStart:], data)
	return buf
}

func strValue(pool []string, s string) (uint32, []string) {
	for i, have := range pool {
		if have == s {
			return uint32(i), pool
		}
	}
	pool = append(pool, s)
	return uint32(len(pool) - 1), pool
}

func simpleStr(idx uint32) arscValue {
	return arscValue{kind: typeString, data: idx}
}

func simpleBool() arscValue {
	return arscValue{kind: typeIntBoolean, data: boolFalse}
}
