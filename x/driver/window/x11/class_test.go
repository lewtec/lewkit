package x11

import "testing"

func TestWMClassBytes(t *testing.T) {
	raw := wmClassBytes("Demo", "br.tec.lew.demo")
	want := append(append([]byte("Demo"), 0), append([]byte("br.tec.lew.demo"), 0)...)
	if string(raw) != string(want) {
		t.Fatalf("class %q", raw)
	}
	if wmClassBytes("", "") != nil {
		t.Fatal("empty class")
	}
}
