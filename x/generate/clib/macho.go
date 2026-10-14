package clib

import (
	"bytes"
	"crypto/sha256"
	"debug/macho"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	mhMagic64      = 0xfeedfacf
	mhDylib        = 6
	mhPIE          = 0x200000
	lcSegment64    = 0x19
	lcIDDylib      = 0xd
	lcLoadDylin    = 0xe
	lcUnixThread   = 0x5
	lcMain         = 0x80000028
	lcDyldInfo     = 0x22
	lcDyldInfoOnly = 0x80000022
	lcCodeSig      = 0x1d

	csPageBits      = 12
	csPage          = 1 << csPageBits
	csMagicEmbedded = 0xfade0cc0
	csMagicDir      = 0xfade0c02
	csSlotDir       = 0
	csHashSHA256    = 2
	csAdhoc         = 0x20002
)

// MakeDylib turns an internal-link Darwin PIE into a dylib dyld can map.
// The filetype becomes MH_DYLIB, LC_ID_DYLIB is added, and the executable-only
// load commands are dropped. An export trie names the C functions. The
// ad-hoc signature is rewritten because the header lives in __TEXT.
func MakeDylib(path string, names []string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if len(buf) < 32 || binary.LittleEndian.Uint32(buf) != mhMagic64 {
		return fmt.Errorf("clib: %s %w", path, ErrIsNotA64BitLittleEndianMachOFile)
	}
	mf, err := macho.NewFile(bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrMakeDylibClib, path, err)
	}
	defer mf.Close()
	textAddr, linkOff, linkVM, linkFile, ok := machoSegs(mf)
	if !ok {
		return fmt.Errorf("clib: %s %w", path, ErrHasNoTextOrLinkedit)
	}
	syms := map[string]uint64{}
	if mf.Symtab != nil {
		for _, sym := range mf.Symtab.Syms {
			if sym.Sect == 0 || sym.Value < textAddr {
				continue
			}
			syms[sym.Name] = sym.Value - textAddr
		}
	}
	exports := make([]exportSym, 0, len(names))
	for _, name := range names {
		addr, found := syms["_"+name]
		if !found {
			addr, found = syms[name]
		}
		if !found {
			return fmt.Errorf("clib: %s: export %s %w", path, name, ErrIsNotInTheSymbolTable)
		}
		exports = append(exports, exportSym{name: name, addr: addr})
	}
	trie := exportTrie(exports)
	contentEnd := len(buf)
	for _, raw := range loadRaws(mf) {
		if binary.LittleEndian.Uint32(raw) == lcCodeSig && len(raw) >= 16 {
			off := int(binary.LittleEndian.Uint32(raw[8:]))
			if off > 0 && off < contentEnd {
				contentEnd = off
			}
		}
	}
	if contentEnd < 32 {
		return fmt.Errorf("clib: %s %w", path, ErrCodeSignatureOverlapsTheHeader)
	}
	body := append([]byte{}, buf[:contentEnd]...)
	for len(body)%8 != 0 {
		body = append(body, 0)
	}
	trieOff := len(body)
	body = append(body, trie...)
	for len(body)%16 != 0 {
		body = append(body, 0)
	}
	codeSize := len(body)
	id := filepath.Base(path)
	sigSize := machoSigSize(codeSize, id)
	linkFile = uint64(codeSize+sigSize) - linkOff
	if linkVM < linkFile {
		linkVM = (linkFile + 0xfff) &^ 0xfff
	}
	cmds, err := machoCommands(mf, id, trieOff, len(trie), codeSize, sigSize, linkFile, linkVM)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrMakeDylibClib, path, err)
	}
	limit := sectionLimit(mf)
	if 32+len(cmds) > limit {
		return fmt.Errorf("clib: %s %w", path, ErrHasNoRoomForDylibLoadCommands)
	}
	binary.LittleEndian.PutUint32(body[12:], mhDylib)
	binary.LittleEndian.PutUint32(body[16:], uint32(len(cmds)))
	sizeof := 0
	for _, c := range cmds {
		sizeof += len(c)
	}
	binary.LittleEndian.PutUint32(body[20:], uint32(sizeof))
	flags := binary.LittleEndian.Uint32(body[24:])
	flags &^= mhPIE
	binary.LittleEndian.PutUint32(body[24:], flags)
	at := 32
	for _, c := range cmds {
		copy(body[at:], c)
		at += len(c)
	}
	oldEnd := 32 + int(binary.LittleEndian.Uint32(buf[20:]))
	if oldEnd > len(body) {
		oldEnd = len(body)
	}
	for i := at; i < oldEnd && i < limit; i++ {
		body[i] = 0
	}
	sig := make([]byte, sigSize)
	textOff, textSize := machoText(mf)
	machoSign(sig, bytes.NewReader(body[:codeSize]), id, int64(codeSize), textOff, textSize)
	body = append(body, sig...)
	return os.WriteFile(path, body, info.Mode())
}

