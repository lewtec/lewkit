package compression

import (
	"bytes"
	"io"
	"testing"

	"github.com/lewtec/lewkit/x/test"
)

// RoundTrip compresses p with c and decompresses the result.
// c must be a [Compressor] and a [Decompressor].
func RoundTrip(tb testing.TB, c Codec, p []byte) {
	tb.Helper()
	wri, ok := c.(Compressor)
	if !ok {
		tb.Fatalf("%s: not a Compressor", c.Name())
	}
	rdr, ok := c.(Decompressor)
	if !ok {
		tb.Fatalf("%s: not a Decompressor", c.Name())
	}
	var buf bytes.Buffer
	w, err := wri.Writer(&buf)
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := w.Write(p); err != nil {
		tb.Fatal(err)
	}
	if err := w.Close(); err != nil {
		tb.Fatal(err)
	}
	r, err := rdr.Reader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		tb.Fatal(err)
	}
	test.CloseOnCleanup(tb, r)
	got, err := io.ReadAll(r)
	if err != nil {
		tb.Fatal(err)
	}
	if !bytes.Equal(got, p) {
		tb.Fatalf("round trip %s: got %q want %q", c.Name(), got, p)
	}
}
