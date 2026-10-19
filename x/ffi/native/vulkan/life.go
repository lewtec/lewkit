package vulkan

import (
	"sync"
	"unsafe"
)

// vkLife is the device handle passed to a cleanup. It does not point at
// *Device or a child. Each child holds one ref, and the owner's cleanup
// holds one. The last release destroys the device after the child that
// dropped that ref has already destroyed itself.
type vkLife struct {
	mu    sync.Mutex
	refs  int
	owner bool
	api   api
	inst  uintptr
	dev   uintptr
	fence uint64
	pool  uint64
}

func (v *vkLife) hold() {
	if v == nil {
		return
	}
	v.mu.Lock()
	if v.refs > 0 {
		v.refs++
	}
	v.mu.Unlock()
}

func (v *vkLife) releaseOwner() {
	if v == nil {
		return
	}
	v.mu.Lock()
	if v.owner {
		v.mu.Unlock()
		return
	}
	v.owner = true
	v.decLocked()
}

// decLocked consumes one ref. The caller holds v.mu.
func (v *vkLife) decLocked() {
	if v.refs == 0 {
		v.mu.Unlock()
		return
	}
	v.refs--
	if v.refs > 0 {
		v.mu.Unlock()
		return
	}
	api, dev, inst := v.api, v.dev, v.inst
	fence, pool := v.fence, v.pool
	v.dev, v.inst = 0, 0
	v.fence, v.pool = 0, 0
	v.mu.Unlock()
	destroyVK(api, dev, inst, fence, pool)
}

func destroyVK(api api, dev, inst uintptr, fence, pool uint64) {
	if dev != 0 {
		if fence != 0 && api.destroyFence != nil {
			api.destroyFence(dev, fence, 0)
		}
		if pool != 0 && api.destroyCommandPool != nil {
			api.destroyCommandPool(dev, pool, 0)
		}
		if api.destroyDevice != nil {
			api.destroyDevice(dev, 0)
		}
	}
	if inst != 0 && api.destroyInstance != nil {
		api.destroyInstance(inst, 0)
	}
}

// vkBuf is one buffer's native handles.
type vkBuf struct {
	mu   sync.Mutex
	done bool
	life *vkLife
	buf  uint64
	mem  uint64
	ptr  unsafe.Pointer
}

func (b *vkBuf) release() {
	if b == nil {
		return
	}
	b.mu.Lock()
	if b.done {
		b.mu.Unlock()
		return
	}
	b.done = true
	life, buf, mem, ptr := b.life, b.buf, b.mem, b.ptr
	b.mu.Unlock()
	if life == nil {
		return
	}
	life.mu.Lock()
	if life.dev != 0 && buf != 0 {
		if ptr != nil && life.api.unmapMemory != nil {
			life.api.unmapMemory(life.dev, mem)
		}
		if life.api.destroyBuffer != nil {
			life.api.destroyBuffer(life.dev, buf, 0)
		}
		if life.api.freeMemory != nil {
			life.api.freeMemory(life.dev, mem, 0)
		}
	}
	life.decLocked()
}

// vkPipe is one shader's native handles.
type vkPipe struct {
	mu             sync.Mutex
	done           bool
	life           *vkLife
	module         uint64
	setLayout      uint64
	pipelineLayout uint64
	pipeline       uint64
	descriptorPool uint64
}

func (s *vkPipe) setPool(pool uint64) {
	if s == nil || pool == 0 {
		return
	}
	s.mu.Lock()
	s.descriptorPool = pool
	s.mu.Unlock()
}

func (s *vkPipe) release() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.done {
		s.mu.Unlock()
		return
	}
	s.done = true
	life := s.life
	pipeline, layout, set, mod := s.pipeline, s.pipelineLayout, s.setLayout, s.module
	pool := s.descriptorPool
	s.mu.Unlock()
	if life == nil {
		return
	}
	life.mu.Lock()
	if life.dev != 0 && pipeline != 0 {
		api := life.api
		dev := life.dev
		if pool != 0 && api.destroyDescriptorPool != nil {
			api.destroyDescriptorPool(dev, pool, 0)
		}
		if api.destroyPipeline != nil {
			api.destroyPipeline(dev, pipeline, 0)
		}
		if api.destroyPipelineLayout != nil {
			api.destroyPipelineLayout(dev, layout, 0)
		}
		if api.destroySetLayout != nil {
			api.destroySetLayout(dev, set, 0)
		}
		if api.destroyShaderModule != nil {
			api.destroyShaderModule(dev, mod, 0)
		}
	}
	life.decLocked()
}
