package vulkan

import (
	"context"
	"errors"
	"fmt"
	"unsafe"
)

const (
	extSurface       = "VK_KHR_surface"
	extSwapchain     = "VK_KHR_swapchain"
	structureSwap    = 1000001000
	structurePresent = 1000001001
	structureImage   = 14
	structureView    = 15

	formatRGBA = 37
	formatBGRA = 44
	colorSRGB  = 0

	image2D        = 1
	tilingOptimal  = 1
	samples1       = 1
	view2D         = 1
	aspectColor    = 1
	layoutGeneral  = 1
	layoutSrc      = 6
	layoutDst      = 7
	layoutPresent  = 1000001002
	usageTransferD = 2
	usageStorage   = 8
	usageColor     = 16
	featureStorage = 2

	descStorageImage = 3
	presentImmediate = 0
	presentMailbox   = 1
	presentFIFO      = 2
	alphaOpaque      = 1
	transformIdent   = 1
	stageBottom      = 0x00002000
	suboptimal       = 1000001003
)

// hostSurface is the native window the swapchain presents to.
type hostSurface interface {
	create(d *Device, w *wsi) (uint64, error)
	destroy()
}

// Screen is a native window plus the swapchain on its device.
type Screen struct {
	d         *Device
	host      hostSurface
	surface   uint64
	swap      uint64
	format    int32
	swapRB    int32
	width     int
	height    int
	images    []uint64
	views     []uint64
	layouts   []int32
	store     uint64
	storeView uint64
	storeMem  uint64
	storeLay  int32
	acqFence  uint64
	wsi       wsi
	pipe      uint64
	pipeLay   uint64
	setLay    uint64
	pool      uint64
	set       uint64
	module    uint64
	spirvLen  int
}

type wsi struct {
	destroySurface   func(inst uintptr, surface uint64, alloc uintptr)
	surfaceSupport   func(phys uintptr, family uint32, surface uint64, supported *uint32) int32
	surfaceCaps      func(phys uintptr, surface uint64, caps *surfaceCaps) int32
	surfaceFormats   func(phys uintptr, surface uint64, count *uint32, formats *surfaceFormat) int32
	presentModes     func(phys uintptr, surface uint64, count *uint32, modes *int32) int32
	createSwapchain  func(dev uintptr, info *swapchainInfo, alloc uintptr, swap *uint64) int32
	destroySwapchain func(dev uintptr, swap uint64, alloc uintptr)
	swapchainImages  func(dev uintptr, swap uint64, count *uint32, images *uint64) int32
	acquire          func(dev uintptr, swap uint64, timeout uint64, sem uint64, fence uint64, index *uint32) int32
	queuePresent     func(queue uintptr, info *presentInfo) int32
	createImage      func(dev uintptr, info *imageInfo, alloc uintptr, image *uint64) int32
	destroyImage     func(dev uintptr, image uint64, alloc uintptr)
	imageReqs        func(dev uintptr, image uint64, reqs *memoryRequirements)
	bindImage        func(dev uintptr, image, memory, offset uint64) int32
	createView       func(dev uintptr, info *imageViewInfo, alloc uintptr, view *uint64) int32
	destroyView      func(dev uintptr, view uint64, alloc uintptr)
	cmdCopyImage     func(cmd uintptr, src uint64, srcLay int32, dst uint64, dstLay int32, count uint32, regions *imageCopy)
	cmdBlitImage     func(cmd uintptr, src uint64, srcLay int32, dst uint64, dstLay int32, count uint32, regions *imageBlit, filter int32)
	formatProps      func(phys uintptr, format int32, props *formatProps)
}

type surfaceCaps struct {
	minImageCount uint32
	maxImageCount uint32
	currentW      uint32
	currentH      uint32
	minW          uint32
	minH          uint32
	maxW          uint32
	maxH          uint32
	maxLayers     uint32
	transforms    uint32
	currentTrans  uint32
	alpha         uint32
	usage         uint32
}

type surfaceFormat struct {
	format     int32
	colorSpace int32
}

