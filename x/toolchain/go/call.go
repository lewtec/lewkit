package gotool

import (
	"bytes"
	"context"
	"debug/gosym"
	"encoding/binary"
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/goobj"
	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/objabi"
)

// CompileSource compiles a package main given as source.
// The compile runs in this process. Linux reads the source from a
// memfd. Darwin and Windows read it from a temporary file. Bind calls
// the compiled functions in this process.
func CompileSource(ctx context.Context, src string) (*Prog, error) {
	if ctx == nil {
		return nil, ErrNilContext
	}
	compileMu.Lock()
	defer compileMu.Unlock()

	dir, err := os.MkdirTemp("", "lewkit-jit-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	cfg, _, err := runtimeDeps(dir)
	if err != nil {
		return nil, err
	}
	stored, err := storedGo(dir)
	if err != nil {
		return nil, err
	}
	if len(stored) != 0 {
		return nil, fmt.Errorf("jit: source was stored: %v", stored)
	}

	srcFile, err := openMem("src")
	if err != nil {
		return nil, err
	}
	defer srcFile.Close()
	if _, err := srcFile.Write([]byte(src)); err != nil {
		return nil, err
	}
	if _, err := srcFile.f.Seek(0, 0); err != nil {
		return nil, err
	}
	arc, err := os.CreateTemp(dir, "pkg-*.a")
	if err != nil {
		return nil, err
	}
	arcName := arc.Name()
	arc.Close()
	srcPath := srcFile.Path()
	if err := compileOnce("main", arcName, cfg, "", "go1.27", false, []string{srcPath}); err != nil {
		return nil, err
	}
	body, err := os.ReadFile(arcName)
	if err != nil {
		return nil, err
	}
	return loadProg(body)
}

// Prog is one compiled package. Its functions run in this process.
type Prog struct {
	mem []byte
	fn  map[string]uintptr
}

// Func returns the entry PC of a compiled function.
// name is the identifier, such as Value, or the symbol main.Value.
func (p *Prog) Func(name string) (uintptr, error) {
	if p == nil {
		return 0, fmt.Errorf("jit: nil program")
	}
	if pc, ok := p.fn[name]; ok {
		return pc, nil
	}
	if pc, ok := p.fn["main."+name]; ok {
		return pc, nil
	}
	return 0, fmt.Errorf("jit: no function %s", name)
}

// Bind returns the compiled function name as a Go function of type T.
func Bind[T any](p *Prog, name string) (T, error) {
	var zero T
	rt := reflect.TypeFor[T]()
	if rt == nil || rt.Kind() != reflect.Func {
		return zero, fmt.Errorf("jit: bind type is not a function")
	}
	pc, err := p.Func(name)
	if err != nil {
		return zero, err
	}
	fv := &struct{ fn uintptr }{fn: pc}
	slot := reflect.New(rt).Elem()
	*(*unsafe.Pointer)(slot.Addr().UnsafePointer()) = unsafe.Pointer(fv)
	return slot.Interface().(T), nil
}

var compileMu sync.Mutex

// compileOnce runs the in-process compiler. A second compile in this
// process finds the compiler tables already initialized and returns
// that failure instead of exiting the process.
func compileOnce(pkgpath, out, importcfg, embedcfg, lang string, shared bool, files []string) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("jit: compile: %v", rec)
		}
	}()
	return compileHere(pkgpath, out, importcfg, embedcfg, lang, shared, "", files)
}

func storedGo(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, ent := range ents {
		if strings.HasSuffix(ent.Name(), ".go") {
			out = append(out, ent.Name())
		}
	}
	return out, nil
}

type placed struct {
	addr uintptr
	size int
	kind objabi.SymKind
}

