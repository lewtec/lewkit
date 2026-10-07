//go:build windows && (amd64 || arm64)

package d3d12

import (
	"fmt"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// COM slots follow the Windows C++ vtable. Void methods are not HRESULTs.
// A CPU descriptor handle is an 8-byte struct written through a pointer
// argument; RAX is that pointer, not the handle. syscall.Errno is
// GetLastError, which these methods do not set.

const (
	slotRelease = 2

	slotDevQueue      = 8
	slotDevAlloc      = 9
	slotDevGfxPSO     = 10
	slotDevComputePSO = 11
	slotDevList       = 12
	slotDevHeap       = 14
	slotDevInc        = 15
	slotDevRoot       = 16
	slotDevRTV        = 20
	slotDevResource   = 27
	slotDevFence      = 36

	slotQueueExec    = 10
	slotQueueSignal  = 14
	slotAllocReset   = 8
	slotListClose    = 9
	slotListReset    = 10
	slotDraw         = 12
	slotDispatch     = 14
	slotCopy         = 15
	slotTopo         = 20
	slotViewport     = 21
	slotScissor      = 22
	slotSetPSO       = 25
	slotBarrier      = 26
	slotComputeRoot  = 29
	slotGfxRoot      = 30
	slotComputeConst = 35
	slotGfxConst     = 36
	slotComputeSRV   = 39
	slotGfxSRV       = 40
	slotComputeUAV   = 41
	slotOM           = 46

	slotFenceValue = 8
	slotFenceEvent = 9
	slotHeapCPU    = 9
	slotResMap     = 8
	slotResUnmap   = 9
	slotResVA      = 11

	slotFactoryEnum   = 7
	slotFactoryAssoc  = 8
	slotAdapterDesc   = 8
	slotSwapPresent   = 8
	slotSwapBuffer    = 9
	slotSwapResize    = 13
	slotSwapChainHwnd = 15
	slotSwapIndex     = 36
	slotBlobPtr       = 3
	slotBlobSize      = 4

	rootDenyTess     = 0x1c
	rootDenyGraphics = 0x3e
	swapUsageRT      = 0x20
	swapEffectFlip   = 4
	swapAlphaIgnore  = 3
	noAltEnter       = 2
	waitInfinite     = 0xFFFFFFFF
	blobLimit        = 16 << 20

	hrRemoved = 0x887A0005
	hrHung    = 0x887A0006
	hrReset   = 0x887A0007
)

var (
	iidDevice   = guid{0x189819f1, 0x1db6, 0x4b57, [8]byte{0xbe, 0x54, 0x18, 0x21, 0x33, 0x9b, 0x85, 0xf7}}
	iidQueue    = guid{0x0ec870a6, 0x5d7e, 0x4c22, [8]byte{0x8c, 0xfc, 0x5b, 0xaa, 0xe0, 0x76, 0x16, 0xed}}
	iidAlloc    = guid{0x6102dee4, 0xaf59, 0x4b09, [8]byte{0xb9, 0x99, 0xb4, 0x4d, 0x73, 0xf0, 0x9b, 0x24}}
	iidList     = guid{0x5b160d0f, 0xac1b, 0x4185, [8]byte{0x8b, 0xa8, 0xb3, 0xae, 0x42, 0xa5, 0xa4, 0x55}}
	iidFence    = guid{0x0a753dcf, 0xc4d8, 0x4b91, [8]byte{0xad, 0xf6, 0xbe, 0x5a, 0x60, 0xd9, 0x5a, 0x76}}
	iidHeap     = guid{0x8efb471d, 0x616c, 0x4f49, [8]byte{0x90, 0xf7, 0x12, 0x7b, 0xb7, 0x63, 0xfa, 0x51}}
	iidRoot     = guid{0xc54a6b66, 0x72df, 0x4ee8, [8]byte{0x8b, 0xe5, 0xa9, 0x46, 0xa1, 0x42, 0x92, 0x14}}
	iidPSO      = guid{0x765a30f3, 0xf624, 0x4c6f, [8]byte{0xa8, 0x28, 0xac, 0xe9, 0x48, 0x62, 0x24, 0x45}}
	iidRes      = guid{0x696442be, 0xa72e, 0x4059, [8]byte{0xbc, 0x79, 0x5b, 0x5c, 0x98, 0x04, 0x0f, 0xad}}
	iidFactory2 = guid{0x50c83a1c, 0xe072, 0x4c48, [8]byte{0x87, 0xb0, 0x36, 0x30, 0xfa, 0x36, 0xa6, 0xd0}}
	iidSwap3    = guid{0x94d99bdb, 0xf1f8, 0x4ab0, [8]byte{0xb2, 0x36, 0x7d, 0xa0, 0x17, 0x0e, 0xda, 0xb1}}

	d3d12CreateDevice func(adapter uintptr, level uint32, riid uintptr, dev *uintptr) uint32
	serializeRootSig  func(desc uintptr, version uint32, blob *uintptr, errBlob *uintptr) uint32
	createFactory2    func(flags uint32, riid uintptr, factory *uintptr) uint32
	d3dCompile        func(src uintptr, n uintptr, name uintptr, defines uintptr, include uintptr, entry uintptr, target uintptr, flags1 uint32, flags2 uint32, code *uintptr, errs *uintptr) uint32

	procCreateEvent = native.ProcOf("kernel32.dll", "CreateEventW")
	procWait        = native.ProcOf("kernel32.dll", "WaitForSingleObject")
	procCloseHandle = native.ProcOf("kernel32.dll", "CloseHandle")
)

var available = native.Once(func() error {
	if err := bindDLLs(); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	dev, _, err := createDevice()
	if err != nil {
		return err
	}
	release(dev)
	return nil
})

// Available reports whether Direct3D 12 can open a device.
func Available() error { return available() }

func bindDLLs() error {
	d3d, err := native.OpenChain(native.Lazy, "d3d12.dll")
	if err != nil {
		return err
	}
	dxgi, err := native.OpenChain(native.Lazy, "dxgi.dll")
	if err != nil {
		return err
	}
	compiler, err := native.OpenChain(native.Lazy, "d3dcompiler_47.dll")
	if err != nil {
		return err
	}
	if err := native.Bind(d3d, "D3D12CreateDevice", &d3d12CreateDevice); err != nil {
		return err
	}
	if err := native.Bind(d3d, "D3D12SerializeRootSignature", &serializeRootSig); err != nil {
		return err
	}
	if err := native.Bind(dxgi, "CreateDXGIFactory2", &createFactory2); err != nil {
		return err
	}
	return native.Bind(compiler, "D3DCompile", &d3dCompile)
}

type pins struct {
	runtime.Pinner
	on bool
}

func (p *pins) keep(obj any) {
	p.Pin(obj)
	p.on = true
}

func (p *pins) done() {
	if !p.on {
		return
	}
	p.on = false
	p.Unpin()
}

func vslot(obj uintptr, slot int) uintptr {
	table := *(*uintptr)(unsafe.Pointer(obj))
	return *(*uintptr)(unsafe.Pointer(table + uintptr(slot)*unsafe.Sizeof(uintptr(0))))
}

func syscallV(obj uintptr, slot int, args ...uintptr) uintptr {
	all := make([]uintptr, 0, len(args)+1)
	all = append(all, obj)
	all = append(all, args...)
	r, _, _ := syscall.SyscallN(vslot(obj, slot), all...)
	return r
}

// cpuHandle reads ID3D12DescriptorHeap::GetCPUDescriptorHandleForHeapStart.
// The method writes the handle through its second argument.
func cpuHandle(heap uintptr) uintptr {
	var handle uintptr
	var p pins
	p.keep(&handle)
	syscallV(heap, slotHeapCPU, uintptr(unsafe.Pointer(&handle)))
	p.done()
	return handle
}

func callHR(obj uintptr, slot int, args ...uintptr) error {
	if obj == 0 {
		return ErrClosed
	}
	return hrErr(uint32(syscallV(obj, slot, args...)))
}

func hrErr(code uint32) error {
	if int32(code) >= 0 {
		return nil
	}
	switch code {
	case hrRemoved, hrHung, hrReset:
		return fmt.Errorf("%w: 0x%08X", ErrLost, code)
	default:
		return fmt.Errorf("%w: 0x%08X", ErrUnavailable, code)
	}
}

func release(obj uintptr) {
	if obj == 0 {
		return
	}
	syscallV(obj, slotRelease)
}

func query(obj uintptr, id guid) (uintptr, error) {
	if obj == 0 {
		return 0, ErrClosed
	}
	var out uintptr
	var p pins
	p.keep(&id)
	p.keep(&out)
	err := callHR(obj, 0, uintptr(unsafe.Pointer(&id)), uintptr(unsafe.Pointer(&out)))
	p.done()
	if err != nil || out == 0 {
		release(out)
		if err == nil {
			err = ErrUnavailable
		}
		return 0, err
	}
	return out, nil
}

func createDevice() (uintptr, uint32, error) {
	if d3d12CreateDevice == nil {
		return 0, 0, ErrUnavailable
	}
	levels := []uint32{0xc100, 0xc000, 0xb100, 0xb000}
	for _, level := range levels {
		var dev uintptr
		var p pins
		p.keep(&dev)
		hr := d3d12CreateDevice(0, level, uintptr(unsafe.Pointer(&iidDevice)), &dev)
		p.done()
		if int32(hr) >= 0 && dev != 0 {
			return dev, level, nil
		}
		release(dev)
	}
	return 0, 0, ErrUnavailable
}

func createFactory() (uintptr, error) {
	if createFactory2 == nil {
		return 0, ErrUnavailable
	}
	var factory uintptr
	var p pins
	p.keep(&factory)
	hr := createFactory2(0, uintptr(unsafe.Pointer(&iidFactory2)), &factory)
	p.done()
	if err := hrErr(hr); err != nil || factory == 0 {
		release(factory)
		if err == nil {
			err = ErrUnavailable
		}
		return 0, err
	}
	return factory, nil
}

// newCOM calls a Create* method whose last two arguments are riid and **out.
// keep lists Go objects that args point at, so they stay put for the call.
func newCOM(obj uintptr, slot int, iid *guid, keep []any, args ...uintptr) (uintptr, error) {
	var out uintptr
	var p pins
	for _, item := range keep {
		p.keep(item)
	}
	p.keep(&out)
	all := make([]uintptr, 0, len(args)+2)
	all = append(all, args...)
	all = append(all, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&out)))
	err := callHR(obj, slot, all...)
	p.done()
	if err != nil || out == 0 {
		release(out)
		if err == nil {
			err = ErrUnavailable
		}
		return 0, err
	}
	return out, nil
}

