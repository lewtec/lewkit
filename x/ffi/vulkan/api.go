package vulkan

import (
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi"
)

const (
	structureApplicationInfo               = 0
	structureInstanceCreateInfo            = 1
	structureDeviceQueueCreateInfo         = 2
	structureDeviceCreateInfo              = 3
	structureSubmitInfo                    = 4
	structureFenceCreateInfo               = 8
	structureMemoryAllocateInfo            = 5
	structureMappedMemoryRange             = 6
	structureBufferCreateInfo              = 12
	structureShaderModuleCreateInfo        = 16
	structurePipelineShaderStageCreateInfo = 18
	structureComputePipelineCreateInfo     = 29
	structurePipelineLayoutCreateInfo      = 30
	structureDescriptorSetLayoutCreateInfo = 32
	structureDescriptorPoolCreateInfo      = 33
	structureDescriptorSetAllocateInfo     = 34
	structureWriteDescriptorSet            = 35
	structureCommandPoolCreateInfo         = 39
	structureCommandBufferAllocateInfo     = 40
	structureCommandBufferBeginInfo        = 42
	structureMemoryBarrier                 = 46

	queueComputeBit = 2

	memoryDeviceLocal  = 1
	memoryHostVisible  = 2
	memoryHostCoherent = 4

	bufferUsageStorage     = 32
	bufferUsageTransferDst = 2
	bufferUsageTransferSrc = 1
	stageTransfer          = 0x00001000
	accessTransferWrite    = 0x00001000
	accessTransferRead     = 0x00000800

	descriptorStorageBuffer = 7
	shaderStageCompute      = 32
	bindPointCompute        = 1

	commandPoolTransient = 1
	commandPoolReset     = 2
	commandOneTimeSubmit = 1

	apiVersion11 = 1<<22 | 1<<12

	instanceEnumeratePortability = 1
	extPortabilityEnum           = "VK_KHR_portability_enumeration"
	extPortabilitySubset         = "VK_KHR_portability_subset"

	accessHostWrite   = 0x00004000
	accessHostRead    = 0x00002000
	accessShaderRead  = 0x00000020
	accessShaderWrite = 0x00000040
	stageHost         = 0x00004000
	stageCompute      = 0x00000800
	wholeSize         = ^uint64(0)
)

type applicationInfo struct {
	sType              int32
	pNext              uintptr
	pApplicationName   *byte
	applicationVersion uint32
	pEngineName        *byte
	engineVersion      uint32
	apiVersion         uint32
}

type instanceCreateInfo struct {
	sType                   int32
	pNext                   uintptr
	flags                   uint32
	pApplicationInfo        *applicationInfo
	enabledLayerCount       uint32
	ppEnabledLayerNames     **byte
	enabledExtensionCount   uint32
	ppEnabledExtensionNames **byte
}

type deviceQueueCreateInfo struct {
	sType            int32
	pNext            uintptr
	flags            uint32
	queueFamilyIndex uint32
	queueCount       uint32
	pQueuePriorities *float32
}

type deviceCreateInfo struct {
	sType                   int32
	pNext                   uintptr
	flags                   uint32
	queueCreateInfoCount    uint32
	pQueueCreateInfos       *deviceQueueCreateInfo
	enabledLayerCount       uint32
	ppEnabledLayerNames     **byte
	enabledExtensionCount   uint32
	ppEnabledExtensionNames **byte
	pEnabledFeatures        uintptr
}

type queueFamilyProperties struct {
	queueFlags                  uint32
	queueCount                  uint32
	timestampValidBits          uint32
	minImageTransferGranularity [3]uint32
}

type memoryType struct {
	propertyFlags uint32
	heapIndex     uint32
}

type memoryHeap struct {
	size  uint64
	flags uint32
	_     uint32
}

type physicalDeviceMemoryProperties struct {
	memoryTypeCount uint32
	memoryTypes     [32]memoryType
	memoryHeapCount uint32
	memoryHeaps     [16]memoryHeap
}

