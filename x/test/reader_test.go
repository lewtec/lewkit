package test

import (
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
)

func TestOnlyReader(t *testing.T) {
	r := OnlyReader{strings.NewReader("x")}
	if _, ok := any(r).(io.ReaderAt); ok {
		t.Fatal("OnlyReader implements io.ReaderAt")
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "x" {
		t.Fatalf("ReadAll = %q want x", b)
	}
}

func TestReaderOnlyFS(t *testing.T) {
	const name = "a.tar"
	fsys := ReaderOnlyFS(name, strings.NewReader("x"))
	f, err := fsys.Open(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := f.(readerOnlyFile); !ok {
		t.Fatalf("Open type %T, want readerOnlyFile value", f)
	}
	if _, ok := f.(io.ReaderAt); ok {
		t.Fatal("file implements io.ReaderAt")
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "x" {
		t.Fatalf("ReadAll = %q want x", b)
	}
	if _, err := f.Stat(); !errors.Is(err, fs.ErrInvalid) {
		t.Fatalf("Stat: %v want ErrInvalid", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := fsys.Open("other"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Open other: %v want ErrNotExist", err)
	}
}
