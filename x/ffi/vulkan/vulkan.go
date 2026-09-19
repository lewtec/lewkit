package vulkan

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"unsafe"
)

// Device is a compute-capable Vulkan device with host-visible buffers.
type Device struct {
	api         api
	inst        uintptr
	phys        uintptr
	dev         uintptr
	queue       uintptr
	family      uint32
	commandPool uint64
	cmd         uintptr
	mem         physicalDeviceMemoryProperties
	name        string
	vendor      string
	closed      bool
	recording   bool
	pending     bool
	recorded    Cmd
}

// Info is a compute-capable physical device. Index is 0-based among
// devices that advertise a compute queue. Vendor is a slug (amd, nvidia,
// llvmpipe, …) for driver weights.
type Info struct {
	Index  int
	Name   string
	Vendor string
}

// Open loads libvulkan, creates an instance, and opens the first
// compute-capable physical device. Use List and OpenIndex for the rest.
func Open(ctx context.Context) (*Device, error) {
	return OpenIndex(ctx, 0)
}

// List returns every compute-capable physical device. It does not create
// a logical device.
func List(ctx context.Context) ([]Info, error) {
	d, err := openInstance(ctx)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	return d.computeDevices()
}

// OpenIndex opens the index-th compute-capable physical device from List.
func OpenIndex(ctx context.Context, index int) (*Device, error) {
	d, err := openInstance(ctx)
	if err != nil {
		return nil, err
	}
	if err := d.pickIndex(index); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

func openInstance(ctx context.Context) (*Device, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ensurePlatform()
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
	return d, nil
}

func (d *Device) physicalDevices() ([]uintptr, error) {
	var n uint32
	if err := check(d.api.enumeratePhysical(d.inst, &n, nil)); err != nil {
		return nil, fmt.Errorf("enumerate devices: %w", err)
	}
	if n == 0 {
		slog.Debug("vulkan enumerate physical", "count", 0)
		return nil, ErrNoDevice
	}
	phys := make([]uintptr, n)
	if err := check(d.api.enumeratePhysical(d.inst, &n, &phys[0])); err != nil {
		return nil, fmt.Errorf("enumerate devices: %w", err)
	}
	slog.Debug("vulkan enumerate physical", "count", n)
	return phys[:n], nil
}

func (d *Device) computeFamily(phys uintptr) (uint32, bool) {
	var nq uint32
	d.api.getQueueFamilies(phys, &nq, nil)
	if nq == 0 {
		return 0, false
	}
	fams := make([]queueFamilyProperties, nq)
	d.api.getQueueFamilies(phys, &nq, &fams[0])
	for i, f := range fams[:nq] {
		if f.queueFlags&queueComputeBit != 0 && f.queueCount > 0 {
			return uint32(i), true
		}
	}
	return 0, false
}

type physicalProperties struct {
	name       string
	vendorID   uint32
	deviceType uint32
}

func (d *Device) physicalProperties(phys uintptr) physicalProperties {
	var raw [4096]byte
	d.api.getPhysProps(phys, &raw[0])
	return physicalProperties{
		name:       cstring(raw[20:276]),
		vendorID:   binary.LittleEndian.Uint32(raw[8:12]),
		deviceType: binary.LittleEndian.Uint32(raw[16:20]),
	}
}

func (d *Device) computeDevices() ([]Info, error) {
	phys, err := d.physicalDevices()
	if err != nil {
		return nil, err
	}
	var out []Info
	for _, p := range phys {
		properties := d.physicalProperties(p)
		if _, ok := d.computeFamily(p); !ok {
			slog.Debug("vulkan skip physical", "name", properties.name, "reason", "no compute")
			continue
		}
		info := Info{
			Index:  len(out),
			Name:   properties.name,
			Vendor: vendorSlug(properties.vendorID, properties.deviceType, properties.name),
		}
		slog.Debug("vulkan physical", "index", info.Index, "name", info.Name, "vendor", info.Vendor)
		out = append(out, info)
	}
	if len(out) == 0 {
		return nil, ErrNoDevice
	}
	return out, nil
}

func (d *Device) pickIndex(index int) error {
	if index < 0 {
		return ErrNoDevice
	}
	phys, err := d.physicalDevices()
	if err != nil {
		return err
	}
	n := 0
	for _, p := range phys {
		if _, ok := d.computeFamily(p); !ok {
			continue
		}
		if n == index {
			if d.try(p) {
				return nil
			}
			return ErrNoDevice
		}
		n++
	}
	return ErrNoDevice
}

func (d *Device) try(phys uintptr) bool {
	family, ok := d.computeFamily(phys)
	if !ok {
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
	d.commandPool = pool
	d.cmd = cmd
	d.api.getMemoryProps(phys, &d.mem)
	properties := d.physicalProperties(phys)
	d.name = properties.name
	d.vendor = vendorSlug(properties.vendorID, properties.deviceType, properties.name)
	return true
}

// Name is the physical device name.
func (d *Device) Name() string {
	if d == nil {
		return ""
	}
	return d.name
}

// Vendor is the driver vendor slug (amd, nvidia, llvmpipe, …).
func (d *Device) Vendor() string {
	if d == nil {
		return ""
	}
	return d.vendor
}

// Close destroys the device and instance.
func (d *Device) Close() error {
	if d == nil || d.closed {
		return nil
	}
	d.closed = true
	if d.dev != 0 {
		if d.commandPool != 0 {
			d.api.destroyCommandPool(d.dev, d.commandPool, 0)
			d.commandPool = 0
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

// Floats is the host mapping as float32. Nil if not host-visible.
func (b *Buffer) Floats() []float32 {
	if b == nil || b.ptr == nil || b.size < 4 {
		return nil
	}
	return unsafe.Slice((*float32)(b.ptr), b.size/4)
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
	d              *Device
	module         uint64
	setLayout      uint64
	pipelineLayout uint64
	pipeline       uint64
	descriptorPool uint64
	descriptorSet  uint64
	bufferInfos    []descriptorBufferInfo
	writes         []writeDescriptorSet
	boundBuffers   []uint64
	boundLengths   []uint64
	bindings       int
	pushBytes      int
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
	layoutInfo := pipelineLayoutCreateInfo{
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
		layoutInfo.pushConstantRangeCount = 1
		layoutInfo.pPushConstantRanges = &push
	}
	var pipelineLayout uint64
	if err := check(d.api.createPipelineLayout(d.dev, &layoutInfo, 0, &pipelineLayout)); err != nil {
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
		layout:            pipelineLayout,
		basePipelineIndex: -1,
	}
	var pipeline uint64
	if err := check(d.api.createComputePipes(d.dev, 0, 1, &comp, 0, &pipeline)); err != nil {
		d.api.destroyPipelineLayout(d.dev, pipelineLayout, 0)
		d.api.destroySetLayout(d.dev, setLayout, 0)
		d.api.destroyShaderModule(d.dev, module, 0)
		return nil, fmt.Errorf("compute pipeline: %w", err)
	}
	return &Shader{
		d:              d,
		module:         module,
		setLayout:      setLayout,
		pipelineLayout: pipelineLayout,
		pipeline:       pipeline,
		bindings:       bindings,
		pushBytes:      cfg.PushBytes,
	}, nil
}

// Close destroys the pipeline.
func (s *Shader) Close() error {
	if s == nil || s.pipeline == 0 {
		return nil
	}
	d := s.d
	pipeline, layout, set, mod := s.pipeline, s.pipelineLayout, s.setLayout, s.module
	pool := s.descriptorPool
	s.pipeline, s.pipelineLayout, s.setLayout, s.module = 0, 0, 0, 0
	s.descriptorPool, s.descriptorSet = 0, 0
	s.boundBuffers, s.boundLengths = nil, nil
	if d == nil || d.closed || d.dev == 0 {
		return nil
	}
	if pool != 0 {
		d.api.destroyDescriptorPool(d.dev, pool, 0)
	}
	d.api.destroyPipeline(d.dev, pipeline, 0)
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