type engine struct {
	dev, queue, alloc, list, fence, factory uintptr
	event                                   uintptr
	fenceVal                                uintptr
	name                                    string
	level                                   uint32
	kind                                    uint32
}

func newEngine(kind uint32, needFactory bool) (eng *engine, err error) {
	if err = available(); err != nil {
		return nil, err
	}
	eng = &engine{kind: kind, name: "Direct3D 12"}
	defer func() {
		if err != nil && eng != nil {
			eng.release()
			eng = nil
		}
	}()
	eng.dev, eng.level, err = createDevice()
	if err != nil {
		return eng, err
	}
	eng.factory, err = createFactory()
	if err != nil && needFactory {
		return eng, err
	}
	if eng.factory != 0 {
		if name := adapterName(eng.factory); name != "" {
			eng.name = name
		}
	}
	err = nil
	if !needFactory {
		release(eng.factory)
		eng.factory = 0
	}
	qd := queueDesc{typ: kind}
	eng.queue, err = newCOM(eng.dev, slotDevQueue, &iidQueue, []any{&qd}, uintptr(unsafe.Pointer(&qd)))
	if err != nil {
		return eng, err
	}
	eng.alloc, err = newCOM(eng.dev, slotDevAlloc, &iidAlloc, nil, uintptr(kind))
	if err != nil {
		return eng, err
	}
	eng.list, err = newCOM(eng.dev, slotDevList, &iidList, nil, 0, uintptr(kind), eng.alloc, 0)
	if err != nil {
		return eng, err
	}
	if err = callHR(eng.list, slotListClose); err != nil {
		return eng, err
	}
	eng.fence, err = newCOM(eng.dev, slotDevFence, &iidFence, nil, 0, 0)
	if err != nil {
		return eng, err
	}
	h, _, _ := procCreateEvent.Call(0, 0, 0, 0)
	if h == 0 {
		err = ErrUnavailable
		return eng, err
	}
	eng.event = h
	return eng, nil
}

