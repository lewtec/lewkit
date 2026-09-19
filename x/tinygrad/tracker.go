package tinygrad

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// Tracker is a stack of views from a contiguous buffer to a logical tensor.
type Tracker struct {
	views []view
}

// Of is a contiguous row-major tensor of shape.
func Of(shape ...int) (Tracker, error) {
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
func (t Tracker) Shape() []int {
	if len(t.views) == 0 {
		return nil
	}
	return slices.Clone(t.last().shape)
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
	if len(t.views) == 0 || hasZero(t.views[0].shape) {
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
		st := v.strides[i]
		a, b := lo*st, (hi-1)*st
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
	sh := t.last().shape
	if len(coords) != len(sh) {
		return 0, false, fmt.Errorf("%w: got %d coords for %v", ErrIndex, len(coords), sh)
	}
	for i, c := range coords {
		if c < 0 || c >= sh[i] {
			return 0, false, fmt.Errorf("%w: %v in %v", ErrIndex, coords, sh)
		}
	}
	off, ok := t.last().index(coords)
	for i := len(t.views) - 2; i >= 0; i-- {
		v := t.views[i]
		if v.size() == 0 {
			return 0, false, nil
		}
		coords = unravel(v.shape, off)
		var vok bool
		off, vok = v.index(coords)
		ok = ok && vok
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
func (t Tracker) Reshape(shape ...int) (Tracker, error) {
	if err := t.check(); err != nil {
		return Tracker{}, err
	}
	shape, err := infer(shape, t.Size())
	if err != nil {
		return Tracker{}, err
	}
	if nv, ok := t.last().reshape(shape); ok {
		return t.replace(nv), nil
	}
	nv, err := create(view{shape: shape})
	if err != nil {
		return Tracker{}, err
	}
	return Tracker{views: append(slices.Clone(t.views), nv)}, nil
}

func infer(shape []int, size int) ([]int, error) {
	hole := -1
	n := 1
	for i, s := range shape {
		if s == -1 {
			if hole >= 0 {
				return nil, fmt.Errorf("%w: more than one -1 in %v", ErrShape, shape)
			}
			hole = i
			continue
		}
		if s < 0 {
			return nil, fmt.Errorf("%w: %v", ErrShape, shape)
		}
		n *= s
	}
	out := slices.Clone(shape)
	if hole >= 0 {
		if n == 0 {
			if size != 0 {
				return nil, fmt.Errorf("%w: %v of %d", ErrSize, shape, size)
			}
			out[hole] = 0
			return out, nil
		}
		if size%n != 0 {
			return nil, fmt.Errorf("%w: %v of %d", ErrSize, shape, size)
		}
		out[hole] = size / n
		return out, nil
	}
	if n != size {
		return nil, fmt.Errorf("%w: %v of %d", ErrSize, shape, size)
	}
	return out, nil
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
func (t Tracker) Expand(shape ...int) (Tracker, error) {
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
		if !isPerm(out, rank) {
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
	b.WriteByte('[')
	for i, s := range t.last().shape {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strconv.Itoa(s))
	}
	b.WriteByte(']')
	if t.Contiguous() {
		b.WriteString(" contiguous")
	}
	b.WriteByte(')')
	return b.String()
}
