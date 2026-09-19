package vulkan

import (
	"context"
	"fmt"
	"unsafe"
)

// Device is a compute-capable Vulkan device with host-visible buffers.
type Device struct {
	api       api
	inst      uintptr
	phys      uintptr
	dev       uintptr
	queue     uintptr
	family    uint32
	cmdPool   uint64
	cmd       uintptr
	mem       physicalDeviceMemoryProperties
	name      string
	closed    bool
	recording bool
	pending   bool
	rec       Cmd
}

// Open loads libvulkan, creates an instance, and picks a compute queue.
func Open(ctx context.Context) (*Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	lib, err := openLib()
	if err != nil {
		return nil, err
	}
	d := &Device{}
	if err := d.api.loadLoader(lib); err != nil {
		return nil, err
	}
	appName := cstr("lewkit")
	app := applicationInfo{
		sType:            structureApplicationInfo,
		pApplicationName: appName,
		apiVersion:       apiVersion11,
	}
	var instExt []string
	var instFlags uint32
	if hasExt(d.api.instanceExts(), extPortabilityEnum) {
		instExt = []string{extPortabilityEnum}
		instFlags = instanceEnumeratePortability
	}
	extPtrs, keepExt := cStrings(instExt)
	_ = keepExt
	info := instanceCreateInfo{
		sType:                   structureInstanceCreateInfo,
		flags:                   instFlags,
		pApplicationInfo:        &app,
		enabledExtensionCount:   uint32(len(instExt)),
		ppEnabledExtensionNames: extPtrs,
	}
	if err := check(d.api.createInstance(&info, 0, &d.inst)); err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}
	if err := d.api.loadInstance(d.inst); err != nil {
		d.api.destroyInstance(d.inst, 0)
		return nil, err
	}
	if err := d.pick(); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

func (d *Device) pick() error {
	var n uint32
	if err := check(d.api.enumeratePhysical(d.inst, &n, nil)); err != nil {
		return fmt.Errorf("enumerate devices: %w", err)
	}
	if n == 0 {
		return ErrNoDevice
	}
	phys := make([]uintptr, n)
	if err := check(d.api.enumeratePhysical(d.inst, &n, &phys[0])); err != nil {
		return fmt.Errorf("enumerate devices: %w", err)
	}
	for _, p := range phys[:n] {
		if d.try(p) {
			return nil
		}
	}
	return ErrNoDevice
}

func (d *Device) try(phys uintptr) bool {
	var nq uint32
	d.api.getQueueFamilies(phys, &nq, nil)
	if nq == 0 {
		return false
	}
	fams := make([]queueFamilyProperties, nq)
	d.api.getQueueFamilies(phys, &nq, &fams[0])
	var family uint32
	found := false
	for i, f := range fams[:nq] {
		if f.queueFlags&queueComputeBit != 0 && f.queueCount > 0 {
			family = uint32(i)
			found = true
			break
		}
	}
	if !found {
		return false
	}
	prio := float32(1)
	qinfo := deviceQueueCreateInfo{
		sType:            structureDeviceQueueCreateInfo,
		queueFamilyIndex: family,
		queueCount:       1,
		pQueuePriorities: &prio,
	}
	var devExt []string
	if hasExt(d.api.deviceExts(phys), extPortabilitySubset) {
		devExt = []string{extPortabilitySubset}
	}
	devExtPtrs, keepDevExt := cStrings(devExt)
	_ = keepDevExt
	dinfo := deviceCreateInfo{
		sType:                   structureDeviceCreateInfo,
		queueCreateInfoCount:    1,
		pQueueCreateInfos:       &qinfo,
		enabledExtensionCount:   uint32(len(devExt)),
		ppEnabledExtensionNames: devExtPtrs,
	}
	var dev uintptr
	if check(d.api.createDevice(phys, &dinfo, 0, &dev)) != nil {
		return false
	}
	if err := d.api.loadDevice(dev); err != nil {
		d.api.destroyDevice(dev, 0)
		return false
	}
	var queue uintptr
	d.api.getDeviceQueue(dev, family, 0, &queue)
	poolInfo := commandPoolCreateInfo{
		sType:            structureCommandPoolCreateInfo,
		flags:            commandPoolReset,
		queueFamilyIndex: family,
	}
	var pool uint64
	if check(d.api.createCommandPool(dev, &poolInfo, 0, &pool)) != nil {
		d.api.destroyDevice(dev, 0)
		return false
	}
	alloc := commandBufferAllocateInfo{
		sType:              structureCommandBufferAllocateInfo,
		commandPool:        pool,
		commandBufferCount: 1,
	}
	var cmd uintptr
	if check(d.api.allocateCmdBuffers(dev, &alloc, &cmd)) != nil {
		d.api.destroyCommandPool(dev, pool, 0)
		d.api.destroyDevice(dev, 0)
		return false
	}
	d.phys = phys
	d.dev = dev
	d.queue = queue
	d.family = family
	d.cmdPool = pool
	d.cmd = cmd
	d.api.getMemoryProps(phys, &d.mem)
	var raw [4096]byte
	d.api.getPhysProps(phys, &raw[0])
	d.name = cstring(raw[20:276])
	return true
}