func loadProg(archive []byte) (*Prog, error) {
	obj, err := openObject(archive)
	if err != nil {
		return nil, err
	}
	n := definedCount(obj)
	place := make([]placed, n)
	off := 0
	for i := uint32(0); i < n; i++ {
		s := obj.Sym(i)
		kind := objabi.SymKind(s.Type())
		if !placeKind(kind) {
			continue
		}
		size := symbolSize(obj, i, s)
		if size == 0 {
			continue
		}
		align := 1
		if kind.IsText() {
			align = 16
		} else if a := int(s.Align()); a > align {
			align = a
		}
		off = alignUp(off, align)
		place[i] = placed{size: size, kind: kind}
		// addr filled after the mapping exists; remember the offset in size's neighbor via addr as offset for now
		place[i].addr = uintptr(off)
		off += size
	}
	if off == 0 {
		return nil, fmt.Errorf("jit: package has no functions")
	}
	hint, err := hostTextHint()
	if err != nil {
		return nil, err
	}
	// The tail holds an absolute jump for each call whose pc-relative
	// displacement does not reach the host text.
	trampAt := alignUp(off, 16)
	slots := relocSlots(obj, place)
	total := off
	if slots > 0 {
		total = trampAt + slots*16
	}
	mem, err := mapNear(total, hint)
	if err != nil {
		return nil, err
	}
	base := uintptr(unsafe.Pointer(&mem[0]))
	// Darwin's MAP_JIT write bit is per OS thread. The copy has to run
	// on the same thread that toggled pthread_jit_write_protect_np.
	if err := withMappedWrite(mem, func() error {
		for i := range place {
			if place[i].size == 0 {
				continue
			}
			place[i].addr = base + place[i].addr
			data := obj.Data(uint32(i))
			dst := mem[place[i].addr-base : place[i].addr-base+uintptr(place[i].size)]
			copy(dst, data)
		}
		var tr *trampoline
		if slots > 0 {
			tr = &trampoline{mem: mem[trampAt:], base: base + uintptr(trampAt)}
		}
		return patchRelocs(obj, place, tr)
	}); err != nil {
		return nil, err
	}
	prog := &Prog{mem: mem, fn: map[string]uintptr{}}
	var texts []textFn
	for i := uint32(0); i < n; i++ {
		if place[i].size == 0 || !place[i].kind.IsText() {
			continue
		}
		s := obj.Sym(i)
		if s.ABIWrapper() {
			continue
		}
		name := s.Name(obj)
		if name == "" {
			continue
		}
		if prev, ok := prog.fn[name]; ok && s.ABI() != 1 && prev != 0 {
			continue
		}
		prog.fn[name] = place[i].addr
		texts = append(texts, textFn{name: name, obj: obj, idx: i, addr: place[i].addr, size: place[i].size, sym: s})
	}
	if len(texts) == 0 {
		return nil, fmt.Errorf("jit: package has no functions")
	}
	if err := publishText(texts); err != nil {
		return nil, err
	}
	return prog, nil
}

// withMappedWrite pins this goroutine while fn stores into mem, then
// makes the mapping executable again. A failing fn still restores
// execute permission so the thread is not left in write mode.
func withMappedWrite(mem []byte, fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := prepareWrite(mem); err != nil {
		return err
	}
	err := fn()
	if rxErr := protectRX(mem); err == nil {
		err = rxErr
	}
	return err
}

func openObject(archive []byte) (*goobj.Reader, error) {
	magic := []byte(goobj.Magic)
	for i := 0; i+len(magic) <= len(archive); i++ {
		if !bytes.Equal(archive[i:i+len(magic)], magic) {
			continue
		}
		r := goobj.NewReaderFromBytes(archive[i:], false)
		if r == nil {
			return nil, fmt.Errorf("jit: object header is not go120ld")
		}
		return r, nil
	}
	return nil, fmt.Errorf("jit: archive has no object")
}

func definedCount(r *goobj.Reader) uint32 {
	return uint32(r.NSym() + r.NHashed64def() + r.NHasheddef() + r.NNonpkgdef())
}

func placeKind(k objabi.SymKind) bool {
	switch k {
	case objabi.STEXT, objabi.STEXTFIPS,
		objabi.SRODATA, objabi.SRODATAFIPS,
		objabi.SNOPTRDATA, objabi.SNOPTRDATAFIPS,
		objabi.SDATA, objabi.SDATAFIPS,
		objabi.SBSS, objabi.SNOPTRBSS:
		return true
	default:
		return false
	}
}

