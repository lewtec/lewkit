package tar

import (
	"bytes"
	"hash/crc32"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	when := time.Date(2024, 3, 15, 12, 30, 45, 123456789, time.UTC)
	payload := bytes.Repeat([]byte{0x11}, 4000)
	file := New().
		Entry("dir").Dir().Mode(fs.ModeDir | 0o700 | fs.ModeSticky).Time(when).
		Entry("dir/nested").Dir().Mode(0o755).Time(when).
		Entry("dir/nested/big.bin").Mode(0o640 | fs.ModeSetuid).Time(when).Bytes(payload).
		Entry("readme.txt").Mode(0o640 | fs.ModeSetgid).Time(when.Add(time.Minute)).Bytes([]byte("hi")).
		Entry("empty").Time(when).Bytes(nil).
		Entry("link").Time(when).Symlink("dir/nested/big.bin").
		Entry("empty-dir").Dir().Time(when)

	first := takeMade(t, file)
	require.Equal(t, views(file), views(first))
	second := takeMade(t, first)
	require.Equal(t, views(first), views(second))
}

func TestEmptyRoundTrip(t *testing.T) {
	t.Parallel()
	got := takeMade(t, New())
	require.Empty(t, got.Entries)
}

func TestEntryStoresTheChild(t *testing.T) {
	t.Parallel()
	when := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	file := New().Entry("dir/file.txt").Mode(0o640).Time(when).Bytes([]byte("abc"))
	require.Len(t, file.Entries, 2)
	require.Equal(t, "dir", file.Entries[0].Name)
	require.Equal(t, fs.ModeDir|dirMode, file.Entries[0].Mode)
	got := file.Entries[1]
	require.Equal(t, "dir/file.txt", got.Name)
	require.Equal(t, fs.FileMode(0o640), got.Mode)
	require.True(t, when.Equal(got.Time))
	require.Equal(t, []byte("abc"), got.Data)

	link := New().Entry("to").Symlink("../out")
	require.Equal(t, fs.ModeSymlink|linkMode, link.Entries[0].Mode)
	require.Equal(t, "../out", link.Entries[0].Target)

	updated := New().Entry("a").Bytes([]byte("one")).Entry("b").Bytes([]byte("two")).Entry("a").Bytes([]byte("three"))
	require.Equal(t, []string{"a", "b"}, names(updated))
	require.Equal(t, []byte("three"), updated.Entries[0].Data)
}

func TestManyEntries(t *testing.T) {
	t.Parallel()
	when := time.Date(2021, 5, 6, 7, 8, 9, 0, time.UTC)
	file := New()
	for i := range 200 {
		file.Entry("batch/n" + strconv.Itoa(i)).Mode(0o640).Time(when).Bytes([]byte{byte(i)})
	}
	first := takeMade(t, file)
	require.Equal(t, views(file), views(first))
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
	err = New().Entry("/abs").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)
	err = New().Entry(`a\b`).Make(io.Discard)
	require.ErrorIs(t, err, ErrName)
	err = New().Entry("link").Symlink("").Make(io.Discard)
	require.ErrorIs(t, err, ErrName)

	err = New().Entry("gone").Bytes([]byte("x")).Entry("gone/child").Make(io.Discard)
	require.ErrorIs(t, err, ErrKind)

	err = New().Entry("dev").Mode(fs.ModeDevice | 0o600).Make(io.Discard)
	require.ErrorIs(t, err, ErrEntry)
	require.ErrorIs(t, err, errs.ErrUnsupported)

	_, err = Take(bytes.NewReader([]byte("nope")))
	require.ErrorIs(t, err, ErrFormat)
	require.ErrorIs(t, err, errs.ErrSyntax)
}

func TestGNUTree(t *testing.T) {
	t.Parallel()
	when := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	file := mustTake(t, "testdata/gnu-tree.tar")
	requireEntries(t, file, treeEntries(when, []byte("hello tar")))
	again := takeMade(t, file)
	require.Equal(t, views(file), views(again))
	requireTarLists(t, file)
}