// Name is the physical device name.
func (d *Device) Name() string {
	if d == nil {
		return ""
	}
	return d.name
}

// Close destroys the device and instance.
func (d *Device) Close() error {
	if d == nil || d.closed {
		return nil
	}
	d.closed = true
	if d.dev != 0 {
		if d.cmdPool != 0 {
			d.api.destroyCommandPool(d.dev, d.cmdPool, 0)
			d.cmdPool = 0
		}
		d.api.destroyDevice(d.dev, 0)
		d.dev = 0
	}
	if d.inst != 0 {
		d.api.destroyInstance(d.inst, 0)
		d.inst = 0
	}
	return nil
}

func (d *Device) live() error {
	if d == nil || d.closed || d.dev == 0 {
		return ErrClosed
	}
	return nil
}

func (d *Device) memoryType(bits, flags uint32) (uint32, bool) {
	for i := range d.mem.memoryTypeCount {
		if bits&(1<<i) == 0 {
			continue
		}
		if d.mem.memoryTypes[i].propertyFlags&flags == flags {
			return i, true
		}
	}
	return 0, false
}

// Memory is where a buffer lives.
type Memory uint8

const (
	// Host is host-visible, coherent, and persistently mapped.
	Host Memory = iota
	// Local is device-local. Use Copy to move data.
	Local
)

// Buffer is a storage buffer.
type Buffer struct {
	d        *Device
	buf      uint64
	mem      uint64
	size     int
	kind     Memory
	ptr      unsafe.Pointer
	coherent bool
}

// Buffer allocates a host-visible storage buffer of size bytes.
func (d *Device) Buffer(size int) (*Buffer, error) {
	return d.Alloc(size, Host)
}

// Alloc creates a storage buffer of size bytes in mem.
func (d *Device) Alloc(size int, mem Memory) (*Buffer, error) {
	if err := d.live(); err != nil {
		return nil, err
	}
	if size < 1 {
		return nil, ErrSize
	}
	info := bufferCreateInfo{
		sType: structureBufferCreateInfo,
		size:  uint64(size),
		usage: bufferUsageStorage | bufferUsageTransferDst | bufferUsageTransferSrc,
	}
	var buf uint64
	if err := check(d.api.createBuffer(d.dev, &info, 0, &buf)); err != nil {
		return nil, fmt.Errorf("create buffer: %w", err)
	}
	var req memoryRequirements
	d.api.getBufferReqs(d.dev, buf, &req)
	want := uint32(memoryHostVisible | memoryHostCoherent)
	if mem == Local {
		want = memoryDeviceLocal
	}
	idx, ok := d.memoryType(req.memoryTypeBits, want)
	if !ok {
		d.api.destroyBuffer(d.dev, buf, 0)
		if mem == Local {
			return nil, fmt.Errorf("%w: no device-local memory", ErrUnavailable)
		}
		return nil, fmt.Errorf("%w: no host-visible memory", ErrUnavailable)
	}
	alloc := memoryAllocateInfo{
		sType:           structureMemoryAllocateInfo,
		allocationSize:  req.size,
		memoryTypeIndex: idx,
	}
	var block uint64
	if err := check(d.api.allocateMemory(d.dev, &alloc, 0, &block)); err != nil {
		d.api.destroyBuffer(d.dev, buf, 0)
		return nil, fmt.Errorf("allocate memory: %w", err)
	}
	if err := check(d.api.bindBufferMemory(d.dev, buf, block, 0)); err != nil {
		d.api.freeMemory(d.dev, block, 0)
		d.api.destroyBuffer(d.dev, buf, 0)
		return nil, fmt.Errorf("bind buffer: %w", err)
	}
	var ptr unsafe.Pointer
	if mem == Host {
		if err := check(d.api.mapMemory(d.dev, block, 0, req.size, 0, &ptr)); err != nil {
			d.api.freeMemory(d.dev, block, 0)
			d.api.destroyBuffer(d.dev, buf, 0)
			return nil, fmt.Errorf("map memory: %w", err)
		}
	}
	return &Buffer{d: d, buf: buf, mem: block, size: size, kind: mem, ptr: ptr, coherent: mem == Host}, nil
}

