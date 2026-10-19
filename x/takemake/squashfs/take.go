package squashfs

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"io/fs"
	"time"
)

const (
	inodeExtDir  = 8
	inodeExtFile = 9
	inodeExtLink = 10
)

type superblock struct {
	blockSize   uint32
	fragCount   uint32
	compression uint16
	idCount     uint16
	rootBlock   uint32
	rootOff     uint16
	bytesUsed   uint64
	idTable     uint64
	inodeTable  uint64
	dirTable    uint64
	fragTable   uint64
}

type fragment struct {
	start      int64
	size       uint32
	compressed bool
	data       []byte
	ready      bool
}

type sqInode struct {
	kind     uint16
	mode     uint16
	mtime    uint32
	dirBlock uint32
	dirOff   uint16
	dirSize  int
	start    uint64
	frag     uint32
	fragOff  uint32
	size     uint64
	blocks   []dataBlock
	target   string
}

type dataBlock struct {
	size       uint32
	compressed bool
}

type dirItem struct {
	name  string
	block uint32
	off   uint16
}

func take(raw []byte) (*File, error) {
	sb, err := parseSuper(raw)
	if err != nil {
		return nil, err
	}
	img := &sqImage{raw: raw, sb: sb}
	if err := img.loadFragments(); err != nil {
		return nil, err
	}
	root, err := img.readInode(sb.rootBlock, sb.rootOff)
	if err != nil {
		return nil, err
	}
	if !isDirKind(root.kind) {
		return nil, ErrFormat
	}
	var entries []*Entry
	if err := img.walk(root, "", &entries); err != nil {
		return nil, err
	}
	return &File{Entries: entries, cur: -1}, nil
}

type sqImage struct {
	raw   []byte
	sb    superblock
	frags []fragment
}

func parseSuper(raw []byte) (superblock, error) {
	if len(raw) < superSize {
		return superblock{}, ErrFormat
	}
	if binary.LittleEndian.Uint32(raw[0:4]) != magic {
		return superblock{}, ErrFormat
	}
	if binary.LittleEndian.Uint16(raw[28:30]) != 4 || binary.LittleEndian.Uint16(raw[30:32]) != 0 {
		return superblock{}, ErrFormat
	}
	block := binary.LittleEndian.Uint32(raw[12:16])
	log := binary.LittleEndian.Uint16(raw[22:24])
	if block < 4096 || block > 1<<20 || block&(block-1) != 0 || 1<<log != block {
		return superblock{}, ErrFormat
	}
	used := binary.LittleEndian.Uint64(raw[40:48])
	if used < superSize || used > uint64(len(raw)) {
		return superblock{}, ErrFormat
	}
	root := binary.LittleEndian.Uint64(raw[32:40])
	return superblock{
		blockSize:   block,
		fragCount:   binary.LittleEndian.Uint32(raw[16:20]),
		compression: binary.LittleEndian.Uint16(raw[20:22]),
		idCount:     binary.LittleEndian.Uint16(raw[26:28]),
		rootBlock:   uint32(root >> 16),
		rootOff:     uint16(root),
		bytesUsed:   used,
		idTable:     binary.LittleEndian.Uint64(raw[48:56]),
		inodeTable:  binary.LittleEndian.Uint64(raw[64:72]),
		dirTable:    binary.LittleEndian.Uint64(raw[72:80]),
		fragTable:   binary.LittleEndian.Uint64(raw[80:88]),
	}, nil
}

