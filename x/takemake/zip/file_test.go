package zip

import (
	"bytes"
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
	require.Zero(t, first.Entries[0].Time.Nanosecond())
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

func TestInfoTree(t *testing.T) {
	t.Parallel()
	when := time.Date(2020, 1, 2, 3, 4, 6, 0, time.UTC)
	file := mustTake(t, "testdata/info-tree.zip")
	requireEntries(t, file, []*Entry{
		{Name: "dir", Mode: fs.ModeDir | 0o750, Time: when},
		{Name: "dir/nested", Mode: fs.ModeDir | 0o750, Time: when},
		{Name: "dir/nested/file.txt", Mode: 0o640, Time: when, Data: []byte("hello zip")},
		{Name: "dir/empty", Mode: 0o644, Time: when},
		{Name: "dir/bin.dat", Mode: 0o600, Time: when, Data: []byte{'a', 0xff, 0x00, 'b'}},
		{Name: "link", Mode: fs.ModeSymlink | 0o777, Time: when, Target: "dir/nested/file.txt"},
		{Name: "世界.txt", Mode: 0o644, Time: when, Data: []byte("utf8")},
	})
	again := takeMade(t, file)
	require.Equal(t, views(file), views(again))
	requireUnzipReads(t, file)
}

func TestGoUnixAndWinXP(t *testing.T) {
	t.Parallel()
	unixWhen := []time.Time{
		time.Date(2011, 12, 8, 10, 4, 24, 0, time.UTC),
		time.Date(2011, 12, 8, 10, 4, 50, 0, time.UTC),
		time.Date(2011, 12, 8, 10, 8, 6, 0, time.UTC),
		time.Date(2011, 12, 8, 10, 6, 8, 0, time.UTC),
	}
	want := []*Entry{
		{Name: "hello", Mode: 0o666, Data: []byte("world \r\n")},
		{Name: "dir/bar", Mode: 0o666, Data: []byte("foo \r\n")},
		{Name: "dir/empty", Mode: fs.ModeDir | 0o777},
		{Name: "readonly", Mode: 0o444, Data: []byte("important \r\n")},
	}
	for _, name := range []string{"testdata/go-unix.zip", "testdata/go-winxp.zip"} {
		file := mustTake(t, name)
		require.Len(t, file.Entries, len(want))
		for i := range want {
			got := file.Entries[i]
			require.Equal(t, want[i].Name, got.Name, name)
			require.Equal(t, want[i].Mode, got.Mode, name+" "+got.Name)
			require.Equal(t, want[i].Data, got.Data, name+" "+got.Name)
			require.True(t, unixWhen[i].Equal(got.Time), "%s %s: got %s", name, got.Name, got.Time)
		}
		require.Equal(t, views(file), views(takeMade(t, file)))
	}
}

func TestGoSymlink(t *testing.T) {
	t.Parallel()
	file := mustTake(t, "testdata/go-symlink.zip")
	requireEntries(t, file, []*Entry{{
		Name:   "symlink",
		Mode:   fs.ModeSymlink | 0o777,
		Time:   time.Date(2012, 2, 3, 21, 56, 48, 0, time.UTC),
		Target: "../target",
	}})
	require.Equal(t, views(file), views(takeMade(t, file)))
}

func TestGoZip64(t *testing.T) {
	t.Parallel()
	file := mustTake(t, "testdata/go-zip64.zip")
	requireEntries(t, file, []*Entry{{
		Name: "README",
		Mode: 0o644,
		Time: time.Date(2012, 8, 10, 14, 33, 32, 0, time.UTC),
		Data: []byte("This small file is in ZIP64 format.\n"),
	}})
	require.Equal(t, views(file), views(takeMade(t, file)))
}

func TestGoDataDescriptor(t *testing.T) {
	t.Parallel()
	file := mustTake(t, "testdata/go-dd.zip")
	require.Len(t, file.Entries, 1)
	got := file.Entries[0]
	require.Equal(t, "filename", got.Name)
	require.Equal(t, fs.FileMode(0o666), got.Mode)
	require.Equal(t, []byte("This is a test textfile.\n"), got.Data)
	require.True(t, time.Date(2011, 2, 2, 13, 6, 20, 0, time.UTC).Equal(got.Time))
	require.Equal(t, views(file), views(takeMade(t, file)))
}

func TestGoUTF8(t *testing.T) {
	t.Parallel()
	when := time.Date(2017, 11, 6, 21, 9, 27, 0, time.UTC)
	seven := mustTake(t, "testdata/go-utf8-7zip.zip")
	requireEntries(t, seven, []*Entry{{
		Name: "世界",
		Mode: 0o666,
		Time: when,
	}})
	require.Equal(t, views(seven), views(takeMade(t, seven)))

	info := mustTake(t, "testdata/go-utf8-infozip.zip")
	requireEntries(t, info, []*Entry{{
		Name: "世界",
		Mode: 0o644,
		Time: when,
	}})
	require.Equal(t, views(info), views(takeMade(t, info)))
}

func TestLiveGitArchive(t *testing.T) {
	t.Parallel()
	bin := test.Need(t, "git")
	dir := gitTree(t, bin)
	archive := filepath.Join(dir, "git.zip")
	cmd := exec.Command(bin, "archive", "--format=zip", "-o", archive, "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	raw, err := os.ReadFile(archive)
	require.NoError(t, err)
	file, err := Take(bytes.NewReader(raw))
	require.NoError(t, err)
	when := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	// Regular files and directories stay DOS modes. Executables and symlinks keep Unix modes.
	requireEntries(t, file, []*Entry{
		{Name: "dir", Mode: fs.ModeDir | 0o777, Time: when},
		{Name: "dir/empty", Mode: 0o666, Time: when},
		{Name: "dir/nested", Mode: fs.ModeDir | 0o777, Time: when},
		{Name: "dir/nested/run.sh", Mode: 0o755, Time: when, Data: []byte("#!/bin/sh\n")},
		{Name: "dir/note.txt", Mode: 0o666, Time: when, Data: []byte("live")},
		{Name: "link", Mode: fs.ModeSymlink | 0o777, Time: when, Target: "dir/note.txt"},
		{Name: strings.Repeat("n", 120), Mode: 0o666, Time: when, Data: []byte("long")},
		{Name: "世界.txt", Mode: 0o666, Time: when, Data: []byte("utf8")},
	})
	require.Equal(t, views(file), views(takeMade(t, file)))
	requireUnzipReads(t, file)
}

func TestLiveInfoZip(t *testing.T) {
	t.Parallel()
	bin := test.Need(t, "zip")
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "dir"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dir", "note.txt"), []byte("live"), 0o640))
	require.NoError(t, os.Symlink("dir/note.txt", filepath.Join(dir, "link")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "世界.txt"), []byte("utf8"), 0o644))
	stamp := "2020-01-02T03:04:06Z"
	for _, name := range []string{"dir", "dir/note.txt", "link", "世界.txt"} {
		touch := exec.Command("touch", "-h", "-d", stamp, filepath.Join(dir, name))
		out, err := touch.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	archive := filepath.Join(dir, "live.zip")
	cmd := exec.Command(bin, "-r", "-y", "-q", archive, "dir", "link", "世界.txt")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "TZ=UTC")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	raw, err := os.ReadFile(archive)
	require.NoError(t, err)
	file, err := Take(bytes.NewReader(raw))
	require.NoError(t, err)
	when := time.Date(2020, 1, 2, 3, 4, 6, 0, time.UTC)
	requireEntries(t, file, []*Entry{
		{Name: "dir", Mode: fs.ModeDir | 0o750, Time: when},
		{Name: "dir/note.txt", Mode: 0o640, Time: when, Data: []byte("live")},
		{Name: "link", Mode: fs.ModeSymlink | 0o777, Time: when, Target: "dir/note.txt"},
		{Name: "世界.txt", Mode: 0o644, Time: when, Data: []byte("utf8")},
	})
	require.Equal(t, views(file), views(takeMade(t, file)))
	requireUnzipReads(t, file)
}

func gitTree(t *testing.T, bin string) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=takemake",
			"GIT_AUTHOR_EMAIL=takemake@example.com",
			"GIT_COMMITTER_NAME=takemake",
			"GIT_COMMITTER_EMAIL=takemake@example.com",
			"GIT_AUTHOR_DATE=2020-01-02T03:04:05Z",
			"GIT_COMMITTER_DATE=2020-01-02T03:04:05Z",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	run("init", "-q", "-b", "main")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "dir", "nested"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dir", "note.txt"), []byte("live"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dir", "empty"), nil, 0o644))
	script := filepath.Join(dir, "dir", "nested", "run.sh")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\n"), 0o755))
	require.NoError(t, os.Chmod(script, 0o755))
	require.NoError(t, os.Symlink("dir/note.txt", filepath.Join(dir, "link")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "世界.txt"), []byte("utf8"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, strings.Repeat("n", 120)), []byte("long"), 0o644))
	run("-c", "user.name=takemake", "-c", "user.email=takemake@example.com", "-c", "commit.gpgsign=false", "add", "-A")
	// core.filemode is false on Windows, so add records a regular file unless the index bit is set.
	run("-c", "user.name=takemake", "-c", "user.email=takemake@example.com", "-c", "commit.gpgsign=false", "update-index", "--chmod=+x", "dir/nested/run.sh")
	run("-c", "user.name=takemake", "-c", "user.email=takemake@example.com", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "tree")
	return dir
}

func requireUnzipReads(t *testing.T, file *File) {
	t.Helper()
	bin, err := exec.LookPath("unzip")
	if err != nil {
		test.Skip(t, "missing unzip")
	}
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	dir := t.TempDir()
	archive := filepath.Join(dir, "out.zip")
	require.NoError(t, os.WriteFile(archive, buf.Bytes(), 0o644))
	out, err := exec.Command(bin, "-t", archive).CombinedOutput()
	require.NoError(t, err, string(out))
	for _, entry := range file.Entries {
		if entry.Mode&fs.ModeDir != 0 {
			continue
		}
		body, err := exec.Command(bin, "-p", archive, entry.Name).CombinedOutput()
		require.NoError(t, err, string(body))
		if entry.Mode&fs.ModeSymlink != 0 {
			require.Equal(t, entry.Target, string(body))
			continue
		}
		if len(body) == 0 {
			body = nil
		}
		require.Equal(t, entry.Data, body)
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
