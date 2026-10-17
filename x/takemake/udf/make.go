package udf

import (
	"encoding/binary"
	"io/fs"
	"path"
	"strings"
	"time"
)

func makeVolume(f *File) ([]byte, error) {
	label := f.Label
	if label == "" {
		label = defaultLabel
	}
	if strings.ContainsRune(label, 0) || len(label) > 31 {
		return nil, ErrName
	}
	root, order, err := buildTree(f.Entries)
	if err != nil {
		return nil, err
	}
	partBlocks, err := place(order)
	if err != nil {
		return nil, err
	}
	part := make([]byte, int(partBlocks)*sectorSize)
	copy(part, encodeFSD(label, root.fe))
	uid := uint64(16)
	for _, node := range order {
		copy(part[int(node.fe)*sectorSize:], encodeFE(node, uid))
		if len(node.payload) > 0 {
			copy(part[int(node.data)*sectorSize:], node.payload)
		}
		uid++
	}
	return assemble(label, part, partBlocks), nil
}

type node struct {
	name     string
	base     string
	entry    *Entry
	parent   *node
	kids     []*node
	ident    []byte
	fileType byte
	fe       uint32
	data     uint32
	blocks   uint32
	payload  []byte
	adLen    uint32
	infoLen  uint64
}

func buildTree(entries []*Entry) (*node, []*node, error) {
	root := &node{fileType: fileTypeDir}
	index := map[string]*node{"": root}
	for _, entry := range entries {
		if entry == nil {
			return nil, nil, ErrName
		}
		cleaned, err := cleanName(entry.Name)
		if err != nil || cleaned != entry.Name {
			return nil, nil, ErrName
		}
		if _, ok := index[cleaned]; ok {
			return nil, nil, ErrName
		}
		parentName := path.Dir(cleaned)
		if parentName == "." {
			parentName = ""
		}
		parent := index[parentName]
		if parent == nil {
			return nil, nil, ErrName
		}
		if parent.fileType != fileTypeDir {
			return nil, nil, ErrKind
		}
		kind, err := kindOf(entry)
		if err != nil {
			return nil, nil, err
		}
		ident, err := encodeIdent(path.Base(cleaned))
		if err != nil {
			return nil, nil, err
		}
		child := &node{
			name:     cleaned,
			base:     path.Base(cleaned),
			entry:    entry,
			parent:   parent,
			ident:    ident,
			fileType: kind,
		}
		parent.kids = append(parent.kids, child)
		index[cleaned] = child
	}
	var order []*node
	var walk func(*node)
	walk = func(n *node) {
		order = append(order, n)
		for _, kid := range n.kids {
			walk(kid)
		}
	}
	walk(root)
	return root, order, nil
}

func kindOf(entry *Entry) (byte, error) {
	switch entry.Mode & fs.ModeType {
	case fs.ModeDir:
		return fileTypeDir, nil
	case fs.ModeSymlink:
		if entry.Target == "" || strings.ContainsRune(entry.Target, 0) {
			return 0, ErrSymlink
		}
		return fileTypeSymlink, nil
	case 0:
		return fileTypeFile, nil
	default:
		return 0, ErrKind
	}
}

func place(order []*node) (uint32, error) {
	for i, node := range order {
		node.fe = uint32(i + 1)
	}
	next := uint32(len(order) + 1)
	for _, node := range order {
		size, last, err := node.measure()
		if err != nil {
			return 0, err
		}
		node.infoLen = uint64(size)
		ad := size
		if node.fileType == fileTypeDir {
			aligned := size
			if rem := aligned % sectorSize; rem != 0 {
				aligned += sectorSize - rem
			}
			if last+fidLookahead > aligned {
				ad = last + fidLookahead
			}
			if ad == 0 {
				return 0, ErrFormat
			}
		}
		if ad >= 1<<30 {
			return 0, ErrFormat
		}
		node.adLen = uint32(ad)
		if ad == 0 {
			continue
		}
		node.data = next
		node.blocks = uint32((ad + sectorSize - 1) / sectorSize)
		next += node.blocks
	}
	for _, node := range order {
		payload, err := node.encodePayload()
		if err != nil {
			return 0, err
		}
		if len(payload) != int(node.infoLen) {
			return 0, ErrFormat
		}
		node.payload = payload
	}
	return next, nil
}