func TestPAXTree(t *testing.T) {
	t.Parallel()
	when := time.Date(2020, 1, 2, 3, 4, 5, 123456789, time.UTC)
	file := mustTake(t, "testdata/pax-tree.tar")
	requireEntries(t, file, treeEntries(when, []byte("hello tar")))
	again := takeMade(t, file)
	require.Equal(t, views(file), views(again))
}

func TestV7Tree(t *testing.T) {
	t.Parallel()
	when := time.Date(1999, 12, 31, 23, 59, 58, 0, time.UTC)
	file := mustTake(t, "testdata/v7-tree.tar")
	requireEntries(t, file, []*Entry{
		{Name: "sub", Mode: fs.ModeDir | 0o755, Time: when},
		{Name: "sub/a.txt", Mode: 0o444, Time: when, Data: []byte("v7")},
	})
	again := takeMade(t, file)
	require.Equal(t, views(file), views(again))
}

func TestGoGNU(t *testing.T) {
	t.Parallel()
	file := mustTake(t, "testdata/go-gnu.tar")
	requireEntries(t, file, []*Entry{
		{Name: "small.txt", Mode: 0o640, Time: time.Unix(1244428340, 0).UTC(), Data: []byte("Kilts")},
		{Name: "small2.txt", Mode: 0o640, Time: time.Unix(1244436044, 0).UTC(), Data: []byte("Google.com\n")},
	})
	again := takeMade(t, file)
	require.Equal(t, views(file), views(again))
}

func TestGoPAX(t *testing.T) {
	t.Parallel()
	file := mustTake(t, "testdata/go-pax.tar")
	require.Len(t, file.Entries, 2)
	long := file.Entries[0]
	require.Greater(t, len(long.Name), 100)
	require.True(t, strings.HasPrefix(long.Name, "a/"))
	require.Equal(t, fs.FileMode(0o664), long.Mode)
	require.True(t, time.Unix(1350244992, 23960108).Equal(long.Time))
	require.Equal(t, []byte("shaner\n"), long.Data)
	link := file.Entries[1]
	require.Equal(t, "a/b", link.Name)
	require.Equal(t, fs.ModeSymlink|0o777, link.Mode)
	require.True(t, time.Unix(1350266320, 910238425).Equal(link.Time))
	require.Equal(t, strings.TrimPrefix(long.Name, "a/"), link.Target)
	again := takeMade(t, file)
	require.Equal(t, views(file), views(again))
}

func TestGoV7AndStar(t *testing.T) {
	t.Parallel()
	v7 := mustTake(t, "testdata/go-v7.tar")
	requireEntries(t, v7, []*Entry{
		{Name: "small.txt", Mode: 0o444, Time: time.Unix(1244593104, 0).UTC(), Data: []byte("Kilts")},
		{Name: "small2.txt", Mode: 0o444, Time: time.Unix(1244593104, 0).UTC(), Data: []byte("Google.com\n")},
	})
	require.Equal(t, views(v7), views(takeMade(t, v7)))

	star := mustTake(t, "testdata/go-star.tar")
	requireEntries(t, star, []*Entry{
		{Name: "small.txt", Mode: 0o640, Time: time.Unix(1244592783, 0).UTC(), Data: []byte("Kilts")},
		{Name: "small2.txt", Mode: 0o640, Time: time.Unix(1244592783, 0).UTC(), Data: []byte("Google.com\n")},
	})
	require.Equal(t, views(star), views(takeMade(t, star)))
}

