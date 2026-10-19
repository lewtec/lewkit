//go:build windows && (amd64 || arm64)

package d3d12

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// Device runs compute kernels. It does not attach a window and it does not
// share a screen's queue.
type Device struct {
	mu       sync.Mutex
	eng      *engine
	slot     *onceEngine
	closed   bool
	children native.Children
	cleanup  runtime.Cleanup
}

// onceEngine releases the COM device once. It does not point at *Device.
type onceEngine struct {
	mu  sync.Mutex
	eng *engine
}

func (o *onceEngine) wait() {
	if o == nil {
		return
	}
	o.mu.Lock()
	eng := o.eng
	o.mu.Unlock()
	if eng != nil {
		_ = eng.wait()
	}
}

func (o *onceEngine) release() {
	if o == nil {
		return
	}
	o.mu.Lock()
	eng := o.eng
	o.eng = nil
	o.mu.Unlock()
	if eng == nil {
		return
	}
	_ = eng.wait()
	eng.release()
}

// d3dBuf is one buffer's COM pointers. It does not point at *Buffer.
type d3dBuf struct {
	mu            sync.Mutex
	done          bool
	up, def, read uintptr
}

func (b *d3dBuf) release() {
	if b == nil {
		return
	}
	b.mu.Lock()
	if b.done {
		b.mu.Unlock()
		return
	}
	b.done = true
	up, def, read := b.up, b.def, b.read
	b.mu.Unlock()
	if up != 0 {
		syscallV(up, slotResUnmap, 0, 0)
	}
	release(up)
	release(def)
	release(read)
}

// d3dPipe is one pipeline's COM pointers. It does not point at *Pipeline.
type d3dPipe struct {
	mu        sync.Mutex
	done      bool
	pso, root uintptr
}

func (p *d3dPipe) release() {
	if p == nil {
		return
	}
	p.mu.Lock()
	if p.done {
		p.mu.Unlock()
		return
	}
	p.done = true
	pso, root := p.pso, p.root
	p.mu.Unlock()
	release(pso)
	release(root)
}

// Pipeline is one compiled compute kernel.
type Pipeline struct {
	dev     *Device
	pso     uintptr
	root    uintptr
	srvs    int
	threads int
	closed  bool
	node    *d3dPipe
	cleanup runtime.Cleanup
}

// Buffer is a storage buffer. Store writes the upload heap. Dispatch copies
// it onto the default heap before the kernel reads it.
type Buffer struct {
	dev     *Device
	size    int
	gpu     int
	def     uintptr
	up      uintptr
	read    uintptr
	upPtr   uintptr
	addr    uintptr
	state   uint32
	dirty   bool
	closed  bool
	node    *d3dBuf
	cleanup runtime.Cleanup
}

// OpenDevice opens the default adapter for compute.
func OpenDevice() (*Device, error) {
	eng, err := newEngine(listCompute, false)
	if err != nil {
		return nil, err
	}
	slog.Debug("d3d12 device", "adapter", eng.name, "level", eng.level)
	d := &Device{eng: eng, slot: &onceEngine{eng: eng}}
	d.cleanup = native.Arm(d, (*onceEngine).release, d.slot)
	return d, nil
}

// Name is the adapter description.
func (d *Device) Name() string {
	if d == nil || d.eng == nil {
		return ""
	}
	return d.eng.name
}

// Close waits for the queue, releases outstanding buffers and pipelines, then the device.
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
	native.Disarm(d, d.cleanup)
	if d.slot != nil {
		d.slot.wait()
	} else if d.eng != nil {
		_ = d.eng.wait()
	}
	d.children.Drop()
	if d.slot != nil {
		d.slot.release()
	} else if d.eng != nil {
		d.eng.release()
	}
	d.eng = nil
	runtime.KeepAlive(d)
	return nil
}

