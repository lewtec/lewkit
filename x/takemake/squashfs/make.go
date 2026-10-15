package squashfs

import (
	"encoding/binary"
	"io/fs"
	"slices"
	"strings"
)

const (
	magic           = 0x73717368
	blockSize       = 4096
	metaMax         = 8192
	compressionGzip = 1
	invalidFrag     = 0xffffffff
	absentTable     = ^uint64(0)

	unixFile = 0o100000
	unixDir  = 0o040000
	unixLink = 0o120000

	inodeDir  = 1
	inodeFile = 2
	inodeLink = 3

	flagUncompInode = 1 << 0
	flagUncompData  = 1 << 1
	flagUncompFrag  = 1 << 3
	flagNoFrag      = 1 << 4
	flagUncompXattr = 1 << 8
	flagNoXattr     = 1 << 9
	flagUncompID    = 1 << 11

	superSize  = 96
	headerSize = 16
)

type node struct {
	name     string
	entry    *Entry
	children []*node
	ino      uint32
	parent   uint32
	dataOff  uint32
	blocks   []uint32
	inoAt    int
}

func makeImage(f *File) ([]byte, error) {
	root, err := tree(f.Entries)
	if err != nil {
		return nil, err
	}
	var nodes []*node
	number(root, 0, &nodes)
	data, layout := fileData(nodes)

	inodes := make([]byte, 0, len(nodes)*48)
	for _, n := range nodes {
		n.inoAt = len(inodes)
		raw, err := inodeBytes(n, layout)
		if err != nil {
			return nil, err
		}
		inodes = append(inodes, raw...)
	}

	dirs := make([]byte, 0)
	dirAt := make([]int, len(nodes))
	for i, n := range nodes {
		if !n.entry.isDir() && n != root {
			dirAt[i] = -1
			continue
		}
		dirAt[i] = len(dirs)
		dirs = append(dirs, directoryBytes(n, nodes)...)
	}

	for i, n := range nodes {
		if dirAt[i] < 0 {
			continue
		}
		listing := directoryBytes(n, nodes)
		if err := patchDirectory(inodes[n.inoAt:], dirs, dirAt[i], len(listing)); err != nil {
			return nil, err
		}
	}

	inodeTable := packMeta(inodes)
	var dirTable []byte
	if len(dirs) > 0 {
		dirTable = packMeta(dirs)
	}

	// Superblock, then file bytes, inode table, directory table, id table.
	image := make([]byte, superSize+len(data)+len(inodeTable)+len(dirTable))
	copy(image[superSize:], data)
	inodeAt := superSize + len(data)
	copy(image[inodeAt:], inodeTable)
	dirAtAbs := inodeAt + len(inodeTable)
	copy(image[dirAtAbs:], dirTable)

	idMeta := len(image)
	idBlock := packMeta([]byte{0, 0, 0, 0})
	image = append(image, idBlock...)
	idIndex := len(image)
	var index [8]byte
	binary.LittleEndian.PutUint64(index[:], uint64(idMeta))
	image = append(image, index[:]...)

	dirTableAbs := uint64(dirAtAbs)
	if len(dirTable) == 0 {
		// An empty directory listing reads three bytes into the next metadata block.
		dirTableAbs = uint64(idMeta)
	}

	when := defaultTime
	for _, n := range nodes {
		if n.entry != nil && n.entry.Time.After(when) {
			when = n.entry.Time
		}
	}
	rootRef := inodeRef(inodes, nodes[0], inodeTable)

	var sb [superSize]byte
	binary.LittleEndian.PutUint32(sb[0:4], magic)
	binary.LittleEndian.PutUint32(sb[4:8], uint32(len(nodes)))
	binary.LittleEndian.PutUint32(sb[8:12], uint32(when.Unix()))
	binary.LittleEndian.PutUint32(sb[12:16], blockSize)
	binary.LittleEndian.PutUint32(sb[16:20], 0)
	binary.LittleEndian.PutUint16(sb[20:22], compressionGzip)
	binary.LittleEndian.PutUint16(sb[22:24], 12)
	flags := uint16(flagUncompInode | flagUncompData | flagUncompFrag | flagNoFrag | flagUncompXattr | flagNoXattr | flagUncompID)
	binary.LittleEndian.PutUint16(sb[24:26], flags)
	binary.LittleEndian.PutUint16(sb[26:28], 1)
	binary.LittleEndian.PutUint16(sb[28:30], 4)
	binary.LittleEndian.PutUint16(sb[30:32], 0)
	binary.LittleEndian.PutUint64(sb[32:40], rootRef)
	binary.LittleEndian.PutUint64(sb[40:48], uint64(len(image)))
	binary.LittleEndian.PutUint64(sb[48:56], uint64(idIndex))
	binary.LittleEndian.PutUint64(sb[56:64], absentTable)
	binary.LittleEndian.PutUint64(sb[64:72], uint64(inodeAt))
	binary.LittleEndian.PutUint64(sb[72:80], dirTableAbs)
	binary.LittleEndian.PutUint64(sb[80:88], absentTable)
	binary.LittleEndian.PutUint64(sb[88:96], absentTable)
	copy(image[:superSize], sb[:])
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
	if uint64(len(entry.Data)) > uint64(^uint32(0)) {
		return ErrSize
	}
	return nil
}

