package nn

import (
	"fmt"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Linear is y = W x + b. W is (out, in), x is (in,), b is (out,).
// Binding 0 is W, 1 is x, 2 is b.
func Linear(w, x, b ndarray.Tracker) (*ndarray.Node, error) {
	ws, xs, bs := w.Shape(), x.Shape(), b.Shape()
	if len(ws) != 2 || len(xs) != 1 || len(bs) != 1 {
		return nil, fmt.Errorf("%w: W %v x %v b %v", ndarray.ErrShape, ws, xs, bs)
	}
	out, in := ws[0], ws[1]
	if xs[0] != in || bs[0] != out {
		return nil, fmt.Errorf("%w: W %v x %v b %v", ndarray.ErrShape, ws, xs, bs)
	}
	if in == 0 {
		return ndarray.In(2, b), nil
	}
	var acc *ndarray.Node
	for j := range in {
		col, err := w.Shrink([][2]int{{0, out}, {j, j + 1}})
		if err != nil {
			return nil, err
		}
		col, err = col.Reshape(out)
		if err != nil {
			return nil, err
		}
		xj, err := x.Shrink([][2]int{{j, j + 1}})
		if err != nil {
			return nil, err
		}
		xj, err = xj.Expand(out)
		if err != nil {
			return nil, err
		}
		term := ndarray.In(0, col).Mul(ndarray.In(1, xj))
		if acc == nil {
			acc = term
		} else {
			acc = acc.Add(term)
		}
	}
	return acc.Add(ndarray.In(2, b)), nil
}