func (m *sqImage) loadFragments() error {
	if m.sb.fragCount == 0 {
		return nil
	}
	blocks := int((m.sb.fragCount + 511) / 512)
	indexAt := int(m.sb.fragTable)
	if m.sb.fragTable == absentTable || indexAt < 0 || indexAt+blocks*8 > len(m.raw) {
		return ErrFormat
	}
	for i := 0; i < blocks && len(m.frags) < int(m.sb.fragCount); i++ {
		loc := binary.LittleEndian.Uint64(m.raw[indexAt+i*8:])
		data, _, err := m.metaBlock(int64(loc))
		if err != nil {
			return err
		}
		for j := 0; j+16 <= len(data) && len(m.frags) < int(m.sb.fragCount); j += 16 {
			size := binary.LittleEndian.Uint32(data[j+8 : j+12])
			m.frags = append(m.frags, fragment{
				start:      int64(binary.LittleEndian.Uint64(data[j : j+8])),
				size:       size & 0x00ffffff,
				compressed: size&(1<<24) == 0,
			})
		}
	}
	if len(m.frags) != int(m.sb.fragCount) {
		return ErrFormat
	}
	return nil
}

func (m *sqImage) walk(dir *sqInode, prefix string, out *[]*Entry) error {
	items, err := m.directory(dir)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.name == "" || item.name == "." || item.name == ".." {
			continue
		}
		child, err := m.readInode(item.block, item.off)
		if err != nil {
			return err
		}
		full := item.name
		if prefix != "" {
			full = prefix + "/" + item.name
		}
		entry, err := m.entry(child, full)
		if err != nil {
			return err
		}
		*out = append(*out, entry)
		if isDirKind(child.kind) {
			if err := m.walk(child, full, out); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *sqImage) entry(n *sqInode, name string) (*Entry, error) {
	entry := &Entry{
		Name: name,
		Mode: goMode(n.mode, n.kind),
		Time: time.Unix(int64(n.mtime), 0).UTC(),
	}
	switch {
	case isDirKind(n.kind):
		return entry, nil
	case isLinkKind(n.kind):
		entry.Target = n.target
		return entry, nil
	case isFileKind(n.kind):
		data, err := m.fileBytes(n)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			entry.Data = data
		}
		return entry, nil
	default:
		return nil, ErrInode
	}
}

func (m *sqImage) directory(dir *sqInode) ([]dirItem, error) {
	if dir.dirSize <= 0 {
		return nil, nil
	}
	raw, err := m.readStream(int64(m.sb.dirTable), dir.dirBlock, dir.dirOff, dir.dirSize)
	if err != nil {
		return nil, err
	}
	var items []dirItem
	pos := 0
	for pos+12 <= len(raw) {
		count := binary.LittleEndian.Uint32(raw[pos:pos+4]) + 1
		block := binary.LittleEndian.Uint32(raw[pos+4 : pos+8])
		pos += 12
		if count == 0 || count > 256 {
			return nil, ErrFormat
		}
		for range count {
			if pos+8 > len(raw) {
				return nil, ErrFormat
			}
			off := binary.LittleEndian.Uint16(raw[pos : pos+2])
			nameLen := int(binary.LittleEndian.Uint16(raw[pos+6:pos+8])) + 1
			pos += 8
			if nameLen <= 0 || nameLen > 256 || pos+nameLen > len(raw) {
				return nil, ErrFormat
			}
			name := string(raw[pos : pos+nameLen])
			pos += nameLen
			// The inode delta in the entry is not needed. Block and offset locate it.
			items = append(items, dirItem{
				name:  name,
				block: block,
				off:   off,
			})
		}
	}
	return items, nil
}