func number(n *node, parent uint32, out *[]*node) {
	n.ino = uint32(len(*out) + 1)
	n.parent = parent
	if n.parent == 0 {
		n.parent = n.ino
	}
	*out = append(*out, n)
	for _, child := range n.children {
		number(child, n.ino, out)
	}
}

func fileData(nodes []*node) ([]byte, map[*node]int) {
	var data []byte
	layout := map[*node]int{}
	for _, n := range nodes {
		if n.entry == nil || !n.entry.isFile() || len(n.entry.Data) == 0 {
			continue
		}
		layout[n] = len(data)
		payload := n.entry.Data
		for len(payload) > 0 {
			nbyte := blockSize
			if nbyte > len(payload) {
				nbyte = len(payload)
			}
			// Bit 24 marks the block as stored uncompressed.
			n.blocks = append(n.blocks, uint32(nbyte)|(1<<24))
			data = append(data, payload[:nbyte]...)
			payload = payload[nbyte:]
		}
		n.dataOff = uint32(superSize + layout[n])
	}
	return data, layout
}

func inodeBytes(n *node, layout map[*node]int) ([]byte, error) {
	when := defaultTime
	mode := fs.FileMode(dirMode)
	if n.entry != nil {
		when = wall(n.entry.Time)
		mode = n.entry.Mode
	}
	kind := inodeDir
	unix := uint16(unixDir | 0o755)
	switch {
	case n.entry != nil && n.entry.isSymlink():
		kind = inodeLink
		unix = uint16(unixLink) | unixPerm(mode)
	case n.entry != nil && n.entry.isFile():
		kind = inodeFile
		unix = uint16(unixFile) | unixPerm(mode)
	default:
		unix = uint16(unixDir) | unixPerm(mode)
	}
	var buf []byte
	buf = appendU16(buf, uint16(kind))
	buf = appendU16(buf, unix)
	buf = appendU16(buf, 0)
	buf = appendU16(buf, 0)
	buf = appendU32(buf, uint32(when.Unix()))
	buf = appendU32(buf, n.ino)
	switch kind {
	case inodeDir:
		// start block, nlink, file size, offset, and parent are patched
		// once the directory table exists. nlink counts the directory and its parent.
		nlink := uint32(2)
		for _, child := range n.children {
			if child.entry.isDir() {
				nlink++
			}
		}
		buf = appendU32(buf, 0)
		buf = appendU32(buf, nlink)
		buf = appendU16(buf, 3)
		buf = appendU16(buf, 0)
		buf = appendU32(buf, n.parent)
	case inodeFile:
		start := uint32(0)
		if _, ok := layout[n]; ok {
			start = n.dataOff
		}
		buf = appendU32(buf, start)
		buf = appendU32(buf, invalidFrag)
		buf = appendU32(buf, 0)
		buf = appendU32(buf, uint32(len(n.entry.Data)))
		for _, block := range n.blocks {
			buf = appendU32(buf, block)
		}
	case inodeLink:
		target := n.entry.Target
		buf = appendU32(buf, 1)
		buf = appendU32(buf, uint32(len(target)))
		buf = append(buf, target...)
	}
	return buf, nil
}

func directoryBytes(n *node, nodes []*node) []byte {
	if len(n.children) == 0 {
		return nil
	}
	var buf []byte
	for _, child := range n.children {
		block, off := locateInode(child, nodes)
		// One header per entry keeps the inode delta inside a signed 16 bits
		// when a sibling directory has a large subtree.
		buf = appendU32(buf, 0)
		buf = appendU32(buf, block)
		buf = appendU32(buf, child.ino)
		buf = appendU16(buf, off)
		buf = appendU16(buf, 0)
		buf = appendU16(buf, uint16(inodeKind(child)))
		name := child.name
		buf = appendU16(buf, uint16(len(name)-1))
		buf = append(buf, name...)
	}
	return buf
}