func symbolSize(r *goobj.Reader, i uint32, s *goobj.Sym) int {
	n := int(s.Siz())
	if d := r.DataSize(i); d > n {
		n = d
	}
	return n
}

func alignUp(off, align int) int {
	if align <= 1 {
		return off
	}
	return (off + align - 1) &^ (align - 1)
}

func relocSlots(obj *goobj.Reader, place []placed) int {
	n := 0
	for i := range place {
		if place[i].size == 0 {
			continue
		}
		n += obj.NReloc(uint32(i))
	}
	return n
}

func patchRelocs(obj *goobj.Reader, place []placed, tr *trampoline) error {
	for i := range place {
		if place[i].size == 0 {
			continue
		}
		symName := obj.Sym(uint32(i)).Name(obj)
		for j := 0; j < obj.NReloc(uint32(i)); j++ {
			rel := obj.Reloc(uint32(i), j)
			if rel.Siz() == 0 {
				continue
			}
			typ := objabi.RelocType(rel.Type())
			switch typ {
			case objabi.R_USETYPE, objabi.R_USEIFACE, objabi.R_USEIFACEMETHOD, objabi.R_USENAMEDMETHOD, objabi.R_KEEP:
				continue
			}
			target, err := relocTarget(obj, place, rel.Sym())
			if err != nil {
				return fmt.Errorf("jit: %s: %w", symName, err)
			}
			target += rel.Add()
			off := int(rel.Off())
			switch typ {
			case objabi.R_CALLARM64:
				if err := applyCallARM64(place[i], off, target, tr); err != nil {
					return fmt.Errorf("jit: %s: %s: %w", symName, typ, err)
				}
				continue
			case objabi.R_ADDRARM64, objabi.R_ARM64_PCREL_LDST8, objabi.R_ARM64_PCREL_LDST16, objabi.R_ARM64_PCREL_LDST32, objabi.R_ARM64_PCREL_LDST64:
				if err := applyAddrARM64(place[i], off, typ, target); err != nil {
					return fmt.Errorf("jit: %s: %s: %w", symName, typ, err)
				}
				continue
			}
			var o int64
			switch typ {
			case objabi.R_ADDR:
				o = target
			case objabi.R_CALL, objabi.R_PCREL:
				o = target - (int64(place[i].addr) + int64(off) + int64(rel.Siz()))
				if typ == objabi.R_CALL && rel.Siz() == 4 && o != int64(int32(o)) {
					dest, err := takeTramp(tr, target)
					if err != nil {
						return fmt.Errorf("jit: %s: %s: %w", symName, typ, err)
					}
					o = int64(dest) - (int64(place[i].addr) + int64(off) + int64(rel.Siz()))
				}
			case objabi.R_CONST:
				o = rel.Add()
			default:
				return fmt.Errorf("jit: %s: reloc %s", symName, typ)
			}
			if err := writeReloc(place[i], rel, o, typ); err != nil {
				name := ""
				if ref := rel.Sym(); !ref.IsZero() {
					name = symbolLabel(obj, ref)
				}
				return fmt.Errorf("jit: %s: %s %s: %w", symName, typ, name, err)
			}
		}
	}
	return nil
}

// trampoline is the absolute-jump tail of one JIT mapping.
type trampoline struct {
	mem  []byte
	base uintptr
	off  int
}

func (t *trampoline) take() (uintptr, error) {
	const n = 16
	if t == nil || t.off+n > len(t.mem) {
		return 0, fmt.Errorf("call distance does not fit")
	}
	addr := t.base + uintptr(t.off)
	t.off += n
	return addr, nil
}

func takeTramp(tr *trampoline, target int64) (uintptr, error) {
	dest, err := tr.take()
	if err != nil {
		return 0, err
	}
	if err := emitTrampoline(tr.mem, tr.base, dest, uint64(target)); err != nil {
		return 0, err
	}
	return dest, nil
}