func (m *sqImage) readInode(block uint32, off uint16) (*sqInode, error) {
	raw, err := m.readStream(int64(m.sb.inodeTable), block, off, headerSize)
	if err != nil {
		return nil, err
	}
	n := &sqInode{
		kind:  binary.LittleEndian.Uint16(raw[0:2]),
		mode:  binary.LittleEndian.Uint16(raw[2:4]),
		mtime: binary.LittleEndian.Uint32(raw[8:12]),
		frag:  invalidFrag,
	}
	fixed := fixedBody(n.kind)
	if fixed < 0 {
		return nil, ErrInode
	}
	raw, err = m.readStream(int64(m.sb.inodeTable), block, off, headerSize+fixed)
	if err != nil {
		return nil, err
	}
	body := raw[headerSize:]
	extra := 0
	switch n.kind {
	case inodeDir:
		n.dirBlock = binary.LittleEndian.Uint32(body[0:4])
		n.dirSize = int(binary.LittleEndian.Uint16(body[8:10]))
		n.dirOff = binary.LittleEndian.Uint16(body[10:12])
	case inodeExtDir:
		n.dirSize = int(binary.LittleEndian.Uint32(body[4:8]))
		n.dirBlock = binary.LittleEndian.Uint32(body[8:12])
		n.dirOff = binary.LittleEndian.Uint16(body[18:20])
	case inodeFile:
		n.start = uint64(binary.LittleEndian.Uint32(body[0:4]))
		n.frag = binary.LittleEndian.Uint32(body[4:8])
		n.fragOff = binary.LittleEndian.Uint32(body[8:12])
		n.size = uint64(binary.LittleEndian.Uint32(body[12:16]))
		extra = blockCount(n.size, uint64(m.sb.blockSize), n.frag) * 4
	case inodeExtFile:
		n.start = binary.LittleEndian.Uint64(body[0:8])
		n.size = binary.LittleEndian.Uint64(body[8:16])
		n.frag = binary.LittleEndian.Uint32(body[28:32])
		n.fragOff = binary.LittleEndian.Uint32(body[32:36])
		extra = blockCount(n.size, uint64(m.sb.blockSize), n.frag) * 4
	case inodeLink:
		extra = int(binary.LittleEndian.Uint32(body[4:8]))
	case inodeExtLink:
		extra = int(binary.LittleEndian.Uint32(body[4:8])) + 4
	default:
		return nil, ErrInode
	}
	if extra > 0 {
		raw, err = m.readStream(int64(m.sb.inodeTable), block, off, headerSize+fixed+extra)
		if err != nil {
			return nil, err
		}
		body = raw[headerSize:]
	}
	switch n.kind {
	case inodeFile, inodeExtFile:
		listAt := fixed
		for i := 0; i < extra; i += 4 {
			word := binary.LittleEndian.Uint32(body[listAt+i : listAt+i+4])
			n.blocks = append(n.blocks, dataBlock{
				size:       word & 0x00ffffff,
				compressed: word&(1<<24) == 0,
			})
		}
	case inodeLink:
		n.target = string(body[8 : 8+extra])
	case inodeExtLink:
		nbyte := extra - 4
		if nbyte < 0 || 8+nbyte > len(body) {
			return nil, ErrFormat
		}
		n.target = string(body[8 : 8+nbyte])
	}
	return n, nil
}

func (m *sqImage) fileBytes(n *sqInode) ([]byte, error) {
	var out []byte
	cursor := int64(n.start)
	for _, blk := range n.blocks {
		if blk.size == 0 {
			out = append(out, make([]byte, m.sb.blockSize)...)
			continue
		}
		if cursor < 0 || cursor+int64(blk.size) > int64(len(m.raw)) {
			return nil, ErrFormat
		}
		chunk := m.raw[cursor : cursor+int64(blk.size)]
		cursor += int64(blk.size)
		if blk.compressed {
			var err error
			chunk, err = inflate(chunk, int(m.sb.blockSize), m.sb.compression)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, chunk...)
	}
	if n.frag != invalidFrag {
		tail := int(n.size % uint64(m.sb.blockSize))
		if tail == 0 && n.size != 0 {
			return out, nil
		}
		if n.size < uint64(m.sb.blockSize) {
			tail = int(n.size)
		}
		frag, err := m.fragmentBytes(n.frag)
		if err != nil {
			return nil, err
		}
		off := int(n.fragOff)
		if off < 0 || off+tail > len(frag) {
			return nil, ErrFormat
		}
		out = append(out, frag[off:off+tail]...)
	}
	if uint64(len(out)) < n.size {
		return nil, ErrFormat
	}
	if uint64(len(out)) > n.size {
		out = out[:n.size]
	}
	return out, nil
}