type extent2D struct {
	w, h uint32
}

type swapchainInfo struct {
	sType            int32
	pNext            uintptr
	flags            uint32
	surface          uint64
	minImageCount    uint32
	imageFormat      int32
	imageColorSpace  int32
	imageExtent      extent2D
	imageArrayLayers uint32
	imageUsage       uint32
	imageSharingMode int32
	queueCount       uint32
	pQueue           *uint32
	preTransform     uint32
	compositeAlpha   uint32
	presentMode      int32
	clipped          uint32
	oldSwapchain     uint64
}

type presentInfo struct {
	sType     int32
	pNext     uintptr
	waitCount uint32
	pWait     uintptr
	swapCount uint32
	pSwaps    *uint64
	pIndices  *uint32
	pResults  uintptr
}

type imageInfo struct {
	sType     int32
	pNext     uintptr
	flags     uint32
	imageType int32
	format    int32
	extent    [3]uint32
	mipLevels uint32
	layers    uint32
	samples   uint32
	tiling    int32
	usage     uint32
	sharing   int32
	queueN    uint32
	pQueue    *uint32
	initial   int32
}

type imageViewInfo struct {
	sType    int32
	pNext    uintptr
	flags    uint32
	image    uint64
	viewType int32
	format   int32
	comp     [4]int32
	sub      imageRange
}

type imageRange struct {
	aspect     uint32
	baseMip    uint32
	mipCount   uint32
	baseLayer  uint32
	layerCount uint32
}

type imageBarrier struct {
	sType     int32
	pNext     uintptr
	srcAccess uint32
	dstAccess uint32
	oldLayout int32
	newLayout int32
	srcQueue  uint32
	dstQueue  uint32
	image     uint64
	sub       imageRange
}

type imageLayers struct {
	aspect    uint32
	mip       uint32
	baseLayer uint32
	layers    uint32
}

type imageCopy struct {
	srcSub imageLayers
	srcOff [3]int32
	dstSub imageLayers
	dstOff [3]int32
	extent [3]uint32
}

type imageBlit struct {
	srcSub imageLayers
	srcOff [2][3]int32
	dstSub imageLayers
	dstOff [2][3]int32
}

type formatProps struct {
	linear  uint32
	optimal uint32
	buffer  uint32
}

type descriptorImageInfo struct {
	sampler uint64
	view    uint64
	layout  int32
}