type memoryRequirements struct {
	size           uint64
	alignment      uint64
	memoryTypeBits uint32
	_              uint32
}

type bufferCreateInfo struct {
	sType                 int32
	pNext                 uintptr
	flags                 uint32
	size                  uint64
	usage                 uint32
	sharingMode           int32
	queueFamilyIndexCount uint32
	pQueueFamilyIndices   *uint32
}

type memoryAllocateInfo struct {
	sType           int32
	pNext           uintptr
	allocationSize  uint64
	memoryTypeIndex uint32
}

type shaderModuleCreateInfo struct {
	sType    int32
	pNext    uintptr
	flags    uint32
	codeSize uintptr
	pCode    *uint32
}

type descriptorSetLayoutBinding struct {
	binding            uint32
	descriptorType     int32
	descriptorCount    uint32
	stageFlags         uint32
	pImmutableSamplers uintptr
}

type descriptorSetLayoutCreateInfo struct {
	sType        int32
	pNext        uintptr
	flags        uint32
	bindingCount uint32
	pBindings    *descriptorSetLayoutBinding
}

type pipelineLayoutCreateInfo struct {
	sType                  int32
	pNext                  uintptr
	flags                  uint32
	setLayoutCount         uint32
	pSetLayouts            *uint64
	pushConstantRangeCount uint32
	pPushConstantRanges    *pushConstantRange
}

type pushConstantRange struct {
	stageFlags uint32
	offset     uint32
	size       uint32
}

type specializationMapEntry struct {
	constantID uint32
	offset     uint32
	size       uintptr
}

type specializationInfo struct {
	mapEntryCount uint32
	pMapEntries   *specializationMapEntry
	dataSize      uintptr
	pData         *byte
}

type bufferCopy struct {
	srcOffset uint64
	dstOffset uint64
	size      uint64
}

type pipelineShaderStageCreateInfo struct {
	sType               int32
	pNext               uintptr
	flags               uint32
	stage               uint32
	module              uint64
	pName               *byte
	pSpecializationInfo uintptr
}

type computePipelineCreateInfo struct {
	sType              int32
	pNext              uintptr
	flags              uint32
	stage              pipelineShaderStageCreateInfo
	layout             uint64
	basePipelineHandle uint64
	basePipelineIndex  int32
}

type commandPoolCreateInfo struct {
	sType            int32
	pNext            uintptr
	flags            uint32
	queueFamilyIndex uint32
}

type commandBufferAllocateInfo struct {
	sType              int32
	pNext              uintptr
	commandPool        uint64
	level              int32
	commandBufferCount uint32
}

type commandBufferBeginInfo struct {
	sType            int32
	pNext            uintptr
	flags            uint32
	pInheritanceInfo uintptr
}

type descriptorPoolSize struct {
	typ             int32
	descriptorCount uint32
}

type descriptorPoolCreateInfo struct {
	sType         int32
	pNext         uintptr
	flags         uint32
	maxSets       uint32
	poolSizeCount uint32
	pPoolSizes    *descriptorPoolSize
}

type descriptorSetAllocateInfo struct {
	sType              int32
	pNext              uintptr
	descriptorPool     uint64
	descriptorSetCount uint32
	pSetLayouts        *uint64
}

type descriptorBufferInfo struct {
	buffer uint64
	offset uint64
	rang   uint64
}

type writeDescriptorSet struct {
	sType            int32
	pNext            uintptr
	dstSet           uint64
	dstBinding       uint32
	dstArrayElement  uint32
	descriptorCount  uint32
	descriptorType   int32
	pImageInfo       uintptr
	pBufferInfo      *descriptorBufferInfo
	pTexelBufferView uintptr
}

type mappedMemoryRange struct {
	sType  int32
	pNext  uintptr
	memory uint64
	offset uint64
	size   uint64
}

