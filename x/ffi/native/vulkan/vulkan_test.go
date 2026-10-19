package vulkan

import (
	"encoding/binary"
	"strings"
	"testing"
	"unsafe"

	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncMappedSkip(t *testing.T) {
	d := &Device{}
	require.NoError(t, d.flush(nil))
	require.NoError(t, d.invalidate(nil))
	require.NoError(t, d.flush(&Buffer{coherent: true}))
	require.NoError(t, d.invalidate(&Buffer{coherent: true}))
}

func TestSmokeSPIRVMagic(t *testing.T) {
	b := SmokeSPIRV()
	require.GreaterOrEqual(t, len(b), 20)
	require.Equal(t, uint32(0x07230203), binary.LittleEndian.Uint32(b[:4]))
	require.Equal(t, 0, len(b)%4)
}

func TestListAllComputeDevices(t *testing.T) {
	infos, err := List(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	require.NotEmpty(t, infos)
	names := map[string]int{}
	for i, info := range infos {
		require.Equal(t, i, info.Index)
		require.NotEmpty(t, info.Name)
		d, err := OpenIndex(t.Context(), i)
		require.NoError(t, err)
		test.CloseOnCleanup(t, d)
		require.Equal(t, info.Name, d.Name())
		require.Equal(t, info.Vendor, d.Vendor())
		require.Equal(t, info.Type, d.Type())
		names[info.Name]++
		lower := strings.ToLower(info.Name)
		switch {
		case strings.Contains(lower, "llvmpipe"):
			require.Equal(t, VendorMesa, info.Vendor)
			require.Equal(t, DeviceTypeSoftware, info.Type)
		case strings.Contains(lower, "nvidia"):
			require.Equal(t, VendorNVIDIA, info.Vendor)
			require.Equal(t, DeviceTypeDedicated, info.Type)
		case strings.Contains(lower, "amd") || strings.Contains(lower, "radv"):
			require.Equal(t, VendorAMD, info.Vendor)
			require.Equal(t, DeviceTypeIntegrated, info.Type)
		}
	}
	_, err = OpenIndex(t.Context(), len(infos))
	require.ErrorIs(t, err, ErrNoDevice)
	_, err = OpenIndex(t.Context(), -1)
	require.ErrorIs(t, err, ErrNoDevice)
	t.Logf("vulkan devices: %v", infos)
	require.GreaterOrEqual(t, len(names), 1)
}

func TestOpenInvalid(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	test.CloseOnCleanup(t, d)
	_, err = d.Buffer(0)
	require.ErrorIs(t, err, ErrSize)
	_, err = d.Shader(t.Context(), nil, 1)
	require.ErrorIs(t, err, ErrShader)
	_, err = d.Shader(t.Context(), SmokeSPIRV(), 0)
	require.ErrorIs(t, err, ErrShader)
}

func TestHostRoundTrip(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	test.CloseOnCleanup(t, d)
	buf, err := d.Buffer(16)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	in := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	require.NoError(t, buf.Write(in))
	got := make([]byte, 16)
	require.NoError(t, buf.Read(got))
	require.Equal(t, in, got)
}

func TestDispatch(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	test.CloseOnCleanup(t, d)
	require.NotEmpty(t, d.Name())

	buf, err := d.Buffer(4)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	require.NoError(t, buf.Write(make([]byte, 4)))

	sh, err := d.Shader(t.Context(), SmokeSPIRV(), 1)
	require.NoError(t, err)
	test.CloseOnCleanup(t, sh)
	require.NoError(t, d.Run(sh, 1, 1, 1, buf))

	got := make([]byte, 4)
	require.NoError(t, buf.Read(got))
	have := binary.LittleEndian.Uint32(got)
	require.Equal(t, uint32(2), have, d.Name())
}

func TestRunBufferCount(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	test.CloseOnCleanup(t, d)
	buf, err := d.Buffer(256)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	sh, err := d.Shader(t.Context(), SmokeSPIRV(), 1)
	require.NoError(t, err)
	test.CloseOnCleanup(t, sh)
	err = d.Run(sh, 1, 1, 1)
	require.ErrorIs(t, err, ErrShader)
	assert.Contains(t, err.Error(), "want 1")
}

func TestWriteTooLong(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	test.CloseOnCleanup(t, d)
	buf, err := d.Buffer(4)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	require.ErrorIs(t, buf.Write([]byte{1, 2, 3, 4, 5}), ErrSize)
}

func TestAllocCopy(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	test.CloseOnCleanup(t, d)
	src, err := d.Alloc(16, Host)
	require.NoError(t, err)
	test.CloseOnCleanup(t, src)
	in := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	require.NoError(t, src.Write(in))
	gpu, err := d.Alloc(16, Local)
	if err != nil {
		test.Skip(t, err)
	}
	test.CloseOnCleanup(t, gpu)
	require.Equal(t, Local, gpu.Memory())
	require.NoError(t, d.Copy(gpu, src))
	dst, err := d.Alloc(16, Host)
	require.NoError(t, err)
	test.CloseOnCleanup(t, dst)
	require.NoError(t, d.Copy(dst, gpu))
	got := make([]byte, 16)
	require.NoError(t, dst.Read(got))
	require.Equal(t, in, got)
}

func TestCmdDispatch(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	test.CloseOnCleanup(t, d)
	buf, err := d.Buffer(4)
	require.NoError(t, err)
	test.CloseOnCleanup(t, buf)
	require.NoError(t, buf.Write(make([]byte, 4)))
	sh, err := d.Shader(t.Context(), SmokeSPIRV(), 1)
	require.NoError(t, err)
	test.CloseOnCleanup(t, sh)
	c, err := d.Begin()
	require.NoError(t, err)
	require.NoError(t, c.Bind(sh, buf))
	require.NoError(t, c.Dispatch(1, 1, 1))
	require.NoError(t, c.Submit())
	require.NoError(t, c.Wait())
	got := make([]byte, 4)
	require.NoError(t, buf.Read(got))
	require.Equal(t, uint32(2), binary.LittleEndian.Uint32(got))
}

func TestBeginBusy(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	test.CloseOnCleanup(t, d)
	c, err := d.Begin()
	require.NoError(t, err)
	_, err = d.Begin()
	require.ErrorIs(t, err, ErrBusy)
	require.NoError(t, c.Abort())
}

func TestCompilePushReject(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	test.CloseOnCleanup(t, d)
	_, err = d.Compile(t.Context(), ShaderConfig{SPIRV: SmokeSPIRV(), Bindings: 1, PushBytes: 3})
	require.ErrorIs(t, err, ErrPush)
}

func TestCloseIdempotent(t *testing.T) {
	d, err := Open(t.Context())
	if err != nil {
		test.Skip(t, err)
	}
	require.NoError(t, d.Close())
	require.NoError(t, d.Close())
	_, err = d.Buffer(16)
	require.ErrorIs(t, err, ErrClosed)
}

func TestDeviceCloseDestroysChildrenFirst(t *testing.T) {
	var order []string
	d := &Device{dev: 1, inst: 2}
	d.api.unmapMemory = func(device uintptr, memory uint64) {
		require.Equal(t, uintptr(1), device)
		require.Equal(t, uint64(5), memory)
		order = append(order, "unmap")
	}
	d.api.destroyBuffer = func(device uintptr, buffer uint64, alloc uintptr) {
		require.Equal(t, uint64(4), buffer)
		order = append(order, "buffer")
	}
	d.api.freeMemory = func(device uintptr, memory uint64, alloc uintptr) {
		order = append(order, "memory")
	}
	d.api.destroyPipeline = func(device uintptr, pipe uint64, alloc uintptr) {
		order = append(order, "pipeline")
	}
	d.api.destroyPipelineLayout = func(device uintptr, layout uint64, alloc uintptr) {
		order = append(order, "pipeline-layout")
	}
	d.api.destroySetLayout = func(device uintptr, layout uint64, alloc uintptr) {
		order = append(order, "set-layout")
	}
	d.api.destroyShaderModule = func(device uintptr, module uint64, alloc uintptr) {
		order = append(order, "module")
	}
	d.api.destroyDevice = func(device uintptr, alloc uintptr) {
		require.Equal(t, uintptr(1), device)
		order = append(order, "device")
	}
	d.api.destroyInstance = func(instance uintptr, alloc uintptr) {
		require.Equal(t, uintptr(2), instance)
		order = append(order, "instance")
	}
	buf := &Buffer{d: d, buf: 4, mem: 5, ptr: unsafe.Pointer(&order), size: 8}
	shader := &Shader{d: d, module: 1, setLayout: 2, pipelineLayout: 3, pipeline: 6}
	d.track(buf)
	d.track(shader)
	require.NoError(t, d.Close())
	require.Equal(t, []string{
		"unmap", "buffer", "memory",
		"pipeline", "pipeline-layout", "set-layout", "module",
		"device", "instance",
	}, order)
	require.NoError(t, buf.Close())
	require.NoError(t, shader.Close())
	require.Len(t, order, 9)
}

func TestBufferCloseBeforeDeviceDoesNotFreeTwice(t *testing.T) {
	var buffers int
	d := &Device{dev: 1, inst: 2}
	d.api.unmapMemory = func(uintptr, uint64) {}
	d.api.destroyBuffer = func(uintptr, uint64, uintptr) { buffers++ }
	d.api.freeMemory = func(uintptr, uint64, uintptr) {}
	d.api.destroyDevice = func(uintptr, uintptr) {}
	d.api.destroyInstance = func(uintptr, uintptr) {}
	buf := &Buffer{d: d, buf: 4, mem: 5, ptr: unsafe.Pointer(&buffers), size: 4}
	d.track(buf)
	require.NoError(t, buf.Close())
	require.Equal(t, 1, buffers)
	require.NoError(t, d.Close())
	require.Equal(t, 1, buffers)
}
