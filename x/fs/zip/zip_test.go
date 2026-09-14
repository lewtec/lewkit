package zip

import (
	stdzip "archive/zip"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/path"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type onlyReader struct{ io.Reader }

type atOnly struct{ b []byte }

func (a atOnly) Read(p []byte) (int, error) {
	n := copy(p, a.b)
	if n == 0 {
		return 0, io.EOF
	}
	return n, io.EOF
}

func (a atOnly) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(len(a.b)) {
		return 0, io.EOF
	}
	n := copy(p, a.b[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

func packZip(t *testing.T, files map[string][]byte, method uint16) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := stdzip.NewWriter(&buf)
	for name, data := range files {
		h := &stdzip.FileHeader{Name: name, Method: method}
		fw, err := w.CreateHeader(h)
		require.NoError(t, err)
		_, err = fw.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return bytes.NewReader(buf.Bytes())
}

func TestNeedReadAt(t *testing.T) {
	t.Parallel()
	_, err := Open(onlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
	pe, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)
}

func TestNeedSize(t *testing.T) {
	t.Parallel()
	_, err := Open(atOnly{b: []byte("PK")})
	require.ErrorIs(t, err, lewfs.ErrNeedSize)
}

func TestInvalidZip(t *testing.T) {
	t.Parallel()
	_, err := Open(bytes.NewReader(make([]byte, 256)))
	require.Error(t, err)
	assert.False(t, errors.Is(err, lewfs.ErrNeedReadAt))
	assert.False(t, errors.Is(err, lewfs.ErrNeedSize))
}

func TestTree(t *testing.T) {
	t.Parallel()
	r := packZip(t, map[string][]byte{
		"a/b.txt": []byte("hello"),
		"z.txt":   []byte("zee"),
	}, stdzip.Deflate)
	fsys, err := Open(r)
	require.NoError(t, err)

	ents, err := fsys.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, ents, 2)
	assert.Equal(t, "a", ents[0].Name())
	assert.True(t, ents[0].IsDir())
	assert.Equal(t, "z.txt", ents[1].Name())
	assert.False(t, ents[1].IsDir())

	ents, err = fsys.ReadDir("a")
	require.NoError(t, err)
	require.Len(t, ents, 1)
	assert.Equal(t, "b.txt", ents[0].Name())

	b, err := fsys.ReadFile("a/b.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))

	st, err := fsys.Stat("a")
	require.NoError(t, err)
	assert.True(t, st.IsDir())

	_, err = fsys.Open("missing")
	require.ErrorIs(t, err, fs.ErrNotExist)

	_, err = fsys.Open("../x")
	require.ErrorIs(t, err, fs.ErrInvalid)

	_, err = fsys.ReadFile("a")
	require.Error(t, err)
}

func TestStoredReadAt(t *testing.T) {
	t.Parallel()
	r := packZip(t, map[string][]byte{"a.bin": []byte("hello world")}, stdzip.Store)
	fsys, err := Open(r)
	require.NoError(t, err)
	f, err := fsys.Open("a.bin")
	require.NoError(t, err)
	t.Cleanup(func() { f.Close() })
	ra, ok := f.(io.ReaderAt)
	require.True(t, ok)
	buf := make([]byte, 5)
	n, err := ra.ReadAt(buf, 6)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "world", string(buf))
}

func TestOpenFS(t *testing.T) {
	t.Parallel()
	raw := packZip(t, map[string][]byte{"README": []byte("hi")}, stdzip.Deflate)
	data, err := io.ReadAll(raw)
	require.NoError(t, err)
	host := fstest.MapFS{"src.zip": {Data: data}}
	z, err := path.OpenFS(path.New("src.zip"), host, Open)
	require.NoError(t, err)
	b, err := path.New("README").ReadFile(z)
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), b)
}

func TestFS(t *testing.T) {
	t.Parallel()
	r := packZip(t, map[string][]byte{
		"a/b.txt": []byte("hello"),
		"z.txt":   []byte("zee"),
	}, stdzip.Deflate)
	fsys, err := Open(r)
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "a/b.txt", "z.txt"))
}

func TestWriteReadOnly(t *testing.T) {
	t.Parallel()
	r := packZip(t, map[string][]byte{"a.txt": []byte("x")}, stdzip.Store)
	fsys, err := Open(r)
	require.NoError(t, err)
	err = path.New("a.txt").WriteFile(fsys, []byte("y"), 0o644)
	require.ErrorIs(t, err, path.ErrReadOnly)
}

func TestOpenFSNeedReadAt(t *testing.T) {
	t.Parallel()
	fsys := &readerOnlyFS{name: "a.zip", r: strings.NewReader("x")}
	_, err := path.OpenFS(path.New("a.zip"), fsys, Open)
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
}

type readerOnlyFS struct {
	name string
	r    io.Reader
}

func (s *readerOnlyFS) Open(name string) (fs.File, error) {
	if name != s.name {
		return nil, fs.ErrNotExist
	}
	return readerOnlyFile{Reader: s.r}, nil
}

type readerOnlyFile struct{ io.Reader }

func (readerOnlyFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (readerOnlyFile) Close() error               { return nil }