type memoryBarrier struct {
	sType         int32
	pNext         uintptr
	srcAccessMask uint32
	dstAccessMask uint32
}

type submitInfo struct {
	sType                int32
	pNext                uintptr
	waitSemaphoreCount   uint32
	pWaitSemaphores      uintptr
	pWaitDstStageMask    uintptr
	commandBufferCount   uint32
	pCommandBuffers      *uintptr
	signalSemaphoreCount uint32
	pSignalSemaphores    uintptr
}

type fenceCreateInfo struct {
	sType int32
	pNext uintptr
	flags uint32
}

type extensionProperties struct {
	name        [256]byte
	specVersion uint32
}

type api struct {
	getInstanceProcAddr func(instance uintptr, name string) uintptr
	getDeviceProcAddr   func(device uintptr, name string) uintptr

	enumerateInstanceExt func(layer *byte, count *uint32, props *extensionProperties) int32
	enumerateDeviceExt   func(phys uintptr, layer *byte, count *uint32, props *extensionProperties) int32

	createInstance    func(info *instanceCreateInfo, alloc uintptr, instance *uintptr) int32
	destroyInstance   func(instance uintptr, alloc uintptr)
	enumeratePhysical func(instance uintptr, count *uint32, devices *uintptr) int32
	getQueueFamilies  func(phys uintptr, count *uint32, props *queueFamilyProperties)
	getMemoryProps    func(phys uintptr, props *physicalDeviceMemoryProperties)
	getPhysProps      func(phys uintptr, props *byte)
	createDevice      func(phys uintptr, info *deviceCreateInfo, alloc uintptr, device *uintptr) int32
	destroyDevice     func(device uintptr, alloc uintptr)
	getDeviceQueue    func(device uintptr, family, index uint32, queue *uintptr)

	createCommandPool      func(device uintptr, info *commandPoolCreateInfo, alloc uintptr, pool *uint64) int32
	destroyCommandPool     func(device uintptr, pool uint64, alloc uintptr)
	resetCommandPool       func(device uintptr, pool uint64, flags uint32) int32
	resetCommandBuffer     func(cmd uintptr, flags uint32) int32
	allocateCmdBuffers     func(device uintptr, info *commandBufferAllocateInfo, bufs *uintptr) int32
	beginCommandBuffer     func(cmd uintptr, info *commandBufferBeginInfo) int32
	endCommandBuffer       func(cmd uintptr) int32
	createShaderModule     func(device uintptr, info *shaderModuleCreateInfo, alloc uintptr, module *uint64) int32
	destroyShaderModule    func(device uintptr, module uint64, alloc uintptr)
	createSetLayout        func(device uintptr, info *descriptorSetLayoutCreateInfo, alloc uintptr, layout *uint64) int32
	destroySetLayout       func(device uintptr, layout uint64, alloc uintptr)
	createPipelineLayout   func(device uintptr, info *pipelineLayoutCreateInfo, alloc uintptr, layout *uint64) int32
	destroyPipelineLayout  func(device uintptr, layout uint64, alloc uintptr)
	createComputePipes     func(device uintptr, cache uint64, count uint32, infos *computePipelineCreateInfo, alloc uintptr, pipes *uint64) int32
	destroyPipeline        func(device uintptr, pipe uint64, alloc uintptr)
	createBuffer           func(device uintptr, info *bufferCreateInfo, alloc uintptr, buffer *uint64) int32
	destroyBuffer          func(device uintptr, buffer uint64, alloc uintptr)
	getBufferReqs          func(device uintptr, buffer uint64, reqs *memoryRequirements)
	allocateMemory         func(device uintptr, info *memoryAllocateInfo, alloc uintptr, memory *uint64) int32
	freeMemory             func(device uintptr, memory uint64, alloc uintptr)
	bindBufferMemory       func(device uintptr, buffer, memory, offset uint64) int32
	mapMemory              func(device uintptr, memory, offset, size uint64, flags uint32, data *unsafe.Pointer) int32
	unmapMemory            func(device uintptr, memory uint64)
	createDescriptorPool   func(device uintptr, info *descriptorPoolCreateInfo, alloc uintptr, pool *uint64) int32
	destroyDescriptorPool  func(device uintptr, pool uint64, alloc uintptr)
	allocateDescriptorSets func(device uintptr, info *descriptorSetAllocateInfo, sets *uint64) int32
	updateDescriptorSets   func(device uintptr, writeCount uint32, writes *writeDescriptorSet, copyCount uint32, copies uintptr)
	cmdBindPipeline        func(cmd uintptr, bindPoint uint32, pipeline uint64)
	cmdBindSets            func(cmd uintptr, bindPoint uint32, layout uint64, firstSet, setCount uint32, sets *uint64, dynCount uint32, dyn *uint32)
	cmdDispatch            func(cmd uintptr, x, y, z uint32)
	cmdCopyBuffer          func(cmd uintptr, src, dst uint64, count uint32, regions *bufferCopy)
	cmdPushConstants       func(cmd uintptr, layout uint64, stages, offset, size uint32, values uintptr)
	cmdUpdateBuffer        func(cmd uintptr, buffer, offset, size uint64, data uintptr)
	cmdBarrier             func(cmd uintptr, srcStage, dstStage, flags uint32, memCount uint32, mem *memoryBarrier, bufCount uint32, bufs uintptr, imgCount uint32, imgs uintptr)
	flushMapped            func(device uintptr, count uint32, ranges *mappedMemoryRange) int32
	invalidateMapped       func(device uintptr, count uint32, ranges *mappedMemoryRange) int32
	queueSubmit            func(queue uintptr, count uint32, submits *submitInfo, fence uint64) int32
	queueWaitIdle          func(queue uintptr) int32
	createFence            func(device uintptr, info *fenceCreateInfo, alloc uintptr, fence *uint64) int32
	destroyFence           func(device uintptr, fence uint64, alloc uintptr)
	resetFences            func(device uintptr, count uint32, fences *uint64) int32
	waitForFences          func(device uintptr, count uint32, fences *uint64, waitAll uint32, timeout uint64) int32
}

