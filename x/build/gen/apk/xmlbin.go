package apk

import (
	"encoding/binary"
	"fmt"
	"sort"
	"unicode/utf16"
)

// Binary Android XML. The file is one RES_XML chunk. The string pool is
// UTF-16. Attribute names in the android namespace occupy the front of
// that pool so the resource map lines up with their indexes.
const (
	xmlChunkXML       = 0x0003
	xmlChunkString    = 0x0001
	xmlChunkResMap    = 0x0180
	xmlStartNamespace = 0x0100
	xmlEndNamespace   = 0x0101
	xmlStartElement   = 0x0102
	xmlEndElement     = 0x0103
	xmlCDATA          = 0x0104
	xmlNoIndex        = 0xffffffff

	androidNS = "http://schemas.android.com/apk/res/android"
)

type xAttr struct {
	ns   string
	name string
	raw  string
	kind byte
	data uint32
}

type xNode struct {
	name  string
	text  string
	attrs []xAttr
	kids  []xNode
}

func aStr(name, raw string) xAttr {
	return xAttr{ns: androidNS, name: name, raw: raw, kind: typeString}
}

func aRef(name, raw string, id uint32) xAttr {
	return xAttr{ns: androidNS, name: name, raw: raw, kind: typeReference, data: id}
}

func aInt(name, raw string, v uint32) xAttr {
	return xAttr{ns: androidNS, name: name, raw: raw, kind: typeIntDec, data: v}
}

func aHex(name, raw string, v uint32) xAttr {
	return xAttr{ns: androidNS, name: name, raw: raw, kind: typeIntHex, data: v}
}

func aBool(name string, on bool) xAttr {
	raw := "false"
	data := uint32(boolFalse)
	if on {
		raw = "true"
		data = boolTrue
	}
	return xAttr{ns: androidNS, name: name, raw: raw, kind: typeIntBoolean, data: data}
}

func aAttrRef(name, raw string, id uint32) xAttr {
	return xAttr{ns: androidNS, name: name, raw: raw, kind: typeAttribute, data: id}
}

func aDP(name string, dp int) xAttr {
	// 23.0 fixed point, unit dp.
	return xAttr{ns: androidNS, name: name, raw: fmt.Sprintf("%ddp", dp), kind: typeDimension, data: uint32(dp<<8) | 1}
}

func plainStr(name, raw string) xAttr {
	return xAttr{name: name, raw: raw, kind: typeString}
}

func plainInt(name, raw string, v uint32) xAttr {
	return xAttr{name: name, raw: raw, kind: typeIntDec, data: v}
}

type xmlPool struct {
	index map[string]uint32
	list  []string
	attrN int
}

func newXMLPool() *xmlPool {
	return &xmlPool{index: map[string]uint32{}}
}

func (p *xmlPool) add(s string) uint32 {
	if i, ok := p.index[s]; ok {
		return i
	}
	i := uint32(len(p.list))
	p.index[s] = i
	p.list = append(p.list, s)
	return i
}

func (p *xmlPool) idx(s string) uint32 {
	i, ok := p.index[s]
	if !ok {
		panic("xml string missing: " + s)
	}
	return i
}

func encodeXML(root xNode, android bool) ([]byte, error) {
	pool := newXMLPool()
	if err := collectXML(pool, root, android); err != nil {
		return nil, err
	}
	var body []byte
	body = append(body, pool.chunk()...)
	if pool.attrN > 0 {
		mapChunk, err := pool.resMap()
		if err != nil {
			return nil, err
		}
		body = append(body, mapChunk...)
	}
	nodes, err := encodeNodes(pool, root, android)
	if err != nil {
		return nil, err
	}
	body = append(body, nodes...)
	out := make([]byte, 8+len(body))
	binary.LittleEndian.PutUint16(out[0:], xmlChunkXML)
	binary.LittleEndian.PutUint16(out[2:], 8)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(out)))
	copy(out[8:], body)
	return out, nil
}

