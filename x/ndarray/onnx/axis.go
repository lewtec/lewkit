package onnx

import "github.com/lewtec/lewkit/x/ndarray"

// axisIndex maps axis into [0, n). A negative axis counts back from n.
func axisIndex(axis, n int) (int, error) {
	if axis < 0 {
		axis += n
	}
	if axis < 0 || axis >= n {
		return 0, ndarray.ErrShape
	}
	return axis, nil
}