func (e *engine) release() {
	if e == nil {
		return
	}
	release(e.list)
	release(e.alloc)
	release(e.queue)
	release(e.fence)
	release(e.factory)
	release(e.dev)
	if e.event != 0 {
		_, _, _ = procCloseHandle.Call(e.event)
		e.event = 0
	}
	e.list, e.alloc, e.queue, e.fence, e.factory, e.dev = 0, 0, 0, 0, 0, 0
}

func (e *engine) wait() error {
	if e == nil || e.fence == 0 {
		return ErrClosed
	}
	if e.fenceVal == 0 {
		return nil
	}
	done := syscallV(e.fence, slotFenceValue)
	if done == ^uintptr(0) {
		return ErrLost
	}
	if done >= e.fenceVal {
		return nil
	}
	if err := callHR(e.fence, slotFenceEvent, e.fenceVal, e.event); err != nil {
		return err
	}
	r, _, _ := procWait.Call(e.event, waitInfinite)
	if r != 0 {
		return ErrLost
	}
	return nil
}

func (e *engine) begin() error {
	if err := e.wait(); err != nil {
		return err
	}
	if err := callHR(e.alloc, slotAllocReset); err != nil {
		return err
	}
	return callHR(e.list, slotListReset, e.alloc, 0)
}

func (e *engine) closeList() error { return callHR(e.list, slotListClose) }

