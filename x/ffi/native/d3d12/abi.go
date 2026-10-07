package d3d12

// ABI layouts match the MSVC x64 and ARM64 headers. BOOL and enums are int32.
// Pointers are 8 bytes. A test checks the sizes that Create* reads.

const (
	heapDefault  = 1
	heapUpload   = 2
	heapReadback = 3

	stateCommon   = 0
	stateRT       = 0x4
	stateUAV      = 0x8
	stateSRV      = 0x40
	stateCopyDst  = 0x400
	stateCopySrc  = 0x800
	stateGeneric  = 0xAC3 // GENERIC_READ
	statePresent  = 0
	barrierAllSub = 0xFFFFFFFF

	resBuffer   = 1
	layoutRow   = 1
	flagUAV     = 0x4
	fmtRGBA8    = 28
	topoStrip   = 5
	topoTri     = 3
	blendZero   = 1
	blendOne    = 2
	blendInvSrc = 6
	blendAdd    = 1
	writeAll    = 0xF
	fillSolid   = 3
	cullNone    = 1
	heapRTV     = 2
	listDirect  = 0
	listCompute = 2

	maxDim   = 16384
	maxGroup = 65535
)

type guid struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

type rootParam struct {
	typ uint32
	_   uint32
	v0  uint32
	v1  uint32
	v2  uint64
	vis uint32
	_   uint32
}

type rootSigDesc struct {
	num     uint32
	_       uint32
	params  uintptr
	nStatic uint32
	_       uint32
	statics uintptr
	flags   uint32
	_       uint32
}

type heapProps struct {
	typ     uint32
	cpu     uint32
	pool    uint32
	create  uint32
	visible uint32
}

type resourceDesc struct {
	dim         uint32
	_           uint32
	align       uint64
	width       uint64
	height      uint32
	depth       uint16
	mips        uint16
	format      uint32
	sampleCount uint32
	sampleQual  uint32
	layout      uint32
	flags       uint32
	_           uint32
}

type barrier struct {
	typ    uint32
	flags  uint32
	res    uintptr
	sub    uint32
	before uint32
	after  uint32
	_      uint32
}

type cpuRange struct {
	begin uint64
	end   uint64
}

type viewport struct {
	x, y, w, h float32
	minZ, maxZ float32
}

type scissor struct {
	left, top, right, bottom int32
}

type shaderBC struct {
	ptr uintptr
	n   uintptr
}

type blendRT struct {
	enable, logic   int32
	src, dst, op    int32
	srcA, dstA, opA int32
	logicOp         int32
	mask            uint8
	_               [3]byte
}

type blendDesc struct {
	alphaToCov  int32
	independent int32
	rt          [8]blendRT
}

type rasterDesc struct {
	fill, cull, front int32
	bias              int32
	biasClamp, slope  float32
	clip, msaa, aa    int32
	forced            uint32
	conservative      int32
}

type depthOp struct {
	fail, depthFail, pass, fn int32
}

type depthDesc struct {
	enable, write, fn, stencil int32
	readMask, writeMask        uint8
	_                          [2]byte
	front, back                depthOp
}

type streamOut struct {
	decl      uintptr
	num       uint32
	_         uint32
	strides   uintptr
	numStride uint32
	raster    uint32
}

type inputLayout struct {
	ptr uintptr
	num uint32
	_   uint32
}

type cachedPSO struct {
	ptr uintptr
	n   uintptr
}

type gfxPSO struct {
	root       uintptr
	vs, ps     shaderBC
	ds, hs, gs shaderBC
	stream     streamOut
	blend      blendDesc
	sampleMask uint32
	raster     rasterDesc
	depth      depthDesc
	_          uint32
	layout     inputLayout
	stripCut   uint32
	topo       uint32
	numRT      uint32
	rtv        [8]uint32
	dsv        uint32
	samples    uint32
	quality    uint32
	node       uint32
	_          uint32
	cached     cachedPSO
	flags      uint32
	_          uint32
}

type computePSO struct {
	root   uintptr
	cs     shaderBC
	node   uint32
	_      uint32
	cached cachedPSO
	flags  uint32
	_      uint32
}

type swapDesc struct {
	w, h        uint32
	format      int32
	stereo      int32
	sampleCount uint32
	sampleQual  uint32
	usage       uint32
	count       uint32
	scaling     int32
	effect      int32
	alpha       int32
	flags       uint32
}

type queueDesc struct {
	typ, priority, flags, node uint32
}

// adapterDesc is DXGI_ADAPTER_DESC. Description is the first 128 WCHARs.
type adapterDesc struct {
	desc                                 [128]uint16
	vendor, deviceID, subsys, rev        uint32
	dedicatedVideo, dedicatedSys, shared uintptr
	luidLow                              uint32
	luidHigh                             int32
}

type heapDesc struct {
	typ, num, flags, node uint32
}

func constParam(reg, n uint32) rootParam {
	return rootParam{typ: 1, v0: reg, v1: 0, v2: uint64(n)}
}

func srvParam(reg uint32) rootParam {
	return rootParam{typ: 3, v0: reg}
}

func uavParam(reg uint32) rootParam {
	return rootParam{typ: 4, v0: reg}
}

func heapOf(typ uint32) heapProps {
	return heapProps{typ: typ, create: 1, visible: 1}
}

func bufferDesc(size int, flags uint32) resourceDesc {
	return resourceDesc{
		dim:         resBuffer,
		width:       uint64(size),
		height:      1,
		depth:       1,
		mips:        1,
		sampleCount: 1,
		layout:      layoutRow,
		flags:       flags,
	}
}