func machoSegs(f *macho.File) (textAddr, linkOff, linkVM, linkFile uint64, ok bool) {
	var text, link bool
	for _, load := range f.Loads {
		seg, isSeg := load.(*macho.Segment)
		if !isSeg {
			continue
		}
		if seg.Name == "__TEXT" {
			textAddr = seg.Addr
			text = true
		}
		if seg.Name == "__LINKEDIT" {
			linkOff = seg.Offset
			linkVM = seg.Memsz
			linkFile = seg.Filesz
			link = true
		}
	}
	return textAddr, linkOff, linkVM, linkFile, text && link
}

func machoText(f *macho.File) (off, size int64) {
	for _, load := range f.Loads {
		seg, isSeg := load.(*macho.Segment)
		if isSeg && seg.Name == "__TEXT" {
			return int64(seg.Offset), int64(seg.Filesz)
		}
	}
	return 0, 0
}

func sectionLimit(f *macho.File) int {
	limit := int(^uint(0) >> 1)
	for _, s := range f.Sections {
		if s.Offset > 0 && int(s.Offset) < limit {
			limit = int(s.Offset)
		}
	}
	for _, load := range f.Loads {
		seg, isSeg := load.(*macho.Segment)
		if isSeg && seg.Offset > 0 && int(seg.Offset) < limit {
			limit = int(seg.Offset)
		}
	}
	return limit
}

func loadRaws(f *macho.File) [][]byte {
	out := make([][]byte, 0, len(f.Loads))
	for _, load := range f.Loads {
		out = append(out, append([]byte(nil), load.Raw()...))
	}
	return out
}

func machoCommands(f *macho.File, id string, trieOff, trieSize, codeSize, sigSize int, linkFile, linkVM uint64) ([][]byte, error) {
	var cmds [][]byte
	var sawInfo, sawSig bool
	for _, raw := range loadRaws(f) {
		if len(raw) < 8 {
			return nil, ErrShortLoadCommand
		}
		cmd := binary.LittleEndian.Uint32(raw)
		switch cmd {
		case lcMain, lcUnixThread, lcLoadDylin:
			continue
		case lcDyldInfo, lcDyldInfoOnly:
			if len(raw) < 48 {
				return nil, ErrShortDyldInfoCommand
			}
			binary.LittleEndian.PutUint32(raw[40:], uint32(trieOff))
			binary.LittleEndian.PutUint32(raw[44:], uint32(trieSize))
			sawInfo = true
		case lcCodeSig:
			if len(raw) < 16 {
				return nil, ErrShortCodeSignatureCommand
			}
			binary.LittleEndian.PutUint32(raw[8:], uint32(codeSize))
			binary.LittleEndian.PutUint32(raw[12:], uint32(sigSize))
			sawSig = true
		case lcSegment64:
			if len(raw) >= 56 && cstring(raw[8:24]) == "__LINKEDIT" {
				binary.LittleEndian.PutUint64(raw[32:], linkVM)
				binary.LittleEndian.PutUint64(raw[48:], linkFile)
			}
		}
		cmds = append(cmds, raw)
	}
	if !sawInfo {
		return nil, ErrNoLcDyldInfoCommand
	}
	idCmd := idDylib(id)
	cmds = append([][]byte{idCmd}, cmds...)
	if !sawSig {
		sig := make([]byte, 16)
		binary.LittleEndian.PutUint32(sig[0:], lcCodeSig)
		binary.LittleEndian.PutUint32(sig[4:], 16)
		binary.LittleEndian.PutUint32(sig[8:], uint32(codeSize))
		binary.LittleEndian.PutUint32(sig[12:], uint32(sigSize))
		cmds = append(cmds, sig)
	}
	return cmds, nil
}

