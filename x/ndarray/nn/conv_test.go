package nn

import (
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/require"
)

func evalCPU[T ndarray.Number](t *testing.T, y *ndarray.Tensor[T]) []T {
	t.Helper()
	got := make([]T, y.Size())
	require.NoError(t, y.Eval(t.Context(), ndarray.CPU, got))
	return got
}

func TestConvolution2DUint8Rejected(t *testing.T) {
	x, err := ndarray.New([]uint8{1, 2, 3, 4}, ndarray.Shape{1, 1, 2, 2})
	require.NoError(t, err)
	w, err := ndarray.New([]uint8{1}, ndarray.Shape{1, 1, 1, 1})
	require.NoError(t, err)
	_, err = Convolution2D(x, w, []int{0, 0, 0, 0}, 1, 1)
	require.ErrorIs(t, err, ndarray.ErrType)
}

func TestMaximumPool2DUndersized(t *testing.T) {
	x, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{1, 1, 2, 2})
	require.NoError(t, err)
	_, err = MaximumPool2D(x, 3, 3, 2, 2, []int{0, 0, 0, 0})
	require.Error(t, err)
}

func TestConvolution2D(t *testing.T) {
	x, err := ndarray.New([]float32{
		1, 2, 3,
		4, 5, 6,
		7, 8, 9,
	}, ndarray.Shape{1, 1, 3, 3})
	require.NoError(t, err)
	w, err := ndarray.New([]float32{1, 1, 1, 1}, ndarray.Shape{1, 1, 2, 2})
	require.NoError(t, err)
	y, err := Convolution2D(x, w, []int{0, 0, 0, 0}, 1, 1)
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{1, 1, 2, 2}, y.Shape())
	require.Equal(t, []float32{12, 16, 24, 28}, evalCPU(t, y))
}

func TestConvolution2DBiasBroadcast(t *testing.T) {
	x, err := ndarray.New(make([]float32, 8*8), ndarray.Shape{1, 1, 8, 8})
	require.NoError(t, err)
	w, err := ndarray.New(make([]float32, 8*3*3), ndarray.Shape{8, 1, 3, 3})
	require.NoError(t, err)
	y, err := Convolution2D(x, w, []int{1, 1, 1, 1}, 1, 1)
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{1, 8, 8, 8}, y.Shape())
	require.Len(t, evalCPU(t, y), 8*8*8)
}

func TestConvolution2DStride(t *testing.T) {
	x, err := ndarray.New([]float32{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}, ndarray.Shape{1, 1, 4, 4})
	require.NoError(t, err)
	w, err := ndarray.New([]float32{1, 0, 0, 1}, ndarray.Shape{1, 1, 2, 2})
	require.NoError(t, err)
	y, err := Convolution2D(x, w, []int{0, 0, 0, 0}, 2, 2)
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{1, 1, 2, 2}, y.Shape())
	require.Equal(t, []float32{7, 11, 23, 27}, evalCPU(t, y))
}

func TestMaximumPool2D(t *testing.T) {
	x, err := ndarray.New([]float32{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}, ndarray.Shape{1, 1, 4, 4})
	require.NoError(t, err)
	y, err := MaximumPool2D(x, 2, 2, 2, 2, nil)
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{1, 1, 2, 2}, y.Shape())
	require.Equal(t, []float32{6, 8, 14, 16}, evalCPU(t, y))
}

func TestMaximumPool2DNegativePad(t *testing.T) {
	x, err := ndarray.New([]float32{
		-5, -1,
		-3, -2,
	}, ndarray.Shape{1, 1, 2, 2})
	require.NoError(t, err)
	y, err := MaximumPool2D(x, 2, 2, 1, 1, []int{1, 1, 1, 1})
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{1, 1, 3, 3}, y.Shape())
	require.Equal(t, []float32{-5, -1, -1, -3, -1, -1, -3, -2, -2}, evalCPU(t, y))
}

func TestMaximumPool2DOfAddStaysLazy(t *testing.T) {
	x, err := ndarray.New([]float32{
		1, 2, 3, 4,
		5, 6, 7, 8,
		9, 10, 11, 12,
		13, 14, 15, 16,
	}, ndarray.Shape{1, 1, 4, 4})
	require.NoError(t, err)
	y, err := MaximumPool2D(x.Add(ndarray.Const(float32(0))), 2, 2, 2, 2, nil)
	require.NoError(t, err)
	_, err = y.Data()
	require.ErrorIs(t, err, ndarray.ErrOp)
	require.Equal(t, []float32{6, 8, 14, 16}, evalCPU(t, y))
}

func TestMaximumPool2DNegativePadOfAdd(t *testing.T) {
	x, err := ndarray.New([]float32{
		-5, -1,
		-3, -2,
	}, ndarray.Shape{1, 1, 2, 2})
	require.NoError(t, err)
	y, err := MaximumPool2D(x.Add(ndarray.Const(float32(0))), 2, 2, 1, 1, []int{1, 1, 1, 1})
	require.NoError(t, err)
	_, err = y.Data()
	require.ErrorIs(t, err, ndarray.ErrOp)
	require.Equal(t, []float32{-5, -1, -1, -3, -1, -1, -3, -2, -2}, evalCPU(t, y))
}

func TestMaximumPool2DOverlap(t *testing.T) {
	x, err := ndarray.New([]float32{
		1, 2, 3,
		4, 5, 6,
		7, 8, 9,
	}, ndarray.Shape{1, 1, 3, 3})
	require.NoError(t, err)
	y, err := MaximumPool2D(x, 2, 2, 1, 1, nil)
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{1, 1, 2, 2}, y.Shape())
	require.Equal(t, []float32{5, 6, 8, 9}, evalCPU(t, y))
}

func TestMatrixMultiply(t *testing.T) {
	a, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{2, 2})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{5, 6, 7, 8}, ndarray.Shape{2, 2})
	require.NoError(t, err)
	y, err := MatrixMultiply(a, b)
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{2, 2}, y.Shape())
	require.Equal(t, []float32{19, 22, 43, 50}, evalCPU(t, y))
}

func TestMatrixMultiply1D(t *testing.T) {
	a, err := ndarray.New([]float32{1, 2, 3}, ndarray.Shape{3})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{4, 5, 6}, ndarray.Shape{3})
	require.NoError(t, err)
	y, err := MatrixMultiply(a, b)
	require.NoError(t, err)
	require.Equal(t, []float32{32}, evalCPU(t, y))
	require.Equal(t, 1, y.Size())
}

func TestMatrixMultiplyBatch(t *testing.T) {
	a, err := ndarray.New([]float32{
		1, 2, 3, 4,
		5, 6, 7, 8,
	}, ndarray.Shape{2, 2, 2})
	require.NoError(t, err)
	b, err := ndarray.New([]float32{
		1, 0, 0, 1,
		1, 1, 1, 1,
	}, ndarray.Shape{2, 2, 2})
	require.NoError(t, err)
	y, err := MatrixMultiply(a, b)
	require.NoError(t, err)
	require.Equal(t, ndarray.Shape{2, 2, 2}, y.Shape())
	require.Equal(t, []float32{1, 2, 3, 4, 11, 11, 15, 15}, evalCPU(t, y))
}
