package vulkan

import (
	"unsafe"
)

const (
	structureVertexInput      = 19
	structureInputAssembly    = 20
	structureViewport         = 22
	structureRaster           = 23
	structureMultisample      = 24
	structureColorBlend       = 26
	structureDynamic          = 27
	structureGraphicsPipeline = 28
	structureFramebuffer      = 37
	structureRenderPass       = 38
	structureRenderPassBegin  = 43
	structureBufferBarrier    = 44

	topologyStrip    = 4
	layoutColor      = 2
	loadClear        = 1
	loadDontCare     = 2
	storeStore       = 0
	storeDontCare    = 1
	dynamicViewport  = 0
	dynamicScissor   = 1
	blendOne         = 1
	blendOneMinusSrc = 7
	colorWriteAll    = 0xF
	pipeStageHost    = 0x00004000
	pipeStageVert    = 0x00000008
	pipeStageFrag    = 0x00000080
	pipeStageColor   = 0x00000400
	accessColorWrite = 0x00000100
	subpassExternal  = ^uint32(0)
	instanceStride   = 64
)

type vertexInputState struct {
	sType          int32
	pNext          uintptr
	flags          uint32
	bindingCount   uint32
	bindings       uintptr
	attributeCount uint32
	attributes     uintptr
}

type inputAssemblyState struct {
	sType    int32
	pNext    uintptr
	flags    uint32
	topology uint32
	restart  uint32
}

type viewportState struct {
	sType         int32
	pNext         uintptr
	flags         uint32
	viewportCount uint32
	viewports     uintptr
	scissorCount  uint32
	scissors      uintptr
}

type rasterState struct {
	sType      int32
	pNext      uintptr
	flags      uint32
	depthClamp uint32
	discard    uint32
	polygon    uint32
	cull       uint32
	front      uint32
	biasEnable uint32
	biasConst  float32
	biasClamp  float32
	biasSlope  float32
	lineWidth  float32
}

type multisampleState struct {
	sType      int32
	pNext      uintptr
	flags      uint32
	samples    uint32
	shading    uint32
	minSample  float32
	sampleMask uintptr
	alphaCover uint32
	alphaToOne uint32
}

type blendAttachment struct {
	enable    uint32
	srcColor  uint32
	dstColor  uint32
	colorOp   uint32
	srcAlpha  uint32
	dstAlpha  uint32
	alphaOp   uint32
	writeMask uint32
}

type colorBlendState struct {
	sType       int32
	pNext       uintptr
	flags       uint32
	logicEnable uint32
	logicOp     uint32
	count       uint32
	attachments *blendAttachment
	constants   [4]float32
}

type dynamicStateInfo struct {
	sType  int32
	pNext  uintptr
	flags  uint32
	count  uint32
	states *uint32
}

type graphicsPipelineInfo struct {
	sType         int32
	pNext         uintptr
	flags         uint32
	stageCount    uint32
	stages        *pipelineShaderStageCreateInfo
	vertexInput   *vertexInputState
	inputAssembly *inputAssemblyState
	tessellation  uintptr
	viewport      *viewportState
	raster        *rasterState
	multisample   *multisampleState
	depth         uintptr
	blend         *colorBlendState
	dynamic       *dynamicStateInfo
	layout        uint64
	renderPass    uint64
	subpass       uint32
	baseHandle    uint64
	baseIndex     int32
}

type attachmentDescription struct {
	flags        uint32
	format       int32
	samples      uint32
	loadOp       uint32
	storeOp      uint32
	stencilLoad  uint32
	stencilStore uint32
	initial      int32
	final        int32
}

type attachmentReference struct {
	attachment uint32
	layout     int32
}

type subpassDescription struct {
	flags         uint32
	bindPoint     uint32
	inputCount    uint32
	inputs        uintptr
	colorCount    uint32
	colors        *attachmentReference
	resolve       uintptr
	depth         uintptr
	preserveCount uint32
	preserve      uintptr
}

type subpassDependency struct {
	srcSubpass uint32
	dstSubpass uint32
	srcStage   uint32
	dstStage   uint32
	srcAccess  uint32
	dstAccess  uint32
	flags      uint32
}

type renderPassInfo struct {
	sType        int32
	pNext        uintptr
	flags        uint32
	attachCount  uint32
	attachments  *attachmentDescription
	subpassCount uint32
	subpasses    *subpassDescription
	dependCount  uint32
	depends      *subpassDependency
}

