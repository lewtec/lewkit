package path

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		parts []string
		want  string
	}{
		{name: "empty", want: "."},
		{name: "one", parts: []string{"a"}, want: "a"},
		{name: "join", parts: []string{"a", "b", "c.txt"}, want: "a/b/c.txt"},
		{name: "dot", parts: []string{".", "a"}, want: "a"},
		{name: "dotdot", parts: []string{"a", "..", "b"}, want: "b"},
		{name: "abs wins", parts: []string{"a", "/b", "c"}, want: "/b/c"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, New(tc.parts...).String())
		})
	}
}

func TestLexical(t *testing.T) {
	t.Parallel()
	p := New("dir", "a.tar.gz")
	assert.Equal(t, "a.tar.gz", p.Name())
	assert.Equal(t, "a.tar", p.Stem())
	assert.Equal(t, ".gz", p.Suffix())
	assert.Equal(t, []string{".tar", ".gz"}, p.Suffixes())
	assert.Equal(t, []string{"dir", "a.tar.gz"}, p.Parts())
	assert.Equal(t, "dir", p.Parent().String())
	assert.True(t, p.Valid())
	assert.False(t, p.IsAbs())

	dot := New()
	assert.Equal(t, ".", dot.String())
	assert.Equal(t, ".", dot.Parent().String())
	assert.Equal(t, []string{"."}, dot.Parts())
	assert.True(t, dot.Valid())

	abs := New("/etc", "passwd")
	assert.True(t, abs.IsAbs())
	assert.False(t, abs.Valid())
	assert.Equal(t, []string{"/", "etc", "passwd"}, abs.Parts())

	hidden := New(".bashrc")
	assert.Equal(t, "", hidden.Suffix())
	assert.Equal(t, ".bashrc", hidden.Stem())
	assert.Empty(t, hidden.Suffixes())
}

func TestWith(t *testing.T) {
	t.Parallel()
	p := New("dir", "a.tar.gz")
	assert.Equal(t, "dir/b.txt", p.WithName("b.txt").String())
	assert.Equal(t, "dir/x.gz", p.WithStem("x").String())
	assert.Equal(t, "dir/a.tar.xz", p.WithSuffix(".xz").String())
}

func TestMatch(t *testing.T) {
	t.Parallel()
	ok, err := New("a/b.go").Match("*.go")
	require.NoError(t, err)
	assert.False(t, ok)
	ok, err = New("a/b.go").Match("a/*.go")
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestRel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		p, b string
		want string
		err  bool
	}{
		{name: "same", p: "a/b", b: "a/b", want: "."},
		{name: "child", p: "a/b/c", b: "a", want: "b/c"},
		{name: "walk up", p: "a/c", b: "a/b", want: "../c"},
		{name: "root", p: "a", b: ".", want: "a"},
		{name: "mixed", p: "/a", b: "a", err: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := New(tc.p).Rel(New(tc.b))
			if tc.err {
				assert.ErrorIs(t, err, ErrRel)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got.String())
		})
	}
}

func TestJoin(t *testing.T) {
	t.Parallel()
	p := New("a")
	assert.Equal(t, p, p.Join())
	assert.Equal(t, "a/b/c", p.Join("b", "c").String())
}