func emitTrampoline(mem []byte, base, addr uintptr, target uint64) error {
	off := int(addr - base)
	if off < 0 || off+16 > len(mem) {
		return fmt.Errorf("trampoline outside mapping")
	}
	switch runtime.GOARCH {
	case "amd64":
		emitAMD64Trampoline(mem[off:], target)
	case "arm64":
		emitARM64Trampoline(mem[off:], target)
	default:
		return fmt.Errorf("no trampoline for %s", runtime.GOARCH)
	}
	return nil
}

// emitAMD64Trampoline writes movabs r11, target; jmp r11.
func emitAMD64Trampoline(dst []byte, target uint64) {
	dst[0] = 0x49
	dst[1] = 0xBB
	binary.LittleEndian.PutUint64(dst[2:], target)
	dst[10] = 0x41
	dst[11] = 0xFF
	dst[12] = 0xE3
}

// emitARM64Trampoline writes ldr x16, #8; br x16; the target address.
func emitARM64Trampoline(dst []byte, target uint64) {
	binary.LittleEndian.PutUint32(dst[0:], 0x58000050)
	binary.LittleEndian.PutUint32(dst[4:], 0xD61F0200)
	binary.LittleEndian.PutUint64(dst[8:], target)
}

func applyCallARM64(p placed, off int, target int64, tr *trampoline) error {
	dst := unsafe.Slice((*byte)(unsafe.Pointer(p.addr)), p.size)
	if off < 0 || off+4 > len(dst) {
		return fmt.Errorf("reloc outside symbol")
	}
	pc := int64(p.addr) + int64(off)
	delta := target - pc
	if delta >= 1<<27 || delta < -1<<27 {
		dest, err := takeTramp(tr, target)
		if err != nil {
			return err
		}
		delta = int64(dest) - pc
		if delta >= 1<<27 || delta < -1<<27 {
			return fmt.Errorf("trampoline distance %#x does not fit", delta)
		}
	}
	word := binary.LittleEndian.Uint32(dst[off:])
	word &^= 0x03ffffff
	word |= uint32(delta>>2) & 0x03ffffff
	binary.LittleEndian.PutUint32(dst[off:], word)
	return nil
}

func applyAddrARM64(p placed, off int, typ objabi.RelocType, target int64) error {
	dst := unsafe.Slice((*byte)(unsafe.Pointer(p.addr)), p.size)
	if off < 0 || off+8 > len(dst) {
		return fmt.Errorf("reloc outside symbol")
	}
	pc := int64(p.addr) + int64(off)
	delta := target - (pc &^ 0xfff)
	if delta >= 1<<32 || delta < -1<<32 {
		return fmt.Errorf("address %#x is out of adrp range", uint64(target))
	}
	o0 := binary.LittleEndian.Uint32(dst[off:])
	o1 := binary.LittleEndian.Uint32(dst[off+4:])
	o0 |= (uint32((delta>>12)&3) << 29) | (uint32((delta>>12>>2)&0x7ffff) << 5)
	low := delta & 0xfff
	switch typ {
	case objabi.R_ARM64_PCREL_LDST8, objabi.R_ADDRARM64:
		o1 |= uint32(low) << 10
	case objabi.R_ARM64_PCREL_LDST16:
		if low&1 != 0 {
			return fmt.Errorf("unaligned 16-bit offset %#x", low)
		}
		o1 |= uint32(low>>1) << 10
	case objabi.R_ARM64_PCREL_LDST32:
		if low&3 != 0 {
			return fmt.Errorf("unaligned 32-bit offset %#x", low)
		}
		o1 |= uint32(low>>2) << 10
	case objabi.R_ARM64_PCREL_LDST64:
		if low&7 != 0 {
			return fmt.Errorf("unaligned 64-bit offset %#x", low)
		}
		o1 |= uint32(low>>3) << 10
	default:
		return fmt.Errorf("reloc %s", typ)
	}
	binary.LittleEndian.PutUint32(dst[off:], o0)
	binary.LittleEndian.PutUint32(dst[off+4:], o1)
	return nil
}

