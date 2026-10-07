//go:build windows && (amd64 || arm64)

package d3d12

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"unsafe"
)

// Device runs compute kernels. It does not attach a window and it does not
// share a screen's queue.
type Device struct {
	mu     sync.Mutex
	eng    *engine
	closed bool
}

// Pipeline is one compiled compute kernel.
type Pipeline struct {
	dev     *Device
	pso     uintptr
	root    uintptr
	srvs    int
	threads int
	closed  bool
}

// Buffer is a storage buffer. Store writes the upload heap. Dispatch copies
// it onto the default heap before the kernel reads it.
type Buffer struct {
	dev    *Device
	size   int
	gpu    int
	def    uintptr
	up     uintptr
	read   uintptr
	upPtr  uintptr
	addr   uintptr
	state  uint32
	dirty  bool
	closed bool
}

// OpenDevice opens the default adapter for compute.
func OpenDevice() (*Device, error) {
	eng, err := newEngine(listCompute, false)
	if err != nil {
		return nil, err
	}
	slog.Debug("d3d12 device", "adapter", eng.name, "level", eng.level)
	return &Device{eng: eng}, nil
}

// Name is the adapter description.
func (d *Device) Name() string {
	if d == nil || d.eng == nil {
		return ""
	}
	return d.eng.name
}

// Close releases the device. Buffers still release their own resources.
func (d *Device) Close() error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	if d.eng != nil {
		_ = d.eng.wait()
		d.eng.release()
		d.eng = nil
	}
	return nil
}

// Buffer allocates a default resource plus upload and readback heaps.
func (d *Device) Buffer(size int) (*Buffer, error) {
	if d == nil {
		return nil, ErrClosed
	}
	if size < 1 {
		return nil, ErrSize
	}
	n := size
	if n < 4 {
		n = 4
	}
	n = (n + 3) &^ 3
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.eng == nil {
		return nil, ErrClosed
	}
	def, err := d.eng.committed(heapDefault, stateCopyDst, bufferDesc(n, flagUAV))
	if err != nil {
		return nil, err
	}
	up, err := d.eng.committed(heapUpload, stateGeneric, bufferDesc(n, 0))
	if err != nil {
		release(def)
		return nil, err
	}
	var ptr uintptr
	empty := cpuRange{}
	var p pins
	p.keep(&empty)
	p.keep(&ptr)
	err = callHR(up, slotResMap, 0, uintptr(unsafe.Pointer(&empty)), uintptr(unsafe.Pointer(&ptr)))
	p.done()
	if err != nil || ptr == 0 {
		release(up)
		release(def)
		if err == nil {
			err = ErrUnavailable
		}
		return nil, err
	}
	back, err := d.eng.committed(heapReadback, stateCopyDst, bufferDesc(n, 0))
	if err != nil {
		syscallV(up, slotResUnmap, 0, 0)
		release(up)
		release(def)
		return nil, err
	}
	return &Buffer{
		dev:   d,
		size:  size,
		gpu:   n,
		def:   def,
		up:    up,
		read:  back,
		upPtr: ptr,
		addr:  syscallV(def, slotResVA),
		state: stateCopyDst,
	}, nil
}

// Len is the size requested at Buffer, in bytes.
func (b *Buffer) Len() int {
	if b == nil {
		return 0
	}
	return b.size
}

// Store copies p into the upload heap. The default heap updates on Dispatch or Read.
func (b *Buffer) Store(p []byte) error {
	if b == nil || b.dev == nil {
		return ErrClosed
	}
	if len(p) > b.size {
		return ErrSize
	}
	b.dev.mu.Lock()
	defer b.dev.mu.Unlock()
	if b.closed || b.upPtr == 0 || b.dev.closed {
		return ErrClosed
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(b.upPtr)), b.gpu)
	copy(dst, p)
	clear(dst[len(p):])
	b.dirty = true
	return nil
}

// Read copies the latest bytes back to p.
func (b *Buffer) Read(p []byte) error {
	if b == nil || b.dev == nil {
		return ErrClosed
	}
	if len(p) > b.size {
		return ErrSize
	}
	if len(p) == 0 {
		return nil
	}
	b.dev.mu.Lock()
	defer b.dev.mu.Unlock()
	if b.closed || b.dev.closed || b.dev.eng == nil {
		return ErrClosed
	}
	if err := b.dev.eng.begin(); err != nil {
		return err
	}
	if err := b.flush(); err != nil {
		_ = b.dev.eng.closeList()
		return err
	}
	if b.state != stateCopySrc {
		b.dev.eng.transition(b.def, b.state, stateCopySrc)
		b.state = stateCopySrc
	}
	b.dev.eng.copyBuf(b.read, b.def, b.gpu)
	b.dev.eng.transition(b.def, stateCopySrc, stateUAV)
	b.state = stateUAV
	if err := b.dev.eng.submit(); err != nil {
		return err
	}
	return b.copyOut(p)
}

func (b *Buffer) flush() error {
	if !b.dirty {
		return nil
	}
	if b.state != stateCopyDst {
		b.dev.eng.transition(b.def, b.state, stateCopyDst)
		b.state = stateCopyDst
	}
	b.dev.eng.copyBuf(b.def, b.up, b.gpu)
	b.dirty = false
	return nil
}