type framebufferInfo struct {
	sType       int32
	pNext       uintptr
	flags       uint32
	renderPass  uint64
	attachCount uint32
	attachments *uint64
	width       uint32
	height      uint32
	layers      uint32
}

type rect2D struct {
	x      int32
	y      int32
	width  uint32
	height uint32
}

type clearColor struct {
	r, g, b, a float32
}

type renderPassBegin struct {
	sType       int32
	pNext       uintptr
	renderPass  uint64
	framebuffer uint64
	area        rect2D
	clearCount  uint32
	clears      *clearColor
}

type viewport struct {
	x, y, width, height, minDepth, maxDepth float32
}

type bufferBarrier struct {
	sType     int32
	pNext     uintptr
	srcAccess uint32
	dstAccess uint32
	srcQueue  uint32
	dstQueue  uint32
	buffer    uint64
	offset    uint64
	size      uint64
}

// Draw paints instances and an optional ink image with graphics pipelines.
// instances is 16 floats per rounded rect (box, color, radius+clip xyz, clip height).
// ink is tightly packed RGBA8, or nil. fillVert, fillFrag, inkVert, and inkFrag are SPIR-V.
func (s *Screen) Draw(instances, ink []byte, width, height int, fillVert, fillFrag, inkVert, inkFrag []byte) error {
	if s == nil || s.d == nil || s.swap == 0 {
		return ErrClosed
	}
	if width < 1 || height < 1 {
		return ErrSize
	}
	if len(instances)%instanceStride != 0 {
		return ErrSize
	}
	if s.host != nil {
		s.host.poll()
	}
	if width != s.width || height != s.height {
		if err := s.resizeTo(width, height); err != nil {
			return err
		}
	}
	if err := s.ensurePass(); err != nil {
		return err
	}
	fills := len(instances) / instanceStride
	if fills > 0 {
		if err := s.ensureGraphics(&s.fillPipe, &s.fillLay, &s.fillSetLay, &s.fillPool, &s.fillSet, &s.fillVert, &s.fillFrag, fillVert, fillFrag, shaderStageVertex|shaderStageFragment); err != nil {
			return err
		}
		if err := s.growBuf(&s.fillBuf, len(instances)); err != nil {
			return err
		}
		if err := s.fillBuf.Write(instances); err != nil {
			return err
		}
	}
	inked := len(ink) >= width*height*4
	if inked {
		if err := s.ensureGraphics(&s.inkPipe, &s.inkLay, &s.inkSetLay, &s.inkPool, &s.inkSet, &s.inkVert, &s.inkFrag, inkVert, inkFrag, shaderStageFragment); err != nil {
			return err
		}
		need := width * height * 4
		if err := s.growBuf(&s.inkBuf, need); err != nil {
			return err
		}
		if err := s.inkBuf.Write(ink[:need]); err != nil {
			return err
		}
	}
	if err := s.ensureFrames(); err != nil {
		return err
	}
	var index uint32
	if err := s.acquire(&index); err != nil {
		return err
	}
	if err := s.recordDraw(index, fills, inked); err != nil {
		return err
	}
	return s.present(index)
}

func (s *Screen) growBuf(slot **Buffer, bytes int) error {
	if bytes < 1 {
		bytes = 4
	}
	if *slot != nil && (*slot).size >= bytes {
		return nil
	}
	if *slot != nil {
		_ = (*slot).Close()
		*slot = nil
	}
	buf, err := s.d.Buffer(bytes)
	if err != nil {
		return err
	}
	*slot = buf
	return nil
}

func (s *Screen) ensurePass() error {
	if s.drawPass != 0 && s.drawFormat == s.format {
		return nil
	}
	s.destroyFrames()
	if s.drawPass != 0 {
		s.d.api.destroyRenderPass(s.d.dev, s.drawPass, 0)
		s.drawPass = 0
		s.destroyPipes()
	}
	attach := attachmentDescription{
		format: s.format, samples: samples1,
		loadOp: loadClear, storeOp: storeStore,
		stencilLoad: loadDontCare, stencilStore: storeDontCare,
		initial: 0, final: layoutPresent,
	}
	ref := attachmentReference{attachment: 0, layout: layoutColor}
	sub := subpassDescription{bindPoint: bindPointGraphics, colorCount: 1, colors: &ref}
	dep := subpassDependency{
		srcSubpass: subpassExternal, dstSubpass: 0,
		srcStage: pipeStageColor, dstStage: pipeStageColor,
		dstAccess: accessColorWrite,
	}
	info := renderPassInfo{
		sType: structureRenderPass, attachCount: 1, attachments: &attach,
		subpassCount: 1, subpasses: &sub, dependCount: 1, depends: &dep,
	}
	var pass uint64
	if err := check(s.d.api.createRenderPass(s.d.dev, uintptr(unsafe.Pointer(&info)), 0, &pass)); err != nil {
		return err
	}
	s.drawPass = pass
	s.drawFormat = s.format
	return nil
}