func TestGoSparse(t *testing.T) {
	t.Parallel()
	file := mustTake(t, "testdata/go-sparse.tar")
	require.Equal(t, []string{
		"sparse-gnu",
		"sparse-posix-0.0",
		"sparse-posix-0.1",
		"sparse-posix-1.0",
		"end",
	}, names(file))
	var logical []byte
	for _, entry := range file.Entries[:4] {
		require.Equal(t, fs.FileMode(0o644), entry.Mode)
		require.Len(t, entry.Data, 200)
		require.Equal(t, uint32(0x5375e1d2), crc32.ChecksumIEEE(entry.Data))
		if logical == nil {
			logical = entry.Data
		}
		require.Equal(t, logical, entry.Data)
	}
	require.Equal(t, []byte("end\n"), file.Entries[4].Data)
	require.Equal(t, uint32(0x8eb179ba), crc32.ChecksumIEEE(file.Entries[4].Data))
	require.Equal(t, views(file), views(takeMade(t, file)))
}

func TestLiveGNUTar(t *testing.T) {
	t.Parallel()
	bin := test.Need(t, "tar")
	dir := t.TempDir()
	when := time.Date(2020, 1, 2, 3, 4, 5, 123456789, time.UTC)
	require.NoError(t, os.Mkdir(filepath.Join(dir, "dir"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dir", "note.txt"), []byte("live"), 0o640))
	require.NoError(t, os.Symlink("dir/note.txt", filepath.Join(dir, "link")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "世界.txt"), []byte("utf8"), 0o644))
	stamp := when.Format("2006-01-02 15:04:05.000000000") + " UTC"
	for _, name := range []string{"dir", "dir/note.txt", "link", "世界.txt"} {
		touch := exec.Command("touch", "-h", "-d", stamp, filepath.Join(dir, name))
		out, err := touch.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	archive := filepath.Join(dir, "live.tar")
	cmd := exec.Command(bin, "--format=pax", "--sort=name", "-C", dir, "-cf", archive, "dir", "link", "世界.txt")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	raw, err := os.ReadFile(archive)
	require.NoError(t, err)
	file, err := Take(bytes.NewReader(raw))
	require.NoError(t, err)
	requireEntries(t, file, []*Entry{
		{Name: "dir", Mode: fs.ModeDir | 0o750, Time: when},
		{Name: "dir/note.txt", Mode: 0o640, Time: when, Data: []byte("live")},
		{Name: "link", Mode: fs.ModeSymlink | 0o777, Time: when, Target: "dir/note.txt"},
		{Name: "世界.txt", Mode: 0o644, Time: when, Data: []byte("utf8")},
	})
	require.Equal(t, views(file), views(takeMade(t, file)))
	requireTarLists(t, file)
}

func TestGNUTarHardLink(t *testing.T) {
	t.Parallel()
	bin := test.Need(t, "tar")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o644))
	require.NoError(t, os.Link(filepath.Join(dir, "a"), filepath.Join(dir, "b")))
	archive := filepath.Join(dir, "hard.tar")
	cmd := exec.Command(bin, "--format=gnu", "-C", dir, "-cf", archive, "a", "b")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	raw, err := os.ReadFile(archive)
	require.NoError(t, err)
	_, err = Take(bytes.NewReader(raw))
	require.ErrorIs(t, err, ErrEntry)
	require.ErrorIs(t, err, errs.ErrUnsupported)
}

func TestGNUTarDuplicateMembers(t *testing.T) {
	t.Parallel()
	bin := test.Need(t, "tar")
	dir := t.TempDir()
	name := filepath.Join(dir, "same.txt")
	require.NoError(t, os.WriteFile(name, []byte("one"), 0o644))
	archive := filepath.Join(dir, "dup.tar")
	cmd := exec.Command(bin, "--format=gnu", "-C", dir, "-cf", archive, "same.txt")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	require.NoError(t, os.WriteFile(name, []byte("two"), 0o644))
	cmd = exec.Command(bin, "--format=gnu", "-C", dir, "-rf", archive, "same.txt")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	raw, err := os.ReadFile(archive)
	require.NoError(t, err)
	file, err := Take(bytes.NewReader(raw))
	require.NoError(t, err)
	require.Len(t, file.Entries, 2)
	require.Equal(t, "same.txt", file.Entries[0].Name)
	require.Equal(t, "same.txt", file.Entries[1].Name)
	require.Equal(t, []byte("one"), file.Entries[0].Data)
	require.Equal(t, []byte("two"), file.Entries[1].Data)
	require.Equal(t, views(file), views(takeMade(t, file)))
}

func treeEntries(when time.Time, hello []byte) []*Entry {
	return []*Entry{
		{Name: "dir", Mode: fs.ModeDir | 0o750, Time: when},
		{Name: "dir/bin.dat", Mode: 0o600, Time: when, Data: []byte{'a', 0xff, 0x00, 'b'}},
		{Name: "dir/empty", Mode: 0o644, Time: when},
		{Name: "dir/nested", Mode: fs.ModeDir | 0o750, Time: when},
		{Name: "dir/nested/file.txt", Mode: 0o640, Time: when, Data: hello},
		{Name: "link", Mode: fs.ModeSymlink | 0o777, Time: when, Target: "dir/nested/file.txt"},
		{Name: strings.Repeat("n", 120), Mode: 0o644, Time: when, Data: []byte("long")},
		{Name: "世界.txt", Mode: 0o644, Time: when, Data: []byte("utf8")},
	}
}

func requireTarLists(t *testing.T, file *File) {
	t.Helper()
	bin, err := exec.LookPath("tar")
	if err != nil {
		test.Skip(t, "missing tar")
	}
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	dir := t.TempDir()
	archive := filepath.Join(dir, "out.tar")
	require.NoError(t, os.WriteFile(archive, buf.Bytes(), 0o644))
	out, err := exec.Command(bin, "-tf", archive).CombinedOutput()
	require.NoError(t, err, string(out))
	var listed []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSuffix(line, "/")
		if line == "" {
			continue
		}
		listed = append(listed, line)
	}
	require.Equal(t, names(file), listed)
	for _, entry := range file.Entries {
		if entry.Mode&fs.ModeDir != 0 || entry.Mode&fs.ModeSymlink != 0 {
			continue
		}
		body, err := exec.Command(bin, "-xOf", archive, entry.Name).CombinedOutput()
		require.NoError(t, err, string(body))
		if len(body) == 0 {
			body = nil
		}
		require.Equal(t, entry.Data, body)
	}
	verbose, err := exec.Command(bin, "-tvf", archive).CombinedOutput()
	require.NoError(t, err, string(verbose))
	text := string(verbose)
	for _, entry := range file.Entries {
		if entry.Mode&fs.ModeSymlink == 0 {
			continue
		}
		require.Contains(t, text, entry.Name+" -> "+entry.Target)
	}
}

