package iso

import (
	"bytes"
	"io"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	when := time.Date(2024, 3, 15, 12, 30, 45, 0, time.UTC)
	payload := append([]byte("MZ\x00\n"), bytes.Repeat([]byte{0x11}, 1000)...)
	file := New().
		Volume("LEWKIT").
		Platform(PlatformEFI).
		Emulation(EmulationNone).
		Boot("EFIBOOT.IMG").
		Entry("EFI/BOOT").Dir().Mode(0o700).Time(when).
		Entry("EFI/BOOT/BOOTX64.EFI").Mode(0o755).Time(when).Bytes(payload).
		Entry("readme.txt").Mode(0o640).Time(when.Add(time.Minute)).Bytes([]byte("hi")).
		Entry("empty").Time(when).Bytes(nil).
		Entry("notes").Dir().Mode(0o755).Time(when).
		Entry("a-very-long-file-name.txt").Mode(0o600).Time(when).Bytes([]byte("long"))

	first := takeMade(t, file)
	require.Equal(t, "LEWKIT", first.Label)
	require.Equal(t, Catalog{
		Platform:  PlatformEFI,
		Emulation: EmulationNone,
		BootFile:  "EFIBOOT.IMG",
	}, first.Catalog)
	require.Equal(t, views(file), views(first))

	second := takeMade(t, first)
	require.Equal(t, views(first), views(second))
	require.Equal(t, first.Label, second.Label)
	require.Equal(t, first.Catalog, second.Catalog)
}

func TestEmptyRoundTrip(t *testing.T) {
	t.Parallel()
	file := New()
	got := takeMade(t, file)
	require.Empty(t, got.Entries)
	require.Equal(t, PlatformEFI, got.Catalog.Platform)
	require.Equal(t, EmulationNone, got.Catalog.Emulation)
	require.Equal(t, "EFIBOOT.IMG", got.Catalog.BootFile)
	require.Equal(t, "LEWKIT", got.Label)
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

	err = New().Entry("EFIBOOT.IMG").Bytes([]byte("nope")).Make(io.Discard)
	require.ErrorIs(t, err, ErrName)

	err = New().Entry("link").Mode(fs.ModeSymlink | 0o777).Make(io.Discard)
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
	Name string
	Mode fs.FileMode
	Unix int64
	Data []byte
}

func views(file *File) []view {
	out := make([]view, len(file.Entries))
	for i, entry := range file.Entries {
		out[i] = view{
			Name: entry.Name,
			Mode: entry.Mode,
			Unix: entry.Time.Unix(),
			Data: entry.Data,
		}
	}
	slices.SortFunc(out, func(a, b view) int {
		return strings.Compare(a.Name, b.Name)
	})
	return out
}