func relocTarget(obj *goobj.Reader, place []placed, ref goobj.SymRef) (int64, error) {
	if ref.IsZero() {
		return 0, nil
	}
	if idx, ok := localIndex(obj, ref); ok {
		if int(idx) >= len(place) || place[idx].size == 0 {
			return 0, fmt.Errorf("reloc to unplaced %s", symbolLabel(obj, ref))
		}
		return int64(place[idx].addr), nil
	}
	if ref.PkgIdx == goobj.PkgIdxBuiltin {
		name, abi := goobj.BuiltinName(int(ref.SymIdx))
		pc, err := hostFunc(name, abi)
		if err != nil {
			return 0, err
		}
		return int64(pc), nil
	}
	if name, ok := refNames(obj)[ref]; ok {
		pc, err := hostFunc(name, 0)
		if err != nil {
			return 0, err
		}
		return int64(pc), nil
	}
	return 0, fmt.Errorf("unknown symbol %s", symbolLabel(obj, ref))
}

func localIndex(r *goobj.Reader, ref goobj.SymRef) (uint32, bool) {
	switch ref.PkgIdx {
	case goobj.PkgIdxSelf:
		if int(ref.SymIdx) < r.NSym() {
			return ref.SymIdx, true
		}
	case goobj.PkgIdxHashed64:
		if int(ref.SymIdx) < r.NHashed64def() {
			return uint32(r.NSym()) + ref.SymIdx, true
		}
	case goobj.PkgIdxHashed:
		if int(ref.SymIdx) < r.NHasheddef() {
			return uint32(r.NSym()+r.NHashed64def()) + ref.SymIdx, true
		}
	case goobj.PkgIdxNone:
		if int(ref.SymIdx) < r.NNonpkgdef() {
			return uint32(r.NSym()+r.NHashed64def()+r.NHasheddef()) + ref.SymIdx, true
		}
	}
	return 0, false
}

func symbolLabel(r *goobj.Reader, ref goobj.SymRef) string {
	if idx, ok := localIndex(r, ref); ok {
		return r.Sym(idx).Name(r)
	}
	if ref.PkgIdx == goobj.PkgIdxBuiltin {
		name, abi := goobj.BuiltinName(int(ref.SymIdx))
		return fmt.Sprintf("%s abi%d", name, abi)
	}
	if name, ok := refNames(r)[ref]; ok {
		return name
	}
	return fmt.Sprintf("%d:%d", ref.PkgIdx, ref.SymIdx)
}

func refNames(r *goobj.Reader) map[goobj.SymRef]string {
	out := make(map[goobj.SymRef]string, r.NRefName())
	for i := 0; i < r.NRefName(); i++ {
		n := r.RefName(i)
		out[n.Sym()] = n.Name(r)
	}
	return out
}

func writeReloc(p placed, rel *goobj.Reloc, o int64, typ objabi.RelocType) error {
	off := int(rel.Off())
	siz := int(rel.Siz())
	if off < 0 || siz < 0 || off+siz > p.size {
		return fmt.Errorf("reloc outside symbol")
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(p.addr)), p.size)
	switch siz {
	case 1:
		if o != int64(int8(o)) {
			return fmt.Errorf("address %#x does not fit", o)
		}
		dst[off] = byte(o)
	case 2:
		if !fits(o, 16, typ) {
			return fmt.Errorf("address %#x does not fit", o)
		}
		binary.LittleEndian.PutUint16(dst[off:], uint16(o))
	case 4:
		if !fits(o, 32, typ) {
			return fmt.Errorf("address %#x does not fit", o)
		}
		binary.LittleEndian.PutUint32(dst[off:], uint32(o))
	case 8:
		binary.LittleEndian.PutUint64(dst[off:], uint64(o))
	default:
		return fmt.Errorf("reloc size %d", siz)
	}
	return nil
}

func fits(o int64, bits int, typ objabi.RelocType) bool {
	if bits == 32 && (typ == objabi.R_CALL || typ == objabi.R_PCREL) {
		return o == int64(int32(o))
	}
	if bits == 16 && (typ == objabi.R_CALL || typ == objabi.R_PCREL) {
		return o == int64(int16(o))
	}
	switch bits {
	case 16:
		return o == int64(int16(o)) || o == int64(uint16(o))
	case 32:
		return o == int64(int32(o)) || o == int64(uint32(o))
	default:
		return false
	}
}