func (s *Screen) ensureGraphics(pipe, lay, setLay, pool, set, vert, frag *uint64, vertCode, fragCode []byte, fragStages uint32) error {
	if *pipe != 0 {
		return nil
	}
	if *vert != 0 || *frag != 0 || *lay != 0 {
		s.destroyPipeSlot(pipe, lay, setLay, pool, vert, frag)
	}
	if s.d.api.createGraphicsPipes == nil {
		return ErrUnavailable
	}
	vmod, err := s.shaderModule(vertCode)
	if err != nil {
		return err
	}
	fmod, err := s.shaderModule(fragCode)
	if err != nil {
		s.d.api.destroyShaderModule(s.d.dev, vmod, 0)
		return err
	}
	*vert, *frag = vmod, fmod
	bind := descriptorSetLayoutBinding{binding: 0, descriptorType: descriptorStorageBuffer, descriptorCount: 1, stageFlags: shaderStageVertex | fragStages}
	setInfo := descriptorSetLayoutCreateInfo{sType: structureDescriptorSetLayoutCreateInfo, bindingCount: 1, pBindings: &bind}
	if err := check(s.d.api.createSetLayout(s.d.dev, &setInfo, 0, setLay)); err != nil {
		return err
	}
	push := pushConstantRange{stageFlags: shaderStageVertex | fragStages, size: 12}
	layInfo := pipelineLayoutCreateInfo{
		sType: structurePipelineLayoutCreateInfo, setLayoutCount: 1, pSetLayouts: setLay,
		pushConstantRangeCount: 1, pPushConstantRanges: &push,
	}
	if err := check(s.d.api.createPipelineLayout(s.d.dev, &layInfo, 0, lay)); err != nil {
		return err
	}
	entry := cstr("main")
	stages := [2]pipelineShaderStageCreateInfo{
		{sType: structurePipelineShaderStageCreateInfo, stage: shaderStageVertex, module: vmod, pName: entry},
		{sType: structurePipelineShaderStageCreateInfo, stage: shaderStageFragment, module: fmod, pName: entry},
	}
	vertex := vertexInputState{sType: structureVertexInput}
	assembly := inputAssemblyState{sType: structureInputAssembly, topology: topologyStrip}
	view := viewportState{sType: structureViewport, viewportCount: 1, scissorCount: 1}
	raster := rasterState{sType: structureRaster, lineWidth: 1}
	multi := multisampleState{sType: structureMultisample, samples: samples1}
	blendAtt := blendAttachment{
		enable: 1, srcColor: blendOne, dstColor: blendOneMinusSrc, srcAlpha: blendOne, dstAlpha: blendOneMinusSrc, writeMask: colorWriteAll,
	}
	blend := colorBlendState{sType: structureColorBlend, count: 1, attachments: &blendAtt}
	dynStates := [2]uint32{dynamicViewport, dynamicScissor}
	dynamic := dynamicStateInfo{sType: structureDynamic, count: 2, states: &dynStates[0]}
	gp := graphicsPipelineInfo{
		sType: structureGraphicsPipeline, stageCount: 2, stages: &stages[0],
		vertexInput: &vertex, inputAssembly: &assembly, viewport: &view,
		raster: &raster, multisample: &multi, blend: &blend, dynamic: &dynamic,
		layout: *lay, renderPass: s.drawPass, baseIndex: -1,
	}
	if err := check(s.d.api.createGraphicsPipes(s.d.dev, 0, 1, uintptr(unsafe.Pointer(&gp)), 0, pipe)); err != nil {
		return err
	}
	sizes := descriptorPoolSize{typ: descriptorStorageBuffer, descriptorCount: 1}
	poolInfo := descriptorPoolCreateInfo{sType: structureDescriptorPoolCreateInfo, maxSets: 1, poolSizeCount: 1, pPoolSizes: &sizes}
	if err := check(s.d.api.createDescriptorPool(s.d.dev, &poolInfo, 0, pool)); err != nil {
		return err
	}
	alloc := descriptorSetAllocateInfo{sType: structureDescriptorSetAllocateInfo, descriptorPool: *pool, descriptorSetCount: 1, pSetLayouts: setLay}
	return check(s.d.api.allocateDescriptorSets(s.d.dev, &alloc, set))
}