func mustTake(t *testing.T, name string) *File {
	t.Helper()
	raw, err := os.ReadFile(name)
	require.NoError(t, err)
	file, err := Take(bytes.NewReader(raw))
	require.NoError(t, err)
	return file
}

func requireEntries(t *testing.T, file *File, want []*Entry) {
	t.Helper()
	require.Len(t, file.Entries, len(want))
	for i := range want {
		got := file.Entries[i]
		require.Equal(t, want[i].Name, got.Name, "name %d", i)
		require.Equal(t, want[i].Mode, got.Mode, "mode %s", got.Name)
		require.True(t, want[i].Time.Equal(got.Time), "time %s: got %s want %s", got.Name, got.Time, want[i].Time)
		require.Equal(t, want[i].Data, got.Data, "data %s", got.Name)
		require.Equal(t, want[i].Target, got.Target, "target %s", got.Name)
	}
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
	Nano   int64
	Data   []byte
	Target string
}

func views(file *File) []view {
	out := make([]view, len(file.Entries))
	for i, entry := range file.Entries {
		out[i] = view{
			Name:   entry.Name,
			Mode:   entry.Mode,
			Nano:   entry.Time.UnixNano(),
			Data:   entry.Data,
			Target: entry.Target,
		}
	}
	return out
}

func names(file *File) []string {
	out := make([]string, len(file.Entries))
	for i, entry := range file.Entries {
		out[i] = entry.Name
	}
	return out
}