func (n *node) measure() (int, int, error) {
	switch n.fileType {
	case fileTypeDir:
		off := fidLen(0)
		last := 0
		for _, kid := range n.kids {
			last = off
			off += fidLen(len(kid.ident))
		}
		return off, last, nil
	case fileTypeSymlink:
		body, err := encodeSymlink(n.entry.Target)
		if err != nil {
			return 0, 0, err
		}
		n.payload = body
		return len(body), 0, nil
	default:
		return len(n.entry.Data), 0, nil
	}
}

func (n *node) encodePayload() ([]byte, error) {
	switch n.fileType {
	case fileTypeSymlink:
		return n.payload, nil
	case fileTypeFile:
		if len(n.entry.Data) == 0 {
			return nil, nil
		}
		return append([]byte(nil), n.entry.Data...), nil
	default:
		parentFE := n.fe
		if n.parent != nil {
			parentFE = n.parent.fe
		}
		buf := make([]byte, 0, n.infoLen)
		buf = appendFID(buf, n.data, 0x08, nil, parentFE)
		for _, kid := range n.kids {
			var ch byte
			if kid.fileType == fileTypeDir {
				ch = 0x02
			}
			buf = appendFID(buf, n.data, ch, kid.ident, kid.fe)
		}
		return buf, nil
	}
}

func appendFID(buf []byte, dataBlock uint32, ch byte, ident []byte, fe uint32) []byte {
	pad := fidLen(len(ident))
	start := len(buf)
	buf = append(buf, make([]byte, pad)...)
	rec := buf[start:]
	binary.LittleEndian.PutUint16(rec[16:], 1)
	rec[18] = ch
	rec[19] = byte(len(ident))
	binary.LittleEndian.PutUint32(rec[20:], sectorSize)
	binary.LittleEndian.PutUint32(rec[24:], fe)
	copy(rec[38:], ident)
	finishTag(rec[:pad], tagFID, dataBlock+uint32(start/sectorSize), pad-16)
	return buf
}

func (n *node) mode() uint32 {
	if n.entry == nil {
		return 0o755
	}
	perm := uint32(n.entry.Mode.Perm())
	if perm == 0 {
		switch n.fileType {
		case fileTypeSymlink:
			return 0o777
		case fileTypeDir:
			return 0o755
		default:
			return 0o644
		}
	}
	return perm
}

func (n *node) when() time.Time {
	if n.entry == nil || n.entry.Time.IsZero() {
		return defaultTime
	}
	return wall(n.entry.Time)
}

func encodeFE(n *node, uid uint64) []byte {
	ad := 0
	if n.adLen > 0 {
		ad = 8
	}
	buf := make([]byte, 176+ad)
	binary.LittleEndian.PutUint16(buf[20:], 4)
	binary.LittleEndian.PutUint16(buf[24:], 1)
	buf[27] = n.fileType
	parent := n.fe
	if n.parent != nil {
		parent = n.parent.fe
	}
	binary.LittleEndian.PutUint32(buf[28:], parent)
	binary.LittleEndian.PutUint32(buf[44:], unixPerm(n.mode()))
	binary.LittleEndian.PutUint16(buf[48:], 1)
	binary.LittleEndian.PutUint64(buf[56:], n.infoLen)
	binary.LittleEndian.PutUint64(buf[64:], uint64(n.blocks))
	when := n.when()
	putTS(buf[72:], when)
	putTS(buf[84:], when)
	putTS(buf[96:], when)
	binary.LittleEndian.PutUint32(buf[108:], 1)
	putRegID(buf[128:160], "*lewkit")
	binary.LittleEndian.PutUint64(buf[160:], uid)
	binary.LittleEndian.PutUint32(buf[172:], uint32(ad))
	if ad == 8 {
		binary.LittleEndian.PutUint32(buf[176:], n.adLen)
		binary.LittleEndian.PutUint32(buf[180:], n.data)
	}
	finishTag(buf, tagFE, n.fe, len(buf)-16)
	return buf
}

func encodeFSD(label string, rootFE uint32) []byte {
	buf := make([]byte, 512)
	putTS(buf[16:], defaultTime)
	binary.LittleEndian.PutUint16(buf[28:], 3)
	binary.LittleEndian.PutUint16(buf[30:], 3)
	binary.LittleEndian.PutUint32(buf[32:], 1)
	binary.LittleEndian.PutUint32(buf[36:], 1)
	putDString(buf[112:240], label)
	putDString(buf[304:336], label)
	binary.LittleEndian.PutUint32(buf[400:], sectorSize)
	binary.LittleEndian.PutUint32(buf[404:], rootFE)
	putRegID(buf[416:448], "*OSTA UDF Compliant")
	finishTag(buf, tagFSD, 0, len(buf)-16)
	return buf
}

