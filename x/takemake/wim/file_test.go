package wim

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	when := time.Date(2024, 3, 15, 12, 30, 45, 0, time.UTC)
	payload := bytes.Repeat([]byte{0x11}, 5000)
	file := New().Image("kit & co").
		Entry("dir").Dir().Mode(fs.ModeDir | 0o700 | fs.ModeSticky).Time(when).
		Entry("dir/nested").Dir().Mode(0o755).Time(when).
		Entry("dir/nested/big.bin").Mode(0o640 | fs.ModeSetuid).Time(when).Bytes(payload).
		Entry("readme.txt").Mode(0o640).Time(when.Add(time.Minute)).Bytes([]byte("hi")).
		Entry("empty").Time(when).Bytes(nil).
		Entry("link").Time(when).Symlink("dir/nested/big.bin").
		Entry("empty-dir").Dir().Time(when).
		Entry("same").Time(when).Bytes([]byte("hi"))

	first := takeMade(t, file)
	require.Equal(t, "kit & co", first.ImageName)
	require.Equal(t, views(file), views(first))
	second := takeMade(t, first)
	require.Equal(t, views(first), views(second))
	require.Equal(t, first.ImageName, second.ImageName)
}

func TestManyEntries(t *testing.T) {
	t.Parallel()
	when := time.Date(2021, 5, 6, 7, 8, 9, 0, time.UTC)
	file := New().Image("many")
	for i := range 200 {
		file.Entry("batch/n" + strconv.Itoa(i)).Mode(0o640).Time(when).Bytes([]byte{byte(i)})
	}
	first := takeMade(t, file)
	require.Equal(t, "many", first.ImageName)
	require.Equal(t, views(file), views(first))
}

// testdata/empty_dacl.wim is wimlib's tests/wims/empty_dacl.wim.
// It is one uncompressed image with an empty DACL and a single file.
func TestEmptyDACL(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/empty_dacl.wim")
	require.NoError(t, err)
	file, err := Take(bytes.NewReader(raw))
	require.NoError(t, err)
	require.Equal(t, "loop3p2", file.ImageName)
	require.Len(t, file.Entries, 1)
	got := file.Entries[0]
	require.Equal(t, "file", got.Name)
	require.Equal(t, fs.FileMode(0o644), got.Mode)
	require.Equal(t, []byte("1\n"), got.Data)
	require.True(t, time.Date(2013, 9, 16, 20, 50, 8, 460728100, time.UTC).Equal(got.Time))

	again := takeMade(t, file)
	require.Equal(t, file.ImageName, again.ImageName)
	require.Equal(t, views(file), views(again))
}

func TestEmptyRoundTrip(t *testing.T) {
	t.Parallel()
	got := takeMade(t, New())
	require.Empty(t, got.Entries)
	require.Equal(t, defaultName, got.ImageName)
}

func TestRejects(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrNilReader)
	require.ErrorIs(t, err, errs.ErrInvalid)

	require.ErrorIs(t, New().Make(nil), ErrNilWriter)
	require.ErrorIs(t, (*File)(nil).Make(io.Discard), ErrNilFile)

	err = New().Entry("..").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)
	err = New().Entry("a:b").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)
	err = New().Entry("link").Symlink("").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)
	err = New().Image("").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)

	err = New().Entry("gone").Bytes([]byte("x")).Entry("gone/child").Make(io.Discard)
	require.ErrorIs(t, err, ErrKind)

	_, err = Take(bytes.NewReader([]byte("MSWIM no")))
	require.ErrorIs(t, err, ErrFormat)
	require.ErrorIs(t, err, errs.ErrSyntax)
}

func TestEntryStoresTheChild(t *testing.T) {
	t.Parallel()
	when := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	file := New().Entry("dir/file.txt").Mode(0o640).Time(when).Bytes([]byte("abc"))
	require.Equal(t, defaultName, file.ImageName)
	require.Len(t, file.Entries, 2)
	require.Equal(t, "dir", file.Entries[0].Name)
	require.Equal(t, fs.ModeDir|0o755, file.Entries[0].Mode)
	got := file.Entries[1]
	require.Equal(t, "dir/file.txt", got.Name)
	require.Equal(t, fs.FileMode(0o640), got.Mode)
	require.True(t, when.Equal(got.Time))
	require.Equal(t, []byte("abc"), got.Data)

	link := New().Entry("to").Symlink(`..\out`)
	require.Equal(t, fs.ModeSymlink|linkMode, link.Entries[0].Mode)
	require.Equal(t, `..\out`, link.Entries[0].Target)
}

func takeMade(t *testing.T, file *File) *File {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	got, err := Take(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	return got
}

type view struct {
	Name   string
	Mode   fs.FileMode
	Unix   int64
	Data   []byte
	Target string
}

func views(file *File) []view {
	out := make([]view, len(file.Entries))
	for i, entry := range file.Entries {
		out[i] = view{
			Name:   entry.Name,
			Mode:   entry.Mode,
			Unix:   entry.Time.Unix(),
			Data:   entry.Data,
			Target: entry.Target,
		}
	}
	slices.SortFunc(out, func(a, b view) int {
		return strings.Compare(a.Name, b.Name)
	})
	return out
}
