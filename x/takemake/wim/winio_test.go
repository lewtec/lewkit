//go:build linux || windows

package wim

import (
	"bytes"
	"io"
	"testing"

	winwim "github.com/Microsoft/go-winio/wim"
	"github.com/stretchr/testify/require"
)

func TestWinIOReadsImage(t *testing.T) {
	t.Parallel()
	file := New().Image("demo").
		Entry("a").Dir().
		Entry("a/b.txt").Bytes([]byte("hello\n")).
		Entry("z.txt").Bytes(nil).
		Entry("link").Symlink("a/b.txt")
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))

	rd, err := winwim.NewReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	t.Cleanup(func() { _ = rd.Close() })
	require.Len(t, rd.Image, 1)
	require.Equal(t, "demo", rd.Image[0].Name)

	root, err := rd.Image[0].Open()
	require.NoError(t, err)
	got := map[string]string{}
	walkWin(t, root, "", got)
	require.Equal(t, "dir", got["a"])
	require.Equal(t, "hello\n", got["a/b.txt"])
	require.Equal(t, "", got["z.txt"])
	require.Equal(t, "link", got["link"])

	back, err := Take(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	require.Equal(t, "demo", back.ImageName)
	require.Equal(t, "a/b.txt", linkTarget(t, back, "link"))
}

func walkWin(t *testing.T, f *winwim.File, prefix string, out map[string]string) {
	t.Helper()
	kids, err := f.Readdir()
	require.NoError(t, err)
	for _, kid := range kids {
		name := kid.Name
		if prefix != "" {
			name = prefix + "/" + name
		}
		switch {
		case kid.IsDir():
			out[name] = "dir"
			walkWin(t, kid, name, out)
		case kid.ReparseTag != 0:
			require.Equal(t, uint32(reparseTag), kid.ReparseTag)
			body, err := kid.Open()
			require.NoError(t, err)
			raw, err := io.ReadAll(body)
			require.NoError(t, err)
			require.NoError(t, body.Close())
			require.NotEmpty(t, raw)
			out[name] = "link"
		default:
			body, err := kid.Open()
			require.NoError(t, err)
			raw, err := io.ReadAll(body)
			require.NoError(t, err)
			require.NoError(t, body.Close())
			out[name] = string(raw)
		}
	}
}

func linkTarget(t *testing.T, file *File, name string) string {
	t.Helper()
	for _, entry := range file.Entries {
		if entry.Name == name {
			return entry.Target
		}
	}
	t.Fatalf("missing %s", name)
	return ""
}