func (s *Screen) shaderModule(spirv []byte) (uint64, error) {
	if len(spirv) < 20 || len(spirv)%4 != 0 {
		return 0, ErrShader
	}
	code := make([]uint32, len(spirv)/4)
	for i := range code {
		code[i] = uint32(spirv[i*4]) | uint32(spirv[i*4+1])<<8 | uint32(spirv[i*4+2])<<16 | uint32(spirv[i*4+3])<<24
	}
	info := shaderModuleCreateInfo{sType: structureShaderModuleCreateInfo, codeSize: uintptr(len(spirv)), pCode: &code[0]}
	var module uint64
	if err := check(s.d.api.createShaderModule(s.d.dev, &info, 0, &module)); err != nil {
		return 0, err
	}
	return module, nil
}

func (s *Screen) ensureFrames() error {
	if len(s.drawFrames) == len(s.images) && (len(s.images) == 0 || s.drawFrames[0] != 0) {
		return nil
	}
	s.destroyFrames()
	s.drawFrames = make([]uint64, len(s.images))
	for i, image := range s.images {
		if s.views[i] == 0 {
			view, err := s.makeView(image, s.format)
			if err != nil {
				return err
			}
			s.views[i] = view
		}
		view := s.views[i]
		info := framebufferInfo{
			sType: structureFramebuffer, renderPass: s.drawPass, attachCount: 1, attachments: &view,
			width: uint32(s.width), height: uint32(s.height), layers: 1,
		}
		var fb uint64
		if err := check(s.d.api.createFramebuffer(s.d.dev, uintptr(unsafe.Pointer(&info)), 0, &fb)); err != nil {
			return err
		}
		s.drawFrames[i] = fb
	}
	return nil
}

func (s *Screen) recordDraw(index uint32, fills int, inked bool) error {
	d := s.d
	if d.pending || d.recording {
		return ErrBusy
	}
	if err := check(d.api.resetCommandBuffer(d.cmd, 0)); err != nil {
		return err
	}
	begin := commandBufferBeginInfo{sType: structureCommandBufferBeginInfo}
	if err := check(d.api.beginCommandBuffer(d.cmd, &begin)); err != nil {
		return err
	}
	d.recording = true
	fail := func(err error) error {
		d.recording = false
		return err
	}
	if fills > 0 {
		s.bindStorage(s.fillSet, s.fillBuf)
		s.bufferBarrier(s.fillBuf)
	}
	if inked {
		s.bindStorage(s.inkSet, s.inkBuf)
		s.bufferBarrier(s.inkBuf)
	}
	clear := clearColor{a: 1}
	area := rect2D{width: uint32(s.width), height: uint32(s.height)}
	pass := renderPassBegin{
		sType: structureRenderPassBegin, renderPass: s.drawPass, framebuffer: s.drawFrames[index],
		area: area, clearCount: 1, clears: &clear,
	}
	d.api.cmdBeginRenderPass(d.cmd, uintptr(unsafe.Pointer(&pass)), 0)
	view := viewport{width: float32(s.width), height: float32(s.height), maxDepth: 1}
	scissor := rect2D{width: uint32(s.width), height: uint32(s.height)}
	d.api.cmdSetViewport(d.cmd, 0, 1, uintptr(unsafe.Pointer(&view)))
	d.api.cmdSetScissor(d.cmd, 0, 1, uintptr(unsafe.Pointer(&scissor)))
	push := [3]uint32{mathFloatBits(float32(s.width)), mathFloatBits(float32(s.height)), uint32(s.swapRB)}
	if fills > 0 {
		d.api.cmdBindPipeline(d.cmd, bindPointGraphics, s.fillPipe)
		d.api.cmdBindSets(d.cmd, bindPointGraphics, s.fillLay, 0, 1, &s.fillSet, 0, nil)
		d.api.cmdPushConstants(d.cmd, s.fillLay, shaderStageVertex|shaderStageFragment, 0, 12, uintptr(unsafe.Pointer(&push[0])))
		d.api.cmdDraw(d.cmd, 4, uint32(fills), 0, 0)
	}
	if inked {
		inkPush := [3]uint32{uint32(s.width), uint32(s.height), uint32(s.swapRB)}
		d.api.cmdBindPipeline(d.cmd, bindPointGraphics, s.inkPipe)
		d.api.cmdBindSets(d.cmd, bindPointGraphics, s.inkLay, 0, 1, &s.inkSet, 0, nil)
		d.api.cmdPushConstants(d.cmd, s.inkLay, shaderStageVertex|shaderStageFragment, 0, 12, uintptr(unsafe.Pointer(&inkPush[0])))
		d.api.cmdDraw(d.cmd, 4, 1, 0, 0)
	}
	d.api.cmdEndRenderPass(d.cmd)
	s.layouts[index] = layoutPresent
	if err := check(d.api.endCommandBuffer(d.cmd)); err != nil {
		return fail(err)
	}
	d.recording = false
	submit := submitInfo{sType: structureSubmitInfo, commandBufferCount: 1, pCommandBuffers: &d.cmd}
	if err := check(d.api.queueSubmit(d.queue, 1, &submit, d.fence)); err != nil {
		return err
	}
	d.pending = true
	if err := check(d.api.waitForFences(d.dev, 1, &d.fence, 1, ^uint64(0))); err != nil {
		return err
	}
	d.pending = false
	return check(d.api.resetFences(d.dev, 1, &d.fence))
}

