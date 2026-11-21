package image

import "github.com/lewtec/lewkit/x/ndarray"

// Fill is a constant RGBA picture of size h×w.
func Fill(h, w int, r, g, b, a float32) (*ndarray.Tensor[float32], error) {
	if h < 1 || w < 1 {
		return nil, ndarray.ErrShape
	}
	channel := ndarray.Coord(2, ndarray.Shape{h, w, 4})
	v := channel.Equal(ndarray.Const[int32](0)).Where(ndarray.Const(r),
		channel.Equal(ndarray.Const[int32](1)).Where(ndarray.Const(g),
			channel.Equal(ndarray.Const[int32](2)).Where(ndarray.Const(b), ndarray.Const(a))))
	if v.Shape() == nil {
		return nil, ndarray.ErrOp
	}
	return v, nil
}
