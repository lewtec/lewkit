package zip

import (
	stdzip "archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestOpenCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := Open(ctx, packZip(t, map[string][]byte{"a.txt": []byte("x")}, stdzip.Store))
	require.ErrorIs(t, err, context.Canceled)
}

func TestNeedReadAt(t *testing.T) {
	t.Parallel()
	_, err := Open(t.Context(), test.OnlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
	pe, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)
}

func TestNeedSize(t *testing.T) {
	t.Parallel()
	_, err := Open(t.Context(), atOnly{b: []byte("PK")})
	require.ErrorIs(t, err, lewfs.ErrNeedSize)
}

func TestInvalidZip(t *testing.T) {
	t.Parallel()
	_, err := Open(t.Context(), bytes.NewReader(make([]byte, 256)))
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
	fsys, err := Open(t.Context(), r)
	require.NoError(t, err)
	test.ArchiveTree(t, fsys)
	_, err = fsys.ReadFile("a")
	require.Error(t, err)
}

func TestStoredReadAt(t *testing.T) {
	t.Parallel()
	r := packZip(t, map[string][]byte{"a.bin": []byte("hello world")}, stdzip.Store)
	fsys, err := Open(t.Context(), r)
	require.NoError(t, err)
	test.ArchiveReadAt(t, fsys)
}

func TestOpenFS(t *testing.T) {
	t.Parallel()
	raw := packZip(t, map[string][]byte{"README": []byte("hi")}, stdzip.Deflate)
	data, err := io.ReadAll(raw)
	require.NoError(t, err)
	host := fstest.MapFS{"src.zip": {Data: data}}
	z, err := path.OpenFS(t.Context(), path.New("src.zip"), host, Open)
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
	fsys, err := Open(t.Context(), r)
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "a/b.txt", "z.txt"))
}

func TestCopyExtract(t *testing.T) {
	t.Parallel()
	r := packZip(t, map[string][]byte{
		"a/b.txt": []byte("hello"),
		"z.txt":   []byte("zee"),
	}, stdzip.Deflate)
	src, err := Open(t.Context(), r)
	require.NoError(t, err)
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	require.NoError(t, lewfs.Copy(t.Context(), dest, lewfs.Walk(t.Context(), src, nil)))
	b, err := path.New("z.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("zee"), b)
	b, err = path.New("a", "b.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), b)
}

func TestWriteReadOnly(t *testing.T) {
	t.Parallel()
	r := packZip(t, map[string][]byte{"a.txt": []byte("x")}, stdzip.Store)
	fsys, err := Open(t.Context(), r)
	require.NoError(t, err)
	err = path.New("a.txt").WriteFile(fsys, []byte("y"), 0o644)
	require.ErrorIs(t, err, path.ErrReadOnly)
}

func TestOpenFSNeedReadAt(t *testing.T) {
	t.Parallel()
	fsys := test.ReaderOnlyFS("a.zip", strings.NewReader("x"))
	_, err := path.OpenFS(t.Context(), path.New("a.zip"), fsys, Open)
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
}