func idDylib(name string) []byte {
	name = "@rpath/" + name
	size := (24 + len(name) + 1 + 7) &^ 7
	buf := make([]byte, size)
	binary.LittleEndian.PutUint32(buf[0:], lcIDDylib)
	binary.LittleEndian.PutUint32(buf[4:], uint32(size))
	binary.LittleEndian.PutUint32(buf[8:], 24)
	binary.LittleEndian.PutUint32(buf[16:], 0x10000)
	binary.LittleEndian.PutUint32(buf[20:], 0x10000)
	copy(buf[24:], name)
	return buf
}

func cstring(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return string(b)
}

type exportSym struct {
	name string
	addr uint64
}

type trieNode struct {
	off  int
	term bool
	addr uint64
	kids []*trieEdge
}

type trieEdge struct {
	label string
	node  *trieNode
}

func exportTrie(syms []exportSym) []byte {
	root := &trieNode{}
	ordered := append([]exportSym(nil), syms...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].name < ordered[j].name })
	for _, sym := range ordered {
		if sym.name != "" {
			root.insert(sym.name, sym.addr)
		}
	}
	width := 2
	if root.total(width) > 0x3fff {
		width = 3
	}
	var order []*trieNode
	at := 0
	var place func(*trieNode)
	place = func(n *trieNode) {
		n.off = at
		at += n.encodedSize(width)
		order = append(order, n)
		for _, kid := range n.kids {
			place(kid.node)
		}
	}
	place(root)
	buf := make([]byte, at)
	for _, n := range order {
		n.write(buf[n.off:], width)
	}
	return buf
}

func (n *trieNode) insert(name string, addr uint64) {
	if name == "" {
		n.term = true
		n.addr = addr
		return
	}
	ch := name[:1]
	for _, kid := range n.kids {
		if kid.label == ch {
			kid.node.insert(name[1:], addr)
			return
		}
	}
	child := &trieNode{}
	n.kids = append(n.kids, &trieEdge{label: ch, node: child})
	sort.Slice(n.kids, func(i, j int) bool { return n.kids[i].label < n.kids[j].label })
	child.insert(name[1:], addr)
}

func (n *trieNode) total(width int) int {
	sum := n.encodedSize(width)
	for _, kid := range n.kids {
		sum += kid.node.total(width)
	}
	return sum
}

func (n *trieNode) payload() []byte {
	if !n.term {
		return nil
	}
	var buf []byte
	buf = appendUleb(buf, 0)
	return appendUleb(buf, n.addr)
}

func (n *trieNode) encodedSize(width int) int {
	payload := n.payload()
	sz := ulebLen(uint64(len(payload))) + len(payload) + 1
	for _, kid := range n.kids {
		sz += len(kid.label) + 1 + width
	}
	return sz
}

func (n *trieNode) write(dst []byte, width int) {
	payload := n.payload()
	i := putUleb(dst, 0, uint64(len(payload)))
	i += copy(dst[i:], payload)
	dst[i] = byte(len(n.kids))
	i++
	for _, kid := range n.kids {
		i += copy(dst[i:], kid.label)
		dst[i] = 0
		i++
		i = putUlebWidth(dst, i, uint64(kid.node.off), width)
	}
}

func putUleb(dst []byte, i int, v uint64) int {
	for {
		c := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			c |= 0x80
		}
		dst[i] = c
		i++
		if v == 0 {
			return i
		}
	}
}

func putUlebWidth(dst []byte, i int, v uint64, width int) int {
	for n := 0; n < width; n++ {
		c := byte(v & 0x7f)
		v >>= 7
		if n+1 != width {
			c |= 0x80
		}
		dst[i] = c
		i++
	}
	return i
}

func ulebLen(v uint64) int {
	n := 1
	for v >= 0x80 {
		v >>= 7
		n++
	}
	return n
}

func appendUleb(dst []byte, v uint64) []byte {
	for {
		c := byte(v & 0x7f)
		v >>= 7
		if v != 0 {
			c |= 0x80
		}
		dst = append(dst, c)
		if v == 0 {
			return dst
		}
	}
}

func machoSigSize(codeSize int, id string) int {
	nhashes := (codeSize + csPage - 1) / csPage
	idOff := 13*4 + 4 + 4*8
	hashOff := idOff + len(id) + 1
	cdir := hashOff + nhashes*sha256.Size
	return 3*4 + 2*4 + cdir
}