// Len is the requested size in bytes.
func (b *Buffer) Len() int {
	if b == nil {
		return 0
	}
	return b.size
}

// Memory is Host or Local.
func (b *Buffer) Memory() Memory {
	if b == nil {
		return Host
	}
	return b.kind
}

func (b *Buffer) bytes() []byte {
	return unsafe.Slice((*byte)(b.ptr), b.size)
}

// Write copies p to the start of the buffer.
func (b *Buffer) Write(p []byte) error {
	if b == nil || b.ptr == nil {
		return ErrClosed
	}
	if len(p) > b.size {
		return ErrSize
	}
	copy(b.bytes(), p)
	return nil
}

// Read copies the buffer into p.
func (b *Buffer) Read(p []byte) error {
	if b == nil || b.ptr == nil {
		return ErrClosed
	}
	if len(p) > b.size {
		return ErrSize
	}
	copy(p, b.bytes())
	return nil
}

// Close unmaps and frees the buffer.
func (b *Buffer) Close() error {
	if b == nil || b.buf == 0 {
		return nil
	}
	d := b.d
	buf, mem, ptr := b.buf, b.mem, b.ptr
	b.buf, b.mem, b.ptr = 0, 0, nil
	if d == nil || d.closed || d.dev == 0 {
		return nil
	}
	if ptr != nil {
		d.api.unmapMemory(d.dev, mem)
	}
	d.api.destroyBuffer(d.dev, buf, 0)
	d.api.freeMemory(d.dev, mem, 0)
	return nil
}

// Shader is a compute pipeline. bindings is the storage-buffer count at set 0.
type Shader struct {
	d          *Device
	module     uint64
	setLayout  uint64
	pipeLayout uint64
	pipe       uint64
	descPool   uint64
	descSet    uint64
	infos      []descriptorBufferInfo
	writes     []writeDescriptorSet
	bindings   int
	pushBytes  int
}

// ShaderConfig is SPIR-V plus optional push constants and spec constants.
type ShaderConfig struct {
	SPIRV     []byte
	Bindings  int
	PushBytes int
	Spec      []uint32
}

// Shader builds a compute pipeline from SPIR-V.
// n is the storage-buffer count at set 0.
func (d *Device) Shader(ctx context.Context, spirv []byte, bindings int) (*Shader, error) {
	return d.Compile(ctx, ShaderConfig{SPIRV: spirv, Bindings: bindings})
}