func (b *Buffer) copyOut(p []byte) error {
	var ptr uintptr
	var pin pins
	pin.keep(&ptr)
	err := callHR(b.read, slotResMap, 0, 0, uintptr(unsafe.Pointer(&ptr)))
	pin.done()
	if err != nil || ptr == 0 {
		if err == nil {
			err = ErrUnavailable
		}
		return err
	}
	src := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(p))
	copy(p, src)
	empty := cpuRange{}
	pin = pins{}
	pin.keep(&empty)
	syscallV(b.read, slotResUnmap, 0, uintptr(unsafe.Pointer(&empty)))
	pin.done()
	return nil
}

// Close releases the buffer. The device may already be closed.
func (b *Buffer) Close() error {
	if b == nil {
		return nil
	}
	dev := b.dev
	if dev != nil {
		dev.mu.Lock()
		defer dev.mu.Unlock()
	}
	if b.closed {
		return nil
	}
	b.closed = true
	if b.up != 0 {
		syscallV(b.up, slotResUnmap, 0, 0)
	}
	release(b.up)
	release(b.def)
	release(b.read)
	b.up, b.def, b.read, b.upPtr = 0, 0, 0, 0
	return nil
}

// Compile builds a cs_5_0 pipeline. source is HLSL with one u0 and contiguous t registers.
func (d *Device) Compile(source string, threads int) (*Pipeline, error) {
	if d == nil {
		return nil, ErrClosed
	}
	srvs, err := shaderBindings(source, threads)
	if err != nil {
		return nil, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.eng == nil {
		return nil, ErrClosed
	}
	cs, err := compileHLSL(source, "ndeval", "cs_5_0")
	if err != nil {
		return nil, err
	}
	root, err := computeRoot(d.eng.dev, srvs)
	if err != nil {
		return nil, err
	}
	pso, err := d.eng.computePSO(root, cs)
	if err != nil {
		release(root)
		return nil, err
	}
	slog.Debug("d3d12 compile", "threads", threads, "srvs", srvs, "bytes", len(source))
	return &Pipeline{dev: d, pso: pso, root: root, srvs: srvs, threads: threads}, nil
}

func shaderBindings(source string, threads int) (int, error) {
	if threads < 1 || threads > 1024 || source == "" {
		return 0, ErrSize
	}
	mark := fmt.Sprintf("[numthreads(%d, 1, 1)]", threads)
	if strings.Count(source, mark) != 1 {
		return 0, fmt.Errorf("%w: numthreads", ErrUnavailable)
	}
	if strings.Count(source, "register(u0)") != 1 {
		return 0, fmt.Errorf("%w: u0", ErrUnavailable)
	}
	srv := 0
	for srv <= 28 && strings.Contains(source, fmt.Sprintf("register(t%d)", srv)) {
		srv++
	}
	if srv > 28 {
		return 0, fmt.Errorf("%w: srvs", ErrSize)
	}
	for i := srv; i < 64; i++ {
		if strings.Contains(source, fmt.Sprintf("register(t%d)", i)) {
			return 0, fmt.Errorf("%w: srv register", ErrUnavailable)
		}
	}
	return srv, nil
}

// Dispatch runs groups thread groups and waits until the GPU finishes.
// bufs[0] is the UAV. The rest are SRVs. push is the 20-byte root constant.
func (p *Pipeline) Dispatch(bufs []*Buffer, push []byte, groups int) error {
	if p == nil || p.dev == nil {
		return ErrClosed
	}
	if groups < 1 {
		return nil
	}
	if groups > maxGroup || len(push) < 20 || len(bufs) != 1+p.srvs {
		return ErrSize
	}
	p.dev.mu.Lock()
	defer p.dev.mu.Unlock()
	if p.closed || p.dev.closed || p.dev.eng == nil || p.pso == 0 {
		return ErrClosed
	}
	for _, b := range bufs {
		if b == nil || b.dev != p.dev || b.closed || b.def == 0 {
			return ErrSize
		}
	}
	if err := p.dev.eng.begin(); err != nil {
		return err
	}
	if err := p.record(bufs, push, groups); err != nil {
		_ = p.dev.eng.closeList()
		return err
	}
	return p.dev.eng.submit()
}

func (p *Pipeline) record(bufs []*Buffer, push []byte, groups int) error {
	eng := p.dev.eng
	for _, b := range bufs {
		if err := b.flush(); err != nil {
			return err
		}
	}
	out := bufs[0]
	if out.state != stateUAV {
		eng.transition(out.def, out.state, stateUAV)
		out.state = stateUAV
	}
	for _, b := range bufs[1:] {
		if b.state != stateSRV {
			eng.transition(b.def, b.state, stateSRV)
			b.state = stateSRV
		}
	}
	syscallV(eng.list, slotComputeRoot, p.root)
	syscallV(eng.list, slotSetPSO, p.pso)
	syscallV(eng.list, slotComputeUAV, 0, out.addr)
	for i, b := range bufs[1:] {
		syscallV(eng.list, slotComputeSRV, uintptr(1+i), b.addr)
	}
	var words [5]uint32
	for i := range words {
		words[i] = binary.LittleEndian.Uint32(push[i*4 : i*4+4])
	}
	eng.constants(slotComputeConst, 1+p.srvs, words[:])
	syscallV(eng.list, slotDispatch, uintptr(groups), 1, 1)
	return nil
}

// Close releases the pipeline state and its root signature.
func (p *Pipeline) Close() error {
	if p == nil {
		return nil
	}
	dev := p.dev
	if dev != nil {
		dev.mu.Lock()
		defer dev.mu.Unlock()
	}
	if p.closed {
		return nil
	}
	p.closed = true
	release(p.pso)
	release(p.root)
	p.pso, p.root = 0, 0
	return nil
}
