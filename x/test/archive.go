package test

import (
	"errors"
	"io"
	"io/fs"
	"testing"
)

// ArchiveTree checks a two-member archive: a/b.txt "hello" and z.txt "zee".
// Reading the directory "a" is left to the caller. Tar reports fs.ErrInvalid;
// zip reports a non-sentinel "is a directory" error.
func ArchiveTree(tb testing.TB, fsys fs.FS) {
	tb.Helper()
	ents, err := fs.ReadDir(fsys, ".")
	if err != nil {
		tb.Fatal(err)
	}
	if len(ents) != 2 {
		tb.Fatalf("ReadDir(.) len=%d want 2", len(ents))
	}
	if ents[0].Name() != "a" || !ents[0].IsDir() {
		tb.Fatalf("ents[0]=%s dir=%v want a/", ents[0].Name(), ents[0].IsDir())
	}
	if ents[1].Name() != "z.txt" || ents[1].IsDir() {
		tb.Fatalf("ents[1]=%s dir=%v want z.txt", ents[1].Name(), ents[1].IsDir())
	}

	ents, err = fs.ReadDir(fsys, "a")
	if err != nil {
		tb.Fatal(err)
	}
	if len(ents) != 1 || ents[0].Name() != "b.txt" {
		got := "<none>"
		if len(ents) > 0 {
			got = ents[0].Name()
		}
		tb.Fatalf("ReadDir(a) len=%d first=%s want one b.txt", len(ents), got)
	}

	b, err := fs.ReadFile(fsys, "a/b.txt")
	if err != nil {
		tb.Fatal(err)
	}
	if string(b) != "hello" {
		tb.Fatalf("a/b.txt = %q want hello", b)
	}

	st, err := fs.Stat(fsys, "a")
	if err != nil {
		tb.Fatal(err)
	}
	if !st.IsDir() {
		tb.Fatal("stat a: not a directory")
	}

	_, err = fsys.Open("missing")
	if !errors.Is(err, fs.ErrNotExist) {
		tb.Fatalf("open missing: %v want ErrNotExist", err)
	}
	_, err = fsys.Open("../x")
	if !errors.Is(err, fs.ErrInvalid) {
		tb.Fatalf("open ../x: %v want ErrInvalid", err)
	}
}

// ArchiveReadAt checks a.bin "hello world" via ReadAt at offset 6.
func ArchiveReadAt(tb testing.TB, fsys fs.FS) {
	tb.Helper()
	f, err := fsys.Open("a.bin")
	if err != nil {
		tb.Fatal(err)
	}
	CloseOnCleanup(tb, f)
	ra, ok := f.(io.ReaderAt)
	if !ok {
		tb.Fatal("open a.bin: not io.ReaderAt")
	}
	buf := make([]byte, 5)
	n, err := ra.ReadAt(buf, 6)
	if err != nil {
		tb.Fatal(err)
	}
	if n != 5 || string(buf) != "world" {
		tb.Fatalf("ReadAt = %d %q want 5 world", n, buf)
	}
}
