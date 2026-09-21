package vulkan

import (
	"errors"
	"fmt"
	"unsafe"
)

// Cmd records copies and dispatches, then submits them as one batch.
type Cmd struct {
	d     *Device
	bound *Shader
	used  []*Buffer
	pools []uint64
}

// Begin starts recording. One command buffer is in flight at a time.
func (d *Device) Begin() (*Cmd, error) {
	if err := d.live(); err != nil {
		return nil, err
	}
	if d.recording || d.pending {
		return nil, ErrBusy
	}
	if err := check(d.api.resetCommandBuffer(d.cmd, 0)); err != nil {
		return nil, fmt.Errorf("reset command buffer: %w", err)
	}
	begin := commandBufferBeginInfo{
		sType: structureCommandBufferBeginInfo,
	}
	if err := check(d.api.beginCommandBuffer(d.cmd, &begin)); err != nil {
		return nil, fmt.Errorf("begin command buffer: %w", err)
	}
	d.recording = true
	c := &d.recorded
	c.d = d
	c.bound = nil
	c.used = c.used[:0]
	c.pools = c.pools[:0]
	return c, nil
}

func (c *Cmd) mustRecord() error {
	if c == nil || c.d == nil || !c.d.recording || c.d.pending {
		return ErrBusy
	}
	return c.d.live()
}

func (c *Cmd) track(bufs ...*Buffer) {
	for _, b := range bufs {
		if b != nil {
			c.used = append(c.used, b)
		}
	}
}

// Bind sets the compute pipeline and storage buffers at set 0.
func (c *Cmd) Bind(s *Shader, bufs ...*Buffer) error {
	if err := c.mustRecord(); err != nil {
		return err
	}
	d := c.d
	if s == nil || s.pipeline == 0 || s.d != d {
		return ErrShader
	}
	if len(bufs) != s.bindings {
		return fmt.Errorf("%w: want %d buffers, got %d", ErrShader, s.bindings, len(bufs))
	}
	for _, b := range bufs {
		if b == nil || b.buf == 0 || b.d != d {
			return ErrClosed
		}
	}
	if err := s.ensureDescriptors(); err != nil {
		return err
	}
	count := len(bufs)
	unchanged := len(s.boundBuffers) == count
	if unchanged {
		for i, b := range bufs {
			if s.boundBuffers[i] != b.buf || s.boundLengths[i] != uint64(b.size) {
				unchanged = false
				break
			}
		}
	}
	if !unchanged {
		if cap(s.bufferInfos) < count {
			s.bufferInfos = make([]descriptorBufferInfo, count)
			s.writes = make([]writeDescriptorSet, count)
			s.boundBuffers = make([]uint64, count)
			s.boundLengths = make([]uint64, count)
		} else {
			s.bufferInfos = s.bufferInfos[:count]
			s.writes = s.writes[:count]
			s.boundBuffers = s.boundBuffers[:count]
			s.boundLengths = s.boundLengths[:count]
		}
		for i, b := range bufs {
			s.bufferInfos[i] = descriptorBufferInfo{buffer: b.buf, rang: uint64(b.size)}
			s.writes[i] = writeDescriptorSet{
				sType:           structureWriteDescriptorSet,
				dstSet:          s.descriptorSet,
				dstBinding:      uint32(i),
				descriptorCount: 1,
				descriptorType:  descriptorStorageBuffer,
				pBufferInfo:     &s.bufferInfos[i],
			}
			s.boundBuffers[i] = b.buf
			s.boundLengths[i] = uint64(b.size)
		}
		if count > 0 {
			d.api.updateDescriptorSets(d.dev, uint32(count), &s.writes[0], 0, 0)
		}
	}
	for _, b := range bufs {
		if err := d.flush(b); err != nil {
			return err
		}
	}
	d.api.cmdBindPipeline(d.cmd, bindPointCompute, s.pipeline)
	d.api.cmdBindSets(d.cmd, bindPointCompute, s.pipelineLayout, 0, 1, &s.descriptorSet, 0, nil)
	host := false
	for _, b := range bufs {
		if b.ptr != nil {
			host = true
			break
		}
	}
	if host {
		bar := memoryBarrier{
			sType:         structureMemoryBarrier,
			srcAccessMask: accessHostWrite,
			dstAccessMask: accessShaderRead,
		}
		d.api.cmdBarrier(d.cmd, stageHost, stageCompute, 0, 1, &bar, 0, 0, 0, 0)
	}
	c.bound = s
	c.track(bufs...)
	return nil
}

