package glslc_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/glslc"
	_ "github.com/lewtec/lewkit/x/driver/glslc/prelude"
	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/stretchr/testify/require"
)

func TestNativePrecedes(t *testing.T) {
	handles, err := driver.List[glslc.Compiler](t.Context())
	require.NoError(t, err)
	require.NotEmpty(t, handles)
	if execdriver.IsBinaryAvailable(t.Context(), "glslc") {
		require.Equal(t, "glslc_native", handles[0].ID)
		return
	}
	require.Equal(t, "glslc_lewkit", handles[0].ID)
	for _, handle := range handles {
		require.NotEqual(t, "glslc_native", handle.ID)
	}
}

func TestLewkitCompiles(t *testing.T) {
	if execdriver.IsBinaryAvailable(t.Context(), "glslc") {
		t.Skip("native glslc is on PATH")
	}
	compiler, err := driver.Get[glslc.Compiler](t.Context())
	require.NoError(t, err)
	src := []byte("#version 450\nlayout(local_size_x = 1) in;\nvoid main() {}\n")
	got, err := compiler.Compile(t.Context(), src)
	require.NoError(t, err)
	require.True(t, glsl.IsSPIRV(got))
}