func libNames() []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"vulkan-1.dll"}
	case "darwin":
		names := []string{"libvulkan.1.dylib", "libvulkan.dylib"}
		// Homebrew and the LunarG SDK are not on dyld's default path.
		for _, dir := range darwinLibDirs() {
			names = append(names,
				dir+"/libvulkan.1.dylib",
				dir+"/libvulkan.dylib",
				dir+"/libMoltenVK.dylib",
			)
		}
		return names
	default:
		return []string{"libvulkan.so.1", "libvulkan.so"}
	}
}

func darwinLibDirs() []string {
	var dirs []string
	seen := map[string]bool{}
	add := func(d string) {
		if d == "" || seen[d] {
			return
		}
		seen[d] = true
		dirs = append(dirs, d)
	}
	if p := os.Getenv("HOMEBREW_PREFIX"); p != "" {
		add(p + "/lib")
	}
	if sdk := os.Getenv("VULKAN_SDK"); sdk != "" {
		add(sdk + "/lib")
	}
	add("/opt/homebrew/lib")
	add("/usr/local/lib")
	return dirs
}

func openLib() (uintptr, error) {
	var last error
	for _, name := range libNames() {
		lib, err := ffi.Open(name, ffi.Lazy)
		if err == nil {
			return lib, nil
		}
		last = err
	}
	if last == nil {
		return 0, ErrUnavailable
	}
	return 0, fmt.Errorf("%w: %v", ErrUnavailable, last)
}

func (a *api) bind(get func(uintptr, string) uintptr, handle uintptr, name string, dst any) error {
	addr := get(handle, name)
	if addr == 0 {
		return fmt.Errorf("%w: missing %s", ErrUnavailable, name)
	}
	ffi.Register(dst, addr)
	return nil
}