func (s *Shader) ensureDescriptors() error {
	if s == nil || s.d == nil || s.pipeline == 0 {
		return ErrShader
	}
	if s.descriptorPool != 0 {
		return nil
	}
	d := s.d
	poolSize := descriptorPoolSize{
		typ:             descriptorStorageBuffer,
		descriptorCount: uint32(max(s.bindings, 1)),
	}
	poolInfo := descriptorPoolCreateInfo{
		sType:         structureDescriptorPoolCreateInfo,
		maxSets:       1,
		poolSizeCount: 1,
		pPoolSizes:    &poolSize,
	}
	var pool uint64
	if err := check(d.api.createDescriptorPool(d.dev, &poolInfo, 0, &pool)); err != nil {
		return fmt.Errorf("descriptor pool: %w", err)
	}
	alloc := descriptorSetAllocateInfo{
		sType:              structureDescriptorSetAllocateInfo,
		descriptorPool:     pool,
		descriptorSetCount: 1,
		pSetLayouts:        &s.setLayout,
	}
	var set uint64
	if err := check(d.api.allocateDescriptorSets(d.dev, &alloc, &set)); err != nil {
		d.api.destroyDescriptorPool(d.dev, pool, 0)
		return fmt.Errorf("descriptor set: %w", err)
	}
	s.descriptorPool, s.descriptorSet = pool, set
	return nil
}

// Push writes push constants for the bound shader, offset 0.
func (c *Cmd) Push(data []byte) error {
	if err := c.mustRecord(); err != nil {
		return err
	}
	if c.bound == nil || c.bound.pushBytes == 0 {
		return ErrPush
	}
	if len(data) == 0 || len(data) > c.bound.pushBytes || len(data)%4 != 0 {
		return ErrPush
	}
	c.d.api.cmdPushConstants(c.d.cmd, c.bound.pipelineLayout, shaderStageCompute, 0, uint32(len(data)), uintptr(unsafe.Pointer(unsafe.SliceData(data))))
	return nil
}

// Dispatch records a compute dispatch. The shader must already be bound.
func (c *Cmd) Dispatch(x, y, z uint32) error {
	if err := c.mustRecord(); err != nil {
		return err
	}
	if c.bound == nil {
		return ErrShader
	}
	c.d.api.cmdDispatch(c.d.cmd, x, y, z)
	return nil
}

// Barrier makes prior copies and dispatches visible to later ones.
func (c *Cmd) Barrier() error {
	if err := c.mustRecord(); err != nil {
		return err
	}
	bar := memoryBarrier{
		sType:         structureMemoryBarrier,
		srcAccessMask: accessShaderWrite | accessTransferWrite,
		dstAccessMask: accessShaderRead | accessShaderWrite | accessTransferRead,
	}
	c.d.api.cmdBarrier(c.d.cmd, stageCompute|stageTransfer, stageCompute|stageTransfer, 0, 1, &bar, 0, 0, 0, 0)
	return nil
}

// Copy records a buffer copy of src.Len() bytes. dst must be at least that large.
func (c *Cmd) Copy(dst, src *Buffer) error {
	if err := c.mustRecord(); err != nil {
		return err
	}
	d := c.d
	if src == nil || dst == nil || src.buf == 0 || dst.buf == 0 || src.d != d || dst.d != d {
		return ErrClosed
	}
	if src.size < 1 || src.size > dst.size {
		return ErrSize
	}
	if err := d.flush(src); err != nil {
		return err
	}
	region := bufferCopy{size: uint64(src.size)}
	d.api.cmdCopyBuffer(d.cmd, src.buf, dst.buf, 1, &region)
	c.track(src, dst)
	return nil
}

