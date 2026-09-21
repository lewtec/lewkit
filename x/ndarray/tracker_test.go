package ndarray

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func gather(t *testing.T, tracker Tracker, buf []int) []int {
	t.Helper()
	out := make([]int, tracker.Size())
	for i := range out {
		off, ok, err := tracker.At(i)
		require.NoError(t, err)
		if !ok {
			continue
		}
		require.GreaterOrEqual(t, off, 0)
		require.Less(t, off, len(buf))
		out[i] = buf[off]
	}
	return out
}

func arange(n int) []int {
	a := make([]int, n)
	for i := range a {
		a[i] = i
	}
	return a
}

func TestOfShapeSize(t *testing.T) {
	tr := mustTracker(t, Shape{2, 3, 4})
	require.Equal(t, Shape{2, 3, 4}, tr.Shape())
	require.Equal(t, 24, tr.Size())
	require.True(t, tr.Contiguous())
	require.Equal(t, "Tracker([2 3 4] contiguous)", tr.String())
}

func TestSplatKeepsOffset(t *testing.T) {
	tr := mustTracker(t, Shape{4})
	tr, err := tr.Shrink([][2]int{{2, 3}})
	require.NoError(t, err)
	tr, err = tr.Splat()
	require.NoError(t, err)
	require.Empty(t, tr.Shape())
	off, ok, err := tr.At(0)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 2, off)
}

func TestScalar(t *testing.T) {
	tr := mustTracker(t, Shape{})
	require.Equal(t, 1, tr.Size())
	require.Empty(t, tr.Shape())
	off, ok, err := tr.Index()
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 0, off)
}

func TestZeroDim(t *testing.T) {
	tr := mustTracker(t, Shape{2, 0, 3})
	require.Equal(t, 0, tr.Size())
	require.True(t, tr.Contiguous())
	_, _, err := tr.At(0)
	require.ErrorIs(t, err, ErrIndex)
}

func TestIndexRowMajor(t *testing.T) {
	tr := mustTracker(t, Shape{2, 3})
	for i, want := range []struct {
		c   []int
		off int
	}{
		{[]int{0, 0}, 0},
		{[]int{0, 2}, 2},
		{[]int{1, 0}, 3},
		{[]int{1, 2}, 5},
	} {
		off, ok, err := tr.Index(want.c...)
		require.NoError(t, err, i)
		require.True(t, ok, i)
		require.Equal(t, want.off, off, i)
	}
}

func TestPermute(t *testing.T) {
	tr := mustTracker(t, Shape{2, 3})
	tr, err := tr.Permute(1, 0)
	require.NoError(t, err)
	require.Equal(t, Shape{3, 2}, tr.Shape())
	require.False(t, tr.Contiguous())
	buf := arange(6)
	require.Equal(t, []int{0, 3, 1, 4, 2, 5}, gather(t, tr, buf))
}

func TestPermuteNegative(t *testing.T) {
	tr, err := mustTracker(t, Shape{2, 3, 4}).Permute(-1, 0, 1)
	require.NoError(t, err)
	require.Equal(t, Shape{4, 2, 3}, tr.Shape())
}

func TestReshape(t *testing.T) {
	tr, err := mustTracker(t, Shape{2, 3}).Reshape(Shape{3, 2})
	require.NoError(t, err)
	require.True(t, tr.Contiguous())
	require.Equal(t, arange(6), gather(t, tr, arange(6)))
	tr, err = mustTracker(t, Shape{2, 3}).Reshape(Shape{-1})
	require.NoError(t, err)
	require.Equal(t, Shape{6}, tr.Shape())
}

func TestPermuteReshape(t *testing.T) {
	// (3,2) T then view as (3,2) is not a single row-major map.
	tr, err := mustTracker(t, Shape{3, 2}).Permute(1, 0)
	require.NoError(t, err)
	tr, err = tr.Reshape(Shape{3, 2})
	require.NoError(t, err)
	require.Equal(t, []int{0, 2, 4, 1, 3, 5}, gather(t, tr, arange(6)))
}

func TestExpand(t *testing.T) {
	tr, err := mustTracker(t, Shape{1, 3}).Expand(Shape{4, 3})
	require.NoError(t, err)
	require.Equal(t, Shape{4, 3}, tr.Shape())
	got := gather(t, tr, arange(3))
	require.Equal(t, []int{0, 1, 2, 0, 1, 2, 0, 1, 2, 0, 1, 2}, got)
}

func TestFlip(t *testing.T) {
	tr, err := mustTracker(t, Shape{2, 3}).Flip(1)
	require.NoError(t, err)
	require.Equal(t, []int{2, 1, 0, 5, 4, 3}, gather(t, tr, arange(6)))
	tr, err = mustTracker(t, Shape{2, 3}).Flip(0)
	require.NoError(t, err)
	require.Equal(t, []int{3, 4, 5, 0, 1, 2}, gather(t, tr, arange(6)))
}

func TestShrink(t *testing.T) {
	tr, err := mustTracker(t, Shape{3, 4}).Shrink([][2]int{{1, 3}, {1, 3}})
	require.NoError(t, err)
	require.Equal(t, Shape{2, 2}, tr.Shape())
	require.Equal(t, []int{5, 6, 9, 10}, gather(t, tr, arange(12)))
}

