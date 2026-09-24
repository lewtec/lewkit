package vulkan

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestChoosePresentMode(t *testing.T) {
	require.Equal(t, int32(presentMailbox), choosePresentMode([]int32{presentImmediate, presentMailbox, presentFIFO}))
	require.Equal(t, int32(presentFIFO), choosePresentMode([]int32{presentFIFO, presentImmediate}))
	require.Equal(t, int32(presentFIFO), choosePresentMode([]int32{presentFIFO}))
	require.Equal(t, int32(presentFIFO), choosePresentMode(nil))
}

func TestGraphicsStructSizes(t *testing.T) {
	require.Equal(t, uintptr(48), unsafe.Sizeof(vertexInputState{}))
	require.Equal(t, uintptr(32), unsafe.Sizeof(inputAssemblyState{}))
	require.Equal(t, uintptr(48), unsafe.Sizeof(viewportState{}))
	require.Equal(t, uintptr(64), unsafe.Sizeof(rasterState{}))
	require.Equal(t, uintptr(48), unsafe.Sizeof(multisampleState{}))
	require.Equal(t, uintptr(32), unsafe.Sizeof(blendAttachment{}))
	require.Equal(t, uintptr(56), unsafe.Sizeof(colorBlendState{}))
	require.Equal(t, uintptr(32), unsafe.Sizeof(dynamicStateInfo{}))
	require.Equal(t, uintptr(144), unsafe.Sizeof(graphicsPipelineInfo{}))
	require.Equal(t, uintptr(36), unsafe.Sizeof(attachmentDescription{}))
	require.Equal(t, uintptr(8), unsafe.Sizeof(attachmentReference{}))
	require.Equal(t, uintptr(72), unsafe.Sizeof(subpassDescription{}))
	require.Equal(t, uintptr(28), unsafe.Sizeof(subpassDependency{}))
	require.Equal(t, uintptr(64), unsafe.Sizeof(renderPassInfo{}))
	require.Equal(t, uintptr(64), unsafe.Sizeof(framebufferInfo{}))
	require.Equal(t, uintptr(16), unsafe.Sizeof(rect2D{}))
	require.Equal(t, uintptr(64), unsafe.Sizeof(renderPassBegin{}))
	require.Equal(t, uintptr(56), unsafe.Sizeof(bufferBarrier{}))
}
