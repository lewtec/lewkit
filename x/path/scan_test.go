package path

import (
	"io"
	"iter"
	"testing"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pathSeq(names ...string) iter.Seq2[Path, error] {
	return func(yield func(Path, error) bool) {
		for _, n := range names {
			if !yield(New(n), nil) {
				return
			}
		}
	}
}

func TestMatchGlob(t *testing.T) {
	t.Parallel()
	ok, err := New("dir/c.py").MatchGlob("**/*.py")
	require.NoError(t, err)
	assert.True(t, ok)
	ok, err = New("dir/c.py").MatchGlob("*.py")
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestSelect(t *testing.T) {
	t.Parallel()
	seq := pathSeq("a.txt", "b.py", "dir/c.py", "dir/sub/d.py", "dir/sub/e.txt")
	cases := []struct {
		name    string
		root    string
		pattern string
		want    []string
	}{
		{name: "one dir", root: "dir", pattern: "*.py", want: []string{"dir/c.py"}},
		{name: "star star py", root: ".", pattern: "**/*.py", want: []string{"b.py", "dir/c.py", "dir/sub/d.py"}},
		{name: "under dir", root: "dir", pattern: "**/*.py", want: []string{"dir/c.py", "dir/sub/d.py"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, names(test.Collect(t, New(tc.root).Select(seq, tc.pattern))))
		})
	}
}

func TestSelectBadPattern(t *testing.T) {
	t.Parallel()
	var err error
	for _, e := range New(".").Select(pathSeq("a.txt"), "[") {
		err = e
		break
	}
	assert.ErrorIs(t, err, doublestar.ErrBadPattern)
}

func TestSelectError(t *testing.T) {
	t.Parallel()
	seq := func(yield func(Path, error) bool) {
		yield(Path{}, io.ErrUnexpectedEOF)
	}
	var err error
	for _, e := range New(".").Select(seq, "*.txt") {
		err = e
		break
	}
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

func TestUnder(t *testing.T) {
	t.Parallel()
	seq := pathSeq("a.txt", "dir/c.py", "dir/sub/d.py", "other/x.txt")
	assert.Equal(t, []string{"dir/c.py", "dir/sub/d.py"}, names(test.Collect(t, New("dir").Under(seq))))
	assert.Equal(t, []string{"a.txt", "dir/c.py", "dir/sub/d.py", "other/x.txt"}, names(test.Collect(t, New(".").Under(seq))))
}
