package ndarray

import (
	"fmt"
	"slices"
	"strings"
)

// Tracker is a stack of views from a contiguous buffer to a logical tensor.
type Tracker struct {
	views []view
}

// Of is a contiguous row-major tensor of shape.
func Of(shape Shape) (Tracker, error) {
	v, err := create(view{shape: shape})
	if err != nil {
		return Tracker{}, err
	}
	return Tracker{views: []view{v}}, nil
}

func (t Tracker) last() view {
	return t.views[len(t.views)-1]
}

func (t Tracker) check() error {
	if len(t.views) == 0 {
		return ErrShape
	}
	return nil
}

func (t Tracker) replace(v view) Tracker {
	views := slices.Clone(t.views)
	views[len(views)-1] = v
	return Tracker{views: views}
}

// Shape is the logical shape.
func (t Tracker) Shape() Shape {
	if len(t.views) == 0 {
		return nil
	}
	return t.last().shape.Clone()
}

// Size is the number of logical cells.
func (t Tracker) Size() int {
	if len(t.views) == 0 {
		return 0
	}
	return t.last().size()
}

// Contiguous reports a single row-major view with no offset or mask.
func (t Tracker) Contiguous() bool {
	return len(t.views) == 1 && t.views[0].contiguous()
}

// RealSize is the host buffer length that covers every valid offset.
func (t Tracker) RealSize() int {
	if len(t.views) == 0 || t.views[0].shape.HasZero() {
		return 0
	}
	v := t.views[0]
	maxOff := v.offset
	for i, s := range v.shape {
		lo, hi := 0, s
		if v.mask != nil {
			lo, hi = v.mask[i][0], v.mask[i][1]
		}
		if hi <= lo {
			return 0
		}
		stride := v.strides[i]
		a, b := lo*stride, (hi-1)*stride
		if a > b {
			a, b = b, a
		}
		maxOff += b
	}
	return max(0, maxOff+1)
}

// Index maps logical coords to a buffer offset. valid is false in padding.
func (t Tracker) Index(coords ...int) (offset int, valid bool, err error) {
	if err := t.check(); err != nil {
		return 0, false, err
	}
	shape := t.last().shape
	if len(coords) != len(shape) {
		return 0, false, fmt.Errorf("%w: got %d coords for %v", ErrIndex, len(coords), shape)
	}
	for i, c := range coords {
		if c < 0 || c >= shape[i] {
			return 0, false, fmt.Errorf("%w: %v in %v", ErrIndex, coords, shape)
		}
	}
	off, ok := t.last().index(coords)
	for i := len(t.views) - 2; i >= 0; i-- {
		v := t.views[i]
		if v.size() == 0 {
			return 0, false, nil
		}
		coords = unravel(v.shape, off)
		var viewOK bool
		off, viewOK = v.index(coords)
		ok = ok && viewOK
	}
	return off, ok, nil
}

// At maps a row-major raveled index to a buffer offset.
func (t Tracker) At(i int) (offset int, valid bool, err error) {
	if err := t.check(); err != nil {
		return 0, false, err
	}
	n := t.Size()
	if i < 0 || (n > 0 && i >= n) || (n == 0) {
		return 0, false, fmt.Errorf("%w: %d of %d", ErrIndex, i, n)
	}
	return t.Index(unravel(t.last().shape, i)...)
}

// Reshape changes the logical shape. Product must match. One -1 is inferred.
func (t Tracker) Reshape(shape Shape) (Tracker, error) {
	if err := t.check(); err != nil {
		return Tracker{}, err
	}
	shape, err := shape.Infer(t.Size())
	if err != nil {
		return Tracker{}, err
	}
	if newView, ok := t.last().reshape(shape); ok {
		return t.replace(newView), nil
	}
	newView, err := create(view{shape: shape})
	if err != nil {
		return Tracker{}, err
	}
	return Tracker{views: append(slices.Clone(t.views), newView)}, nil
}

// Permute reorders axes. Axes may be negative.
func (t Tracker) Permute(axes ...int) (Tracker, error) {
	if err := t.check(); err != nil {
		return Tracker{}, err
	}
	norm, err := normalizeAxes(axes, len(t.last().shape), true)
	if err != nil {
		return Tracker{}, err
	}
	v, err := t.last().permute(norm)
	if err != nil {
		return Tracker{}, err
	}
	return t.replace(v), nil
}

// Expand broadcasts size-1 axes. Rank stays the same.
func (t Tracker) Expand(shape Shape) (Tracker, error) {
	if err := t.check(); err != nil {
		return Tracker{}, err
	}
	v, err := t.last().expand(shape)
	if err != nil {
		return Tracker{}, err
	}
	return t.replace(v), nil
}

// Pad adds zeros before and after each axis. arg[i] is {before, after}.
func (t Tracker) Pad(arg [][2]int) (Tracker, error) {
	if err := t.check(); err != nil {
		return Tracker{}, err
	}
	v, err := t.last().pad(arg)
	if err != nil {
		return Tracker{}, err
	}
	return t.replace(v), nil
}

// Shrink keeps [start, end) on each axis. arg[i] is {start, end}.
func (t Tracker) Shrink(arg [][2]int) (Tracker, error) {
	if err := t.check(); err != nil {
		return Tracker{}, err
	}
	v, err := t.last().shrink(arg)
	if err != nil {
		return Tracker{}, err
	}
	return t.replace(v), nil
}

// Flip reverses the given axes. Axes may be negative.
func (t Tracker) Flip(axes ...int) (Tracker, error) {
	if err := t.check(); err != nil {
		return Tracker{}, err
	}
	rank := len(t.last().shape)
	which := make([]bool, rank)
	if len(axes) == 0 {
		return t, nil
	}
	norm, err := normalizeAxes(axes, rank, false)
	if err != nil {
		return Tracker{}, err
	}
	for _, a := range norm {
		which[a] = true
	}
	v, err := t.last().flip(which)
	if err != nil {
		return Tracker{}, err
	}
	return t.replace(v), nil
}

func normalizeAxes(axes []int, rank int, perm bool) ([]int, error) {
	out := make([]int, len(axes))
	for i, a := range axes {
		if a < 0 {
			a += rank
		}
		if a < 0 || a >= rank {
			return nil, fmt.Errorf("%w: %d of rank %d", ErrAxis, axes[i], rank)
		}
		out[i] = a
	}
	if perm {
		if !isPermutation(out, rank) {
			return nil, fmt.Errorf("%w: permute %v of rank %d", ErrAxis, axes, rank)
		}
	}
	return out, nil
}

// String is shape and whether the view is a single contiguous map.
func (t Tracker) String() string {
	if len(t.views) == 0 {
		return "Tracker{}"
	}
	var b strings.Builder
	b.WriteString("Tracker(")
	b.WriteString(t.last().shape.String())
	if t.Contiguous() {
		b.WriteString(" contiguous")
	}
	b.WriteByte(')')
	return b.String()
}