func assemble(label string, part []byte, partBlocks uint32) []byte {
	total := partSector + int(partBlocks) + vdsSectors
	img := make([]byte, total*sectorSize)
	putVRS(img, 16, "BEA01")
	putVRS(img, 17, "NSR03")
	putVRS(img, 18, "TEA01")
	reserve := uint32(partSector) + partBlocks
	copy(img[256*sectorSize:], encodeAVDP(vdsSector, reserve))
	writeVDS(img, vdsSector, label, partBlocks)
	copy(img[partSector*sectorSize:], part)
	writeVDS(img, reserve, label, partBlocks)
	return img
}

func putVRS(img []byte, sector int, id string) {
	off := sector * sectorSize
	copy(img[off+1:], id)
	img[off+6] = 1
}

func writeVDS(img []byte, at uint32, label string, partBlocks uint32) {
	put := func(i uint32, b []byte) {
		copy(img[int(at+i)*sectorSize:], b)
	}
	put(0, encodePVD(label, at))
	put(1, encodePD(at+1, partBlocks))
	put(2, encodeLVD(label, at+2))
	put(3, encodeTD(at+3))
}

func encodeAVDP(main, reserve uint32) []byte {
	buf := make([]byte, 512)
	extent := uint32(vdsSectors * sectorSize)
	binary.LittleEndian.PutUint32(buf[16:], extent)
	binary.LittleEndian.PutUint32(buf[20:], main)
	binary.LittleEndian.PutUint32(buf[24:], extent)
	binary.LittleEndian.PutUint32(buf[28:], reserve)
	finishTag(buf, tagAVDP, 256, len(buf)-16)
	return buf
}

func encodePVD(label string, sector uint32) []byte {
	buf := make([]byte, 512)
	binary.LittleEndian.PutUint32(buf[16:], 1)
	putDString(buf[24:56], label)
	binary.LittleEndian.PutUint16(buf[56:], 1)
	binary.LittleEndian.PutUint16(buf[58:], 1)
	binary.LittleEndian.PutUint16(buf[60:], 2)
	binary.LittleEndian.PutUint16(buf[62:], 2)
	binary.LittleEndian.PutUint32(buf[64:], 1)
	binary.LittleEndian.PutUint32(buf[68:], 1)
	putDString(buf[72:200], label)
	putTS(buf[376:], defaultTime)
	putRegID(buf[388:420], "*lewkit")
	finishTag(buf, tagPVD, sector, len(buf)-16)
	return buf
}

func encodePD(sector, partBlocks uint32) []byte {
	buf := make([]byte, 512)
	binary.LittleEndian.PutUint32(buf[16:], 1)
	binary.LittleEndian.PutUint16(buf[20:], 1)
	putRegID(buf[24:56], "+NSR03")
	binary.LittleEndian.PutUint32(buf[184:], 1)
	binary.LittleEndian.PutUint32(buf[188:], partSector)
	binary.LittleEndian.PutUint32(buf[192:], partBlocks)
	putRegID(buf[196:228], "*lewkit")
	finishTag(buf, tagPD, sector, len(buf)-16)
	return buf
}

func encodeLVD(label string, sector uint32) []byte {
	buf := make([]byte, 512)
	binary.LittleEndian.PutUint32(buf[16:], 1)
	putDString(buf[84:212], label)
	binary.LittleEndian.PutUint32(buf[212:], sectorSize)
	putRegID(buf[216:248], "*OSTA UDF Compliant")
	binary.LittleEndian.PutUint32(buf[248:], sectorSize)
	binary.LittleEndian.PutUint32(buf[264:], 6)
	binary.LittleEndian.PutUint32(buf[268:], 1)
	putRegID(buf[272:304], "*lewkit")
	buf[440] = 1
	buf[441] = 6
	binary.BigEndian.PutUint16(buf[442:], 1)
	finishTag(buf, tagLVD, sector, len(buf)-16)
	return buf
}

func encodeTD(sector uint32) []byte {
	buf := make([]byte, 512)
	finishTag(buf, tagTD, sector, len(buf)-16)
	return buf
}
