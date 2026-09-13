package path

import (
	"slices"
	"testing"
	"testing/fstest"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlob(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"a.txt":         {Data: []byte("a")},
		"b.py":          {Data: []byte("b")},
		"dir/c.py":      {Data: []byte("c")},
		"dir/sub/d.py":  {Data: []byte("d")},
		"dir/sub/e.txt": {Data: []byte("e")},
		"dir/sub/f.pyc": {Data: []byte("f")},
	}
	cases := []struct {
		name    string
		root    string
		pattern string
		want    []string
	}{
		{name: "one dir", root: "dir", pattern: "*.py", want: []string{"dir/c.py"}},
		{name: "star star py", root: ".", pattern: "**/*.py", want: []string{"b.py", "dir/c.py", "dir/sub/d.py"}},
		{name: "under dir", root: "dir", pattern: "**/*.py", want: []string{"dir/c.py", "dir/sub/d.py"}},
		{name: "middle", root: ".", pattern: "dir/**/*.txt", want: []string{"dir/sub/e.txt"}},
		{name: "class", root: ".", pattern: "*.[pt]xt", want: []string{"a.txt"}},
		{name: "only dirs", root: ".", pattern: "**/", want: []string{".", "dir", "dir/sub"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, names(test.Collect(t, New(tc.root).Glob(fsys, tc.pattern))))
		})
	}
}

func TestRglob(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"b.py":         {Data: []byte("b")},
		"dir/c.py":     {Data: []byte("c")},
		"dir/sub/d.py": {Data: []byte("d")},
	}
	assert.Equal(t, []string{"dir/c.py", "dir/sub/d.py"}, names(test.Collect(t, New("dir").Rglob(fsys, "*.py"))))
}

func TestGlobStarAll(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"a.txt":    {Data: []byte("a")},
		"dir/b.py": {Data: []byte("b")},
	}
	assert.Equal(t, []string{".", "a.txt", "dir", "dir/b.py"}, names(test.Collect(t, New(".").Glob(fsys, "**"))))
}

func TestGlobDedup(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{"dir/a.txt": {Data: []byte("a")}}
	assert.Equal(t, []string{"dir/a.txt"}, names(test.Collect(t, New(".").Glob(fsys, "**/**/*.txt"))))
}

func TestGlobStop(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"a.py": {Data: []byte("a")},
		"b.py": {Data: []byte("b")},
		"c.py": {Data: []byte("c")},
	}
	n := 0
	for _, err := range New(".").Glob(fsys, "*.py") {
		require.NoError(t, err)
		n++
		if n == 1 {
			break
		}
	}
	assert.Equal(t, 1, n)
}

func TestGlobBadPattern(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{"a.txt": {Data: []byte("a")}}
	for _, pat := range []string{"["} {
		t.Run(pat, func(t *testing.T) {
			t.Parallel()
			var err error
			for _, e := range New(".").Glob(fsys, pat) {
				err = e
				break
			}
			assert.ErrorIs(t, err, doublestar.ErrBadPattern)
		})
	}
}

func TestGlobNoFollowStarStar(t *testing.T) {
	t.Parallel()
	root, err := Open(t.TempDir())
	require.NoError(t, err)
	test.Close(t, root)
	require.NoError(t, New("real").Mkdir(root, 0o755))
	require.NoError(t, New("real", "hit.py").WriteFile(root, nil, 0o644))
	require.NoError(t, New("link").Symlink(root, New("real")))

	assert.Equal(t, []string{"real/hit.py"}, names(test.Collect(t, New(".").Glob(root, "**/*.py"))))
	assert.Equal(t, []string{"real/hit.py"}, names(test.Collect(t, New(".").Glob(root, "*/*.py"))))
}

func names(ps []Path) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.String()
	}
	slices.Sort(out)
	return out
}
