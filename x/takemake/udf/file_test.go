package udf

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"testing"
	"time"

	xudf "github.com/Xmister/udf"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	when := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	file := New().
		Volume("LEWKIT").
		Entry("dir").Dir().Mode(0o750).Time(when).
		Entry("dir/héllo.txt").Mode(0o640).Time(when).Bytes([]byte("café")).
		Entry("dir/empty").Time(when).Bytes(nil).
		Entry("dir/blob").Mode(0o600).Time(when.Add(time.Minute)).Bytes(bytes.Repeat([]byte{0xab}, 3000)).
		Entry("link").Symlink("../dir/héllo.txt").
		Entry("abs").Symlink("/tmp/x").
		Entry("dot").Symlink("./here")
	for i := range 50 {
		file.Entry(fmt.Sprintf("n/%02d", i)).Mode(0o644).Time(when).Bytes([]byte{byte(i)})
	}

	raw := takeMade(t, file)
	again := takeMade(t, file)
	require.Equal(t, raw, again)

	got, err := Take(bytes.NewReader(raw))
	require.NoError(t, err)
	require.Equal(t, "LEWKIT", got.Label)
	require.Equal(t, views(file), views(got))

	var replay bytes.Buffer
	require.NoError(t, got.Make(&replay))
	require.Equal(t, raw, replay.Bytes())

	volume, err := xudf.NewUdfFromReader(bytes.NewReader(raw))
	require.NoError(t, err)
	names := map[string][]byte{}
	modes := map[string]fs.FileMode{}
	readXmister(t, volume, nil, "", names, modes)
	require.Equal(t, fs.ModeDir|0o750, modes["dir"])
	require.Equal(t, fs.FileMode(0o640), modes["dir/héllo.txt"])
	require.Equal(t, []byte("café"), names["dir/héllo.txt"])
	require.Empty(t, names["dir/empty"])
	require.Equal(t, bytes.Repeat([]byte{0xab}, 3000), names["dir/blob"])
	require.Equal(t, fs.FileMode(0o777), modes["link"])
	require.Equal(t, fs.ModeDir|0o755, modes["n"])
	require.Equal(t, []byte{7}, names["n/07"])
}

func TestEmptyRoundTrip(t *testing.T) {
	t.Parallel()
	file := New()
	raw := takeMade(t, file)
	got, err := Take(bytes.NewReader(raw))
	require.NoError(t, err)
	require.Empty(t, got.Entries)
	require.Equal(t, "LEWKIT", got.Label)
	var replay bytes.Buffer
	require.NoError(t, got.Make(&replay))
	require.Equal(t, raw, replay.Bytes())
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

	err = New().Entry("link").Symlink("").Make(io.Discard)
	require.ErrorIs(t, err, ErrSymlink)

	err = New().Entry("dev").Mode(fs.ModeDevice | 0o644).Make(io.Discard)
	require.ErrorIs(t, err, ErrKind)

	_, err = Take(bytes.NewReader([]byte("nope")))
	require.ErrorIs(t, err, ErrFormat)
	require.ErrorIs(t, err, errs.ErrSyntax)
}

func TestEntryStoresTheChild(t *testing.T) {
	t.Parallel()
	when := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	file := New().Entry("dir/file.txt").Mode(0o640).Time(when).Bytes([]byte("abc"))
	require.Len(t, file.Entries, 2)
	require.Equal(t, "dir", file.Entries[0].Name)
	require.Equal(t, fs.ModeDir|0o755, file.Entries[0].Mode)
	got := file.Entries[1]
	require.Equal(t, "dir/file.txt", got.Name)
	require.Equal(t, fs.FileMode(0o640), got.Mode)
	require.True(t, when.Equal(got.Time))
	require.Equal(t, []byte("abc"), got.Data)

	link := New().Entry("a").Symlink("../b")
	require.Equal(t, fs.ModeSymlink|0o777, link.Entries[0].Mode)
	require.Equal(t, "../b", link.Entries[0].Target)
}

func takeMade(t *testing.T, file *File) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	return buf.Bytes()
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
	return out
}

func readXmister(t *testing.T, volume *xudf.Udf, fe xudf.FileEntryInterface, prefix string, names map[string][]byte, modes map[string]fs.FileMode) {
	t.Helper()
	for _, entry := range volume.ReadDir(fe) {
		name := entry.Name()
		if prefix != "" {
			name = prefix + "/" + name
		}
		modes[name] = entry.Mode()
		if entry.IsDir() {
			readXmister(t, volume, entry.FileEntry(), name, names, modes)
			continue
		}
		body, err := io.ReadAll(entry.NewReader())
		require.NoError(t, err)
		names[name] = body
	}
}
