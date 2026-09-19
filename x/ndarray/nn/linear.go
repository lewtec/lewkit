package nn

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Linear is y = W x + b. W is (out, in), x is (in,), b is (out,).
func Linear(w, x, b *ndarray.Tensor) (*ndarray.Tensor, error) {
	if w == nil || x == nil || b == nil {
		return nil, ndarray.ErrOp
	}
	ws, xs, bs := w.Shape(), x.Shape(), b.Shape()
	if len(ws) != 2 || len(xs) != 1 || len(bs) != 1 {
		return nil, fmt.Errorf("%w: W %v x %v b %v", ndarray.ErrShape, ws, xs, bs)
	}
	out, in := ws[0], ws[1]
	if xs[0] != in || bs[0] != out {
		return nil, fmt.Errorf("%w: W %v x %v b %v", ndarray.ErrShape, ws, xs, bs)
	}
	if in == 0 {
		return b, nil
	}
	var acc *ndarray.Tensor
	for j := range in {
		col, err := w.Shrink([][2]int{{0, out}, {j, j + 1}})
		if err != nil {
			return nil, err
		}
		col, err = col.Reshape(ndarray.Shape{out})
		if err != nil {
			return nil, err
		}
		xj, err := x.Shrink([][2]int{{j, j + 1}})
		if err != nil {
			return nil, err
		}
		xj, err = xj.Expand(ndarray.Shape{out})
		if err != nil {
			return nil, err
		}
		term := col.Mul(xj)
		if acc == nil {
			acc = term
		} else {
			acc = acc.Add(term)
		}
	}
	return acc.Add(b), nil
}