func (a *api) loadLoader(lib uintptr) error {
	ffi.Func(lib, "vkGetInstanceProcAddr", &a.getInstanceProcAddr)
	if a.getInstanceProcAddr == nil {
		return fmt.Errorf("%w: vkGetInstanceProcAddr", ErrUnavailable)
	}
	if err := a.bind(a.getInstanceProcAddr, 0, "vkCreateInstance", &a.createInstance); err != nil {
		return err
	}
	return a.bind(a.getInstanceProcAddr, 0, "vkEnumerateInstanceExtensionProperties", &a.enumerateInstanceExt)
}

func (a *api) loadInstance(inst uintptr) error {
	type pair struct {
		name string
		dst  any
	}
	for _, p := range []pair{
		{"vkDestroyInstance", &a.destroyInstance},
		{"vkEnumeratePhysicalDevices", &a.enumeratePhysical},
		{"vkEnumerateDeviceExtensionProperties", &a.enumerateDeviceExt},
		{"vkGetPhysicalDeviceQueueFamilyProperties", &a.getQueueFamilies},
		{"vkGetPhysicalDeviceMemoryProperties", &a.getMemoryProps},
		{"vkGetPhysicalDeviceProperties", &a.getPhysProps},
		{"vkCreateDevice", &a.createDevice},
		{"vkGetDeviceQueue", &a.getDeviceQueue},
		{"vkDestroyDevice", &a.destroyDevice},
		{"vkCreateCommandPool", &a.createCommandPool},
		{"vkDestroyCommandPool", &a.destroyCommandPool},
		{"vkResetCommandPool", &a.resetCommandPool},
		{"vkResetCommandBuffer", &a.resetCommandBuffer},
		{"vkAllocateCommandBuffers", &a.allocateCmdBuffers},
		{"vkBeginCommandBuffer", &a.beginCommandBuffer},
		{"vkEndCommandBuffer", &a.endCommandBuffer},
		{"vkCreateShaderModule", &a.createShaderModule},
		{"vkDestroyShaderModule", &a.destroyShaderModule},
		{"vkCreateDescriptorSetLayout", &a.createSetLayout},
		{"vkDestroyDescriptorSetLayout", &a.destroySetLayout},
		{"vkCreatePipelineLayout", &a.createPipelineLayout},
		{"vkDestroyPipelineLayout", &a.destroyPipelineLayout},
		{"vkCreateComputePipelines", &a.createComputePipes},
		{"vkDestroyPipeline", &a.destroyPipeline},
		{"vkCreateBuffer", &a.createBuffer},
		{"vkDestroyBuffer", &a.destroyBuffer},
		{"vkGetBufferMemoryRequirements", &a.getBufferReqs},
		{"vkAllocateMemory", &a.allocateMemory},
		{"vkFreeMemory", &a.freeMemory},
		{"vkBindBufferMemory", &a.bindBufferMemory},
		{"vkMapMemory", &a.mapMemory},
		{"vkUnmapMemory", &a.unmapMemory},
		{"vkCreateDescriptorPool", &a.createDescriptorPool},
		{"vkDestroyDescriptorPool", &a.destroyDescriptorPool},
		{"vkAllocateDescriptorSets", &a.allocateDescriptorSets},
		{"vkUpdateDescriptorSets", &a.updateDescriptorSets},
		{"vkCmdBindPipeline", &a.cmdBindPipeline},
		{"vkCmdBindDescriptorSets", &a.cmdBindSets},
		{"vkCmdDispatch", &a.cmdDispatch},
		{"vkCmdCopyBuffer", &a.cmdCopyBuffer},
		{"vkCmdPushConstants", &a.cmdPushConstants},
		{"vkCmdUpdateBuffer", &a.cmdUpdateBuffer},
		{"vkCmdPipelineBarrier", &a.cmdBarrier},
		{"vkFlushMappedMemoryRanges", &a.flushMapped},
		{"vkInvalidateMappedMemoryRanges", &a.invalidateMapped},
		{"vkQueueSubmit", &a.queueSubmit},
		{"vkQueueWaitIdle", &a.queueWaitIdle},
		{"vkCreateFence", &a.createFence},
		{"vkDestroyFence", &a.destroyFence},
		{"vkResetFences", &a.resetFences},
		{"vkWaitForFences", &a.waitForFences},
	} {
		if err := a.bind(a.getInstanceProcAddr, inst, p.name, p.dst); err != nil {
			return err
		}
	}
	return a.bind(a.getInstanceProcAddr, inst, "vkGetDeviceProcAddr", &a.getDeviceProcAddr)
}

