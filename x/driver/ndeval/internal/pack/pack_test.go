package pack

import (
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
)

func TestBytes(t *testing.T) {
	if Bytes(ndarray.U8, 3) != 12 {
		t.Fatalf("u8 %d", Bytes(ndarray.U8, 3))
	}
	if Bytes(ndarray.F32, 3) != 3 {
		t.Fatalf("f32 %d", Bytes(ndarray.F32, 3))
	}
}

func TestWiden(t *testing.T) {
	raw := []byte{1, 2}
	data, next := Widen(ndarray.F32, raw, []byte{9})
	if len(data) != 2 || data[0] != 1 || len(next) != 1 || next[0] != 9 {
		t.Fatalf("passthrough data %v staging %v", data, next)
	}
	data, next = Widen(ndarray.U8, raw, nil)
	if len(data) != 8 || len(next) != 8 || data[0] != 1 || data[4] != 2 {
		t.Fatalf("widen %v", data)
	}
	if &data[0] != &next[0] {
		t.Fatal("data and staging differ")
	}
	dirty := []byte{7, 7, 7, 7, 7, 7, 7, 7, 7}
	data, next = Widen(ndarray.U8, raw, dirty)
	if cap(next) < 9 || data[1] != 0 || data[5] != 0 {
		t.Fatalf("reuse %v cap %d", data, cap(next))
	}
}