func (m *sqImage) fragmentBytes(index uint32) ([]byte, error) {
	if int(index) >= len(m.frags) {
		return nil, ErrFormat
	}
	frag := &m.frags[index]
	if frag.ready {
		return frag.data, nil
	}
	if frag.start < 0 || frag.start+int64(frag.size) > int64(len(m.raw)) {
		return nil, ErrFormat
	}
	chunk := m.raw[frag.start : frag.start+int64(frag.size)]
	if frag.compressed {
		var err error
		chunk, err = inflate(chunk, int(m.sb.blockSize), m.sb.compression)
		if err != nil {
			return nil, err
		}
	}
	frag.data = chunk
	frag.ready = true
	return frag.data, nil
}

func (m *sqImage) readStream(origin int64, block uint32, off uint16, n int) ([]byte, error) {
	if n <= 0 {
		return nil, nil
	}
	var out []byte
	pos := origin + int64(block)
	skip := int(off)
	for len(out) < n {
		data, disk, err := m.metaBlock(pos)
		if err != nil {
			return nil, err
		}
		if skip > len(data) {
			return nil, ErrFormat
		}
		out = append(out, data[skip:]...)
		skip = 0
		if disk < 2 {
			return nil, ErrFormat
		}
		pos += int64(disk)
	}
	return out[:n], nil
}

func (m *sqImage) metaBlock(at int64) ([]byte, int, error) {
	if at < 0 || at+2 > int64(len(m.raw)) {
		return nil, 0, ErrFormat
	}
	hdr := binary.LittleEndian.Uint16(m.raw[at : at+2])
	size := int(hdr & 0x7fff)
	compressed := hdr&0x8000 == 0
	if size < 0 || at+2+int64(size) > int64(len(m.raw)) {
		return nil, 0, ErrFormat
	}
	payload := m.raw[at+2 : at+2+int64(size)]
	if compressed {
		var err error
		payload, err = inflate(payload, metaMax, m.sb.compression)
		if err != nil {
			return nil, 0, err
		}
	}
	if len(payload) > metaMax {
		return nil, 0, ErrFormat
	}
	return payload, 2 + size, nil
}

func inflate(src []byte, limit int, compression uint16) ([]byte, error) {
	if compression != compressionGzip {
		return nil, ErrCompress
	}
	r, err := zlib.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, ErrFormat
	}
	defer r.Close()
	out, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return nil, ErrFormat
	}
	if len(out) > limit {
		return nil, ErrFormat
	}
	return out, nil
}

func fixedBody(kind uint16) int {
	switch kind {
	case inodeDir:
		return 16
	case inodeExtDir:
		return 24
	case inodeFile:
		return 16
	case inodeExtFile:
		return 40
	case inodeLink, inodeExtLink:
		return 8
	default:
		return -1
	}
}

func blockCount(size, block uint64, frag uint32) int {
	if block == 0 {
		return 0
	}
	n := int(size / block)
	if size%block != 0 && frag == invalidFrag {
		n++
	}
	return n
}

func isDirKind(kind uint16) bool {
	return kind == inodeDir || kind == inodeExtDir
}

func isFileKind(kind uint16) bool {
	return kind == inodeFile || kind == inodeExtFile
}

func isLinkKind(kind uint16) bool {
	return kind == inodeLink || kind == inodeExtLink
}

func goMode(unix uint16, kind uint16) fs.FileMode {
	mode := fs.FileMode(unix & 0o777)
	if unix&0o4000 != 0 {
		mode |= fs.ModeSetuid
	}
	if unix&0o2000 != 0 {
		mode |= fs.ModeSetgid
	}
	if unix&0o1000 != 0 {
		mode |= fs.ModeSticky
	}
	switch {
	case isDirKind(kind):
		mode |= fs.ModeDir
	case isLinkKind(kind):
		mode |= fs.ModeSymlink
	}
	return mode
}
