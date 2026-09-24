package sniff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fake struct {
	name string
	exts []string
	mag  [][]byte
}

func (f fake) Name() string         { return f.name }
func (f fake) Extensions() []string { return f.exts }
func (f fake) Magic() [][]byte      { return f.mag }

func TestByExtension(t *testing.T) {
	t.Parallel()
	gz := fake{name: "gzip", exts: []string{".gz", ""}}
	br := fake{name: "brotli", exts: []string{"br"}}
	list := []fake{gz, br}

	got, ok := ByExtension(list, "src.tar.gz")
	require.True(t, ok)
	assert.Equal(t, "gzip", got.Name())

	got, ok = ByExtension(list, "A.TAR.BR")
	require.True(t, ok)
	assert.Equal(t, "brotli", got.Name())

	_, ok = ByExtension(list, "a.tar")
	assert.False(t, ok)

	// Same length: the earlier entry wins.
	tar := fake{name: "tar", exts: []string{".tar"}}
	also := fake{name: "also", exts: []string{".tar"}}
	got, ok = ByExtension([]fake{tar, also}, "a.tar")
	require.True(t, ok)
	assert.Equal(t, "tar", got.Name())
}

func TestByMagic(t *testing.T) {
	t.Parallel()
	short := fake{name: "short", mag: [][]byte{{0x1f}, {}}}
	gz := fake{name: "gzip", mag: [][]byte{{0x1f, 0x8b}}}
	list := []fake{short, gz}

	got, ok := ByMagic(list, []byte{0x1f, 0x8b, 0x08})
	require.True(t, ok)
	assert.Equal(t, "gzip", got.Name())

	got, ok = ByMagic(list, []byte{0x1f, 0x00})
	require.True(t, ok)
	assert.Equal(t, "short", got.Name())

	_, ok = ByMagic(list, []byte{0x00})
	assert.False(t, ok)

	_, ok = ByMagic(list, nil)
	assert.False(t, ok)
}

func TestHasName(t *testing.T) {
	t.Parallel()
	list := []fake{{name: "gzip"}, {name: "brotli"}}
	assert.True(t, HasName(list, "gzip"))
	assert.False(t, HasName(list, "Gzip"))
	assert.False(t, HasName(list, "xz"))
}