func (a *api) loadDevice(dev uintptr) error {
	if a.getDeviceProcAddr == nil {
		return nil
	}
	type pair struct {
		name string
		dst  any
	}
	for _, p := range []pair{
		{"vkCreateComputePipelines", &a.createComputePipes},
		{"vkCmdBindDescriptorSets", &a.cmdBindSets},
		{"vkCmdBindPipeline", &a.cmdBindPipeline},
		{"vkCmdDispatch", &a.cmdDispatch},
		{"vkCmdCopyBuffer", &a.cmdCopyBuffer},
		{"vkCmdPushConstants", &a.cmdPushConstants},
		{"vkCmdUpdateBuffer", &a.cmdUpdateBuffer},
		{"vkCmdPipelineBarrier", &a.cmdBarrier},
		{"vkFlushMappedMemoryRanges", &a.flushMapped},
		{"vkInvalidateMappedMemoryRanges", &a.invalidateMapped},
		{"vkQueueSubmit", &a.queueSubmit},
		{"vkQueueWaitIdle", &a.queueWaitIdle},
		{"vkCreateFence", &a.createFence},
		{"vkDestroyFence", &a.destroyFence},
		{"vkResetFences", &a.resetFences},
		{"vkWaitForFences", &a.waitForFences},
		{"vkUpdateDescriptorSets", &a.updateDescriptorSets},
	} {
		if err := a.bind(a.getDeviceProcAddr, dev, p.name, p.dst); err != nil {
			return err
		}
	}
	return nil
}

func cstr(s string) *byte {
	b := append([]byte(s), 0)
	return unsafe.SliceData(b)
}

func cstring(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func check(r int32) error {
	if r == 0 {
		return nil
	}
	return Result(r)
}

func hasExt(exts []string, name string) bool {
	for _, e := range exts {
		if e == name {
			return true
		}
	}
	return false
}

func cStrings(names []string) (**byte, []*byte) {
	if len(names) == 0 {
		return nil, nil
	}
	ptrs := make([]*byte, len(names))
	for i, n := range names {
		ptrs[i] = cstr(n)
	}
	return &ptrs[0], ptrs
}

func (a *api) instanceExts() []string {
	if a.enumerateInstanceExt == nil {
		return nil
	}
	var n uint32
	if check(a.enumerateInstanceExt(nil, &n, nil)) != nil || n == 0 {
		return nil
	}
	props := make([]extensionProperties, n)
	if check(a.enumerateInstanceExt(nil, &n, &props[0])) != nil {
		return nil
	}
	out := make([]string, 0, n)
	for _, p := range props[:n] {
		out = append(out, cstring(p.name[:]))
	}
	return out
}

func (a *api) deviceExts(phys uintptr) []string {
	if a.enumerateDeviceExt == nil {
		return nil
	}
	var n uint32
	if check(a.enumerateDeviceExt(phys, nil, &n, nil)) != nil || n == 0 {
		return nil
	}
	props := make([]extensionProperties, n)
	if check(a.enumerateDeviceExt(phys, nil, &n, &props[0])) != nil {
		return nil
	}
	out := make([]string, 0, n)
	for _, p := range props[:n] {
		out = append(out, cstring(p.name[:]))
	}
	return out
}