// OpenScreen opens an X11 window and a swapchain on the best present-capable GPU.
func OpenScreen(ctx context.Context, width, height int, title string) (*Screen, error) {
	if width < 1 || height < 1 {
		return nil, ErrSize
	}
	d, err := openPresentInstance(ctx)
	if err != nil {
		return nil, err
	}
	s := &Screen{d: d, width: width, height: height}
	if err := s.wsi.load(d); err != nil {
		d.Close()
		return nil, err
	}
	if err := s.openWindow(title); err != nil {
		_ = s.Close()
		return nil, err
	}
	if err := s.openDevice(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}
	if err := s.makeSwapchain(); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

// Device is the GPU that owns the swapchain.
func (s *Screen) Device() *Device {
	if s == nil {
		return nil
	}
	return s.d
}

// Present copies a uint-per-channel RGBA buffer into the window.
// spirv is the present compute shader; the pipeline is cached.
func (s *Screen) Present(buf *Buffer, width, height int, spirv []byte) error {
	if s == nil || s.d == nil || s.swap == 0 {
		return ErrClosed
	}
	if buf == nil || buf.d != s.d || width != s.width || height != s.height {
		return ErrSize
	}
	if err := s.ensurePipe(spirv); err != nil {
		return err
	}
	if err := s.ensureStore(); err != nil {
		return err
	}
	var index uint32
	if err := s.acquire(&index); err != nil {
		return err
	}
	if err := s.record(buf, index); err != nil {
		return err
	}
	return s.present(index)
}

// Close destroys the swapchain, device, and X11 window.
func (s *Screen) Close() error {
	if s == nil {
		return nil
	}
	d := s.d
	var err error
	if d != nil && d.dev != 0 {
		err = d.WaitIdle()
		s.destroyPipe()
		if s.storeView != 0 {
			s.wsi.destroyView(d.dev, s.storeView, 0)
		}
		if s.store != 0 {
			s.wsi.destroyImage(d.dev, s.store, 0)
		}
		if s.storeMem != 0 {
			d.api.freeMemory(d.dev, s.storeMem, 0)
		}
		for _, v := range s.views {
			if v != 0 {
				s.wsi.destroyView(d.dev, v, 0)
			}
		}
		if s.acqFence != 0 {
			d.api.destroyFence(d.dev, s.acqFence, 0)
		}
		if s.swap != 0 {
			s.wsi.destroySwapchain(d.dev, s.swap, 0)
		}
	}
	if d != nil && d.inst != 0 && s.surface != 0 && s.wsi.destroySurface != nil {
		s.wsi.destroySurface(d.inst, s.surface, 0)
	}
	if s.host != nil {
		s.host.destroy()
		s.host = nil
	}
	s.surface, s.swap = 0, 0
	if d != nil {
		err = errors.Join(err, d.Close())
		s.d = nil
	}
	return err
}

func openPresentInstance(ctx context.Context) (*Device, error) {
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
	exts := append([]string{extSurface}, surfaceExtensions()...)
	if hasExt(d.api.instanceExts(), extPortabilityEnum) {
		exts = append(exts, extPortabilityEnum)
	}
	appName := cstr("lewkit")
	app := applicationInfo{
		sType:            structureApplicationInfo,
		pApplicationName: appName,
		apiVersion:       apiVersion11,
	}
	var flags uint32
	if hasExt(exts, extPortabilityEnum) {
		flags = instanceEnumeratePortability
	}
	extPtrs, keep := cStrings(exts)
	_ = keep
	info := instanceCreateInfo{
		sType:                   structureInstanceCreateInfo,
		flags:                   flags,
		pApplicationInfo:        &app,
		enabledExtensionCount:   uint32(len(exts)),
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

func (w *wsi) load(d *Device) error {
	bind := func(name string, dst any) error {
		return d.api.bind(d.api.getInstanceProcAddr, d.inst, name, dst)
	}
	pairs := []struct {
		name string
		dst  any
	}{
		{"vkDestroySurfaceKHR", &w.destroySurface},
		{"vkGetPhysicalDeviceSurfaceSupportKHR", &w.surfaceSupport},
		{"vkGetPhysicalDeviceSurfaceCapabilitiesKHR", &w.surfaceCaps},
		{"vkGetPhysicalDeviceSurfaceFormatsKHR", &w.surfaceFormats},
		{"vkGetPhysicalDeviceSurfacePresentModesKHR", &w.presentModes},
		{"vkGetPhysicalDeviceFormatProperties", &w.formatProps},
	}
	for _, p := range pairs {
		if err := bind(p.name, p.dst); err != nil {
			return err
		}
	}
	return loadHost(w, d)
}

func (w *wsi) loadDevice(d *Device) error {
	bind := func(name string, dst any) error {
		return d.api.bind(d.api.getDeviceProcAddr, d.dev, name, dst)
	}
	pairs := []struct {
		name string
		dst  any
	}{
		{"vkCreateSwapchainKHR", &w.createSwapchain},
		{"vkDestroySwapchainKHR", &w.destroySwapchain},
		{"vkGetSwapchainImagesKHR", &w.swapchainImages},
		{"vkAcquireNextImageKHR", &w.acquire},
		{"vkQueuePresentKHR", &w.queuePresent},
		{"vkCreateImage", &w.createImage},
		{"vkDestroyImage", &w.destroyImage},
		{"vkGetImageMemoryRequirements", &w.imageReqs},
		{"vkBindImageMemory", &w.bindImage},
		{"vkCreateImageView", &w.createView},
		{"vkDestroyImageView", &w.destroyView},
		{"vkCmdCopyImage", &w.cmdCopyImage},
		{"vkCmdBlitImage", &w.cmdBlitImage},
	}
	for _, p := range pairs {
		if err := bind(p.name, p.dst); err != nil {
			return err
		}
	}
	return nil
}

func (s *Screen) openWindow(title string) error {
	host, err := openHost(s.width, s.height, title)
	if err != nil {
		return err
	}
	s.host = host
	surface, err := host.create(s.d, &s.wsi)
	if err != nil {
		return err
	}
	s.surface = surface
	return nil
}

func (s *Screen) openDevice(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	phys, err := s.d.physicalDevices()
	if err != nil {
		return err
	}
	bestW := -1
	var best uintptr
	var bestFam uint32
	for _, p := range phys {
		fam, ok := presentFamily(s, p)
		if !ok {
			continue
		}
		props := s.d.physicalProperties(p)
		weight := deviceTypeFrom(props.deviceType, props.name).Weight()
		if weight > bestW {
			bestW = weight
			best = p
			bestFam = fam
		}
	}
	if best == 0 {
		return fmt.Errorf("%w: no present queue", ErrNoDevice)
	}
	s.d.family = bestFam
	s.d.familyPinned = true
	s.d.wantExt = []string{extSwapchain}
	if !s.d.try(best) {
		return ErrNoDevice
	}
	return s.wsi.loadDevice(s.d)
}

func presentFamily(s *Screen, phys uintptr) (uint32, bool) {
	var nq uint32
	s.d.api.getQueueFamilies(phys, &nq, nil)
	if nq == 0 {
		return 0, false
	}
	fams := make([]queueFamilyProperties, nq)
	s.d.api.getQueueFamilies(phys, &nq, &fams[0])
	for i, f := range fams[:nq] {
		if f.queueFlags&queueComputeBit == 0 || f.queueCount == 0 {
			continue
		}
		var supported uint32
		if check(s.wsi.surfaceSupport(phys, uint32(i), s.surface, &supported)) != nil || supported == 0 {
			continue
		}
		return uint32(i), true
	}
	return 0, false
}

func (s *Screen) makeSwapchain() error {
	var caps surfaceCaps
	if err := check(s.wsi.surfaceCaps(s.d.phys, s.surface, &caps)); err != nil {
		return fmt.Errorf("surface caps: %w", err)
	}
	format, swapRB, err := s.pickFormat()
	if err != nil {
		return err
	}
	s.format, s.swapRB = format, swapRB
	mode := s.pickMode()
	count := caps.minImageCount
	if count < 2 {
		count = 2
	}
	if caps.maxImageCount != 0 && count > caps.maxImageCount {
		count = caps.maxImageCount
	}
	info := swapchainInfo{
		sType:            structureSwap,
		surface:          s.surface,
		minImageCount:    count,
		imageFormat:      format,
		imageColorSpace:  colorSRGB,
		imageExtent:      extent2D{uint32(s.width), uint32(s.height)},
		imageArrayLayers: 1,
		imageUsage:       usageTransferD | usageColor,
		preTransform:     caps.currentTrans,
		compositeAlpha:   alphaOpaque,
		presentMode:      mode,
		clipped:          1,
	}
	if caps.currentTrans == 0 {
		info.preTransform = transformIdent
	}
	if err := check(s.wsi.createSwapchain(s.d.dev, &info, 0, &s.swap)); err != nil {
		return fmt.Errorf("swapchain: %w", err)
	}
	var n uint32
	if err := check(s.wsi.swapchainImages(s.d.dev, s.swap, &n, nil)); err != nil || n == 0 {
		return fmt.Errorf("swapchain images: %w", err)
	}
	s.images = make([]uint64, n)
	if err := check(s.wsi.swapchainImages(s.d.dev, s.swap, &n, &s.images[0])); err != nil {
		return fmt.Errorf("swapchain images: %w", err)
	}
	s.images = s.images[:n]
	s.views = make([]uint64, n)
	s.layouts = make([]int32, n)
	fence := fenceCreateInfo{sType: structureFenceCreateInfo}
	if err := check(s.d.api.createFence(s.d.dev, &fence, 0, &s.acqFence)); err != nil {
		return fmt.Errorf("acquire fence: %w", err)
	}
	return nil
}

func (s *Screen) pickFormat() (int32, int32, error) {
	var n uint32
	if err := check(s.wsi.surfaceFormats(s.d.phys, s.surface, &n, nil)); err != nil || n == 0 {
		return 0, 0, fmt.Errorf("surface formats: %w", err)
	}
	formats := make([]surfaceFormat, n)
	if err := check(s.wsi.surfaceFormats(s.d.phys, s.surface, &n, &formats[0])); err != nil {
		return 0, 0, fmt.Errorf("surface formats: %w", err)
	}
	for _, f := range formats[:n] {
		if f.format == formatRGBA {
			return formatRGBA, 0, nil
		}
	}
	for _, f := range formats[:n] {
		if f.format == formatBGRA {
			return formatBGRA, 1, nil
		}
	}
	return 0, 0, fmt.Errorf("%w: rgba8 surface", ErrUnavailable)
}

func (s *Screen) pickMode() int32 {
	var n uint32
	if check(s.wsi.presentModes(s.d.phys, s.surface, &n, nil)) != nil || n == 0 {
		return presentFIFO
	}
	modes := make([]int32, n)
	if check(s.wsi.presentModes(s.d.phys, s.surface, &n, &modes[0])) != nil {
		return presentFIFO
	}
	has := func(want int32) bool {
		for _, m := range modes[:n] {
			if m == want {
				return true
			}
		}
		return false
	}
	if has(presentImmediate) {
		return presentImmediate
	}
	if has(presentMailbox) {
		return presentMailbox
	}
	return presentFIFO
}

func (s *Screen) ensureStore() error {
	if s.store != 0 {
		return nil
	}
	var props formatProps
	s.wsi.formatProps(s.d.phys, formatRGBA, &props)
	if props.optimal&featureStorage == 0 {
		return fmt.Errorf("%w: storage image", ErrUnavailable)
	}
	info := imageInfo{
		sType:     structureImage,
		imageType: image2D,
		format:    formatRGBA,
		extent:    [3]uint32{uint32(s.width), uint32(s.height), 1},
		mipLevels: 1,
		layers:    1,
		samples:   samples1,
		tiling:    tilingOptimal,
		usage:     usageStorage | usageTransferD | 1,
		initial:   0,
	}
	var image uint64
	if err := check(s.wsi.createImage(s.d.dev, &info, 0, &image)); err != nil {
		return fmt.Errorf("storage image: %w", err)
	}
	var req memoryRequirements
	s.wsi.imageReqs(s.d.dev, image, &req)
	idx, ok := s.d.memoryType(req.memoryTypeBits, memoryDeviceLocal)
	if !ok {
		s.wsi.destroyImage(s.d.dev, image, 0)
		return fmt.Errorf("%w: device local image", ErrUnavailable)
	}
	alloc := memoryAllocateInfo{sType: structureMemoryAllocateInfo, allocationSize: req.size, memoryTypeIndex: idx}
	var mem uint64
	if err := check(s.d.api.allocateMemory(s.d.dev, &alloc, 0, &mem)); err != nil {
		s.wsi.destroyImage(s.d.dev, image, 0)
		return fmt.Errorf("image memory: %w", err)
	}
	if err := check(s.wsi.bindImage(s.d.dev, image, mem, 0)); err != nil {
		s.d.api.freeMemory(s.d.dev, mem, 0)
		s.wsi.destroyImage(s.d.dev, image, 0)
		return fmt.Errorf("bind image: %w", err)
	}
	view, err := s.makeView(image, formatRGBA)
	if err != nil {
		s.d.api.freeMemory(s.d.dev, mem, 0)
		s.wsi.destroyImage(s.d.dev, image, 0)
		return err
	}
	s.store, s.storeView, s.storeMem = image, view, mem
	return nil
}

func (s *Screen) makeView(image uint64, format int32) (uint64, error) {
	info := imageViewInfo{
		sType:    structureView,
		image:    image,
		viewType: view2D,
		format:   format,
		sub:      imageRange{aspect: aspectColor, mipCount: 1, layerCount: 1},
	}
	var view uint64
	if err := check(s.wsi.createView(s.d.dev, &info, 0, &view)); err != nil {
		return 0, fmt.Errorf("image view: %w", err)
	}
	return view, nil
}

func (s *Screen) ensurePipe(spirv []byte) error {
	if s.pipe != 0 {
		return nil
	}
	if len(spirv) < 20 || len(spirv)%4 != 0 {
		return ErrShader
	}
	code := make([]uint32, len(spirv)/4)
	for i := range code {
		code[i] = uint32(spirv[i*4]) | uint32(spirv[i*4+1])<<8 | uint32(spirv[i*4+2])<<16 | uint32(spirv[i*4+3])<<24
	}
	modInfo := shaderModuleCreateInfo{sType: structureShaderModuleCreateInfo, codeSize: uintptr(len(spirv)), pCode: &code[0]}
	var module uint64
	if err := check(s.d.api.createShaderModule(s.d.dev, &modInfo, 0, &module)); err != nil {
		return fmt.Errorf("present shader: %w", err)
	}
	binds := [2]descriptorSetLayoutBinding{
		{binding: 0, descriptorType: descriptorStorageBuffer, descriptorCount: 1, stageFlags: shaderStageCompute},
		{binding: 1, descriptorType: descStorageImage, descriptorCount: 1, stageFlags: shaderStageCompute},
	}
	setInfo := descriptorSetLayoutCreateInfo{sType: structureDescriptorSetLayoutCreateInfo, bindingCount: 2, pBindings: &binds[0]}
	var setLay uint64
	if err := check(s.d.api.createSetLayout(s.d.dev, &setInfo, 0, &setLay)); err != nil {
		s.d.api.destroyShaderModule(s.d.dev, module, 0)
		return err
	}
	push := pushConstantRange{stageFlags: shaderStageCompute, size: 12}
	layInfo := pipelineLayoutCreateInfo{
		sType:                  structurePipelineLayoutCreateInfo,
		setLayoutCount:         1,
		pSetLayouts:            &setLay,
		pushConstantRangeCount: 1,
		pPushConstantRanges:    &push,
	}
	var pipeLay uint64
	if err := check(s.d.api.createPipelineLayout(s.d.dev, &layInfo, 0, &pipeLay)); err != nil {
		s.d.api.destroySetLayout(s.d.dev, setLay, 0)
		s.d.api.destroyShaderModule(s.d.dev, module, 0)
		return err
	}
	entry := cstr("main")
	comp := computePipelineCreateInfo{
		sType:  structureComputePipelineCreateInfo,
		stage:  pipelineShaderStageCreateInfo{sType: structurePipelineShaderStageCreateInfo, stage: shaderStageCompute, module: module, pName: entry},
		layout: pipeLay, basePipelineIndex: -1,
	}
	var pipe uint64
	if err := check(s.d.api.createComputePipes(s.d.dev, 0, 1, &comp, 0, &pipe)); err != nil {
		s.d.api.destroyPipelineLayout(s.d.dev, pipeLay, 0)
		s.d.api.destroySetLayout(s.d.dev, setLay, 0)
		s.d.api.destroyShaderModule(s.d.dev, module, 0)
		return err
	}
	sizes := [2]descriptorPoolSize{
		{typ: descriptorStorageBuffer, descriptorCount: 1},
		{typ: descStorageImage, descriptorCount: 1},
	}
	poolInfo := descriptorPoolCreateInfo{sType: structureDescriptorPoolCreateInfo, maxSets: 1, poolSizeCount: 2, pPoolSizes: &sizes[0]}
	var pool uint64
	if err := check(s.d.api.createDescriptorPool(s.d.dev, &poolInfo, 0, &pool)); err != nil {
		s.d.api.destroyPipeline(s.d.dev, pipe, 0)
		s.d.api.destroyPipelineLayout(s.d.dev, pipeLay, 0)
		s.d.api.destroySetLayout(s.d.dev, setLay, 0)
		s.d.api.destroyShaderModule(s.d.dev, module, 0)
		return err
	}
	alloc := descriptorSetAllocateInfo{sType: structureDescriptorSetAllocateInfo, descriptorPool: pool, descriptorSetCount: 1, pSetLayouts: &setLay}
	var set uint64
	if err := check(s.d.api.allocateDescriptorSets(s.d.dev, &alloc, &set)); err != nil {
		s.d.api.destroyDescriptorPool(s.d.dev, pool, 0)
		s.d.api.destroyPipeline(s.d.dev, pipe, 0)
		s.d.api.destroyPipelineLayout(s.d.dev, pipeLay, 0)
		s.d.api.destroySetLayout(s.d.dev, setLay, 0)
		s.d.api.destroyShaderModule(s.d.dev, module, 0)
		return err
	}
	s.module, s.setLay, s.pipeLay, s.pipe, s.pool, s.set = module, setLay, pipeLay, pipe, pool, set
	s.spirvLen = len(spirv)
	return nil
}

func (s *Screen) destroyPipe() {
	d := s.d
	if d == nil || d.dev == 0 {
		return
	}
	if s.pool != 0 {
		d.api.destroyDescriptorPool(d.dev, s.pool, 0)
	}
	if s.pipe != 0 {
		d.api.destroyPipeline(d.dev, s.pipe, 0)
	}
	if s.pipeLay != 0 {
		d.api.destroyPipelineLayout(d.dev, s.pipeLay, 0)
	}
	if s.setLay != 0 {
		d.api.destroySetLayout(d.dev, s.setLay, 0)
	}
	if s.module != 0 {
		d.api.destroyShaderModule(d.dev, s.module, 0)
	}
}

func (s *Screen) acquire(index *uint32) error {
	r := s.wsi.acquire(s.d.dev, s.swap, ^uint64(0), 0, s.acqFence, index)
	if err := presentOK(r); err != nil {
		return fmt.Errorf("acquire: %w", err)
	}
	if err := check(s.d.api.waitForFences(s.d.dev, 1, &s.acqFence, 1, ^uint64(0))); err != nil {
		return fmt.Errorf("acquire wait: %w", err)
	}
	return check(s.d.api.resetFences(s.d.dev, 1, &s.acqFence))
}

func (s *Screen) record(buf *Buffer, index uint32) error {
	d := s.d
	if d.pending || d.recording {
		return ErrBusy
	}
	view := s.views[index]
	if view == 0 {
		v, err := s.makeView(s.images[index], s.format)
		if err != nil {
			return err
		}
		s.views[index] = v
		view = v
	}
	img := descriptorImageInfo{view: s.storeView, layout: layoutGeneral}
	binfo := descriptorBufferInfo{buffer: buf.buf, rang: uint64(buf.size)}
	writes := [2]writeDescriptorSet{
		{sType: structureWriteDescriptorSet, dstSet: s.set, dstBinding: 0, descriptorCount: 1, descriptorType: descriptorStorageBuffer, pBufferInfo: &binfo},
		{sType: structureWriteDescriptorSet, dstSet: s.set, dstBinding: 1, descriptorCount: 1, descriptorType: descStorageImage, pImageInfo: uintptr(unsafe.Pointer(&img))},
	}
	d.api.updateDescriptorSets(d.dev, 2, &writes[0], 0, 0)
	if err := check(d.api.resetCommandBuffer(d.cmd, 0)); err != nil {
		return err
	}
	begin := commandBufferBeginInfo{sType: structureCommandBufferBeginInfo}
	if err := check(d.api.beginCommandBuffer(d.cmd, &begin)); err != nil {
		return err
	}
	d.recording = true
	if err := d.flush(buf); err != nil {
		d.recording = false
		return err
	}
	s.barrier(s.store, s.storeLay, layoutGeneral, 0, accessShaderWrite, 0, stageCompute)
	s.storeLay = layoutGeneral
	d.api.cmdBindPipeline(d.cmd, bindPointCompute, s.pipe)
	d.api.cmdBindSets(d.cmd, bindPointCompute, s.pipeLay, 0, 1, &s.set, 0, nil)
	push := [3]uint32{uint32(s.width), uint32(s.height), uint32(s.swapRB)}
	d.api.cmdPushConstants(d.cmd, s.pipeLay, shaderStageCompute, 0, 12, uintptr(unsafe.Pointer(&push[0])))
	gx := uint32((s.width + 15) / 16)
	gy := uint32((s.height + 15) / 16)
	d.api.cmdDispatch(d.cmd, gx, gy, 1)
	s.barrier(s.store, layoutGeneral, layoutSrc, accessShaderWrite, accessTransferRead, stageCompute, stageTransfer)
	s.storeLay = layoutSrc
	s.barrier(s.images[index], s.layouts[index], layoutDst, 0, accessTransferWrite, 0, stageTransfer)
	if s.format == formatRGBA {
		region := imageCopy{
			srcSub: imageLayers{aspect: aspectColor, layers: 1},
			dstSub: imageLayers{aspect: aspectColor, layers: 1},
			extent: [3]uint32{uint32(s.width), uint32(s.height), 1},
		}
		s.wsi.cmdCopyImage(d.cmd, s.store, layoutSrc, s.images[index], layoutDst, 1, &region)
	} else {
		blit := imageBlit{
			srcSub: imageLayers{aspect: aspectColor, layers: 1},
			dstSub: imageLayers{aspect: aspectColor, layers: 1},
		}
		blit.srcOff[1] = [3]int32{int32(s.width), int32(s.height), 1}
		blit.dstOff[1] = blit.srcOff[1]
		s.wsi.cmdBlitImage(d.cmd, s.store, layoutSrc, s.images[index], layoutDst, 1, &blit, 0)
	}
	s.barrier(s.images[index], layoutDst, layoutPresent, accessTransferWrite, 0, stageTransfer, stageBottom)
	s.layouts[index] = layoutPresent
	if err := check(d.api.endCommandBuffer(d.cmd)); err != nil {
		d.recording = false
		return err
	}
	d.recording = false
	submit := submitInfo{sType: structureSubmitInfo, commandBufferCount: 1, pCommandBuffers: &d.cmd}
	if err := check(d.api.queueSubmit(d.queue, 1, &submit, d.fence)); err != nil {
		return fmt.Errorf("present submit: %w", err)
	}
	d.pending = true
	if err := check(d.api.waitForFences(d.dev, 1, &d.fence, 1, ^uint64(0))); err != nil {
		return err
	}
	d.pending = false
	return check(d.api.resetFences(d.dev, 1, &d.fence))
}

func (s *Screen) barrier(image uint64, old, new int32, srcAcc, dstAcc, srcStage, dstStage uint32) {
	if old == new {
		return
	}
	bar := imageBarrier{
		sType: structureImageBarrier, srcAccess: srcAcc, dstAccess: dstAcc,
		oldLayout: old, newLayout: new, srcQueue: ^uint32(0), dstQueue: ^uint32(0),
		image: image, sub: imageRange{aspect: aspectColor, mipCount: 1, layerCount: 1},
	}
	s.d.api.cmdBarrier(s.d.cmd, srcStage, dstStage, 0, 0, nil, 0, 0, 1, uintptr(unsafe.Pointer(&bar)))
}

func (s *Screen) present(index uint32) error {
	swap := s.swap
	info := presentInfo{sType: structurePresent, swapCount: 1, pSwaps: &swap, pIndices: &index}
	if err := presentOK(s.wsi.queuePresent(s.d.queue, &info)); err != nil {
		return fmt.Errorf("queue present: %w", err)
	}
	return nil
}

func presentOK(r int32) error {
	if r == 0 || r == suboptimal {
		return nil
	}
	return Result(r)
}

const structureImageBarrier = 45