type textFn struct {
	name string
	obj  *goobj.Reader
	idx  uint32
	addr uintptr
	size int
	sym  *goobj.Sym
}

func publishText(fns []textFn) error {
	if len(fns) == 0 {
		return nil
	}
	text := fns[0].addr
	end := fns[0].addr + uintptr(fns[0].size)
	for _, fn := range fns[1:] {
		if fn.addr < text {
			text = fn.addr
		}
		if fn.addr+uintptr(fn.size) > end {
			end = fn.addr + uintptr(fn.size)
		}
	}
	names := []byte{0}
	var pctab []byte
	pctab = append(pctab, 0)
	var pcln []byte
	var gofunc []byte
	ftab := make([]functab, 0, len(fns)+1)
	for _, fn := range fns {
		nameOff := len(names)
		names = append(names, fn.name...)
		names = append(names, 0)

		pcspOff, pcdataOffs, err := appendPCData(&pctab, fn)
		if err != nil {
			return err
		}
		fdOffs := appendFuncData(&gofunc, fn)

		rec := encodeFunc(uint32(fn.addr-text), int32(nameOff), pcspOff, pcdataOffs, fdOffs)
		if len(pcln)%8 != 0 {
			pcln = append(pcln, make([]byte, 8-len(pcln)%8)...)
		}
		ftab = append(ftab, functab{entryoff: uint32(fn.addr - text), funcoff: uint32(len(pcln))})
		pcln = append(pcln, rec...)
	}
	ftab = append(ftab, functab{entryoff: uint32(end - text)})

	span := end - text
	nbucket := int((span + funcTabBucket - 1) / funcTabBucket)
	if nbucket < 1 {
		nbucket = 1
	}
	find := make([]findfuncbucket, nbucket)

	hdr := &pcHeader{
		magic:   0xfffffff1,
		minLC:   1,
		ptrSize: 8,
		nfunc:   len(fns),
	}
	md := &moduleData{
		pcHeader:    hdr,
		funcnametab: names,
		pctab:       pctab,
		pclntable:   pcln,
		ftab:        ftab,
		findfunctab: uintptr(unsafe.Pointer(&find[0])),
		minpc:       text,
		maxpc:       end,
		text:        text,
		etext:       end,
		gofunc:      gofuncPtr(gofunc),
		modulename:  "jit",
	}
	moduleMu.Lock()
	defer moduleMu.Unlock()
	if err := hostModuleLayout(); err != nil {
		return err
	}
	modulePin = append(modulePin, pinned{md: md, find: find, gofunc: gofunc})
	lastmoduledatap.next = md
	lastmoduledatap = md
	return nil
}

func hostModuleLayout() error {
	if lastmoduledatap == nil || lastmoduledatap.pcHeader == nil {
		return fmt.Errorf("jit: runtime module list is empty")
	}
	if lastmoduledatap.pcHeader.magic != 0xfffffff1 || lastmoduledatap.pcHeader.ptrSize != 8 {
		return fmt.Errorf("jit: runtime module header is %x", lastmoduledatap.pcHeader.magic)
	}
	pc, err := hostFunc("runtime.morestack_noctxt", 0)
	if err != nil {
		return err
	}
	if pc < lastmoduledatap.minpc || pc >= lastmoduledatap.maxpc {
		return fmt.Errorf("jit: runtime text %x-%x does not cover morestack %x", lastmoduledatap.minpc, lastmoduledatap.maxpc, pc)
	}
	return nil
}

func gofuncPtr(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&b[0]))
}

