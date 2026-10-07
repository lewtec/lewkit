//go:build !darwin && !ios

package opengl

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCoverGroups(t *testing.T) {
	wide := [3]uint32{65535, 65535, 65535}
	got := func(groups uint32, max [3]uint32) [3]uint32 {
		t.Helper()
		x, y, z, err := coverGroups(groups, max)
		require.NoError(t, err)
		return [3]uint32{x, y, z}
	}
	require.Equal(t, [3]uint32{1, 1, 1}, got(1, wide))
	require.Equal(t, [3]uint32{65535, 1, 1}, got(65535, wide))
	require.Equal(t, [3]uint32{65535, 2, 1}, got(65536, wide))
	require.Equal(t, [3]uint32{4, 3, 1}, got(10, [3]uint32{4, 8, 4}))
	require.Equal(t, [3]uint32{4, 4, 2}, got(20, [3]uint32{4, 4, 4}))
	require.Equal(t, [3]uint32{4, 4, 4}, got(64, [3]uint32{4, 4, 4}))
	_, _, _, err := coverGroups(65, [3]uint32{4, 4, 4})
	require.ErrorIs(t, err, ErrUnavailable)
	_, _, _, err = coverGroups(0, wide)
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestDispatchPastOneAxis(t *testing.T) {
	device, err := OpenDevice()
	if err != nil {
		t.Skip(err)
	}
	defer device.Close()
	device.limit = [3]uint32{4, 4, 4}

	const n = 20
	src := "#version 430\nlayout(local_size_x = 1) in;\nlayout(std430, binding = 0) buffer Out { uint o[]; };\nvoid main() {\n    uint span = gl_NumWorkGroups.x * gl_WorkGroupSize.x;\n    uint gi = gl_GlobalInvocationID.x + gl_GlobalInvocationID.y * span + gl_GlobalInvocationID.z * span * gl_NumWorkGroups.y;\n    if (gi >= 20u) return;\n    o[gi] = gi;\n}\n"
	if device.GLES() {
		src = "#version 310 es\nprecision highp float;\nprecision highp int;\n" + src[len("#version 430\n"):]
	}
	prog, err := device.Compile(src)
	require.NoError(t, err)
	defer prog.Close()
	buf, err := device.Buffer(n * 4)
	require.NoError(t, err)
	defer buf.Close()
	require.NoError(t, device.Run(prog, n, []*Buffer{buf}, nil))
	raw := make([]byte, n*4)
	require.NoError(t, buf.Read(raw))
	for i := range n {
		require.Equal(t, uint32(i), binary.LittleEndian.Uint32(raw[i*4:]), "index %d", i)
	}
}
