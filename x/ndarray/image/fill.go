package image

import "github.com/lewtec/lewkit/x/ndarray"

// Fill is a constant RGBA picture of size h×w.
func Fill(h, w int, r, g, b, a float32) (*ndarray.Tensor[float32], error) {
	if h < 1 || w < 1 {
		return nil, ndarray.ErrShape
	}
	channel := ndarray.Coord(2, ndarray.Shape{h, w, 4})
	v := ndarray.Where(channel.Equal(ndarray.ConstInt(0)), ndarray.Const(r),
		ndarray.Where(channel.Equal(ndarray.ConstInt(1)), ndarray.Const(g),
			ndarray.Where(channel.Equal(ndarray.ConstInt(2)), ndarray.Const(b), ndarray.Const(a))))
	if v.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return v, nil
}