func (e *engine) submit() error {
	if err := e.closeList(); err != nil {
		return err
	}
	list := e.list
	var p pins
	p.keep(&list)
	syscallV(e.queue, slotQueueExec, 1, uintptr(unsafe.Pointer(&list)))
	p.done()
	e.fenceVal++
	if err := callHR(e.queue, slotQueueSignal, e.fence, e.fenceVal); err != nil {
		return err
	}
	return e.wait()
}

func (e *engine) committed(heap, state uint32, desc resourceDesc) (uintptr, error) {
	props := heapOf(heap)
	var out uintptr
	var p pins
	p.keep(&props)
	p.keep(&desc)
	p.keep(&out)
	err := callHR(e.dev, slotDevResource,
		uintptr(unsafe.Pointer(&props)),
		0,
		uintptr(unsafe.Pointer(&desc)),
		uintptr(state),
		0,
		uintptr(unsafe.Pointer(&iidRes)),
		uintptr(unsafe.Pointer(&out)),
	)
	p.done()
	if err != nil || out == 0 {
		release(out)
		if err == nil {
			err = ErrUnavailable
		}
		return 0, err
	}
	return out, nil
}

func (e *engine) transition(res uintptr, before, after uint32) {
	if res == 0 || before == after {
		return
	}
	b := barrier{res: res, sub: barrierAllSub, before: before, after: after}
	var p pins
	p.keep(&b)
	defer p.done()
	syscallV(e.list, slotBarrier, 1, uintptr(unsafe.Pointer(&b)))
}