func locateInode(n *node, nodes []*node) (uint32, uint16) {
	total := 0
	for _, other := range nodes {
		total += inodeLen(other)
	}
	pos := 0
	for _, other := range nodes {
		if other == n {
			return metaDiskOff(pos, total), uint16(pos % metaMax)
		}
		pos += inodeLen(other)
	}
	return 0, 0
}

func metaDiskOff(at, total int) uint32 {
	if total < at {
		total = at
	}
	var disk int
	for pos := 0; pos < at; {
		n := metaMax
		if pos+n > total {
			n = total - pos
		}
		if n <= 0 {
			break
		}
		if at < pos+n {
			return uint32(disk)
		}
		disk += 2 + n
		pos += n
	}
	return uint32(disk)
}

func inodeLen(n *node) int {
	switch {
	case n.entry.isSymlink():
		return headerSize + 8 + len(n.entry.Target)
	case n.entry.isFile():
		blocks := 0
		if len(n.entry.Data) > 0 {
			blocks = (len(n.entry.Data) + blockSize - 1) / blockSize
		}
		return headerSize + 16 + blocks*4
	default:
		return headerSize + 16
	}
}

func inodeKind(n *node) int {
	switch {
	case n.entry.isSymlink():
		return inodeLink
	case n.entry.isFile():
		return inodeFile
	default:
		return inodeDir
	}
}

func patchDirectory(inode []byte, stream []byte, at, listing int) error {
	size := listing + 3
	if size > 0xffff {
		return ErrSize
	}
	block, off := streamPos(stream, at)
	if at == len(stream) {
		// The empty listing sits on the metadata block that follows the directory table.
		block, off = packedLen(stream), 0
	}
	binary.LittleEndian.PutUint32(inode[headerSize:headerSize+4], block)
	binary.LittleEndian.PutUint16(inode[headerSize+8:headerSize+10], uint16(size))
	binary.LittleEndian.PutUint16(inode[headerSize+10:headerSize+12], off)
	return nil
}

func streamPos(stream []byte, at int) (uint32, uint16) {
	if at < 0 {
		at = 0
	}
	return packedOff(stream, at), uint16(at % metaMax)
}

func packedOff(stream []byte, at int) uint32 {
	var disk int
	for pos := 0; pos < at; {
		n := metaMax
		if pos+n > len(stream) {
			n = len(stream) - pos
		}
		if at < pos+n {
			return uint32(disk)
		}
		disk += 2 + n
		pos += n
	}
	return uint32(disk)
}

func packedLen(stream []byte) uint32 {
	if len(stream) == 0 {
		return 0
	}
	return packedOff(stream, len(stream))
}

func inodeRef(inodes []byte, n *node, table []byte) uint64 {
	block := packedOff(inodes, n.inoAt)
	off := uint16(n.inoAt % metaMax)
	_ = table
	return uint64(block)<<16 | uint64(off)
}

func packMeta(raw []byte) []byte {
	if len(raw) == 0 {
		return []byte{0x00, 0x80}
	}
	var disk []byte
	for pos := 0; pos < len(raw); {
		n := metaMax
		if pos+n > len(raw) {
			n = len(raw) - pos
		}
		var hdr [2]byte
		binary.LittleEndian.PutUint16(hdr[:], 0x8000|uint16(n))
		disk = append(disk, hdr[:]...)
		disk = append(disk, raw[pos:pos+n]...)
		pos += n
	}
	return disk
}

func unixPerm(mode fs.FileMode) uint16 {
	u := uint16(mode & 0o777)
	if mode&fs.ModeSetuid != 0 {
		u |= 0o4000
	}
	if mode&fs.ModeSetgid != 0 {
		u |= 0o2000
	}
	if mode&fs.ModeSticky != 0 {
		u |= 0o1000
	}
	return u
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

func appendU16(b []byte, v uint16) []byte {
	var buf [2]byte
	binary.LittleEndian.PutUint16(buf[:], v)
	return append(b, buf[:]...)
}

func appendU32(b []byte, v uint32) []byte {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	return append(b, buf[:]...)
}