func machoSign(out []byte, data io.Reader, id string, codeSize, textOff, textSize int64) {
	nhashes := (codeSize + csPage - 1) / csPage
	idOff := 13*4 + 4 + 4*8
	hashOff := idOff + len(id) + 1
	sz := len(out)
	p := out
	p = put32be(p, csMagicEmbedded)
	p = put32be(p, uint32(sz))
	p = put32be(p, 1)
	p = put32be(p, csSlotDir)
	p = put32be(p, uint32(3*4+2*4))
	p = put32be(p, csMagicDir)
	p = put32be(p, uint32(sz-(3*4+2*4)))
	p = put32be(p, 0x20400)
	p = put32be(p, csAdhoc)
	p = put32be(p, uint32(hashOff))
	p = put32be(p, uint32(idOff))
	p = put32be(p, 0)
	p = put32be(p, uint32(nhashes))
	p = put32be(p, uint32(codeSize))
	p = put8(p, sha256.Size)
	p = put8(p, csHashSHA256)
	p = put8(p, 0)
	p = put8(p, csPageBits)
	p = put32be(p, 0)
	p = put32be(p, 0)
	p = put32be(p, 0)
	p = put32be(p, 0)
	p = put64be(p, uint64(codeSize))
	p = put64be(p, uint64(textOff))
	p = put64be(p, uint64(textSize))
	p = put64be(p, 0)
	n := copy(p, id)
	p = p[n:]
	p[0] = 0
	p = p[1:]
	var buf [csPage]byte
	filled := 0
	for filled < int(codeSize) {
		want := csPage
		if remaining := int(codeSize) - filled; remaining < want {
			want = remaining
		}
		if _, err := io.ReadFull(data, buf[:want]); err != nil {
			return
		}
		sum := sha256.Sum256(buf[:want])
		copy(p, sum[:])
		p = p[sha256.Size:]
		filled += want
	}
}

func put32be(b []byte, x uint32) []byte {
	binary.BigEndian.PutUint32(b, x)
	return b[4:]
}

func put64be(b []byte, x uint64) []byte {
	binary.BigEndian.PutUint64(b, x)
	return b[8:]
}

func put8(b []byte, x byte) []byte {
	b[0] = x
	return b[1:]
}

func machoExportTrie(path string) ([]byte, error) {
	f, err := macho.Open(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrMachoExportTrieClib, path, err)
	}
	defer f.Close()
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	for _, load := range f.Loads {
		cmdb := load.Raw()
		if len(cmdb) < 48 {
			continue
		}
		cmd := binary.LittleEndian.Uint32(cmdb)
		if cmd != lcDyldInfo && cmd != lcDyldInfoOnly {
			continue
		}
		off := binary.LittleEndian.Uint32(cmdb[40:])
		sz := binary.LittleEndian.Uint32(cmdb[44:])
		if sz == 0 || uint64(off)+uint64(sz) > uint64(len(raw)) {
			return nil, fmt.Errorf("clib: %s %w", path, ErrExportTrieIsEmpty)
		}
		return raw[off : off+sz], nil
	}
	return nil, fmt.Errorf("clib: %s %w", path, ErrHasNoExportTrie)
}

func trieLookup(buf []byte, name string) (uint64, bool) {
	return trieWalk(buf, 0, name)
}

func trieWalk(buf []byte, off int, name string) (uint64, bool) {
	if off < 0 || off >= len(buf) {
		return 0, false
	}
	term, n := readUleb(buf[off:])
	p := off + n
	if term > 0 {
		if p+int(term) > len(buf) {
			return 0, false
		}
		payload := buf[p : p+int(term)]
		p += int(term)
		_, n1 := readUleb(payload)
		addr, _ := readUleb(payload[n1:])
		if name == "" {
			return addr, true
		}
	} else if name == "" {
		return 0, false
	}
	if p >= len(buf) {
		return 0, false
	}
	count := int(buf[p])
	p++
	for i := 0; i < count; i++ {
		end := p
		for end < len(buf) && buf[end] != 0 {
			end++
		}
		if end >= len(buf) {
			return 0, false
		}
		edge := string(buf[p:end])
		p = end + 1
		child, cn := readUleb(buf[p:])
		p += cn
		if strings.HasPrefix(name, edge) {
			return trieWalk(buf, int(child), name[len(edge):])
		}
	}
	return 0, false
}

func readUleb(buf []byte) (uint64, int) {
	var v uint64
	for i := 0; i < len(buf) && i < 10; i++ {
		v |= uint64(buf[i]&0x7f) << (7 * i)
		if buf[i]&0x80 == 0 {
			return v, i + 1
		}
	}
	return 0, 0
}
