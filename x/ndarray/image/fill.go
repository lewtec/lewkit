package image

import "github.com/lewtec/lewkit/x/ndarray"

// Fill is a constant RGBA picture of size h×w.
func Fill(h, w int, r, g, b, a float32) (*ndarray.Tensor, error) {
	if h < 1 || w < 1 {
		return nil, ndarray.ErrShape
	}
	channel := ndarray.Coord(2, ndarray.Shape{h, w, 4})
	v := channel.Equal(ndarray.ConstInt(0)).Where(ndarray.Const(r),
		channel.Equal(ndarray.ConstInt(1)).Where(ndarray.Const(g),
			channel.Equal(ndarray.ConstInt(2)).Where(ndarray.Const(b), ndarray.Const(a))))
	if v.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return v, nil
}
