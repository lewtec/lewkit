package compression

import (
	"io"
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
func (f fake) Reader(io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(nil), nil
}

func TestByExtensionLongest(t *testing.T) {
	t.Parallel()
	gz := fake{name: "gzip", exts: []string{".gz", ".tgz", ".tar.gz"}}
	br := fake{name: "brotli", exts: []string{".br", ".tar.br"}}
	reg := New(gz, br)

	c, ok := reg.ByExtension("src.tar.gz")
	require.True(t, ok)
	assert.Equal(t, "gzip", c.Name())

	c, ok = reg.ByExtension("SRC.TGZ")
	require.True(t, ok)
	assert.Equal(t, "gzip", c.Name())

	c, ok = reg.ByExtension("a.tar.br")
	require.True(t, ok)
	assert.Equal(t, "brotli", c.Name())

	_, ok = reg.ByExtension("a.tar")
	assert.False(t, ok)
}

func TestByMagicLongest(t *testing.T) {
	t.Parallel()
	short := fake{name: "short", mag: [][]byte{{0x1f}}}
	gz := fake{name: "gzip", mag: [][]byte{{0x1f, 0x8b}}}
	reg := New(short, gz)
	c, ok := reg.ByMagic([]byte{0x1f, 0x8b, 0x08})
	require.True(t, ok)
	assert.Equal(t, "gzip", c.Name())
}

func TestDetectPrefersExtension(t *testing.T) {
	t.Parallel()
	gz := fake{name: "gzip", exts: []string{".gz"}, mag: [][]byte{{0x1f, 0x8b}}}
	br := fake{name: "brotli", exts: []string{".br"}}
	reg := New(gz, br)
	c, ok := reg.Detect("x.br", []byte{0x1f, 0x8b})
	require.True(t, ok)
	assert.Equal(t, "brotli", c.Name())
}

func TestDetectMagicWhenNoName(t *testing.T) {
	t.Parallel()
	gz := fake{name: "gzip", exts: []string{".gz"}, mag: [][]byte{{0x1f, 0x8b}}}
	reg := New(gz)
	c, ok := reg.Detect("", []byte{0x1f, 0x8b, 0x08})
	require.True(t, ok)
	assert.Equal(t, "gzip", c.Name())
}