func collectXML(pool *xmlPool, root xNode, android bool) error {
	if android {
		names := map[string]struct{}{}
		walkXML(root, func(n xNode) {
			for _, a := range n.attrs {
				if a.ns == androidNS {
					names[a.name] = struct{}{}
				}
			}
		})
		list := make([]string, 0, len(names))
		for name := range names {
			list = append(list, name)
		}
		sort.Strings(list)
		known := androidAttrs()
		for _, name := range list {
			if _, ok := known[name]; !ok {
				return fmt.Errorf("%w: android attr %s", ErrAssembleAPK, name)
			}
			pool.add(name)
		}
		pool.attrN = len(list)
		pool.add("android")
		pool.add(androidNS)
	}
	walkXML(root, func(n xNode) {
		pool.add(n.name)
		if n.text != "" {
			pool.add(n.text)
		}
		for _, a := range n.attrs {
			if a.ns != "" && a.ns != androidNS {
				pool.add(a.ns)
			}
			pool.add(a.name)
			if a.raw != "" {
				pool.add(a.raw)
			}
		}
	})
	return nil
}

func walkXML(n xNode, fn func(xNode)) {
	fn(n)
	for _, kid := range n.kids {
		walkXML(kid, fn)
	}
}

func (p *xmlPool) chunk() []byte {
	var data []byte
	offs := make([]uint32, len(p.list))
	for i, s := range p.list {
		offs[i] = uint32(len(data))
		data = appendUTF16(data, s)
	}
	headerSize := 28
	stringsStart := headerSize + 4*len(p.list)
	size := stringsStart + len(data)
	if pad := (4 - size%4) % 4; pad != 0 {
		data = append(data, make([]byte, pad)...)
		size += pad
	}
	buf := make([]byte, size)
	binary.LittleEndian.PutUint16(buf[0:], xmlChunkString)
	binary.LittleEndian.PutUint16(buf[2:], uint16(headerSize))
	binary.LittleEndian.PutUint32(buf[4:], uint32(size))
	binary.LittleEndian.PutUint32(buf[8:], uint32(len(p.list)))
	binary.LittleEndian.PutUint32(buf[20:], uint32(stringsStart))
	for i, off := range offs {
		binary.LittleEndian.PutUint32(buf[headerSize+4*i:], off)
	}
	copy(buf[stringsStart:], data)
	return buf
}

func appendUTF16(dst []byte, s string) []byte {
	u := utf16.Encode([]rune(s))
	n := len(u)
	if n >= 0x8000 {
		dst = binary.LittleEndian.AppendUint16(dst, uint16((n>>16)|0x8000))
		dst = binary.LittleEndian.AppendUint16(dst, uint16(n))
	} else {
		dst = binary.LittleEndian.AppendUint16(dst, uint16(n))
	}
	for _, c := range u {
		dst = binary.LittleEndian.AppendUint16(dst, c)
	}
	return binary.LittleEndian.AppendUint16(dst, 0)
}

func (p *xmlPool) resMap() ([]byte, error) {
	known := androidAttrs()
	body := make([]byte, 4*p.attrN)
	for i := 0; i < p.attrN; i++ {
		id, ok := known[p.list[i]]
		if !ok {
			return nil, fmt.Errorf("%w: android attr %s", ErrAssembleAPK, p.list[i])
		}
		binary.LittleEndian.PutUint32(body[4*i:], id)
	}
	buf := make([]byte, 8+len(body))
	binary.LittleEndian.PutUint16(buf[0:], xmlChunkResMap)
	binary.LittleEndian.PutUint16(buf[2:], 8)
	binary.LittleEndian.PutUint32(buf[4:], uint32(len(buf)))
	copy(buf[8:], body)
	return buf, nil
}

func encodeNodes(pool *xmlPool, root xNode, android bool) ([]byte, error) {
	var buf []byte
	if android {
		buf = append(buf, xmlNamespace(pool, xmlStartNamespace)...)
	}
	encoded, err := encodeElement(pool, root)
	if err != nil {
		return nil, err
	}
	buf = append(buf, encoded...)
	if android {
		buf = append(buf, xmlNamespace(pool, xmlEndNamespace)...)
	}
	return buf, nil
}

func xmlNamespace(pool *xmlPool, kind uint16) []byte {
	buf := make([]byte, 24)
	putChunk(buf, kind, 16, 24)
	binary.LittleEndian.PutUint32(buf[8:], 1)
	binary.LittleEndian.PutUint32(buf[12:], xmlNoIndex)
	binary.LittleEndian.PutUint32(buf[16:], pool.idx("android"))
	binary.LittleEndian.PutUint32(buf[20:], pool.idx(androidNS))
	return buf
}

func encodeElement(pool *xmlPool, n xNode) ([]byte, error) {
	var buf []byte
	start, err := startElement(pool, n)
	if err != nil {
		return nil, err
	}
	buf = append(buf, start...)
	if n.text != "" {
		buf = append(buf, cdataNode(pool, n.text)...)
	}
	for _, kid := range n.kids {
		kidBuf, err := encodeElement(pool, kid)
		if err != nil {
			return nil, err
		}
		buf = append(buf, kidBuf...)
	}
	buf = append(buf, endElement(pool, n.name)...)
	return buf, nil
}

