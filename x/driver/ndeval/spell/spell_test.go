package spell

import (
	"math"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/require"
)

func TestDialectTokens(t *testing.T) {
	glsl := Dialect{NonFinite: "uintBitsToFloat(%du)", MinInt32: "int(0x80000000u)"}
	metal := Dialect{NonFinite: "as_type<float>(%du)", MinInt32: "int(0x80000000u)"}
	hlsl := Dialect{NonFinite: "asfloat(%du)", MinInt32: "asint(0x80000000u)", IntRelSelect: true}

	low, err := ndarray.Full(float32(math.Inf(-1)), ndarray.Shape{1})
	require.NoError(t, err)
	require.NoError(t, low.Resize(low.Shape()))
	require.Contains(t, body(t, low.Kernel(), glsl), "uintBitsToFloat(")
	require.Contains(t, body(t, low.Kernel(), metal), "as_type<float>(")
	require.NotContains(t, body(t, low.Kernel(), metal), "uintBitsToFloat")
	require.Contains(t, body(t, low.Kernel(), hlsl), "asfloat(")
	require.NotContains(t, body(t, low.Kernel(), hlsl), "uintBitsToFloat")

	min, err := ndarray.Full(int32(math.MinInt32), ndarray.Shape{1})
	require.NoError(t, err)
	require.NoError(t, min.Resize(min.Shape()))
	require.Contains(t, body(t, min.Kernel(), glsl), "int(0x80000000u)")
	require.Contains(t, body(t, min.Kernel(), hlsl), "asint(0x80000000u)")

	a, err := ndarray.New([]int32{1, 2}, ndarray.Shape{2})
	require.NoError(t, err)
	b, err := ndarray.New([]int32{2, 2}, ndarray.Shape{2})
	require.NoError(t, err)
	expr := a.CmpLt(b)
	require.NoError(t, expr.Resize(expr.Shape()))
	require.NotContains(t, body(t, expr.Kernel(), glsl), "?1:0")
	require.Contains(t, body(t, expr.Kernel(), hlsl), "?1:0")
}

func body(t *testing.T, k *ndarray.Kernel, d Dialect) string {
	t.Helper()
	code, err := k.Code()
	require.NoError(t, err)
	var b strings.Builder
	WriteBody(&b, code, d)
	return b.String()
}
