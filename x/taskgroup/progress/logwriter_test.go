package progress

import (
	"strings"
	"testing"
)

func TestLinePrinterSplitsOnNewline(t *testing.T) {
	var got []string
	w := &linePrinter{print: func(s string) { got = append(got, s) }}
	if _, err := w.Write([]byte("hello\nwor")); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("ld\n")); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "hello" || got[1] != "world" {
		t.Fatalf("got %#v", got)
	}
}

func TestLinePrinterHoldsPartial(t *testing.T) {
	var got []string
	w := &linePrinter{print: func(s string) { got = append(got, s) }}
	if _, err := w.Write([]byte("no-nl")); err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("partial flushed: %#v", got)
	}
	w.close()
	if _, err := w.Write([]byte("after\n")); err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "" {
		t.Fatalf("print after close: %#v", got)
	}
}
