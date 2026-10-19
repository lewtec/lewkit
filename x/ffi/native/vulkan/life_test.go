package vulkan

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLifeDestroysChildBeforeDevice(t *testing.T) {
	for _, ownerFirst := range []bool{true, false} {
		var order []string
		life := &vkLife{refs: 1, dev: 1, inst: 2}
		life.api.destroyBuffer = func(uintptr, uint64, uintptr) { order = append(order, "buffer") }
		life.api.freeMemory = func(uintptr, uint64, uintptr) { order = append(order, "memory") }
		life.api.destroyPipeline = func(uintptr, uint64, uintptr) { order = append(order, "pipeline") }
		life.api.destroyPipelineLayout = func(uintptr, uint64, uintptr) { order = append(order, "pipeline-layout") }
		life.api.destroySetLayout = func(uintptr, uint64, uintptr) { order = append(order, "set-layout") }
		life.api.destroyShaderModule = func(uintptr, uint64, uintptr) { order = append(order, "module") }
		life.api.destroyDevice = func(uintptr, uintptr) { order = append(order, "device") }
		life.api.destroyInstance = func(uintptr, uintptr) { order = append(order, "instance") }
		life.hold()
		life.hold()
		buf := &vkBuf{life: life, buf: 4, mem: 5}
		pipe := &vkPipe{life: life, module: 1, setLayout: 2, pipelineLayout: 3, pipeline: 6}
		if ownerFirst {
			life.releaseOwner()
			buf.release()
			pipe.release()
		} else {
			buf.release()
			pipe.release()
			life.releaseOwner()
		}
		require.Equal(t, []string{
			"buffer", "memory",
			"pipeline", "pipeline-layout", "set-layout", "module",
			"device", "instance",
		}, order)
		buf.release()
		pipe.release()
		life.releaseOwner()
		require.Len(t, order, 8)
	}
}