func appendPCData(pctab *[]byte, fn textFn) (uint32, []uint32, error) {
	var pcsp []byte
	var pcdata [][]byte
	for j := 0; j < fn.obj.NAux(fn.idx); j++ {
		a := fn.obj.Aux(fn.idx, j)
		switch a.Type() {
		case goobj.AuxPcsp:
			pcsp = auxData(fn.obj, a.Sym())
		case goobj.AuxPcdata:
			pcdata = append(pcdata, auxData(fn.obj, a.Sym()))
		}
	}
	if len(pcsp) == 0 {
		pcsp = pcvalue(0, fn.size)
	}
	if len(pcdata) == 0 && fn.sym.NoSplit() {
		// A nosplit leaf has no preemption point. The whole body is unsafe.
		pcdata = [][]byte{pcvalue(-2, fn.size)}
	}
	pcspOff := uint32(len(*pctab))
	*pctab = append(*pctab, pcsp...)
	offs := make([]uint32, len(pcdata))
	for i, tab := range pcdata {
		if len(tab) == 0 {
			continue
		}
		offs[i] = uint32(len(*pctab))
		*pctab = append(*pctab, tab...)
	}
	return pcspOff, offs, nil
}

func appendFuncData(blob *[]byte, fn textFn) []uint32 {
	var offs []uint32
	for j := 0; j < fn.obj.NAux(fn.idx); j++ {
		a := fn.obj.Aux(fn.idx, j)
		if a.Type() != goobj.AuxFuncdata {
			continue
		}
		data := auxData(fn.obj, a.Sym())
		if len(data) == 0 {
			offs = append(offs, ^uint32(0))
			continue
		}
		*blob = alignBytes(*blob, 8)
		offs = append(offs, uint32(len(*blob)))
		*blob = append(*blob, data...)
	}
	return offs
}

func auxData(r *goobj.Reader, ref goobj.SymRef) []byte {
	idx, ok := localIndex(r, ref)
	if !ok {
		return nil
	}
	return r.Data(idx)
}

func alignBytes(b []byte, n int) []byte {
	if pad := len(b) % n; pad != 0 {
		b = append(b, make([]byte, n-pad)...)
	}
	return b
}

// pcvalue encodes one pc-value step: val holds for size bytes from entry.
// The runtime starts the running value at -1.
func pcvalue(val int32, size int) []byte {
	delta := int64(val) - (-1)
	var buf []byte
	buf = appendVarint(buf, zigzag(delta))
	buf = appendVarint(buf, uint32(size))
	buf = append(buf, 0)
	return buf
}

func zigzag(v int64) uint32 {
	return uint32((v << 1) ^ (v >> 63))
}

func appendVarint(b []byte, v uint32) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func encodeFunc(entryOff uint32, nameOff int32, pcsp uint32, pcdata, funcdata []uint32) []byte {
	var b []byte
	put32 := func(v uint32) {
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], v)
		b = append(b, buf[:]...)
	}
	put32(entryOff)
	put32(uint32(nameOff))
	put32(0) // args
	put32(0) // deferreturn
	put32(pcsp)
	put32(0) // pcfile
	put32(0) // pcln
	put32(uint32(len(pcdata)))
	put32(^uint32(0)) // cuOffset
	put32(0)          // startLine
	b = append(b, 0, 0, 0, byte(len(funcdata)))
	for _, off := range pcdata {
		put32(off)
	}
	for _, off := range funcdata {
		put32(off)
	}
	return b
}

const funcTabBucket = 4096

// Layout matches runtime.moduledata on linux/amd64.
// The leading not-in-heap marker has size zero, so pcHeader is first.
type moduleData struct {
	pcHeader    *pcHeader
	funcnametab []byte
	cutab       []uint32
	filetab     []byte
	pctab       []byte
	pclntable   []byte
	ftab        []functab
	findfunctab uintptr
	minpc       uintptr
	maxpc       uintptr

	text, etext                uintptr
	noptrdata, enoptrdata      uintptr
	data, edata                uintptr
	bss, ebss                  uintptr
	noptrbss, enoptrbss        uintptr
	covctrs, ecovctrs          uintptr
	end, gcdata, gcbss         uintptr
	types, typedesclen, etypes uintptr
	itaboffset, itabsize       uintptr
	rodata                     uintptr
	gofunc                     uintptr
	epclntab                   uintptr

	textsectmap []textsect
	ptab        []ptabEntry

	pluginpath string
	pkghashes  []moduleHash

	inittasks []*struct{}

	modulename   string
	modulehashes []moduleHash

	hasmain uint8
	bad     bool

	gcdatamask, gcbssmask bitvector

	typemap map[*byte]*byte
	next    *moduleData
}

