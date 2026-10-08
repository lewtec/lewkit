package d3d12

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestOnStack(t *testing.T) {
	var local byte
	require.True(t, onStack(uintptr(unsafe.Pointer(&local))))
	require.False(t, onStack(0))
	require.False(t, onStack(uintptr(unsafe.Pointer(&local))+1<<30))
}

func TestABILayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip()
	}
	require.Equal(t, uintptr(16), unsafe.Sizeof(guid{}))
	require.Equal(t, uintptr(32), unsafe.Sizeof(rootParam{}))
	require.Equal(t, uintptr(40), unsafe.Sizeof(rootSigDesc{}))
	require.Equal(t, uintptr(20), unsafe.Sizeof(heapProps{}))
	require.Equal(t, uintptr(56), unsafe.Sizeof(resourceDesc{}))
	require.Equal(t, uintptr(8), unsafe.Offsetof(resourceDesc{}.align))
	require.Equal(t, uintptr(16), unsafe.Offsetof(resourceDesc{}.width))
	require.Equal(t, uintptr(32), unsafe.Sizeof(barrier{}))
	require.Equal(t, uintptr(8), unsafe.Offsetof(barrier{}.res))
	require.Equal(t, uintptr(40), unsafe.Sizeof(blendRT{}))
	require.Equal(t, uintptr(328), unsafe.Sizeof(blendDesc{}))
	require.Equal(t, uintptr(44), unsafe.Sizeof(rasterDesc{}))
	require.Equal(t, uintptr(52), unsafe.Sizeof(depthDesc{}))
	require.Equal(t, uintptr(32), unsafe.Sizeof(streamOut{}))
	require.Equal(t, uintptr(16), unsafe.Sizeof(inputLayout{}))

	var pso gfxPSO
	require.Equal(t, uintptr(656), unsafe.Sizeof(pso))
	require.Equal(t, uintptr(120), unsafe.Offsetof(pso.blend))
	require.Equal(t, uintptr(448), unsafe.Offsetof(pso.sampleMask))
	require.Equal(t, uintptr(452), unsafe.Offsetof(pso.raster))
	require.Equal(t, uintptr(496), unsafe.Offsetof(pso.depth))
	require.Equal(t, uintptr(552), unsafe.Offsetof(pso.layout))
	require.Equal(t, uintptr(580), unsafe.Offsetof(pso.rtv))
	require.Equal(t, uintptr(616), unsafe.Offsetof(pso.samples))
	require.Equal(t, uintptr(632), unsafe.Offsetof(pso.cached))
	require.Equal(t, uintptr(648), unsafe.Offsetof(pso.flags))

	var cs computePSO
	require.Equal(t, uintptr(56), unsafe.Sizeof(cs))
	require.Equal(t, uintptr(8), unsafe.Offsetof(cs.cs))
	require.Equal(t, uintptr(32), unsafe.Offsetof(cs.cached))
	require.Equal(t, uintptr(48), unsafe.Sizeof(swapDesc{}))
	require.Equal(t, uintptr(16), unsafe.Sizeof(queueDesc{}))
	require.Equal(t, uintptr(16), unsafe.Sizeof(heapDesc{}))
	require.Equal(t, uintptr(304), unsafe.Sizeof(adapterDesc{}))
}

func TestDrawEnumsMatchD3D12(t *testing.T) {
	require.Equal(t, 5, topoStrip)
	require.Equal(t, 2, blendOne)
	require.Equal(t, 6, blendInvSrc)
	require.Equal(t, 28, fmtRGBA8)
	require.Equal(t, 1, heapDefault)
}