func (e *engine) copyBuf(dst, src uintptr, n int) {
	if n < 1 || dst == 0 || src == 0 {
		return
	}
	syscallV(e.list, slotCopy, dst, 0, src, 0, uintptr(n))
}

func (e *engine) viewport(w, h int) {
	vp := viewport{w: float32(w), h: float32(h), maxZ: 1}
	sc := scissor{right: int32(w), bottom: int32(h)}
	var p pins
	p.keep(&vp)
	p.keep(&sc)
	defer p.done()
	syscallV(e.list, slotViewport, 1, uintptr(unsafe.Pointer(&vp)))
	syscallV(e.list, slotScissor, 1, uintptr(unsafe.Pointer(&sc)))
}

func (e *engine) bindRT(handle uintptr) {
	h := handle
	var p pins
	p.keep(&h)
	defer p.done()
	syscallV(e.list, slotOM, 1, uintptr(unsafe.Pointer(&h)), 0, 0)
}

func (e *engine) constants(slot, index int, vals []uint32) {
	if len(vals) == 0 {
		return
	}
	var p pins
	p.keep(&vals[0])
	defer p.done()
	syscallV(e.list, slot, uintptr(index), uintptr(len(vals)), uintptr(unsafe.Pointer(&vals[0])), 0)
}

func adapterName(factory uintptr) string {
	if factory == 0 {
		return ""
	}
	var adapter uintptr
	var p pins
	p.keep(&adapter)
	err := callHR(factory, slotFactoryEnum, 0, uintptr(unsafe.Pointer(&adapter)))
	p.done()
	if err != nil || adapter == 0 {
		release(adapter)
		return ""
	}
	defer release(adapter)
	var desc adapterDesc
	p = pins{}
	p.keep(&desc)
	err = callHR(adapter, slotAdapterDesc, uintptr(unsafe.Pointer(&desc)))
	p.done()
	if err != nil {
		return ""
	}
	n := 0
	for n < len(desc.desc) && desc.desc[n] != 0 {
		n++
	}
	return string(utf16.Decode(desc.desc[:n]))
}