type pcHeader struct {
	magic          uint32
	pad1, pad2     uint8
	minLC          uint8
	ptrSize        uint8
	nfunc          int
	nfiles         uint
	_              uintptr
	funcnameOffset uintptr
	cuOffset       uintptr
	filetabOffset  uintptr
	pctabOffset    uintptr
	pclnOffset     uintptr
}

type functab struct {
	entryoff uint32
	funcoff  uint32
}

type textsect struct {
	vaddr    uintptr
	end      uintptr
	baseaddr uintptr
}

type ptabEntry struct {
	name int32
	typ  int32
}

type moduleHash struct {
	modulename   string
	linktimehash string
	runtimehash  *string
}

type bitvector struct {
	n        int32
	bytedata *uint8
}

type findfuncbucket struct {
	idx        uint32
	subbuckets [16]byte
}

type pinned struct {
	md     *moduleData
	find   []findfuncbucket
	gofunc []byte
}

var (
	moduleMu  sync.Mutex
	modulePin []pinned
)

//go:linkname lastmoduledatap runtime.lastmoduledatap
var lastmoduledatap *moduleData

type hostSyms struct {
	bias int64
	fn   map[string][]uint64
}

var (
	hostOnce sync.Once
	hostSym  hostSyms
	hostErr  error
)

func hostFunc(name string, abi int) (uintptr, error) {
	hostOnce.Do(func() { hostSym, hostErr = readHostSyms() })
	if hostErr != nil {
		return 0, hostErr
	}
	pcs := hostSym.fn[name]
	if len(pcs) == 0 && abi == 0 {
		pcs = hostSym.fn[name+".abi0"]
	}
	if len(pcs) == 0 {
		return 0, fmt.Errorf("unknown symbol %s", name)
	}
	return uintptr(int64(pcs[0]) + hostSym.bias), nil
}

func hostTextHint() (uintptr, error) {
	return hostFunc("runtime.morestack_noctxt", 0)
}

// readHostSyms reads the pclntab of the main module already mapped in
// this process. lastmoduledatap is that module until a JIT image is
// published, and hostFunc caches the result of this first read.
// The header and epclntab bound the table. The text base is the runtime
// address, so the returned entries need no load bias.
func readHostSyms() (hostSyms, error) {
	data, text, err := modulePcln(lastmoduledatap)
	if err != nil {
		return hostSyms{}, err
	}
	tab, err := gosym.NewTable(nil, gosym.NewLineTable(data, uint64(text)))
	if err != nil {
		return hostSyms{}, err
	}
	if len(tab.Funcs) == 0 {
		return hostSyms{}, fmt.Errorf("jit: executable has no pclntab")
	}
	out := hostSyms{fn: make(map[string][]uint64, len(tab.Funcs))}
	for i := range tab.Funcs {
		fn := &tab.Funcs[i]
		out.fn[fn.Sym.Name] = append(out.fn[fn.Sym.Name], fn.Entry)
	}
	if len(out.fn["runtime.morestack_noctxt"]) == 0 && len(out.fn["runtime.morestack_noctxt.abi0"]) == 0 {
		return hostSyms{}, fmt.Errorf("jit: executable has no runtime.morestack_noctxt")
	}
	return out, nil
}

func modulePcln(md *moduleData) ([]byte, uintptr, error) {
	if md == nil || md.pcHeader == nil || md.text == 0 || md.epclntab == 0 {
		return nil, 0, fmt.Errorf("jit: executable has no pclntab")
	}
	start := uintptr(unsafe.Pointer(md.pcHeader))
	if md.epclntab <= start {
		return nil, 0, fmt.Errorf("jit: executable has no pclntab")
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(start)), md.epclntab-start), md.text, nil
}