// Compile builds a compute pipeline.
func (d *Device) Compile(ctx context.Context, cfg ShaderConfig) (*Shader, error) {
	if err := d.live(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	spirv := cfg.SPIRV
	bindings := cfg.Bindings
	if bindings < 1 || len(spirv) < 20 || len(spirv)%4 != 0 {
		return nil, ErrShader
	}
	if cfg.PushBytes < 0 || cfg.PushBytes > 256 || cfg.PushBytes%4 != 0 {
		return nil, ErrPush
	}
	code := make([]uint32, len(spirv)/4)
	for i := range code {
		code[i] = uint32(spirv[i*4]) | uint32(spirv[i*4+1])<<8 | uint32(spirv[i*4+2])<<16 | uint32(spirv[i*4+3])<<24
	}
	modInfo := shaderModuleCreateInfo{
		sType:    structureShaderModuleCreateInfo,
		codeSize: uintptr(len(spirv)),
		pCode:    &code[0],
	}
	var module uint64
	if err := check(d.api.createShaderModule(d.dev, &modInfo, 0, &module)); err != nil {
		return nil, fmt.Errorf("shader module: %w", err)
	}
	binds := make([]descriptorSetLayoutBinding, bindings)
	for i := range binds {
		binds[i] = descriptorSetLayoutBinding{
			binding:         uint32(i),
			descriptorType:  descriptorStorageBuffer,
			descriptorCount: 1,
			stageFlags:      shaderStageCompute,
		}
	}
	setInfo := descriptorSetLayoutCreateInfo{
		sType:        structureDescriptorSetLayoutCreateInfo,
		bindingCount: uint32(bindings),
		pBindings:    &binds[0],
	}
	var setLayout uint64
	if err := check(d.api.createSetLayout(d.dev, &setInfo, 0, &setLayout)); err != nil {
		d.api.destroyShaderModule(d.dev, module, 0)
		return nil, fmt.Errorf("descriptor layout: %w", err)
	}
	pipeInfo := pipelineLayoutCreateInfo{
		sType:          structurePipelineLayoutCreateInfo,
		setLayoutCount: 1,
		pSetLayouts:    &setLayout,
	}
	var push pushConstantRange
	if cfg.PushBytes > 0 {
		push = pushConstantRange{
			stageFlags: shaderStageCompute,
			size:       uint32(cfg.PushBytes),
		}
		pipeInfo.pushConstantRangeCount = 1
		pipeInfo.pPushConstantRanges = &push
	}
	var pipeLayout uint64
	if err := check(d.api.createPipelineLayout(d.dev, &pipeInfo, 0, &pipeLayout)); err != nil {
		d.api.destroySetLayout(d.dev, setLayout, 0)
		d.api.destroyShaderModule(d.dev, module, 0)
		return nil, fmt.Errorf("pipeline layout: %w", err)
	}
	entry := cstr("main")
	stage := pipelineShaderStageCreateInfo{
		sType:  structurePipelineShaderStageCreateInfo,
		stage:  shaderStageCompute,
		module: module,
		pName:  entry,
	}
	var spec specializationInfo
	var specEntries []specializationMapEntry
	var specData []byte
	if n := len(cfg.Spec); n > 0 {
		specData = make([]byte, n*4)
		specEntries = make([]specializationMapEntry, n)
		for i, v := range cfg.Spec {
			specData[i*4] = byte(v)
			specData[i*4+1] = byte(v >> 8)
			specData[i*4+2] = byte(v >> 16)
			specData[i*4+3] = byte(v >> 24)
			specEntries[i] = specializationMapEntry{
				constantID: uint32(i),
				offset:     uint32(i * 4),
				size:       4,
			}
		}
		spec = specializationInfo{
			mapEntryCount: uint32(n),
			pMapEntries:   &specEntries[0],
			dataSize:      uintptr(len(specData)),
			pData:         &specData[0],
		}
		stage.pSpecializationInfo = uintptr(unsafe.Pointer(&spec))
	}
	comp := computePipelineCreateInfo{
		sType:             structureComputePipelineCreateInfo,
		stage:             stage,
		layout:            pipeLayout,
		basePipelineIndex: -1,
	}
	var pipe uint64
	if err := check(d.api.createComputePipes(d.dev, 0, 1, &comp, 0, &pipe)); err != nil {
		d.api.destroyPipelineLayout(d.dev, pipeLayout, 0)
		d.api.destroySetLayout(d.dev, setLayout, 0)
		d.api.destroyShaderModule(d.dev, module, 0)
		return nil, fmt.Errorf("compute pipeline: %w", err)
	}
	return &Shader{
		d:          d,
		module:     module,
		setLayout:  setLayout,
		pipeLayout: pipeLayout,
		pipe:       pipe,
		bindings:   bindings,
		pushBytes:  cfg.PushBytes,
	}, nil
}

// Close destroys the pipeline.
func (s *Shader) Close() error {
	if s == nil || s.pipe == 0 {
		return nil
	}
	d := s.d
	pipe, layout, set, mod := s.pipe, s.pipeLayout, s.setLayout, s.module
	pool := s.descPool
	s.pipe, s.pipeLayout, s.setLayout, s.module = 0, 0, 0, 0
	s.descPool, s.descSet = 0, 0
	if d == nil || d.closed || d.dev == 0 {
		return nil
	}
	if pool != 0 {
		d.api.destroyDescriptorPool(d.dev, pool, 0)
	}
	d.api.destroyPipeline(d.dev, pipe, 0)
	d.api.destroyPipelineLayout(d.dev, layout, 0)
	d.api.destroySetLayout(d.dev, set, 0)
	d.api.destroyShaderModule(d.dev, mod, 0)
	return nil
}

func (d *Device) flush(b *Buffer) error {
	if b == nil || b.ptr == nil || b.coherent {
		return nil
	}
	rng := mappedMemoryRange{
		sType:  structureMappedMemoryRange,
		memory: b.mem,
		size:   wholeSize,
	}
	if err := check(d.api.flushMapped(d.dev, 1, &rng)); err != nil {
		return fmt.Errorf("flush: %w", err)
	}
	return nil
}

func (d *Device) invalidate(b *Buffer) error {
	if b == nil || b.ptr == nil || b.coherent {
		return nil
	}
	rng := mappedMemoryRange{
		sType:  structureMappedMemoryRange,
		memory: b.mem,
		size:   wholeSize,
	}
	if err := check(d.api.invalidateMapped(d.dev, 1, &rng)); err != nil {
		return fmt.Errorf("invalidate: %w", err)
	}
	return nil
}