// attrResID is the framework id for an android: attribute.
// Other attributes sort first, as id 0. The package parser merge-joins
// attributes with the styleable list, which is sorted by these ids.
// An icon that follows allowBackup is never visited.
func attrResID(a xAttr) uint32 {
	if a.ns != androidNS {
		return 0
	}
	return androidAttrs()[a.name]
}

func startElement(pool *xmlPool, n xNode) ([]byte, error) {
	attrs := append([]xAttr(nil), n.attrs...)
	sort.SliceStable(attrs, func(i, j int) bool {
		return attrResID(attrs[i]) < attrResID(attrs[j])
	})
	count := len(attrs)
	total := 16 + 20 + 20*count
	buf := make([]byte, total)
	putChunk(buf, xmlStartElement, 16, total)
	binary.LittleEndian.PutUint32(buf[8:], 1)
	binary.LittleEndian.PutUint32(buf[12:], xmlNoIndex)
	binary.LittleEndian.PutUint32(buf[16:], xmlNoIndex)
	binary.LittleEndian.PutUint32(buf[20:], pool.idx(n.name))
	binary.LittleEndian.PutUint16(buf[24:], 20)
	binary.LittleEndian.PutUint16(buf[26:], 20)
	binary.LittleEndian.PutUint16(buf[28:], uint16(count))
	var styleIndex uint16
	var idIndex uint16
	for i, a := range attrs {
		if a.name == "style" && a.ns == "" {
			styleIndex = uint16(i + 1)
		}
		if a.ns == androidNS && a.name == "id" {
			idIndex = uint16(i + 1)
		}
	}
	binary.LittleEndian.PutUint16(buf[30:], idIndex)
	binary.LittleEndian.PutUint16(buf[32:], 0)
	binary.LittleEndian.PutUint16(buf[34:], styleIndex)
	off := 36
	for _, a := range attrs {
		ns := uint32(xmlNoIndex)
		if a.ns != "" {
			ns = pool.idx(a.ns)
		}
		// A typed value stores no raw string. aapt leaves that index empty.
		// A raw "@mipmap/…" or "@string/…" makes the package parser keep the
		// symbol text and drop the resource id, so the launcher shows the
		// stock icon and the package name.
		raw := uint32(xmlNoIndex)
		data := a.data
		kind := a.kind
		if kind == typeString {
			raw = pool.idx(a.raw)
			data = raw
		}
		binary.LittleEndian.PutUint32(buf[off:], ns)
		binary.LittleEndian.PutUint32(buf[off+4:], pool.idx(a.name))
		binary.LittleEndian.PutUint32(buf[off+8:], raw)
		binary.LittleEndian.PutUint16(buf[off+12:], 8)
		buf[off+14] = 0
		buf[off+15] = kind
		binary.LittleEndian.PutUint32(buf[off+16:], data)
		off += 20
	}
	return buf, nil
}

func endElement(pool *xmlPool, name string) []byte {
	buf := make([]byte, 24)
	putChunk(buf, xmlEndElement, 16, 24)
	binary.LittleEndian.PutUint32(buf[8:], 1)
	binary.LittleEndian.PutUint32(buf[12:], xmlNoIndex)
	binary.LittleEndian.PutUint32(buf[16:], xmlNoIndex)
	binary.LittleEndian.PutUint32(buf[20:], pool.idx(name))
	return buf
}

func cdataNode(pool *xmlPool, text string) []byte {
	buf := make([]byte, 28)
	putChunk(buf, xmlCDATA, 16, 28)
	binary.LittleEndian.PutUint32(buf[8:], 1)
	binary.LittleEndian.PutUint32(buf[12:], xmlNoIndex)
	idx := pool.idx(text)
	binary.LittleEndian.PutUint32(buf[16:], idx)
	binary.LittleEndian.PutUint16(buf[20:], 8)
	buf[23] = typeString
	binary.LittleEndian.PutUint32(buf[24:], idx)
	return buf
}

func putChunk(buf []byte, kind uint16, headerSize, size int) {
	binary.LittleEndian.PutUint16(buf[0:], kind)
	binary.LittleEndian.PutUint16(buf[2:], uint16(headerSize))
	binary.LittleEndian.PutUint32(buf[4:], uint32(size))
}