func TestPad(t *testing.T) {
	tr, err := mustTracker(t, Shape{2, 2}).Pad([][2]int{{1, 0}, {0, 1}})
	require.NoError(t, err)
	require.Equal(t, Shape{3, 3}, tr.Shape())
	require.Equal(t, []int{0, 0, 0, 0, 1, 0, 2, 3, 0}, gather(t, tr, arange(4)))
}

func TestPadThenShrink(t *testing.T) {
	tr, err := mustTracker(t, Shape{2, 2}).Pad([][2]int{{1, 1}, {1, 1}})
	require.NoError(t, err)
	tr, err = tr.Shrink([][2]int{{1, 3}, {1, 3}})
	require.NoError(t, err)
	require.Equal(t, arange(4), gather(t, tr, arange(4)))
}

func TestFlipNegative(t *testing.T) {
	tr, err := mustTracker(t, Shape{2, 3}).Flip(-1)
	require.NoError(t, err)
	require.Equal(t, []int{2, 1, 0, 5, 4, 3}, gather(t, tr, arange(6)))
}

func TestReshapeMergeAfterPermute(t *testing.T) {
	tr, err := mustTracker(t, Shape{2, 3, 4}).Permute(2, 0, 1)
	require.NoError(t, err)
	tr, err = tr.Reshape(Shape{4, 6})
	require.NoError(t, err)
	// same cells as permute then a dense reshape of that layout
	perm, err := mustTracker(t, Shape{2, 3, 4}).Permute(2, 0, 1)
	require.NoError(t, err)
	want := gather(t, perm, arange(24))
	require.Equal(t, want, gather(t, tr, arange(24)))
}

func TestPadReshape(t *testing.T) {
	tr, err := mustTracker(t, Shape{2, 3}).Pad([][2]int{{0, 0}, {1, 1}})
	require.NoError(t, err)
	tr, err = tr.Reshape(Shape{2, 5})
	require.NoError(t, err)
	require.Equal(t, []int{0, 0, 1, 2, 0, 0, 3, 4, 5, 0}, gather(t, tr, arange(6)))
}

func TestErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		err  error
		fn   func() error
	}{
		{"neg shape", ErrShape, func() error { _, err := Of(Shape{-1}); return err }},
		{"reshape size", ErrSize, func() error { _, err := mustTracker(t, Shape{2, 3}).Reshape(Shape{5}); return err }},
		{"reshape two -1", ErrShape, func() error { _, err := mustTracker(t, Shape{2, 3}).Reshape(Shape{-1, -1}); return err }},
		{"permute", ErrAxis, func() error { _, err := mustTracker(t, Shape{2, 3}).Permute(0, 0); return err }},
		{"permute oob", ErrAxis, func() error { _, err := mustTracker(t, Shape{2, 3}).Permute(2, 0); return err }},
		{"expand", ErrExpand, func() error { _, err := mustTracker(t, Shape{2, 3}).Expand(Shape{2, 4}); return err }},
		{"pad", ErrPad, func() error { _, err := mustTracker(t, Shape{2}).Pad([][2]int{{-1, 0}}); return err }},
		{"shrink", ErrShrink, func() error { _, err := mustTracker(t, Shape{2}).Shrink([][2]int{{0, 3}}); return err }},
		{"index rank", ErrIndex, func() error { _, _, err := mustTracker(t, Shape{2, 3}).Index(0); return err }},
		{"index oob", ErrIndex, func() error { _, _, err := mustTracker(t, Shape{2, 3}).Index(2, 0); return err }},
		{"empty tracker", ErrShape, func() error { _, _, err := Tracker{}.Index(); return err }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.fn()
			require.Error(t, err)
			require.True(t, errors.Is(err, tc.err), err)
		})
	}
}

func TestCompose(t *testing.T) {
	tr, err := mustTracker(t, Shape{2, 3, 4}).Permute(1, 2, 0)
	require.NoError(t, err)
	tr, err = tr.Flip(0)
	require.NoError(t, err)
	tr, err = tr.Shrink([][2]int{{0, 2}, {1, 4}, {0, 2}})
	require.NoError(t, err)
	tr, err = tr.Reshape(Shape{2, 6})
	require.NoError(t, err)
	require.Equal(t, Shape{2, 6}, tr.Shape())
	got := gather(t, tr, arange(24))
	// permute (1,2,0) of (2,3,4) is (3,4,2), flip axis 0, shrink to (2,3,2), reshape (2,6)
	perm, err := mustTracker(t, Shape{2, 3, 4}).Permute(1, 2, 0)
	require.NoError(t, err)
	perm, err = perm.Flip(0)
	require.NoError(t, err)
	perm, err = perm.Shrink([][2]int{{0, 2}, {1, 4}, {0, 2}})
	require.NoError(t, err)
	want := gather(t, perm, arange(24))
	require.Equal(t, want, got)
}

func TestIdentity(t *testing.T) {
	tr, err := mustTracker(t, Shape{2, 3}).Permute(0, 1)
	require.NoError(t, err)
	require.True(t, tr.Contiguous())
	tr, err = tr.Reshape(Shape{2, 3})
	require.NoError(t, err)
	require.True(t, tr.Contiguous())
	tr, err = tr.Pad([][2]int{{0, 0}, {0, 0}})
	require.NoError(t, err)
	require.Equal(t, arange(6), gather(t, tr, arange(6)))
}

func TestShapeCopy(t *testing.T) {
	tracker := mustTracker(t, Shape{2, 3})
	shape := tracker.Shape()
	shape[0] = 9
	require.Equal(t, Shape{2, 3}, tracker.Shape())
}