func (s *Screen) bindStorage(set uint64, buf *Buffer) {
	info := descriptorBufferInfo{buffer: buf.buf, rang: uint64(buf.size)}
	write := writeDescriptorSet{
		sType: structureWriteDescriptorSet, dstSet: set, dstBinding: 0, descriptorCount: 1,
		descriptorType: descriptorStorageBuffer, pBufferInfo: &info,
	}
	s.d.api.updateDescriptorSets(s.d.dev, 1, &write, 0, 0)
}

func (s *Screen) bufferBarrier(buf *Buffer) {
	bar := bufferBarrier{
		sType: structureBufferBarrier, srcAccess: accessHostWrite, dstAccess: accessShaderRead,
		srcQueue: ^uint32(0), dstQueue: ^uint32(0), buffer: buf.buf, size: uint64(buf.size),
	}
	s.d.api.cmdBarrier(s.d.cmd, pipeStageHost, pipeStageVert|pipeStageFrag, 0, 0, nil, 1, uintptr(unsafe.Pointer(&bar)), 0, 0)
}

func (s *Screen) destroyFrames() {
	if s.d == nil || s.d.dev == 0 || s.d.api.destroyFramebuffer == nil {
		s.drawFrames = nil
		return
	}
	for _, fb := range s.drawFrames {
		if fb != 0 {
			s.d.api.destroyFramebuffer(s.d.dev, fb, 0)
		}
	}
	s.drawFrames = nil
}

func (s *Screen) destroyPipes() {
	s.destroyPipeSlot(&s.fillPipe, &s.fillLay, &s.fillSetLay, &s.fillPool, &s.fillVert, &s.fillFrag)
	s.destroyPipeSlot(&s.inkPipe, &s.inkLay, &s.inkSetLay, &s.inkPool, &s.inkVert, &s.inkFrag)
}

func (s *Screen) destroyPipeSlot(pipe, lay, setLay, pool, vert, frag *uint64) {
	d := s.d
	if d == nil || d.dev == 0 {
		return
	}
	if *pool != 0 {
		d.api.destroyDescriptorPool(d.dev, *pool, 0)
	}
	if *pipe != 0 {
		d.api.destroyPipeline(d.dev, *pipe, 0)
	}
	if *lay != 0 {
		d.api.destroyPipelineLayout(d.dev, *lay, 0)
	}
	if *setLay != 0 {
		d.api.destroySetLayout(d.dev, *setLay, 0)
	}
	if *vert != 0 {
		d.api.destroyShaderModule(d.dev, *vert, 0)
	}
	if *frag != 0 {
		d.api.destroyShaderModule(d.dev, *frag, 0)
	}
	*pipe, *lay, *setLay, *pool, *vert, *frag = 0, 0, 0, 0, 0, 0
}

func (s *Screen) destroyDraw() {
	s.destroyFrames()
	s.destroyPipes()
	if s.d != nil && s.d.dev != 0 && s.drawPass != 0 && s.d.api.destroyRenderPass != nil {
		s.d.api.destroyRenderPass(s.d.dev, s.drawPass, 0)
	}
	s.drawPass = 0
	if s.fillBuf != nil {
		_ = s.fillBuf.Close()
		s.fillBuf = nil
	}
	if s.inkBuf != nil {
		_ = s.inkBuf.Close()
		s.inkBuf = nil
	}
}

func mathFloatBits(v float32) uint32 {
	return *(*uint32)(unsafe.Pointer(&v))
}