// Buffer allocates a default resource plus upload and readback heaps.
func (d *Device) Buffer(size int) (*Buffer, error) {
	if d == nil {
		return nil, ErrClosed
	}
	if size < 1 {
		return nil, fmt.Errorf("%w: buffer", ErrSize)
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
	out := &Buffer{
		dev:   d,
		size:  size,
		gpu:   n,
		def:   def,
		up:    up,
		read:  back,
		upPtr: ptr,
		addr:  syscallV(def, slotResVA),
		state: stateCopyDst,
		node:  &d3dBuf{up: up, def: def, read: back},
	}
	d.children.Add(out)
	out.cleanup = native.Arm(out, (*d3dBuf).release, out.node)
	return out, nil
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
		return fmt.Errorf("%w: store", ErrSize)
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
		return fmt.Errorf("%w: read", ErrSize)
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
	if b.dev != nil {
		b.dev.mu.Lock()
		defer b.dev.mu.Unlock()
	}
	b.Drop()
	return nil
}

// Drop releases the buffer. The device lock is already held when the device is set.
func (b *Buffer) Drop() {
	if b == nil {
		return
	}
	native.Disarm(b, b.cleanup)
	if b.dev != nil {
		b.dev.children.Forget(b)
	}
	if b.closed {
		return
	}
	b.closed = true
	if b.node != nil {
		b.up, b.def, b.read, b.upPtr = 0, 0, 0, 0
		b.node.release()
		return
	}
	if b.up != 0 {
		syscallV(b.up, slotResUnmap, 0, 0)
	}
	release(b.up)
	release(b.def)
	release(b.read)
	b.up, b.def, b.read, b.upPtr = 0, 0, 0, 0
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
	out := &Pipeline{dev: d, pso: pso, root: root, srvs: srvs, threads: threads, node: &d3dPipe{pso: pso, root: root}}
	d.children.Add(out)
	out.cleanup = native.Arm(out, (*d3dPipe).release, out.node)
	return out, nil
}

func shaderBindings(source string, threads int) (int, error) {
	if threads < 1 || threads > 1024 || source == "" {
		return 0, fmt.Errorf("%w: threads", ErrSize)
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
// Groups above one dimension stack on Y. The sixth root constant is that span.
func (p *Pipeline) Dispatch(bufs []*Buffer, push []byte, groups int) error {
	if p == nil || p.dev == nil {
		return ErrClosed
	}
	if groups < 1 {
		return nil
	}
	gx, gy, err := DispatchGrid(groups)
	if err != nil {
		return err
	}
	if len(push) < 20 {
		return fmt.Errorf("%w: push", ErrSize)
	}
	if len(bufs) != 1+p.srvs {
		return fmt.Errorf("%w: bindings", ErrSize)
	}
	p.dev.mu.Lock()
	defer p.dev.mu.Unlock()
	if p.closed || p.dev.closed || p.dev.eng == nil || p.pso == 0 {
		return ErrClosed
	}
	for _, b := range bufs {
		if b == nil || b.dev != p.dev || b.closed || b.def == 0 {
			return fmt.Errorf("%w: binding", ErrSize)
		}
	}
	if err := p.dev.eng.begin(); err != nil {
		return err
	}
	if err := p.record(bufs, push, gx, gy); err != nil {
		_ = p.dev.eng.closeList()
		return err
	}
	return p.dev.eng.submit()
}

func (p *Pipeline) record(bufs []*Buffer, push []byte, gx, gy uint32) error {
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
	var words [pushWords]uint32
	for i := 0; i < pushWords-1; i++ {
		words[i] = binary.LittleEndian.Uint32(push[i*4 : i*4+4])
	}
	words[pushWords-1] = gx * uint32(p.threads)
	eng.constants(slotComputeConst, 1+p.srvs, words[:])
	syscallV(eng.list, slotDispatch, uintptr(gx), uintptr(gy), 1)
	return nil
}

// Close releases the pipeline state and its root signature.
func (p *Pipeline) Close() error {
	if p == nil {
		return nil
	}
	if p.dev != nil {
		p.dev.mu.Lock()
		defer p.dev.mu.Unlock()
	}
	p.Drop()
	return nil
}

// Drop releases the pipeline state and its root signature.
// The device lock is already held when the device is set.
func (p *Pipeline) Drop() {
	if p == nil {
		return
	}
	native.Disarm(p, p.cleanup)
	if p.dev != nil {
		p.dev.children.Forget(p)
	}
	if p.closed {
		return
	}
	p.closed = true
	if p.node != nil {
		p.pso, p.root = 0, 0
		p.node.release()
		return
	}
	release(p.pso)
	release(p.root)
	p.pso, p.root = 0, 0
}