func compileHLSL(source, entry, target string) ([]byte, error) {
	if d3dCompile == nil || source == "" || entry == "" || target == "" {
		return nil, ErrUnavailable
	}
	src := []byte(source)
	entryB := native.CString(entry)
	targetB := native.CString(target)
	var code, errs uintptr
	var p pins
	p.keep(&src[0])
	p.keep(&entryB[0])
	p.keep(&targetB[0])
	p.keep(&code)
	p.keep(&errs)
	hr := d3dCompile(
		uintptr(unsafe.Pointer(&src[0])),
		uintptr(len(src)),
		0, 0, 0,
		uintptr(unsafe.Pointer(&entryB[0])),
		uintptr(unsafe.Pointer(&targetB[0])),
		0, 0,
		&code, &errs,
	)
	p.done()
	defer release(code)
	defer release(errs)
	if err := hrErr(hr); err != nil {
		msg := blobString(errs)
		if msg == "" {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", err, msg)
	}
	return blobCopy(code)
}

func blobCopy(blob uintptr) ([]byte, error) {
	if blob == 0 {
		return nil, ErrUnavailable
	}
	ptr := syscallV(blob, slotBlobPtr)
	n := syscallV(blob, slotBlobSize)
	if ptr == 0 || n == 0 || n > blobLimit {
		return nil, fmt.Errorf("%w: blob %d", ErrUnavailable, n)
	}
	out := make([]byte, n)
	copy(out, unsafe.Slice((*byte)(unsafe.Pointer(ptr)), n))
	return out, nil
}

func blobString(blob uintptr) string {
	if blob == 0 {
		return ""
	}
	ptr := syscallV(blob, slotBlobPtr)
	if ptr == 0 {
		return ""
	}
	return native.GoString(ptr)
}

func makeRoot(dev uintptr, params []rootParam, flags uint32) (uintptr, error) {
	if dev == 0 || len(params) == 0 || serializeRootSig == nil {
		return 0, ErrUnavailable
	}
	desc := rootSigDesc{
		num:    uint32(len(params)),
		params: uintptr(unsafe.Pointer(&params[0])),
		flags:  flags,
	}
	var blob, errBlob uintptr
	var p pins
	p.keep(&params[0])
	p.keep(&desc)
	p.keep(&blob)
	p.keep(&errBlob)
	hr := serializeRootSig(uintptr(unsafe.Pointer(&desc)), 1, &blob, &errBlob)
	p.done()
	defer release(blob)
	defer release(errBlob)
	if err := hrErr(hr); err != nil {
		msg := blobString(errBlob)
		if msg == "" {
			return 0, err
		}
		return 0, fmt.Errorf("%w: %s", err, msg)
	}
	code, err := blobCopy(blob)
	if err != nil {
		return 0, err
	}
	return newCOM(dev, slotDevRoot, &iidRoot, []any{&code[0]}, 0, uintptr(unsafe.Pointer(&code[0])), uintptr(len(code)))
}

func graphicsRoot(dev uintptr) (uintptr, error) {
	params := []rootParam{srvParam(0), constParam(0, 4)}
	return makeRoot(dev, params, rootDenyTess)
}

func computeRoot(dev uintptr, srvs int) (uintptr, error) {
	params := make([]rootParam, 0, srvs+2)
	params = append(params, uavParam(0))
	for i := 0; i < srvs; i++ {
		params = append(params, srvParam(uint32(i)))
	}
	params = append(params, constParam(0, 5))
	return makeRoot(dev, params, rootDenyGraphics)
}

func blendState(on bool) blendDesc {
	var rt blendRT
	rt.mask = writeAll
	if on {
		rt.enable = 1
		rt.src = blendOne
		rt.dst = blendInvSrc
		rt.op = blendAdd
		rt.srcA = blendOne
		rt.dstA = blendInvSrc
		rt.opA = blendAdd
	}
	var desc blendDesc
	for i := range desc.rt {
		desc.rt[i] = rt
	}
	return desc
}

func (e *engine) graphicsPSO(root uintptr, vs, ps []byte, blendOn bool) (uintptr, error) {
	if len(vs) == 0 || len(ps) == 0 || root == 0 {
		return 0, ErrUnavailable
	}
	desc := gfxPSO{
		root:       root,
		vs:         shaderBC{ptr: uintptr(unsafe.Pointer(&vs[0])), n: uintptr(len(vs))},
		ps:         shaderBC{ptr: uintptr(unsafe.Pointer(&ps[0])), n: uintptr(len(ps))},
		blend:      blendState(blendOn),
		sampleMask: 0xFFFFFFFF,
		raster:     rasterDesc{fill: fillSolid, cull: cullNone, clip: 1},
		topo:       uint32(topoTri),
		numRT:      1,
		samples:    1,
	}
	desc.rtv[0] = fmtRGBA8
	return newCOM(e.dev, slotDevGfxPSO, &iidPSO, []any{&desc, &vs[0], &ps[0]}, uintptr(unsafe.Pointer(&desc)))
}

func (e *engine) computePSO(root uintptr, cs []byte) (uintptr, error) {
	if len(cs) == 0 || root == 0 {
		return 0, ErrUnavailable
	}
	desc := computePSO{
		root: root,
		cs:   shaderBC{ptr: uintptr(unsafe.Pointer(&cs[0])), n: uintptr(len(cs))},
	}
	return newCOM(e.dev, slotDevComputePSO, &iidPSO, []any{&desc, &cs[0]}, uintptr(unsafe.Pointer(&desc)))
}