// Submit ends recording and queues the work on a fence. Call Wait before Begin again.
func (c *Cmd) Submit() error {
	if err := c.mustRecord(); err != nil {
		return err
	}
	d := c.d
	host := false
	for _, b := range c.used {
		if b.ptr != nil {
			host = true
			break
		}
	}
	if host {
		bar := memoryBarrier{
			sType:         structureMemoryBarrier,
			srcAccessMask: accessShaderWrite | accessTransferWrite,
			dstAccessMask: accessHostRead,
		}
		d.api.cmdBarrier(d.cmd, stageCompute|stageTransfer, stageHost, 0, 1, &bar, 0, 0, 0, 0)
	}
	if err := check(d.api.endCommandBuffer(d.cmd)); err != nil {
		d.recording = false
		return fmt.Errorf("end command buffer: %w", err)
	}
	submit := submitInfo{
		sType:              structureSubmitInfo,
		commandBufferCount: 1,
		pCommandBuffers:    &d.cmd,
	}
	if err := check(d.api.queueSubmit(d.queue, 1, &submit, d.fence)); err != nil {
		d.recording = false
		return fmt.Errorf("queue submit: %w", err)
	}
	d.recording = false
	d.pending = true
	return nil
}

// Wait waits for the submit fence and invalidates mapped buffers.
func (c *Cmd) Wait() error {
	if c == nil || c.d == nil {
		return ErrClosed
	}
	d := c.d
	if d.recording {
		return ErrBusy
	}
	if !d.pending {
		c.release()
		return nil
	}
	if d.fence != 0 && d.api.waitForFences != nil {
		if err := check(d.api.waitForFences(d.dev, 1, &d.fence, 1, ^uint64(0))); err != nil {
			return fmt.Errorf("fence wait: %w", err)
		}
		if err := check(d.api.resetFences(d.dev, 1, &d.fence)); err != nil {
			return fmt.Errorf("reset fence: %w", err)
		}
	} else if err := check(d.api.queueWaitIdle(d.queue)); err != nil {
		return fmt.Errorf("queue wait: %w", err)
	}
	d.pending = false
	var first error
	for _, b := range c.used {
		if err := d.invalidate(b); err != nil && first == nil {
			first = err
		}
	}
	c.release()
	return first
}

func (c *Cmd) release() {
	if c.d == nil || c.d.dev == 0 {
		c.pools = nil
		return
	}
	for _, p := range c.pools {
		c.d.api.destroyDescriptorPool(c.d.dev, p, 0)
	}
	c.pools = nil
}

// Copy uploads or downloads via a one-shot command buffer. src.Len() must
// be <= dst.Len().
func (d *Device) Copy(dst, src *Buffer) error {
	c, err := d.Begin()
	if err != nil {
		return err
	}
	if err := c.Copy(dst, src); err != nil {
		return errors.Join(err, c.Abort())
	}
	if err := c.Submit(); err != nil {
		return err
	}
	return c.Wait()
}

// Abort drops a recording command buffer without submitting it.
func (c *Cmd) Abort() error {
	if c == nil || c.d == nil || !c.d.recording {
		return nil
	}
	err := check(c.d.api.endCommandBuffer(c.d.cmd))
	c.d.recording = false
	c.release()
	return err
}

// Run binds buffers to set 0 and dispatches the shader.
func (d *Device) Run(s *Shader, groupsX, groupsY, groupsZ uint32, bufs ...*Buffer) error {
	c, err := d.Begin()
	if err != nil {
		return err
	}
	if err := c.Bind(s, bufs...); err != nil {
		return errors.Join(err, c.Abort())
	}
	if err := c.Dispatch(groupsX, groupsY, groupsZ); err != nil {
		return errors.Join(err, c.Abort())
	}
	if err := c.Submit(); err != nil {
		return err
	}
	return c.Wait()
}
